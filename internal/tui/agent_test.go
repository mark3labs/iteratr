package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// --- Initialization Tests ---

func TestAgentOutput_NewAgentOutput(t *testing.T) {
	ao := NewAgentOutput()

	if ao == nil {
		t.Fatal("expected non-nil agent output")
	}

	// Verify input field is initialized
	if ao.input.Placeholder != "Send a message..." {
		t.Errorf("expected placeholder 'Send a message...', got %q", ao.input.Placeholder)
	}
	if ao.input.Prompt != "> " {
		t.Errorf("expected prompt '> ', got %q", ao.input.Prompt)
	}

	// Verify initial state
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex -1, got %d", ao.focusedIndex)
	}
	if ao.ready {
		t.Error("expected ready=false before UpdateSize")
	}
	if ao.input.Focused() {
		t.Error("expected input unfocused initially")
	}
}

// --- Size and Layout Tests ---

func TestAgentOutput_Append(t *testing.T) {
	ao := NewAgentOutput()

	// Append some content
	cmd := ao.Append("Test content")

	// Command can be nil - just verify it doesn't panic
	_ = cmd

	// Verify content was added
	if len(ao.messages) == 0 {
		t.Error("expected at least one message after Append")
	}
}

func TestAgentOutput_UpdateSize(t *testing.T) {
	ao := NewAgentOutput()

	cmd := ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Command can be nil - just verify it doesn't panic
	_ = cmd

	if ao.width != testfixtures.TestTermWidth {
		t.Errorf("width: got %d, want %d", ao.width, testfixtures.TestTermWidth)
	}
	if ao.height != testfixtures.TestTermHeight {
		t.Errorf("height: got %d, want %d", ao.height, testfixtures.TestTermHeight)
	}
	if !ao.ready {
		t.Error("expected viewport to be ready after UpdateSize")
	}

	// Verify input width is configured
	expectedInputWidth := testfixtures.TestTermWidth - 4
	if ao.input.Width() != expectedInputWidth {
		t.Errorf("input width: got %d, want %d", ao.input.Width(), expectedInputWidth)
	}
}

func TestAgentOutput_Render(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Render returns empty string when not ready or no content
	// Just verify it doesn't panic
	_ = ao.Render()

	// Add content and verify it renders
	ao.AppendText("Test message")
	output := ao.Render()
	if output == "" {
		t.Error("expected non-empty output with content")
	}
}

// --- Message Appending Tests ---

func TestAgentOutput_AppendText(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// AppendText uses Append which combines consecutive text into single message
	ao.AppendText("First message")

	if len(ao.messages) == 0 {
		t.Fatal("expected at least one message")
	}

	// Verify it's a text message
	if _, ok := ao.messages[0].(*TextMessageItem); !ok {
		t.Errorf("expected TextMessageItem, got %T", ao.messages[0])
	}
}

func TestAgentOutput_AppendThinking(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	ao.AppendThinking("Processing request...")

	if len(ao.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ao.messages))
	}

	thinkingMsg, ok := ao.messages[0].(*ThinkingMessageItem)
	if !ok {
		t.Fatalf("expected ThinkingMessageItem, got %T", ao.messages[0])
	}
	if thinkingMsg.finished {
		t.Error("expected thinking message to not be finished initially")
	}
}

func TestAgentOutput_AppendToolCall_NewTool(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "pending",
		Input:      map[string]any{"filePath": "test.go"},
	}
	ao.AppendToolCall(toolMsg)

	if len(ao.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ao.messages))
	}

	toolItem, ok := ao.messages[0].(*ToolMessageItem)
	if !ok {
		t.Fatalf("expected ToolMessageItem, got %T", ao.messages[0])
	}
	if toolItem.ID() != "tool-1" {
		t.Errorf("expected ID 'tool-1', got %q", toolItem.ID())
	}
	if toolItem.status != ToolStatusPending {
		t.Errorf("expected status Pending, got %d", toolItem.status)
	}
}

func TestAgentOutput_AppendToolCall_UpdateExisting(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add initial pending tool
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "pending",
		Input:      map[string]any{},
	})

	// Update to completed with output
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "file contents here",
	})

	// Should still have only 1 message (updated, not added)
	if len(ao.messages) != 1 {
		t.Fatalf("expected 1 message after update, got %d", len(ao.messages))
	}

	toolItem := ao.messages[0].(*ToolMessageItem)
	if toolItem.status != ToolStatusSuccess {
		t.Errorf("expected status Success, got %d", toolItem.status)
	}
	if toolItem.output != "file contents here" {
		t.Errorf("expected output 'file contents here', got %q", toolItem.output)
	}
}

