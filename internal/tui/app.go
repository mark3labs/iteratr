package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/nats-io/nats.go"
)

// App is the main Bubbletea model that manages the TUI application.
type App struct {
	// View components
	dashboard      *Dashboard
	logs           *LogViewer
	agent          *AgentOutput
	status         *StatusBar
	sidebar        *Sidebar
	dialog         *Dialog
	taskModal      *TaskModal
	noteModal      *NoteModal
	noteInputModal *NoteInputModal

	// Layout management
	layout      Layout
	layoutDirty bool

	// State
	logsVisible    bool // Toggle for logs modal overlay
	sidebarVisible bool // Toggle for sidebar visibility in compact mode
	iteration      int  // Current iteration number (for note tagging)
	store          *session.Store
	sessionName    string
	nc             *nats.Conn
	ctx            context.Context
	width          int
	height         int
	quitting       bool
	eventChan      chan session.Event // Channel for receiving NATS events
	sendChan       chan string        // Channel for sending user messages to orchestrator
}

// NewApp creates a new TUI application with the given session store and NATS connection.
func NewApp(ctx context.Context, store *session.Store, sessionName string, nc *nats.Conn, sendChan chan string) *App {
	agent := NewAgentOutput()
	sidebar := NewSidebar()
	return &App{
		store:          store,
		sessionName:    sessionName,
		nc:             nc,
		ctx:            ctx,
		sendChan:       sendChan,
		sidebarVisible: false, // Sidebar hidden by default in compact mode
		dashboard:      NewDashboard(agent, sidebar),
		logs:           NewLogViewer(),
		agent:          agent,
		status:         NewStatusBar(sessionName),
		sidebar:        sidebar,
		dialog:         NewDialog(),
		taskModal:      NewTaskModal(),
		noteModal:      NewNoteModal(),
		noteInputModal: NewNoteInputModal(),
		eventChan:      make(chan session.Event, 100), // Buffered channel for events
		layoutDirty:    true,                          // Calculate layout on first render
	}
}

// Init initializes the application and returns any initial commands.
// In Bubbletea v2, Init returns only tea.Cmd (not Model).
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.subscribeToEvents(),
		a.waitForEvents(),
		a.loadInitialState(),
		a.agent.Init(),
		a.checkConnectionHealth(), // Start periodic connection health checks
		a.status.StartDurationTick(),
	)
}

