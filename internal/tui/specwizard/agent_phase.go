package specwizard

import (
	"context"
	"time"

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

	// Listener lifecycle management
	listenerCtx    context.Context
	listenerCancel context.CancelFunc

	// Cancel confirmation modal
	showConfirmCancel bool // True if cancel confirmation modal is visible

	// Dimensions
	width  int
	height int
}

// NewAgentPhase creates a new agent phase component.
func NewAgentPhase(mcpServer *specmcp.Server) *AgentPhase {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentPhase{
		mcpServer:       mcpServer,
		questionReqCh:   mcpServer.QuestionChan(),
		specContentCh:   mcpServer.SpecContentChan(),
		waitingForAgent: true,
		spinner:         tui.NewDefaultSpinner(),
		statusText:      "Agent is analyzing requirements...",
		listenerCtx:     ctx,
		listenerCancel:  cancel,
	}
}

// Init initializes the agent phase.
func (a *AgentPhase) Init() tea.Cmd {
	// Start listening for question requests and spec content
	return tea.Batch(
		a.spinner.Tick(),
		waitForQuestionRequest(a.listenerCtx, a.questionReqCh),
		waitForSpecContent(a.listenerCtx, a.specContentCh),
	)
}

// Update handles messages for the agent phase.
func (a *AgentPhase) Update(msg tea.Msg) (*AgentPhase, tea.Cmd) {
	// Handle cancel confirmation modal
	if a.showConfirmCancel {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "y", "Y":
				// Confirm cancellation - return CancelWizardMsg
				a.showConfirmCancel = false
				return a, func() tea.Msg {
					return CancelWizardMsg{}
				}
			case "n", "N", "esc":
				// Cancel the cancellation (don't cancel)
				a.showConfirmCancel = false
				return a, nil
			}
		}
		return a, nil
	}

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

		if len(msg.Request.Questions) == 0 {
			logger.Warn("Agent phase: received empty question set")
			if msg.Request.ResultCh != nil {
				resultCh := msg.Request.ResultCh
				go func() {
					select {
					case resultCh <- []interface{}{}:
					case <-time.After(5 * time.Second):
						logger.Warn("Timeout sending empty answers to MCP handler")
					}
				}()
			}
			// Re-register listener so the agent continues to receive future requests
			return a, waitForQuestionRequest(a.listenerCtx, a.questionReqCh)
		}

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
		a.questionView.SetSize(a.width, a.height)
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
			a.questionView.SetSize(a.width, a.height)
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
			a.questionView.SetSize(a.width, a.height)
		}

		return a, nil

	case SubmitAnswersMsg:
		// Validate current answer
		if !a.questionView.validateAnswer() {
			return a, func() tea.Msg {
				return ShowErrorMsg{err: "Please select an answer or enter custom text"}
			}
		}

		// Save final answer
		a.questionView.saveCurrentAnswer()
		a.answers = a.questionView.answers

		// Validate all answers are non-empty
		for i, ans := range a.answers {
			if ans.IsMulti {
				values, ok := ans.Value.([]string)
				if !ok || len(values) == 0 {
					return a, func() tea.Msg {
						return ShowErrorMsg{err: "All questions must be answered. Please go back and answer all questions."}
					}
				}
			} else {
				value, ok := ans.Value.(string)
				if !ok || value == "" {
					return a, func() tea.Msg {
						return ShowErrorMsg{err: "All questions must be answered. Please go back and answer all questions."}
					}
				}
			}
			logger.Debug("Answer %d: %+v", i, ans)
		}

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
				select {
				case resultCh <- mcpAnswers:
					// Successfully sent
				case <-time.After(5 * time.Second):
					logger.Warn("Timeout sending answers to MCP handler")
				}
			}()
		}

		// Return to spinner while agent processes
		a.waitingForAgent = true
		a.statusText = "Agent is processing your answers..."
		a.questionView = nil
		a.currentReq = nil

		// Cancel old listeners and create new context
		a.listenerCancel()
		ctx, cancel := context.WithCancel(context.Background())
		a.listenerCtx = ctx
		a.listenerCancel = cancel

		return a, tea.Batch(
			a.spinner.Tick(),
			waitForQuestionRequest(a.listenerCtx, a.questionReqCh),
			waitForSpecContent(a.listenerCtx, a.specContentCh),
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
		logger.Warn("Validation error: %s", msg.err)
		// Surface the error in the active question view so the user sees it
		if a.questionView != nil {
			a.questionView.errText = msg.err
		}
		return a, nil

	case ShowCancelConfirmMsg:
		// ESC pressed on first question: show cancel confirmation modal
		a.showConfirmCancel = true
		return a, nil

	default:
		// Handle keyboard input during waiting state
		if a.waitingForAgent {
			switch msg := msg.(type) {
			case tea.KeyPressMsg:
				switch msg.String() {
				case "esc":
					// ESC during thinking shows cancel confirmation modal
					a.showConfirmCancel = true
					return a, nil
				}
			}
			// Update spinner
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

	var baseView string
	if a.waitingForAgent {
		// Show spinner - simple inline view, wizard modal handles centering
		spinnerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(currentTheme.FgMuted))
		baseView = spinnerStyle.Render(a.spinner.View() + " " + a.statusText)
	} else if a.questionView != nil {
		// Show question view
		baseView = a.questionView.View()
	} else {
		baseView = "Initializing..."
	}

	// If showing confirmation modal, overlay it on top
	if a.showConfirmCancel {
		modal := a.renderCancelConfirmationModal()
		// Center modal overlay
		overlay := lipgloss.Place(
			a.width,
			a.height,
			lipgloss.Center,
			lipgloss.Center,
			modal,
		)
		return overlay
	}

	return baseView
}

