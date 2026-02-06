package tui

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// Compile-time interface checks
var _ FocusableComponent = (*Dashboard)(nil)

// FocusPane represents which pane of the dashboard has keyboard focus.
type FocusPane int

const (
	FocusAgent FocusPane = iota
	FocusTasks
	FocusNotes
	FocusInput
)

// Dashboard displays session overview, progress, and current task.
type Dashboard struct {
	sessionName  string
	iteration    int
	state        *session.State
	width        int
	height       int
	agentOutput  *AgentOutput // Reference to agent output for rendering
	sidebar      *Sidebar     // Sidebar on the right (tasks + notes)
	focusPane    FocusPane    // Which pane has keyboard focus
	focused      bool         // Whether the dashboard has focus
	inputFocused bool         // Whether the input field is focused
	agentBusy    bool         // Whether the agent is currently processing (used for input placeholder)
}

// NewDashboard creates a new Dashboard component.
// The sidebar parameter is shared with App to ensure keyboard navigation
// and rendering operate on the same instance.
func NewDashboard(agentOutput *AgentOutput, sidebar *Sidebar) *Dashboard {
	return &Dashboard{
		agentOutput: agentOutput,
		sidebar:     sidebar,
		focusPane:   FocusAgent,
	}
}

// Update handles messages for the dashboard.
func (d *Dashboard) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		// Forward spinner ticks to AgentOutput for subagent spinner animation
		if d.agentOutput != nil {
			return d.agentOutput.Update(msg)
		}
		return nil
	case tea.PasteMsg:
		// Forward paste messages to AgentOutput when input is focused
		if d.focusPane == FocusInput && d.agentOutput != nil {
			return d.agentOutput.Update(msg)
		}
		return nil

	case tea.KeyPressMsg:
		// Global 'i' key: focus input from any state
		if msg.String() == "i" && d.focusPane != FocusInput {
			d.focusPane = FocusInput
			d.inputFocused = true
			if d.agentOutput != nil {
				d.agentOutput.SetInputFocused(true)
			}
			return nil
		}

		// When input is focused (FocusInput), handle Enter and Escape
		if d.focusPane == FocusInput {
			switch msg.String() {
			case "enter":
				// Handle user input submission
				if d.agentOutput != nil {
					text := d.agentOutput.InputValue()
					if text != "" {
						// Always emit immediately - orchestrator handles queueing
						d.agentOutput.ResetInput()
						d.inputFocused = false
						d.agentOutput.SetInputFocused(false)
						d.focusPane = FocusAgent
						return func() tea.Msg {
							return UserInputMsg{Text: text}
						}
					}
				}
				return nil
			case "esc":
				// Exit input and return to FocusAgent
				d.inputFocused = false
				if d.agentOutput != nil {
					d.agentOutput.SetInputFocused(false)
				}
				d.focusPane = FocusAgent
				return nil
			default:
				// Forward all other keys to the input field
				if d.agentOutput != nil {
					return d.agentOutput.Update(msg)
				}
			}
			return nil
		}

		// Tab: cycle through panes (Agent → Tasks → Notes → Agent)
		if msg.String() == "tab" {
			switch d.focusPane {
			case FocusAgent:
				d.focusPane = FocusTasks
			case FocusTasks:
				d.focusPane = FocusNotes
			case FocusNotes:
				d.focusPane = FocusAgent
			}
			d.updateScrollListFocus()
			return nil
		}

		// Update ScrollList focus states based on active pane
		d.updateScrollListFocus()

		// Forward keys based on focusPane
		switch d.focusPane {
		case FocusTasks, FocusNotes:
			return d.sidebar.Update(msg)
		case FocusAgent:
			if d.agentOutput != nil {
				return d.agentOutput.Update(msg)
			}
		}
	}

	return nil
}

// Draw renders the dashboard to a screen buffer using the Screen/Draw pattern.
func (d *Dashboard) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Draw title with rule line: "Agent Output ────────"
	agentPanelFocused := d.focusPane == FocusAgent && d.focusPane != FocusInput
	inner := DrawPanel(scr, area, "Agent Output", agentPanelFocused)

	// Add 1-row padding between header rule and messages viewport
	inner.Min.Y += 1

	// Delegate to AgentOutput.Draw for content rendering
	if d.agentOutput != nil {
		return d.agentOutput.Draw(scr, inner)
	}

	return nil
}

// SetSize updates the dashboard dimensions (implements Sizable interface).
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height

	// Update agent output viewport size
	if d.agentOutput != nil {
		// Account for border (2 chars each side)
		d.agentOutput.UpdateSize(width-2, height-2)
	}
}

// SetIteration sets the current iteration number.
func (d *Dashboard) SetIteration(n int) tea.Cmd {
	d.iteration = n
	return nil
}

// SetState updates the dashboard with new session state (implements Stateful interface).
func (d *Dashboard) SetState(state *session.State) {
	d.state = state
	// Update session name from state
	if state != nil {
		d.sessionName = state.Session
	}
}

// SetFocus sets the focus state of the dashboard (implements Focusable interface).
func (d *Dashboard) SetFocus(focused bool) {
	d.focused = focused
}

// IsFocused returns whether the dashboard has focus (implements Focusable interface).
func (d *Dashboard) IsFocused() bool {
	return d.focused
}

// SetAgentBusy sets whether the agent is currently processing.
// Updates the input placeholder to show "Agent is working..." when busy.
// The busy state is kept for input placeholder text only.
func (d *Dashboard) SetAgentBusy(busy bool) tea.Cmd {
	d.agentBusy = busy
	if d.agentOutput != nil {
		d.agentOutput.SetBusy(busy)
	}
	return nil
}

// SetQueueDepth updates the queue depth indicator in the agent output.
// Shows how many user messages are waiting in the orchestrator queue.
func (d *Dashboard) SetQueueDepth(depth int) tea.Cmd {
	if d.agentOutput != nil {
		d.agentOutput.SetQueueDepth(depth)
	}
	return nil
}

// updateScrollListFocus sets the focused state on ScrollLists based on the active pane.
// Only the active pane's ScrollList should have focused=true to receive keyboard events.
func (d *Dashboard) updateScrollListFocus() {
	// Agent output ScrollList (only focused when FocusAgent and input not focused)
	if d.agentOutput != nil {
		d.agentOutput.SetScrollFocused(d.focusPane == FocusAgent && d.focusPane != FocusInput)
	}

	// Sidebar ScrollLists
	if d.sidebar != nil {
		d.sidebar.SetTasksScrollFocused(d.focusPane == FocusTasks && d.focusPane != FocusInput)
		d.sidebar.SetNotesScrollFocused(d.focusPane == FocusNotes && d.focusPane != FocusInput)
	}
}
