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

	// Go to second question
	phase.questionView.optionSelector.items[0].selected = true
	phase, _ = phase.Update(NextQuestionMsg{})
	assert.Equal(t, 1, phase.currentIndex)

	// Go back to first question
	phase, cmd := phase.Update(PrevQuestionMsg{})
	assert.Equal(t, 0, phase.currentIndex, "should go back to previous question")
	assert.NotNil(t, phase.questionView, "should still have question view")
	assert.Nil(t, cmd)
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
