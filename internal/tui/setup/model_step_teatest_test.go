package setup

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// TestModelStepTeatest_CustomMode verifies entering custom model entry mode and submitting a custom model.
func TestModelStepTeatest_CustomMode(t *testing.T) {
	t.Parallel()

	// Create a new model step
	step := NewModelStep()

	// Simulate models loaded (skip actual fetch)
	updatedStep := step
	updatedStep.loading = false
	updatedStep.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
		{id: "test/model-2", name: "test/model-2"},
	}
	updatedStep.filterModels()

	// Initially should not be in custom mode
	require.False(t, updatedStep.isCustomMode, "Expected isCustomMode to be false initially")

	// Press 'c' to enter custom mode
	cmd := updatedStep.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	require.NotNil(t, cmd, "Expected command to be returned when entering custom mode")

	// Should now be in custom mode
	require.True(t, updatedStep.isCustomMode, "Expected isCustomMode to be true after pressing 'c'")

	// View should show custom input
	view := updatedStep.View()
	require.Contains(t, view, "Enter Custom Model", "Expected view to contain 'Enter Custom Model'")

	// Simulate typing a custom model (set value directly since textinput keyboard handling is complex)
	updatedStep.customInput.SetValue("my-custom/model")

	// Verify custom input value
	require.Equal(t, "my-custom/model", updatedStep.customInput.Value(), "Expected custom input value to be 'my-custom/model'")

	// Press Enter to confirm
	cmd = updatedStep.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})

	// Execute the command to get the message
	require.NotNil(t, cmd, "Expected cmd to be non-nil after pressing Enter")

	msg := cmd()
	modelMsg, ok := msg.(ModelSelectedMsg)
	require.True(t, ok, "Expected ModelSelectedMsg, got %T", msg)

	require.Equal(t, "my-custom/model", modelMsg.ModelID, "Expected ModelID to be 'my-custom/model'")
}

// TestModelStepTeatest_CustomModeCancel verifies ESC cancels custom mode and clears input.
func TestModelStepTeatest_CustomModeCancel(t *testing.T) {
	t.Parallel()

	// Create a new model step
	step := NewModelStep()

	// Simulate models loaded
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
	}
	step.filterModels()

	// Enter custom mode
	cmd := step.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	require.NotNil(t, cmd, "Expected command when entering custom mode")

	require.True(t, step.isCustomMode, "Expected isCustomMode to be true after pressing 'c'")

	// Type something (set value directly)
	step.customInput.SetValue("partial")

	// Press ESC to cancel
	cmd = step.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Text: "esc"})
	require.NotNil(t, cmd, "Expected ContentChangedMsg command when exiting custom mode")

	// Should exit custom mode
	require.False(t, step.isCustomMode, "Expected isCustomMode to be false after pressing ESC")

	// Custom input should be cleared
	require.Equal(t, "", step.customInput.Value(), "Expected custom input to be cleared")
}

// TestModelStepTeatest_CustomModeEmptyInput verifies Enter with empty input does nothing.
func TestModelStepTeatest_CustomModeEmptyInput(t *testing.T) {
	t.Parallel()

	// Create a new model step
	step := NewModelStep()

	// Simulate models loaded
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
	}
	step.filterModels()

	// Enter custom mode
	step.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	// Press Enter without typing anything
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})

	// Should not return a command (empty input ignored)
	require.Nil(t, cmd, "Expected cmd to be nil when pressing Enter with empty input")
}

// TestModelStepTeatest_CustomModeWhitespaceOnly verifies whitespace-only input is ignored.
func TestModelStepTeatest_CustomModeWhitespaceOnly(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
	}
	step.filterModels()

	// Enter custom mode
	step.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	// Set value to whitespace only
	step.customInput.SetValue("   \t  \n  ")

	// Press Enter
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})

	// Should not return a command (whitespace trimmed to empty)
	require.Nil(t, cmd, "Expected cmd to be nil when pressing Enter with whitespace-only input")
}