// --- Message Expansion Tests ---

func TestAgentOutput_ToggleExpanded(t *testing.T) {
	ao := NewAgentOutput()
	// Don't set ready manually - UpdateSize will initialize scrollList and set ready
	ao.UpdateSize(80, 20)

	// Add a tool message with long output (more than maxLines)
	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
	}
	ao.AppendToolCall(toolMsg)

	// Set focus on this tool message
	ao.focusedIndex = 0

	toolItem := ao.messages[0].(*ToolMessageItem)

	// Render at initial width to populate cache
	toolItem.Render(76) // contentWidth = width - 4 = 80 - 4 = 76
	initialCachedWidth := toolItem.cachedWidth
	initialCachedRender := toolItem.cachedRender

	if initialCachedWidth != 76 {
		t.Errorf("expected cachedWidth to be 76, got %d", initialCachedWidth)
	}
	if initialCachedRender == "" {
		t.Fatal("expected cachedRender to be populated")
	}

	// Verify message is not expanded initially
	if toolItem.IsExpanded() {
		t.Fatal("expected tool message to not be expanded initially")
	}

	// Toggle expansion with space key
	keyMsg := tea.KeyPressMsg{Code: ' ', Text: " "}
	ao.Update(keyMsg)

	// Message should now be expanded
	if !toolItem.IsExpanded() {
		t.Error("expected tool message to be expanded after toggle")
	}

	// With ScrollList, items are rendered lazily when View() is called
	// So we need to call View() to trigger rendering and populate the cache
	scrollListContent := ao.scrollList.View()

	// The scroll list should have content (not empty)
	if scrollListContent == "" {
		t.Error("expected scroll list content to be non-empty after refresh")
	}

	// After View() is called, cache should be populated
	// contentWidth = width(80) - 5 (border, padding, left margin)
	if toolItem.cachedWidth != 75 {
		t.Errorf("expected cachedWidth to be refreshed to 75, got %d", toolItem.cachedWidth)
	}

	// The cached render should be different from initial (because expansion state changed)
	newCachedRender := toolItem.cachedRender
	if newCachedRender == initialCachedRender {
		t.Error("expected cachedRender to change after toggle (different expansion state)")
	}

	// The new cached render should contain more lines (expanded shows all output)
	if !containsSubstring(newCachedRender, "line11") || !containsSubstring(newCachedRender, "line12") {
		t.Error("expected expanded output to contain all lines including line11 and line12")
	}
}

// --- Subagent Detection Tests ---

func TestAgentOutput_SubagentDetection(t *testing.T) {
	ao := NewAgentOutput()
	ao.ready = true
	ao.UpdateSize(80, 20)

	// Simulate ACP flow: first call is "pending" with empty RawInput
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "task-1",
		Title:      "Task",
		Status:     "pending",
		Input:      map[string]any{}, // Empty on pending per ACP protocol
	})

	// Verify it was created as ToolMessageItem (since no subagent_type yet)
	if len(ao.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ao.messages))
	}
	if _, ok := ao.messages[0].(*ToolMessageItem); !ok {
		t.Fatal("expected message to be ToolMessageItem on pending (no subagent_type)")
	}

	// Second call is "in_progress" with populated RawInput containing subagent_type
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "task-1",
		Title:      "Task",
		Status:     "in_progress",
		Input: map[string]any{
			"subagent_type": "codebase-analyzer",
			"prompt":        "Analyze the codebase",
		},
	})

	// Verify it was converted to SubagentMessageItem
	if len(ao.messages) != 1 {
		t.Fatalf("expected 1 message after update, got %d", len(ao.messages))
	}
	subagentMsg, ok := ao.messages[0].(*SubagentMessageItem)
	if !ok {
		t.Fatal("expected message to be converted to SubagentMessageItem on update with subagent_type")
	}
	if subagentMsg.subagentType != "codebase-analyzer" {
		t.Errorf("expected subagentType to be 'codebase-analyzer', got %q", subagentMsg.subagentType)
	}
	if subagentMsg.description != "Analyze the codebase" {
		t.Errorf("expected description to be 'Analyze the codebase', got %q", subagentMsg.description)
	}
	if subagentMsg.status != ToolStatusRunning {
		t.Errorf("expected status to be Running, got %d", subagentMsg.status)
	}

	// Third call is "completed" with sessionID
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "task-1",
		Title:      "Task",
		Status:     "completed",
		Input: map[string]any{
			"subagent_type": "codebase-analyzer",
			"prompt":        "Analyze the codebase",
		},
		SessionID: "session-123",
	})

	// Verify sessionID was updated
	subagentMsg, ok = ao.messages[0].(*SubagentMessageItem)
	if !ok {
		t.Fatal("expected message to still be SubagentMessageItem")
	}
	if subagentMsg.sessionID != "session-123" {
		t.Errorf("expected sessionID to be 'session-123', got %q", subagentMsg.sessionID)
	}
	if subagentMsg.status != ToolStatusSuccess {
		t.Errorf("expected status to be Success, got %d", subagentMsg.status)
	}
}

