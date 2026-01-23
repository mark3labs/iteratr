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
// It contains all view components and handles routing between them.
type App struct {
	// View components
	dashboard *Dashboard
	logs      *LogViewer
	notes     *NotesPanel
	agent     *AgentOutput
	header    *Header
	footer    *Footer
	status    *StatusBar
	sidebar   *Sidebar
	dialog    *Dialog
	taskModal *TaskModal
	noteModal *NoteModal

	// Layout management
	layout      Layout
	layoutDirty bool

	// Mouse interaction (coordinate-based hit detection, no zone manager needed)

	// State
	activeView     ViewType
	sidebarVisible bool // Toggle for sidebar visibility in compact mode
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
		activeView:     ViewDashboard,
		sidebarVisible: false, // Sidebar hidden by default in compact mode
		dashboard:      NewDashboard(agent, sidebar),
		logs:           NewLogViewer(),
		notes:          NewNotesPanel(),
		agent:          agent,
		header:         NewHeader(sessionName),
		footer:         NewFooter(),
		status:         NewStatusBar(),
		sidebar:        sidebar,
		dialog:         NewDialog(),
		taskModal:      NewTaskModal(),
		noteModal:      NewNoteModal(),
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
	)
}

// Update handles incoming messages and updates the model state.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return a.handleKeyPress(msg)

	case tea.MouseClickMsg:
		return a.handleMouse(msg)

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
		busyCmd := a.dashboard.SetAgentBusy(true)
		return a, tea.Batch(
			a.dashboard.SetIteration(msg.Number),
			a.agent.AddIterationDivider(msg.Number),
			busyCmd,
		)

	case StateUpdateMsg:
		// Propagate state updates to all components
		a.header.SetState(msg.State)
		a.status.SetState(msg.State)
		a.sidebar.SetState(msg.State)
		a.dashboard.UpdateState(msg.State)
		a.logs.SetState(msg.State)
		a.notes.UpdateState(msg.State)
		return a, nil

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
		// Show completion dialog instead of quitting
		a.dialog.Show(
			"Session Complete",
			"All tasks have been completed successfully!",
			func() tea.Cmd {
				a.quitting = true
				return tea.Quit
			},
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
	}

	// Update status bar (for spinner animation) - always visible
	statusCmd := a.status.Update(msg)

	// Update sidebar if visible (desktop mode or toggled in compact mode)
	var sidebarCmd tea.Cmd
	if a.layout.Mode == LayoutDesktop || a.sidebarVisible {
		sidebarCmd = a.sidebar.Update(msg)
	}

	// Delegate to active view component
	var cmd tea.Cmd
	switch a.activeView {
	case ViewDashboard:
		cmd = a.dashboard.Update(msg)
	case ViewLogs:
		cmd = a.logs.Update(msg)
	case ViewNotes:
		cmd = a.notes.Update(msg)
	}

	return a, tea.Batch(statusCmd, sidebarCmd, cmd)
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

	// 2. Global keys (highest priority)
	if cmd := a.handleGlobalKeys(msg); cmd != nil {
		return a, cmd
	}

	// 2. View-level keys (switching views)
	if cmd := a.handleViewKeys(msg); cmd != nil {
		return a, cmd
	}

	// 3. Focus-specific keys (tab navigation, etc.)
	if cmd := a.handleFocusKeys(msg); cmd != nil {
		return a, cmd
	}

	// 4. Delegate to active component
	return a, a.delegateToActive(msg)
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

	// Check if a footer button was clicked
	if action := a.footer.ActionAtPosition(mouse.X, mouse.Y); action != "" {
		return a, a.handleFooterAction(action)
	}

	// Check if input area was clicked (focus text input)
	if a.activeView == ViewDashboard && a.agent.IsInputAreaClick(mouse.X, mouse.Y) {
		// Set input focus via dashboard
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

// handleFooterAction processes a footer button click action.
func (a *App) handleFooterAction(action FooterAction) tea.Cmd {
	switch action {
	case FooterActionDashboard:
		a.activeView = ViewDashboard
		a.footer.SetActiveView(ViewDashboard)
	case FooterActionLogs:
		a.activeView = ViewLogs
		a.footer.SetActiveView(ViewLogs)
	case FooterActionNotes:
		a.activeView = ViewNotes
		a.footer.SetActiveView(ViewNotes)
	case FooterActionSidebar:
		a.sidebarVisible = !a.sidebarVisible
	case FooterActionQuit:
		a.quitting = true
		return tea.Quit
	}
	return func() tea.Msg { return nil }
}

// handleGlobalKeys processes global keyboard shortcuts (highest priority).
// Returns tea.Quit for quit commands, nil for unhandled keys.
func (a *App) handleGlobalKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		a.quitting = true
		return tea.Quit
	case "?":
		// TODO: Toggle help view (Phase 14+)
		return nil
	}
	return nil
}

