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

	// Section 2: Progress Indicator
	if d.state != nil {
		progressInfo := d.renderProgressIndicator()
		sections = append(sections, "", progressInfo)
	}

	// Section 3: Current Task
	if d.state != nil {
		currentTask := d.renderCurrentTask()
		if currentTask != "" {
			sections = append(sections, "", currentTask)
		}
	}

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

// renderProgressIndicator renders a progress bar showing task completion.
func (d *Dashboard) renderProgressIndicator() string {
	// Count tasks by status
	var total, completed int
	for _, task := range d.state.Tasks {
		total++
		if task.Status == "completed" {
			completed++
		}
	}

	// Build progress bar
	const barWidth = 40
	var completedWidth int
	if total > 0 {
		completedWidth = (completed * barWidth) / total
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
	progressText := fmt.Sprintf("%d/%d tasks", completed, total)

	// Combine bar and text
	bar := styleProgressFill.Render(filled) + styleDim.Render(empty)
	label := styleStatLabel.Render("Progress:")
	return fmt.Sprintf("%s [%s] %s", label, bar, styleStatValue.Render(progressText))
}

// renderCurrentTask renders the current in_progress task (if any).
func (d *Dashboard) renderCurrentTask() string {
	// Find first in_progress task
	var currentTask *session.Task
	for _, task := range d.state.Tasks {
		if task.Status == "in_progress" {
			currentTask = task
			break
		}
	}

	// Return empty string if no in_progress task
	if currentTask == nil {
		return ""
	}

	// Format task ID (8 char prefix)
	taskIDPrefix := currentTask.ID
	if len(taskIDPrefix) > 8 {
		taskIDPrefix = taskIDPrefix[:8]
	}

	// Build current task display
	label := styleStatLabel.Render("Current Task:")
	taskText := fmt.Sprintf("[%s] %s", taskIDPrefix, currentTask.Content)
	taskBox := styleCurrentTask.Render(taskText)

	return fmt.Sprintf("%s\n%s", label, taskBox)
}