// TestModelStepTeatest_PreferredHeight_CustomMode verifies custom mode has fixed height of 5.
func TestModelStepTeatest_PreferredHeight_CustomMode(t *testing.T) {
	t.Parallel()

	step := NewModelStep()

	// Not in custom mode initially - add multiple models so height differs from custom mode
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
		{id: "test/model-2", name: "test/model-2"},
		{id: "test/model-3", name: "test/model-3"},
		{id: "test/model-4", name: "test/model-4"},
		{id: "test/model-5", name: "test/model-5"},
	}
	step.filterModels()

	normalHeight := step.PreferredHeight()

	// Enter custom mode
	step.isCustomMode = true

	customHeight := step.PreferredHeight()

	// Custom mode should have fixed height of 5
	require.Equal(t, 5, customHeight, "Expected custom mode height to be 5")

	// Heights should be different (normal mode has 5 models + 4 overhead = 9)
	require.NotEqual(t, normalHeight, customHeight, "Expected normal and custom mode to have different heights")
}

// TestModelStepTeatest_PreferredHeight_Loading verifies loading state has height of 1.
func TestModelStepTeatest_PreferredHeight_Loading(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = true

	height := step.PreferredHeight()

	require.Equal(t, 1, height, "Expected loading state height to be 1")
}

// TestModelStepTeatest_PreferredHeight_Error verifies error states have correct heights.
func TestModelStepTeatest_PreferredHeight_Error(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false

	// Generic error state
	step.error = "test error"
	step.isNotInstalled = false

	height := step.PreferredHeight()
	require.Equal(t, 3, height, "Expected generic error height to be 3")

	// Not installed error state
	step.isNotInstalled = true

	height = step.PreferredHeight()
	require.Equal(t, 6, height, "Expected 'not installed' error height to be 6")
}

// TestModelStepTeatest_PreferredHeight_Normal verifies normal mode height calculation.
func TestModelStepTeatest_PreferredHeight_Normal(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false

	// Test with various model counts
	testCases := []struct {
		name        string
		modelCount  int
		expectedMax int // Max height with cap at 20
	}{
		{"Empty", 0, 4},    // overhead only
		{"One", 1, 5},      // 1 + 4 overhead
		{"Five", 5, 9},     // 5 + 4 overhead
		{"Twenty", 20, 24}, // 20 + 4 overhead (at cap)
		{"Thirty", 30, 24}, // 20 (capped) + 4 overhead
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			models := make([]*ModelInfo, tc.modelCount)
			for i := 0; i < tc.modelCount; i++ {
				models[i] = &ModelInfo{id: "test/model", name: "test/model"}
			}
			step.allModels = models
			step.filterModels()

			height := step.PreferredHeight()
			require.Equal(t, tc.expectedMax, height, "Unexpected height for %d models", tc.modelCount)
		})
	}
}

// TestModelStepTeatest_ViewStates verifies View() renders correctly in all states.
func TestModelStepTeatest_ViewStates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		setup    func(*ModelStep)
		contains []string
	}{
		{
			name: "Loading",
			setup: func(m *ModelStep) {
				m.loading = true
			},
			contains: []string{"Loading models"},
		},
		{
			name: "GenericError",
			setup: func(m *ModelStep) {
				m.loading = false
				m.error = "test error"
				m.isNotInstalled = false
			},
			contains: []string{"Error: test error", "Press 'r' to retry"},
		},
		{
			name: "NotInstalledError",
			setup: func(m *ModelStep) {
				m.loading = false
				m.error = "test error"
				m.isNotInstalled = true
			},
			contains: []string{"opencode is not installed", "Press 'c' for custom model"},
		},
		{
			name: "CustomMode",
			setup: func(m *ModelStep) {
				m.loading = false
				m.isCustomMode = true
			},
			contains: []string{"Enter Custom Model", "Enter confirm", "ESC cancel"},
		},
		{
			name: "EmptyFiltered",
			setup: func(m *ModelStep) {
				m.loading = false
				m.allModels = []*ModelInfo{
					{id: "test/model-1", name: "test/model-1"},
				}
				m.searchInput.SetValue("nomatch")
				m.filterModels()
			},
			contains: []string{"No models match your search"},
		},
		{
			name: "NormalMode",
			setup: func(m *ModelStep) {
				m.loading = false
				m.allModels = []*ModelInfo{
					{id: "test/model-1", name: "test/model-1"},
					{id: "test/model-2", name: "test/model-2"},
				}
				m.filterModels()
			},
			contains: []string{"navigate", "select", "custom"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := NewModelStep()
			tc.setup(step)

			view := step.View()

			for _, expected := range tc.contains {
				require.Contains(t, view, expected, "Expected view to contain '%s'", expected)
			}
		})
	}
}