// --- Keyboard Navigation Tests ---

func TestAgentOutput_ScrollingBehavior(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add enough content to require scrolling
	for i := 0; i < 10; i++ {
		ao.AppendText("Line of text that should create scrollable content")
	}

	// Scroll should start at bottom (auto-scroll enabled)
	if !ao.scrollList.AtBottom() {
		t.Errorf("expected scroll to be at bottom initially")
	}

	// Press Up arrow - should scroll up and disable auto-scroll
	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	ao.Update(upKey)
	if ao.scrollList.autoScroll {
		t.Errorf("expected autoScroll to be disabled after scrolling up")
	}

	// Press Down arrow - should scroll down
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	ao.Update(downKey)

	// Press 'k' - should also scroll up (vim-style backup)
	kKey := tea.KeyPressMsg{Code: 'k'}
	ao.Update(kKey)

	// Press 'j' - should also scroll down (vim-style backup)
	jKey := tea.KeyPressMsg{Code: 'j'}
	ao.Update(jKey)

	// Scrolling to bottom should re-enable auto-scroll
	for i := 0; i < 20; i++ {
		ao.Update(downKey)
	}
	if !ao.scrollList.AtBottom() || !ao.scrollList.autoScroll {
		t.Errorf("expected autoScroll to be re-enabled when reaching bottom")
	}
}

func TestAgentOutput_UpDownKeyHandling_NoExpandableMessages(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add only text messages (not expandable)
	ao.AppendText("First text message")
	ao.AppendText("Second text message")

	// focusedIndex should always be -1 (focus navigation removed)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to be -1, got %d", ao.focusedIndex)
	}

	// Up/Down keys now scroll the viewport (not focus navigation)
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	ao.Update(downKey)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to remain -1, got %d", ao.focusedIndex)
	}

	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	ao.Update(upKey)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to remain -1, got %d", ao.focusedIndex)
	}
}

func TestAgentOutput_UpDownKeyHandling_EmptyMessages(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// No messages at all
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to be -1, got %d", ao.focusedIndex)
	}

	// Press Down arrow - should not change focus (no messages)
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	ao.Update(downKey)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to remain -1, got %d", ao.focusedIndex)
	}

	// Press Up arrow - should not change focus (no messages)
	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	ao.Update(upKey)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to remain -1, got %d", ao.focusedIndex)
	}
}

func TestAgentOutput_UpDownKeyHandling_SingleExpandableMessage(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add only one expandable message
	ao.AppendThinking("Single thinking block")

	// focusedIndex should always be -1 (focus navigation removed, up/down scroll instead)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to be -1, got %d", ao.focusedIndex)
	}

	// Up/Down keys now scroll the viewport (not focus navigation)
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	ao.Update(downKey)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to remain -1, got %d", ao.focusedIndex)
	}

	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	ao.Update(upKey)
	if ao.focusedIndex != -1 {
		t.Errorf("expected focusedIndex to remain -1, got %d", ao.focusedIndex)
	}
}

// --- Mouse Interaction Tests ---

func TestAgentOutput_ToggleExpandedViaClick(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add a tool message
	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
	}
	ao.AppendToolCall(toolMsg)

	toolItem := ao.messages[0].(*ToolMessageItem)

	// Verify message is not expanded initially
	if toolItem.IsExpanded() {
		t.Fatal("expected tool message to not be expanded initially")
	}

	// Toggle expansion via direct method call (simulating click)
	toolItem.ToggleExpanded()

	// Message should now be expanded
	if !toolItem.IsExpanded() {
		t.Error("expected tool message to be expanded after toggle")
	}

	// Toggle back
	toolItem.ToggleExpanded()

	// Message should now be collapsed
	if toolItem.IsExpanded() {
		t.Error("expected tool message to be collapsed after second toggle")
	}
}

// --- Finish Message Tests ---

