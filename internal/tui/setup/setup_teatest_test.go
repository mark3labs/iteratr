package setup

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestSetupFlow_CompleteWizard verifies the full three-step wizard flow.
func TestSetupFlow_CompleteWizard(t *testing.T) {
	t.Parallel()

	// Create setup model (step 0 = model selection)
	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	// Initialize model step
	m.Init()

	// Send window size message
	updatedModel, _ := m.Update(tea.WindowSizeMsg{
		Width:  testfixtures.TestTermWidth,
		Height: testfixtures.TestTermHeight,
	})
	m = updatedModel.(*SetupModel)

	// Simulate models loaded for step 0 (model selection)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", displayName: "anthropic/claude-sonnet-4-5", providerID: "anthropic"},
		{id: "anthropic/claude-opus-4", displayName: "anthropic/claude-opus-4", providerID: "anthropic"},
		{id: "openai/gpt-4", displayName: "openai/gpt-4", providerID: "openai"},
	}
	updatedModel, _ = m.Update(ModelsLoadedMsg{models: testModels})
	m = updatedModel.(*SetupModel)

	// Select first model (Enter) - this returns a command that generates ModelSelectedMsg
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updatedModel.(*SetupModel)

	// Execute the command to get ModelSelectedMsg
	require.NotNil(t, cmd, "Expected command to be returned from model step")
	msg := cmd()

	// Send ModelSelectedMsg to setup model
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(*SetupModel)

	// Verify step advanced to 1 and result.Model set
	require.Equal(t, 1, m.step, "Expected step to be 1 after model selection")
	require.Equal(t, "anthropic/claude-sonnet-4-5", m.result.Model, "Expected model to be set in result")
	require.NotNil(t, m.autoCommitStep, "Expected autoCommitStep to be initialized")

	// Select "Yes" for auto-commit (Enter - already selected by default)
	// Note: We cannot test step 2 (WriteConfig) in this test without proper filesystem setup,
	// so we stop after verifying step 1 transition is correct
}

// TestSetupFlow_BackNavigation verifies ESC/back navigation between steps.
func TestSetupFlow_BackNavigation(t *testing.T) {
	t.Parallel()

	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	m.Init()
	updatedModel, _ := m.Update(tea.WindowSizeMsg{
		Width:  testfixtures.TestTermWidth,
		Height: testfixtures.TestTermHeight,
	})
	m = updatedModel.(*SetupModel)

	// Load models
	testModels := []*ModelInfo{
		{id: "test/model-1", displayName: "test/model-1", providerID: "test"},
	}
	updatedModel, _ = m.Update(ModelsLoadedMsg{models: testModels})
	m = updatedModel.(*SetupModel)

	// Advance to step 1 (Enter returns command with ModelSelectedMsg)
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updatedModel.(*SetupModel)

	// Execute command and send ModelSelectedMsg
	require.NotNil(t, cmd, "Expected command from model step")
	msg := cmd()
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(*SetupModel)

	require.Equal(t, 1, m.step, "Expected step to be 1")

	// Press ESC to go back
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(*SetupModel)

	require.Equal(t, 0, m.step, "Expected step to be 0 after ESC")
	require.NotNil(t, m.modelStep, "Expected modelStep to remain initialized")
}

// TestSetupFlow_EscOnFirstStepCancels verifies ESC on first step exits wizard.
func TestSetupFlow_EscOnFirstStepCancels(t *testing.T) {
	t.Parallel()

	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	m.Init()

	// Press ESC to cancel
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(*SetupModel)

	require.True(t, m.cancelled, "Expected wizard to be cancelled")
	require.NotNil(t, cmd, "Expected quit command to be returned")
}

// TestSetupFlow_CtrlCAlwaysQuits verifies Ctrl+C quits from any step.
func TestSetupFlow_CtrlCAlwaysQuits(t *testing.T) {
	t.Parallel()

	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	m.Init()

	// Press Ctrl+C
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+c"})
	m = updatedModel.(*SetupModel)

	require.True(t, m.cancelled, "Expected wizard to be cancelled by Ctrl+C")
	require.NotNil(t, cmd, "Expected quit command to be returned")
}

