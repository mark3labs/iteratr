package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
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
	eventChan   chan session.Event // Channel for receiving NATS events
}

// NewApp creates a new TUI application with the given session store and NATS connection.
func NewApp(ctx context.Context, store *session.Store, sessionName string, nc *nats.Conn) *App {
	agent := NewAgentOutput()
	return &App{
		store:       store,
		sessionName: sessionName,
		nc:          nc,
		ctx:         ctx,
		activeView:  ViewDashboard,
		dashboard:   NewDashboard(agent), // Pass agent output to dashboard
		logs:        NewLogViewer(),
		notes:       NewNotesPanel(),
		inbox:       NewInboxPanel(),
		agent:       agent,
		eventChan:   make(chan session.Event, 100), // Buffered channel for events
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
		// Calculate content height (minus header, status bar, and footer)
		contentHeight := msg.Height - 5 // header(1) + statusbar(1) + footer(1) + padding(2)
		if contentHeight < 5 {
			contentHeight = 5
		}
		// Propagate size to all views
		// Note: agent output is sized by dashboard (it owns the agent component)
		a.dashboard.UpdateSize(msg.Width, contentHeight)
		a.logs.SetSize(msg.Width, contentHeight)
		a.notes.UpdateSize(msg.Width, contentHeight)
		a.inbox.UpdateSize(msg.Width, contentHeight)
		return a, nil

	case AgentOutputMsg:
		return a, a.agent.AppendText(msg.Content)

	case AgentToolMsg:
		return a, a.agent.AppendTool(msg.Tool, msg.Input)

	case IterationStartMsg:
		return a, tea.Batch(
			a.dashboard.SetIteration(msg.Number),
			a.agent.AddIterationDivider(msg.Number),
		)

	case StateUpdateMsg:
		// Propagate state updates to all views
		a.dashboard.UpdateState(msg.State)
		a.logs.SetState(msg.State)
		a.notes.UpdateState(msg.State)
		a.inbox.UpdateState(msg.State)
		return a, nil

	case EventMsg:
		// Forward event to log viewer, reload state, and wait for next event
		return a, tea.Batch(
			a.logs.AddEvent(msg.Event),
			a.loadInitialState(), // Reload state to reflect changes
			a.waitForEvents(),    // Recursively wait for next event
		)
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
		a.activeView = ViewLogs
		return a, nil
	case "3":
		a.activeView = ViewNotes
		return a, nil
	case "4":
		a.activeView = ViewInbox
		return a, nil
	case "q", "ctrl+c":
		a.quitting = true
		return a, tea.Quit
	}

	// Forward unhandled keys to active view for scrolling etc.
	var cmd tea.Cmd
	switch a.activeView {
	case ViewDashboard:
		cmd = a.dashboard.Update(msg)
	case ViewLogs:
		cmd = a.logs.Update(msg)
	case ViewNotes:
		cmd = a.notes.Update(msg)
	case ViewInbox:
		cmd = a.inbox.Update(msg)
	}

	return a, cmd
}

// View renders the current view. In Bubbletea v2, this returns tea.View
// with display options like AltScreen and MouseMode.
func (a *App) View() tea.View {
	if a.quitting {
		// Return minimal view when quitting - don't render full UI
		return tea.NewView("")
	}

	// Render header, content, status bar, and footer
	header := a.renderHeader()
	content := a.renderActiveView()
	statusBar := a.renderStatusBar()
	footer := a.renderFooter()

	// Join vertically with lipgloss
	output := header + "\n" + content + "\n" + statusBar + "\n" + footer

	// Create view with display options
	v := tea.NewView(output)
	v.AltScreen = true                    // Full-screen mode
	v.MouseMode = tea.MouseModeCellMotion // Enable mouse events
	v.ReportFocus = true                  // Enable focus events
	return v
}

