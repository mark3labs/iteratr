package setup

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// CompletionStep is the final step showing the config file location and next steps.
type CompletionStep struct {
	width     int
	height    int
	isProject bool // True if config was written to project directory
}

// NewCompletionStep creates a new completion step.
func NewCompletionStep(isProject bool) *CompletionStep {
	return &CompletionStep{
		isProject: isProject,
	}
}

// Init initializes the completion step.
func (m *CompletionStep) Init() tea.Cmd {
	return nil
}

// Update handles messages for the completion step.
func (m *CompletionStep) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyPressMsg:
		// Any key press exits the wizard
		return tea.Quit
	}
	return nil
}

// View renders the completion step.
func (m *CompletionStep) View() string {
	var sections []string

	// Success message
	successMsg := theme.Current().S().Success.Render("✓ Configuration created successfully!")
	sections = append(sections, successMsg)
	sections = append(sections, "")

	// Config file path
	var configPath string
	if m.isProject {
		configPath = config.ProjectPath()
	} else {
		configPath = config.GlobalPath()
	}

	pathLabel := theme.Current().S().ModalLabel.Render("Config written to:")
	pathValue := theme.Current().S().ModalValue.Render(configPath)
	sections = append(sections, fmt.Sprintf("%s\n%s", pathLabel, pathValue))
	sections = append(sections, "")

	// Next steps
	nextStepsLabel := theme.Current().S().ModalLabel.Render("Next steps:")
	sections = append(sections, nextStepsLabel)
	sections = append(sections, theme.Current().S().ModalHint.Render("  Run 'iteratr build' to get started"))
	sections = append(sections, "")

	// Exit hint
	exitHint := theme.Current().S().ModalHint.Render("press any key to exit")
	sections = append(sections, exitHint)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// SetSize updates the dimensions of the completion step.
func (m *CompletionStep) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// PreferredHeight returns the preferred content height for this step.
func (m *CompletionStep) PreferredHeight() int {
	// Success message: 1 line
	// Blank: 1 line
	// Path label: 1 line
	// Path value: 1 line
	// Blank: 1 line
	// Next steps label: 1 line
	// Next steps hint: 1 line
	// Blank: 1 line
	// Exit hint: 1 line
	return 9
}
