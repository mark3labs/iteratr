package specwizard

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

// TitleStep handles the title input with validation.
type TitleStep struct {
	input  textinput.Model
	width  int
	height int
	err    string // Validation error message
}

// collapseNewlines replaces all newline sequences with a single space,
// then collapses consecutive whitespace into single space.
// Used for single-line text inputs to collapse multi-line paste content.
func collapseNewlines(content string) string {
	// Replace all newline sequences (\n, \r\n, \r) with a single space
	content = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(content)

	// Collapse consecutive spaces into single space
	return strings.Join(strings.Fields(content), " ")
}

// NewTitleStep creates a new title input step.
func NewTitleStep() *TitleStep {
	ti := textinput.New()
	ti.Placeholder = "e.g., User Authentication"
	ti.CharLimit = 100 // Enforce max length
	ti.SetWidth(54)    // Width for placeholder visibility (box is 60 - padding)
	ti.Focus()

	return &TitleStep{
		input: ti,
	}
}

// validateTitle checks if the title is valid.
func validateTitle(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(s) > 100 {
		return fmt.Errorf("title too long (max 100 characters)")
	}
	return nil
}

// Init initializes the title step.
func (t *TitleStep) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the title step.
func (t *TitleStep) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return nil

	case tea.KeyPressMsg:
		keyStr := msg.String()
		switch keyStr {
		case "enter":
			// Validate before proceeding
			value := strings.TrimSpace(t.input.Value())
			if err := validateTitle(value); err != nil {
				t.err = err.Error()
				return nil
			}
			// Clear error and notify completion
			t.err = ""
			return func() tea.Msg {
				return TitleSubmittedMsg{Title: value}
			}
		case "tab":
			// Signal to wizard to move focus to buttons
			return func() tea.Msg {
				return wizard.TabExitForwardMsg{}
			}
		case "shift+tab":
			// Signal to wizard to move focus to buttons from end
			return func() tea.Msg {
				return wizard.TabExitBackwardMsg{}
			}
		default:
			// Clear error on any other input
			if t.err != "" {
				t.err = ""
			}
		}

	case tea.PasteMsg:
		// Intercept paste messages for single-line input - sanitize and collapse newlines
		sanitized := tui.SanitizePaste(msg.Content)
		collapsed := collapseNewlines(sanitized)

		// Create modified paste message with collapsed content
		modifiedMsg := tea.PasteMsg{Content: collapsed}

		// Forward to input
		var cmd tea.Cmd
		t.input, cmd = t.input.Update(modifiedMsg)

		// Clear error on paste
		if t.err != "" {
			t.err = ""
		}

		return cmd
	}

	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	return cmd
}

// View renders the title step content (returns string for embedding in wizard).
func (t *TitleStep) View() string {
	currentTheme := theme.Current()

	// Instruction
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgBase)).
		MarginBottom(1)

	instruction := instructionStyle.Render("Enter a human-readable name for your feature:")

	// Input box
	inputStyle := lipgloss.NewStyle().
		Width(60).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(currentTheme.BorderDefault))

	input := inputStyle.Render(t.input.View())

	// Error message (if any)
	errorText := ""
	if t.err != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(currentTheme.Error)).
			Bold(true)
		errorText = errorStyle.Render("✗ " + t.err)
	}

	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Left,
		instruction,
		input,
		errorText,
	)
}

// GetTitle returns the current title value (trimmed).
func (t *TitleStep) GetTitle() string {
	return strings.TrimSpace(t.input.Value())
}

// SetSize updates the size of the title step.
func (t *TitleStep) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// Focus focuses the title input.
func (t *TitleStep) Focus() {
	t.input.Focus()
}

// Blur blurs the title input.
func (t *TitleStep) Blur() {
	t.input.Blur()
}

// Submit submits the title (validates and sends TitleSubmittedMsg).
func (t *TitleStep) Submit() tea.Cmd {
	value := strings.TrimSpace(t.input.Value())
	if err := validateTitle(value); err != nil {
		t.err = err.Error()
		return nil
	}
	t.err = ""
	return func() tea.Msg {
		return TitleSubmittedMsg{Title: value}
	}
}
