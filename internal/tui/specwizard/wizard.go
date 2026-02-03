package specwizard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/gosimple/slug"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/specmcp"
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

// ProgramSender is an interface for sending messages to the Bubbletea program.
// This allows for easier testing by mocking the Send method.
type ProgramSender interface {
	Send(tea.Msg)
}

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
	ctx       context.Context // Context for ACP operations

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

	// Agent infrastructure
	mcpServer   *specmcp.Server
	agentRunner *agent.Runner
	agentError  *error // Error from agent startup or runtime

	// Program reference for sending messages from callbacks
	program ProgramSender
}

// Run is the entry point for the spec wizard.
// It creates a standalone BubbleTea program, runs it, and returns any error.
func Run(cfg *config.Config) error {
	ctx := context.Background()

	m := &WizardModel{
		step:      StepTitle,
		cancelled: false,
		cfg:       cfg,
		ctx:       ctx,
	}

	p := tea.NewProgram(m)
	m.program = p // Store program reference for callbacks
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

	// Clean up agent resources
	if wizModel.agentRunner != nil {
		wizModel.agentRunner.Stop()
	}
	if wizModel.mcpServer != nil {
		_ = wizModel.mcpServer.Stop()
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
			// During agent step, only handle ESC at wizard level if agent isn't running
			// If agent is running (agentStep exists and waitingForAgent or has questions),
			// the agent step will handle ESC by showing confirmation modal
			if m.step == StepAgent && m.agentStep != nil && (m.agentStep.waitingForAgent || m.agentStep.questionView != nil) {
				// Pass through to agent step - it will show the cancel modal
				break
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
		cmd := m.initCurrentStep()
		return m, cmd

	case DescriptionSubmittedMsg:
		// Description submitted, advance to model selection
		m.result.Description = msg.Description
		m.step = StepModel
		m.buttonFocused = false
		cmd := m.initCurrentStep()
		return m, cmd

	case wizard.ModelSelectedMsg:
		// Model selected, advance to agent phase
		m.result.Model = msg.ModelID
		m.step = StepAgent
		m.buttonFocused = false
		m.initCurrentStep()
		return m, m.startAgentPhase

	case SpecContentReceivedMsg:
		// Spec content received from agent, advance to review
		m.result.SpecContent = msg.Content
		m.step = StepReview
		m.buttonFocused = false
		cmd := m.initCurrentStep()
		return m, cmd

	case SpecSavedMsg:
		// Spec saved, advance to completion
		m.result.SpecPath = msg.Path
		m.step = StepCompletion
		m.buttonFocused = false
		cmd := m.initCurrentStep()
		return m, cmd

	case AgentErrorMsg:
		// Agent failed to start or encountered error
		logger.Error("Agent error: %v", msg.Err)
		// Show error to user with helpful message
		m.agentError = &msg.Err
		return m, nil

	case RestartWizardMsg:
		// User confirmed restart - go back to title step
		logger.Debug("Restarting wizard from title step")
		m.step = StepTitle
		m.buttonFocused = false
		m.agentError = nil
		// Reset result
		m.result = WizardResult{}
		// Clean up agent resources
		if m.agentRunner != nil {
			m.agentRunner.Stop()
			m.agentRunner = nil
		}
		if m.mcpServer != nil {
			_ = m.mcpServer.Stop()
			m.mcpServer = nil
		}
		cmd := m.initCurrentStep()
		return m, cmd

	case CancelWizardMsg:
		// User confirmed cancellation during agent phase
		logger.Debug("Cancelling wizard from agent phase")
		m.cancelled = true
		// Clean up agent resources before quitting
		if m.agentRunner != nil {
			m.agentRunner.Stop()
			m.agentRunner = nil
		}
		if m.mcpServer != nil {
			_ = m.mcpServer.Stop()
			m.mcpServer = nil
		}
		return m, tea.Quit

	case CheckFileExistsMsg:
		// Check if spec file exists before saving
		logger.Debug("Checking if spec file exists")

		// Generate file path
		slugTitle := slug.Make(m.result.Title)
		if slugTitle == "" {
			slugTitle = "unnamed-spec"
		}
		specPath := filepath.Join(m.cfg.SpecDir, slugTitle+".md")

		// Check if file exists
		if _, err := os.Stat(specPath); err == nil {
			// File exists - show overwrite confirmation in review step
			logger.Debug("File %s already exists, showing overwrite confirmation", specPath)
			if m.reviewStep != nil {
				m.reviewStep.showConfirmOverwrite = true
			}
			return m, nil
		}

		// File doesn't exist - proceed with save
		logger.Debug("File doesn't exist, proceeding with save")
		return m, func() tea.Msg {
			return SaveSpecMsg{}
		}

	case SaveSpecMsg:
		// User clicked Save button in review step (or confirmed overwrite)
		logger.Debug("Save spec button clicked")

		// Save spec to file and update README
		specPath, err := saveSpec(m.cfg.SpecDir, m.result.Title, m.result.Description, m.result.SpecContent)
		if err != nil {
			logger.Error("Failed to save spec: %v", err)
			// Show error to user
			// TODO: Add error handling UI (could show error modal)
			return m, nil
		}

		logger.Debug("Spec saved to %s", specPath)

		// Advance to completion step with saved path
		return m, func() tea.Msg {
			return SpecSavedMsg{Path: specPath}
		}

	case StartBuildMsg:
		// User clicked Start Build button in completion step
		logger.Debug("Start Build button clicked, launching build with spec: %s", m.result.SpecPath)

		// Execute iteratr build --spec <path> directly
		// This will spawn a new iteratr build process and exit the spec wizard
		return m, func() tea.Msg {
			return ExecBuildMsg{SpecPath: m.result.SpecPath}
		}

	case ExecBuildMsg:
		// Execute iteratr build command in a new process
		// First quit the TUI to restore terminal, then exec the build command
		return m, tea.Sequence(
			tea.Quit,
			m.execBuild(msg.SpecPath),
		)

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

// initCurrentStep initializes the current step component and returns any init commands.
func (m *WizardModel) initCurrentStep() tea.Cmd {
	var cmd tea.Cmd
	switch m.step {
	case StepTitle:
		m.titleStep = NewTitleStep()
		cmd = m.titleStep.Init()
	case StepDescription:
		m.descriptionStep = NewDescriptionStep()
		cmd = m.descriptionStep.Init()
	case StepModel:
		m.modelStep = wizard.NewModelSelectorStep()
		cmd = m.modelStep.Init()
	case StepAgent:
		// Agent phase initialized by startAgentPhase(), but create placeholder if needed
		if m.agentStep == nil && m.mcpServer != nil {
			m.agentStep = NewAgentPhase(m.mcpServer)
		}
	case StepReview:
		m.reviewStep = NewReviewStep(m.result.SpecContent, m.cfg)
	case StepCompletion:
		m.completionStep = NewCompletionStep(m.result.SpecPath)
	}
	m.updateCurrentStepSize()
	return cmd
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

	// If there's an agent error, show error screen
	if m.agentError != nil {
		return m.renderErrorScreen(*m.agentError)
	}

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

	// Button bar (for steps that have buttons)
	var buttonBarContent string
	if m.hasButtons() {
		m.ensureButtonBar()
		buttonBarContent = m.buttonBar.Render()
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

	var content string
	if buttonBarContent != "" {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			stepContent,
			"",
			buttonBarContent,
			"",
			hint,
		)
	} else {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			stepContent,
			"",
			hint,
		)
	}

	return modalStyle.Render(content)
}

// renderErrorScreen renders an error screen with helpful troubleshooting info.
func (m *WizardModel) renderErrorScreen(err error) string {
	currentTheme := theme.Current()

	// Parse error for user-friendly message
	errorMsg := err.Error()
	var helpText string

	if strings.Contains(errorMsg, "failed to start opencode") ||
		strings.Contains(errorMsg, "executable file not found") ||
		strings.Contains(errorMsg, "no such file or directory") {
		helpText = `opencode is not installed or not in PATH.

Install opencode:
  npm install -g opencode

Verify installation:
  opencode --version`
	} else if strings.Contains(errorMsg, "failed to start MCP server") ||
		strings.Contains(errorMsg, "failed to find available port") {
		helpText = `Failed to start internal MCP server.

Possible causes:
  • No available ports (unlikely)
  • System firewall blocking local connections
  • Too many open files (check ulimit)

Try restarting the wizard.`
	} else if strings.Contains(errorMsg, "ACP initialize failed") ||
		strings.Contains(errorMsg, "ACP new session failed") {
		helpText = `Failed to initialize agent communication.

Possible causes:
  • opencode version mismatch
  • Corrupted opencode installation

Try reinstalling opencode:
  npm install -g opencode`
	} else {
		// Generic error
		helpText = `An unexpected error occurred.

Please check the logs for more details.`
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(currentTheme.Error)).
		MarginBottom(1)
	title := titleStyle.Render("⚠ Agent Startup Failed")

	// Error message
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgBase)).
		MarginBottom(1)
	errorText := errorStyle.Render(fmt.Sprintf("Error: %s", errorMsg))

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgMuted)).
		MarginBottom(1)
	help := helpStyle.Render(helpText)

	// Hint
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgMuted))
	hint := hintStyle.Render("Press ESC or Ctrl+C to exit")

	// Combine
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		errorText,
		"",
		help,
		"",
		hint,
	)

	// Modal styling
	modalStyle := lipgloss.NewStyle().
		Width(70).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(currentTheme.Error))

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
		// Special handling for review step - show confirmation modal in review step itself
		if m.step == StepReview && m.reviewStep != nil {
			m.reviewStep.showConfirmRestart = true
			return m, nil
		}

		m.step--
		m.buttonFocused = false
		cmd := m.initCurrentStep()
		return m, cmd
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

