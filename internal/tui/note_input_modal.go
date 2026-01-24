package tui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
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

// Update handles keyboard input for the modal.
// For now, this is a minimal implementation that will be expanded in later tasks.
func (m *NoteInputModal) Update(msg tea.Msg) tea.Cmd {
	if !m.visible {
		return nil
	}

	// Handle key presses
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			// ESC closes the modal
			m.Close()
			return nil
		case "ctrl+enter":
			// Ctrl+Enter submits the note
			// Get the content from textarea
			content := strings.TrimSpace(m.textarea.Value())

			// Don't submit if empty (validation)
			if content == "" {
				return nil
			}

			// Return a function that creates the CreateNoteMsg
			// The iteration will be set by App when it receives this
			return m.submit(content)
		}
	}

	// Forward all other messages to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return cmd
}

// submit returns a command that creates a CreateNoteMsg.
// The App will receive this message and fill in the iteration number.
func (m *NoteInputModal) submit(content string) tea.Cmd {
	noteType := m.noteType
	return func() tea.Msg {
		return CreateNoteMsg{
			Content:   content,
			NoteType:  noteType,
			Iteration: 0, // Will be filled in by App
		}
	}
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

// Draw renders the modal centered on the screen buffer.
func (m *NoteInputModal) Draw(scr uv.Screen, area uv.Rectangle) {
	if !m.visible {
		return
	}

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
	if modalHeight < 8 {
		modalHeight = 8
	}

	// Build modal content using View()
	content := m.View()

	// Style the modal with border and background
	modalStyle := styleModalContainer.
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
