package specwizard

import (
	tea "charm.land/bubbletea/v2"
)

// CompletionStep shows the completion screen with Build/Exit buttons.
// TODO: Implement success message and action buttons.
type CompletionStep struct {
	specPath string
	width    int
	height   int
}

// NewCompletionStep creates a new completion step.
func NewCompletionStep(specPath string) *CompletionStep {
	return &CompletionStep{
		specPath: specPath,
	}
}

// Init initializes the completion step.
func (s *CompletionStep) Init() tea.Cmd {
	return nil
}

// Update handles messages for the completion step.
func (s *CompletionStep) Update(msg tea.Msg) tea.Cmd {
	return nil
}

// View renders the completion step.
func (s *CompletionStep) View() string {
	return "Completion step (TODO)"
}

// SetSize updates the size of the completion step.
func (s *CompletionStep) SetSize(width, height int) {
	s.width = width
	s.height = height
}
