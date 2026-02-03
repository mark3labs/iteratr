package specwizard

import (
	tea "charm.land/bubbletea/v2"
)

// DescriptionStep handles the description input step.
// TODO: Implement multi-line textarea for description input.
type DescriptionStep struct {
	width  int
	height int
}

// NewDescriptionStep creates a new description step.
func NewDescriptionStep() *DescriptionStep {
	return &DescriptionStep{}
}

// Init initializes the description step.
func (s *DescriptionStep) Init() tea.Cmd {
	return nil
}

// Update handles messages for the description step.
func (s *DescriptionStep) Update(msg tea.Msg) tea.Cmd {
	return nil
}

// View renders the description step.
func (s *DescriptionStep) View() string {
	return "Description step (TODO)"
}

// SetSize updates the size of the description step.
func (s *DescriptionStep) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Focus focuses the description step.
func (s *DescriptionStep) Focus() {
	// TODO: Focus textarea
}

// Blur blurs the description step.
func (s *DescriptionStep) Blur() {
	// TODO: Blur textarea
}

// Submit submits the description.
func (s *DescriptionStep) Submit() tea.Cmd {
	// TODO: Validate and submit description
	return func() tea.Msg {
		return DescriptionSubmittedMsg{Description: ""}
	}
}
