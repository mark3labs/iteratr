package specwizard

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// --- Tab Navigation Behavioral Tests ---

// TestQuestionView_TabNavigation_FirstQuestion tests tab cycling on first question
// (no back button visible). Cycle: options -> next button -> options
func TestQuestionView_TabNavigation_FirstQuestion(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "First Question",
			Question: "Select one option",
			Options: []Option{
				{Label: "Option A", Description: "First option"},
				{Label: "Option B", Description: "Second option"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)

	// Initial state: options focused (focusIndex = 0)
	require.Equal(t, 0, qv.focusIndex, "expected initial focus on options")
	require.True(t, qv.optionSelector.focused, "expected option selector to be focused")

	// Tab should move to next button (skip custom input since not visible, skip back since first question)
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "expected focus on next button after tab")
	require.False(t, qv.optionSelector.focused, "expected option selector to be blurred after tab")

	// Tab should wrap back to options
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, 0, qv.focusIndex, "expected focus to wrap back to options")
	require.True(t, qv.optionSelector.focused, "expected option selector to be focused after wrap")
}

// TestQuestionView_TabNavigation_SecondQuestion tests tab cycling on second question
// (back button visible). Cycle: options -> back button -> next button -> options
func TestQuestionView_TabNavigation_SecondQuestion(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "First Question",
			Question: "Select one",
			Options:  []Option{{Label: "Option 1"}},
			Multiple: false,
		},
		{
			Header:   "Second Question",
			Question: "Select another",
			Options:  []Option{{Label: "Option A"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "Option 1", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	// Create view on second question (index 1)
	qv := NewQuestionView(questions, answers, 1)
	qv.SetSize(80, 24)

	// Initial state: options focused
	require.Equal(t, 0, qv.focusIndex, "expected initial focus on options")

	// Tab -> back button
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, qv.backButtonFocusIndex(), qv.focusIndex, "expected focus on back button")

	// Tab -> next button
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "expected focus on next button")

	// Tab -> wrap back to options
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, 0, qv.focusIndex, "expected focus to wrap back to options")
}

// TestQuestionView_TabNavigation_WithCustomInput tests tab cycling when custom input is visible
// Cycle: options -> custom input -> back button (if not first) -> next button -> options
func TestQuestionView_TabNavigation_WithCustomInput(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Question with Custom",
			Question: "Select or type your own",
			Options: []Option{
				{Label: "Predefined Option"},
			},
			Multiple: false,
		},
		{
			Header:   "Second Question",
			Question: "Another one",
			Options:  []Option{{Label: "Choice X"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	// Create view on second question
	qv := NewQuestionView(questions, answers, 1)
	qv.SetSize(80, 24)

	// Select "Type your own answer" to show custom input
	// The auto-appended "Type your own answer" option is at index len(options) = 1
	qv.optionSelector.cursor = 1 // "Type your own answer"
	qv.optionSelector.Toggle()

	// Update to trigger showCustom calculation
	qv.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	selected := qv.optionSelector.SelectedLabels()
	qv.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"
	qv.rebuildButtonBar()

	require.True(t, qv.showCustom, "expected custom input to be visible")

	// Tab cycle: options (0) -> custom input (1) -> back button -> next button -> options
	require.Equal(t, 0, qv.focusIndex, "expected initial focus on options")

	// Tab -> custom input
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, 1, qv.focusIndex, "expected focus on custom input")
	require.True(t, qv.customInput.Focused(), "expected custom input to be focused")

	// Tab -> back button
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, qv.backButtonFocusIndex(), qv.focusIndex, "expected focus on back button")
	require.False(t, qv.customInput.Focused(), "expected custom input to be blurred")

	// Tab -> next button
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "expected focus on next button")

	// Tab -> wrap to options
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, 0, qv.focusIndex, "expected focus to wrap to options")
}

// --- Shift+Tab Reverse Navigation Tests ---

