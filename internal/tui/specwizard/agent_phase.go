package specwizard

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/specmcp"
	"github.com/mark3labs/iteratr/internal/tui"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// AgentPhase manages the agent interview phase with question handling.
type AgentPhase struct {
	// Question state
	questions    []Question
	answers      []QuestionAnswer
	currentIndex int

	// Question view component
	questionView *QuestionView

	// Agent state
	waitingForAgent bool
	spinner         tui.Spinner
	statusText      string

	// MCP communication
	mcpServer *specmcp.Server

	// Channel for receiving question requests
	questionReqCh <-chan specmcp.QuestionRequest
	currentReq    *specmcp.QuestionRequest // Current pending request

	// Channel for receiving spec content from finish-spec
	specContentCh  <-chan specmcp.SpecContentRequest
	currentSpecReq *specmcp.SpecContentRequest // Current pending spec content request

	// Dimensions
	width  int
	height int
}

// NewAgentPhase creates a new agent phase component.
func NewAgentPhase(mcpServer *specmcp.Server) *AgentPhase {
	return &AgentPhase{
		mcpServer:       mcpServer,
		questionReqCh:   mcpServer.QuestionChan(),
		specContentCh:   mcpServer.SpecContentChan(),
		waitingForAgent: true,
		spinner:         tui.NewDefaultSpinner(),
		statusText:      "Agent is analyzing requirements...",
	}
}

// Init initializes the agent phase.
func (a *AgentPhase) Init() tea.Cmd {
	// Start listening for question requests and spec content
	return tea.Batch(
		a.spinner.Tick(),
		waitForQuestionRequest(a.questionReqCh),
		waitForSpecContent(a.specContentCh),
	)
}

