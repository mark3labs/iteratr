package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// NoteInputModal is an interactive modal for creating new notes.
// It displays a textarea for content input and allows the user to submit notes.
type NoteInputModal struct {
	visible  bool
	textarea textarea.Model
	noteType string // Current selected type (hardcoded to "learning" for now)
	width    int
	height   int
}

// NewNoteInputModal creates a new NoteInputModal component.
func NewNoteInputModal() *NoteInputModal {
	// Create and configure textarea
	ta := textarea.New()
	ta.Placeholder = "Enter your note..."
	ta.CharLimit = 500
	ta.ShowLineNumbers = false
	ta.Prompt = "" // No prompt character
	ta.SetWidth(50)
	ta.SetHeight(6)

	return &NoteInputModal{
		visible:  false,
		textarea: ta,
		noteType: "learning", // Hardcoded for tracer bullet
		width:    60,
		height:   16,
	}
}

// IsVisible returns whether the modal is currently visible.
func (m *NoteInputModal) IsVisible() bool {
	return m.visible
}

// Show makes the modal visible and focuses the textarea.
func (m *NoteInputModal) Show() tea.Cmd {
	m.visible = true
	return m.textarea.Focus()
}

// Close hides the modal.
func (m *NoteInputModal) Close() {
	m.visible = false
}

// View renders the modal content (for testing and integration).
func (m *NoteInputModal) View() string {
	if !m.visible {
		return ""
	}

	var sections []string

	// Title
	title := renderModalTitle("New Note", m.width-4)
	sections = append(sections, title)
	sections = append(sections, "")

	// Textarea
	sections = append(sections, m.textarea.View())
	sections = append(sections, "")

	// Submit button (static, unfocused state for now)
	button := m.renderButton()
	buttonLine := lipgloss.NewStyle().Width(m.width - 4).Align(lipgloss.Right).Render(button)
	sections = append(sections, buttonLine)

	return strings.Join(sections, "\n")
}

// renderButton renders the submit button in its current state.
// For now, this is static (unfocused). Focus states will be added in a later task.
func (m *NoteInputModal) renderButton() string {
	buttonStyle := styleBadgeMuted.Copy()
	return buttonStyle.Render("  Save Note  ")
}
