package specwizard

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/specmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentPhase_Init(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	assert.True(t, phase.waitingForAgent, "should start in waiting state")
	assert.Equal(t, "Agent is analyzing requirements...", phase.statusText)
	assert.Nil(t, phase.questionView)

	// Init should return commands (spinner tick + wait for questions)
	cmd := phase.Init()
	assert.NotNil(t, cmd)
}

func TestAgentPhase_ReceiveQuestions(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Create a question request
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "What is your favorite color?",
				Header:   "Color Preference",
				Options: []specmcp.Option{
					{Label: "Red", Description: "The color red"},
					{Label: "Blue", Description: "The color blue"},
				},
				Multiple: false,
			},
			{
				Question: "What features do you want?",
				Header:   "Features",
				Options: []specmcp.Option{
					{Label: "Feature A", Description: "First feature"},
					{Label: "Feature B", Description: "Second feature"},
				},
				Multiple: true,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	// Send question request message
	msg := QuestionRequestMsg{Request: req}
	phase, cmd := phase.Update(msg)

	assert.False(t, phase.waitingForAgent, "should no longer be waiting")
	assert.Len(t, phase.questions, 2, "should have 2 questions")
	assert.Len(t, phase.answers, 2, "should have 2 answer slots")
	assert.Equal(t, 0, phase.currentIndex, "should start at first question")
	assert.NotNil(t, phase.questionView, "should create question view")
	assert.Nil(t, cmd)

	// Verify question conversion
	assert.Equal(t, "What is your favorite color?", phase.questions[0].Question)
	assert.Equal(t, "Color Preference", phase.questions[0].Header)
	assert.Len(t, phase.questions[0].Options, 2)
	assert.False(t, phase.questions[0].Multiple)

	assert.Equal(t, "What features do you want?", phase.questions[1].Question)
	assert.True(t, phase.questions[1].Multiple)

	// Verify answers initialized
	assert.Equal(t, "", phase.answers[0].Value)
	assert.False(t, phase.answers[0].IsMulti)
	assert.Equal(t, []string{}, phase.answers[1].Value)
	assert.True(t, phase.answers[1].IsMulti)
}

func TestAgentPhase_NextQuestion(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with 2 questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Option 1", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Question 2?",
				Header:   "Q2",
				Options:  []specmcp.Option{{Label: "Option 2", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})
	assert.Equal(t, 0, phase.currentIndex)

	// Answer first question
	phase.questionView.optionSelector.items[0].selected = true

	// Navigate to next question
	phase, cmd := phase.Update(NextQuestionMsg{})
	assert.Equal(t, 1, phase.currentIndex, "should advance to next question")
	assert.NotNil(t, phase.questionView, "should still have question view")
	assert.Nil(t, cmd)

	// Verify answer was saved
	assert.Equal(t, "Option 1", phase.answers[0].Value)
	assert.False(t, phase.answers[0].IsMulti)
}

func TestAgentPhase_NextQuestion_ValidationFails(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with 2 questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Option 1", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Question 2?",
				Header:   "Q2",
				Options:  []specmcp.Option{{Label: "Option 2", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})
	assert.Equal(t, 0, phase.currentIndex)

	// Don't answer first question - leave it empty

	// Try to navigate to next question
	phase, cmd := phase.Update(NextQuestionMsg{})

	// Should stay on same question
	assert.Equal(t, 0, phase.currentIndex, "should stay on current question")
	assert.NotNil(t, phase.questionView, "should still have question view")

	// Should return error command
	assert.NotNil(t, cmd, "should return error command")

	// Execute command and verify it returns ShowErrorMsg
	msg := cmd()
	errorMsg, ok := msg.(ShowErrorMsg)
	assert.True(t, ok, "command should return ShowErrorMsg")
	assert.Contains(t, errorMsg.err, "Please select an answer", "error should mention selection")
}

func TestAgentPhase_NextQuestion_CustomTextEmpty(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with 1 question
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Option 1", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Select "Type your own answer" but leave custom text empty
	lastIdx := len(phase.questionView.optionSelector.items) - 1
	phase.questionView.optionSelector.items[lastIdx].selected = true
	phase.questionView.showCustom = true
	phase.questionView.customInput.SetValue("") // Empty custom text

	// Try to navigate to next question
	phase, cmd := phase.Update(NextQuestionMsg{})

	// Should stay on same question
	assert.Equal(t, 0, phase.currentIndex, "should stay on current question")

	// Should return error command
	assert.NotNil(t, cmd, "should return error command")

	// Execute command and verify it returns ShowErrorMsg
	msg := cmd()
	errorMsg, ok := msg.(ShowErrorMsg)
	assert.True(t, ok, "command should return ShowErrorMsg")
	assert.Contains(t, errorMsg.err, "Please select an answer", "error should mention missing text")
}

