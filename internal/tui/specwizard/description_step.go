package specwizard

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

// DescriptionStep handles the description textarea input with validation.
type DescriptionStep struct {
	textarea textarea.Model
	width    int
	height   int
	err      string // Validation error message
}

// NewDescriptionStep creates a new description input step.
func NewDescriptionStep() *DescriptionStep {
	ta := textarea.New()
	ta.Placeholder = "Describe your feature in detail...\n\nExample:\n- What problem does it solve?\n- Who will use it?\n- What are the key requirements?"
	ta.CharLimit = 5000 // Enforce max length
	ta.SetHeight(8)     // Default height for textarea
	ta.SetWidth(60)     // Default width for textarea
	ta.Focus()

	return &DescriptionStep{
		textarea: ta,
	}
}

// validateDescription checks if the description is valid.
func validateDescription(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("description cannot be empty")
	}
	if len(s) < 10 {
		return fmt.Errorf("description too short (minimum 10 characters)")
	}
	if len(s) > 5000 {
		return fmt.Errorf("description too long (max 5000 characters)")
	}
	return nil
}

// Init initializes the description step.
func (d *DescriptionStep) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the description step.
func (d *DescriptionStep) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		return nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+d":
			// Ctrl+D submits the description
			value := strings.TrimSpace(d.textarea.Value())
			if err := validateDescription(value); err != nil {
				d.err = err.Error()
				return nil
			}
			// Clear error and notify completion
			d.err = ""
			return func() tea.Msg {
				return DescriptionSubmittedMsg{Description: value}
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
			if d.err != "" {
				d.err = ""
			}
		}
	}

	var cmd tea.Cmd
	d.textarea, cmd = d.textarea.Update(msg)
	return cmd
}

// View renders the description step content (returns string for embedding in wizard).
func (d *DescriptionStep) View() string {
	currentTheme := theme.Current()

	// Instruction
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgBase)).
		MarginBottom(1)

	instruction := instructionStyle.Render("Provide a detailed description of your feature:")

	// Textarea box
	textareaStyle := lipgloss.NewStyle().
		Width(62). // Slightly wider to account for padding
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(currentTheme.BorderDefault))

	textareaView := textareaStyle.Render(d.textarea.View())

	// Hint
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgMuted)).
		Italic(true).
		MarginTop(1)

	hint := hintStyle.Render("Press Ctrl+D when finished")

	// Error message (if any)
	errorText := ""
	if d.err != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(currentTheme.Error)).
			Bold(true).
			MarginTop(1)
		errorText = errorStyle.Render("✗ " + d.err)
	}

	// Combine all parts
	parts := []string{
		instruction,
		textareaView,
		hint,
	}
	if errorText != "" {
		parts = append(parts, errorText)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		parts...,
	)
}

// GetDescription returns the current description value (trimmed).
func (d *DescriptionStep) GetDescription() string {
	return strings.TrimSpace(d.textarea.Value())
}

// SetSize updates the size of the description step.
func (d *DescriptionStep) SetSize(width, height int) {
	d.width = width
	d.height = height
	// Adjust textarea size based on available space
	// Leave room for instruction, hint, and borders
	maxTextareaHeight := height - 12 // Reserve space for other elements
	if maxTextareaHeight < 6 {
		maxTextareaHeight = 6 // Minimum height
	}
	if maxTextareaHeight > 15 {
		maxTextareaHeight = 15 // Maximum height
	}
	d.textarea.SetHeight(maxTextareaHeight)
}

// Focus focuses the description textarea.
func (d *DescriptionStep) Focus() {
	d.textarea.Focus()
}

// Blur blurs the description textarea.
func (d *DescriptionStep) Blur() {
	d.textarea.Blur()
}

// Submit submits the description (validates and sends DescriptionSubmittedMsg).
func (d *DescriptionStep) Submit() tea.Cmd {
	value := strings.TrimSpace(d.textarea.Value())
	if err := validateDescription(value); err != nil {
		d.err = err.Error()
		return nil
	}
	d.err = ""
	return func() tea.Msg {
		return DescriptionSubmittedMsg{Description: value}
	}
}