// Update handles messages for the agent phase.
func (a *AgentPhase) Update(msg tea.Msg) (*AgentPhase, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.questionView != nil {
			a.questionView.Update(msg)
		}
		return a, nil

	case QuestionRequestMsg:
		// Received questions from agent via MCP
		logger.Debug("Agent phase: received %d questions", len(msg.Request.Questions))

		// Convert from specmcp.Question to Question
		a.questions = make([]Question, len(msg.Request.Questions))
		for i, q := range msg.Request.Questions {
			opts := make([]Option, len(q.Options))
			for j, opt := range q.Options {
				opts[j] = Option{
					Label:       opt.Label,
					Description: opt.Description,
				}
			}
			a.questions[i] = Question{
				Question: q.Question,
				Header:   q.Header,
				Options:  opts,
				Multiple: q.Multiple,
			}
		}

		// Initialize answers array
		a.answers = make([]QuestionAnswer, len(a.questions))
		for i, q := range a.questions {
			if q.Multiple {
				a.answers[i] = QuestionAnswer{Value: []string{}, IsMulti: true}
			} else {
				a.answers[i] = QuestionAnswer{Value: "", IsMulti: false}
			}
		}

		// Store the request so we can respond when answers are submitted
		a.currentReq = &msg.Request

		// Show first question
		a.currentIndex = 0
		a.waitingForAgent = false
		a.questionView = NewQuestionView(a.questions, a.answers, a.currentIndex)

		return a, nil

	case NextQuestionMsg:
		// Validate current answer
		if !a.questionView.validateAnswer() {
			return a, func() tea.Msg {
				return ShowErrorMsg{err: "Please select an answer or enter custom text"}
			}
		}

		// Save answer
		a.questionView.saveCurrentAnswer()
		a.answers = a.questionView.answers

		// Navigate to next question
		if a.currentIndex < len(a.questions)-1 {
			a.currentIndex++
			a.questionView = NewQuestionView(a.questions, a.answers, a.currentIndex)
		}

		return a, nil

	case PrevQuestionMsg:
		// Save current answer (no validation)
		a.questionView.saveCurrentAnswer()
		a.answers = a.questionView.answers

		// Navigate to previous question
		if a.currentIndex > 0 {
			a.currentIndex--
			a.questionView = NewQuestionView(a.questions, a.answers, a.currentIndex)
		}

		return a, nil

	case SubmitAnswersMsg:
		// Validate all answers
		if !a.questionView.validateAnswer() {
			return a, func() tea.Msg {
				return ShowErrorMsg{err: "Please select an answer or enter custom text"}
			}
		}

		// Save final answer
		a.questionView.saveCurrentAnswer()
		a.answers = a.questionView.answers

		// Send answers back to MCP handler
		logger.Debug("Agent phase: submitting %d answers", len(a.answers))

		// Format answers for MCP
		mcpAnswers := make([]interface{}, len(a.answers))
		for i, ans := range a.answers {
			if ans.IsMulti {
				mcpAnswers[i] = ans.Value // []string
			} else {
				mcpAnswers[i] = ans.Value // string
			}
		}

		// Send to MCP handler's result channel (capture in local var before clearing)
		if a.currentReq != nil {
			resultCh := a.currentReq.ResultCh
			go func() {
				resultCh <- mcpAnswers
			}()
		}

		// Return to spinner while agent processes
		a.waitingForAgent = true
		a.statusText = "Agent is processing your answers..."
		a.questionView = nil
		a.currentReq = nil

		return a, tea.Batch(
			a.spinner.Tick(),
			waitForQuestionRequest(a.questionReqCh),
			waitForSpecContent(a.specContentCh),
		)

	case SpecContentRequestMsg:
		// Received spec content from finish-spec handler
		logger.Debug("Agent phase: received spec content (%d bytes)", len(msg.Request.Content))

		// Store the request so we can respond after user review
		a.currentSpecReq = &msg.Request

		// Emit SpecContentReceivedMsg to wizard so it can transition to review step
		return a, func() tea.Msg {
			return SpecContentReceivedMsg{Content: msg.Request.Content}
		}

	case ShowErrorMsg:
		// TODO: Display error message (for now just log)
		logger.Warn("Validation error: %s", msg.err)
		return a, nil

	default:
		// Update spinner if waiting
		if a.waitingForAgent {
			cmd := a.spinner.Update(msg)
			return a, cmd
		}

		// Update question view if showing questions
		if a.questionView != nil {
			cmd := a.questionView.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

// View renders the agent phase.
func (a *AgentPhase) View() string {
	currentTheme := theme.Current()

	if a.waitingForAgent {
		// Show spinner
		spinnerView := lipgloss.JoinHorizontal(
			lipgloss.Left,
			a.spinner.View(),
			" "+a.statusText,
		)

		centeredStyle := lipgloss.NewStyle().
			Width(a.width).
			Height(a.height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Foreground(lipgloss.Color(currentTheme.FgMuted))

		return centeredStyle.Render(spinnerView)
	}

	// Show question view
	if a.questionView != nil {
		return a.questionView.View()
	}

	return "Initializing..."
}

// SetSize updates the size of the agent phase.
func (a *AgentPhase) SetSize(width, height int) {
	a.width = width
	a.height = height
	if a.questionView != nil {
		a.questionView.SetSize(width, height)
	}
}

// ConfirmSpecSave sends confirmation to the finish-spec MCP handler that the spec was saved.
// This unblocks the MCP handler and allows the agent to complete.
func (a *AgentPhase) ConfirmSpecSave() {
	if a.currentSpecReq != nil {
		resultCh := a.currentSpecReq.ResultCh
		go func() {
			resultCh <- nil // Send nil to indicate success
		}()
		a.currentSpecReq = nil
	}
}

// QuestionRequestMsg wraps a question request from the MCP server.
type QuestionRequestMsg struct {
	Request specmcp.QuestionRequest
}

// waitForQuestionRequest returns a command that waits for a question request.
func waitForQuestionRequest(ch <-chan specmcp.QuestionRequest) tea.Cmd {
	return func() tea.Msg {
		req := <-ch
		return QuestionRequestMsg{Request: req}
	}
}

// waitForSpecContent returns a command that waits for spec content from finish-spec.
func waitForSpecContent(ch <-chan specmcp.SpecContentRequest) tea.Cmd {
	return func() tea.Msg {
		req := <-ch
		return SpecContentRequestMsg{Request: req}
	}
}

// ListenForQuestions starts a goroutine that listens for question requests
// and sends them as messages to the Bubbletea program.
func ListenForQuestions(ctx context.Context, mcpServer *specmcp.Server) tea.Cmd {
	return func() tea.Msg {
		select {
		case req := <-mcpServer.QuestionChan():
			return QuestionRequestMsg{Request: req}
		case <-ctx.Done():
			return nil
		}
	}
}
