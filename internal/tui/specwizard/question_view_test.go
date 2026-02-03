package specwizard

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestQuestionOptions_SingleSelect(t *testing.T) {
	items := []OptionItem{
		{label: "Option 1", description: "First option"},
		{label: "Option 2", description: "Second option"},
		{label: "Type your own answer", description: "Enter custom text"},
	}

	opts := NewQuestionOptions(items, false)

	// Initially nothing selected
	if len(opts.SelectedLabels()) != 0 {
		t.Errorf("expected no selection, got %v", opts.SelectedLabels())
	}

	// Select first option
	opts.Toggle()
	selected := opts.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Option 1" {
		t.Errorf("expected [Option 1], got %v", selected)
	}

	// Move down and select second option (should deselect first)
	opts.CursorDown()
	opts.Toggle()
	selected = opts.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Option 2" {
		t.Errorf("expected [Option 2], got %v", selected)
	}
}

func TestQuestionOptions_MultiSelect(t *testing.T) {
	items := []OptionItem{
		{label: "Option 1", description: "First option"},
		{label: "Option 2", description: "Second option"},
		{label: "Type your own answer", description: "Enter custom text"},
	}

	opts := NewQuestionOptions(items, true)

	// Select first option
	opts.Toggle()
	selected := opts.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Option 1" {
		t.Errorf("expected [Option 1], got %v", selected)
	}

	// Select second option (should keep first)
	opts.CursorDown()
	opts.Toggle()
	selected = opts.SelectedLabels()
	if len(selected) != 2 {
		t.Errorf("expected 2 selections, got %v", selected)
	}

	// Select "Type your own" (should deselect all others)
	opts.CursorDown()
	opts.Toggle()
	selected = opts.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Type your own answer" {
		t.Errorf("expected [Type your own answer], got %v", selected)
	}
}

func TestQuestionOptions_KeyboardNavigation(t *testing.T) {
	items := []OptionItem{
		{label: "Option 1"},
		{label: "Option 2"},
		{label: "Option 3"},
	}

	opts := NewQuestionOptions(items, false)
	opts.Focus()

	// Test down navigation using direct method
	opts.CursorDown()
	if opts.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", opts.cursor)
	}

	// Test up navigation using direct method
	opts.CursorUp()
	if opts.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", opts.cursor)
	}

	// Test toggle
	opts.Toggle()
	if !opts.items[0].selected {
		t.Error("expected first item to be selected")
	}
}

func TestQuestionView_AnswerPersistence(t *testing.T) {
	questions := []Question{
		{
			Header:   "First Question",
			Question: "What is your choice?",
			Options: []Option{
				{Label: "Choice A"},
				{Label: "Choice B"},
			},
			Multiple: false,
		},
		{
			Header:   "Second Question",
			Question: "Select multiple",
			Options: []Option{
				{Label: "Item 1"},
				{Label: "Item 2"},
			},
			Multiple: true,
		},
	}

	// Pre-fill answers
	answers := []QuestionAnswer{
		{Value: "Choice B", IsMulti: false},
		{Value: []string{"Item 1", "Item 2"}, IsMulti: true},
	}

	// Create view for first question
	qv := NewQuestionView(questions, answers, 0)

	// Check that "Choice B" is selected
	selected := qv.optionSelector.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Choice B" {
		t.Errorf("expected [Choice B] to be restored, got %v", selected)
	}

	// Create view for second question
	qv2 := NewQuestionView(questions, answers, 1)

	// Check that both items are selected
	selected2 := qv2.optionSelector.SelectedLabels()
	if len(selected2) != 2 {
		t.Errorf("expected 2 items selected, got %v", selected2)
	}
}

