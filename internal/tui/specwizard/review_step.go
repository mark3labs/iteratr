package specwizard

import (
	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
)

// ReviewStep handles the spec review and editing step.
// TODO: Implement viewport with markdown highlighting and editor support.
type ReviewStep struct {
	content string
	cfg     *config.Config
	width   int
	height  int
}

// NewReviewStep creates a new review step.
func NewReviewStep(content string, cfg *config.Config) *ReviewStep {
	return &ReviewStep{
		content: content,
		cfg:     cfg,
	}
}

// Init initializes the review step.
func (s *ReviewStep) Init() tea.Cmd {
	return nil
}

// Update handles messages for the review step.
func (s *ReviewStep) Update(msg tea.Msg) tea.Cmd {
	return nil
}

// View renders the review step.
func (s *ReviewStep) View() string {
	return "Review step (TODO)"
}

// SetSize updates the size of the review step.
func (s *ReviewStep) SetSize(width, height int) {
	s.width = width
	s.height = height
}
