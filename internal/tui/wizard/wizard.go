package wizard

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// ContentChangedMsg is sent when a step's content changes in a way that affects preferred height.
// The wizard handles this by recalculating modal dimensions.
type ContentChangedMsg struct{}

// WizardResult holds the output values from the wizard.
// These are applied to buildFlags before orchestrator creation.
type WizardResult struct {
	SpecPath    string // Path to selected spec file
	Model       string // Selected model ID (e.g. "anthropic/claude-sonnet-4-5")
	Template    string // Full edited template content
	SessionName string // Validated session name
	Iterations  int    // Max iterations (0 = infinite)
	ResumeMode  bool   // True if resuming existing session (skip spec/model/template setup)
}

// WizardModel is the main BubbleTea model for the build wizard.
// It manages the five-step flow: session selector → file picker → model selector → template editor → config.
type WizardModel struct {
	step      int          // Current step (0-4)
	cancelled bool         // User cancelled via ESC
	result    WizardResult // Accumulated result from each step
	width     int          // Terminal width
	height    int          // Terminal height

	// Session store for session operations
	sessionStore *session.Store
	// Template path from config (empty = use default)
	templatePath string

	// Step components
	sessionSelectorStep *SessionSelectorStep
	filePickerStep      *FilePickerStep
	modelSelectorStep   *ModelSelectorStep
	templateEditorStep  *TemplateEditorStep
	configStep          *ConfigStep

	// Button bar with focus tracking
	buttonBar     *ButtonBar // Current button bar instance
	buttonFocused bool       // True if buttons have focus (vs step content)
}

// RunWizard is the entry point for the build wizard.
// It creates a standalone BubbleTea program, runs it, and returns the result.
// templatePath is the custom template path from config (empty string means use default).
// Returns nil result and error if user cancels or an error occurs.
func RunWizard(ctx context.Context, sessionStore *session.Store, templatePath string) (*WizardResult, error) {
	// Create initial model
	m := &WizardModel{
		step:         0,
		cancelled:    false,
		sessionStore: sessionStore,
		templatePath: templatePath,
	}

	// Create BubbleTea program with context for graceful cancellation
	p := tea.NewProgram(m, tea.WithContext(ctx))

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard failed: %w", err)
	}

	// Extract result from final model
	wizModel, ok := finalModel.(*WizardModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// Check if user cancelled
	if wizModel.cancelled {
		return nil, fmt.Errorf("wizard cancelled by user")
	}

	return &wizModel.result, nil
}

// Init initializes the wizard model.
func (m *WizardModel) Init() tea.Cmd {
	// Initialize session selector (step 0)
	m.sessionSelectorStep = NewSessionSelectorStep(m.sessionStore)
	return m.sessionSelectorStep.Init()
}