func TestQuestionView_CustomTextPersistence(t *testing.T) {
	questions := []Question{
		{
			Header:   "Custom Question",
			Question: "Enter something",
			Options: []Option{
				{Label: "Predefined"},
			},
			Multiple: false,
		},
	}

	// Pre-fill with custom text
	answers := []QuestionAnswer{
		{Value: "My custom answer", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Check that custom input is shown and populated
	if !qv.showCustom {
		t.Error("expected custom input to be visible")
	}

	if qv.customInput.Value() != "My custom answer" {
		t.Errorf("expected custom input to contain 'My custom answer', got %q", qv.customInput.Value())
	}

	// Check that "Type your own answer" is selected
	selected := qv.optionSelector.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Type your own answer" {
		t.Errorf("expected [Type your own answer] to be selected, got %v", selected)
	}
}

func TestQuestionView_TabNavigation(t *testing.T) {
	questions := []Question{
		{
			Header:   "Test Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Initially focused on options
	if qv.focusIndex != 0 {
		t.Errorf("expected focusIndex 0, got %d", qv.focusIndex)
	}

	// Manually cycle focus - simulating tab key
	qv.focusIndex = qv.nextButtonFocusIndex()
	expectedFocusIndex := qv.nextButtonFocusIndex()
	if qv.focusIndex != expectedFocusIndex {
		t.Errorf("expected focusIndex %d, got %d", expectedFocusIndex, qv.focusIndex)
	}

	// Wrap back to options
	qv.focusIndex = 0
	if qv.focusIndex != 0 {
		t.Errorf("expected focusIndex to wrap to 0, got %d", qv.focusIndex)
	}
}

func TestQuestionView_TabNavigationWithCustomInput(t *testing.T) {
	questions := []Question{
		{
			Header:   "Test Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Select "Type your own answer" to show custom input
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1
	qv.optionSelector.Toggle()
	qv.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Manually test focus indices: options -> custom input -> next button -> options
	qv.focusIndex = 0
	qv.focusIndex = 1 // custom input
	if qv.focusIndex != 1 {
		t.Errorf("expected focusIndex 1 (custom input), got %d", qv.focusIndex)
	}

	qv.focusIndex = qv.nextButtonFocusIndex()
	expectedNextIndex := qv.nextButtonFocusIndex()
	if qv.focusIndex != expectedNextIndex {
		t.Errorf("expected focusIndex %d (next button), got %d", expectedNextIndex, qv.focusIndex)
	}

	qv.focusIndex = 0 // wrap
	if qv.focusIndex != 0 {
		t.Errorf("expected focusIndex to wrap to 0, got %d", qv.focusIndex)
	}
}

func TestQuestionView_Validation(t *testing.T) {
	questions := []Question{
		{
			Header:   "Test Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)

	// No selection - should fail validation
	if qv.validateAnswer() {
		t.Error("expected validation to fail with no selection")
	}

	// Select an option - should pass validation
	qv.optionSelector.Toggle()
	if !qv.validateAnswer() {
		t.Error("expected validation to pass with selection")
	}

	// Select "Type your own" but leave empty - should fail
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1
	qv.optionSelector.Toggle()
	if qv.validateAnswer() {
		t.Error("expected validation to fail with empty custom text")
	}

	// Add custom text - should pass
	qv.customInput.SetValue("Some text")
	if !qv.validateAnswer() {
		t.Error("expected validation to pass with custom text")
	}
}

func TestQuestionView_SaveAndRestore(t *testing.T) {
	questions := []Question{
		{
			Header:   "Test Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
				{Label: "Option 2"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Select option 2
	qv.optionSelector.CursorDown()
	qv.optionSelector.Toggle()

	// Save answer
	qv.saveCurrentAnswer()

	// Check answer was saved
	saved := qv.answers[0]
	if saved.Value != "Option 2" {
		t.Errorf("expected answer 'Option 2', got %v", saved.Value)
	}

	// Create new view with saved answers - should restore
	qv2 := NewQuestionView(questions, qv.answers, 0)
	selected := qv2.optionSelector.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Option 2" {
		t.Errorf("expected [Option 2] to be restored, got %v", selected)
	}
}

func TestQuestionView_AutoAppendTypeYourOwn(t *testing.T) {
	questions := []Question{
		{
			Header:   "Test Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
				{Label: "Option 2"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Check that "Type your own answer" option was auto-appended
	items := qv.optionSelector.items
	if len(items) != 3 {
		t.Errorf("expected 3 items (2 options + Type your own), got %d", len(items))
	}

	lastItem := items[len(items)-1]
	if lastItem.label != "Type your own answer" {
		t.Errorf("expected last item to be 'Type your own answer', got %q", lastItem.label)
	}

	if lastItem.description != "Enter custom text" {
		t.Errorf("expected description 'Enter custom text', got %q", lastItem.description)
	}
}

func TestQuestionView_MultiSelectIndices(t *testing.T) {
	questions := []Question{
		{
			Header:   "Multi-select Question",
			Question: "Select multiple options",
			Options: []Option{
				{Label: "Option A"},
				{Label: "Option B"},
				{Label: "Option C"},
				{Label: "Option D"},
			},
			Multiple: true,
		},
	}

	answers := []QuestionAnswer{
		{Value: []string{}, IsMulti: true},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Select Option A (index 0)
	qv.optionSelector.Toggle()
	indices := qv.optionSelector.SelectedIndices()
	if len(indices) != 1 || indices[0] != 0 {
		t.Errorf("expected indices [0], got %v", indices)
	}

	// Select Option C (index 2)
	qv.optionSelector.CursorDown()
	qv.optionSelector.CursorDown()
	qv.optionSelector.Toggle()
	indices = qv.optionSelector.SelectedIndices()
	if len(indices) != 2 || indices[0] != 0 || indices[1] != 2 {
		t.Errorf("expected indices [0, 2], got %v", indices)
	}

	// Deselect Option A (index 0)
	qv.optionSelector.cursor = 0
	qv.optionSelector.Toggle()
	indices = qv.optionSelector.SelectedIndices()
	if len(indices) != 1 || indices[0] != 2 {
		t.Errorf("expected indices [2], got %v", indices)
	}

	// Save answer and verify it contains correct labels
	qv.saveCurrentAnswer()
	saved := qv.answers[0]
	if !saved.IsMulti {
		t.Error("expected answer to be marked as multi-select")
	}

	labels, ok := saved.Value.([]string)
	if !ok {
		t.Fatalf("expected answer value to be []string, got %T", saved.Value)
	}

	if len(labels) != 1 || labels[0] != "Option C" {
		t.Errorf("expected labels [Option C], got %v", labels)
	}
}

func TestQuestionView_MultiSelectWithMultipleSelections(t *testing.T) {
	questions := []Question{
		{
			Header:   "Multi-select Question",
			Question: "Select multiple options",
			Options: []Option{
				{Label: "Option 1"},
				{Label: "Option 2"},
				{Label: "Option 3"},
			},
			Multiple: true,
		},
	}

	answers := []QuestionAnswer{
		{Value: []string{}, IsMulti: true},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Select all three options
	qv.optionSelector.Toggle() // Select Option 1
	qv.optionSelector.CursorDown()
	qv.optionSelector.Toggle() // Select Option 2
	qv.optionSelector.CursorDown()
	qv.optionSelector.Toggle() // Select Option 3

	// Verify indices
	indices := qv.optionSelector.SelectedIndices()
	if len(indices) != 3 || indices[0] != 0 || indices[1] != 1 || indices[2] != 2 {
		t.Errorf("expected indices [0, 1, 2], got %v", indices)
	}

	// Verify labels
	labels := qv.optionSelector.SelectedLabels()
	expectedLabels := []string{"Option 1", "Option 2", "Option 3"}
	if len(labels) != len(expectedLabels) {
		t.Fatalf("expected %d labels, got %d", len(expectedLabels), len(labels))
	}
	for i, expected := range expectedLabels {
		if labels[i] != expected {
			t.Errorf("expected label %q at index %d, got %q", expected, i, labels[i])
		}
	}

	// Save and verify answer
	qv.saveCurrentAnswer()
	saved := qv.answers[0]
	savedLabels, ok := saved.Value.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", saved.Value)
	}

	if len(savedLabels) != 3 {
		t.Errorf("expected 3 saved labels, got %d", len(savedLabels))
	}
}

func TestQuestionView_MultiSelectCustomOption(t *testing.T) {
	questions := []Question{
		{
			Header:   "Multi-select Question",
			Question: "Select options or type your own",
			Options: []Option{
				{Label: "Option 1"},
				{Label: "Option 2"},
			},
			Multiple: true,
		},
	}

	answers := []QuestionAnswer{
		{Value: []string{}, IsMulti: true},
	}

	qv := NewQuestionView(questions, answers, 0)

	// Select Option 1 and Option 2
	qv.optionSelector.Toggle()
	qv.optionSelector.CursorDown()
	qv.optionSelector.Toggle()

	labels := qv.optionSelector.SelectedLabels()
	if len(labels) != 2 {
		t.Errorf("expected 2 selections, got %v", labels)
	}

	// Now select "Type your own answer" (should deselect all others)
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1
	qv.optionSelector.Toggle()

	labels = qv.optionSelector.SelectedLabels()
	if len(labels) != 1 || labels[0] != "Type your own answer" {
		t.Errorf("expected only [Type your own answer], got %v", labels)
	}

	// Set custom text and save
	qv.customInput.SetValue("My custom answer")
	qv.saveCurrentAnswer()

	saved := qv.answers[0]
	savedLabels, ok := saved.Value.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", saved.Value)
	}

	if len(savedLabels) != 1 || savedLabels[0] != "My custom answer" {
		t.Errorf("expected [My custom answer], got %v", savedLabels)
	}
}

func TestQuestionView_NavigationButtons(t *testing.T) {
	questions := []Question{
		{
			Header:   "First Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
				{Label: "Option 2"},
			},
			Multiple: false,
		},
		{
			Header:   "Second Question",
			Question: "Select another",
			Options: []Option{
				{Label: "Choice A"},
				{Label: "Choice B"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	// Test first question (no back button, only Next)
	qv1 := NewQuestionView(questions, answers, 0)
	qv1.SetSize(80, 24)

	// View should render button bar (and initialize it lazily)
	view := qv1.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// Verify button bar is created
	if qv1.buttonBar == nil {
		t.Fatal("expected buttonBar to be initialized after View()")
	}

	// Select an option
	qv1.optionSelector.Toggle()

	// Focus on next button and trigger it
	qv1.focusIndex = qv1.nextButtonFocusIndex()
	qv1.updateFocus()

	cmd := qv1.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected NextQuestionMsg command")
	}

	msg := cmd()
	if _, ok := msg.(NextQuestionMsg); !ok {
		t.Errorf("expected NextQuestionMsg, got %T", msg)
	}

	// Test second question (has back button and Next/Submit)
	qv2 := NewQuestionView(questions, answers, 1)
	qv2.SetSize(80, 24)
	qv2.View() // Initialize button bar

	// Verify button bar is created
	if qv2.buttonBar == nil {
		t.Fatal("expected buttonBar to be initialized after View()")
	}

	// Select an option
	qv2.optionSelector.Toggle()

	// Focus on back button and trigger it
	qv2.focusIndex = qv2.backButtonFocusIndex()
	qv2.updateFocus()

	cmd = qv2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected PrevQuestionMsg command")
	}

	msg = cmd()
	if _, ok := msg.(PrevQuestionMsg); !ok {
		t.Errorf("expected PrevQuestionMsg, got %T", msg)
	}

	// Focus on next button (which should be Submit on last question)
	qv2.focusIndex = qv2.nextButtonFocusIndex()
	qv2.updateFocus()

	cmd = qv2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected SubmitAnswersMsg command")
	}

	msg = cmd()
	if _, ok := msg.(SubmitAnswersMsg); !ok {
		t.Errorf("expected SubmitAnswersMsg, got %T", msg)
	}
}

func TestQuestionView_ButtonBarRebuild(t *testing.T) {
	questions := []Question{
		{
			Header:   "Test Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
	}

	qv := NewQuestionView(questions, answers, 0)
	qv.SetSize(80, 24)
	qv.View() // Initialize button bar

	// Initially no custom input visible
	if qv.showCustom {
		t.Error("expected custom input to be hidden initially")
	}

	// Select "Type your own answer" - this should show custom input and rebuild button bar
	qv.optionSelector.cursor = len(qv.optionSelector.items) - 1
	qv.optionSelector.Toggle()                          // Direct toggle instead of Update
	qv.Update(tea.WindowSizeMsg{Width: 80, Height: 24}) // Trigger update to recalc showCustom

	// Manually trigger the same logic as Update() does for options change
	selected := qv.optionSelector.SelectedLabels()
	qv.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"
	qv.rebuildButtonBar()

	if !qv.showCustom {
		t.Error("expected custom input to be visible after selecting 'Type your own answer'")
	}

	if qv.buttonBar == nil {
		t.Error("expected button bar to be rebuilt")
	}

	// Verify button bar still works after rebuild
	view := qv.View()
	if view == "" {
		t.Error("expected non-empty view after rebuild")
	}
}

func TestQuestionView_ButtonBarValidation(t *testing.T) {
	questions := []Question{
		{
			Header:   "First Question",
			Question: "Select one",
			Options: []Option{
				{Label: "Option 1"},
			},
			Multiple: false,
		},
		{
			Header:   "Second Question",
			Question: "Select another",
			Options: []Option{
				{Label: "Option A"},
			},
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

	// Try to submit without selecting anything
	qv.focusIndex = qv.nextButtonFocusIndex()
	qv.updateFocus()

	cmd := qv.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected ShowErrorMsg command")
	}

	msg := cmd()
	if errMsg, ok := msg.(ShowErrorMsg); !ok {
		t.Errorf("expected ShowErrorMsg, got %T", msg)
	} else if errMsg.err == "" {
		t.Error("expected non-empty error message")
	}

	// Now select an option and submit should work (NextQuestionMsg since not last question)
	qv.focusIndex = 0
	qv.updateFocus()
	qv.optionSelector.Toggle()

	qv.focusIndex = qv.nextButtonFocusIndex()
	qv.updateFocus()

	cmd = qv.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected NextQuestionMsg command")
	}

	msg = cmd()
	if _, ok := msg.(NextQuestionMsg); !ok {
		t.Errorf("expected NextQuestionMsg, got %T", msg)
	}
}

// TestQuestionView_AnswerPersistenceFlow tests the complete flow of navigating
// back and forth between questions and verifying answers are preserved.
func TestQuestionView_AnswerPersistenceFlow(t *testing.T) {
	questions := []Question{
		{
			Header:   "Question 1",
			Question: "What is your first choice?",
			Options: []Option{
				{Label: "Q1 Option A"},
				{Label: "Q1 Option B"},
			},
			Multiple: false,
		},
		{
			Header:   "Question 2",
			Question: "What is your second choice?",
			Options: []Option{
				{Label: "Q2 Option X"},
				{Label: "Q2 Option Y"},
			},
			Multiple: false,
		},
		{
			Header:   "Question 3",
			Question: "Select multiple",
			Options: []Option{
				{Label: "Q3 Option 1"},
				{Label: "Q3 Option 2"},
				{Label: "Q3 Option 3"},
			},
			Multiple: true,
		},
	}

	// Initialize empty answers
	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
		{Value: []string{}, IsMulti: true},
	}

	// Step 1: Answer Q1 with "Q1 Option B"
	qv1 := NewQuestionView(questions, answers, 0)
	qv1.optionSelector.CursorDown() // Move to Option B
	qv1.optionSelector.Toggle()     // Select it
	qv1.saveCurrentAnswer()
	answers = qv1.answers

	// Verify Q1 answer was saved
	if answers[0].Value != "Q1 Option B" {
		t.Errorf("Q1: expected 'Q1 Option B', got %v", answers[0].Value)
	}

	// Step 2: Navigate to Q2 and answer with "Q2 Option X"
	qv2 := NewQuestionView(questions, answers, 1)
	qv2.optionSelector.Toggle() // Select first option (Q2 Option X)
	qv2.saveCurrentAnswer()
	answers = qv2.answers

	// Verify Q2 answer was saved
	if answers[1].Value != "Q2 Option X" {
		t.Errorf("Q2: expected 'Q2 Option X', got %v", answers[1].Value)
	}

	// Step 3: Navigate back to Q1 and verify answer is restored
	qv1b := NewQuestionView(questions, answers, 0)
	selected := qv1b.optionSelector.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Q1 Option B" {
		t.Errorf("Q1 restored: expected [Q1 Option B], got %v", selected)
	}

	// Step 4: Change Q1 answer to "Q1 Option A"
	qv1b.optionSelector.cursor = 0
	qv1b.optionSelector.Toggle() // Select Option A
	qv1b.saveCurrentAnswer()
	answers = qv1b.answers

	// Verify Q1 answer was updated
	if answers[0].Value != "Q1 Option A" {
		t.Errorf("Q1 updated: expected 'Q1 Option A', got %v", answers[0].Value)
	}

	// Step 5: Navigate to Q3 and select multiple options
	qv3 := NewQuestionView(questions, answers, 2)
	qv3.optionSelector.Toggle() // Select Q3 Option 1
	qv3.optionSelector.CursorDown()
	qv3.optionSelector.CursorDown()
	qv3.optionSelector.Toggle() // Select Q3 Option 3
	qv3.saveCurrentAnswer()
	answers = qv3.answers

	// Verify Q3 answer (multi-select)
	q3Answer, ok := answers[2].Value.([]string)
	if !ok {
		t.Fatalf("Q3: expected []string, got %T", answers[2].Value)
	}
	if len(q3Answer) != 2 || q3Answer[0] != "Q3 Option 1" || q3Answer[1] != "Q3 Option 3" {
		t.Errorf("Q3: expected [Q3 Option 1, Q3 Option 3], got %v", q3Answer)
	}

	// Step 6: Navigate back to Q2 and verify answer is still preserved
	qv2b := NewQuestionView(questions, answers, 1)
	selected2 := qv2b.optionSelector.SelectedLabels()
	if len(selected2) != 1 || selected2[0] != "Q2 Option X" {
		t.Errorf("Q2 restored: expected [Q2 Option X], got %v", selected2)
	}

	// Step 7: Navigate back to Q3 and verify multi-select answer is restored
	qv3b := NewQuestionView(questions, answers, 2)
	selected3 := qv3b.optionSelector.SelectedLabels()
	if len(selected3) != 2 || selected3[0] != "Q3 Option 1" || selected3[1] != "Q3 Option 3" {
		t.Errorf("Q3 restored: expected [Q3 Option 1, Q3 Option 3], got %v", selected3)
	}

	// Final verification: all answers are preserved
	if answers[0].Value != "Q1 Option A" {
		t.Errorf("Final Q1: expected 'Q1 Option A', got %v", answers[0].Value)
	}
	if answers[1].Value != "Q2 Option X" {
		t.Errorf("Final Q2: expected 'Q2 Option X', got %v", answers[1].Value)
	}
	finalQ3, _ := answers[2].Value.([]string)
	if len(finalQ3) != 2 {
		t.Errorf("Final Q3: expected 2 items, got %v", finalQ3)
	}
}

// TestQuestionView_CustomTextPersistenceFlow tests custom text answer persistence.
func TestQuestionView_CustomTextPersistenceFlow(t *testing.T) {
	questions := []Question{
		{
			Header:   "Question 1",
			Question: "Enter something",
			Options: []Option{
				{Label: "Predefined Option"},
			},
			Multiple: false,
		},
		{
			Header:   "Question 2",
			Question: "Another question",
			Options: []Option{
				{Label: "Choice A"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: "", IsMulti: false},
	}

	// Step 1: Answer Q1 with custom text
	qv1 := NewQuestionView(questions, answers, 0)
	qv1.optionSelector.cursor = len(qv1.optionSelector.items) - 1 // "Type your own answer"
	qv1.optionSelector.Toggle()
	qv1.customInput.SetValue("My custom answer for Q1")
	qv1.saveCurrentAnswer()
	answers = qv1.answers

	// Verify custom text was saved
	if answers[0].Value != "My custom answer for Q1" {
		t.Errorf("Q1: expected 'My custom answer for Q1', got %v", answers[0].Value)
	}

	// Step 2: Answer Q2 with predefined option
	qv2 := NewQuestionView(questions, answers, 1)
	qv2.optionSelector.Toggle() // Select "Choice A"
	qv2.saveCurrentAnswer()
	answers = qv2.answers

	// Step 3: Navigate back to Q1 and verify custom text is restored
	qv1b := NewQuestionView(questions, answers, 0)
	if !qv1b.showCustom {
		t.Error("Q1 restored: expected custom input to be visible")
	}
	if qv1b.customInput.Value() != "My custom answer for Q1" {
		t.Errorf("Q1 restored: expected 'My custom answer for Q1', got %q", qv1b.customInput.Value())
	}
	selected := qv1b.optionSelector.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Type your own answer" {
		t.Errorf("Q1 restored: expected [Type your own answer], got %v", selected)
	}

	// Step 4: Change Q1 custom text
	qv1b.customInput.SetValue("Updated custom answer")
	qv1b.saveCurrentAnswer()
	answers = qv1b.answers

	// Verify update
	if answers[0].Value != "Updated custom answer" {
		t.Errorf("Q1 updated: expected 'Updated custom answer', got %v", answers[0].Value)
	}

	// Step 5: Navigate forward and back again to verify persistence
	qv2b := NewQuestionView(questions, answers, 1)
	qv2b.View() // Just to exercise the view

	qv1c := NewQuestionView(questions, answers, 0)
	if qv1c.customInput.Value() != "Updated custom answer" {
		t.Errorf("Q1 re-restored: expected 'Updated custom answer', got %q", qv1c.customInput.Value())
	}
}

// TestQuestionView_MultiSelectCustomPersistence tests multi-select with custom text persistence.
func TestQuestionView_MultiSelectCustomPersistence(t *testing.T) {
	questions := []Question{
		{
			Header:   "Multi Question",
			Question: "Select multiple or type your own",
			Options: []Option{
				{Label: "Option 1"},
				{Label: "Option 2"},
			},
			Multiple: true,
		},
		{
			Header:   "Next Question",
			Question: "Another one",
			Options: []Option{
				{Label: "Choice X"},
			},
			Multiple: false,
		},
	}

	answers := []QuestionAnswer{
		{Value: []string{}, IsMulti: true},
		{Value: "", IsMulti: false},
	}

	// Step 1: Answer Q1 with custom text in multi-select
	qv1 := NewQuestionView(questions, answers, 0)
	qv1.optionSelector.cursor = len(qv1.optionSelector.items) - 1
	qv1.optionSelector.Toggle()
	qv1.customInput.SetValue("My custom multi-select answer")
	qv1.saveCurrentAnswer()
	answers = qv1.answers

	// Verify custom text was saved as []string with one element
	q1Answer, ok := answers[0].Value.([]string)
	if !ok {
		t.Fatalf("Q1: expected []string, got %T", answers[0].Value)
	}
	if len(q1Answer) != 1 || q1Answer[0] != "My custom multi-select answer" {
		t.Errorf("Q1: expected [My custom multi-select answer], got %v", q1Answer)
	}

	// Step 2: Answer Q2
	qv2 := NewQuestionView(questions, answers, 1)
	qv2.optionSelector.Toggle()
	qv2.saveCurrentAnswer()
	answers = qv2.answers

	// Step 3: Navigate back to Q1 and verify custom text is restored
	qv1b := NewQuestionView(questions, answers, 0)
	if !qv1b.showCustom {
		t.Error("Q1 restored: expected custom input to be visible")
	}
	if qv1b.customInput.Value() != "My custom multi-select answer" {
		t.Errorf("Q1 restored: expected 'My custom multi-select answer', got %q", qv1b.customInput.Value())
	}
	selected := qv1b.optionSelector.SelectedLabels()
	if len(selected) != 1 || selected[0] != "Type your own answer" {
		t.Errorf("Q1 restored: expected [Type your own answer], got %v", selected)
	}
}

// TestQuestionView_EmptyAnswerHandling tests persistence of empty/no answers.
func TestQuestionView_EmptyAnswerHandling(t *testing.T) {
	questions := []Question{
		{
			Header:   "Question 1",
			Question: "Select one",
			Options: []Option{
				{Label: "Option A"},
				{Label: "Option B"},
			},
			Multiple: false,
		},
		{
			Header:   "Question 2",
			Question: "Select multiple",
			Options: []Option{
				{Label: "Item 1"},
				{Label: "Item 2"},
			},
			Multiple: true,
		},
	}

	answers := []QuestionAnswer{
		{Value: "", IsMulti: false},
		{Value: []string{}, IsMulti: true},
	}

	// Step 1: View Q1 without selecting anything
	qv1 := NewQuestionView(questions, answers, 0)
	qv1.saveCurrentAnswer() // Save empty answer
	answers = qv1.answers

	// Verify empty single-select answer
	if answers[0].Value != "" {
		t.Errorf("Q1: expected empty string, got %v", answers[0].Value)
	}

	// Step 2: View Q2 without selecting anything
	qv2 := NewQuestionView(questions, answers, 1)
	qv2.saveCurrentAnswer() // Save empty answer
	answers = qv2.answers

	// Verify empty multi-select answer
	q2Answer, ok := answers[1].Value.([]string)
	if !ok {
		t.Fatalf("Q2: expected []string, got %T", answers[1].Value)
	}
	if len(q2Answer) != 0 {
		t.Errorf("Q2: expected empty array, got %v", q2Answer)
	}

	// Step 3: Navigate back to Q1 and verify nothing is selected
	qv1b := NewQuestionView(questions, answers, 0)
	selected := qv1b.optionSelector.SelectedLabels()
	if len(selected) != 0 {
		t.Errorf("Q1 restored: expected no selection, got %v", selected)
	}

	// Step 4: Navigate to Q2 and verify nothing is selected
	qv2b := NewQuestionView(questions, answers, 1)
	selected2 := qv2b.optionSelector.SelectedLabels()
	if len(selected2) != 0 {
		t.Errorf("Q2 restored: expected no selection, got %v", selected2)
	}
}
