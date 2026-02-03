package specwizard

import (
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// ConfirmationModal represents a reusable confirmation modal.
type ConfirmationModal struct {
	title   string
	message string
	visible bool
}

// NewConfirmationModal creates a new confirmation modal.
func NewConfirmationModal(title, message string) *ConfirmationModal {
	return &ConfirmationModal{
		title:   title,
		message: message,
		visible: false,
	}
}

// Show makes the modal visible.
func (m *ConfirmationModal) Show() {
	m.visible = true
}

// Hide hides the modal.
func (m *ConfirmationModal) Hide() {
	m.visible = false
}

// IsVisible returns whether the modal is currently visible.
func (m *ConfirmationModal) IsVisible() bool {
	return m.visible
}

// Render renders the confirmation modal with the given title and message.
func (m *ConfirmationModal) Render() string {
	return RenderConfirmationModal(m.title, m.message)
}

// RenderConfirmationModal renders a confirmation modal with the given title and message.
// This is a standalone helper that can be used without creating a ConfirmationModal struct.
func RenderConfirmationModal(title, message string) string {
	t := theme.Current()

	// Title with warning icon
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Warning)).
		MarginBottom(1)
	titleText := titleStyle.Render("⚠ " + title)

	// Message
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgBase)).
		MarginBottom(1)
	messageText := messageStyle.Render(message)

	// Buttons
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgMuted))
	buttons := buttonStyle.Render("Press Y to confirm, N or ESC to cancel")

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleText,
		messageText,
		"",
		buttons,
	)

	// Modal styling
	modalStyle := lipgloss.NewStyle().
		Width(50).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Warning))

	return modalStyle.Render(content)
}