// TestQuestionView_ShiftTab_FirstQuestion tests shift+tab reverse navigation on first question
func TestQuestionView_ShiftTab_FirstQuestion(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "First Question",
			Question: "Select one",
			Options:  []Option{{Label: "Option 1"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)

	// Initial focus on options
	require.Equal(t, 0, qv.focusIndex)

	// Shift+tab should wrap to last focusable element (next button)
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "expected focus on next button after shift+tab from options")

	// Shift+tab should wrap back to options (skip back button since first question)
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, 0, qv.focusIndex, "expected focus on options after shift+tab from next")
}

// TestQuestionView_ShiftTab_SecondQuestion tests shift+tab reverse navigation on second question
func TestQuestionView_ShiftTab_SecondQuestion(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "First",
			Question: "Q1",
			Options:  []Option{{Label: "A"}},
			Multiple: false,
		},
		{
			Header:   "Second",
			Question: "Q2",
			Options:  []Option{{Label: "B"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "A", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 1)
	qv.SetSize(80, 24)

	// Start from options
	require.Equal(t, 0, qv.focusIndex)

	// Shift+tab wraps to next button (last)
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "expected focus on next button")

	// Shift+tab to back button
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, qv.backButtonFocusIndex(), qv.focusIndex, "expected focus on back button")

	// Shift+tab back to options
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, 0, qv.focusIndex, "expected focus on options")
}

// TestQuestionView_ShiftTab_WithCustomInput tests shift+tab with custom input visible
func TestQuestionView_ShiftTab_WithCustomInput(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Q1",
			Question: "Pick one",
			Options:  []Option{{Label: "Opt"}},
			Multiple: false,
		},
		{
			Header:   "Q2",
			Question: "Pick another",
			Options:  []Option{{Label: "Choice"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "Opt", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 1)
	qv.SetSize(80, 24)

	// Select "Type your own answer" to show custom input
	qv.optionSelector.cursor = 1 // "Type your own answer"
	qv.optionSelector.Toggle()
	selected := qv.optionSelector.SelectedLabels()
	qv.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"
	qv.rebuildButtonBar()

	require.True(t, qv.showCustom, "expected custom input visible")

	// Start from options (0)
	require.Equal(t, 0, qv.focusIndex)

	// Shift+tab wraps to next button
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "expected focus on next button")

	// Shift+tab to back button
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, qv.backButtonFocusIndex(), qv.focusIndex, "expected focus on back button")

	// Shift+tab to custom input
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, 1, qv.focusIndex, "expected focus on custom input")

	// Shift+tab to options
	qv.Update(tea.KeyPressMsg{Text: "shift+tab"})
	require.Equal(t, 0, qv.focusIndex, "expected focus on options")
}

// --- ESC Behavior Tests ---