// Update handles messages for the wizard.
func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Handle button-focused keyboard input
		if m.buttonFocused && m.buttonBar != nil {
			switch msg.String() {
			case "tab", "right":
				// Cycle to next button, wrap to content if at end
				if !m.buttonBar.FocusNext() {
					m.buttonFocused = false
					m.buttonBar.Blur()
					return m, m.focusStepContentFirst()
				}
				return m, nil
			case "shift+tab", "left":
				// Cycle to previous button, wrap to content if at start
				if !m.buttonBar.FocusPrev() {
					m.buttonFocused = false
					m.buttonBar.Blur()
					return m, m.focusStepContentLast()
				}
				return m, nil
			case "enter", " ":
				// Activate focused button
				return m.activateButton(m.buttonBar.FocusedButton())
			}
		}

		// Global keybindings
		switch msg.String() {
		case "ctrl+c":
			// Always allow Ctrl+C to quit
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			// ESC behavior depends on step and state
			if m.step == 0 {
				// On first step, check if session selector is in confirmation state
				if m.sessionSelectorStep != nil && m.sessionSelectorStep.IsConfirming() {
					// Let session selector handle ESC (return to listing)
					cmd := m.sessionSelectorStep.Update(msg)
					return m, cmd
				}
				// Not confirming - exit wizard
				m.cancelled = true
				return m, tea.Quit
			} else {
				// On other steps, go back
				return m.goBack()
			}
		case "ctrl+enter":
			// Ctrl+Enter finishes wizard if all steps valid
			if m.isComplete() {
				return m, tea.Quit
			}
		case "tab":
			// Tab moves focus to buttons (unless already there)
			// For step 4 (config), let config_step handle Tab internally first
			if !m.buttonFocused && m.step != 4 {
				m.buttonFocused = true
				m.blurStepContent()
				m.ensureButtonBar()
				m.buttonBar.FocusFirst() // Start at first button (Back) for sequential cycling
				return m, nil
			}
		case "shift+tab":
			// Shift+Tab from content wraps to buttons (from the end)
			// For step 4 (config), let config_step handle Shift+Tab internally first
			if !m.buttonFocused && m.step != 4 {
				m.buttonFocused = true
				m.blurStepContent()
				m.ensureButtonBar()
				m.buttonBar.FocusLast() // Start at last button (Next) for reverse cycling
				return m, nil
			}
		}

	case tea.MouseClickMsg:
		// Handle mouse clicks on buttons
		mouse := msg.Mouse()
		if mouse.Button == tea.MouseLeft && m.buttonBar != nil {
			if btnID := m.buttonBar.ButtonAtPosition(mouse.X, mouse.Y); btnID != ButtonNone {
				return m.activateButton(btnID)
			}
		}

	case tea.WindowSizeMsg:
		// Store terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		// Update size of current step
		m.updateCurrentStepSize()
		return m, nil

	case SessionSelectedMsg:
		// Session selected in step 0
		if msg.IsNew {
			// New session selected - proceed to file picker (step 1)
			m.step++
			m.buttonFocused = false
			m.initCurrentStep()
			return m, nil
		} else {
			// Existing session selected - exit wizard in resume mode
			m.result.SessionName = msg.Name
			m.result.ResumeMode = true
			return m, tea.Quit
		}

	case FileSelectedMsg:
		// File selected in step 1
		m.result.SpecPath = msg.Path
		m.step++
		m.buttonFocused = false
		m.initCurrentStep()
		return m, m.modelSelectorStep.Init()

	case ModelSelectedMsg:
		// Model selected in step 2
		m.result.Model = msg.ModelID
		m.step++
		m.buttonFocused = false
		m.initCurrentStep()
		return m, m.templateEditorStep.Init()

	case ConfigCompleteMsg:
		// Config complete in step 4
		// Only set template if user edited it; otherwise let config template be used
		if m.templateEditorStep.WasEdited() {
			m.result.Template = m.templateEditorStep.Content()
		}
		m.result.SessionName = m.configStep.SessionName()
		m.result.Iterations = m.configStep.Iterations()
		return m, tea.Quit

	case TabExitForwardMsg:
		// Tab pressed on last input in config step - move to buttons
		m.buttonFocused = true
		m.blurStepContent()
		m.ensureButtonBar()
		m.buttonBar.FocusFirst() // Start at first button (Back) for sequential cycling
		return m, nil

	case TabExitBackwardMsg:
		// Shift+Tab pressed on first input in config step - move to buttons from end
		m.buttonFocused = true
		m.blurStepContent()
		m.ensureButtonBar()
		m.buttonBar.FocusLast() // Start at last button (Next) for reverse cycling
		return m, nil

	case ContentChangedMsg:
		// A step's content changed, recalculate modal dimensions
		m.updateCurrentStepSize()
		return m, nil
	}

	// Forward to current step (only if not button focused)
	if m.buttonFocused {
		return m, nil
	}

	var cmd tea.Cmd
	switch m.step {
	case 0:
		if m.sessionSelectorStep != nil {
			cmd = m.sessionSelectorStep.Update(msg)
		}
	case 1:
		if m.filePickerStep != nil {
			cmd = m.filePickerStep.Update(msg)
		}
	case 2:
		if m.modelSelectorStep != nil {
			cmd = m.modelSelectorStep.Update(msg)
		}
	case 3:
		if m.templateEditorStep != nil {
			cmd = m.templateEditorStep.Update(msg)
		}
		// Handle Enter to advance from template editor
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "enter" {
			// Move to config step
			m.step++
			m.buttonFocused = false
			m.initCurrentStep()
			return m, m.configStep.Init()
		}
	case 4:
		if m.configStep != nil {
			cmd = m.configStep.Update(msg)
		}
	}

	return m, cmd
}

