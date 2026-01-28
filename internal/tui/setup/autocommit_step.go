package setup

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// AutoCommitStep manages the auto-commit selection step.
type AutoCommitStep struct {
	selectedIdx int // 0 = Yes, 1 = No
	width       int
	height      int
}

// NewAutoCommitStep creates a new auto-commit selector step.
func NewAutoCommitStep() *AutoCommitStep {
	return &AutoCommitStep{
		selectedIdx: 0, // Default to "Yes"
		width:       60,
		height:      10,
	}
}

// Init initializes the auto-commit step.
func (a *AutoCommitStep) Init() tea.Cmd {
	return nil
}

// SetSize updates the dimensions for the auto-commit selector.
func (a *AutoCommitStep) SetSize(width, height int) {
	a.width = width
	a.height = height
}

// Update handles messages for the auto-commit step.
func (a *AutoCommitStep) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if a.selectedIdx > 0 {
				a.selectedIdx--
			}
			return nil

		case "down", "j":
			if a.selectedIdx < 1 {
				a.selectedIdx++
			}
			return nil

		case "enter":
			// Selection made
			enabled := a.selectedIdx == 0 // 0 = Yes, 1 = No
			return func() tea.Msg {
				return AutoCommitSelectedMsg{Enabled: enabled}
			}
		}
	}

	return nil
}

// View renders the auto-commit selection step.
func (a *AutoCommitStep) View() string {
	var b strings.Builder

	// Title/question
	questionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Bold(true)
	b.WriteString(questionStyle.Render("Auto-commit changes after each iteration?"))
	b.WriteString("\n\n")

	// Options
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#b4befe")).
		Bold(true).
		Padding(0, 1)

	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4")).
		Padding(0, 1)

	recommendedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")).
		Italic(true)

	// Yes option (recommended)
	if a.selectedIdx == 0 {
		b.WriteString(selectedStyle.Render("Yes"))
		b.WriteString(" ")
		b.WriteString(recommendedStyle.Render("(recommended)"))
	} else {
		b.WriteString(unselectedStyle.Render("Yes"))
		b.WriteString(" ")
		b.WriteString(recommendedStyle.Render("(recommended)"))
	}
	b.WriteString("\n")

	// No option
	if a.selectedIdx == 1 {
		b.WriteString(selectedStyle.Render("No"))
	} else {
		b.WriteString(unselectedStyle.Render("No"))
	}
	b.WriteString("\n\n")

	// Hint bar
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	b.WriteString(hintStyle.Render("↑↓/j/k navigate • Enter select • ESC back"))

	return b.String()
}

// PreferredHeight returns the preferred height for this step's content.
func (a *AutoCommitStep) PreferredHeight() int {
	// Question: 1
	// Blank: 1
	// Yes option: 1
	// No option: 1
	// Blank: 1
	// Hint: 1
	// Total: 6
	return 6
}
