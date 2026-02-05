package wizard

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/stretchr/testify/require"
)

// TestModelSelectorTeatest_PreFillFromConfig verifies that the model selector
// pre-selects the model from config after models are loaded.
func TestModelSelectorTeatest_PreFillFromConfig(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create temp directory for config
	tmpDir := t.TempDir()

	// Write a test config with a specific model
	testModel := "test/model-from-config"
	cfg := &config.Config{
		Model:      testModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME to temp dir
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write config using WriteGlobal
	require.NoError(t, config.WriteGlobal(cfg))

	// Create a model selector
	selector := NewModelSelectorStep()

	// Simulate models loaded (including our test model)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: testModel, name: testModel}, // Our configured model
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	// Send ModelsLoadedMsg
	msg := ModelsLoadedMsg{models: testModels}
	cmd := selector.Update(msg)

	// Verify command is returned (ContentChangedMsg)
	require.NotNil(t, cmd, "Expected cmd from Update, got nil")

	// Execute the command to get the message
	resultMsg := cmd()
	_, ok := resultMsg.(ContentChangedMsg)
	require.True(t, ok, "Expected ContentChangedMsg, got %T", resultMsg)

	// Verify the test model is selected
	selectedModel := selector.SelectedModel()
	require.Equal(t, testModel, selectedModel, "Expected model to be pre-selected from config")

	// Verify the selectedIdx is correct (index 1 in testModels)
	require.Equal(t, 1, selector.selectedIdx, "Expected selectedIdx 1")
}

// TestModelSelectorTeatest_NoConfig verifies that the model selector defaults to
// first model when no config exists.
func TestModelSelectorTeatest_NoConfig(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() and t.Chdir() - they are incompatible

	// Ensure no config exists by using empty temp dir
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Also ensure no project config by changing to temp dir
	origWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	// Create a model selector
	selector := NewModelSelectorStep()

	// Simulate models loaded
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	// Send ModelsLoadedMsg
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is selected by default
	selectedModel := selector.SelectedModel()
	require.Equal(t, testModels[0].id, selectedModel, "Expected first model to be selected by default")

	// Verify selectedIdx is 0
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx 0")
}

// TestModelSelectorTeatest_ConfigModelNotInList verifies fallback behavior when
// configured model is not in the available models list.
func TestModelSelectorTeatest_ConfigModelNotInList(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create a temporary config directory
	tmpDir := t.TempDir()

	// Write a test config with a model that won't be in the list
	cfg := &config.Config{
		Model:      "nonexistent/model",
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME to temp dir
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write config using WriteGlobal
	require.NoError(t, config.WriteGlobal(cfg))

	// Create a model selector
	selector := NewModelSelectorStep()

	// Simulate models loaded (not including the configured model)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	// Send ModelsLoadedMsg
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is selected as fallback
	selectedModel := selector.SelectedModel()
	require.Equal(t, testModels[0].id, selectedModel, "Expected first model to be selected as fallback")

	// Verify selectedIdx is 0
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx 0")
}

// TestModelSelectorTeatest_UserOverride verifies that user can navigate and select
// a different model than the pre-selected one.
func TestModelSelectorTeatest_UserOverride(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create a temporary config directory
	tmpDir := t.TempDir()

	// Write a test config
	cfg := &config.Config{
		Model:      "anthropic/claude-sonnet-4-5",
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write config using WriteGlobal
	require.NoError(t, config.WriteGlobal(cfg))

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is pre-selected
	require.Equal(t, "anthropic/claude-sonnet-4-5", selector.SelectedModel(), "Expected first model to be pre-selected")

	// Simulate user pressing "down" to move to second model
	keyMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	_ = selector.Update(keyMsg)

	// Verify second model is now selected
	selectedModel := selector.SelectedModel()
	require.Equal(t, "openai/gpt-4", selectedModel, "Expected second model after down key")

	// Simulate user pressing "enter" to confirm selection
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmd := selector.Update(enterMsg)

	// Verify ModelSelectedMsg is returned
	require.NotNil(t, cmd, "Expected cmd from enter key")

	resultMsg := cmd()
	selectedMsg, ok := resultMsg.(ModelSelectedMsg)
	require.True(t, ok, "Expected ModelSelectedMsg, got %T", resultMsg)

	// Verify the correct model is in the message
	require.Equal(t, "openai/gpt-4", selectedMsg.ModelID, "Expected correct model in ModelSelectedMsg")
}

// TestModelSelectorTeatest_Navigation verifies up/down navigation behavior.
func TestModelSelectorTeatest_Navigation(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Initial position is 0
	require.Equal(t, 0, selector.selectedIdx, "Expected initial selectedIdx 0")

	// Down key moves to index 1
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	_ = selector.Update(downKey)
	require.Equal(t, 1, selector.selectedIdx, "Expected selectedIdx 1 after down")

	// Down key moves to index 2
	_ = selector.Update(downKey)
	require.Equal(t, 2, selector.selectedIdx, "Expected selectedIdx 2 after second down")

	// Down at end stays at end
	_ = selector.Update(downKey)
	require.Equal(t, 2, selector.selectedIdx, "Expected selectedIdx to stay at 2 at end")

	// Up key moves to index 1
	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	_ = selector.Update(upKey)
	require.Equal(t, 1, selector.selectedIdx, "Expected selectedIdx 1 after up")

	// Up key moves to index 0
	_ = selector.Update(upKey)
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx 0 after second up")

	// Up at start stays at start
	_ = selector.Update(upKey)
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx to stay at 0 at start")
}

// TestModelSelectorTeatest_VimNavigation verifies j/k vim-style navigation.
func TestModelSelectorTeatest_VimNavigation(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// j key moves down
	jKey := tea.KeyPressMsg{Code: 'j'}
	_ = selector.Update(jKey)
	require.Equal(t, 1, selector.selectedIdx, "Expected selectedIdx 1 after j")

	// k key moves up
	kKey := tea.KeyPressMsg{Code: 'k'}
	_ = selector.Update(kKey)
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx 0 after k")
}

// TestModelSelectorTeatest_EmptyModelList verifies behavior with empty model list.
func TestModelSelectorTeatest_EmptyModelList(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Load empty models
	msg := ModelsLoadedMsg{models: []*ModelInfo{}}
	_ = selector.Update(msg)

	// SelectedModel should return empty string
	require.Equal(t, "", selector.SelectedModel(), "Expected empty model with empty list")

	// Navigation should not panic
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	require.NotPanics(t, func() {
		_ = selector.Update(downKey)
	}, "Down key should not panic with empty list")

	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	require.NotPanics(t, func() {
		_ = selector.Update(upKey)
	}, "Up key should not panic with empty list")

	// Enter should not panic
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	require.NotPanics(t, func() {
		_ = selector.Update(enterKey)
	}, "Enter key should not panic with empty list")
}

// TestModelSelectorTeatest_SingleModel verifies behavior with single model.
func TestModelSelectorTeatest_SingleModel(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Load single model
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Should be selected
	require.Equal(t, "anthropic/claude-sonnet-4-5", selector.SelectedModel(), "Expected single model to be selected")

	// Navigation should stay at index 0
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	_ = selector.Update(downKey)
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx to stay at 0 with single model")

	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	_ = selector.Update(upKey)
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx to stay at 0 with single model")

	// Enter should work
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmd := selector.Update(enterKey)
	require.NotNil(t, cmd, "Expected cmd from enter key")

	resultMsg := cmd()
	selectedMsg, ok := resultMsg.(ModelSelectedMsg)
	require.True(t, ok, "Expected ModelSelectedMsg")
	require.Equal(t, "anthropic/claude-sonnet-4-5", selectedMsg.ModelID, "Expected correct model in message")
}

// TestModelSelectorTeatest_SearchFilter verifies search filtering functionality.
// Note: This test verifies the filterModels logic by directly manipulating the searchInput value.
// Testing actual keyboard input through textinput.Update is complex and covered by bubbles tests.
func TestModelSelectorTeatest_SearchFilter(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
		{id: "openai/gpt-3.5-turbo", name: "openai/gpt-3.5-turbo"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify all models are initially shown
	require.Len(t, selector.filtered, 4, "Expected all 4 models initially")

	// Set search value directly (testing filterModels logic)
	selector.searchInput.SetValue("claude")
	selector.filterModels()

	// Verify filtered list contains only claude models
	require.Len(t, selector.filtered, 2, "Expected 2 claude models in filtered list")
	require.Equal(t, "anthropic/claude-sonnet-4-5", selector.filtered[0].id)
	require.Equal(t, "anthropic/claude-opus-4", selector.filtered[1].id)

	// Clear search
	selector.searchInput.SetValue("")
	selector.filterModels()

	// Verify all models are back
	require.Len(t, selector.filtered, 4, "Expected all 4 models after clearing search")
}

// TestModelSelectorTeatest_MultipleUpdates verifies subsequent ModelsLoadedMsg resets to default.
func TestModelSelectorTeatest_MultipleUpdates(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Load initial models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Navigate to second model
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	_ = selector.Update(downKey)
	require.Equal(t, 1, selector.selectedIdx, "Expected selectedIdx 1 after navigation")

	// Load models again (simulating refresh)
	// This should reset to default (config model or index 0)
	msg2 := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg2)

	// Selection should be reset to default (index 0 since no config)
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx to reset to 0 after refresh")
	require.Equal(t, "anthropic/claude-sonnet-4-5", selector.SelectedModel(), "Expected first model after refresh")
}

// TestModelSelectorTeatest_ViewNotEmpty verifies that View renders without panicking.
func TestModelSelectorTeatest_ViewNotEmpty(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Set dimensions
	selector.SetSize(80, 20)

	// Load models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Render view
	view := selector.View()

	// Should not be empty
	require.NotEmpty(t, view, "Expected non-empty view")

	// Should contain expected elements
	require.Contains(t, view, "anthropic/claude-sonnet-4-5", "Expected view to contain model name")
	require.Contains(t, view, "openai/gpt-4", "Expected view to contain second model name")
}