func TestAgentPhase_NextQuestion_AtLastQuestion(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with 1 question (already at last)
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Option 1", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})
	assert.Equal(t, 0, phase.currentIndex)

	// Answer the question
	phase.questionView.optionSelector.items[0].selected = true

	// Try to navigate to next question (should stay at last)
	phase, cmd := phase.Update(NextQuestionMsg{})

	// Should stay at index 0 (last question)
	assert.Equal(t, 0, phase.currentIndex, "should stay at last question")
	assert.NotNil(t, phase.questionView, "should still have question view")
	assert.Nil(t, cmd)

	// Verify answer was still saved
	assert.Equal(t, "Option 1", phase.answers[0].Value)
}

func TestAgentPhase_PrevQuestion(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with 2 questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Option 1", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Question 2?",
				Header:   "Q2",
				Options:  []specmcp.Option{{Label: "Option 2", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question with "Option 1"
	phase.questionView.optionSelector.items[0].selected = true
	phase, _ = phase.Update(NextQuestionMsg{})
	assert.Equal(t, 1, phase.currentIndex)

	// Verify first answer was saved
	assert.Equal(t, "Option 1", phase.answers[0].Value, "first answer should be saved")

	// Answer second question with "Option 2"
	phase.questionView.optionSelector.items[0].selected = true

	// Go back to first question - should save second answer
	phase, cmd := phase.Update(PrevQuestionMsg{})
	assert.Equal(t, 0, phase.currentIndex, "should go back to previous question")
	assert.NotNil(t, phase.questionView, "should still have question view")
	assert.Nil(t, cmd)

	// Verify second answer was saved even though we went back
	assert.Equal(t, "Option 2", phase.answers[1].Value, "second answer should be saved when going back")

	// Verify first answer is still intact
	assert.Equal(t, "Option 1", phase.answers[0].Value, "first answer should still be intact")

	// Verify first answer is restored in the question view
	selected := phase.questionView.optionSelector.SelectedLabels()
	assert.Equal(t, 1, len(selected), "should have one selected option")
	assert.Equal(t, "Option 1", selected[0], "previously selected answer should be restored")
}

func TestAgentPhase_PrevQuestionMultiSelect(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with 2 multi-select questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Select colors",
				Header:   "Colors",
				Options: []specmcp.Option{
					{Label: "Red", Description: ""},
					{Label: "Blue", Description: ""},
					{Label: "Green", Description: ""},
				},
				Multiple: true,
			},
			{
				Question: "Select features",
				Header:   "Features",
				Options: []specmcp.Option{
					{Label: "Feature A", Description: ""},
					{Label: "Feature B", Description: ""},
				},
				Multiple: true,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question with multiple selections
	phase.questionView.optionSelector.items[0].selected = true // Red
	phase.questionView.optionSelector.items[2].selected = true // Green
	phase, _ = phase.Update(NextQuestionMsg{})

	// Verify first answer was saved as multi-select
	assert.True(t, phase.answers[0].IsMulti, "first answer should be multi-select")
	firstAnswer, ok := phase.answers[0].Value.([]string)
	assert.True(t, ok, "first answer value should be []string")
	assert.Equal(t, 2, len(firstAnswer), "should have saved 2 selections")
	assert.Contains(t, firstAnswer, "Red")
	assert.Contains(t, firstAnswer, "Green")

	// Answer second question with one selection
	phase.questionView.optionSelector.items[1].selected = true // Feature B

	// Go back to first question - should save second answer
	phase, _ = phase.Update(PrevQuestionMsg{})
	assert.Equal(t, 0, phase.currentIndex, "should go back to previous question")

	// Verify second answer was saved
	assert.True(t, phase.answers[1].IsMulti, "second answer should be multi-select")
	secondAnswer, ok := phase.answers[1].Value.([]string)
	assert.True(t, ok, "second answer value should be []string")
	assert.Equal(t, 1, len(secondAnswer), "should have saved 1 selection")
	assert.Equal(t, "Feature B", secondAnswer[0])

	// Verify first answer is still intact and restored in question view
	selected := phase.questionView.optionSelector.SelectedLabels()
	assert.Equal(t, 2, len(selected), "should have two selected options")
	assert.Contains(t, selected, "Red", "previously selected Red should be restored")
	assert.Contains(t, selected, "Green", "previously selected Green should be restored")
}

func TestAgentPhase_PrevQuestionCustomText(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with 2 questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "What is your name?",
				Header:   "Name",
				Options:  []specmcp.Option{{Label: "John", Description: ""}},
				Multiple: false,
			},
			{
				Question: "What is your role?",
				Header:   "Role",
				Options:  []specmcp.Option{{Label: "Developer", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question with custom text
	lastIdx := len(phase.questionView.optionSelector.items) - 1
	phase.questionView.optionSelector.items[lastIdx].selected = true // "Type your own answer"
	phase.questionView.customInput.SetValue("Alice")
	phase.questionView.showCustom = true
	phase, _ = phase.Update(NextQuestionMsg{})

	// Verify first answer was saved as custom text
	assert.False(t, phase.answers[0].IsMulti, "first answer should be single-select")
	assert.Equal(t, "Alice", phase.answers[0].Value, "should have saved custom text")

	// Answer second question with regular option
	phase.questionView.optionSelector.items[0].selected = true

	// Go back to first question - should save second answer
	phase, _ = phase.Update(PrevQuestionMsg{})
	assert.Equal(t, 0, phase.currentIndex, "should go back to previous question")

	// Verify second answer was saved
	assert.Equal(t, "Developer", phase.answers[1].Value, "second answer should be saved")

	// Verify first custom answer is restored in question view
	assert.True(t, phase.questionView.showCustom, "custom input should be shown")
	assert.Equal(t, "Alice", phase.questionView.customInput.Value(), "custom text should be restored")
	selected := phase.questionView.optionSelector.SelectedLabels()
	assert.Equal(t, 1, len(selected), "should have one selected option")
	assert.Equal(t, "Type your own answer", selected[0], "custom option should be selected")
}

func TestAgentPhase_SubmitAnswers(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	resultCh := make(chan []interface{}, 1)

	// Setup with 2 questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Red", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Question 2?",
				Header:   "Q2",
				Options: []specmcp.Option{
					{Label: "Feature A", Description: ""},
					{Label: "Feature B", Description: ""},
				},
				Multiple: true,
			},
		},
		ResultCh: resultCh,
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question
	phase.questionView.optionSelector.items[0].selected = true

	// Go to second question (this saves the first answer)
	phase, _ = phase.Update(NextQuestionMsg{})

	// Answer second question (multi-select)
	phase.questionView.optionSelector.items[0].selected = true
	phase.questionView.optionSelector.items[1].selected = true

	// Submit answers (this saves the second answer)
	phase, cmd := phase.Update(SubmitAnswersMsg{})

	// Should return to waiting state
	assert.True(t, phase.waitingForAgent, "should return to waiting state")
	assert.Nil(t, phase.questionView, "question view should be cleared")
	assert.Nil(t, phase.currentReq, "request should be cleared")
	assert.NotNil(t, cmd, "should return command for spinner tick + wait for next questions")

	// Verify answers were sent to result channel (goroutine should send immediately)
	select {
	case answers := <-resultCh:
		require.Len(t, answers, 2)
		assert.Equal(t, "Red", answers[0])
		assert.Equal(t, []string{"Feature A", "Feature B"}, answers[1])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected answers to be sent to result channel")
	}
}

func TestAgentPhase_SubmitAnswers_ValidationFailsCurrentQuestion(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	resultCh := make(chan []interface{}, 1)

	// Setup with 2 questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Red", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Question 2?",
				Header:   "Q2",
				Options:  []specmcp.Option{{Label: "Feature A", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: resultCh,
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question
	phase.questionView.optionSelector.items[0].selected = true

	// Go to second question (this saves the first answer)
	phase, _ = phase.Update(NextQuestionMsg{})

	// Don't answer second question - leave it empty

	// Try to submit answers
	phase, cmd := phase.Update(SubmitAnswersMsg{})

	// Should stay in question state
	assert.False(t, phase.waitingForAgent, "should stay in question state")
	assert.NotNil(t, phase.questionView, "question view should still exist")
	assert.NotNil(t, phase.currentReq, "request should not be cleared")

	// Should return error command
	require.NotNil(t, cmd, "should return error command")

	// Execute command and verify it returns ShowErrorMsg
	msg := cmd()
	errorMsg, ok := msg.(ShowErrorMsg)
	assert.True(t, ok, "command should return ShowErrorMsg")
	assert.Contains(t, errorMsg.err, "Please select an answer", "error should mention selection")

	// Verify no answers were sent to result channel
	select {
	case <-resultCh:
		t.Fatal("should not send answers when validation fails")
	case <-time.After(10 * time.Millisecond):
		// Expected - no answers sent
	}
}

func TestAgentPhase_SubmitAnswers_ValidationFailsPreviousQuestion(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	resultCh := make(chan []interface{}, 1)

	// Setup with 3 questions
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Red", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Question 2?",
				Header:   "Q2",
				Options:  []specmcp.Option{{Label: "Blue", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Question 3?",
				Header:   "Q3",
				Options:  []specmcp.Option{{Label: "Green", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: resultCh,
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question
	phase.questionView.optionSelector.items[0].selected = true
	phase, _ = phase.Update(NextQuestionMsg{})

	// Skip second question - don't answer it, just go to third
	phase, _ = phase.Update(NextQuestionMsg{}) // This should fail validation and stay on Q2

	// Since validation failed on Q2, we're still on Q2. Let's manually skip it by not answering
	// and force navigation (simulate user clearing answer after initial selection)
	phase.currentIndex = 2 // Manually move to Q3
	phase.questionView = NewQuestionView(phase.questions, phase.answers, phase.currentIndex)

	// Answer third question
	phase.questionView.optionSelector.items[0].selected = true

	// Try to submit - should fail because Q2 was never answered
	phase, cmd := phase.Update(SubmitAnswersMsg{})

	// Should stay in question state
	assert.False(t, phase.waitingForAgent, "should stay in question state")
	assert.NotNil(t, phase.questionView, "question view should still exist")

	// Should return error command
	require.NotNil(t, cmd, "should return error command")

	// Execute command and verify it returns ShowErrorMsg
	msg := cmd()
	errorMsg, ok := msg.(ShowErrorMsg)
	assert.True(t, ok, "command should return ShowErrorMsg")
	assert.Contains(t, errorMsg.err, "All questions must be answered", "error should mention all questions")

	// Verify no answers were sent to result channel
	select {
	case <-resultCh:
		t.Fatal("should not send answers when validation fails")
	case <-time.After(10 * time.Millisecond):
		// Expected - no answers sent
	}
}

func TestAgentPhase_SubmitAnswers_ValidationFailsMultiSelectEmpty(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	resultCh := make(chan []interface{}, 1)

	// Setup with 3 questions - first is single, second is multi-select, third is single
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Red", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Select features",
				Header:   "Q2",
				Options: []specmcp.Option{
					{Label: "Feature A", Description: ""},
					{Label: "Feature B", Description: ""},
				},
				Multiple: true,
			},
			{
				Question: "Question 3?",
				Header:   "Q3",
				Options:  []specmcp.Option{{Label: "Blue", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: resultCh,
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question
	phase.questionView.optionSelector.items[0].selected = true
	phase, _ = phase.Update(NextQuestionMsg{})

	// Skip second question without answering - force move to third question
	// by manually advancing index (simulating bug or edge case)
	phase.currentIndex = 2
	phase.questionView = NewQuestionView(phase.questions, phase.answers, phase.currentIndex)
	phase.answers[1] = QuestionAnswer{Value: []string{}, IsMulti: true} // Second question left empty

	// Answer third question
	phase.questionView.optionSelector.items[0].selected = true

	// Try to submit - should fail because multi-select question was never answered
	phase, cmd := phase.Update(SubmitAnswersMsg{})

	// Should stay in question state
	assert.False(t, phase.waitingForAgent, "should stay in question state")
	assert.NotNil(t, phase.questionView, "question view should still exist")

	// Should return error command
	require.NotNil(t, cmd, "should return error command")

	// Execute command and verify it returns ShowErrorMsg
	msg := cmd()
	errorMsg, ok := msg.(ShowErrorMsg)
	assert.True(t, ok, "command should return ShowErrorMsg")
	assert.Contains(t, errorMsg.err, "All questions must be answered", "error should mention all questions")

	// Verify no answers were sent to result channel
	select {
	case <-resultCh:
		t.Fatal("should not send answers when validation fails")
	case <-time.After(10 * time.Millisecond):
		// Expected - no answers sent
	}
}

func TestAgentPhase_SubmitAnswers_AllAnswersValid(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	resultCh := make(chan []interface{}, 1)

	// Setup with 3 questions - mix of single-select and multi-select
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "Question 1?",
				Header:   "Q1",
				Options:  []specmcp.Option{{Label: "Red", Description: ""}},
				Multiple: false,
			},
			{
				Question: "Select features",
				Header:   "Q2",
				Options: []specmcp.Option{
					{Label: "Feature A", Description: ""},
					{Label: "Feature B", Description: ""},
				},
				Multiple: true,
			},
			{
				Question: "Question 3?",
				Header:   "Q3",
				Options:  []specmcp.Option{{Label: "Blue", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: resultCh,
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})

	// Answer first question
	phase.questionView.optionSelector.items[0].selected = true
	phase, _ = phase.Update(NextQuestionMsg{})

	// Answer second question (multi-select)
	phase.questionView.optionSelector.items[0].selected = true
	phase, _ = phase.Update(NextQuestionMsg{})

	// Answer third question
	phase.questionView.optionSelector.items[0].selected = true

	// Submit answers - should succeed
	phase, cmd := phase.Update(SubmitAnswersMsg{})

	// Should return to waiting state
	assert.True(t, phase.waitingForAgent, "should return to waiting state")
	assert.Nil(t, phase.questionView, "question view should be cleared")
	assert.Nil(t, phase.currentReq, "request should be cleared")
	assert.NotNil(t, cmd, "should return command for spinner tick + wait for next questions")

	// Verify answers were sent to result channel
	select {
	case answers := <-resultCh:
		require.Len(t, answers, 3)
		assert.Equal(t, "Red", answers[0])
		assert.Equal(t, []string{"Feature A"}, answers[1])
		assert.Equal(t, "Blue", answers[2])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected answers to be sent to result channel")
	}
}

func TestAgentPhase_WindowSize(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	phase, cmd := phase.Update(msg)

	assert.Equal(t, 100, phase.width)
	assert.Equal(t, 50, phase.height)
	assert.Nil(t, cmd)
}

func TestAgentPhase_View_Waiting(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)
	phase.width = 80
	phase.height = 24

	view := phase.View()
	assert.NotEmpty(t, view)
	// Should contain spinner and status text
	assert.Contains(t, view, "Agent is analyzing requirements...")
}

func TestAgentPhase_View_WithQuestion(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Setup with a question
	req := specmcp.QuestionRequest{
		Questions: []specmcp.Question{
			{
				Question: "What is your favorite color?",
				Header:   "Color",
				Options:  []specmcp.Option{{Label: "Red", Description: ""}},
				Multiple: false,
			},
		},
		ResultCh: make(chan []interface{}, 1),
	}

	phase, _ = phase.Update(QuestionRequestMsg{Request: req})
	view := phase.View()

	assert.NotEmpty(t, view)
	// Should contain question text
	assert.Contains(t, view, "Color")
	assert.Contains(t, view, "What is your favorite color?")
}

func TestAgentPhase_SpecContentRequest(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Create spec content request
	resultCh := make(chan error, 1)
	req := specmcp.SpecContentRequest{
		Content:  "## Overview\n\nTest spec content\n\n## Tasks\n\n- [ ] Task 1",
		ResultCh: resultCh,
	}

	// Send spec content request message
	msg := SpecContentRequestMsg{Request: req}
	phase, cmd := phase.Update(msg)

	// Should store the request
	assert.NotNil(t, phase.currentSpecReq, "should store spec content request")
	assert.Equal(t, req.Content, phase.currentSpecReq.Content)

	// Should return command that emits SpecContentReceivedMsg
	require.NotNil(t, cmd, "should return command")

	// Execute command and verify it returns SpecContentReceivedMsg
	resultMsg := cmd()
	specContentMsg, ok := resultMsg.(SpecContentReceivedMsg)
	assert.True(t, ok, "command should return SpecContentReceivedMsg")
	assert.Equal(t, req.Content, specContentMsg.Content, "content should match")
}

func TestAgentPhase_ConfirmSpecSave(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Create spec content request
	resultCh := make(chan error, 1)
	req := specmcp.SpecContentRequest{
		Content:  "## Overview\n\nTest spec",
		ResultCh: resultCh,
	}

	// Send spec content request
	phase, _ = phase.Update(SpecContentRequestMsg{Request: req})
	assert.NotNil(t, phase.currentSpecReq)

	// Confirm save
	phase.ConfirmSpecSave()

	// Should send nil to result channel
	select {
	case err := <-resultCh:
		assert.Nil(t, err, "should send nil to indicate success")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected confirmation to be sent to result channel")
	}

	// Should clear current request
	assert.Nil(t, phase.currentSpecReq, "should clear current spec request")
}

func TestAgentPhase_Init_StartsListeningForBothChannels(t *testing.T) {
	mcpServer := specmcp.New("test-spec", "./specs")
	phase := NewAgentPhase(mcpServer)

	// Verify channels are set up
	assert.NotNil(t, phase.questionReqCh, "question channel should be set")
	assert.NotNil(t, phase.specContentCh, "spec content channel should be set")

	// Init should return batch command with spinner + both listeners
	cmd := phase.Init()
	assert.NotNil(t, cmd, "init should return command")
}