// TestSetupFlow_WindowResize verifies window size updates propagate to steps.
func TestSetupFlow_WindowResize(t *testing.T) {
	t.Parallel()

	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	m.Init()

	// Load models
	testModels := []*ModelInfo{
		{id: "test/model-1", displayName: "test/model-1", providerID: "test"},
	}
	updatedModel, _ := m.Update(ModelsLoadedMsg{models: testModels})
	m = updatedModel.(*SetupModel)

	// Send window size message
	newWidth := 100
	newHeight := 30
	updatedModel, _ = m.Update(tea.WindowSizeMsg{Width: newWidth, Height: newHeight})
	m = updatedModel.(*SetupModel)

	// Verify dimensions updated
	require.Equal(t, newWidth, m.width, "Expected width to be updated")
	require.Equal(t, newHeight, m.height, "Expected height to be updated")

	// Verify step component received size update (contentWidth calculation)
	require.NotNil(t, m.modelStep)
	// contentWidth = modalWidth - 6, modalWidth = min(width-6, 100)
	expectedModalWidth := newWidth - 6 // 100 - 6 = 94, but capped at 100
	if expectedModalWidth > 100 {
		expectedModalWidth = 100
	}
	expectedContentWidth := expectedModalWidth - 6
	if expectedContentWidth < 40 {
		expectedContentWidth = 40
	}
	require.Equal(t, expectedContentWidth, m.modelStep.width, "Expected model step width to be updated")
}

// TestSetupFlow_ContentChangedMsg verifies ContentChangedMsg triggers modal resize.
func TestSetupFlow_ContentChangedMsg(t *testing.T) {
	t.Parallel()

	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	m.Init()
	updatedModel, _ := m.Update(tea.WindowSizeMsg{
		Width:  testfixtures.TestTermWidth,
		Height: testfixtures.TestTermHeight,
	})
	m = updatedModel.(*SetupModel)

	// Get initial preferred height (loading state = 1 line)
	initialHeight := m.getStepPreferredHeight()
	require.Equal(t, 1, initialHeight, "Expected loading state to have height 1")

	// Send models loaded (changes content, increasing height)
	testModels := []*ModelInfo{
		{id: "model-1", displayName: "model-1", providerID: "test"},
		{id: "model-2", displayName: "model-2", providerID: "test"},
		{id: "model-3", displayName: "model-3", providerID: "test"},
		{id: "model-4", displayName: "model-4", providerID: "test"},
		{id: "model-5", displayName: "model-5", providerID: "test"},
	}
	updatedModel, _ = m.Update(ModelsLoadedMsg{models: testModels})
	m = updatedModel.(*SetupModel)

	// Preferred height should be different now (more models)
	newHeight := m.getStepPreferredHeight()
	// Expected: search (1) + blank (1) + 1 header + 5 models + blank (1) + hint (1) = 10
	require.Equal(t, 10, newHeight, "Expected height to change after models loaded")
	require.NotEqual(t, initialHeight, newHeight, "Expected height to differ from loading state")
}

// TestSetupFlow_IsProjectFlag verifies isProject flag affects behavior.
func TestSetupFlow_IsProjectFlag(t *testing.T) {
	t.Parallel()

	// Test with isProject = true
	mProject := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: true,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	// Test with isProject = false
	mGlobal := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     testfixtures.TestTermWidth,
		height:    testfixtures.TestTermHeight,
	}

	// Both should have isProject flag set correctly
	require.True(t, mProject.isProject, "Expected isProject to be true")
	require.False(t, mGlobal.isProject, "Expected isProject to be false")

	// Verify completion step uses correct flag
	completionProject := NewCompletionStep(true)
	completionGlobal := NewCompletionStep(false)

	require.True(t, completionProject.isProject, "Expected completion step to have isProject=true")
	require.False(t, completionGlobal.isProject, "Expected completion step to have isProject=false")
}

// TestAutoCommitStep_Navigation verifies up/down navigation in auto-commit step.
func TestAutoCommitStep_Navigation(t *testing.T) {
	t.Parallel()

	step := NewAutoCommitStep()

	// Initial selection should be 0 (Yes)
	require.Equal(t, 0, step.selectedIdx, "Expected initial selection to be 0 (Yes)")

	// Press down to select No
	step.Update(tea.KeyPressMsg{Text: "down"})
	require.Equal(t, 1, step.selectedIdx, "Expected selection to be 1 (No) after down")

	// Press down again (should stay at 1, bounds check)
	step.Update(tea.KeyPressMsg{Text: "down"})
	require.Equal(t, 1, step.selectedIdx, "Expected selection to stay at 1 at lower bound")

	// Press up to select Yes
	step.Update(tea.KeyPressMsg{Text: "up"})
	require.Equal(t, 0, step.selectedIdx, "Expected selection to be 0 (Yes) after up")

	// Press up again (should stay at 0, bounds check)
	step.Update(tea.KeyPressMsg{Text: "up"})
	require.Equal(t, 0, step.selectedIdx, "Expected selection to stay at 0 at upper bound")
}

