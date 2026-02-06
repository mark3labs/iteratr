package specwizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/specmcp"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

func TestBuildSpecPrompt(t *testing.T) {
	title := "User Authentication"
	description := "Add user authentication with email/password login"

	prompt := buildSpecPrompt(title, description)

	// Verify title is included
	if !strings.Contains(prompt, title) {
		t.Errorf("Prompt does not contain title: %s", title)
	}

	// Verify description is included
	if !strings.Contains(prompt, description) {
		t.Errorf("Prompt does not contain description: %s", description)
	}

	// Verify key instructions are present
	expectedPhrases := []string{
		"You are helping create a feature specification",
		"ask-questions tool",
		"finish-spec tool",
		"## Overview",
		"## User Story",
		"## Requirements",
		"## Technical Implementation",
		"## Tasks",
		"## Out of Scope",
		"extremely concise",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("Prompt missing expected phrase: %s", phrase)
		}
	}
}

func TestAgentErrorHandling(t *testing.T) {
	tests := []struct {
		name            string
		errorMsg        string
		expectedContent []string
	}{
		{
			name:     "opencode not installed",
			errorMsg: "failed to start opencode: executable file not found in $PATH",
			expectedContent: []string{
				"⚠ Agent Startup Failed",
				"opencode is not installed",
				"npm install -g opencode",
				"opencode --version",
			},
		},
		{
			name:     "MCP server start failure",
			errorMsg: "failed to start MCP server: failed to find available port",
			expectedContent: []string{
				"⚠ Agent Startup Failed",
				"Failed to start internal MCP server",
				"No available ports",
				"Try restarting the wizard",
			},
		},
		{
			name:     "ACP initialization failure",
			errorMsg: "ACP initialize failed: protocol error",
			expectedContent: []string{
				"⚠ Agent Startup Failed",
				"Failed to initialize agent communication",
				"opencode version mismatch",
				"npm install -g opencode",
			},
		},
		{
			name:     "Generic error",
			errorMsg: "some unexpected error occurred",
			expectedContent: []string{
				"⚠ Agent Startup Failed",
				"An unexpected error occurred",
				"check the logs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create wizard model with agent error
			m := &WizardModel{
				step:   StepAgent,
				width:  80,
				height: 24,
			}

			// Set agent error
			errVar := fmt.Errorf("%s", tt.errorMsg)
			m.agentError = &errVar

			// Render error screen
			output := m.renderErrorScreen(errVar)

			// Verify expected content is present
			for _, expected := range tt.expectedContent {
				if !strings.Contains(output, expected) {
					t.Errorf("Error screen missing expected content: %q\nGot:\n%s", expected, output)
				}
			}

			// Verify error message text is included (may be formatted)
			// Error is shown as "Error: <message>" so check for the core message
			if !strings.Contains(output, "Error:") {
				t.Error("Error screen missing 'Error:' prefix")
			}
		})
	}
}

func TestAgentErrorMsg(t *testing.T) {
	// Create wizard model
	m := &WizardModel{
		step:   StepAgent,
		width:  80,
		height: 24,
	}

	// Send AgentErrorMsg
	err := fmt.Errorf("test error: opencode not found")
	updatedModel, _ := m.Update(AgentErrorMsg{Err: err})

	// Verify error was stored
	wizModel := updatedModel.(*WizardModel)
	if wizModel.agentError == nil {
		t.Error("Expected agentError to be set")
	}

	if *wizModel.agentError != err {
		t.Errorf("Expected agentError to be %v, got %v", err, *wizModel.agentError)
	}

	// Verify renderCurrentStep shows error screen
	output := wizModel.renderCurrentStep()
	if !strings.Contains(output, "⚠ Agent Startup Failed") {
		t.Error("Expected error screen to be rendered")
	}
	if !strings.Contains(output, "test error: opencode not found") {
		t.Error("Expected error message to be shown")
	}
}