// handleViewKeys processes view switching shortcuts.
// Returns non-nil cmd if key was handled, nil otherwise.
func (a *App) handleViewKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "1":
		a.activeView = ViewDashboard
		a.footer.SetActiveView(ViewDashboard)
		return func() tea.Msg { return nil }
	case "2":
		a.activeView = ViewLogs
		a.footer.SetActiveView(ViewLogs)
		return func() tea.Msg { return nil }
	case "3":
		a.activeView = ViewNotes
		a.footer.SetActiveView(ViewNotes)
		return func() tea.Msg { return nil }
	case "s":
		// Toggle sidebar visibility (only relevant in compact mode)
		a.sidebarVisible = !a.sidebarVisible
		return func() tea.Msg { return nil }
	}
	return nil
}

// handleFocusKeys processes focus navigation shortcuts (tab, shift+tab).
// Returns non-nil cmd if key was handled, nil otherwise.
func (a *App) handleFocusKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "tab":
		// TODO: Cycle focus forward (Phase 14+)
		return nil
	case "shift+tab":
		// TODO: Cycle focus backward (Phase 14+)
		return nil
	}
	return nil
}

// delegateToActive forwards key messages to the active view component.
// This allows components to handle their own keyboard shortcuts (scrolling, etc).
func (a *App) delegateToActive(msg tea.KeyPressMsg) tea.Cmd {
	switch a.activeView {
	case ViewDashboard:
		return a.dashboard.Update(msg)
	case ViewLogs:
		return a.logs.Update(msg)
	case ViewNotes:
		return a.notes.Update(msg)
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
	var cursor *tea.Cursor

	// Draw all regions using the calculated layout
	a.header.Draw(scr, a.layout.Header)
	cursor = a.drawActiveView(scr, a.layout.Main)
	a.status.Draw(scr, a.layout.Status)
	a.footer.Draw(scr, a.layout.Footer)

	// Draw sidebar based on mode:
	// - Desktop mode: always show sidebar on the right
	// - Compact mode: show sidebar only if toggled visible (overlay on main content)
	if a.layout.Mode == LayoutDesktop {
		// Desktop mode: sidebar is always visible in dedicated area
		a.sidebar.Draw(scr, a.layout.Sidebar)
	} else if a.sidebarVisible {
		// Compact mode: sidebar overlays main content when toggled visible
		// Use sidebar width from desktop mode
		sidebarWidth := SidebarWidthDesktop
		if a.layout.Main.Dx()/2 < sidebarWidth {
			sidebarWidth = a.layout.Main.Dx() / 2
		}
		// Position sidebar on the right side of main area
		sidebarRect := uv.Rect(
			a.layout.Main.Max.X-sidebarWidth,
			a.layout.Main.Min.Y,
			sidebarWidth,
			a.layout.Main.Dy(),
		)
		a.sidebar.Draw(scr, sidebarRect)
	}

	// Draw overlays on top (modals, then dialog)
	if a.taskModal.IsVisible() {
		a.taskModal.Draw(scr, area)
	}
	if a.noteModal.IsVisible() {
		a.noteModal.Draw(scr, area)
	}
	if a.dialog.IsVisible() {
		a.dialog.Draw(scr, area)
	}

	return cursor
}

// drawActiveView renders the currently active view component to the screen.
func (a *App) drawActiveView(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	switch a.activeView {
	case ViewDashboard:
		return a.dashboard.Draw(scr, area)
	case ViewLogs:
		return a.logs.Draw(scr, area)
	case ViewNotes:
		return a.notes.Draw(scr, area)
	}
	return nil
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
		sub.Unsubscribe()
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

type AgentToolCallMsg struct {
	ToolCallID string
	Title      string
	Status     string
	Input      map[string]any
	Output     string
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

// propagateSizes updates component sizes based on the current layout.
// This is called when the layout changes (on window resize or mode switch).
func (a *App) propagateSizes() {
	// Propagate sizes to header, footer, and status bar
	a.header.SetSize(a.layout.Header.Dx(), a.layout.Header.Dy())
	a.footer.SetSize(a.layout.Footer.Dx(), a.layout.Footer.Dy())
	a.status.SetSize(a.layout.Status.Dx(), a.layout.Status.Dy())

	// Propagate layout mode to header, footer, and status bar
	a.header.SetLayoutMode(a.layout.Mode)
	a.footer.SetLayoutMode(a.layout.Mode)
	a.status.SetLayoutMode(a.layout.Mode)
	a.footer.SetActiveView(a.activeView)

	// Propagate sizes to main content components
	// Note: dashboard owns the agent output component, so we only size the dashboard
	a.dashboard.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())
	a.logs.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())
	a.notes.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())

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
