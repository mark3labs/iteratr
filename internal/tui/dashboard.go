package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
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
	agentBusy    bool         // Whether the agent is currently processing
	queuedMsg    string       // Message queued while agent is busy
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
						if d.agentBusy {
							// Agent is busy - queue the message for later
							d.queuedMsg = text
							d.agentOutput.ResetInput()
							d.inputFocused = false
							d.agentOutput.SetInputFocused(false)
							d.focusPane = FocusAgent
						} else {
							// Agent is not busy - emit message immediately
							d.agentOutput.ResetInput()
							d.inputFocused = false
							d.agentOutput.SetInputFocused(false)
							d.focusPane = FocusAgent
							return func() tea.Msg {
								return UserInputMsg{Text: text}
							}
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
	// Draw panel border with title
	// Show accent border when FocusAgent (unless input is focused - in which case input gets the highlight)
	agentPanelFocused := d.focusPane == FocusAgent && d.focusPane != FocusInput
	inner := DrawPanel(scr, area, "Agent Output", agentPanelFocused)

	// Delegate to AgentOutput.Draw for content rendering
	if d.agentOutput != nil {
		return d.agentOutput.Draw(scr, inner)
	}

	return nil
}

// renderMainContent renders the main content area (session info + agent output).
func (d *Dashboard) renderMainContent(width int) string {
	// Build header sections (fixed height)
	var headerSections []string

	// Section 1: Session Info
	sessionInfo := d.renderSessionInfo()
	headerSections = append(headerSections, sessionInfo)

	// Section 2: Progress Indicator
	if d.state != nil {
		progressInfo := d.renderProgressIndicator()
		headerSections = append(headerSections, "") // blank line
		headerSections = append(headerSections, progressInfo)
	}

	// Render header
	header := lipgloss.JoinVertical(lipgloss.Left, headerSections...)

	// Section 3: Agent Output (takes remaining space)
	var agentSection string
	if d.agentOutput != nil {
		focusIndicator := ""
		if d.focusPane == FocusAgent {
			focusIndicator = " " + styleStatusInProgress.Render("●")
		}
		agentLabel := styleStatLabel.Render("Agent Output:") + focusIndicator
		agentContent := d.agentOutput.Render()
		agentSection = lipgloss.JoinVertical(lipgloss.Left, "", agentLabel, "", agentContent)
	}

	// Join header and agent sections
	content := lipgloss.JoinVertical(lipgloss.Left, header, agentSection)

	// Apply width constraint
	return lipgloss.NewStyle().Width(width).Render(content)
}

// renderSessionInfo renders the session name and iteration number.
func (d *Dashboard) renderSessionInfo() string {
	var parts []string

	// Session name
	sessionLabel := styleStatLabel.Render("Session:")
	sessionValue := styleStatValue.Render(d.sessionName)
	parts = append(parts, sessionLabel+" "+sessionValue)

	// Iteration number
	iterationLabel := styleStatLabel.Render("Iteration:")
	iterationValue := styleStatValue.Render(fmt.Sprintf("#%d", d.iteration))
	parts = append(parts, iterationLabel+" "+iterationValue)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
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

// UpdateSize updates the dashboard dimensions (legacy method for backward compatibility).
func (d *Dashboard) UpdateSize(width, height int) tea.Cmd {
	d.SetSize(width, height)
	// Note: sidebar sizing is handled by App.propagateSizes() directly
	// since Dashboard shares the same Sidebar instance with App.
	return nil
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

// UpdateState updates the dashboard with new session state (legacy method for backward compatibility).
func (d *Dashboard) UpdateState(state *session.State) tea.Cmd {
	d.SetState(state)
	// Note: sidebar state is propagated by App directly via a.sidebar.SetState()
	// since Dashboard shares the same Sidebar instance with App.
	return nil
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
// When the agent becomes available and there's a queued message, emits it immediately.
func (d *Dashboard) SetAgentBusy(busy bool) tea.Cmd {
	d.agentBusy = busy
	if d.agentOutput != nil {
		d.agentOutput.SetBusy(busy)
	}

	// When agent becomes available (!busy) and there's a queued message, emit it
	if !busy && d.queuedMsg != "" {
		msg := d.queuedMsg
		d.queuedMsg = "" // Clear the queue
		return func() tea.Msg {
			return UserInputMsg{Text: msg}
		}
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

// renderProgressIndicator renders a progress bar showing task completion.
func (d *Dashboard) renderProgressIndicator() string {
	// Count tasks by status
	stats := d.getTaskStats()

	// Build progress bar
	const barWidth = 40
	var completedWidth int
	if stats.Total > 0 {
		completedWidth = (stats.Completed * barWidth) / stats.Total
	}

	// Create the bar with filled and empty portions
	filled := ""
	empty := ""
	for i := 0; i < completedWidth; i++ {
		filled += "█"
	}
	for i := completedWidth; i < barWidth; i++ {
		empty += "░"
	}

	// Format the progress text
	progressText := fmt.Sprintf("%d/%d tasks", stats.Completed, stats.Total)

	// Combine bar and text
	bar := styleProgressFill.Render(filled) + styleDim.Render(empty)
	label := styleStatLabel.Render("Progress:")
	return fmt.Sprintf("%s [%s] %s", label, bar, styleStatValue.Render(progressText))
}

// progressStats holds task statistics.
type progressStats struct {
	Total      int
	Remaining  int
	InProgress int
	Completed  int
	Blocked    int
}

// getTaskStats computes task statistics.
func (d *Dashboard) getTaskStats() progressStats {
	var stats progressStats
	if d.state == nil {
		return stats
	}
	for _, task := range d.state.Tasks {
		stats.Total++
		switch task.Status {
		case "remaining":
			stats.Remaining++
		case "in_progress":
			stats.InProgress++
		case "completed":
			stats.Completed++
		case "blocked":
			stats.Blocked++
		}
	}
	return stats
}