// Update handles incoming messages and updates the model state.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return a.handleKeyPress(msg)

	case tea.MouseClickMsg:
		return a.handleMouse(msg)

	case tea.MouseWheelMsg:
		return a.handleMouseWheel(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.layoutDirty = true
		// Recalculate layout and propagate sizes
		a.layout = CalculateLayout(a.width, a.height)
		a.propagateSizes()
		a.layoutDirty = false
		return a, nil

	case AgentOutputMsg:
		return a, a.agent.AppendText(msg.Content)

	case AgentToolCallMsg:
		return a, a.agent.AppendToolCall(msg)

	case AgentThinkingMsg:
		return a, a.agent.AppendThinking(msg.Content)

	case AgentFinishMsg:
		queueCmd := a.dashboard.SetAgentBusy(false)
		return a, tea.Batch(a.agent.AppendFinish(msg), queueCmd)

	case IterationStartMsg:
		a.iteration = msg.Number // Track current iteration for note creation
		busyCmd := a.dashboard.SetAgentBusy(true)
		return a, tea.Batch(
			a.dashboard.SetIteration(msg.Number),
			a.agent.AddIterationDivider(msg.Number),
			busyCmd,
		)

	case StateUpdateMsg:
		// Propagate state updates to all components
		a.status.SetState(msg.State)
		a.sidebar.SetState(msg.State)
		a.dashboard.UpdateState(msg.State)
		a.logs.SetState(msg.State)
		return a, a.status.Tick()

	case EventMsg:
		// Forward event to log viewer, reload state, and wait for next event
		return a, tea.Batch(
			a.logs.AddEvent(msg.Event),
			a.loadInitialState(), // Reload state to reflect changes
			a.waitForEvents(),    // Recursively wait for next event
		)

	case ConnectionStatusMsg:
		// Update connection status in status bar
		a.status.SetConnectionStatus(msg.Connected)
		// Reschedule health check
		return a, a.checkConnectionHealth()

	case SessionCompleteMsg:
		// Show completion dialog - user can dismiss and continue viewing or quit manually
		a.dialog.Show(
			"Session Complete",
			"All tasks have been completed successfully!",
			nil, // Just close the modal, don't quit
		)
		return a, nil

	case UserInputMsg:
		// Handle user input from the text field
		// TODO: Wire to orchestrator via sendChan to call runner.SendMessage()
		// For now, this is a placeholder that will be completed when sendChan is added
		// Non-blocking send to avoid UI freeze if orchestrator isn't ready
		if a.sendChan != nil {
			select {
			case a.sendChan <- msg.Text:
				// Message sent successfully
			default:
				// Channel full or not ready, drop message
				// TODO: Add visual feedback for dropped messages
			}
		}
		return a, nil

	case OpenTaskModalMsg:
		// Open task modal with the selected task
		a.taskModal.SetTask(msg.Task)
		return a, nil

	case CreateNoteMsg:
		// Create a new note via Store.NoteAdd()
		// The note will be published to NATS and picked up by event subscription
		// Use App's iteration field (set by IterationStartMsg) instead of message field
		iteration := a.iteration
		if msg.Iteration != 0 {
			iteration = msg.Iteration // Allow override if explicitly set
		}
		go func() {
			_, err := a.store.NoteAdd(a.ctx, a.sessionName, session.NoteAddParams{
				Content:   msg.Content,
				Type:      msg.NoteType,
				Iteration: iteration,
			})
			if err != nil {
				// TODO: Add error handling/visual feedback
				// For now, errors are silently ignored
			}
		}()
		// Close the modal after submitting
		a.noteInputModal.Close()
		return a, nil
	}

	// Update status bar (for spinner animation) - always visible
	statusCmd := a.status.Update(msg)

	// Update sidebar if visible (desktop mode or toggled in compact mode)
	var sidebarCmd tea.Cmd
	if a.layout.Mode == LayoutDesktop || a.sidebarVisible {
		sidebarCmd = a.sidebar.Update(msg)
	}

	// Delegate to dashboard and logs if visible
	dashCmd := a.dashboard.Update(msg)
	var logsCmd tea.Cmd
	if a.logsVisible {
		logsCmd = a.logs.Update(msg)
	}

	return a, tea.Batch(statusCmd, sidebarCmd, dashCmd, logsCmd)
}

// handleKeyPress processes keyboard input using hierarchical priority routing.
// Priority: Dialog → Modal → Global → View → Focus → Component
func (a *App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 0. Dialog gets priority when visible
	if a.dialog.IsVisible() {
		if cmd := a.dialog.Update(msg); cmd != nil {
			return a, cmd
		}
		return a, nil // Consume all keys when dialog is visible
	}

	// 1. Modal gets priority when visible
	if a.taskModal != nil && a.taskModal.IsVisible() {
		// ESC key closes the modal
		if msg.String() == "esc" {
			a.taskModal.Close()
			if a.sidebar != nil {
				a.sidebar.ClearActiveTask()
			}
			return a, nil
		}
		// Block all other keys when modal is visible
		return a, nil
	}

	if a.noteModal != nil && a.noteModal.IsVisible() {
		if msg.String() == "esc" {
			a.noteModal.Close()
			if a.sidebar != nil {
				a.sidebar.ClearActiveNote()
			}
			return a, nil
		}
		return a, nil
	}

	// Note input modal gets priority when visible
	if a.noteInputModal != nil && a.noteInputModal.IsVisible() {
		return a, a.noteInputModal.Update(msg)
	}

	// 2. Logs modal captures keys when visible
	if a.logsVisible {
		switch msg.String() {
		case "esc", "ctrl+l":
			a.logsVisible = false
			return a, nil
		default:
			// Forward scroll keys to log viewport
			return a, a.logs.Update(msg)
		}
	}

	// 3. Global keys (highest priority)
	if cmd := a.handleGlobalKeys(msg); cmd != nil {
		return a, cmd
	}

	// 4. Delegate to dashboard
	return a, a.dashboard.Update(msg)
}

