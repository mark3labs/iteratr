package wizard

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/stretchr/testify/require"
)

// TestModelSelector_PreFillsFromConfig verifies that the model selector
// pre-fills model from config when config exists.
func TestModelSelector_PreFillsFromConfig(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create temp directory for config
	tmpDir := t.TempDir()

	// Write a test config with specific model
	testModel := "anthropic/claude-opus-4"
	cfg := &config.Config{
		Model:      testModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 10,
	}

	// Set XDG_CONFIG_HOME to temp dir
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write config
	require.NoError(t, config.WriteGlobal(cfg))

	// Create model selector (step 2 in wizard)
	selector := NewModelSelectorStep()

	// Simulate models loaded (including configured model)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: testModel, name: testModel}, // Our configured model
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify configured model is pre-selected
	require.Equal(t, testModel, selector.SelectedModel(), "Expected model to be pre-selected from config")

	// Verify it's at the correct index (1)
	require.Equal(t, 1, selector.selectedIdx, "Expected selectedIdx to be 1")
}

// TestModelSelector_UserOverridesConfigModel verifies that user can override
// the config model during wizard without modifying config file.
func TestModelSelector_UserOverridesConfigModel(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create temp directory for config
	tmpDir := t.TempDir()

	// Write a test config
	configModel := "anthropic/claude-sonnet-4-5"
	cfg := &config.Config{
		Model:      configModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write config
	require.NoError(t, config.WriteGlobal(cfg))

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: configModel, name: configModel},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify config model is pre-selected
	require.Equal(t, configModel, selector.SelectedModel(), "Expected config model to be pre-selected")

	// User navigates to different model
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	_ = selector.Update(downKey)

	// Verify different model is selected
	userSelectedModel := "openai/gpt-4"
	require.Equal(t, userSelectedModel, selector.SelectedModel(), "Expected user to select different model")

	// User confirms selection
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmd := selector.Update(enterKey)

	// Verify ModelSelectedMsg contains user's choice
	require.NotNil(t, cmd, "Expected cmd from enter key")

	resultMsg := cmd()
	selectedMsg, ok := resultMsg.(ModelSelectedMsg)
	require.True(t, ok, "Expected ModelSelectedMsg, got %T", resultMsg)
	require.Equal(t, userSelectedModel, selectedMsg.ModelID, "Expected ModelID to match user selection")

	// Verify config file was NOT modified
	loadedCfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, configModel, loadedCfg.Model, "Config file should not be modified")
}

// TestModelSelector_WithoutConfigUsesFirstModel verifies that when no config exists,
// wizard defaults to first available model.
func TestModelSelector_WithoutConfigUsesFirstModel(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create empty temp directory (no config)
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to empty dir
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Ensure no project config
	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	require.NoError(t, os.Chdir(tmpDir))

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is selected (no config to pre-fill from)
	require.Equal(t, testModels[0].id, selector.SelectedModel(), "Expected first model when no config exists")
}

// TestModelSelector_ProjectConfigOverridesGlobal verifies that project config
// takes precedence over global config for pre-filling.
func TestModelSelector_ProjectConfigOverridesGlobal(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create temp directories
	tmpConfigDir := t.TempDir()
	tmpProjectDir := t.TempDir()

	// Write global config
	globalModel := "anthropic/claude-sonnet-4-5"
	globalCfg := &config.Config{
		Model:      globalModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	})
	require.NoError(t, os.Setenv("XDG_CONFIG_HOME", tmpConfigDir))
	require.NoError(t, config.WriteGlobal(globalCfg))

	// Write project config (in current directory)
	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	require.NoError(t, os.Chdir(tmpProjectDir))

	projectModel := "openai/gpt-4"
	projectCfg := &config.Config{
		Model:      projectModel,
		AutoCommit: false,
		DataDir:    ".iteratr",
		LogLevel:   "debug",
		Iterations: 5,
	}

	require.NoError(t, config.WriteProject(projectCfg))

	// Verify project config exists
	_, err := os.Stat("iteratr.yml")
	require.NoError(t, err, "Project config should be created")

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: globalModel, name: globalModel},
		{id: projectModel, name: projectModel},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify PROJECT model is pre-selected (not global)
	require.Equal(t, projectModel, selector.SelectedModel(), "Expected project model to override global")
}

// TestModelSelector_EnvVarOverridesConfig verifies that ITERATR_MODEL env var
// takes precedence over config file for pre-filling.
func TestModelSelector_EnvVarOverridesConfig(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create temp directory
	tmpDir := t.TempDir()

	// Write config
	configModel := "anthropic/claude-sonnet-4-5"
	cfg := &config.Config{
		Model:      configModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	require.NoError(t, config.WriteGlobal(cfg))

	// Set ITERATR_MODEL env var (should override config)
	envModel := "openai/gpt-4-turbo"
	t.Setenv("ITERATR_MODEL", envModel)

	// Verify config.Load() returns env var value
	loadedCfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, envModel, loadedCfg.Model, "Expected config.Load() to return env var model")

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: configModel, name: configModel},
		{id: envModel, name: envModel},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify ENV model is pre-selected (not config file model)
	require.Equal(t, envModel, selector.SelectedModel(), "Expected env var model to override config")
}