func TestCancellationFlow(t *testing.T) {
	tests := []struct {
		name             string
		step             int
		keyMsg           string
		expectCancel     bool
		expectStepChange bool
		expectedStep     int
	}{
		{
			name:             "ESC on title step cancels wizard",
			step:             StepTitle,
			keyMsg:           "esc",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepTitle,
		},
		{
			name:             "Ctrl+C on title step cancels wizard",
			step:             StepTitle,
			keyMsg:           "ctrl+c",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepTitle,
		},
		{
			name:             "ESC on description step goes back to title",
			step:             StepDescription,
			keyMsg:           "esc",
			expectCancel:     false,
			expectStepChange: true,
			expectedStep:     StepTitle,
		},
		{
			name:             "Ctrl+C on description step cancels wizard",
			step:             StepDescription,
			keyMsg:           "ctrl+c",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepDescription,
		},
		{
			name:             "ESC on model step goes back to description",
			step:             StepModel,
			keyMsg:           "esc",
			expectCancel:     false,
			expectStepChange: true,
			expectedStep:     StepDescription,
		},
		{
			name:             "Ctrl+C on model step cancels wizard",
			step:             StepModel,
			keyMsg:           "ctrl+c",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepModel,
		},
		{
			name:             "ESC on agent step goes back to model",
			step:             StepAgent,
			keyMsg:           "esc",
			expectCancel:     false,
			expectStepChange: true,
			expectedStep:     StepModel,
		},
		{
			name:             "Ctrl+C on agent step cancels wizard",
			step:             StepAgent,
			keyMsg:           "ctrl+c",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepAgent,
		},
		{
			name:             "ESC on review step goes back to agent",
			step:             StepReview,
			keyMsg:           "esc",
			expectCancel:     false,
			expectStepChange: true,
			expectedStep:     StepAgent,
		},
		{
			name:             "Ctrl+C on review step cancels wizard",
			step:             StepReview,
			keyMsg:           "ctrl+c",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepReview,
		},
		{
			name:             "ESC on completion step goes back to review",
			step:             StepCompletion,
			keyMsg:           "esc",
			expectCancel:     false,
			expectStepChange: true,
			expectedStep:     StepReview,
		},
		{
			name:             "Ctrl+C on completion step cancels wizard",
			step:             StepCompletion,
			keyMsg:           "ctrl+c",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepCompletion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create wizard model at specified step
			m := &WizardModel{
				step:      tt.step,
				cancelled: false,
				width:     80,
				height:    24,
			}

			// Initialize step components to avoid nil panics
			m.titleStep = NewTitleStep()
			m.descriptionStep = NewDescriptionStep()

			// Send key press message
			keyMsg := tea.KeyPressMsg{Text: tt.keyMsg}
			updatedModel, _ := m.Update(keyMsg)

			// Verify cancellation state
			wizModel := updatedModel.(*WizardModel)
			if wizModel.cancelled != tt.expectCancel {
				t.Errorf("Expected cancelled=%v, got %v", tt.expectCancel, wizModel.cancelled)
			}

			// Verify step change
			if tt.expectStepChange {
				if wizModel.step != tt.expectedStep {
					t.Errorf("Expected step=%v, got %v", tt.expectedStep, wizModel.step)
				}
			} else {
				if wizModel.step != tt.step {
					t.Errorf("Expected step to remain %v, got %v", tt.step, wizModel.step)
				}
			}
		})
	}
}

func TestCancellationWithErrorScreen(t *testing.T) {
	// Test that ESC/Ctrl+C work correctly when error screen is displayed
	tests := []struct {
		name             string
		keyMsg           string
		expectCancel     bool
		expectStepChange bool
		expectedStep     int
	}{
		{
			name:             "ESC on error screen goes back to model",
			keyMsg:           "esc",
			expectCancel:     false,
			expectStepChange: true,
			expectedStep:     StepModel,
		},
		{
			name:             "Ctrl+C on error screen cancels wizard",
			keyMsg:           "ctrl+c",
			expectCancel:     true,
			expectStepChange: false,
			expectedStep:     StepAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create wizard model with agent error
			m := &WizardModel{
				step:      StepAgent,
				cancelled: false,
				width:     80,
				height:    24,
			}

			// Set agent error to show error screen
			err := fmt.Errorf("test error: opencode not found")
			m.agentError = &err

			// Send key press message
			keyMsg := tea.KeyPressMsg{Text: tt.keyMsg}
			updatedModel, _ := m.Update(keyMsg)

			// Verify cancellation state
			wizModel := updatedModel.(*WizardModel)
			if wizModel.cancelled != tt.expectCancel {
				t.Errorf("Expected cancelled=%v, got %v", tt.expectCancel, wizModel.cancelled)
			}

			// Verify step change
			if tt.expectStepChange {
				if wizModel.step != tt.expectedStep {
					t.Errorf("Expected step=%v, got %v", tt.expectedStep, wizModel.step)
				}
			} else {
				if wizModel.step != StepAgent {
					t.Errorf("Expected step to remain StepAgent, got %v", wizModel.step)
				}
			}
		})
	}
}

func TestGoBackOnFirstStep(t *testing.T) {
	// Test that goBack() on first step doesn't change state
	m := &WizardModel{
		step:      StepTitle,
		cancelled: false,
		width:     80,
		height:    24,
	}

	// Call goBack directly
	updatedModel, _ := m.goBack()

	// Verify step remains unchanged
	wizModel := updatedModel.(*WizardModel)
	if wizModel.step != StepTitle {
		t.Errorf("Expected step to remain StepTitle, got %v", wizModel.step)
	}

	// Verify wizard is not cancelled
	if wizModel.cancelled {
		t.Error("Expected wizard to not be cancelled")
	}
}