// handleMouse processes mouse click events using coordinate-based hit detection.
func (a *App) handleMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()

	// Only handle left mouse button
	if mouse.Button != tea.MouseLeft {
		return a, nil
	}

	// Dialog takes priority - any click dismisses it
	if a.dialog.IsVisible() {
		return a, a.dialog.HandleClick(mouse.X, mouse.Y)
	}

	// Check if note input modal is open - handle button clicks or close
	if a.noteInputModal.IsVisible() {
		if cmd := a.noteInputModal.HandleClick(mouse.X, mouse.Y); cmd != nil {
			return a, cmd
		}
		// Click outside modal closes it
		a.noteInputModal.Close()
		return a, nil
	}

	// Check if task modal is open - click outside closes it
	if a.taskModal.IsVisible() {
		// If click is on a different task, switch to that task
		if task := a.sidebar.TaskAtPosition(mouse.X, mouse.Y); task != nil {
			a.taskModal.SetTask(task)
			a.sidebar.SetActiveTask(task.ID)
			return a, nil
		}
		// Click anywhere else closes the modal
		a.taskModal.Close()
		a.sidebar.ClearActiveTask()
		return a, nil
	}

	// Check if note modal is open - click outside closes it
	if a.noteModal.IsVisible() {
		// If click is on a different note, switch to that note
		if note := a.sidebar.NoteAtPosition(mouse.X, mouse.Y); note != nil {
			a.noteModal.SetNote(note)
			a.sidebar.SetActiveNote(note.ID)
			return a, nil
		}
		// Click anywhere else closes the modal
		a.noteModal.Close()
		a.sidebar.ClearActiveNote()
		return a, nil
	}

	// Determine which pane was clicked and update focus
	a.focusPaneAtPosition(mouse.X, mouse.Y)

	// Check if a task was clicked
	if task := a.sidebar.TaskAtPosition(mouse.X, mouse.Y); task != nil {
		a.taskModal.SetTask(task)
		a.sidebar.SetActiveTask(task.ID)
		return a, nil
	}

	// Check if a note was clicked
	if note := a.sidebar.NoteAtPosition(mouse.X, mouse.Y); note != nil {
		a.noteModal.SetNote(note)
		a.sidebar.SetActiveNote(note.ID)
		return a, nil
	}

	// Check if input area was clicked (focus text input)
	if a.agent.IsInputAreaClick(mouse.X, mouse.Y) {
		// Set input focus via dashboard (same as pressing 'i')
		a.dashboard.focusPane = FocusInput
		a.dashboard.inputFocused = true
		if a.agent != nil {
			a.agent.SetInputFocused(true)
		}
		return a, nil
	}

	// Check if agent output was clicked (expand/collapse tool output)
	if a.agent.HandleClick(mouse.X, mouse.Y) {
		return a, nil
	}

	return a, nil
}

// focusPaneAtPosition updates the focused pane based on click coordinates.
// This enables mouse-based pane focus switching.
func (a *App) focusPaneAtPosition(x, y int) {
	prevPane := a.dashboard.focusPane

	switch {
	case a.agent.IsViewportArea(x, y):
		a.dashboard.focusPane = FocusAgent
		a.dashboard.inputFocused = false
		if a.agent != nil {
			a.agent.SetInputFocused(false)
		}
	case a.sidebar.IsTasksArea(x, y):
		a.dashboard.focusPane = FocusTasks
		a.dashboard.inputFocused = false
		if a.agent != nil {
			a.agent.SetInputFocused(false)
		}
	case a.sidebar.IsNotesArea(x, y):
		a.dashboard.focusPane = FocusNotes
		a.dashboard.inputFocused = false
		if a.agent != nil {
			a.agent.SetInputFocused(false)
		}
	default:
		return
	}

	if a.dashboard.focusPane != prevPane {
		a.dashboard.updateScrollListFocus()
	}
}