// startAgentPhase initializes the MCP server, ACP subprocess, and sends the initial prompt.
func (m *WizardModel) startAgentPhase() tea.Msg {
	// Start MCP server
	logger.Debug("Starting MCP server for spec wizard")
	m.mcpServer = specmcp.New(m.result.Title, m.cfg.SpecDir)
	port, err := m.mcpServer.Start(m.ctx)
	if err != nil {
		logger.Error("Failed to start MCP server: %v", err)
		// Add more context to help user troubleshoot
		return AgentErrorMsg{Err: fmt.Errorf("failed to start MCP server: %w", err)}
	}
	logger.Debug("MCP server started on port %d", port)

	// Initialize agent phase component with MCP server
	m.agentStep = NewAgentPhase(m.mcpServer)
	if m.width > 0 && m.height > 0 {
		m.agentStep.SetSize(m.width, m.height)
	}

	// Build MCP server URL
	mcpServerURL := fmt.Sprintf("http://localhost:%d/mcp", port)

	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get working directory: %v", err)
		return AgentErrorMsg{Err: fmt.Errorf("failed to get working directory: %w", err)}
	}

	// Create agent runner
	logger.Debug("Creating agent runner")
	m.agentRunner = agent.NewRunner(agent.RunnerConfig{
		Model:        m.result.Model,
		WorkDir:      workDir,
		SessionName:  "spec-wizard",
		NATSPort:     0, // Not using NATS for spec wizard
		MCPServerURL: mcpServerURL,
		OnText: func(text string) {
			// Agent text output - not shown in spec wizard
			logger.Debug("Agent text: %s", text)
		},
		OnToolCall: func(event agent.ToolCallEvent) {
			// Tool call events - not shown in spec wizard
			logger.Debug("Agent tool call: %s [%s]", event.Title, event.Status)
		},
		OnThinking: func(content string) {
			// Thinking output - not shown in spec wizard
			logger.Debug("Agent thinking: %s", content)
		},
		OnFinish: func(event agent.FinishEvent) {
			logger.Debug("Agent finished: %s (error: %s)", event.StopReason, event.Error)
			// Check if agent terminated with an error
			if event.StopReason == "error" || event.Error != "" {
				// Send error message to UI
				if m.program != nil {
					m.program.Send(AgentErrorMsg{Err: fmt.Errorf("agent error: %s", event.Error)})
				}
			}
		},
		OnFileChange: func(change agent.FileChange) {
			// File changes not relevant in spec wizard
		},
	})

	// Start ACP subprocess
	logger.Debug("Starting ACP subprocess")
	if err := m.agentRunner.Start(m.ctx); err != nil {
		logger.Error("Failed to start ACP: %v", err)
		// Check if it's an "executable not found" error for better messaging
		if strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			return AgentErrorMsg{Err: fmt.Errorf("failed to start opencode: executable file not found in $PATH")}
		}
		return AgentErrorMsg{Err: fmt.Errorf("failed to start opencode: %w", err)}
	}

	// Build spec prompt
	prompt := buildSpecPrompt(m.result.Title, m.result.Description)
	logger.Debug("Sending spec prompt (%d bytes)", len(prompt))

	// Send prompt in goroutine to avoid blocking
	go func() {
		err := m.agentRunner.RunIteration(m.ctx, prompt, "")
		if err != nil {
			logger.Error("Agent iteration failed: %v", err)
			// Error will be handled via OnFinish callback
		}
	}()

	return nil
}