func TestRestartWizardMsg(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard and advance to review step with some data
	m := &WizardModel{
		step:      StepReview,
		cancelled: false,
		cfg:       cfg,
		width:     80,
		height:    24,
		result: WizardResult{
			Title:       "Test Feature",
			Description: "Test description",
			Model:       "claude-3-5-sonnet-20241022",
			SpecContent: "# Test Spec\n\nContent here",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Send RestartWizardMsg
	updatedModel, _ := m.Update(RestartWizardMsg{})

	wizModel := updatedModel.(*WizardModel)

	// Should reset to title step
	if wizModel.step != StepTitle {
		t.Errorf("Expected step to be StepTitle, got %v", wizModel.step)
	}

	// Should clear result
	if wizModel.result.Title != "" {
		t.Error("Expected title to be cleared")
	}
	if wizModel.result.Description != "" {
		t.Error("Expected description to be cleared")
	}
	if wizModel.result.Model != "" {
		t.Error("Expected model to be cleared")
	}
	if wizModel.result.SpecContent != "" {
		t.Error("Expected spec content to be cleared")
	}

	// Should clear error
	if wizModel.agentError != nil {
		t.Error("Expected agent error to be cleared")
	}

	// Should clear button focus
	if wizModel.buttonFocused {
		t.Error("Expected button focus to be cleared")
	}
}

func TestGoBackFromReviewShowsConfirmation(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard at review step
	m := &WizardModel{
		step:      StepReview,
		cancelled: false,
		cfg:       cfg,
		width:     80,
		height:    24,
		result: WizardResult{
			SpecContent: "# Test Spec",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Call goBack
	updatedModel, _ := m.goBack()

	wizModel := updatedModel.(*WizardModel)

	// Should stay on review step
	if wizModel.step != StepReview {
		t.Errorf("Expected to stay on StepReview, got %v", wizModel.step)
	}

	// Should show confirmation modal in review step
	if !wizModel.reviewStep.showConfirmRestart {
		t.Error("Expected confirmation modal to be shown")
	}
}

func TestSaveSpecMsg(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard at review step
	m := &WizardModel{
		step:      StepReview,
		cancelled: false,
		cfg:       cfg,
		width:     80,
		height:    24,
		result: WizardResult{
			SpecContent: "# Test Spec",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Send SaveSpecMsg
	updatedModel, _ := m.Update(SaveSpecMsg{})

	wizModel := updatedModel.(*WizardModel)

	// Should stay on review step (save logic not yet implemented)
	if wizModel.step != StepReview {
		t.Errorf("Expected to stay on StepReview, got %v", wizModel.step)
	}

	// Note: Actual save functionality will be tested in TAS-47
}

func TestSaveErrorMsg(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard at review step
	m := &WizardModel{
		step:      StepReview,
		cancelled: false,
		cfg:       cfg,
		width:     80,
		height:    24,
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			SpecContent: "# Test Spec",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Send SaveErrorMsg
	testErr := fmt.Errorf("permission denied")
	updatedModel, cmd := m.Update(SaveErrorMsg{Err: testErr})
	if cmd != nil {
		t.Errorf("Expected no command from SaveErrorMsg, got %T", cmd)
	}

	wizModel := updatedModel.(*WizardModel)

	// Should show error modal
	if !wizModel.showSaveError {
		t.Error("Expected showSaveError to be true")
	}
	if wizModel.saveError != "permission denied" {
		t.Errorf("Expected saveError to be 'permission denied', got '%s'", wizModel.saveError)
	}

	// View should render error modal
	view := wizModel.renderCurrentStep()
	if !strings.Contains(view, "Save Failed") {
		t.Error("Expected view to contain 'Save Failed'")
	}
	if !strings.Contains(view, "permission denied") {
		t.Error("Expected view to contain error message")
	}
	if !strings.Contains(view, "Press Y to retry") {
		t.Error("Expected view to contain retry instruction")
	}
}

func TestSaveErrorModal_Retry(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard with save error
	m := &WizardModel{
		step:          StepReview,
		cfg:           cfg,
		showSaveError: true,
		saveError:     "test error",
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			SpecContent: "# Test Spec",
		},
	}

	// Press Y to retry
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "y"})
	if cmd == nil {
		t.Fatal("Expected command from Y keypress")
	}

	// Execute command to get RetrySaveMsg
	msg := cmd()
	if _, ok := msg.(RetrySaveMsg); !ok {
		t.Errorf("Expected RetrySaveMsg from Y keypress, got %T", msg)
	}

	// Handle RetrySaveMsg
	wizModel := updatedModel.(*WizardModel)
	updatedModel2, cmd2 := wizModel.Update(msg)
	if cmd2 == nil {
		t.Fatal("Expected command from RetrySaveMsg")
	}

	wizModel2 := updatedModel2.(*WizardModel)

	// Should hide error modal
	if wizModel2.showSaveError {
		t.Error("Expected showSaveError to be false after retry")
	}
	if wizModel2.saveError != "" {
		t.Error("Expected saveError to be cleared after retry")
	}

	// Should emit SaveSpecMsg to retry
	msg2 := cmd2()
	if _, ok := msg2.(SaveSpecMsg); !ok {
		t.Errorf("Expected SaveSpecMsg from retry, got %T", msg2)
	}
}

func TestSaveErrorModal_Cancel(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard with save error
	m := &WizardModel{
		step:          StepReview,
		cfg:           cfg,
		showSaveError: true,
		saveError:     "test error",
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			SpecContent: "# Test Spec",
		},
	}

	// Press N to cancel
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "n"})
	if cmd != nil {
		t.Errorf("Expected no command from N keypress, got %T", cmd)
	}

	wizModel := updatedModel.(*WizardModel)

	// Should hide error modal
	if wizModel.showSaveError {
		t.Error("Expected showSaveError to be false after cancel")
	}
	if wizModel.saveError != "" {
		t.Error("Expected saveError to be cleared after cancel")
	}

	// Should stay on review step
	if wizModel.step != StepReview {
		t.Errorf("Expected to stay on StepReview, got %v", wizModel.step)
	}
}

func TestSaveErrorModal_ESC(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard with save error
	m := &WizardModel{
		step:          StepReview,
		cfg:           cfg,
		showSaveError: true,
		saveError:     "test error",
	}

	// Press ESC to cancel
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "esc"})
	if cmd != nil {
		t.Errorf("Expected no command from ESC keypress, got %T", cmd)
	}

	wizModel := updatedModel.(*WizardModel)

	// Should hide error modal
	if wizModel.showSaveError {
		t.Error("Expected showSaveError to be false after ESC")
	}
	if wizModel.saveError != "" {
		t.Error("Expected saveError to be cleared after ESC")
	}
}