// handleMouseWheel processes mouse wheel events for viewport scrolling.
// Scrolls the viewport under the cursor regardless of which pane has keyboard focus.
func (a *App) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()

	const scrollLines = 3

	var lines int
	switch mouse.Button {
	case tea.MouseWheelUp:
		lines = -scrollLines
	case tea.MouseWheelDown:
		lines = scrollLines
	default:
		return a, nil
	}

	// Scroll the viewport under the cursor
	if a.agent.IsViewportArea(mouse.X, mouse.Y) {
		a.agent.ScrollViewport(lines)
		return a, nil
	}

	if a.sidebar.IsTasksArea(mouse.X, mouse.Y) {
		a.sidebar.ScrollTasks(lines)
		return a, nil
	}

	if a.sidebar.IsNotesArea(mouse.X, mouse.Y) {
		a.sidebar.ScrollNotes(lines)
		return a, nil
	}

	return a, nil
}

// handleGlobalKeys processes global keyboard shortcuts (highest priority).
func (a *App) handleGlobalKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		a.quitting = true
		return tea.Quit
	case "ctrl+l":
		a.logsVisible = !a.logsVisible
		return nil
	case "ctrl+s":
		a.sidebarVisible = !a.sidebarVisible
		return nil
	case "ctrl+n":
		// Guard: don't open note input modal if another modal/dialog is visible
		// This prevents modal stacking and ensures clean UI state
		if a.dialog.IsVisible() || a.taskModal.IsVisible() || a.noteModal.IsVisible() || a.logsVisible {
			return nil // No-op if any other modal/dialog is visible
		}
		// Guard: don't open note input modal if there's no active iteration
		// Notes must be tagged to an iteration, so we need at least iteration 1
		if a.iteration == 0 {
			return nil // No-op if no iteration has started yet
		}
		// Open note input modal
		return a.noteInputModal.Show()
	}
	return nil
}

// View renders the current view. In Bubbletea v2, this returns tea.View
// with display options like AltScreen and MouseMode.
func (a *App) View() tea.View {
	var view tea.View
	view.AltScreen = true                    // Full-screen mode
	view.MouseMode = tea.MouseModeCellMotion // Enable mouse events
	view.ReportFocus = true                  // Enable focus events
	view.KeyboardEnhancements = tea.KeyboardEnhancements{
		ReportEventTypes: true, // Required for ctrl+enter and other enhanced key events
	}

	if a.quitting {
		// Return minimal view when quitting - exit alt screen for proper terminal restoration
		view.AltScreen = false
		view.MouseMode = 0
		view.ReportFocus = false
		view.Content = lipglossv2.NewLayer("")
		return view
	}

	// Recalculate layout if needed
	if a.layoutDirty {
		a.layout = CalculateLayout(a.width, a.height)
		a.propagateSizes()
		a.layoutDirty = false
	}

	// Create screen buffer for drawing
	canvas := uv.NewScreenBuffer(a.width, a.height)

	// Draw all components to canvas
	view.Cursor = a.Draw(canvas, canvas.Bounds())

	// Render canvas to string
	content := canvas.Render()

	view.Content = lipglossv2.NewLayer(content)

	return view
}

// Draw renders all components to the screen buffer.
func (a *App) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Draw main content (always dashboard)
	cursor := a.dashboard.Draw(scr, a.layout.Main)
	a.status.Draw(scr, a.layout.Status)

	// Draw sidebar based on mode
	if a.layout.Mode == LayoutDesktop {
		a.sidebar.Draw(scr, a.layout.Sidebar)
	} else if a.sidebarVisible {
		sidebarWidth := SidebarWidthDesktop
		if a.layout.Main.Dx()/2 < sidebarWidth {
			sidebarWidth = a.layout.Main.Dx() / 2
		}
		sidebarRect := uv.Rect(
			a.layout.Main.Max.X-sidebarWidth,
			a.layout.Main.Min.Y,
			sidebarWidth,
			a.layout.Main.Dy(),
		)
		a.sidebar.Draw(scr, sidebarRect)
	}

	// Draw overlays
	if a.logsVisible {
		a.logs.Draw(scr, area)
	}
	if a.taskModal.IsVisible() {
		a.taskModal.Draw(scr, area)
	}
	if a.noteModal.IsVisible() {
		a.noteModal.Draw(scr, area)
	}
	if a.noteInputModal.IsVisible() {
		a.noteInputModal.Draw(scr, area)
	}
	if a.dialog.IsVisible() {
		a.dialog.Draw(scr, area)
	}

	return cursor
}

