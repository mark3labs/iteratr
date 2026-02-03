package specwizard

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