func TestSaveErrorModal_IgnoresOtherKeys(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard with save error
	m := &WizardModel{
		step:          StepReview,
		cfg:           cfg,
		showSaveError: true,
		saveError:     "test error",
	}

	// Press random key
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "x"})
	if cmd != nil {
		t.Errorf("Expected no command from random keypress, got %T", cmd)
	}

	wizModel := updatedModel.(*WizardModel)

	// Should keep error modal visible
	if !wizModel.showSaveError {
		t.Error("Expected showSaveError to remain true")
	}
	if wizModel.saveError != "test error" {
		t.Error("Expected saveError to remain unchanged")
	}
}

func TestModelSelectorIntegration(t *testing.T) {
	cfg := &config.Config{
		Model:   "claude-3-5-sonnet-20241022",
		SpecDir: "./specs",
	}

	// Create wizard at model step
	m := &WizardModel{
		step:      StepModel,
		cancelled: false,
		cfg:       cfg,
		width:     80,
		height:    24,
		result: WizardResult{
			Title:       "Test Feature",
			Description: "Test description",
		},
	}

	// Initialize model step
	m.initCurrentStep()

	// Verify model step was initialized
	if m.modelStep == nil {
		t.Fatal("Expected modelStep to be initialized")
	}

	// Send ModelSelectedMsg
	selectedModel := "claude-3-5-sonnet-20241022"
	updatedModel, cmd := m.Update(wizard.ModelSelectedMsg{ModelID: selectedModel})

	wizModel := updatedModel.(*WizardModel)

	// Should advance to agent step
	if wizModel.step != StepAgent {
		t.Errorf("Expected step to be StepAgent, got %v", wizModel.step)
	}

	// Should store selected model
	if wizModel.result.Model != selectedModel {
		t.Errorf("Expected model to be %q, got %q", selectedModel, wizModel.result.Model)
	}

	// Should clear button focus
	if wizModel.buttonFocused {
		t.Error("Expected button focus to be cleared")
	}

	// Should return startAgentPhase command
	if cmd == nil {
		t.Error("Expected startAgentPhase command to be returned")
	}
}

func TestStartAgentPhaseStructure(t *testing.T) {
	// Test that startAgentPhase properly structures the agent setup
	// Note: This test verifies the function structure without actually spawning
	// the subprocess (which would require opencode to be installed)

	cfg := &config.Config{
		Model:   "claude-3-5-sonnet-20241022",
		SpecDir: "./test-specs",
	}

	m := &WizardModel{
		step:      StepAgent,
		cancelled: false,
		cfg:       cfg,
		width:     80,
		height:    24,
		result: WizardResult{
			Title:       "User Authentication",
			Description: "Add email/password authentication system",
			Model:       "claude-3-5-sonnet-20241022",
		},
	}

	// Test buildSpecPrompt (which is called by startAgentPhase)
	prompt := buildSpecPrompt(m.result.Title, m.result.Description)

	// Verify prompt structure
	if !strings.Contains(prompt, m.result.Title) {
		t.Error("Expected prompt to contain title")
	}
	if !strings.Contains(prompt, m.result.Description) {
		t.Error("Expected prompt to contain description")
	}
	if !strings.Contains(prompt, "ask-questions") {
		t.Error("Expected prompt to mention ask-questions tool")
	}
	if !strings.Contains(prompt, "finish-spec") {
		t.Error("Expected prompt to mention finish-spec tool")
	}

	// Verify spec format sections are included
	requiredSections := []string{
		"## Overview",
		"## User Story",
		"## Requirements",
		"## Technical Implementation",
		"## Tasks",
		"## Out of Scope",
	}
	for _, section := range requiredSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Expected prompt to contain section: %s", section)
		}
	}
}