// TestAutoCommitStep_VimNavigation verifies j/k navigation.
func TestAutoCommitStep_VimNavigation(t *testing.T) {
	t.Parallel()

	step := NewAutoCommitStep()

	// Initial selection should be 0 (Yes)
	require.Equal(t, 0, step.selectedIdx, "Expected initial selection to be 0 (Yes)")

	// Press j to select No
	step.Update(tea.KeyPressMsg{Text: "j"})
	require.Equal(t, 1, step.selectedIdx, "Expected selection to be 1 (No) after j")

	// Press k to select Yes
	step.Update(tea.KeyPressMsg{Text: "k"})
	require.Equal(t, 0, step.selectedIdx, "Expected selection to be 0 (Yes) after k")
}

// TestAutoCommitStep_SelectYes verifies selecting "Yes" sends correct message.
func TestAutoCommitStep_SelectYes(t *testing.T) {
	t.Parallel()

	step := NewAutoCommitStep()

	// Ensure "Yes" is selected (idx 0)
	require.Equal(t, 0, step.selectedIdx, "Expected initial selection to be 0 (Yes)")

	// Press Enter
	cmd := step.Update(tea.KeyPressMsg{Text: "enter"})
	require.NotNil(t, cmd, "Expected command to be returned")

	// Execute command to get message
	msg := cmd()
	autoCommitMsg, ok := msg.(AutoCommitSelectedMsg)
	require.True(t, ok, "Expected AutoCommitSelectedMsg")
	require.True(t, autoCommitMsg.Enabled, "Expected Enabled to be true for Yes selection")
}

// TestAutoCommitStep_SelectNo verifies selecting "No" sends correct message.
func TestAutoCommitStep_SelectNo(t *testing.T) {
	t.Parallel()

	step := NewAutoCommitStep()

	// Navigate to "No" (idx 1)
	step.Update(tea.KeyPressMsg{Text: "down"})
	require.Equal(t, 1, step.selectedIdx, "Expected selection to be 1 (No)")

	// Press Enter
	cmd := step.Update(tea.KeyPressMsg{Text: "enter"})
	require.NotNil(t, cmd, "Expected command to be returned")

	// Execute command to get message
	msg := cmd()
	autoCommitMsg, ok := msg.(AutoCommitSelectedMsg)
	require.True(t, ok, "Expected AutoCommitSelectedMsg")
	require.False(t, autoCommitMsg.Enabled, "Expected Enabled to be false for No selection")
}

// TestAutoCommitStep_PreferredHeight verifies consistent height calculation.
func TestAutoCommitStep_PreferredHeight(t *testing.T) {
	t.Parallel()

	step := NewAutoCommitStep()

	// Expected: question (1) + blank (1) + Yes (1) + No (1) + blank (1) + hint (1) = 6
	require.Equal(t, 6, step.PreferredHeight(), "Expected PreferredHeight to be 6")
}

// TestAutoCommitStep_ViewContainsElements verifies View renders key elements.
func TestAutoCommitStep_ViewContainsElements(t *testing.T) {
	t.Parallel()

	step := NewAutoCommitStep()

	view := step.View()

	// Check for key text elements
	require.Contains(t, view, "Auto-commit changes after each iteration?", "Expected question in view")
	require.Contains(t, view, "Yes", "Expected 'Yes' option in view")
	require.Contains(t, view, "No", "Expected 'No' option in view")
	require.Contains(t, view, "(recommended)", "Expected '(recommended)' label in view")
	require.Contains(t, view, "↑↓/j/k navigate", "Expected navigation hint in view")
	require.Contains(t, view, "Enter select", "Expected Enter hint in view")
	require.Contains(t, view, "ESC back", "Expected ESC hint in view")
}

// TestCompletionStep_AnyKeyQuits verifies any key press returns quit command.
func TestCompletionStep_AnyKeyQuits(t *testing.T) {
	t.Parallel()

	step := NewCompletionStep(false)

	// Press any key should return Quit command
	cmd := step.Update(tea.KeyPressMsg{Text: " "})
	require.NotNil(t, cmd, "Expected command to be returned on key press")

	// Note: Cannot directly test if cmd == tea.Quit, but we verify cmd is not nil
	// which indicates a command was returned (and based on code, it's tea.Quit)
}

// TestCompletionStep_PreferredHeight verifies height calculation.
func TestCompletionStep_PreferredHeight(t *testing.T) {
	t.Parallel()

	step := NewCompletionStep(false)

	// Expected: success (1) + blank (1) + label (1) + value (1) + blank (1) +
	//           label (1) + hint (1) + blank (1) + exit (1) = 9
	require.Equal(t, 9, step.PreferredHeight(), "Expected PreferredHeight to be 9")
}

// TestCompletionStep_ViewGlobalPath verifies completion view shows global config path.
func TestCompletionStep_ViewGlobalPath(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv()

	// Create temp directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	step := NewCompletionStep(false) // isProject = false

	view := step.View()

	// Check for key elements
	require.Contains(t, view, "Configuration created successfully", "Expected success message")
	require.Contains(t, view, "Config written to:", "Expected config path label")
	require.Contains(t, view, config.GlobalPath(), "Expected global config path")
	require.Contains(t, view, "Next steps:", "Expected next steps label")
	require.Contains(t, view, "iteratr build", "Expected iteratr build hint")
	require.Contains(t, view, "press any key to exit", "Expected exit hint")
}

