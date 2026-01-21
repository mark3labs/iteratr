package tui

import (
	tea "charm.land/bubbletea/v2"
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
	// TODO: Implement dashboard rendering with lipgloss
	return "Dashboard view (TODO)"
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
	return nil
}