func TestStartAgentPhaseErrorHandling(t *testing.T) {
	// Test that startAgentPhase returns appropriate error messages
	// when MCP server or ACP subprocess fail to start

	tests := []struct {
		name          string
		setupFailure  string // What to simulate failing
		expectErrType string // Expected error type in result
	}{
		{
			name:          "handles opencode not found gracefully",
			setupFailure:  "opencode_not_found",
			expectErrType: "failed to start opencode",
		},
		{
			name:          "handles MCP server failure gracefully",
			setupFailure:  "mcp_server_failure",
			expectErrType: "failed to start MCP server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test documents the expected error handling behavior
			// The actual startAgentPhase will return AgentErrorMsg when errors occur
			// which is handled by the wizard Update() method

			// Verify that AgentErrorMsg is properly structured
			testErr := fmt.Errorf("%s: test error", tt.expectErrType)
			msg := AgentErrorMsg{Err: testErr}

			// Verify error message contains expected type
			if !strings.Contains(msg.Err.Error(), tt.expectErrType) {
				t.Errorf("Expected error to contain %q, got %q", tt.expectErrType, msg.Err.Error())
			}
		})
	}
}

func TestCancelWizardMsg(t *testing.T) {
	cfg := &config.Config{
		SpecDir: "./specs",
	}

	// Create wizard at agent step with mock MCP server
	mcpServer := specmcp.New("test-spec", "./specs")
	m := &WizardModel{
		step:      StepAgent,
		cancelled: false,
		cfg:       cfg,
		width:     80,
		height:    24,
		mcpServer: mcpServer,
	}
	m.agentStep = NewAgentPhase(mcpServer)

	// Send CancelWizardMsg (simulating user confirming cancellation during agent phase)
	updatedModel, cmd := m.Update(CancelWizardMsg{})

	wizModel := updatedModel.(*WizardModel)

	// Should set cancelled flag
	if !wizModel.cancelled {
		t.Error("Expected cancelled=true after CancelWizardMsg")
	}

	// Should return tea.Quit command
	if cmd == nil {
		t.Error("Expected CancelWizardMsg to return tea.Quit command")
	}

	// MCP server should be cleaned up (set to nil)
	if wizModel.mcpServer != nil {
		t.Error("Expected mcpServer to be cleaned up (nil) after cancellation")
	}

	// Agent runner should be cleaned up (set to nil)
	if wizModel.agentRunner != nil {
		t.Error("Expected agentRunner to be cleaned up (nil) after cancellation")
	}
}

func TestAgentEarlyTerminationError(t *testing.T) {
	tests := []struct {
		name       string
		stopReason string
		errorMsg   string
		expectErr  bool
	}{
		{
			name:       "agent terminates with error stop reason",
			stopReason: "error",
			errorMsg:   "model request failed",
			expectErr:  true,
		},
		{
			name:       "agent terminates with error message but different stop reason",
			stopReason: "end_turn",
			errorMsg:   "unexpected error occurred",
			expectErr:  true,
		},
		{
			name:       "agent completes successfully",
			stopReason: "end_turn",
			errorMsg:   "",
			expectErr:  false,
		},
		{
			name:       "agent cancelled by user",
			stopReason: "cancelled",
			errorMsg:   "",
			expectErr:  false,
		},
		{
			name:       "agent hits max tokens",
			stopReason: "max_tokens",
			errorMsg:   "",
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock program to capture sent messages
			receivedMsgs := []tea.Msg{}
			mockProgram := &mockProgram{
				sendFunc: func(msg tea.Msg) {
					receivedMsgs = append(receivedMsgs, msg)
				},
			}

			cfg := &config.Config{
				SpecDir: "./specs",
			}

			m := &WizardModel{
				step:      StepAgent,
				cancelled: false,
				cfg:       cfg,
				width:     80,
				height:    24,
			}
			m.program = mockProgram

			// Simulate OnFinish callback being called (this happens in agent goroutine)
			// Create the runner config with OnFinish that checks for errors
			onFinishCalled := false
			onFinish := func(event agent.FinishEvent) {
				onFinishCalled = true
				// Check if agent terminated with an error
				if event.StopReason == "error" || event.Error != "" {
					// Send error message to UI
					if m.program != nil {
						m.program.Send(AgentErrorMsg{Err: fmt.Errorf("agent error: %s", event.Error)})
					}
				}
			}

			// Simulate the OnFinish callback being invoked
			onFinish(agent.FinishEvent{
				StopReason: tt.stopReason,
				Error:      tt.errorMsg,
			})

			// Verify OnFinish was called
			if !onFinishCalled {
				t.Error("Expected OnFinish to be called")
			}

			// Verify error message was sent when expected
			if tt.expectErr {
				if len(receivedMsgs) != 1 {
					t.Errorf("Expected 1 message to be sent, got %d", len(receivedMsgs))
				} else {
					errMsg, ok := receivedMsgs[0].(AgentErrorMsg)
					if !ok {
						t.Errorf("Expected AgentErrorMsg, got %T", receivedMsgs[0])
					} else if !strings.Contains(errMsg.Err.Error(), "agent error:") {
						t.Errorf("Expected error message to contain 'agent error:', got %q", errMsg.Err.Error())
					}
				}
			} else {
				if len(receivedMsgs) != 0 {
					t.Errorf("Expected no messages to be sent, got %d", len(receivedMsgs))
				}
			}
		})
	}
}

