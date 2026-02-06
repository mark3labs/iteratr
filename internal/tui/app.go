package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/git"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/state"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/nats-io/nats.go"
)

// Orchestrator defines the interface for pause/resume control.
// This interface allows the TUI to control orchestrator state without creating a circular dependency.
type Orchestrator interface {
	RequestPause()
	CancelPause()
	Resume()
	IsPaused() bool
}

// loadUIState loads the UI state from persistent storage.
// Returns default state if loading fails.
func loadUIState(dataDir string) *state.UIState {
	return state.Load(dataDir)
}

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
	taskInputModal *TaskInputModal
	subagentModal  *SubagentModal
	toast          *Toast

	// Layout management
	layout      Layout
	layoutDirty bool

	// State
	logsVisible       bool      // Toggle for logs modal overlay
	sidebarVisible    bool      // Toggle for sidebar visibility in compact mode
	sidebarUserHidden bool      // True if user manually hid sidebar (vs auto-hidden)
	iteration         int       // Current iteration number (for note tagging)
	queueDepth        int       // Number of messages waiting in orchestrator queue
	modifiedFileCount int       // Number of files modified in current iteration
	awaitingPrefixKey bool      // True when waiting for second key after ctrl+x
	lastGitCheck      time.Time // Last time git info was fetched (for throttling)
	store             *session.Store
	sessionName       string
	workDir           string // Working directory for agent (needed for subagent modal)
	dataDir           string // Data directory for persistent storage
	nc                *nats.Conn
	ctx               context.Context
	width             int
	height            int
	quitting          bool
	eventChan         chan session.Event // Channel for receiving NATS events
	sendChan          chan string        // Channel for sending user messages to orchestrator
	orchestrator      Orchestrator       // Interface to orchestrator for pause/resume control
}

// NewApp creates a new TUI application with the given session store and NATS connection.
func NewApp(ctx context.Context, store *session.Store, sessionName, workDir, dataDir string, nc *nats.Conn, sendChan chan string, orch Orchestrator) *App {
	agent := NewAgentOutput()
	sidebar := NewSidebar()

	// Load UI state from persistent storage
	uiState := loadUIState(dataDir)

	// Create status bar and initialize with sidebar state
	statusBar := NewStatusBar(sessionName)
	statusBar.SetSidebarHidden(!uiState.Sidebar.Visible)

	return &App{
		store:             store,
		sessionName:       sessionName,
		workDir:           workDir,
		dataDir:           dataDir,
		nc:                nc,
		ctx:               ctx,
		sendChan:          sendChan,
		orchestrator:      orch,
		sidebarVisible:    uiState.Sidebar.Visible, // Load from persistent state
		sidebarUserHidden: false,                   // Initialize as not user-hidden
		dashboard:         NewDashboard(agent, sidebar),
		logs:              NewLogViewer(),
		agent:             agent,
		status:            statusBar,
		sidebar:           sidebar,
		dialog:            NewDialog(),
		taskModal:         NewTaskModal(),
		noteModal:         NewNoteModal(),
		noteInputModal:    NewNoteInputModal(),
		taskInputModal:    NewTaskInputModal(),
		toast:             NewToast(),
		eventChan:         make(chan session.Event, 1000), // Buffered channel for events (needs capacity for large task batches)
		layoutDirty:       true,                           // Calculate layout on first render
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
		a.fetchGitInfo(), // Fetch git repository status on startup
	)
}

