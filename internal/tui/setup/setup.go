package setup

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// ContentChangedMsg is sent when a step's content changes in a way that affects preferred height.
// The setup wizard handles this by recalculating modal dimensions.
type ContentChangedMsg struct{}

// SetupResult holds the output values from the setup wizard.
type SetupResult struct {
	Model      string // Selected model ID (e.g. "anthropic/claude-sonnet-4-5")
	AutoCommit bool   // Auto-commit after iterations
}

// SetupModel is the main BubbleTea model for the setup wizard.
// It manages the three-step flow: model selector → auto-commit selector → completion.
type SetupModel struct {
	step      int         // Current step (0-2)
	cancelled bool        // User cancelled via ESC
	result    SetupResult // Accumulated result from each step
	width     int         // Terminal width
	height    int         // Terminal height
	isProject bool        // True if --project flag set (write to ./iteratr.yml)

	// Step components
	modelStep      *ModelStep
	autoCommitStep *AutoCommitStep
	completionStep *CompletionStep
}

// RunSetup is the entry point for the setup wizard.
// It creates a standalone BubbleTea program, runs it, and returns the result.
// Returns nil result and error if user cancels or an error occurs.
func RunSetup(isProject bool) (*SetupResult, error) {
	// Create initial model
	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: isProject,
	}

	// Create BubbleTea program
	p := tea.NewProgram(m)

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("setup wizard failed: %w", err)
	}

	// Extract result from final model
	setupModel, ok := finalModel.(*SetupModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// Check if user cancelled
	if setupModel.cancelled {
		return nil, fmt.Errorf("setup wizard cancelled by user")
	}

	return &setupModel.result, nil
}

// Init initializes the setup model.
func (m *SetupModel) Init() tea.Cmd {
	// Initialize model step (step 0)
	m.modelStep = NewModelStep()
	return m.modelStep.Init()
}

// Update handles messages for the setup wizard.
func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Global keybindings
		switch msg.String() {
		case "ctrl+c":
			// Always allow Ctrl+C to quit
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			// ESC behavior depends on step
			if m.step == 0 {
				// On first step, exit wizard
				m.cancelled = true
				return m, tea.Quit
			} else {
				// On other steps, go back
				return m.goBack()
			}
		}

	case tea.WindowSizeMsg:
		// Store terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		// Update size of current step
		m.updateCurrentStepSize()
		return m, nil

	case ModelSelectedMsg:
		// Model selected in step 0
		m.result.Model = msg.ModelID
		m.step++
		m.initCurrentStep()
		return m, m.autoCommitStep.Init()

	case AutoCommitSelectedMsg:
		// Auto-commit selected in step 1
		m.result.AutoCommit = msg.Enabled
		// Write config file
		if err := m.WriteConfig(); err != nil {
			// On error, just quit (error handling could be improved)
			m.cancelled = true
			return m, tea.Quit
		}
		// Move to completion step
		m.step++
		m.initCurrentStep()
		return m, m.completionStep.Init()

	case ContentChangedMsg:
		// A step's content changed, recalculate modal dimensions
		m.updateCurrentStepSize()
		return m, nil
	}

	// Forward to current step
	var cmd tea.Cmd
	switch m.step {
	case 0:
		if m.modelStep != nil {
			cmd = m.modelStep.Update(msg)
		}
	case 1:
		if m.autoCommitStep != nil {
			cmd = m.autoCommitStep.Update(msg)
		}
	case 2:
		if m.completionStep != nil {
			cmd = m.completionStep.Update(msg)
		}
	}

	return m, cmd
}

// goBack returns to the previous step.
func (m *SetupModel) goBack() (tea.Model, tea.Cmd) {
	if m.step > 0 {
		m.step--
		m.initCurrentStep()
	}
	return m, nil
}

// initCurrentStep initializes the current step component if not already initialized.
func (m *SetupModel) initCurrentStep() {
	switch m.step {
	case 0:
		if m.modelStep == nil {
			m.modelStep = NewModelStep()
		}
	case 1:
		if m.autoCommitStep == nil {
			m.autoCommitStep = NewAutoCommitStep()
		}
	case 2:
		if m.completionStep == nil {
			m.completionStep = NewCompletionStep(m.isProject)
		}
	}
	m.updateCurrentStepSize()
}

// getStepPreferredHeight returns the preferred content height for the current step.
func (m *SetupModel) getStepPreferredHeight() int {
	switch m.step {
	case 0:
		if m.modelStep != nil {
			return m.modelStep.PreferredHeight()
		}
	case 1:
		if m.autoCommitStep != nil {
			return m.autoCommitStep.PreferredHeight()
		}
	case 2:
		if m.completionStep != nil {
			return m.completionStep.PreferredHeight()
		}
	}
	return 10 // Default fallback
}