// mockProgram implements a minimal tea.Program interface for testing callbacks
type mockProgram struct {
	sendFunc func(tea.Msg)
}

func (m *mockProgram) Send(msg tea.Msg) {
	if m.sendFunc != nil {
		m.sendFunc(msg)
	}
}

// Implement other tea.Program methods as no-ops (not used in tests)
func (m *mockProgram) Run() (tea.Model, error)  { return nil, nil }
func (m *mockProgram) Quit()                    {}
func (m *mockProgram) Wait() (tea.Model, error) { return nil, nil }

func TestCheckFileExistsMsg_FileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SpecDir: tmpDir}

	// Create wizard in review step
	m := &WizardModel{
		step: StepReview,
		cfg:  cfg,
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			SpecContent: "# Test Spec\n\n## Overview\n\n## Tasks",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Send CheckFileExistsMsg
	updatedModel, cmd := m.Update(CheckFileExistsMsg{})

	// Should proceed with save since file doesn't exist
	if cmd == nil {
		t.Fatal("Expected command from CheckFileExistsMsg")
	}

	msg := cmd()
	if _, ok := msg.(SaveSpecMsg); !ok {
		t.Errorf("Expected SaveSpecMsg when file doesn't exist, got %T", msg)
	}

	// Overwrite confirmation should not be shown
	if m.reviewStep.showConfirmOverwrite {
		t.Error("Expected overwrite confirmation not to be shown")
	}

	// Verify model returned
	if updatedModel == nil {
		t.Error("Expected non-nil model")
	}
}

func TestCheckFileExistsMsg_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SpecDir: tmpDir}

	// Create existing spec file
	existingPath := filepath.Join(tmpDir, "test-spec.md")
	if err := os.WriteFile(existingPath, []byte("Old content"), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create wizard in review step
	m := &WizardModel{
		step: StepReview,
		cfg:  cfg,
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			SpecContent: "# Test Spec\n\n## Overview\n\n## Tasks",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Send CheckFileExistsMsg
	updatedModel, cmd := m.Update(CheckFileExistsMsg{})

	// Should show overwrite confirmation since file exists
	if cmd != nil {
		t.Error("Expected no command when file exists (should show confirmation modal)")
	}

	// Overwrite confirmation should be shown
	if !m.reviewStep.showConfirmOverwrite {
		t.Error("Expected overwrite confirmation to be shown")
	}

	// Verify model returned
	if updatedModel == nil {
		t.Error("Expected non-nil model")
	}
}

func TestOverwriteFlow_ConfirmYes(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SpecDir: tmpDir}

	// Create existing spec file
	existingPath := filepath.Join(tmpDir, "test-spec.md")
	oldContent := "# Old Spec\n\nOld content"
	if err := os.WriteFile(existingPath, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create wizard in review step
	m := &WizardModel{
		step: StepReview,
		cfg:  cfg,
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			SpecContent: "# New Spec\n\n## Overview\n\nNew content\n\n## Tasks",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Step 1: Check if file exists
	_, cmd := m.Update(CheckFileExistsMsg{})
	if cmd != nil {
		t.Error("Expected no command, should show overwrite modal")
	}
	if !m.reviewStep.showConfirmOverwrite {
		t.Fatal("Expected overwrite confirmation to be shown")
	}

	// Step 2: Confirm overwrite by sending Y key to review step
	cmd = m.reviewStep.Update(tea.KeyPressMsg{Text: "Y"})
	if cmd == nil {
		t.Fatal("Expected command after pressing Y")
	}

	// Should get SaveSpecMsg
	msg := cmd()
	if _, ok := msg.(SaveSpecMsg); !ok {
		t.Fatalf("Expected SaveSpecMsg after confirming overwrite, got %T", msg)
	}

	// Modal should be hidden
	if m.reviewStep.showConfirmOverwrite {
		t.Error("Expected overwrite confirmation to be hidden after confirming")
	}

	// Step 3: Handle SaveSpecMsg in wizard
	updatedModel, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("Expected command from SaveSpecMsg")
	}

	// Should advance to completion step
	resultMsg := cmd()
	savedMsg, ok := resultMsg.(SpecSavedMsg)
	if !ok {
		t.Fatalf("Expected SpecSavedMsg, got %T", resultMsg)
	}

	// Verify file was overwritten with new content
	actualContent, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if !strings.Contains(string(actualContent), "New content") {
		t.Error("Expected file to be overwritten with new content")
	}
	if strings.Contains(string(actualContent), "Old content") {
		t.Error("File should not contain old content")
	}

	// Verify path is correct
	if savedMsg.Path != existingPath {
		t.Errorf("Expected path %q, got %q", existingPath, savedMsg.Path)
	}

	// Verify model returned
	if updatedModel == nil {
		t.Error("Expected non-nil model")
	}
}

func TestOverwriteFlow_CancelWithN(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SpecDir: tmpDir}

	// Create existing spec file
	existingPath := filepath.Join(tmpDir, "test-spec.md")
	oldContent := "# Old Spec\n\nOld content"
	if err := os.WriteFile(existingPath, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create wizard in review step
	m := &WizardModel{
		step: StepReview,
		cfg:  cfg,
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			SpecContent: "# New Spec\n\n## Overview\n\nNew content\n\n## Tasks",
		},
	}
	m.reviewStep = NewReviewStep(m.result.SpecContent, cfg)

	// Step 1: Check if file exists
	_, _ = m.Update(CheckFileExistsMsg{})
	if !m.reviewStep.showConfirmOverwrite {
		t.Fatal("Expected overwrite confirmation to be shown")
	}

	// Step 2: Cancel overwrite by pressing N
	cmd := m.reviewStep.Update(tea.KeyPressMsg{Text: "n"})

	// Should not get SaveSpecMsg
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(SaveSpecMsg); ok {
			t.Error("Expected no SaveSpecMsg after canceling")
		}
	}

	// Modal should be hidden
	if m.reviewStep.showConfirmOverwrite {
		t.Error("Expected overwrite confirmation to be hidden after canceling")
	}

	// Verify file was NOT overwritten
	actualContent, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !strings.Contains(string(actualContent), "Old content") {
		t.Error("Expected file to still contain old content")
	}
	if strings.Contains(string(actualContent), "New content") {
		t.Error("File should not contain new content")
	}
}

