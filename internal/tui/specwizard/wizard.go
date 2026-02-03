package specwizard

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

// Step enumeration for wizard flow
const (
	StepTitle       = 0 // Title input
	StepDescription = 1 // Description textarea
	StepModel       = 2 // Model selection
	StepAgent       = 3 // Agent interview phase
	StepReview      = 4 // Review and edit spec
	StepCompletion  = 5 // Success screen with Build/Exit
)

// WizardResult holds the accumulated data from the wizard flow.
type WizardResult struct {
	Title       string // User-provided spec title
	Description string // User-provided description
	Model       string // Selected model ID
	SpecContent string // Generated spec content
	SpecPath    string // Final saved spec path
}

// WizardModel is the main BubbleTea model for the spec wizard.
// It manages the multi-step flow: title → description → model → agent → review → completion.
type WizardModel struct {
	step      int          // Current step (0-5)
	cancelled bool         // User cancelled via ESC
	result    WizardResult // Accumulated result from each step
	width     int          // Terminal width
	height    int          // Terminal height
	cfg       *config.Config

	// Step components
	titleStep       *TitleStep
	descriptionStep *DescriptionStep
	modelStep       *wizard.ModelSelectorStep
	agentStep       *AgentPhase
	reviewStep      *ReviewStep
	completionStep  *CompletionStep

	// Button bar with focus tracking
	buttonBar     *wizard.ButtonBar
	buttonFocused bool // True if buttons have focus (vs step content)
}

// Run is the entry point for the spec wizard.
// It creates a standalone BubbleTea program, runs it, and returns any error.
func Run(cfg *config.Config) error {
	m := &WizardModel{
		step:      StepTitle,
		cancelled: false,
		cfg:       cfg,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	wizModel, ok := finalModel.(*WizardModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if wizModel.cancelled {
		return fmt.Errorf("wizard cancelled by user")
	}

	return nil
}

// Init initializes the wizard model.
func (m *WizardModel) Init() tea.Cmd {
	// Initialize title step (step 0)
	m.titleStep = NewTitleStep()
	return m.titleStep.Init()
}

// Update handles messages for the wizard.
func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Handle button-focused keyboard input
		if m.buttonFocused && m.buttonBar != nil {
			switch msg.String() {
			case "tab", "right":
				if !m.buttonBar.FocusNext() {
					m.buttonFocused = false
					m.buttonBar.Blur()
					return m, m.focusStepContentFirst()
				}
				return m, nil
			case "shift+tab", "left":
				if !m.buttonBar.FocusPrev() {
					m.buttonFocused = false
					m.buttonBar.Blur()
					return m, m.focusStepContentLast()
				}
				return m, nil
			case "enter", " ":
				return m.activateButton(m.buttonBar.FocusedButton())
			}
		}

		// Global keybindings
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			if m.step == StepTitle {
				// On first step, cancel wizard
				m.cancelled = true
				return m, tea.Quit
			}
			// On other steps, go back
			return m.goBack()
		case "tab":
			// Tab moves focus to buttons
			if !m.buttonFocused && m.hasButtons() {
				m.buttonFocused = true
				m.blurStepContent()
				m.ensureButtonBar()
				m.buttonBar.FocusFirst()
				return m, nil
			}
		case "shift+tab":
			// Shift+Tab wraps to buttons from the end
			if !m.buttonFocused && m.hasButtons() {
				m.buttonFocused = true
				m.blurStepContent()
				m.ensureButtonBar()
				m.buttonBar.FocusLast()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateCurrentStepSize()
		return m, nil

	case TitleSubmittedMsg:
		// Title submitted, advance to description
		m.result.Title = msg.Title
		m.step = StepDescription
		m.buttonFocused = false
		m.initCurrentStep()
		return m, nil

	case DescriptionSubmittedMsg:
		// Description submitted, advance to model selection
		m.result.Description = msg.Description
		m.step = StepModel
		m.buttonFocused = false
		m.initCurrentStep()
		return m, nil

	case wizard.ModelSelectedMsg:
		// Model selected, advance to agent phase
		m.result.Model = msg.ModelID
		m.step = StepAgent
		m.buttonFocused = false
		m.initCurrentStep()
		// TODO: Start agent phase (spawn ACP, MCP server)
		return m, nil

	case SpecContentReceivedMsg:
		// Spec content received from agent, advance to review
		m.result.SpecContent = msg.Content
		m.step = StepReview
		m.buttonFocused = false
		m.initCurrentStep()
		return m, nil

	case SpecSavedMsg:
		// Spec saved, advance to completion
		m.result.SpecPath = msg.Path
		m.step = StepCompletion
		m.buttonFocused = false
		m.initCurrentStep()
		return m, nil

	case wizard.TabExitForwardMsg:
		// Tab from last input - move to buttons
		m.buttonFocused = true
		m.blurStepContent()
		m.ensureButtonBar()
		m.buttonBar.FocusFirst()
		return m, nil

	case wizard.TabExitBackwardMsg:
		// Shift+Tab from first input - move to buttons from end
		m.buttonFocused = true
		m.blurStepContent()
		m.ensureButtonBar()
		m.buttonBar.FocusLast()
		return m, nil
	}

	// Forward messages to current step
	return m.updateCurrentStep(msg)
}

// View renders the wizard.
func (m *WizardModel) View() tea.View {
	var view tea.View
	view.AltScreen = true

	if m.width == 0 || m.height == 0 {
		// Not ready to render
		view.Content = lipgloss.NewLayer("")
		return view
	}

	// Render current step content
	content := m.renderCurrentStep()

	// Center on screen
	centered := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	// Draw to canvas using ultraviolet
	canvas := uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(centered).Draw(canvas, uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: m.width, Y: m.height},
	})

	view.Content = lipgloss.NewLayer(canvas.Render())
	return view
}