func TestAgentOutput_AppendFinish_Normal(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add a thinking message
	ao.AppendThinking("Processing request...")

	// Initially the thinking message should not be finished
	thinkingMsg := ao.messages[0].(*ThinkingMessageItem)
	if thinkingMsg.finished {
		t.Error("expected thinking message to not be finished initially")
	}

	// Call AppendFinish with normal completion
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 5000000000, // 5 seconds in nanoseconds
	}
	ao.AppendFinish(finishMsg)

	// Verify thinking message is now finished with duration
	if !thinkingMsg.finished {
		t.Error("expected thinking message to be finished after AppendFinish")
	}
	if thinkingMsg.duration != finishMsg.Duration {
		t.Errorf("expected thinking duration to be %v, got %v", finishMsg.Duration, thinkingMsg.duration)
	}

	// Note: cache will be repopulated by refreshContent() at end of AppendFinish,
	// so we don't check cachedWidth=0 here. The important thing is that finished=true
	// and duration is set, which will affect the rendered output (footer with duration).

	// Verify InfoMessageItem was appended
	if len(ao.messages) < 2 {
		t.Fatal("expected at least 2 messages after AppendFinish")
	}
	infoMsg, ok := ao.messages[1].(*InfoMessageItem)
	if !ok {
		t.Fatal("expected second message to be InfoMessageItem")
	}
	if infoMsg.model != finishMsg.Model {
		t.Errorf("expected info model to be %s, got %s", finishMsg.Model, infoMsg.model)
	}
	if infoMsg.provider != finishMsg.Provider {
		t.Errorf("expected info provider to be %s, got %s", finishMsg.Provider, infoMsg.provider)
	}
	if infoMsg.duration != finishMsg.Duration {
		t.Errorf("expected info duration to be %v, got %v", finishMsg.Duration, infoMsg.duration)
	}

	// Verify no error or cancel messages were added
	if len(ao.messages) != 2 {
		t.Errorf("expected exactly 2 messages for normal finish, got %d", len(ao.messages))
	}
}

func TestAgentOutput_AppendFinish_WithError(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add a thinking message
	ao.AppendThinking("Processing request...")

	// Call AppendFinish with error
	finishMsg := AgentFinishMsg{
		Reason:   "error",
		Error:    "Connection timeout",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 2000000000, // 2 seconds
	}
	ao.AppendFinish(finishMsg)

	// Verify thinking message is finished
	thinkingMsg := ao.messages[0].(*ThinkingMessageItem)
	if !thinkingMsg.finished {
		t.Error("expected thinking message to be finished")
	}

	// Verify InfoMessageItem was appended
	if len(ao.messages) < 2 {
		t.Fatal("expected at least 2 messages")
	}
	_, ok := ao.messages[1].(*InfoMessageItem)
	if !ok {
		t.Fatal("expected second message to be InfoMessageItem")
	}

	// Verify error TextMessageItem was appended
	if len(ao.messages) < 3 {
		t.Fatal("expected at least 3 messages (thinking, info, error)")
	}
	errorMsg, ok := ao.messages[2].(*TextMessageItem)
	if !ok {
		t.Fatal("expected third message to be TextMessageItem for error")
	}
	if !containsSubstring(errorMsg.content, "Error") || !containsSubstring(errorMsg.content, "Connection timeout") {
		t.Errorf("expected error message to contain error text, got: %s", errorMsg.content)
	}

	// Total should be 3 messages: thinking, info, error
	if len(ao.messages) != 3 {
		t.Errorf("expected exactly 3 messages for error finish, got %d", len(ao.messages))
	}
}

func TestAgentOutput_AppendFinish_Canceled(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add a thinking message
	ao.AppendThinking("Processing request...")

	// Call AppendFinish with canceled
	finishMsg := AgentFinishMsg{
		Reason:   "cancelled",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 1000000000, // 1 second
	}
	ao.AppendFinish(finishMsg)

	// Verify thinking message is finished
	thinkingMsg := ao.messages[0].(*ThinkingMessageItem)
	if !thinkingMsg.finished {
		t.Error("expected thinking message to be finished")
	}

	// Verify InfoMessageItem was appended
	if len(ao.messages) < 2 {
		t.Fatal("expected at least 2 messages")
	}
	_, ok := ao.messages[1].(*InfoMessageItem)
	if !ok {
		t.Fatal("expected second message to be InfoMessageItem")
	}

	// Verify cancel TextMessageItem was appended
	if len(ao.messages) < 3 {
		t.Fatal("expected at least 3 messages (thinking, info, cancel)")
	}
	cancelMsg, ok := ao.messages[2].(*TextMessageItem)
	if !ok {
		t.Fatal("expected third message to be TextMessageItem for cancel")
	}
	if !containsSubstring(cancelMsg.content, "canceled") {
		t.Errorf("expected cancel message to contain 'canceled', got: %s", cancelMsg.content)
	}

	// Total should be 3 messages: thinking, info, cancel
	if len(ao.messages) != 3 {
		t.Errorf("expected exactly 3 messages for canceled finish, got %d", len(ao.messages))
	}
}