// execBuild returns a tea.Cmd that executes iteratr build --spec <path> after the TUI quits.
// The build command is executed using syscall.Exec to replace the current process.
func (m *WizardModel) execBuild(specPath string) tea.Cmd {
	return func() tea.Msg {
		// Find the iteratr binary path
		// Use os.Executable() to get the current binary path
		execPath, err := os.Executable()
		if err != nil {
			logger.Error("Failed to get executable path: %v", err)
			// Fall back to "iteratr" in PATH
			execPath = "iteratr"
		}

		// Build the command: iteratr build --spec <path>
		cmd := exec.Command(execPath, "build", "--spec", specPath)

		// Inherit stdin, stdout, stderr from current process
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run the command and wait for it to complete
		logger.Debug("Executing: %s build --spec %s", execPath, specPath)
		if err := cmd.Run(); err != nil {
			logger.Error("Failed to execute build command: %v", err)
			fmt.Fprintf(os.Stderr, "Failed to start build: %v\n", err)
		}

		// Return nil message since we're exiting anyway
		return nil
	}
}

// buildSpecPrompt constructs the agent prompt for spec creation.
func buildSpecPrompt(title, description string) string {
	specFormat := `# [Title]

## Overview
Brief description of the feature

## User Story
Who benefits and why

## Requirements
Detailed requirements

## Technical Implementation
Implementation details

## Tasks
Byte-sized implementation tasks

## Out of Scope
What's not included in v1`

	return fmt.Sprintf(`You are helping create a feature specification.

Feature: %s
Description: %s

Follow the user instructions and interview me in detail using the ask-questions 
tool about literally anything: technical implementation, UI & UX, concerns, 
tradeoffs, edge cases, dependencies, testing, etc. Be very in-depth and continue 
interviewing me continually until you have enough information. Then write the 
complete spec using the finish-spec tool.

The spec MUST follow this format:

%s

Make the spec extremely concise. Sacrifice grammar for the sake of concision.`,
		title, description, specFormat)
}
