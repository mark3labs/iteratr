package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/nats-io/nats.go"
)

// ViewType represents the different views in the TUI
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewTasks
	ViewLogs
	ViewNotes
	ViewInbox
)

// App is the main Bubbletea model that manages the TUI application.
// It contains all view components and handles routing between them.
type App struct {
	// View components
	dashboard *Dashboard
	tasks     *TaskList
	logs      *LogViewer
	notes     *NotesPanel
	inbox     *InboxPanel
	agent     *AgentOutput

	// State
	activeView  ViewType
	store       *session.Store
	sessionName string
	nc          *nats.Conn
	ctx         context.Context
	width       int
	height      int
	quitting    bool
}

// NewApp creates a new TUI application with the given session store and NATS connection.
func NewApp(ctx context.Context, store *session.Store, sessionName string, nc *nats.Conn) *App {
	return &App{
		store:       store,
		sessionName: sessionName,
		nc:          nc,
		ctx:         ctx,
		activeView:  ViewDashboard,
		dashboard:   NewDashboard(),
		tasks:       NewTaskList(),
		logs:        NewLogViewer(),
		notes:       NewNotesPanel(),
		inbox:       NewInboxPanel(),
		agent:       NewAgentOutput(),
	}
}

// Init initializes the application and returns any initial commands.
// In Bubbletea v2, Init returns only tea.Cmd (not Model).
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.subscribeToEvents(),
		a.loadInitialState(),
		a.agent.Init(),
	)
}

// Update handles incoming messages and updates the model state.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return a.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Propagate size to all views
		return a, tea.Batch(
			a.dashboard.UpdateSize(msg.Width, msg.Height),
			a.tasks.UpdateSize(msg.Width, msg.Height),
			a.logs.UpdateSize(msg.Width, msg.Height),
			a.notes.UpdateSize(msg.Width, msg.Height),
			a.inbox.UpdateSize(msg.Width, msg.Height),
			a.agent.UpdateSize(msg.Width, msg.Height),
		)

	case AgentOutputMsg:
		return a, a.agent.Append(msg.Content)

	case IterationStartMsg:
		return a, a.dashboard.SetIteration(msg.Number)

	case StateUpdateMsg:
		// Propagate state updates to all views
		return a, tea.Batch(
			a.dashboard.UpdateState(msg.State),
			a.tasks.UpdateState(msg.State),
			a.logs.UpdateState(msg.State),
			a.notes.UpdateState(msg.State),
			a.inbox.UpdateState(msg.State),
		)
	}

	// Delegate to active view component
	var cmd tea.Cmd
	switch a.activeView {
	case ViewDashboard:
		cmd = a.dashboard.Update(msg)
	case ViewTasks:
		cmd = a.tasks.Update(msg)
	case ViewLogs:
		cmd = a.logs.Update(msg)
	case ViewNotes:
		cmd = a.notes.Update(msg)
	case ViewInbox:
		cmd = a.inbox.Update(msg)
	}

	return a, cmd
}

// handleKeyPress processes keyboard input for navigation and control.
func (a *App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// Global navigation keys
	switch k {
	case "1":
		a.activeView = ViewDashboard
		return a, nil
	case "2":
		a.activeView = ViewTasks
		return a, nil
	case "3":
		a.activeView = ViewLogs
		return a, nil
	case "4":
		a.activeView = ViewNotes
		return a, nil
	case "5":
		a.activeView = ViewInbox
		return a, nil
	case "q", "ctrl+c":
		a.quitting = true
		return a, tea.Quit
	}

	return a, nil
}

// View renders the current view. In Bubbletea v2, this returns tea.View
// with display options like AltScreen and MouseMode.
func (a *App) View() tea.View {
	if a.quitting {
		v := tea.NewView("Goodbye!\n")
		return v
	}

	// Render header, content, and footer
	header := a.renderHeader()
	content := a.renderActiveView()
	footer := a.renderFooter()

	// Join vertically with lipgloss
	output := header + "\n" + content + "\n" + footer

	// Create view with display options
	v := tea.NewView(output)
	v.AltScreen = true                    // Full-screen mode
	v.MouseMode = tea.MouseModeCellMotion // Enable mouse events
	v.ReportFocus = true                  // Enable focus events
	return v
}

// renderHeader renders the top header bar with session info and navigation.
func (a *App) renderHeader() string {
	// TODO: Implement with lipgloss styles
	return "iteratr | " + a.sessionName
}

// renderActiveView renders the currently active view component.
func (a *App) renderActiveView() string {
	switch a.activeView {
	case ViewDashboard:
		return a.dashboard.Render()
	case ViewTasks:
		return a.tasks.Render()
	case ViewLogs:
		return a.logs.Render()
	case ViewNotes:
		return a.notes.Render()
	case ViewInbox:
		return a.inbox.Render()
	default:
		return "Unknown view"
	}
}

// renderFooter renders the bottom footer bar with navigation hints.
func (a *App) renderFooter() string {
	// TODO: Implement with lipgloss styles
	return "[1] Dashboard [2] Tasks [3] Logs [4] Notes [5] Inbox    q=quit"
}

// subscribeToEvents subscribes to NATS events for this session.
// This runs in a managed goroutine and sends messages to the Update loop.
func (a *App) subscribeToEvents() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement NATS subscription
		// Subscribe to iteratr.{session}.> and forward events to Update loop
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

// Custom message types for the TUI
type AgentOutputMsg struct {
	Content string
}

type IterationStartMsg struct {
	Number int
}

type StateUpdateMsg struct {
	State *session.State
}