// renderStatusBar renders the task status bar above the footer.
func (a *App) renderStatusBar() string {
	if a.dashboard == nil || a.dashboard.state == nil {
		return styleStatusBar.Width(a.width).Render("")
	}

	// Get task stats from dashboard
	stats := a.dashboard.getTaskStats()

	var parts []string

	if stats.Remaining > 0 {
		parts = append(parts, styleStatusRemaining.Render(fmt.Sprintf("%d remaining", stats.Remaining)))
	}
	if stats.InProgress > 0 {
		parts = append(parts, styleStatusInProgress.Render(fmt.Sprintf("%d in progress", stats.InProgress)))
	}
	if stats.Completed > 0 {
		parts = append(parts, styleStatusCompleted.Render(fmt.Sprintf("%d completed", stats.Completed)))
	}
	if stats.Blocked > 0 {
		parts = append(parts, styleStatusBlocked.Render(fmt.Sprintf("%d blocked", stats.Blocked)))
	}

	if len(parts) == 0 {
		return styleStatusBar.Width(a.width).Render(styleDim.Render("No tasks"))
	}

	separator := styleDim.Render(" | ")
	statusText := strings.Join(parts, separator)

	return styleStatusBar.Width(a.width).Render(statusText)
}

// renderHeader renders the top header bar with session info and navigation.
func (a *App) renderHeader() string {
	// Build header components
	title := styleHeaderTitle.Render("iteratr")
	sep := styleHeaderSeparator.Render(" | ")
	session := styleHeaderInfo.Render(a.sessionName)

	// Get current iteration number
	iteration := ""
	if a.dashboard != nil && a.dashboard.iteration > 0 {
		iteration = styleHeaderSeparator.Render(" | ") +
			styleHeaderInfo.Render(fmt.Sprintf("Iteration #%d", a.dashboard.iteration))
	}

	// Build view tabs
	tabs := a.renderViewTabs()

	// Left side: title + session + iteration
	left := title + sep + session + iteration

	// Right side: view tabs
	right := tabs

	// Calculate spacing to fill width
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := a.width - leftWidth - rightWidth - 2 // -2 for side padding
	if padding < 1 {
		padding = 1
	}

	// Join with spacing
	header := left + strings.Repeat(" ", padding) + right

	// Apply header style and fill width
	return styleHeader.Width(a.width).Render(header)
}

// renderActiveView renders the currently active view component.
func (a *App) renderActiveView() string {
	switch a.activeView {
	case ViewDashboard:
		return a.dashboard.Render()
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

// renderViewTabs renders the view navigation tabs for the header.
func (a *App) renderViewTabs() string {
	views := []struct {
		key  string
		name string
		view ViewType
	}{
		{"1", "Dashboard", ViewDashboard},
		{"2", "Logs", ViewLogs},
		{"3", "Notes", ViewNotes},
		{"4", "Inbox", ViewInbox},
	}

	var tabs []string
	for _, v := range views {
		if v.view == a.activeView {
			// Active view - highlight
			tabs = append(tabs, styleFooterActive.Render(fmt.Sprintf("[%s]", v.key)))
		} else {
			// Inactive view - dim
			tabs = append(tabs, styleFooterKey.Render(v.key))
		}
	}

	return strings.Join(tabs, " ")
}

// renderFooter renders the bottom footer bar with navigation hints.
func (a *App) renderFooter() string {
	// Build footer components
	var parts []string

	// View navigation
	views := []struct {
		key  string
		name string
		view ViewType
	}{
		{"1", "Dashboard", ViewDashboard},
		{"2", "Logs", ViewLogs},
		{"3", "Notes", ViewNotes},
		{"4", "Inbox", ViewInbox},
	}

	for _, v := range views {
		key := styleFooterKey.Render(fmt.Sprintf("[%s]", v.key))
		label := styleFooterLabel.Render(v.name)
		if v.view == a.activeView {
			// Highlight active view
			label = styleFooterActive.Render(v.name)
		}
		parts = append(parts, key+" "+label)
	}

	// Add tab hint for sidebar focus
	tabHint := styleFooterKey.Render("tab") + styleFooterLabel.Render("=tasks")
	parts = append(parts, tabHint)

	// Add quit hint on the right
	footer := strings.Join(parts, "  ")
	quit := styleFooterKey.Render("q") + styleFooterLabel.Render("=quit")

	// Calculate spacing
	footerWidth := lipgloss.Width(footer)
	quitWidth := lipgloss.Width(quit)
	padding := a.width - footerWidth - quitWidth - 2 // -2 for side padding
	if padding < 2 {
		padding = 2
	}

	footer = footer + strings.Repeat(" ", padding) + quit

	// Apply footer style and fill width
	return styleFooter.Width(a.width).Render(footer)
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

// Custom message types for the TUI
type AgentOutputMsg struct {
	Content string
}

type AgentToolMsg struct {
	Tool  string
	Input map[string]any
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