// renderCancelConfirmationModal renders the cancel confirmation modal.
func (a *AgentPhase) renderCancelConfirmationModal() string {
	t := theme.Current()

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Warning)).
		MarginBottom(1)
	title := titleStyle.Render("⚠ Cancel Agent Interview?")

	// Message
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgBase)).
		MarginBottom(1)
	message := messageStyle.Render("This will stop the agent and return to the beginning.")

	// Buttons
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgMuted))
	buttons := buttonStyle.Render("Press Y to cancel, N or ESC to continue")

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		message,
		"",
		buttons,
	)

	// Modal styling
	modalStyle := lipgloss.NewStyle().
		Width(50).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Warning))

	return modalStyle.Render(content)
}

// SetSize updates the size of the agent phase.
func (a *AgentPhase) SetSize(width, height int) {
	a.width = width
	a.height = height
	if a.questionView != nil {
		a.questionView.SetSize(width, height)
	}
}

// Cleanup cancels any active channel listeners to prevent goroutine leaks.
func (a *AgentPhase) Cleanup() {
	if a.listenerCancel != nil {
		a.listenerCancel()
	}
}

// ConfirmSpecSave sends confirmation to the finish-spec MCP handler that the spec was saved.
// This unblocks the MCP handler and allows the agent to complete.
func (a *AgentPhase) ConfirmSpecSave() {
	if a.currentSpecReq != nil {
		resultCh := a.currentSpecReq.ResultCh
		go func() {
			select {
			case resultCh <- nil: // Send nil to indicate success
				// Successfully sent
			case <-time.After(5 * time.Second):
				logger.Warn("Timeout confirming spec save to MCP handler")
			}
		}()
		a.currentSpecReq = nil
	}
}

// QuestionRequestMsg wraps a question request from the MCP server.
type QuestionRequestMsg struct {
	Request specmcp.QuestionRequest
}

// waitForQuestionRequest returns a command that waits for a question request with context.
func waitForQuestionRequest(ctx context.Context, ch <-chan specmcp.QuestionRequest) tea.Cmd {
	return func() tea.Msg {
		select {
		case req := <-ch:
			return QuestionRequestMsg{Request: req}
		case <-ctx.Done():
			return nil
		}
	}
}

// waitForSpecContent returns a command that waits for spec content with context.
func waitForSpecContent(ctx context.Context, ch <-chan specmcp.SpecContentRequest) tea.Cmd {
	return func() tea.Msg {
		select {
		case req := <-ch:
			return SpecContentRequestMsg{Request: req}
		case <-ctx.Done():
			return nil
		}
	}
}

// CancelWizardMsg is sent when the user confirms cancelling the wizard during agent phase.
type CancelWizardMsg struct{}