// initCurrentStep initializes the current step component.
func (m *WizardModel) initCurrentStep() {
	switch m.step {
	case StepTitle:
		m.titleStep = NewTitleStep()
	case StepDescription:
		m.descriptionStep = NewDescriptionStep()
	case StepModel:
		m.modelStep = wizard.NewModelSelectorStep()
	case StepAgent:
		// TODO: Initialize agent phase (requires MCP server start)
		// For now, create placeholder that will be replaced when MCP server starts
		m.agentStep = nil
	case StepReview:
		// TODO: Initialize review step
		m.reviewStep = NewReviewStep(m.result.SpecContent, m.cfg)
	case StepCompletion:
		// TODO: Initialize completion step
		m.completionStep = NewCompletionStep(m.result.SpecPath)
	}
	m.updateCurrentStepSize()
}

// updateCurrentStep forwards a message to the current step.
func (m *WizardModel) updateCurrentStep(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.step {
	case StepTitle:
		if m.titleStep != nil {
			cmd = m.titleStep.Update(msg)
		}
	case StepDescription:
		if m.descriptionStep != nil {
			cmd = m.descriptionStep.Update(msg)
		}
	case StepModel:
		if m.modelStep != nil {
			cmd = m.modelStep.Update(msg)
		}
	case StepAgent:
		if m.agentStep != nil {
			var updatedAgent *AgentPhase
			updatedAgent, cmd = m.agentStep.Update(msg)
			m.agentStep = updatedAgent
		}
	case StepReview:
		if m.reviewStep != nil {
			cmd = m.reviewStep.Update(msg)
		}
	case StepCompletion:
		if m.completionStep != nil {
			cmd = m.completionStep.Update(msg)
		}
	}

	return m, cmd
}

// updateCurrentStepSize updates the size of the current step.
func (m *WizardModel) updateCurrentStepSize() {
	switch m.step {
	case StepTitle:
		if m.titleStep != nil {
			m.titleStep.SetSize(m.width, m.height)
		}
	case StepDescription:
		if m.descriptionStep != nil {
			m.descriptionStep.SetSize(m.width, m.height)
		}
	case StepModel:
		if m.modelStep != nil {
			m.modelStep.SetSize(m.width, m.height)
		}
	case StepAgent:
		if m.agentStep != nil {
			m.agentStep.SetSize(m.width, m.height)
		}
	case StepReview:
		if m.reviewStep != nil {
			m.reviewStep.SetSize(m.width, m.height)
		}
	case StepCompletion:
		if m.completionStep != nil {
			m.completionStep.SetSize(m.width, m.height)
		}
	}
}