// activateButton performs the action for the given button.
func (m *WizardModel) activateButton(btnID ButtonID) (tea.Model, tea.Cmd) {
	switch btnID {
	case ButtonBack:
		if m.step == 0 {
			// On first step, check if in confirmation state
			if m.sessionSelectorStep != nil && m.sessionSelectorStep.IsConfirming() {
				// Return to session listing
				m.sessionSelectorStep.ReturnToListing()
				return m, nil
			}
			// Cancel wizard
			m.cancelled = true
			return m, tea.Quit
		}
		return m.goBack()
	case ButtonNext:
		return m.goNext()
	}
	return m, nil
}

// goBack returns to the previous step.
func (m *WizardModel) goBack() (tea.Model, tea.Cmd) {
	if m.step > 0 {
		m.step--
		m.buttonFocused = false
		m.initCurrentStep()
	}
	return m, nil
}

// goNext advances to the next step if valid.
func (m *WizardModel) goNext() (tea.Model, tea.Cmd) {
	if !m.isStepValid() {
		return m, nil
	}

	switch m.step {
	case 0:
		// Session selector - trigger the current selection (same as Enter key)
		if m.sessionSelectorStep != nil {
			cmd := m.sessionSelectorStep.TriggerSelection()
			return m, cmd
		}
		return m, nil
	case 1:
		// File picker - emit selection message
		if m.filePickerStep != nil {
			path := m.filePickerStep.SelectedPath()
			if path != "" {
				m.result.SpecPath = path
				m.step++
				m.buttonFocused = false
				m.initCurrentStep()
				return m, m.modelSelectorStep.Init()
			}
		}
	case 2:
		// Model selector - emit selection message
		if m.modelSelectorStep != nil {
			modelID := m.modelSelectorStep.SelectedModel()
			if modelID != "" {
				m.result.Model = modelID
				m.step++
				m.buttonFocused = false
				m.initCurrentStep()
				return m, m.templateEditorStep.Init()
			}
		}
	case 3:
		// Template editor - move to config
		m.step++
		m.buttonFocused = false
		m.initCurrentStep()
		return m, m.configStep.Init()
	case 4:
		// Config step - finish wizard
		if m.configStep != nil && m.configStep.IsValid() {
			// Only set template if user edited it; otherwise let config template be used
			if m.templateEditorStep.WasEdited() {
				m.result.Template = m.templateEditorStep.Content()
			}
			m.result.SessionName = m.configStep.SessionName()
			m.result.Iterations = m.configStep.Iterations()
			return m, tea.Quit
		}
	}
	return m, nil
}

// blurStepContent removes focus from the current step's content.
func (m *WizardModel) blurStepContent() {
	if m.step == 4 && m.configStep != nil {
		// Blur the config step's text inputs
		m.configStep.Blur()
	}
}

// focusStepContentFirst gives focus to the current step's first focusable item.
func (m *WizardModel) focusStepContentFirst() tea.Cmd {
	if m.step == 4 && m.configStep != nil {
		return m.configStep.Focus()
	}
	return nil
}

// focusStepContentLast gives focus to the current step's last focusable item.
func (m *WizardModel) focusStepContentLast() tea.Cmd {
	if m.step == 4 && m.configStep != nil {
		return m.configStep.FocusLast()
	}
	return nil
}

