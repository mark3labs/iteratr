package tui

import (
	"fmt"
	"strings"

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
	agentOutput *AgentOutput // Reference to agent output for rendering
}

// NewDashboard creates a new Dashboard component.
func NewDashboard(agentOutput *AgentOutput) *Dashboard {
	return &Dashboard{
		agentOutput: agentOutput,
	}
}

// Update handles messages for the dashboard.
func (d *Dashboard) Update(msg tea.Msg) tea.Cmd {
	// Forward scroll events to agent output viewport
	if d.agentOutput != nil {
		return d.agentOutput.Update(msg)
	}
	return nil
}

// Render returns the dashboard view as a string.
func (d *Dashboard) Render() string {
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

		// Section 2.5: Task Stats
		taskStats := d.renderTaskStats()
		if taskStats != "" {
			headerSections = append(headerSections, taskStats)
		}
	}

	// Section 3: Current Task
	if d.state != nil {
		currentTask := d.renderCurrentTask()
		if currentTask != "" {
			headerSections = append(headerSections, "") // blank line
			headerSections = append(headerSections, currentTask)
		}
	}

	// Render header
	header := lipgloss.JoinVertical(lipgloss.Left, headerSections...)

	// Section 4: Agent Output (takes remaining space)
	var agentSection string
	if d.agentOutput != nil {
		agentLabel := styleStatLabel.Render("Agent Output:")
		agentContent := d.agentOutput.Render()
		agentSection = lipgloss.JoinVertical(lipgloss.Left, "", agentLabel, "", agentContent)
	}

	// Join header and agent sections
	return lipgloss.JoinVertical(lipgloss.Left, header, agentSection)
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

	// Update agent output viewport size
	// Reserve space for: session info (2) + progress (2) + current task (3) + agent label (2) + padding (3)
	if d.agentOutput != nil {
		agentHeight := height - 12
		if agentHeight < 5 {
			agentHeight = 5
		}
		d.agentOutput.UpdateSize(width, agentHeight)
	}
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

// renderTaskStats renders detailed task completion statistics.
func (d *Dashboard) renderTaskStats() string {
	stats := d.getTaskStats()

	// Build stats line with color-coded counts
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
		return ""
	}

	label := styleStatLabel.Render("Status:")
	// Join with separator for readability
	separator := styleDim.Render(" | ")
	statusText := strings.Join(parts, separator)
	return fmt.Sprintf("%s %s", label, statusText)
}

// taskStats holds task statistics by status.
type taskStats struct {
	Total      int
	Remaining  int
	InProgress int
	Completed  int
	Blocked    int
}

// getTaskStats computes task statistics from current state.
func (d *Dashboard) getTaskStats() taskStats {
	var stats taskStats
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