// renderCurrentStep renders the content for the current step.
func (m *WizardModel) renderCurrentStep() string {
	currentTheme := theme.Current()

	// Step title
	var stepTitle string
	switch m.step {
	case StepTitle:
		stepTitle = "Spec Wizard - Step 1: Title"
	case StepDescription:
		stepTitle = "Spec Wizard - Step 2: Description"
	case StepModel:
		stepTitle = "Spec Wizard - Step 3: Model"
	case StepAgent:
		stepTitle = "Spec Wizard - Interview"
	case StepReview:
		stepTitle = "Spec Wizard - Review"
	case StepCompletion:
		stepTitle = "Spec Wizard - Complete"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(currentTheme.Primary)).
		MarginBottom(1)

	title := titleStyle.Render(stepTitle)

	// Step content
	var stepContent string
	switch m.step {
	case StepTitle:
		if m.titleStep != nil {
			stepContent = m.titleStep.View()
		}
	case StepDescription:
		if m.descriptionStep != nil {
			stepContent = m.descriptionStep.View()
		}
	case StepModel:
		if m.modelStep != nil {
			stepContent = m.modelStep.View()
		}
	case StepAgent:
		if m.agentStep != nil {
			stepContent = m.agentStep.View()
		}
	case StepReview:
		if m.reviewStep != nil {
			stepContent = m.reviewStep.View()
		}
	case StepCompletion:
		if m.completionStep != nil {
			stepContent = m.completionStep.View()
		}
	}

	// Hint
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgMuted)).
		Render("tab to navigate • esc to cancel")

	// Combine with modal styling
	modalStyle := lipgloss.NewStyle().
		Width(70).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(currentTheme.BorderDefault))

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		stepContent,
		"",
		hint,
	)

	return modalStyle.Render(content)
}

// hasButtons returns true if the current step has navigation buttons.
func (m *WizardModel) hasButtons() bool {
	// Most steps have buttons, except agent phase (has custom navigation)
	return m.step != StepAgent && m.step != StepCompletion
}

// ensureButtonBar creates the button bar if needed.
func (m *WizardModel) ensureButtonBar() {
	var buttons []wizard.Button

	// Back button (not on first step)
	if m.step > StepTitle {
		buttons = append(buttons, wizard.Button{
			Label: "← Back",
			State: wizard.ButtonNormal,
		})
	}

	// Next/Continue button
	nextLabel := "Next →"
	if m.step == StepReview {
		nextLabel = "Save"
	}
	buttons = append(buttons, wizard.Button{
		Label: nextLabel,
		State: wizard.ButtonNormal,
	})

	m.buttonBar = wizard.NewButtonBar(buttons)
}

// activateButton handles button activation.
func (m *WizardModel) activateButton(btnID wizard.ButtonID) (tea.Model, tea.Cmd) {
	switch btnID {
	case wizard.ButtonBack:
		return m.goBack()
	case wizard.ButtonNext:
		return m.goNext()
	}
	return m, nil
}

// goBack moves to the previous step.
func (m *WizardModel) goBack() (tea.Model, tea.Cmd) {
	if m.step > StepTitle {
		m.step--
		m.buttonFocused = false
		m.initCurrentStep()
	}
	return m, nil
}

// goNext moves to the next step (validates current step first).
func (m *WizardModel) goNext() (tea.Model, tea.Cmd) {
	// Validation happens via step-specific submit messages
	// This is called directly by button activation
	switch m.step {
	case StepTitle:
		if m.titleStep != nil {
			return m, m.titleStep.Submit()
		}
	case StepDescription:
		if m.descriptionStep != nil {
			return m, m.descriptionStep.Submit()
		}
	}
	return m, nil
}

// focusStepContentFirst focuses the first element in step content.
func (m *WizardModel) focusStepContentFirst() tea.Cmd {
	switch m.step {
	case StepTitle:
		if m.titleStep != nil {
			m.titleStep.Focus()
		}
	case StepDescription:
		if m.descriptionStep != nil {
			m.descriptionStep.Focus()
		}
	}
	return nil
}

// focusStepContentLast focuses the last element in step content.
func (m *WizardModel) focusStepContentLast() tea.Cmd {
	// For single-input steps, same as first
	return m.focusStepContentFirst()
}

// blurStepContent blurs all step content.
func (m *WizardModel) blurStepContent() {
	switch m.step {
	case StepTitle:
		if m.titleStep != nil {
			m.titleStep.Blur()
		}
	case StepDescription:
		if m.descriptionStep != nil {
			m.descriptionStep.Blur()
		}
	}
}
