package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/iteratr/internal/session"
)

// Dashboard displays session overview, progress, and current task.
type Dashboard struct {
	sessionName string
	iteration   int
	state       *session.State
	width       int
	height      int
}

// NewDashboard creates a new Dashboard component.
func NewDashboard() *Dashboard {
	return &Dashboard{}
}

// Update handles messages for the dashboard.
func (d *Dashboard) Update(msg tea.Msg) tea.Cmd {
	// TODO: Implement dashboard-specific updates
	return nil
}

// Render returns the dashboard view as a string.
func (d *Dashboard) Render() string {
	// Build dashboard sections
	var sections []string

	// Section 1: Session Info
	sessionInfo := d.renderSessionInfo()
	sections = append(sections, sessionInfo)

	// Join sections with spacing
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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

// UpdateSize updates the dashboard dimensions.
func (d *Dashboard) UpdateSize(width, height int) tea.Cmd {
	d.width = width
	d.height = height
	return nil
}

// SetIteration sets the current iteration number.
func (d *Dashboard) SetIteration(n int) tea.Cmd {
	d.iteration = n
	return nil
}

// UpdateState updates the dashboard with new session state.
func (d *Dashboard) UpdateState(state *session.State) tea.Cmd {
	d.state = state
	// Update session name from state
	if state != nil {
		d.sessionName = state.Session
	}
	return nil
}
