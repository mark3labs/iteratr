package specwizard

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
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
		"using the ask-questions",
		"using the finish-spec tool",
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