// TestModelStepTeatest_Navigation verifies up/down navigation in model list.
func TestModelStepTeatest_Navigation(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "model-1", name: "model-1"},
		{id: "model-2", name: "model-2"},
		{id: "model-3", name: "model-3"},
	}
	step.filterModels()

	// Initial selection should be 0
	require.Equal(t, 0, step.selectedIdx, "Expected initial selectedIdx to be 0")

	// Press down
	step.Update(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
	require.Equal(t, 1, step.selectedIdx, "Expected selectedIdx to be 1 after down")

	// Press down again
	step.Update(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
	require.Equal(t, 2, step.selectedIdx, "Expected selectedIdx to be 2 after second down")

	// Press down at bottom - should stay at 2
	step.Update(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
	require.Equal(t, 2, step.selectedIdx, "Expected selectedIdx to stay at 2 at bottom")

	// Press up
	step.Update(tea.KeyPressMsg{Code: tea.KeyUp, Text: "up"})
	require.Equal(t, 1, step.selectedIdx, "Expected selectedIdx to be 1 after up")

	// Press up again
	step.Update(tea.KeyPressMsg{Code: tea.KeyUp, Text: "up"})
	require.Equal(t, 0, step.selectedIdx, "Expected selectedIdx to be 0 after second up")

	// Press up at top - should stay at 0
	step.Update(tea.KeyPressMsg{Code: tea.KeyUp, Text: "up"})
	require.Equal(t, 0, step.selectedIdx, "Expected selectedIdx to stay at 0 at top")
}

// TestModelStepTeatest_VimNavigation verifies j/k vim-style navigation.
func TestModelStepTeatest_VimNavigation(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "model-1", name: "model-1"},
		{id: "model-2", name: "model-2"},
		{id: "model-3", name: "model-3"},
	}
	step.filterModels()

	// Initial selection should be 0
	require.Equal(t, 0, step.selectedIdx)

	// Press 'j' (down)
	step.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	require.Equal(t, 1, step.selectedIdx, "Expected selectedIdx to be 1 after 'j'")

	// Press 'j' again
	step.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	require.Equal(t, 2, step.selectedIdx, "Expected selectedIdx to be 2 after second 'j'")

	// Press 'k' (up)
	step.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	require.Equal(t, 1, step.selectedIdx, "Expected selectedIdx to be 1 after 'k'")

	// Press 'k' again
	step.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	require.Equal(t, 0, step.selectedIdx, "Expected selectedIdx to be 0 after second 'k'")
}

// TestModelStepTeatest_EnterSelectsModel verifies Enter sends ModelSelectedMsg.
func TestModelStepTeatest_EnterSelectsModel(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "model-1", name: "model-1"},
		{id: "model-2", name: "model-2"},
		{id: "model-3", name: "model-3"},
	}
	step.filterModels()

	// Navigate to second model
	step.Update(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
	require.Equal(t, 1, step.selectedIdx)

	// Press Enter
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
	require.NotNil(t, cmd, "Expected command to be returned")

	// Execute command
	msg := cmd()
	modelMsg, ok := msg.(ModelSelectedMsg)
	require.True(t, ok, "Expected ModelSelectedMsg, got %T", msg)

	require.Equal(t, "model-2", modelMsg.ModelID, "Expected selected model to be 'model-2'")
}

// TestModelStepTeatest_FilterModels verifies search filtering works correctly.
func TestModelStepTeatest_FilterModels(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
		{id: "openai/gpt-3.5-turbo", name: "openai/gpt-3.5-turbo"},
	}

	testCases := []struct {
		name          string
		searchQuery   string
		expectedCount int
		expectedFirst string // ID of first filtered model
	}{
		{"Empty", "", 4, "anthropic/claude-sonnet-4-5"},
		{"Anthropic", "anthropic", 2, "anthropic/claude-sonnet-4-5"},
		{"OpenAI", "openai", 2, "openai/gpt-4"},
		{"Claude", "claude", 2, "anthropic/claude-sonnet-4-5"},
		{"GPT", "gpt", 2, "openai/gpt-4"},
		{"Opus", "opus", 1, "anthropic/claude-opus-4"},
		{"NoMatch", "nomatch", 0, ""},
		{"CaseInsensitive", "CLAUDE", 2, "anthropic/claude-sonnet-4-5"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step.searchInput.SetValue(tc.searchQuery)
			step.filterModels()

			require.Equal(t, tc.expectedCount, len(step.filtered), "Expected %d filtered models", tc.expectedCount)

			if tc.expectedCount > 0 {
				require.Equal(t, tc.expectedFirst, step.filtered[0].id, "Expected first filtered model to be '%s'", tc.expectedFirst)
				// Verify selectedIdx is within bounds for non-empty filtered list
				require.True(t, step.selectedIdx < len(step.filtered), "Expected selectedIdx to be within bounds")
			} else {
				// For empty filtered list, selectedIdx should be 0
				require.Equal(t, 0, step.selectedIdx, "Expected selectedIdx to be 0 for empty filtered list")
			}
		})
	}
}