// ensureButtonBar creates the button bar if it doesn't exist.
func (m *WizardModel) ensureButtonBar() {
	modalWidth := m.width - 6
	if modalWidth < 60 {
		modalWidth = 60
	}
	if m.step != 3 && modalWidth > 100 {
		modalWidth = 100
	}

	var buttons []Button
	nextLabel := "Next →"
	isValid := m.isStepValid()

	switch m.step {
	case 0:
		buttons = CreateCancelNextButtons(isValid, nextLabel)
	case 4:
		nextLabel = "Finish"
		buttons = CreateBackNextButtons(true, isValid, nextLabel)
	default:
		buttons = CreateBackNextButtons(true, isValid, nextLabel)
	}

	m.buttonBar = NewButtonBar(buttons)
	m.buttonBar.SetWidth(modalWidth)
}

// initCurrentStep initializes the current step component if not already initialized.
func (m *WizardModel) initCurrentStep() {
	switch m.step {
	case 0:
		if m.sessionSelectorStep == nil {
			m.sessionSelectorStep = NewSessionSelectorStep(m.sessionStore)
		}
	case 1:
		if m.filePickerStep == nil {
			m.filePickerStep = NewFilePickerStep()
		}
	case 2:
		if m.modelSelectorStep == nil {
			m.modelSelectorStep = NewModelSelectorStep()
		}
	case 3:
		if m.templateEditorStep == nil {
			m.templateEditorStep = NewTemplateEditorStep(m.templatePath)
		}
	case 4:
		if m.configStep == nil {
			m.configStep = NewConfigStep(m.result.SpecPath, m.sessionStore)
		}
	}
	m.updateCurrentStepSize()
}

// getStepPreferredHeight returns the preferred content height for the current step.
func (m *WizardModel) getStepPreferredHeight() int {
	switch m.step {
	case 0:
		if m.sessionSelectorStep != nil {
			return m.sessionSelectorStep.PreferredHeight()
		}
	case 1:
		if m.filePickerStep != nil {
			return m.filePickerStep.PreferredHeight()
		}
	case 2:
		if m.modelSelectorStep != nil {
			return m.modelSelectorStep.PreferredHeight()
		}
	case 3:
		if m.templateEditorStep != nil {
			return m.templateEditorStep.PreferredHeight()
		}
	case 4:
		if m.configStep != nil {
			return m.configStep.PreferredHeight()
		}
	}
	return 15 // Default fallback
}