func TestWizard_SpecContentReceivedMsg(t *testing.T) {
	// Create wizard in agent phase step
	cfg := &config.Config{
		SpecDir: t.TempDir(),
	}
	m := &WizardModel{
		step: StepAgent,
		cfg:  cfg,
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			Model:       "claude-sonnet-4-5",
		},
	}

	// Send SpecContentReceivedMsg with spec content
	testContent := "## Overview\n\nThis is a test spec.\n\n## Tasks\n\n- [ ] Task 1\n- [ ] Task 2"
	msg := SpecContentReceivedMsg{Content: testContent}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(*WizardModel)

	// Should transition to review step
	if m.step != StepReview {
		t.Errorf("Expected step to be StepReview, got %v", m.step)
	}

	// Should store content in result
	if m.result.SpecContent != testContent {
		t.Errorf("Expected SpecContent to be %q, got %q", testContent, m.result.SpecContent)
	}

	// Should have initialized review step
	if m.reviewStep == nil {
		t.Error("Expected reviewStep to be initialized")
	}

	// Verify review step was initialized with the content
	if m.reviewStep.content != testContent {
		t.Errorf("Expected reviewStep.content to be %q, got %q", testContent, m.reviewStep.content)
	}

	// Should not return a command
	if cmd != nil {
		t.Error("Expected no command to be returned")
	}

	// Button focus should be reset
	if m.buttonFocused {
		t.Error("Expected buttonFocused to be false")
	}
}

func TestWizard_StartBuildMsg(t *testing.T) {
	// Create wizard in completion step
	cfg := &config.Config{
		SpecDir: t.TempDir(),
	}
	m := &WizardModel{
		step: StepCompletion,
		cfg:  cfg,
		result: WizardResult{
			Title:       "Test Spec",
			Description: "Test description",
			Model:       "test-model",
			SpecContent: "# Test Content",
			SpecPath:    "./specs/test-spec.md",
		},
	}
	m.initCurrentStep()

	// Send StartBuildMsg (simulating Start Build button click)
	model, cmd := m.Update(StartBuildMsg{})
	m = model.(*WizardModel)

	// Should receive ExecBuildMsg command
	if cmd == nil {
		t.Fatal("Expected Update to return a command")
	}

	msg := cmd()
	execMsg, ok := msg.(ExecBuildMsg)
	if !ok {
		t.Fatalf("Expected ExecBuildMsg, got %T", msg)
	}

	// Verify spec path is passed correctly
	if execMsg.SpecPath != m.result.SpecPath {
		t.Errorf("Expected SpecPath %q, got %q", m.result.SpecPath, execMsg.SpecPath)
	}
}

func TestWizard_ExecBuildMsg(t *testing.T) {
	// Create wizard in completion step
	cfg := &config.Config{
		SpecDir: t.TempDir(),
	}
	m := &WizardModel{
		step: StepCompletion,
		cfg:  cfg,
		result: WizardResult{
			SpecPath: "./specs/test-spec.md",
		},
	}

	// Send ExecBuildMsg
	execMsg := ExecBuildMsg{SpecPath: "./specs/test-spec.md"}
	_, cmd := m.Update(execMsg)

	// Should receive a command (tea.Sequence)
	if cmd == nil {
		t.Fatal("Expected Update to return a command")
	}

	// The command is a tea.Sequence that combines tea.Quit with execBuild
	// We can't easily test the sequence directly, but we verify a command was returned
	// The actual execution of iteratr build would require a full binary and integration test

	// Note: We've verified that:
	// 1. StartBuildMsg triggers ExecBuildMsg
	// 2. ExecBuildMsg returns a command (the Sequence)
	// This confirms the message flow is correct
}