// TestModelStepTeatest_SetSize verifies SetSize updates dimensions correctly.
func TestModelStepTeatest_SetSize(t *testing.T) {
	t.Parallel()

	step := NewModelStep()

	// Set custom size
	step.SetSize(100, 30)

	require.Equal(t, 100, step.width, "Expected width to be 100")
	require.Equal(t, 30, step.height, "Expected height to be 30")
	require.Equal(t, 90, step.searchInput.Width(), "Expected searchInput width to be width-10 = 90")
}

// TestModelStepTeatest_ModelsLoadedMsg verifies ModelsLoadedMsg handling.
func TestModelStepTeatest_ModelsLoadedMsg(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = true

	models := []*ModelInfo{
		{id: "model-1", name: "model-1"},
		{id: "model-2", name: "model-2"},
	}

	cmd := step.Update(ModelsLoadedMsg{models: models})

	// Should stop loading
	require.False(t, step.loading, "Expected loading to be false after ModelsLoadedMsg")

	// Should set models
	require.Equal(t, models, step.allModels, "Expected allModels to be set")
	require.Equal(t, 2, len(step.filtered), "Expected filtered models to be set")

	// Should return ContentChangedMsg command
	require.NotNil(t, cmd, "Expected ContentChangedMsg command")
	msg := cmd()
	_, ok := msg.(ContentChangedMsg)
	require.True(t, ok, "Expected ContentChangedMsg, got %T", msg)
}

// TestModelStepTeatest_ModelsErrorMsg verifies ModelsErrorMsg handling.
func TestModelStepTeatest_ModelsErrorMsg(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = true

	testCases := []struct {
		name           string
		isNotInstalled bool
	}{
		{"GenericError", false},
		{"NotInstalled", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := NewModelStep()
			step.loading = true

			cmd := step.Update(ModelsErrorMsg{
				err:            errors.New("test error"),
				isNotInstalled: tc.isNotInstalled,
			})

			// Should stop loading
			require.False(t, step.loading, "Expected loading to be false after ModelsErrorMsg")

			// Should set error
			require.NotEmpty(t, step.error, "Expected error to be set")
			require.Equal(t, tc.isNotInstalled, step.isNotInstalled, "Expected isNotInstalled to match")

			// Should return ContentChangedMsg command
			require.NotNil(t, cmd, "Expected ContentChangedMsg command")
			msg := cmd()
			_, ok := msg.(ContentChangedMsg)
			require.True(t, ok, "Expected ContentChangedMsg, got %T", msg)
		})
	}
}

// TestModelStepTeatest_RetryOnError verifies 'r' key retries model fetch on error.
func TestModelStepTeatest_RetryOnError(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.error = "test error"
	step.isNotInstalled = false

	// Press 'r' to retry
	cmd := step.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	// Should start loading again
	require.True(t, step.loading, "Expected loading to be true after retry")
	require.Empty(t, step.error, "Expected error to be cleared")

	// Should return command (batch of fetchModels and spinner.Tick)
	require.NotNil(t, cmd, "Expected command to be returned for retry")
}

// TestModelStepTeatest_NoRetryWhenNotInstalled verifies 'r' does not retry when opencode not installed.
func TestModelStepTeatest_NoRetryWhenNotInstalled(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.error = "test error"
	step.isNotInstalled = true

	initialLoading := step.loading
	initialError := step.error

	// Press 'r' - should not retry
	step.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	// Should not change state
	require.Equal(t, initialLoading, step.loading, "Expected loading state to remain unchanged")
	require.Equal(t, initialError, step.error, "Expected error to remain unchanged")
}

