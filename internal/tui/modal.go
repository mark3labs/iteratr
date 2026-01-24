package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// TaskModal displays detailed information about a single task in a centered overlay.
type TaskModal struct {
	task    *session.Task
	visible bool
	width   int // Modal width
	height  int // Modal height

	// Render cache
	cachedContent      string
	cachedContentWidth int
	cachedTask         *session.Task
}

// NewTaskModal creates a new TaskModal component.
func NewTaskModal() *TaskModal {
	return &TaskModal{
		visible: false,
		width:   60, // Default width
		height:  20, // Default height
	}
}

// SetTask sets the task to display in the modal and shows it.
func (m *TaskModal) SetTask(task *session.Task) {
	m.task = task
	m.visible = true
	m.cachedContent = "" // Invalidate cache
}

// Close hides the modal.
func (m *TaskModal) Close() {
	m.visible = false
	m.task = nil
}

// IsVisible returns whether the modal is currently visible.
func (m *TaskModal) IsVisible() bool {
	return m.visible
}

// Draw renders the modal centered on the screen buffer (Screen/Draw pattern).
func (m *TaskModal) Draw(scr uv.Screen, area uv.Rectangle) {
	if !m.visible || m.task == nil {
		return
	}

	// Calculate modal dimensions (max 60x20, but adapt to screen size)
	modalWidth := m.width
	modalHeight := m.height

	// Ensure modal fits on screen with margins
	if modalWidth > area.Dx()-4 {
		modalWidth = area.Dx() - 4
	}
	if modalHeight > area.Dy()-4 {
		modalHeight = area.Dy() - 4
	}

	// Ensure minimum dimensions
	if modalWidth < 30 {
		modalWidth = 30
	}
	if modalHeight < 10 {
		modalHeight = 10
	}

	// Build modal content
	content := m.buildContent(modalWidth - 4) // Account for padding and borders

	// Style the modal with border and background
	modalStyle := styleModalContainer.Copy().
		Width(modalWidth).
		Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Calculate center position
	renderedWidth := lipgloss.Width(modalContent)
	renderedHeight := lipgloss.Height(modalContent)
	x := (area.Dx() - renderedWidth) / 2
	y := (area.Dy() - renderedHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Draw modal centered on screen
	modalArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X + x, Y: area.Min.Y + y},
		Max: uv.Position{X: area.Min.X + x + renderedWidth, Y: area.Min.Y + y + renderedHeight},
	}
	uv.NewStyledString(modalContent).Draw(scr, modalArea)
}

// buildContent builds the modal content string with task details.
func (m *TaskModal) buildContent(width int) string {
	if m.task == nil {
		return ""
	}

	// Return cached content if task and width haven't changed
	if m.cachedTask == m.task && m.cachedContentWidth == width && m.cachedContent != "" {
		return m.cachedContent
	}

	var sections []string

	// === Title Section (with diagonal hatching decoration) ===
	title := renderModalTitle("Task Details", width-2)
	sections = append(sections, title)
	sections = append(sections, "") // Blank line

	// === ID Section ===
	idLine := styleModalLabel.Render("ID: ") + styleModalValue.Render(m.task.ID)
	sections = append(sections, idLine)
	sections = append(sections, "") // Blank line

	// === Status and Priority Section ===
	statusBadge := m.renderStatusBadge(m.task.Status)
	priorityBadge := m.renderPriorityBadge(m.task.Priority)
	statusLine := fmt.Sprintf("%s %s    %s %s",
		styleModalLabel.Render("Status:"),
		statusBadge,
		styleModalLabel.Render("Priority:"),
		priorityBadge)
	sections = append(sections, statusLine)
	sections = append(sections, "") // Blank line

	// === Separator ===
	separator := styleModalSeparator.Render(strings.Repeat("─", width-2))
	sections = append(sections, separator)
	sections = append(sections, "") // Blank line

	// === Content Section ===
	// Word-wrap content to fit modal width
	wrappedContent := styleModalSection.Render(m.wordWrap(m.task.Content, width-2))
	sections = append(sections, wrappedContent)
	sections = append(sections, "") // Blank line

	// === Separator ===
	sections = append(sections, separator)
	sections = append(sections, "") // Blank line

	// === Dependencies Section ===
	if len(m.task.DependsOn) > 0 {
		depsLabel := styleModalLabel.Render("Depends on: ")
		depsContent := styleModalValue.Render(strings.Join(m.task.DependsOn, ", "))
		sections = append(sections, depsLabel+depsContent)
		sections = append(sections, "") // Blank line
	}

	// === Timestamps Section ===
	createdLine := styleModalLabel.Render("Created:  ") + styleModalValue.Render(m.formatTime(m.task.CreatedAt))
	updatedLine := styleModalLabel.Render("Updated:  ") + styleModalValue.Render(m.formatTime(m.task.UpdatedAt))
	sections = append(sections, createdLine)
	sections = append(sections, updatedLine)
	sections = append(sections, "") // Blank line

	// === Close Instructions (key/description differentiation) ===
	closeHint := styleHintKey.Render("esc") + " " +
		styleHintDesc.Render("close") + " " +
		styleHintSeparator.Render("•") + " " +
		styleHintKey.Render("click outside") + " " +
		styleHintDesc.Render("dismiss")
	closeText := lipgloss.NewStyle().Width(width - 2).Align(lipgloss.Center).Render(closeHint)
	sections = append(sections, closeText)

	result := strings.Join(sections, "\n")

	// Cache the result
	m.cachedContent = result
	m.cachedContentWidth = width
	m.cachedTask = m.task

	return result
}

// renderStatusBadge renders a styled badge for the task status.
func (m *TaskModal) renderStatusBadge(status string) string {
	var badge lipgloss.Style
	var text string

	switch status {
	case "in_progress":
		badge = styleBadgeWarning
		text = "► in_progress"
	case "remaining":
		badge = styleBadgeMuted
		text = "○ remaining"
	case "completed":
		badge = styleBadgeSuccess
		text = "✓ completed"
	case "blocked":
		badge = styleBadgeError
		text = "⊘ blocked"
	default:
		badge = styleBadgeMuted
		text = "○ " + status
	}

	return badge.Render(text)
}

// renderPriorityBadge renders a styled badge for the task priority.
func (m *TaskModal) renderPriorityBadge(priority int) string {
	var badge lipgloss.Style
	var text string

	switch priority {
	case 0:
		badge = styleBadgeError
		text = "critical"
	case 1:
		badge = styleBadgeWarning
		text = "high"
	case 2:
		badge = styleBadgeInfo
		text = "medium"
	case 3:
		badge = styleBadgeMuted
		text = "low"
	case 4:
		badge = styleBadgeMuted.Copy().Faint(true)
		text = "backlog"
	default:
		badge = styleBadgeMuted
		text = fmt.Sprintf("p%d", priority)
	}

	return badge.Render(text)
}

// formatTime formats a timestamp for display.
func (m *TaskModal) formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// wordWrap wraps text to fit within the specified width.
func (m *TaskModal) wordWrap(text string, width int) string {
	if width <= 0 {
		width = 40
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		// If adding this word would exceed width, start new line
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > width {
			// Current line is full, save it and start new line
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	// Add remaining line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// Update handles messages for the modal.
func (m *TaskModal) Update(msg tea.Msg) tea.Cmd {
	return nil
}