// Update handles incoming messages and updates the model state.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return a.handleKeyPress(msg)

	case tea.PasteMsg:
		return a.handlePaste(msg)

	case tea.MouseClickMsg:
		return a.handleMouse(msg)

	case tea.MouseWheelMsg:
		return a.handleMouseWheel(msg)

	case tea.WindowSizeMsg:
		oldWidth := a.width
		a.width = msg.Width
		a.height = msg.Height
		a.layoutDirty = true

		// Handle responsive sidebar visibility based on width changes
		wasWide := oldWidth >= CompactWidthBreakpoint
		isWide := a.width >= CompactWidthBreakpoint

		if wasWide && !isWide {
			// Narrowing below threshold: auto-hide if not already user-hidden
			if !a.sidebarUserHidden && a.sidebarVisible {
				a.sidebarVisible = false
				// Don't set sidebarUserHidden - this is auto-hide
				a.saveUIState()
				a.status.SetSidebarHidden(true)
			}
		} else if !wasWide && isWide {
			// Widening past threshold: auto-restore only if not user-hidden
			if !a.sidebarUserHidden && !a.sidebarVisible {
				a.sidebarVisible = true
				a.saveUIState()
				a.status.SetSidebarHidden(false)
			}
		}

		// Recalculate layout and propagate sizes
		a.layout = CalculateLayout(a.width, a.height, !a.sidebarVisible)
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
		// Also notify status bar that agent is no longer busy (for pause display)
		statusCmd := func() tea.Msg { return AgentBusyMsg{Busy: false} }
		return a, tea.Batch(a.agent.AppendFinish(msg), queueCmd, statusCmd)

	case HookStartMsg:
		return a, a.agent.AppendHook(msg.HookID, msg.HookType, msg.Command)

	case HookCompleteMsg:
		return a, a.agent.UpdateHook(msg.HookID, msg.Status, msg.Output, msg.Duration)

	case IterationStartMsg:
		a.iteration = msg.Number // Track current iteration for note creation
		a.modifiedFileCount = 0  // Reset modified file count for new iteration
		a.status.SetModifiedFileCount(0)
		a.lastGitCheck = time.Now() // Update throttle timestamp
		busyCmd := a.dashboard.SetAgentBusy(true)
		// Also notify status bar that agent is busy (for pause display)
		statusCmd := func() tea.Msg { return AgentBusyMsg{Busy: true} }
		return a, tea.Batch(
			a.dashboard.SetIteration(msg.Number),
			a.agent.AddIterationDivider(msg.Number),
			busyCmd,
			statusCmd,
			a.fetchGitInfo(), // Refresh git status at iteration start
		)

	case StateUpdateMsg:
		// Propagate state updates to all components
		a.status.SetState(msg.State)
		a.sidebar.SetState(msg.State)
		a.dashboard.SetState(msg.State)
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
		// Stop the duration timer
		a.status.StopDurationTick()
		// Show completion dialog - user can dismiss and continue viewing or quit manually
		a.dialog.Show(
			"Session Complete",
			"All tasks have been completed successfully!",
			nil, // Just close the modal, don't quit
		)
		// Reload state to ensure UI shows latest task counts
		// (SessionCompleteMsg is sent directly by orchestrator, bypassing NATS event flow)
		return a, a.loadInitialState()

	case UserInputMsg:
		// Handle user input from the text field - send to orchestrator queue
		// Increment queue depth and send to channel
		a.queueDepth++
		if a.sendChan != nil {
			select {
			case a.sendChan <- msg.Text:
				// Message queued successfully - show as queued in viewport immediately
				var cmds []tea.Cmd
				if a.dashboard != nil && a.dashboard.agentOutput != nil {
					_, cmd := a.dashboard.agentOutput.AppendQueuedUserMessage(msg.Text)
					cmds = append(cmds, cmd)
				}
				if a.dashboard != nil {
					cmds = append(cmds, a.dashboard.SetQueueDepth(a.queueDepth))
				}
				return a, tea.Batch(cmds...)
			default:
				// Channel full - message dropped
				a.queueDepth-- // Revert increment since message wasn't queued
				logger.Warn("sendChan full, message dropped: %s", msg.Text)
				// TODO: Show visual feedback to user (toast/status message)
			}
		}
		return a, nil

	case QueuedMessageProcessingMsg:
		// User message delivered - finalize the queued message in viewport (remove QUEUED badge)
		a.queueDepth--
		if a.queueDepth < 0 {
			a.queueDepth = 0
		}

		var cmds []tea.Cmd

		// Finalize the oldest queued message (converts QueuedUserMessageItem → UserMessageItem)
		if a.dashboard != nil && a.dashboard.agentOutput != nil {
			cmd := a.dashboard.agentOutput.FinalizeQueuedMessage(msg.Text)
			cmds = append(cmds, cmd)
		}

		// Update queue depth display
		if a.dashboard != nil {
			cmd := a.dashboard.SetQueueDepth(a.queueDepth)
			cmds = append(cmds, cmd)
		}

		return a, tea.Batch(cmds...)

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
				// TODO: Add visual feedback for user
				logger.Warn("failed to add note: %v", err)
			}
		}()
		// Close the modal after submitting
		a.noteInputModal.Close()
		return a, nil

	case CreateTaskMsg:
		// Create a new task via Store.TaskAdd()
		// The task will be published to NATS and picked up by event subscription
		// Use App's iteration field (set by IterationStartMsg) instead of message field
		iteration := a.iteration
		if msg.Iteration != 0 {
			iteration = msg.Iteration // Allow override if explicitly set
		}
		go func() {
			_, err := a.store.TaskAdd(a.ctx, a.sessionName, session.TaskAddParams{
				Content:   msg.Content,
				Priority:  msg.Priority,
				Iteration: iteration,
			})
			if err != nil {
				// TODO: Add visual feedback for user
				logger.Warn("failed to add task: %v", err)
			}
		}()
		// Close the modal after submitting
		a.taskInputModal.Close()
		return a, nil

	case FileChangeMsg:
		// Increment modified file count when a file is modified
		a.modifiedFileCount++
		// Update status bar to reflect new count
		a.status.SetModifiedFileCount(a.modifiedFileCount)

		// Check git info with throttling: skip if < 500ms since last check,
		// but always check on first file change of iteration (modifiedFileCount == 1)
		var gitCmd tea.Cmd
		now := time.Now()
		if a.modifiedFileCount == 1 || now.Sub(a.lastGitCheck) >= 500*time.Millisecond {
			a.lastGitCheck = now
			gitCmd = a.fetchGitInfo()
		}

		return a, tea.Batch(a.status.Tick(), gitCmd)

	case OpenSubagentModalMsg:
		// Close existing modal if any (shouldn't happen with full-screen modal)
		if a.subagentModal != nil {
			a.subagentModal.Close()
		}
		// Create and start new subagent modal
		modal := NewSubagentModal(msg.SessionID, msg.SubagentType, a.workDir)
		a.subagentModal = modal
		return a, modal.Start() // Spawns ACP, loads session, starts streaming (TAS-16)

	case SubagentTextMsg, SubagentToolCallMsg, SubagentThinkingMsg, SubagentUserMsg, SubagentStreamMsg:
		// Forward streaming messages to modal (TAS-17, TAS-18)
		if a.subagentModal != nil {
			return a, a.subagentModal.HandleUpdate(msg)
		}

	case SubagentDoneMsg:
		// All history replayed - modal stays open for viewing until user presses ESC
		// No action needed

	case SubagentErrorMsg:
		if a.subagentModal != nil {
			a.subagentModal.err = msg.Err
		}

	case ShowToastMsg:
		return a, a.toast.Show(msg.Text)
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

	// Update toast (for auto-dismiss timing)
	toastCmd := a.toast.Update(msg)

	return a, tea.Batch(statusCmd, sidebarCmd, dashCmd, logsCmd, toastCmd)
}