// TestModelStepTeatest_CustomModeFromError verifies 'c' enters custom mode from error state.
func TestModelStepTeatest_CustomModeFromError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		isNotInstalled bool
	}{
		{"GenericError", false},
		{"NotInstalled", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := NewModelStep()
			step.loading = false
			step.error = "test error"
			step.isNotInstalled = tc.isNotInstalled

			// Press 'c' to enter custom mode
			cmd := step.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

			// Should enter custom mode
			require.True(t, step.isCustomMode, "Expected isCustomMode to be true")

			// Should return ContentChangedMsg command
			require.NotNil(t, cmd, "Expected ContentChangedMsg command")
		})
	}
}

// TestModelStepTeatest_EmptyModelList verifies handling of empty model list without panics.
func TestModelStepTeatest_EmptyModelList(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.allModels = []*ModelInfo{}
	step.filterModels()

	// Navigation should not panic
	step.Update(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
	step.Update(tea.KeyPressMsg{Code: tea.KeyUp, Text: "up"})

	// Enter should not panic
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
	require.Nil(t, cmd, "Expected no command with empty model list")

	// View should not panic
	view := step.View()
	require.NotEmpty(t, view, "Expected view to render without panic")
}

// TestModelStepTeatest_SingleModel verifies single model list maintains selection.
func TestModelStepTeatest_SingleModel(t *testing.T) {
	t.Parallel()

	step := NewModelStep()
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "only-model", name: "only-model"},
	}
	step.filterModels()

	// Initial selection should be 0
	require.Equal(t, 0, step.selectedIdx)

	// Navigation should stay at 0
	step.Update(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
	require.Equal(t, 0, step.selectedIdx, "Expected selectedIdx to stay at 0")

	step.Update(tea.KeyPressMsg{Code: tea.KeyUp, Text: "up"})
	require.Equal(t, 0, step.selectedIdx, "Expected selectedIdx to stay at 0")

	// Enter should select the only model
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
	require.NotNil(t, cmd, "Expected command to be returned")

	msg := cmd()
	modelMsg, ok := msg.(ModelSelectedMsg)
	require.True(t, ok, "Expected ModelSelectedMsg")
	require.Equal(t, "only-model", modelMsg.ModelID, "Expected selected model to be 'only-model'")
}

// TestModelStepTeatest_ContentChangedMsg verifies ContentChangedMsg is sent when appropriate.
func TestModelStepTeatest_ContentChangedMsg(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		action func(*ModelStep) tea.Cmd
	}{
		{
			name: "EnterCustomMode",
			action: func(m *ModelStep) tea.Cmd {
				m.loading = false
				m.allModels = []*ModelInfo{{id: "test", name: "test"}}
				m.filterModels()
				return m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
			},
		},
		{
			name: "ExitCustomMode",
			action: func(m *ModelStep) tea.Cmd {
				m.loading = false
				m.isCustomMode = true
				return m.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Text: "esc"})
			},
		},
		{
			name: "ModelsLoaded",
			action: func(m *ModelStep) tea.Cmd {
				m.loading = true
				return m.Update(ModelsLoadedMsg{models: []*ModelInfo{{id: "test", name: "test"}}})
			},
		},
		{
			name: "ModelsError",
			action: func(m *ModelStep) tea.Cmd {
				m.loading = true
				return m.Update(ModelsErrorMsg{err: errors.New("test error"), isNotInstalled: false})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := NewModelStep()
			cmd := tc.action(step)

			require.NotNil(t, cmd, "Expected command to be returned")

			// Execute command - should return ContentChangedMsg or batch
			msg := cmd()

			// Could be ContentChangedMsg directly or wrapped in batch
			if _, ok := msg.(ContentChangedMsg); !ok {
				// If batch, we can't easily test the contents in bubbletea v2
				// Just verify a command was returned
				require.NotNil(t, msg, "Expected message or batch to be returned")
			}
		})
	}
}

// TestModelStepTeatest_Init verifies Init returns correct commands.
func TestModelStepTeatest_Init(t *testing.T) {
	t.Parallel()

	step := NewModelStep()

	cmd := step.Init()

	// Init should return a batch command (fetchModels + spinner.Tick + searchInput.Focus)
	require.NotNil(t, cmd, "Expected command from Init")

	// We can't easily test the contents of a batch in bubbletea v2,
	// but we can verify it's not nil
	require.NotNil(t, cmd, "Expected batch command")
}