// waitForEvents listens on the event channel and converts events to messages.
// This command recursively calls itself to continuously receive events.
func (a *App) waitForEvents() tea.Cmd {
	return func() tea.Msg {
		// Block waiting for next event
		event, ok := <-a.eventChan
		if !ok {
			// Channel closed, stop receiving
			return nil
		}
		return EventMsg{Event: event}
	}
}

// subscribeToEvents subscribes to NATS events for this session.
// This runs in a managed goroutine and sends messages to the Update loop.
func (a *App) subscribeToEvents() tea.Cmd {
	return func() tea.Msg {
		// Subscribe to all events for this session using wildcard pattern
		subject := fmt.Sprintf("iteratr.%s.>", a.sessionName)

		// Create subscription that forwards events to the event channel
		sub, err := a.nc.Subscribe(subject, func(msg *nats.Msg) {
			// Parse event from message data
			var event session.Event
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				// Skip malformed events
				return
			}

			// Send to channel (non-blocking)
			select {
			case a.eventChan <- event:
			default:
				// Channel full, drop event
			}
		})

		if err != nil {
			// Return error message
			return fmt.Errorf("failed to subscribe to events: %w", err)
		}

		// Clean up when context is cancelled
		<-a.ctx.Done()
		_ = sub.Unsubscribe()
		close(a.eventChan)

		return nil
	}
}

// loadInitialState loads the current session state from the event log.
func (a *App) loadInitialState() tea.Cmd {
	return func() tea.Msg {
		state, err := a.store.LoadState(a.ctx, a.sessionName)
		if err != nil {
			// TODO: Handle error properly
			return nil
		}
		return StateUpdateMsg{State: state}
	}
}

// checkConnectionHealth monitors NATS connection status and sends updates.
// It checks the connection every 2 seconds and sends a ConnectionStatusMsg
// when the status changes.
func (a *App) checkConnectionHealth() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		connected := a.nc != nil && a.nc.IsConnected()
		return ConnectionStatusMsg{Connected: connected}
	})
}

// Custom message types for the TUI
type AgentOutputMsg struct {
	Content string
}

// FileDiff contains before/after file content from an edit tool call.
type FileDiff struct {
	File      string // Absolute file path
	Before    string // Full file content before edit
	After     string // Full file content after edit
	Additions int    // Number of added lines
	Deletions int    // Number of deleted lines
}

type AgentToolCallMsg struct {
	ToolCallID string
	Title      string
	Status     string
	Kind       string
	Input      map[string]any
	Output     string
	FileDiff   *FileDiff
}

type AgentThinkingMsg struct {
	Content string
}

type AgentFinishMsg struct {
	Reason   string
	Error    string
	Model    string
	Provider string
	Duration time.Duration
}

type IterationStartMsg struct {
	Number int
}

type StateUpdateMsg struct {
	State *session.State
}

type EventMsg struct {
	Event session.Event
}

// ConnectionStatusMsg is sent when NATS connection status changes.
type ConnectionStatusMsg struct {
	Connected bool
}

// UserInputMsg is sent when the user types a message in the input field.
type UserInputMsg struct {
	Text string
}

// CreateNoteMsg is sent when the user creates a new note from the input modal.
type CreateNoteMsg struct {
	Content   string
	NoteType  string
	Iteration int
}

// propagateSizes updates component sizes based on the current layout.
// This is called when the layout changes (on window resize or mode switch).
func (a *App) propagateSizes() {
	// Propagate sizes to status bar
	a.status.SetSize(a.layout.Status.Dx(), a.layout.Status.Dy())
	a.status.SetLayoutMode(a.layout.Mode)

	// Propagate sizes to main content components
	a.dashboard.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())
	a.logs.SetSize(a.width, a.height)

	// Propagate sidebar size based on layout mode
	if a.layout.Mode == LayoutDesktop {
		// Desktop mode: use dedicated sidebar area
		a.sidebar.SetSize(a.layout.Sidebar.Dx(), a.layout.Sidebar.Dy())
	} else {
		// Compact mode: sidebar overlays with fixed width
		sidebarWidth := SidebarWidthDesktop
		if a.layout.Main.Dx()/2 < sidebarWidth {
			sidebarWidth = a.layout.Main.Dx() / 2
		}
		a.sidebar.SetSize(sidebarWidth, a.layout.Main.Dy())
	}
}