// handleKeyPress processes keyboard input using hierarchical priority routing.
// Priority: Global Keys (ctrl+c) → Dialog → Prefix Mode → Modal → View → Focus → Component
func (a *App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 0. Global keys (ctrl+x, ctrl+c) - must work everywhere, even with dialog open
	if cmd := a.handleGlobalKeys(msg); cmd != nil {
		return a, cmd
	}

	// 1. Dialog gets priority when visible (after global keys)
	if a.dialog.IsVisible() {
		if cmd := a.dialog.Update(msg); cmd != nil {
			return a, cmd
		}
		return a, nil // Consume all keys when dialog is visible
	}

	// 2. Handle prefix key sequences (ctrl+x followed by another key)
	if a.awaitingPrefixKey {
		a.awaitingPrefixKey = false // Exit prefix mode after handling
		a.status.SetPrefixMode(false)

		switch msg.String() {
		case "l":
			// ctrl+x l -> toggle logs
			a.logsVisible = !a.logsVisible
			return a, nil
		case "b":
			// ctrl+x b -> toggle sidebar
			return a, a.handleSidebarToggle()
		case "n":
			// ctrl+x n -> create note
			if a.dialog.IsVisible() || a.taskModal.IsVisible() || a.noteModal.IsVisible() ||
				a.noteInputModal.IsVisible() || a.taskInputModal.IsVisible() || a.logsVisible {
				return a, nil
			}
			if a.iteration == 0 {
				return a, nil
			}
			return a, a.noteInputModal.Show()
		case "t":
			// ctrl+x t -> create task
			if a.dialog.IsVisible() || a.taskModal.IsVisible() || a.noteModal.IsVisible() ||
				a.noteInputModal.IsVisible() || a.taskInputModal.IsVisible() || a.logsVisible {
				return a, nil
			}
			if a.iteration == 0 {
				return a, nil
			}
			return a, a.taskInputModal.Show()
		case "p":
			// ctrl+x p -> toggle pause/resume
			return a, a.togglePause()
		case "ctrl+c", "esc":
			// Allow escape or ctrl+c to exit prefix mode
			return a, nil
		default:
			// Any other key exits prefix mode without action
			return a, nil
		}
	}

	// 3. Modal gets priority when visible
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

	// Task input modal gets priority when visible
	if a.taskInputModal != nil && a.taskInputModal.IsVisible() {
		return a, a.taskInputModal.Update(msg)
	}

	// Subagent modal gets priority when visible
	if a.subagentModal != nil {
		// ESC key closes the modal
		if msg.String() == "esc" {
			a.subagentModal.Close()
			a.subagentModal = nil
			return a, nil
		}
		// Forward scroll keys to modal
		return a, a.subagentModal.Update(msg)
	}

	// 4. Logs modal captures remaining keys when visible
	if a.logsVisible {
		switch msg.String() {
		case "esc":
			a.logsVisible = false
			return a, nil
		default:
			// Forward scroll keys to log viewport
			return a, a.logs.Update(msg)
		}
	}

	// 5. Delegate to dashboard for focused component handling
	return a, a.dashboard.Update(msg)
}

