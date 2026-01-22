package tui

import (
	"context"
	"encoding/json"
	"fmt"

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
	inbox     *InboxPanel
	agent     *AgentOutput
	header    *Header
	footer    *Footer
	status    *StatusBar

	// Layout management
	layout      Layout
	layoutDirty bool

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
		header:      NewHeader(sessionName),
		footer:      NewFooter(),
		status:      NewStatusBar(),
		eventChan:   make(chan session.Event, 100), // Buffered channel for events
		layoutDirty: true,                          // Calculate layout on first render
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
		a.layoutDirty = true
		// Recalculate layout and propagate sizes
		a.layout = CalculateLayout(a.width, a.height)
		a.propagateSizes()
		a.layoutDirty = false
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
		// Propagate state updates to all components
		a.header.SetState(msg.State)
		a.status.SetState(msg.State)
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
	var view tea.View
	view.AltScreen = true                    // Full-screen mode
	view.MouseMode = tea.MouseModeCellMotion // Enable mouse events
	view.ReportFocus = true                  // Enable focus events

	if a.quitting {
		// Return minimal view when quitting - don't render full UI
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

	// Convert canvas to lipgloss Layer
	view.Content = lipglossv2.NewLayer(canvas.Render())

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

	// TODO: Draw sidebar in desktop mode (Phase 14)
	// if a.layout.Mode == LayoutDesktop {
	// 	a.sidebar.Draw(scr, a.layout.Sidebar)
	// }

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
	case ViewInbox:
		return a.inbox.Draw(scr, area)
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

// propagateSizes updates component sizes based on the current layout.
// This is called when the layout changes (on window resize or mode switch).
func (a *App) propagateSizes() {
	// Propagate sizes to header, footer, and status bar
	a.header.SetSize(a.layout.Header.Dx(), a.layout.Header.Dy())
	a.footer.SetSize(a.layout.Footer.Dx(), a.layout.Footer.Dy())
	a.status.SetSize(a.layout.Status.Dx(), a.layout.Status.Dy())

	// Propagate sizes to main content components
	// Note: dashboard owns the agent output component, so we only size the dashboard
	a.dashboard.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())
	a.logs.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())
	a.notes.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())
	a.inbox.SetSize(a.layout.Main.Dx(), a.layout.Main.Dy())
}
