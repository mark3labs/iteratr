package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/mark3labs/iteratr/internal/agent"
)

// SubagentModal displays a full-screen modal that loads and replays a subagent session.
// It reuses the existing ScrollList and MessageItem infrastructure from AgentOutput.
type SubagentModal struct {
	// Content display (reuses AgentOutput infrastructure)
	messages  []MessageItem
	toolIndex map[string]int // toolCallId → message index

	// Session metadata
	sessionID    string
	subagentType string
	workDir      string

	// ACP subprocess will be added when implementing Start()

	// State
	loading bool

	// Spinner for loading state (created lazily when needed)
	spinner *GradientSpinner

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSubagentModal creates a new SubagentModal.
func NewSubagentModal(sessionID, subagentType, workDir string) *SubagentModal {
	ctx, cancel := context.WithCancel(context.Background())
	spinner := NewDefaultGradientSpinner("Loading session...")
	return &SubagentModal{
		sessionID:    sessionID,
		subagentType: subagentType,
		workDir:      workDir,
		messages:     make([]MessageItem, 0),
		toolIndex:    make(map[string]int),
		loading:      true,
		ctx:          ctx,
		cancel:       cancel,
		spinner:      &spinner,
	}
}

// Start spawns the ACP subprocess, initializes it, and begins loading the session.
// Returns a command that will start the session loading process.
func (m *SubagentModal) Start() tea.Cmd {
	// This will be implemented in task TAS-16
	return nil
}

// Draw renders the modal as a full-screen overlay.
func (m *SubagentModal) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// This will be implemented in task TAS-19
	return nil
}

// Update handles keyboard input for scrolling.
func (m *SubagentModal) Update(msg tea.Msg) tea.Cmd {
	// This will be implemented in task TAS-20
	return nil
}

// HandleUpdate processes streaming messages from the subagent session.
// Returns a command to continue streaming if Continue is true.
func (m *SubagentModal) HandleUpdate(msg tea.Msg) tea.Cmd {
	// This will be implemented in task TAS-17 (continuous streaming)
	return nil
}

// Close terminates the ACP subprocess and cleans up resources.
func (m *SubagentModal) Close() {
	// This will be implemented in task TAS-21
}

// appendText adds a text message to the modal.
// Mirrors AgentOutput.AppendText() logic.
func (m *SubagentModal) appendText(content string) {
	// If last message is a TextMessageItem, append to it
	if len(m.messages) > 0 {
		if textMsg, ok := m.messages[len(m.messages)-1].(*TextMessageItem); ok {
			textMsg.content += content
			// Invalidate cache - will re-render on next View() call
			textMsg.cachedWidth = 0
			return
		}
	}

	// Create new TextMessageItem
	newMsg := &TextMessageItem{
		id:      fmt.Sprintf("text-%d", len(m.messages)),
		content: content,
	}
	m.messages = append(m.messages, newMsg)
}

// appendToolCall handles tool lifecycle events in the subagent session.
// Mirrors AgentOutput.AppendToolCall() logic.
func (m *SubagentModal) appendToolCall(event agent.ToolCallEvent) {
	idx, exists := m.toolIndex[event.ToolCallID]
	if !exists {
		// Map status strings to ToolStatus enum
		status := mapToolStatus(event.Status)

		// Convert agent.FileDiff to tui.FileDiff if present
		var fileDiff *FileDiff
		if event.FileDiff != nil {
			fileDiff = &FileDiff{
				File:      event.FileDiff.File,
				Before:    event.FileDiff.Before,
				After:     event.FileDiff.After,
				Additions: event.FileDiff.Additions,
				Deletions: event.FileDiff.Deletions,
			}
		}

		// Create new ToolMessageItem
		newMsg := &ToolMessageItem{
			id:       event.ToolCallID,
			toolName: event.Title,
			kind:     event.Kind,
			status:   status,
			input:    event.RawInput,
			output:   event.Output,
			fileDiff: fileDiff,
			maxLines: 10,
		}
		m.messages = append(m.messages, newMsg)
		m.toolIndex[event.ToolCallID] = len(m.messages) - 1
	} else {
		// Update existing tool call in-place
		if toolMsg, ok := m.messages[idx].(*ToolMessageItem); ok {
			toolMsg.status = mapToolStatus(event.Status)
			if event.Kind != "" {
				toolMsg.kind = event.Kind
			}
			if len(event.RawInput) > 0 {
				toolMsg.input = event.RawInput
			}
			if event.Output != "" {
				toolMsg.output = event.Output
			}
			if event.FileDiff != nil {
				toolMsg.fileDiff = &FileDiff{
					File:      event.FileDiff.File,
					Before:    event.FileDiff.Before,
					After:     event.FileDiff.After,
					Additions: event.FileDiff.Additions,
					Deletions: event.FileDiff.Deletions,
				}
			}
			// Invalidate cache - will re-render on next View() call
			toolMsg.cachedWidth = 0
		}
	}
}

// appendThinking adds thinking/reasoning content to the modal.
// Mirrors AgentOutput.AppendThinking() logic.
func (m *SubagentModal) appendThinking(content string) {
	// If last message is a ThinkingMessageItem, append to it
	if len(m.messages) > 0 {
		if thinkingMsg, ok := m.messages[len(m.messages)-1].(*ThinkingMessageItem); ok {
			thinkingMsg.content += content
			// Invalidate cache - will re-render on next View() call
			thinkingMsg.cachedWidth = 0
			return
		}
	}

	// Create new ThinkingMessageItem
	newMsg := &ThinkingMessageItem{
		id:        fmt.Sprintf("thinking-%d", len(m.messages)),
		content:   content,
		collapsed: true, // default true
	}
	m.messages = append(m.messages, newMsg)
}

// appendUserMessage adds a user message to the modal.
// Mirrors AgentOutput.AppendUserMessage() logic.
func (m *SubagentModal) appendUserMessage(text string) {
	// Generate unique ID with nanosecond timestamp
	id := fmt.Sprintf("user-%d", time.Now().UnixNano())

	// Create new UserMessageItem
	newMsg := &UserMessageItem{
		id:      id,
		content: text,
	}

	// Append to messages slice
	m.messages = append(m.messages, newMsg)
}