// handlePaste processes paste messages using hierarchical priority routing.
// Mirrors the priority from handleKeyPress: noteInputModal → taskInputModal →
// subagentModal → dashboard.
func (a *App) handlePaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	// Sanitize pasted content
	content := SanitizePaste(msg.Content)

	// 1. Dialog has no text input — consume paste
	if a.dialog.IsVisible() {
		return a, nil
	}

	// 2. Read-only modals have no text input — consume paste
	if a.taskModal != nil && a.taskModal.IsVisible() {
		return a, nil
	}
	if a.noteModal != nil && a.noteModal.IsVisible() {
		return a, nil
	}

	// 3. Note input modal gets priority when visible
	if a.noteInputModal != nil && a.noteInputModal.IsVisible() {
		return a, a.noteInputModal.Update(tea.PasteMsg{Content: content})
	}

	// 4. Task input modal gets priority when visible
	if a.taskInputModal != nil && a.taskInputModal.IsVisible() {
		return a, a.taskInputModal.Update(tea.PasteMsg{Content: content})
	}

	// 5. Subagent modal gets priority when visible
	if a.subagentModal != nil {
		return a, a.subagentModal.Update(tea.PasteMsg{Content: content})
	}

	// 6. Logs has no text input, so paste is no-op when logs are visible
	if a.logsVisible {
		return a, nil
	}

	// 7. Delegate to dashboard for focused component handling
	return a, a.dashboard.Update(tea.PasteMsg{Content: content})
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

	// Subagent modal takes priority when visible - handle clicks for expand/collapse
	if a.subagentModal != nil {
		// Handle click within modal (for expand/collapse on messages)
		a.subagentModal.HandleClick(mouse.X, mouse.Y)
		// All clicks are consumed when modal is visible
		return a, nil
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

	// Check if agent output was clicked (expand/collapse tool output or open subagent modal)
	if cmd := a.agent.HandleClick(mouse.X, mouse.Y); cmd != nil {
		return a, cmd
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

	// Subagent modal takes priority - scroll modal content when visible
	if a.subagentModal != nil {
		a.subagentModal.ScrollViewport(lines)
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
	case "ctrl+x":
		// Enter prefix mode - wait for next key
		a.awaitingPrefixKey = true
		a.status.SetPrefixMode(true)
		// Return a no-op command to signal we handled this key
		return func() tea.Msg { return nil }
	case "ctrl+c":
		a.quitting = true
		return tea.Quit
	}
	return nil
}

// togglePause handles the ctrl+x p keyboard shortcut to toggle pause/resume.
// Behavior depends on current state:
// - If not paused: request pause (will take effect after current iteration)
// - If paused and agent still working: cancel pause request
// - If paused and agent blocked: resume immediately
func (a *App) togglePause() tea.Cmd {
	// Guard: if orchestrator is nil, do nothing
	if a.orchestrator == nil {
		return nil
	}

	paused := a.orchestrator.IsPaused()
	working := a.dashboard != nil && a.dashboard.agentBusy

	if !paused {
		// Not paused -> request pause
		a.orchestrator.RequestPause()
		return func() tea.Msg {
			return PauseStateMsg{Paused: true}
		}
	} else if working {
		// Paused but still working -> cancel pause request
		a.orchestrator.CancelPause()
		return func() tea.Msg {
			return PauseStateMsg{Paused: false}
		}
	} else {
		// Paused and blocked -> resume
		a.orchestrator.Resume()
		return func() tea.Msg {
			return PauseStateMsg{Paused: false}
		}
	}
}

// handleSidebarToggle toggles the sidebar visibility and manages focus and persistence.
// When hiding: moves focus from sidebar to messages panel and sets user-hidden flag.
// When showing: restores sidebar and clears user-hidden flag.
func (a *App) handleSidebarToggle() tea.Cmd {
	// Toggle visibility
	a.sidebarVisible = !a.sidebarVisible

	// Set user-hidden flag when manually toggling
	a.sidebarUserHidden = !a.sidebarVisible

	// Persist state
	a.saveUIState()

	// Update status bar to show/hide sidebar hint
	a.status.SetSidebarHidden(!a.sidebarVisible)

	// Move focus to messages panel if hiding sidebar while focus is on sidebar
	if !a.sidebarVisible && a.dashboard != nil {
		if a.dashboard.focusPane == FocusTasks || a.dashboard.focusPane == FocusNotes {
			a.dashboard.focusPane = FocusAgent
			a.dashboard.updateScrollListFocus()
		}
	}

	// Mark layout dirty to trigger recalculation
	a.layoutDirty = true

	return nil
}

// saveUIState persists the current UI state to disk.
func (a *App) saveUIState() {
	uiState := &state.UIState{
		Sidebar: state.SidebarState{
			Visible: a.sidebarVisible,
		},
	}
	if err := state.Save(a.dataDir, uiState); err != nil {
		logger.Warn("failed to save UI state: %v", err)
	}
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
		a.layout = CalculateLayout(a.width, a.height, !a.sidebarVisible)
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

	// Set global background color for the entire terminal
	view.BackgroundColor = theme.HexToColor(theme.Current().BgCrust)

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
	if a.subagentModal != nil {
		a.subagentModal.Draw(scr, area)
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
	if a.taskInputModal.IsVisible() {
		a.taskInputModal.Draw(scr, area)
	}
	if a.dialog.IsVisible() {
		a.dialog.Draw(scr, area)
	}

	// Draw toast last so it appears on top of everything
	if a.toast.IsVisible() {
		toastContent := a.toast.View(area.Dx(), area.Dy())
		if toastContent != "" {
			// Calculate position at bottom-right with 1 cell padding from edges
			// Position above the status bar (area.Max.Y - 1 - contentHeight)
			contentWidth := lipglossv2.Width(toastContent)
			contentHeight := lipglossv2.Height(toastContent)
			toastX := area.Max.X - contentWidth - 1
			toastY := area.Max.Y - 1 - contentHeight
			if toastX < area.Min.X {
				toastX = area.Min.X
			}
			if toastY < area.Min.Y {
				toastY = area.Min.Y
			}
			toastArea := uv.Rectangle{
				Min: uv.Position{X: toastX, Y: toastY},
				Max: uv.Position{X: toastX + contentWidth, Y: toastY + contentHeight},
			}
			uv.NewStyledString(toastContent).Draw(scr, toastArea)
		}
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
	SessionID  string // Session ID for subagent tasks (from rawOutput.metadata.sessionId)
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

// QueuedMessageProcessingMsg is sent by the orchestrator when it starts processing a queued message.
type QueuedMessageProcessingMsg struct {
	Text string
}

// QueueDepthMsg is sent to update the UI with the current queue depth.
type QueueDepthMsg struct {
	Depth int
}

// CreateNoteMsg is sent when the user creates a new note from the input modal.
type CreateNoteMsg struct {
	Content   string
	NoteType  string
	Iteration int
}

// CreateTaskMsg is sent when the user submits a task from the task input modal.
type CreateTaskMsg struct {
	Content   string
	Priority  int
	Iteration int
}

// HookStartMsg is sent when a hook command starts executing.
type HookStartMsg struct {
	HookID   string // Unique ID for this hook execution
	HookType string // "session_start", "pre_iteration", etc.
	Command  string // The shell command being run
}

// HookCompleteMsg is sent when a hook command finishes executing.
type HookCompleteMsg struct {
	HookID   string        // Matches the HookStartMsg.HookID
	Status   HookStatus    // Success or Error
	Output   string        // Command output (stdout + stderr)
	Duration time.Duration // How long the hook took
}

// FileChangeMsg is sent when a file is modified during an iteration.
type FileChangeMsg struct {
	Path      string
	IsNew     bool
	Additions int
	Deletions int
}

// OpenSubagentModalMsg is sent when the user clicks a subagent message item with a sessionID.
type OpenSubagentModalMsg struct {
	SessionID    string
	SubagentType string
}

// SubagentTextMsg is sent when the subagent modal receives an agent_message_chunk during session replay.
type SubagentTextMsg struct {
	Text     string
	Continue bool // True to continue streaming, false if done
}

// SubagentToolCallMsg is sent when the subagent modal receives a tool_call or tool_call_update during session replay.
type SubagentToolCallMsg struct {
	Event    agent.ToolCallEvent
	Continue bool // True to continue streaming, false if done
}

// SubagentThinkingMsg is sent when the subagent modal receives an agent_thought_chunk during session replay.
type SubagentThinkingMsg struct {
	Content  string
	Continue bool // True to continue streaming, false if done
}

// SubagentUserMsg is sent when the subagent modal receives a user_message_chunk during session replay.
type SubagentUserMsg struct {
	Text     string
	Continue bool // True to continue streaming, false if done
}

// SubagentDoneMsg is sent when the subagent modal finishes replaying the session (EOF reached).
type SubagentDoneMsg struct{}

// SubagentErrorMsg is sent when the subagent modal encounters an error during session loading or streaming.
type SubagentErrorMsg struct {
	Err error
}

// SubagentStreamMsg is sent to continue streaming when an unknown notification is received.
type SubagentStreamMsg struct {
	Continue bool // Always true to continue streaming
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

	// Propagate sidebar size based on layout mode and visibility
	if a.layout.Mode == LayoutDesktop && a.sidebarVisible {
		// Desktop mode with sidebar visible: use dedicated sidebar area
		a.sidebar.SetSize(a.layout.Sidebar.Dx(), a.layout.Sidebar.Dy())
	} else if a.layout.Mode == LayoutCompact {
		// Compact mode: sidebar overlays with fixed width
		sidebarWidth := SidebarWidthDesktop
		if a.layout.Main.Dx()/2 < sidebarWidth {
			sidebarWidth = a.layout.Main.Dx() / 2
		}
		a.sidebar.SetSize(sidebarWidth, a.layout.Main.Dy())
	}
	// If desktop mode with sidebar hidden, skip size propagation to sidebar
}

// fetchGitInfo returns a command that fetches git repository status
// and sends a GitInfoMsg to update the status bar.
func (a *App) fetchGitInfo() tea.Cmd {
	return func() tea.Msg {
		info, err := git.GetInfo(a.workDir)
		if err != nil || info == nil {
			// Not a git repo or error - mark as invalid
			return GitInfoMsg{Valid: false}
		}
		return GitInfoMsg{
			Branch: info.Branch,
			Hash:   info.Hash,
			Dirty:  info.Dirty,
			Ahead:  info.Ahead,
			Behind: info.Behind,
			Valid:  true,
		}
	}
}