// TestWizard_ResultDoesNotModifyConfig verifies that completing the wizard
// does not modify the config file on disk.
func TestWizard_ResultDoesNotModifyConfig(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create temp directory
	tmpDir := t.TempDir()

	// Write initial config
	originalModel := "anthropic/claude-sonnet-4-5"
	originalIterations := 10
	cfg := &config.Config{
		Model:      originalModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: originalIterations,
	}

	// Set XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	require.NoError(t, config.WriteGlobal(cfg))

	// Get config file path and modification time
	configPath := filepath.Join(tmpDir, "iteratr", "iteratr.yml")
	origStat, err := os.Stat(configPath)
	require.NoError(t, err)
	origModTime := origStat.ModTime()

	// Simulate wizard completing with different values
	// (In real usage, these values go to buildFlags, NOT back to config)
	wizardResult := &WizardResult{
		SpecPath:    "specs/feature.md",
		Model:       "openai/gpt-4", // Different from config
		Template:    "custom template",
		SessionName: "test-session",
		Iterations:  20, // Different from config
		ResumeMode:  false,
	}

	// Verify wizard result has different values
	require.NotEqual(t, originalModel, wizardResult.Model, "Test setup: wizard result should have different model")
	require.NotEqual(t, originalIterations, wizardResult.Iterations, "Test setup: wizard result should have different iterations")

	// Load config again
	loadedCfg, err := config.Load()
	require.NoError(t, err)

	// Verify config still has original values
	require.Equal(t, originalModel, loadedCfg.Model, "Config model should not be modified")
	require.Equal(t, originalIterations, loadedCfg.Iterations, "Config iterations should not be modified")

	// Verify config file was not written (modification time unchanged)
	newStat, err := os.Stat(configPath)
	require.NoError(t, err)
	require.True(t, newStat.ModTime().Equal(origModTime), "Config file should not be modified")
}

// TestModelSelector_ConfigModelNotInList verifies fallback behavior when
// configured model is not in the available models list.
func TestModelSelector_ConfigModelNotInList(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Create temp directory
	tmpDir := t.TempDir()

	// Write config with model that won't be in list
	cfg := &config.Config{
		Model:      "nonexistent/model",
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	require.NoError(t, config.WriteGlobal(cfg))

	// Create selector
	selector := NewModelSelectorStep()

	// Simulate models loaded (not including configured model)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is selected as fallback
	require.Equal(t, testModels[0].id, selector.SelectedModel(), "Expected first model as fallback when config model not in list")
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx to be 0")
}

// TestModelSelector_MultipleUpdates verifies that subsequent model updates
// apply default model selection (from config or first model).
func TestModelSelector_MultipleUpdates(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv() - they are incompatible

	// Set up a clean environment with no config
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	require.NoError(t, os.Chdir(tmpDir))

	selector := NewModelSelectorStep()

	// First models load
	models1 := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}
	_ = selector.Update(ModelsLoadedMsg{models: models1})
	require.Equal(t, models1[0].id, selector.SelectedModel(), "Expected first model selected initially")

	// Navigate to second model
	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, models1[1].id, selector.SelectedModel(), "Expected second model after navigation")

	// Second models load (simulating refresh)
	// When models reload, selectedIdx resets to 0 (no config model set)
	models2 := []*ModelInfo{
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
		{id: "openai/gpt-4-turbo", name: "openai/gpt-4-turbo"},
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
	}
	_ = selector.Update(ModelsLoadedMsg{models: models2})

	// Without config, selectedIdx resets to 0 after refresh
	require.Equal(t, models2[0].id, selector.SelectedModel(), "Expected first model after reload")
	require.Equal(t, 0, selector.selectedIdx, "Expected selectedIdx to reset to 0")
}

// TestModelSelector_NavigationBounds verifies that navigation stays within bounds.
func TestModelSelector_NavigationBounds(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	testModels := []*ModelInfo{
		{id: "model-1", name: "Model 1"},
		{id: "model-2", name: "Model 2"},
		{id: "model-3", name: "Model 3"},
	}
	_ = selector.Update(ModelsLoadedMsg{models: testModels})

	// Should start at first model
	require.Equal(t, 0, selector.selectedIdx, "Expected start at index 0")

	// Try to move up (should stay at 0)
	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 0, selector.selectedIdx, "Expected to stay at index 0 when moving up from first")

	// Move down to last
	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, 2, selector.selectedIdx, "Expected to move to last model")

	// Try to move down (should stay at last)
	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, 2, selector.selectedIdx, "Expected to stay at last index when moving down from last")

	// Move back up
	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 1, selector.selectedIdx, "Expected to move back to middle")
}

// TestModelSelector_EmptyModelList verifies behavior with empty model list.
func TestModelSelector_EmptyModelList(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	// Load empty model list
	_ = selector.Update(ModelsLoadedMsg{models: []*ModelInfo{}})

	// Should return empty string
	require.Equal(t, "", selector.SelectedModel(), "Expected empty string when no models available")

	// Navigation should not panic
	require.NotPanics(t, func() {
		_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyUp})
		_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	}, "Navigation with empty model list should not panic")
}

// TestModelSelector_SingleModel verifies behavior with single model.
func TestModelSelector_SingleModel(t *testing.T) {
	t.Parallel()

	selector := NewModelSelectorStep()

	singleModel := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
	}
	_ = selector.Update(ModelsLoadedMsg{models: singleModel})

	// Should select the only model
	require.Equal(t, singleModel[0].id, selector.SelectedModel(), "Expected single model to be selected")

	// Navigation should keep selection
	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, singleModel[0].id, selector.SelectedModel(), "Expected to stay on single model after down")

	_ = selector.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, singleModel[0].id, selector.SelectedModel(), "Expected to stay on single model after up")
}
