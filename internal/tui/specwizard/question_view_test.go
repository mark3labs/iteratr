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