// calculateModalDimensions calculates the modal dimensions based on terminal size and content.
// Returns modalWidth, modalHeight, contentWidth, contentHeight.
func (m *WizardModel) calculateModalDimensions() (int, int, int, int) {
	// Calculate modal width
	modalWidth := m.width - 6
	if modalWidth < 60 {
		modalWidth = 60
	}
	if m.step != 3 && modalWidth > 100 {
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
	// - blank before buttons: 1
	// - button bar: 1
	// Total overhead: 8
	const modalOverhead = 8

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
func (m *WizardModel) updateCurrentStepSize() {
	_, _, contentWidth, contentHeight := m.calculateModalDimensions()

	switch m.step {
	case 0:
		if m.sessionSelectorStep != nil {
			m.sessionSelectorStep.SetSize(contentWidth, contentHeight)
		}
	case 1:
		if m.filePickerStep != nil {
			m.filePickerStep.SetSize(contentWidth, contentHeight)
		}
	case 2:
		if m.modelSelectorStep != nil {
			m.modelSelectorStep.SetSize(contentWidth, contentHeight)
		}
	case 3:
		if m.templateEditorStep != nil {
			m.templateEditorStep.SetSize(contentWidth, contentHeight)
		}
	case 4:
		if m.configStep != nil {
			m.configStep.SetSize(contentWidth, contentHeight)
		}
	}
}

// View renders the wizard UI.
func (m *WizardModel) View() tea.View {
	var view tea.View
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion // Enable mouse clicks
	view.KeyboardEnhancements = tea.KeyboardEnhancements{
		ReportEventTypes: true, // Required for ctrl+enter
	}

	// Render current step content
	var stepContent string
	switch m.step {
	case 0:
		if m.sessionSelectorStep != nil {
			stepContent = m.sessionSelectorStep.View()
		}
	case 1:
		if m.filePickerStep != nil {
			stepContent = m.filePickerStep.View()
		}
	case 2:
		if m.modelSelectorStep != nil {
			stepContent = m.modelSelectorStep.View()
		}
	case 3:
		if m.templateEditorStep != nil {
			stepContent = m.templateEditorStep.View()
		}
	case 4:
		if m.configStep != nil {
			stepContent = m.configStep.View()
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

	// Set cursor position from the focused input component.
	// The textinput.Cursor() returns coordinates relative to the input itself.
	// We must offset by the modal's screen position and internal styling.
	view.Cursor = m.getCursor()

	return view
}

// getCursor returns the adjusted cursor position for the currently focused input.
func (m *WizardModel) getCursor() *tea.Cursor {
	if m.buttonFocused {
		return nil
	}

	// Get raw cursor from the current step
	var cur *tea.Cursor
	switch m.step {
	case 2:
		if m.modelSelectorStep != nil {
			cur = m.modelSelectorStep.Cursor()
		}
	case 4:
		if m.configStep != nil {
			cur = m.configStep.Cursor()
		}
	}

	if cur == nil {
		return nil
	}

	// Calculate modal position and styling offsets
	modalWidth, modalHeight, _, _ := m.calculateModalDimensions()
	modalStyle := theme.Current().S().ModalContainer

	// Modal is centered on screen
	modalX := (m.width - modalWidth) / 2
	modalY := (m.height - modalHeight) / 2

	// Offset for modal border and padding
	borderLeft := modalStyle.GetBorderLeftSize()
	paddingLeft := modalStyle.GetPaddingLeft()
	borderTop := modalStyle.GetBorderTopSize()
	paddingTop := modalStyle.GetPaddingTop()

	// Lines inside modal before step content:
	// - title line: 1
	// - blank line after title: 1
	const linesBeforeContent = 2

	cur.X += modalX + borderLeft + paddingLeft
	cur.Y += modalY + borderTop + paddingTop + linesBeforeContent

	return cur
}

// renderModal wraps the step content in a modal container with title, buttons, and step indicator.
func (m *WizardModel) renderModal(stepContent string) string {
	// Calculate modal dimensions dynamically based on content and terminal size
	modalWidth, modalHeight, _, _ := m.calculateModalDimensions()

	// Calculate modal position (centered on screen)
	modalX := (m.width - modalWidth) / 2
	modalY := (m.height - modalHeight) / 2

	var sections []string

	// Title with step indicator and step name
	stepNames := []string{
		"Select Session",
		"Select Spec File",
		"Select Model",
		"Edit Prompt Template",
		"Session Configuration",
	}
	title := fmt.Sprintf("Build Wizard - Step %d of 5: %s", m.step+1, stepNames[m.step])
	sections = append(sections, theme.Current().S().ModalTitle.Render(title))
	sections = append(sections, "")

	// Step content
	sections = append(sections, stepContent)

	// Add spacing before buttons
	sections = append(sections, "")

	// Calculate button Y position relative to modal content
	stepLines := strings.Count(stepContent, "\n") + 1
	buttonContentY := 1 + 1 + stepLines + 1 // title + blank + content + blank

	// Add button bar based on current step
	buttonBar := m.createButtonBar(modalX, modalY, modalWidth, buttonContentY)
	sections = append(sections, buttonBar)

	// Join all sections
	content := strings.Join(sections, "\n")

	// Apply modal container style with fixed dimensions
	modalStyle := theme.Current().S().ModalContainer.Width(modalWidth).Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Center the modal on screen
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modalContent,
	)
}

// createButtonBar creates the button bar for the current step.
// Buttons are context-aware based on step and validation state.
// Also calculates button hit areas for mouse click detection.
func (m *WizardModel) createButtonBar(modalX, modalY, modalWidth, contentStartY int) string {
	var buttons []Button
	nextLabel := "Next →"

	// Determine next button label and validation state
	isValid := m.isStepValid()

	switch m.step {
	case 0:
		// First step: Cancel + Next, or Back + Next if in confirmation state
		if m.sessionSelectorStep != nil && m.sessionSelectorStep.IsConfirming() {
			buttons = CreateBackNextButtons(true, isValid, nextLabel)
		} else {
			buttons = CreateCancelNextButtons(isValid, nextLabel)
		}
	case 4:
		// Last step: Back + Finish
		nextLabel = "Finish"
		buttons = CreateBackNextButtons(true, isValid, nextLabel)
	default:
		// Middle steps: Back + Next
		buttons = CreateBackNextButtons(true, isValid, nextLabel)
	}

	// Create or update button bar, preserving focus state
	focusIndex := -1
	if m.buttonBar != nil && m.buttonFocused {
		focusIndex = m.buttonBar.focusIndex
	}

	m.buttonBar = NewButtonBar(buttons)
	m.buttonBar.SetWidth(modalWidth)

	// Restore focus state
	if focusIndex >= 0 {
		m.buttonBar.focusIndex = focusIndex
	}

	// Calculate button hit areas for mouse clicks
	// Buttons are centered in the modal width
	// Button format: [margin][padding]Label[padding][margin]
	// Each button: 1 margin + 2 padding + label + 2 padding + 1 margin = label + 6
	btn1Width := len(buttons[0].Label) + 6
	btn2Width := len(buttons[1].Label) + 6
	totalButtonWidth := btn1Width + btn2Width

	// Center offset within modal
	centerOffset := (modalWidth - totalButtonWidth) / 2

	// Modal content has padding (2 on each side) + border (1 on each side) = 3 from edge
	// Actually modalX is already the left edge of the modal on screen
	// The content inside has padding, so buttonBarX = modalX + padding
	padding := 2 // lipgloss padding
	border := 1  // lipgloss border

	btn1X := modalX + border + padding + centerOffset
	btn2X := btn1X + btn1Width

	// Y position: modalY + border + padding + contentStartY (lines from top of content area)
	btnY := modalY + border + padding + contentStartY

	areas := []uv.Rectangle{
		{
			Min: uv.Position{X: btn1X, Y: btnY},
			Max: uv.Position{X: btn1X + btn1Width, Y: btnY + 1},
		},
		{
			Min: uv.Position{X: btn2X, Y: btnY},
			Max: uv.Position{X: btn2X + btn2Width, Y: btnY + 1},
		},
	}
	m.buttonBar.SetButtonAreas(areas)

	return m.buttonBar.Render()
}

// isStepValid checks if the current step has valid data.
// Used to enable/disable the Next button.
func (m *WizardModel) isStepValid() bool {
	switch m.step {
	case 0:
		// Session selector: valid if loaded, no error, and not in confirmation state
		if m.sessionSelectorStep != nil {
			return m.sessionSelectorStep.IsReady()
		}
		return false
	case 1:
		// File picker: valid if a file (not directory) is selected
		if m.filePickerStep != nil {
			return m.filePickerStep.SelectedPath() != ""
		}
		return false
	case 2:
		// Model selector: valid if a model is selected and not loading/error
		if m.modelSelectorStep != nil {
			return m.modelSelectorStep.SelectedModel() != ""
		}
		return false
	case 3:
		// Template editor: always valid (can have empty template)
		return true
	case 4:
		// Config: valid if all inputs pass validation
		if m.configStep != nil {
			return m.configStep.IsValid()
		}
		return false
	}
	return false
}

// isComplete checks if all required steps have valid data.
func (m *WizardModel) isComplete() bool {
	// Template is optional here - can come from config if not edited in wizard
	return m.result.SpecPath != "" &&
		m.result.Model != "" &&
		m.result.SessionName != ""
}