func TestAgentOutput_AppendFinish_StopsSpinner(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Start streaming (which starts spinner)
	ao.AppendThinking("Processing...")
	ao.isStreaming = true
	ao.spinner = &GradientSpinner{label: "Thinking..."}

	// Verify spinner is active
	if !ao.isStreaming {
		t.Error("expected streaming to be active")
	}
	if ao.spinner == nil {
		t.Error("expected spinner to be present")
	}

	// Call AppendFinish
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 3000000000,
	}
	ao.AppendFinish(finishMsg)

	// Verify spinner was stopped
	if ao.isStreaming {
		t.Error("expected streaming to be stopped after AppendFinish")
	}
	if ao.spinner != nil {
		t.Error("expected spinner to be nil after AppendFinish")
	}
}

func TestAgentOutput_AppendFinish_NoThinkingMessage(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// No thinking message - just call AppendFinish
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 5000000000,
	}
	ao.AppendFinish(finishMsg)

	// Should still append InfoMessageItem
	if len(ao.messages) < 1 {
		t.Fatal("expected at least 1 message (info)")
	}
	infoMsg, ok := ao.messages[0].(*InfoMessageItem)
	if !ok {
		t.Fatal("expected first message to be InfoMessageItem")
	}
	if infoMsg.model != finishMsg.Model {
		t.Errorf("expected info model to be %s, got %s", finishMsg.Model, infoMsg.model)
	}
}

func TestAgentOutput_AppendFinish_CancelsPendingTools(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Add some tool calls in different states
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool1",
		Title:      "bash",
		Status:     "pending",
		Input:      map[string]any{"command": "ls -la"},
	})
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool2",
		Title:      "read",
		Status:     "in_progress",
		Input:      map[string]any{"filePath": "file.txt"},
	})
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool3",
		Title:      "write",
		Status:     "completed",
		Input:      map[string]any{"filePath": "output.txt"},
		Output:     "success",
	})

	// Call AppendFinish with cancelled reason
	finishMsg := AgentFinishMsg{
		Reason:   "cancelled",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 2000000000,
	}
	ao.AppendFinish(finishMsg)

	// Verify tool1 (pending) is now canceled
	tool1, ok := ao.messages[0].(*ToolMessageItem)
	if !ok {
		t.Fatal("expected first message to be ToolMessageItem")
	}
	if tool1.status != ToolStatusCanceled {
		t.Errorf("expected tool1 status to be canceled, got %d", tool1.status)
	}

	// Verify tool2 (in_progress) is now canceled
	tool2, ok := ao.messages[1].(*ToolMessageItem)
	if !ok {
		t.Fatal("expected second message to be ToolMessageItem")
	}
	if tool2.status != ToolStatusCanceled {
		t.Errorf("expected tool2 status to be canceled, got %d", tool2.status)
	}

	// Verify tool3 (completed) remains completed
	tool3, ok := ao.messages[2].(*ToolMessageItem)
	if !ok {
		t.Fatal("expected third message to be ToolMessageItem")
	}
	if tool3.status != ToolStatusSuccess {
		t.Errorf("expected tool3 status to remain success, got %d", tool3.status)
	}

	// Verify cancel message was appended (it's the last message after info)
	if len(ao.messages) < 5 {
		t.Errorf("expected at least 5 messages (3 tools, info, cancel), got %d", len(ao.messages))
	}
}

// --- Input Field Tests ---

func TestAgentOutput_InputFieldManagement(t *testing.T) {
	ao := NewAgentOutput()
	ao.UpdateSize(80, 20)

	// Test SetInputFocused
	ao.SetInputFocused(true)
	if !ao.input.Focused() {
		t.Error("expected input focused")
	}

	ao.SetInputFocused(false)
	if ao.input.Focused() {
		t.Error("expected input unfocused")
	}

	// Test SetValue/InputValue
	ao.input.SetValue("test message")
	if ao.InputValue() != "test message" {
		t.Errorf("expected 'test message', got %q", ao.InputValue())
	}

	// Test ResetInput
	ao.ResetInput()
	if ao.InputValue() != "" {
		t.Errorf("expected empty after reset, got %q", ao.InputValue())
	}
}

// --- Helper Functions ---

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