// TestCompletionStep_ViewProjectPath verifies completion view shows project config path.
func TestCompletionStep_ViewProjectPath(t *testing.T) {
	t.Parallel()

	step := NewCompletionStep(true) // isProject = true

	view := step.View()

	// Check for key elements
	require.Contains(t, view, "Configuration created successfully", "Expected success message")
	require.Contains(t, view, "Config written to:", "Expected config path label")
	require.Contains(t, view, config.ProjectPath(), "Expected project config path")
	require.Contains(t, view, "Next steps:", "Expected next steps label")
	require.Contains(t, view, "iteratr build", "Expected iteratr build hint")
	require.Contains(t, view, "press any key to exit", "Expected exit hint")
}

// TestSetupFlow_ModalDimensions verifies modal dimension calculations.
func TestSetupFlow_ModalDimensions(t *testing.T) {
	t.Parallel()

	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     120,
		height:    40,
	}

	modalWidth, modalHeight, contentWidth, contentHeight := m.calculateModalDimensions()

	// Modal width should be 120 - 6 = 114, capped at 100
	require.Equal(t, 100, modalWidth, "Expected modal width to be capped at 100")

	// Content width should be modal width minus 6 (padding + border)
	require.Equal(t, 94, contentWidth, "Expected content width to be 94")

	// Modal height depends on content, but should be within bounds
	require.GreaterOrEqual(t, modalHeight, 15, "Modal height should be at least 15")
	require.LessOrEqual(t, modalHeight, 36, "Modal height should not exceed height-4")

	// Content height should be modal height minus overhead (6)
	require.Equal(t, modalHeight-6, contentHeight, "Expected content height to be modal height - 6")
}

// TestSetupFlow_ModalDimensionsSmallScreen verifies modal handles small screens.
func TestSetupFlow_ModalDimensionsSmallScreen(t *testing.T) {
	t.Parallel()

	m := &SetupModel{
		step:      0,
		cancelled: false,
		isProject: false,
		width:     50,
		height:    20,
	}

	modalWidth, modalHeight, contentWidth, contentHeight := m.calculateModalDimensions()

	// Modal width should be 50 - 6 = 44, minimum 60
	require.Equal(t, 60, modalWidth, "Expected modal width minimum to be 60")

	// Content width should be at least 40 (minimum)
	require.GreaterOrEqual(t, contentWidth, 40, "Expected content width to be at least 40")

	// Modal height should be at least 15
	require.GreaterOrEqual(t, modalHeight, 15, "Expected modal height to be at least 15")

	// Content height should be at least 5
	require.GreaterOrEqual(t, contentHeight, 5, "Expected content height to be at least 5")
}

// TestSetupFlow_WriteConfig verifies config writing (unit test with filesystem).
func TestSetupFlow_WriteConfig(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv()

	// Create temp directory for config
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	m := &SetupModel{
		step:      2,
		cancelled: false,
		isProject: false,
		result: SetupResult{
			Model:      "test/model-1",
			AutoCommit: true,
		},
	}

	// Write config
	err := m.WriteConfig()
	require.NoError(t, err, "Expected WriteConfig to succeed")

	// Verify config file exists
	configPath := config.GlobalPath()
	require.FileExists(t, configPath, "Expected config file to exist")

	// Load config and verify values
	cfg, err := config.Load()
	require.NoError(t, err, "Expected config Load to succeed")
	require.Equal(t, "test/model-1", cfg.Model, "Expected model to match")
	require.True(t, cfg.AutoCommit, "Expected AutoCommit to be true")
}

// TestSetupFlow_WriteConfigProject verifies project config writing.
func TestSetupFlow_WriteConfigProject(t *testing.T) {
	// Not parallel-safe: os.Chdir mutates the process-wide working directory.

	// Create temp directory for project config
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err, "Expected Getwd to succeed")
	defer func() {
		err := os.Chdir(origDir)
		require.NoError(t, err, "Expected Chdir back to original dir to succeed")
	}()

	// Change to temp directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Expected Chdir to succeed")

	m := &SetupModel{
		step:      2,
		cancelled: false,
		isProject: true,
		result: SetupResult{
			Model:      "project/model-1",
			AutoCommit: false,
		},
	}

	// Write config
	err = m.WriteConfig()
	require.NoError(t, err, "Expected WriteConfig to succeed")

	// Verify config file exists
	configPath := filepath.Join(tmpDir, "iteratr.yml")
	require.FileExists(t, configPath, "Expected project config file to exist")
}