// calculateModalDimensions calculates the modal dimensions based on terminal size and content.
// Returns modalWidth, modalHeight, contentWidth, contentHeight.
func (m *SetupModel) calculateModalDimensions() (int, int, int, int) {
	// Calculate modal width
	modalWidth := m.width - 6
	if modalWidth < 60 {
		modalWidth = 60
	}
	if modalWidth > 100 {
		modalWidth = 100
	}

	// Content width = modal width minus padding (2 each side) minus border (1 each side)
	contentWidth := modalWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Calculate max modal height (don't overflow screen)
	maxModalHeight := m.height - 4
	if maxModalHeight < 15 {
		maxModalHeight = 15
	}

	// Modal overhead:
	// - padding top/bottom: 2
	// - border top/bottom: 2
	// - title line: 1
	// - blank after title: 1
	// Total overhead: 6
	const modalOverhead = 6

	// Get preferred content height from current step
	preferredContentHeight := m.getStepPreferredHeight()

	// Calculate ideal modal height based on content
	idealModalHeight := preferredContentHeight + modalOverhead

	// Clamp modal height between min and max
	modalHeight := idealModalHeight
	if modalHeight > maxModalHeight {
		modalHeight = maxModalHeight
	}
	if modalHeight < 15 {
		modalHeight = 15
	}

	// Calculate actual content height
	contentHeight := modalHeight - modalOverhead
	if contentHeight < 5 {
		contentHeight = 5
	}

	return modalWidth, modalHeight, contentWidth, contentHeight
}

// updateCurrentStepSize updates the size of the current step component.
func (m *SetupModel) updateCurrentStepSize() {
	_, _, contentWidth, contentHeight := m.calculateModalDimensions()

	switch m.step {
	case 0:
		if m.modelStep != nil {
			m.modelStep.SetSize(contentWidth, contentHeight)
		}
	case 1:
		if m.autoCommitStep != nil {
			m.autoCommitStep.SetSize(contentWidth, contentHeight)
		}
	case 2:
		if m.completionStep != nil {
			m.completionStep.SetSize(contentWidth, contentHeight)
		}
	}
}

// View renders the setup wizard UI.
func (m *SetupModel) View() tea.View {
	var view tea.View
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion // Enable mouse clicks
	view.KeyboardEnhancements = tea.KeyboardEnhancements{
		ReportEventTypes: true,
	}

	// Render current step content
	var stepContent string
	switch m.step {
	case 0:
		if m.modelStep != nil {
			stepContent = m.modelStep.View()
		}
	case 1:
		if m.autoCommitStep != nil {
			stepContent = m.autoCommitStep.View()
		}
	case 2:
		if m.completionStep != nil {
			stepContent = m.completionStep.View()
		}
	}

	// Wrap in modal container with title
	content := m.renderModal(stepContent)

	canvas := uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(content).Draw(canvas, uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: m.width, Y: m.height},
	})

	view.Content = lipgloss.NewLayer(canvas.Render())
	return view
}

// renderModal wraps the step content in a modal container with title and step indicator.
func (m *SetupModel) renderModal(stepContent string) string {
	// Calculate modal dimensions dynamically based on content and terminal size
	modalWidth, modalHeight, _, _ := m.calculateModalDimensions()

	var sections []string

	// Title with step indicator and step name
	stepNames := []string{
		"Select Model",
		"Auto-Commit",
		"Complete",
	}
	// Don't show step indicator on completion screen
	var title string
	if m.step == 2 {
		title = "Setup Complete"
	} else {
		title = fmt.Sprintf("Setup - Step %d of 2: %s", m.step+1, stepNames[m.step])
	}
	sections = append(sections, theme.Current().S().ModalTitle.Render(title))
	sections = append(sections, "")

	// Step content
	sections = append(sections, stepContent)

	// Join all sections
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Apply modal container style with fixed dimensions
	modalStyle := theme.Current().S().ModalContainer.Width(modalWidth).Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Center the modal on screen
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modalContent,
	)
}

// WriteConfig writes the setup result to the appropriate config file.
func (m *SetupModel) WriteConfig() error {
	cfg := &config.Config{
		Model:      m.result.Model,
		AutoCommit: m.result.AutoCommit,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		LogFile:    "",
		Iterations: 0,
		Headless:   false,
		Template:   "",
	}

	if m.isProject {
		return config.WriteProject(cfg)
	}
	return config.WriteGlobal(cfg)
}

// ModelSelectedMsg is sent when a model is selected in step 0.
type ModelSelectedMsg struct {
	ModelID string
}

// AutoCommitSelectedMsg is sent when auto-commit choice is made in step 1.
type AutoCommitSelectedMsg struct {
	Enabled bool
}