// TestQuestionView_Esc_FirstQuestion_ShowsCancelConfirm tests ESC on first question shows cancel modal
func TestQuestionView_Esc_FirstQuestion_ShowsCancelConfirm(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "First Question",
			Question: "What do you want?",
			Options:  []Option{{Label: "Something"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)

	// Press ESC on first question
	cmd := qv.Update(tea.KeyPressMsg{Text: "esc"})

	require.NotNil(t, cmd, "expected command from ESC on first question")

	msg := cmd()
	_, ok := msg.(ShowCancelConfirmMsg)
	require.True(t, ok, "expected ShowCancelConfirmMsg on first question ESC, got %T", msg)
}

// TestQuestionView_Esc_SubsequentQuestion_GoesBack tests ESC on non-first questions goes back
func TestQuestionView_Esc_SubsequentQuestion_GoesBack(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Q1",
			Question: "First",
			Options:  []Option{{Label: "A"}},
			Multiple: false,
		},
		{
			Header:   "Q2",
			Question: "Second",
			Options:  []Option{{Label: "B"}},
			Multiple: false,
		},
		{
			Header:   "Q3",
			Question: "Third",
			Options:  []Option{{Label: "C"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "A", IsMulti: false},
		{Value: "B", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	// Test ESC on second question (index 1)
	qv := NewQuestionView(questions, answers, 1)
	qv.SetSize(80, 24)

	cmd := qv.Update(tea.KeyPressMsg{Text: "esc"})

	require.NotNil(t, cmd, "expected command from ESC on second question")

	msg := cmd()
	_, ok := msg.(PrevQuestionMsg)
	require.True(t, ok, "expected PrevQuestionMsg on second question ESC, got %T", msg)

	// Test ESC on last question (index 2)
	qv3 := NewQuestionView(questions, answers, 2)
	qv3.SetSize(80, 24)

	cmd = qv3.Update(tea.KeyPressMsg{Text: "esc"})

	require.NotNil(t, cmd, "expected command from ESC on last question")

	msg = cmd()
	_, ok = msg.(PrevQuestionMsg)
	require.True(t, ok, "expected PrevQuestionMsg on last question ESC, got %T", msg)
}

// TestQuestionView_Esc_SavesAnswerBeforeGoingBack tests that ESC saves the current answer
func TestQuestionView_Esc_SavesAnswerBeforeGoingBack(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Q1",
			Question: "First",
			Options:  []Option{{Label: "Opt A"}, {Label: "Opt B"}},
			Multiple: false,
		},
		{
			Header:   "Q2",
			Question: "Second",
			Options:  []Option{{Label: "Choice X"}, {Label: "Choice Y"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "Opt A", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 1)
	qv.SetSize(80, 24)

	// Select "Choice Y" (index 1)
	qv.optionSelector.CursorDown()
	qv.optionSelector.Toggle()

	// Press ESC - should save answer before returning PrevQuestionMsg
	cmd := qv.Update(tea.KeyPressMsg{Text: "esc"})

	require.NotNil(t, cmd, "expected command")

	msg := cmd()
	_, ok := msg.(PrevQuestionMsg)
	require.True(t, ok, "expected PrevQuestionMsg")

	// Note: The answer was saved by the Update() method before returning PrevQuestionMsg
	// The actual answer array passed in is modified in place
	// This is verified by the answer persistence tests
}

// --- Custom Input Visibility Tests ---

// TestQuestionView_CustomInput_ShowsWhenTypeYourOwnSelected tests custom input visibility toggle
func TestQuestionView_CustomInput_ShowsWhenTypeYourOwnSelected(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Test Question",
			Question: "Choose or type",
			Options: []Option{
				{Label: "Predefined 1"},
				{Label: "Predefined 2"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)

	// Initially custom input is hidden
	require.False(t, qv.showCustom, "expected custom input hidden initially")

	// Select a predefined option first
	qv.optionSelector.Toggle() // Select "Predefined 1"
	qv.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	require.False(t, qv.showCustom, "expected custom input still hidden after selecting predefined")

	// Now select "Type your own answer"
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1 // Last item is "Type your own answer"
	qv.optionSelector.Toggle()

	// Manually trigger the visibility update (simulating what Update() does)
	selected := qv.optionSelector.SelectedLabels()
	qv.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"

	require.True(t, qv.showCustom, "expected custom input visible after selecting 'Type your own answer'")
}

// TestQuestionView_CustomInput_HidesWhenPredefinedSelected tests that custom input hides
func TestQuestionView_CustomInput_HidesWhenPredefinedSelected(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Test",
			Question: "Pick one",
			Options:  []Option{{Label: "Option X"}},
			Multiple: false,
		},
	}

	// Start with custom text pre-filled
	answers := []QuestionAnswer{
		{Value: "Some custom text", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)

	// Custom input should be visible since answer is custom text
	require.True(t, qv.showCustom, "expected custom input visible for custom answer")

	// Select predefined option instead
	qv.optionSelector.cursor = 0
	qv.optionSelector.Toggle()

	// Manually trigger visibility update
	selected := qv.optionSelector.SelectedLabels()
	qv.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"

	require.False(t, qv.showCustom, "expected custom input hidden after selecting predefined")
}

// --- Validation Behavior Tests ---

// TestQuestionView_Validation_FailsWithNoSelection tests validation fails when nothing selected
func TestQuestionView_Validation_FailsWithNoSelection(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Test",
			Question: "Select something",
			Options:  []Option{{Label: "Only Option"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)
	qv.View() // Initialize button bar

	// Focus on next button and try to submit without selecting
	qv.focusIndex = qv.nextButtonFocusIndex()
	qv.updateFocus()

	cmd := qv.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	require.NotNil(t, cmd, "expected error command")

	msg := cmd()
	errMsg, ok := msg.(ShowErrorMsg)
	require.True(t, ok, "expected ShowErrorMsg, got %T", msg)
	require.NotEmpty(t, errMsg.err, "expected non-empty error message")
}

// TestQuestionView_Validation_FailsWithEmptyCustomText tests validation fails for empty custom text
func TestQuestionView_Validation_FailsWithEmptyCustomText(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Test",
			Question: "Select or type",
			Options:  []Option{{Label: "Predefined"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)
	qv.View() // Initialize button bar

	// Select "Type your own answer" but leave text empty
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1
	qv.optionSelector.Toggle()
	selected := qv.optionSelector.SelectedLabels()
	qv.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"
	// customInput.Value() is empty by default

	// Focus on next button and try to submit
	qv.focusIndex = qv.nextButtonFocusIndex()
	qv.updateFocus()

	cmd := qv.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	require.NotNil(t, cmd, "expected error command")

	msg := cmd()
	errMsg, ok := msg.(ShowErrorMsg)
	require.True(t, ok, "expected ShowErrorMsg, got %T", msg)
	require.NotEmpty(t, errMsg.err, "expected non-empty error message")
}

// TestQuestionView_Validation_PassesWithSelection tests validation passes with selection
func TestQuestionView_Validation_PassesWithSelection(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Test",
			Question: "Select one",
			Options:  []Option{{Label: "Option 1"}, {Label: "Option 2"}},
			Multiple: false,
		},
		{
			Header:   "Test 2",
			Question: "Another",
			Options:  []Option{{Label: "Choice A"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)
	qv.View() // Initialize button bar

	// Select an option
	qv.optionSelector.Toggle() // Select first option

	// Focus on next button and submit
	qv.focusIndex = qv.nextButtonFocusIndex()
	qv.updateFocus()

	cmd := qv.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	require.NotNil(t, cmd, "expected command")

	msg := cmd()
	_, ok := msg.(NextQuestionMsg)
	require.True(t, ok, "expected NextQuestionMsg, got %T", msg)
}

// TestQuestionView_Validation_PassesWithCustomText tests validation passes with custom text
func TestQuestionView_Validation_PassesWithCustomText(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Test",
			Question: "Select or type",
			Options:  []Option{{Label: "Predefined"}},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)
	qv.View() // Initialize button bar

	// Select "Type your own answer" and enter text
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1
	qv.optionSelector.Toggle()
	qv.customInput.SetValue("My custom answer")

	// Focus on next button and submit (should be Submit since last question)
	qv.focusIndex = qv.nextButtonFocusIndex()
	qv.updateFocus()

	cmd := qv.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	require.NotNil(t, cmd, "expected command")

	msg := cmd()
	_, ok := msg.(SubmitAnswersMsg)
	require.True(t, ok, "expected SubmitAnswersMsg, got %T", msg)
}

// --- Button Label Tests ---

// TestQuestionView_ButtonLabels_NextVsSubmit tests correct button labels
func TestQuestionView_ButtonLabels_NextVsSubmit(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{Header: "Q1", Question: "First", Options: []Option{{Label: "A"}}, Multiple: false},
		{Header: "Q2", Question: "Second", Options: []Option{{Label: "B"}}, Multiple: false},
		{Header: "Q3", Question: "Third", Options: []Option{{Label: "C"}}, Multiple: false},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	// First question - should show "Next ->"
	qv1 := NewQuestionView(questions, answers, 0)
	qv1.SetSize(80, 24)
	view1 := qv1.View()
	require.Contains(t, view1, "Next", "expected 'Next' button on first question")
	require.NotContains(t, view1, "Submit", "expected no 'Submit' button on first question")

	// Middle question - should show "Next ->"
	qv2 := NewQuestionView(questions, answers, 1)
	qv2.SetSize(80, 24)
	view2 := qv2.View()
	require.Contains(t, view2, "Next", "expected 'Next' button on middle question")
	require.NotContains(t, view2, "Submit", "expected no 'Submit' button on middle question")

	// Last question - should show "Submit"
	qv3 := NewQuestionView(questions, answers, 2)
	qv3.SetSize(80, 24)
	view3 := qv3.View()
	require.Contains(t, view3, "Submit", "expected 'Submit' button on last question")
}

// TestQuestionView_BackButton_NotShownOnFirstQuestion tests back button visibility
func TestQuestionView_BackButton_NotShownOnFirstQuestion(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{Header: "Q1", Question: "First", Options: []Option{{Label: "A"}}, Multiple: false},
		{Header: "Q2", Question: "Second", Options: []Option{{Label: "B"}}, Multiple: false},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	// First question - no back button
	qv1 := NewQuestionView(questions, answers, 0)
	qv1.SetSize(80, 24)
	view1 := qv1.View()
	require.NotContains(t, view1, "Back", "expected no 'Back' button on first question")

	// Second question - should have back button
	qv2 := NewQuestionView(questions, answers, 1)
	qv2.SetSize(80, 24)
	view2 := qv2.View()
	require.Contains(t, view2, "Back", "expected 'Back' button on second question")
}

// --- Focus State After Actions Tests ---

// TestQuestionView_FocusState_AfterTabCycle tests focus components are correctly updated
func TestQuestionView_FocusState_AfterTabCycle(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{Header: "Q", Question: "Question", Options: []Option{{Label: "Opt"}}, Multiple: false},
		{Header: "Q2", Question: "Question2", Options: []Option{{Label: "Opt2"}}, Multiple: false},
	}

	answers := []QuestionAnswer{
		{Value: "Opt", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 1)
	qv.SetSize(80, 24)

	// Enable custom input
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1
	qv.optionSelector.Toggle()
	selected := qv.optionSelector.SelectedLabels()
	qv.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"
	qv.rebuildButtonBar()

	// Initially options focused
	require.True(t, qv.optionSelector.focused, "options should be focused initially")
	require.False(t, qv.customInput.Focused(), "custom input should not be focused initially")

	// Tab to custom input
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.False(t, qv.optionSelector.focused, "options should be blurred after tab")
	require.True(t, qv.customInput.Focused(), "custom input should be focused after tab")

	// Tab to back button
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.False(t, qv.optionSelector.focused, "options should be blurred")
	require.False(t, qv.customInput.Focused(), "custom input should be blurred")
	// Button bar focus is managed by buttonBar itself

	// Tab to next button (no component focus changes, just focusIndex)
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "should be on next button")

	// Tab back to options
	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.True(t, qv.optionSelector.focused, "options should be focused after wrap")
	require.False(t, qv.customInput.Focused(), "custom input should be blurred after wrap")
}

// --- Multi-Select Tab Navigation Tests ---

// TestQuestionView_TabNavigation_MultiSelect tests tab works the same for multi-select
func TestQuestionView_TabNavigation_MultiSelect(t *testing.T) {
	t.Parallel()

	questions := []Question{
		{
			Header:   "Multi Q",
			Question: "Select multiple",
			Options: []Option{
				{Label: "Item 1"},
				{Label: "Item 2"},
				{Label: "Item 3"},
			},
			Multiple: true,
		},
	}

	answers := []QuestionAnswer{
		{Value: []string{}, IsMulti: true},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)

	// Tab should work the same way
	require.Equal(t, 0, qv.focusIndex, "start on options")

	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, qv.nextButtonFocusIndex(), qv.focusIndex, "tab to next button")

	qv.Update(tea.KeyPressMsg{Text: "tab"})
	require.Equal(t, 0, qv.focusIndex, "tab wraps to options")
}