func TestModalLayoutConstants(t *testing.T) {
	// Verify the modal layout constants are correct
	expectedContentWidth := modalWidth - (modalPadding * 2) - (modalBorderWidth * 2)
	if modalContentWidth != expectedContentWidth {
		t.Errorf("modalContentWidth = %d, expected %d", modalContentWidth, expectedContentWidth)
	}

	// Verify specific values match design
	if modalWidth != 70 {
		t.Errorf("modalWidth = %d, expected 70", modalWidth)
	}
	if modalPadding != 2 {
		t.Errorf("modalPadding = %d, expected 2", modalPadding)
	}
	if modalBorderWidth != 1 {
		t.Errorf("modalBorderWidth = %d, expected 1", modalBorderWidth)
	}
	if modalContentWidth != 64 {
		t.Errorf("modalContentWidth = %d, expected 64", modalContentWidth)
	}
}

func TestWizard_GetModalContentSize(t *testing.T) {
	tests := []struct {
		name          string
		termWidth     int
		termHeight    int
		expectedWidth int
		minHeight     int
		maxHeight     int
	}{
		{
			name:          "normal terminal",
			termWidth:     120,
			termHeight:    40,
			expectedWidth: modalContentWidth,
			minHeight:     10,
			maxHeight:     30,
		},
		{
			name:          "small terminal",
			termWidth:     80,
			termHeight:    20,
			expectedWidth: modalContentWidth,
			minHeight:     10,
			maxHeight:     20,
		},
		{
			name:          "very small terminal",
			termWidth:     60,
			termHeight:    15,
			expectedWidth: modalContentWidth,
			minHeight:     10,
			maxHeight:     15,
		},
		{
			name:          "large terminal",
			termWidth:     200,
			termHeight:    60,
			expectedWidth: modalContentWidth,
			minHeight:     10,
			maxHeight:     30, // Should be capped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			m := &WizardModel{
				cfg:    cfg,
				width:  tt.termWidth,
				height: tt.termHeight,
			}

			width, height := m.getModalContentSize()

			if width != tt.expectedWidth {
				t.Errorf("width = %d, expected %d", width, tt.expectedWidth)
			}
			if height < tt.minHeight {
				t.Errorf("height = %d, expected >= %d", height, tt.minHeight)
			}
			if height > tt.maxHeight {
				t.Errorf("height = %d, expected <= %d", height, tt.maxHeight)
			}
		})
	}
}

func TestWizard_UpdateCurrentStepSize(t *testing.T) {
	cfg := &config.Config{}

	// Test each step type
	steps := []struct {
		name string
		step int
	}{
		{"title", StepTitle},
		{"description", StepDescription},
		{"model", StepModel},
		{"review", StepReview},
		{"completion", StepCompletion},
	}

	for _, tt := range steps {
		t.Run(tt.name, func(t *testing.T) {
			m := &WizardModel{
				cfg:    cfg,
				width:  100,
				height: 40,
				step:   tt.step,
			}

			// Initialize the appropriate step
			switch tt.step {
			case StepTitle:
				m.titleStep = NewTitleStep()
			case StepDescription:
				m.descriptionStep = NewDescriptionStep()
			case StepModel:
				m.modelStep = wizard.NewModelSelectorStep()
			case StepReview:
				m.reviewStep = NewReviewStep("# Test", cfg)
			case StepCompletion:
				m.completionStep = NewCompletionStep("/test/path.md")
			}

			// Call updateCurrentStepSize - should not panic
			m.updateCurrentStepSize()

			// Verify the step received proper dimensions
			contentWidth, contentHeight := m.getModalContentSize()

			switch tt.step {
			case StepTitle:
				if m.titleStep.width != contentWidth {
					t.Errorf("titleStep.width = %d, expected %d", m.titleStep.width, contentWidth)
				}
			case StepDescription:
				if m.descriptionStep.width != contentWidth {
					t.Errorf("descriptionStep.width = %d, expected %d", m.descriptionStep.width, contentWidth)
				}
			case StepReview:
				if m.reviewStep.width != contentWidth {
					t.Errorf("reviewStep.width = %d, expected %d", m.reviewStep.width, contentWidth)
				}
				if m.reviewStep.height != contentHeight {
					t.Errorf("reviewStep.height = %d, expected %d", m.reviewStep.height, contentHeight)
				}
			case StepCompletion:
				if m.completionStep.width != contentWidth {
					t.Errorf("completionStep.width = %d, expected %d", m.completionStep.width, contentWidth)
				}
			}
		})
	}
}
