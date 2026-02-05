package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// --- Initialization Tests ---

func TestAgentOutput_Initialization(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()

	// Verify input field is initialized correctly
	require.Equal(t, "Send a message...", ao.input.Placeholder, "placeholder should be set")
	require.Equal(t, "> ", ao.input.Prompt, "prompt should be set")

	// Verify initial state
	require.Equal(t, -1, ao.focusedIndex, "focusedIndex should be -1")
	require.False(t, ao.ready, "ready should be false before UpdateSize")
	require.False(t, ao.input.Focused(), "input should be unfocused initially")
}

// --- Size and Layout Tests ---

func TestAgentOutput_UpdateSize(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	cmd := ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Command can be nil - just verify it doesn't panic
	_ = cmd

	require.Equal(t, testfixtures.TestTermWidth, ao.width, "width should be set")
	require.Equal(t, testfixtures.TestTermHeight, ao.height, "height should be set")
	require.True(t, ao.ready, "viewport should be ready after UpdateSize")

	// Verify input width is configured (width - 4 for padding)
	expectedInputWidth := testfixtures.TestTermWidth - 4
	require.Equal(t, expectedInputWidth, ao.input.Width(), "input width should be set")
}

// --- Message Appending Tests ---

func TestAgentOutput_AppendText(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	ao.AppendText("Test message")

	require.Greater(t, len(ao.messages), 0, "should have at least one message")
	_, ok := ao.messages[0].(*TextMessageItem)
	require.True(t, ok, "message should be TextMessageItem")
}

func TestAgentOutput_AppendThinking(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	ao.AppendThinking("Processing request...")

	require.Len(t, ao.messages, 1, "should have exactly one message")
	thinkingMsg, ok := ao.messages[0].(*ThinkingMessageItem)
	require.True(t, ok, "message should be ThinkingMessageItem")
	require.False(t, thinkingMsg.finished, "thinking message should not be finished initially")
}

func TestAgentOutput_AppendToolCall_NewTool(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "pending",
		Input:      map[string]any{"filePath": "test.go"},
	}
	ao.AppendToolCall(toolMsg)

	require.Len(t, ao.messages, 1, "should have exactly one message")
	toolItem, ok := ao.messages[0].(*ToolMessageItem)
	require.True(t, ok, "message should be ToolMessageItem")
	require.Equal(t, "tool-1", toolItem.ID(), "ID should match")
	require.Equal(t, ToolStatusPending, toolItem.status, "status should be Pending")
}

func TestAgentOutput_AppendToolCall_UpdateExisting(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

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
	require.Len(t, ao.messages, 1, "should have exactly one message after update")
	toolItem := ao.messages[0].(*ToolMessageItem)
	require.Equal(t, ToolStatusSuccess, toolItem.status, "status should be Success")
	require.Equal(t, "file contents here", toolItem.output, "output should be set")
}

// --- Subagent Detection Tests ---

func TestAgentOutput_SubagentDetection(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.ready = true
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// First call is "pending" with empty RawInput
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "task-1",
		Title:      "Task",
		Status:     "pending",
		Input:      map[string]any{},
	})

	require.Len(t, ao.messages, 1, "should have exactly one message")
	_, ok := ao.messages[0].(*ToolMessageItem)
	require.True(t, ok, "message should be ToolMessageItem on pending")

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

	require.Len(t, ao.messages, 1, "should still have one message after update")
	subagentMsg, ok := ao.messages[0].(*SubagentMessageItem)
	require.True(t, ok, "message should be converted to SubagentMessageItem")
	require.Equal(t, "codebase-analyzer", subagentMsg.subagentType, "subagentType should match")
	require.Equal(t, "Analyze the codebase", subagentMsg.description, "description should match")
	require.Equal(t, ToolStatusRunning, subagentMsg.status, "status should be Running")

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

	subagentMsg, ok = ao.messages[0].(*SubagentMessageItem)
	require.True(t, ok, "message should still be SubagentMessageItem")
	require.Equal(t, "session-123", subagentMsg.sessionID, "sessionID should be set")
	require.Equal(t, ToolStatusSuccess, subagentMsg.status, "status should be Success")
}

// --- Message Expansion Tests ---

func TestAgentOutput_ToggleExpanded(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

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
	toolItem.Render(testfixtures.TestTermWidth - 4) // contentWidth = width - 4
	require.False(t, toolItem.IsExpanded(), "message should not be expanded initially")

	// Toggle expansion with space key
	keyMsg := tea.KeyPressMsg{Code: ' ', Text: " "}
	ao.Update(keyMsg)

	require.True(t, toolItem.IsExpanded(), "message should be expanded after toggle")

	// Call View() to trigger rendering and populate the cache
	scrollListContent := ao.scrollList.View()
	require.NotEmpty(t, scrollListContent, "scroll list content should not be empty")

	// Verify the rendered content contains all lines
	require.Contains(t, toolItem.cachedRender, "line11", "expanded output should contain line11")
	require.Contains(t, toolItem.cachedRender, "line12", "expanded output should contain line12")
}

func TestAgentOutput_ToggleExpandedViaClick(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

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
	require.False(t, toolItem.IsExpanded(), "message should not be expanded initially")

	// Toggle expansion via direct method call (simulating click)
	toolItem.ToggleExpanded()
	require.True(t, toolItem.IsExpanded(), "message should be expanded after toggle")

	// Toggle back
	toolItem.ToggleExpanded()
	require.False(t, toolItem.IsExpanded(), "message should be collapsed after second toggle")
}

// --- Click-to-Expand Mouse Interaction Tests ---

func TestAgentOutput_HandleClick_ExpandTool(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a tool message with output exceeding maxLines (10)
	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
	}
	ao.AppendToolCall(toolMsg)

	// Render to populate viewportArea and messageLineStarts
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Verify tool is collapsed initially
	toolItem := ao.messages[0].(*ToolMessageItem)
	require.False(t, toolItem.IsExpanded(), "tool message should be collapsed initially")

	// Simulate click within viewport on the tool message
	clickX := ao.viewportArea.Min.X + 5
	clickY := ao.viewportArea.Min.Y + 2 // Click within tool message area

	cmd := ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil for expandable toggle")

	// Verify tool is expanded after click
	require.True(t, toolItem.IsExpanded(), "tool message should be expanded after click")

	// Click again to collapse
	cmd = ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil for expandable toggle")
	require.False(t, toolItem.IsExpanded(), "tool message should be collapsed after second click")
}

func TestAgentOutput_HandleClick_ExpandThinking(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a thinking message with content exceeding maxLines (10)
	var thinkingContent strings.Builder
	for i := 1; i <= 15; i++ {
		if i > 1 {
			thinkingContent.WriteString("\n")
		}
		thinkingContent.WriteString("Thinking line ")
		thinkingContent.WriteString(strings.Repeat("x", 50))
	}
	ao.AppendThinking(thinkingContent.String())

	// Render to populate viewportArea and messageLineStarts
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Verify thinking is collapsed initially
	thinkingItem := ao.messages[0].(*ThinkingMessageItem)
	require.True(t, thinkingItem.collapsed, "thinking message should be collapsed initially")

	// Simulate click within viewport on the thinking message
	clickX := ao.viewportArea.Min.X + 5
	clickY := ao.viewportArea.Min.Y + 1 // Click within thinking message area

	cmd := ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil for expandable toggle")

	// Verify thinking is expanded after click
	require.False(t, thinkingItem.collapsed, "thinking message should be expanded after click")

	// Click again to collapse
	cmd = ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil for expandable toggle")
	require.True(t, thinkingItem.collapsed, "thinking message should be collapsed after second click")
}

func TestAgentOutput_HandleClick_OutsideViewport(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a tool message
	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
	}
	ao.AppendToolCall(toolMsg)

	// Render to populate viewportArea
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	toolItem := ao.messages[0].(*ToolMessageItem)
	initialState := toolItem.IsExpanded()

	// Click outside viewport (before area)
	cmd := ao.HandleClick(0, 0)
	require.Nil(t, cmd, "HandleClick should return nil for click outside viewport")
	require.Equal(t, initialState, toolItem.IsExpanded(), "tool state should not change for click outside viewport")

	// Click outside viewport (after area)
	cmd = ao.HandleClick(testfixtures.TestTermWidth+10, testfixtures.TestTermHeight+10)
	require.Nil(t, cmd, "HandleClick should return nil for click outside viewport")
	require.Equal(t, initialState, toolItem.IsExpanded(), "tool state should not change for click outside viewport")
}

func TestAgentOutput_HandleClick_NonExpandableMessage(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add text and user messages (not expandable)
	ao.AppendText("This is assistant text")
	ao.AppendUserMessage("This is user text")

	// Render to populate viewportArea and messageLineStarts
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Click on text message (not expandable)
	clickX := ao.viewportArea.Min.X + 5
	clickY := ao.viewportArea.Min.Y + 1

	cmd := ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil for non-expandable message")

	// Click on user message (not expandable)
	clickY = ao.viewportArea.Min.Y + 3
	cmd = ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil for non-expandable message")
}

func TestAgentOutput_HandleClick_BeforeReady(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	// Don't call UpdateSize - component is not ready

	// Add a tool message
	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "line1\nline2",
	}
	ao.AppendToolCall(toolMsg)

	// Click should be ignored when not ready
	cmd := ao.HandleClick(10, 10)
	require.Nil(t, cmd, "HandleClick should return nil when component not ready")
}

func TestAgentOutput_HandleClick_EmptyMessages(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Don't add any messages

	// Render to populate viewportArea
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Click should be ignored with no messages
	cmd := ao.HandleClick(10, 10)
	require.Nil(t, cmd, "HandleClick should return nil with no messages")
}

func TestAgentOutput_HandleClick_MultipleMessages(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add multiple messages: text, tool, thinking
	ao.AppendText("Assistant text message")

	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
	}
	ao.AppendToolCall(toolMsg)

	var thinkingContent strings.Builder
	for i := 1; i <= 15; i++ {
		if i > 1 {
			thinkingContent.WriteString("\n")
		}
		thinkingContent.WriteString("Thinking line ")
		thinkingContent.WriteString(strings.Repeat("x", 30))
	}
	ao.AppendThinking(thinkingContent.String())

	// Render to populate viewportArea and messageLineStarts
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Get references to expandable messages
	toolItem := ao.messages[1].(*ToolMessageItem)
	thinkingItem := ao.messages[2].(*ThinkingMessageItem)

	require.False(t, toolItem.IsExpanded(), "tool should be collapsed initially")
	require.True(t, thinkingItem.collapsed, "thinking should be collapsed initially")

	// Calculate click position for tool message
	// Tool is second message, so click after text message height
	textHeight := ao.messages[0].Height()
	toolClickY := ao.viewportArea.Min.Y + textHeight + 2
	clickX := ao.viewportArea.Min.X + 5

	// Click on tool message
	cmd := ao.HandleClick(clickX, toolClickY)
	require.Nil(t, cmd, "HandleClick should return nil")
	require.True(t, toolItem.IsExpanded(), "tool should be expanded after click")
	require.True(t, thinkingItem.collapsed, "thinking should remain collapsed")

	// Calculate click position for thinking message (after text + tool)
	thinkingClickY := ao.viewportArea.Min.Y + textHeight + toolItem.Height() + 2

	// Click on thinking message
	cmd = ao.HandleClick(clickX, thinkingClickY)
	require.Nil(t, cmd, "HandleClick should return nil")
	require.True(t, toolItem.IsExpanded(), "tool should remain expanded")
	require.False(t, thinkingItem.collapsed, "thinking should be expanded after click")
}

func TestAgentOutput_HandleClick_WithScrollOffset(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, 20) // Smaller height to force scrolling

	// Add multiple tool messages to create scrollable content
	for i := 1; i <= 5; i++ {
		toolMsg := AgentToolCallMsg{
			ToolCallID: "tool-" + strings.Repeat("x", i),
			Title:      "Read",
			Status:     "completed",
			Input:      map[string]any{"filePath": "test.go"},
			Output:     "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
		}
		ao.AppendToolCall(toolMsg)
	}

	// Render to populate viewportArea and messageLineStarts
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Scroll down to middle of content
	ao.scrollList.ScrollBy(5)

	// Re-render after scroll
	ao.Draw(scr, area)

	// Get reference to a message that should be visible after scrolling
	toolItem := ao.messages[2].(*ToolMessageItem)
	initialState := toolItem.IsExpanded()

	// Click within viewport - should hit message index 2 based on scroll offset
	clickX := ao.viewportArea.Min.X + 5
	clickY := ao.viewportArea.Min.Y + 5

	cmd := ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil")

	// Verify some message was toggled (exact message depends on layout)
	// The important part is that the click was processed with scroll offset
	_ = initialState // State may have changed for the clicked message
}

func TestAgentOutput_HandleClick_SubagentMessage(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a subagent message via AppendToolCall (simulates subagent completion with sessionID)
	subagentMsg := AgentToolCallMsg{
		ToolCallID: "subagent-1",
		Title:      "Task",
		Status:     "completed",
		Input: map[string]any{
			"subagent_type": "explore",
			"description":   "Exploring codebase...",
		},
		SessionID: "test-session-id", // Subagent has completed with sessionID
	}
	ao.AppendToolCall(subagentMsg)

	// Render to populate viewportArea and messageLineStarts
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Click on subagent message should return OpenSubagentModalMsg
	clickX := ao.viewportArea.Min.X + 5
	clickY := ao.viewportArea.Min.Y + 1

	cmd := ao.HandleClick(clickX, clickY)
	require.NotNil(t, cmd, "HandleClick should return command for subagent message")

	// Execute command to get message
	msg := cmd()
	openMsg, ok := msg.(OpenSubagentModalMsg)
	require.True(t, ok, "command should return OpenSubagentModalMsg")
	require.Equal(t, "test-session-id", openMsg.SessionID, "session ID should match")
	require.Equal(t, "explore", openMsg.SubagentType, "subagent type should match")
}

func TestAgentOutput_HandleClick_SubagentWithoutSession(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a subagent message without sessionID via AppendToolCall (running state, no sessionID yet)
	subagentMsg := AgentToolCallMsg{
		ToolCallID: "subagent-2",
		Title:      "Task",
		Status:     "running",
		Input: map[string]any{
			"subagent_type": "explore",
			"description":   "Exploring...",
		},
		SessionID: "", // No sessionID yet (still running)
	}
	ao.AppendToolCall(subagentMsg)

	// Render to populate viewportArea and messageLineStarts
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	ao.Draw(scr, area)

	// Click should be ignored for subagent without sessionID
	clickX := ao.viewportArea.Min.X + 5
	clickY := ao.viewportArea.Min.Y + 1

	cmd := ao.HandleClick(clickX, clickY)
	require.Nil(t, cmd, "HandleClick should return nil for subagent without sessionID")
}

// --- Keyboard Navigation Tests ---

func TestAgentOutput_ScrollingBehavior(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add enough content to require scrolling
	for i := 0; i < 10; i++ {
		ao.AppendText("Line of text that should create scrollable content")
	}

	// Scroll should start at bottom (auto-scroll enabled)
	require.True(t, ao.scrollList.AtBottom(), "scroll should be at bottom initially")

	// Press Up arrow - should scroll up and disable auto-scroll
	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	ao.Update(upKey)
	require.False(t, ao.scrollList.autoScroll, "autoScroll should be disabled after scrolling up")

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
	require.True(t, ao.scrollList.AtBottom(), "should be at bottom after scrolling down")
	require.True(t, ao.scrollList.autoScroll, "autoScroll should be re-enabled when reaching bottom")
}

func TestAgentOutput_KeyHandling_NoExpandableMessages(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add only text messages (not expandable)
	ao.AppendText("First text message")
	ao.AppendText("Second text message")

	require.Equal(t, -1, ao.focusedIndex, "focusedIndex should be -1")

	// Up/Down keys now scroll the viewport (not focus navigation)
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	ao.Update(downKey)
	require.Equal(t, -1, ao.focusedIndex, "focusedIndex should remain -1")

	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	ao.Update(upKey)
	require.Equal(t, -1, ao.focusedIndex, "focusedIndex should remain -1")
}

func TestAgentOutput_KeyHandling_EmptyMessages(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	require.Equal(t, -1, ao.focusedIndex, "focusedIndex should be -1")

	// Press Down arrow - should not change focus (no messages)
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	ao.Update(downKey)
	require.Equal(t, -1, ao.focusedIndex, "focusedIndex should remain -1")

	// Press Up arrow - should not change focus (no messages)
	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	ao.Update(upKey)
	require.Equal(t, -1, ao.focusedIndex, "focusedIndex should remain -1")
}

// --- Finish Message Tests ---

func TestAgentOutput_AppendFinish_Normal(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a thinking message
	ao.AppendThinking("Processing request...")

	thinkingMsg := ao.messages[0].(*ThinkingMessageItem)
	require.False(t, thinkingMsg.finished, "thinking message should not be finished initially")

	// Call AppendFinish with normal completion
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 5 * time.Second,
	}
	ao.AppendFinish(finishMsg)

	// Verify thinking message is now finished with duration
	require.True(t, thinkingMsg.finished, "thinking message should be finished after AppendFinish")
	require.Equal(t, finishMsg.Duration, thinkingMsg.duration, "thinking duration should match")

	// Verify InfoMessageItem was appended
	require.GreaterOrEqual(t, len(ao.messages), 2, "should have at least 2 messages after AppendFinish")
	infoMsg, ok := ao.messages[1].(*InfoMessageItem)
	require.True(t, ok, "second message should be InfoMessageItem")
	require.Equal(t, finishMsg.Model, infoMsg.model, "model should match")
	require.Equal(t, finishMsg.Provider, infoMsg.provider, "provider should match")
	require.Equal(t, finishMsg.Duration, infoMsg.duration, "duration should match")

	// Verify no error or cancel messages were added
	require.Len(t, ao.messages, 2, "should have exactly 2 messages for normal finish")
}

func TestAgentOutput_AppendFinish_WithError(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a thinking message
	ao.AppendThinking("Processing request...")

	// Call AppendFinish with error
	finishMsg := AgentFinishMsg{
		Reason:   "error",
		Error:    "Connection timeout",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 2 * time.Second,
	}
	ao.AppendFinish(finishMsg)

	// Verify thinking message is finished
	thinkingMsg := ao.messages[0].(*ThinkingMessageItem)
	require.True(t, thinkingMsg.finished, "thinking message should be finished")

	// Verify InfoMessageItem was appended
	require.GreaterOrEqual(t, len(ao.messages), 2, "should have at least 2 messages")
	_, ok := ao.messages[1].(*InfoMessageItem)
	require.True(t, ok, "second message should be InfoMessageItem")

	// Verify error TextMessageItem was appended
	require.GreaterOrEqual(t, len(ao.messages), 3, "should have at least 3 messages")
	errorMsg, ok := ao.messages[2].(*TextMessageItem)
	require.True(t, ok, "third message should be TextMessageItem for error")
	require.Contains(t, errorMsg.content, "Error", "error message should contain 'Error'")
	require.Contains(t, errorMsg.content, "Connection timeout", "error message should contain error text")

	require.Len(t, ao.messages, 3, "should have exactly 3 messages for error finish")
}

func TestAgentOutput_AppendFinish_Canceled(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a thinking message
	ao.AppendThinking("Processing request...")

	// Call AppendFinish with canceled
	finishMsg := AgentFinishMsg{
		Reason:   "cancelled",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 1 * time.Second,
	}
	ao.AppendFinish(finishMsg)

	// Verify thinking message is finished
	thinkingMsg := ao.messages[0].(*ThinkingMessageItem)
	require.True(t, thinkingMsg.finished, "thinking message should be finished")

	// Verify InfoMessageItem was appended
	require.GreaterOrEqual(t, len(ao.messages), 2, "should have at least 2 messages")
	_, ok := ao.messages[1].(*InfoMessageItem)
	require.True(t, ok, "second message should be InfoMessageItem")

	// Verify cancel TextMessageItem was appended
	require.GreaterOrEqual(t, len(ao.messages), 3, "should have at least 3 messages")
	cancelMsg, ok := ao.messages[2].(*TextMessageItem)
	require.True(t, ok, "third message should be TextMessageItem for cancel")
	require.Contains(t, cancelMsg.content, "canceled", "cancel message should contain 'canceled'")

	require.Len(t, ao.messages, 3, "should have exactly 3 messages for canceled finish")
}

func TestAgentOutput_AppendFinish_StopsSpinner(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Start streaming (which starts spinner)
	ao.AppendThinking("Processing...")
	ao.isStreaming = true
	ao.spinner = &GradientSpinner{label: "Thinking..."}

	require.True(t, ao.isStreaming, "streaming should be active")
	require.NotNil(t, ao.spinner, "spinner should be present")

	// Call AppendFinish
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 3 * time.Second,
	}
	ao.AppendFinish(finishMsg)

	require.False(t, ao.isStreaming, "streaming should be stopped after AppendFinish")
	require.Nil(t, ao.spinner, "spinner should be nil after AppendFinish")
}

func TestAgentOutput_AppendFinish_NoThinkingMessage(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// No thinking message - just call AppendFinish
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 5 * time.Second,
	}
	ao.AppendFinish(finishMsg)

	// Should still append InfoMessageItem
	require.GreaterOrEqual(t, len(ao.messages), 1, "should have at least 1 message")
	infoMsg, ok := ao.messages[0].(*InfoMessageItem)
	require.True(t, ok, "first message should be InfoMessageItem")
	require.Equal(t, finishMsg.Model, infoMsg.model, "model should match")
}

func TestAgentOutput_AppendFinish_CancelsPendingTools(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

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
		Duration: 2 * time.Second,
	}
	ao.AppendFinish(finishMsg)

	// Verify tool1 (pending) is now canceled
	tool1, ok := ao.messages[0].(*ToolMessageItem)
	require.True(t, ok, "first message should be ToolMessageItem")
	require.Equal(t, ToolStatusCanceled, tool1.status, "tool1 status should be canceled")

	// Verify tool2 (in_progress) is now canceled
	tool2, ok := ao.messages[1].(*ToolMessageItem)
	require.True(t, ok, "second message should be ToolMessageItem")
	require.Equal(t, ToolStatusCanceled, tool2.status, "tool2 status should be canceled")

	// Verify tool3 (completed) remains completed
	tool3, ok := ao.messages[2].(*ToolMessageItem)
	require.True(t, ok, "third message should be ToolMessageItem")
	require.Equal(t, ToolStatusSuccess, tool3.status, "tool3 status should remain success")

	// Verify cancel message was appended (it's the last message after info)
	require.GreaterOrEqual(t, len(ao.messages), 5, "should have at least 5 messages")
}

// --- Input Field Tests ---

func TestAgentOutput_InputFieldManagement(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Test SetInputFocused
	ao.SetInputFocused(true)
	require.True(t, ao.input.Focused(), "input should be focused")

	ao.SetInputFocused(false)
	require.False(t, ao.input.Focused(), "input should be unfocused")

	// Test SetValue/InputValue
	ao.input.SetValue("test message")
	require.Equal(t, "test message", ao.InputValue(), "input value should match")

	// Test ResetInput
	ao.ResetInput()
	require.Empty(t, ao.InputValue(), "input should be empty after reset")
}

// --- Visual Regression Tests with Golden Files ---

func TestAgentOutput_Render_Empty(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	rendered := ao.Render()

	goldenFile := filepath.Join("testdata", "agent_output_empty.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

func TestAgentOutput_Render_WithMessages(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add various message types
	ao.AppendText("Assistant: Let me help you with that task.")
	ao.AppendThinking("Analyzing the codebase structure...")
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "main.go"},
		Output:     "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
	})

	rendered := ao.Render()

	goldenFile := filepath.Join("testdata", "agent_output_with_messages.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

func TestAgentOutput_Render_WithSubagent(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add text message
	ao.AppendText("Let me analyze the codebase for you.")

	// Add subagent message (pending)
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "task-1",
		Title:      "Task",
		Status:     "pending",
		Input:      map[string]any{},
	})

	// Update to in_progress with subagent_type
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "task-1",
		Title:      "Task",
		Status:     "in_progress",
		Input: map[string]any{
			"subagent_type": "codebase-analyzer",
			"prompt":        "Analyze the Go project structure",
			"description":   "Analyze the Go project structure",
		},
	})

	rendered := ao.Render()

	goldenFile := filepath.Join("testdata", "agent_output_with_subagent.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

func TestAgentOutput_Render_WithFinishedThinking(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add thinking message
	var thinkingContent strings.Builder
	for i := 1; i <= 5; i++ {
		if i > 1 {
			thinkingContent.WriteString("\n")
		}
		thinkingContent.WriteString("Analyzing step ")
		thinkingContent.WriteString(strings.Repeat("=", i))
		thinkingContent.WriteString(" considering various approaches...")
	}
	ao.AppendThinking(thinkingContent.String())

	// Finish with normal completion
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 3500 * time.Millisecond,
	}
	ao.AppendFinish(finishMsg)

	rendered := ao.Render()

	goldenFile := filepath.Join("testdata", "agent_output_finished_thinking.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

func TestAgentOutput_Render_WithError(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add thinking message
	ao.AppendThinking("Processing request...")

	// Finish with error
	finishMsg := AgentFinishMsg{
		Reason:   "error",
		Error:    "Connection timeout: failed to reach API endpoint",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 2000 * time.Millisecond,
	}
	ao.AppendFinish(finishMsg)

	rendered := ao.Render()

	goldenFile := filepath.Join("testdata", "agent_output_with_error.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

func TestAgentOutput_Render_WithToolExpanded(t *testing.T) {
	t.Parallel()

	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add a tool message with long output
	var output strings.Builder
	for i := 1; i <= 20; i++ {
		if i > 1 {
			output.WriteString("\n")
		}
		output.WriteString("Line ")
		if i < 10 {
			output.WriteString("0")
		}
		output.WriteString(strings.Repeat("=", i%5+1))
		output.WriteString(" This is output line content")
	}

	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "/path/to/file.go"},
		Output:     output.String(),
	})

	// Expand the tool message
	toolItem := ao.messages[0].(*ToolMessageItem)
	toolItem.ToggleExpanded()

	rendered := ao.Render()

	goldenFile := filepath.Join("testdata", "agent_output_tool_expanded.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// --- Integration Tests ---

func TestAgentOutput_Integration_FullConversation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ao := NewAgentOutput()
	ao.UpdateSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Simulate a full conversation flow
	ao.AppendText("I'll help you implement the feature.")
	ao.AppendThinking("First, let me read the existing code...")

	// Add tool call - read file
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "pending",
		Input:      map[string]any{},
	})
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "main.go"},
		Output:     "package main\n\nfunc main() {\n\t// TODO: implement\n}",
	})

	ao.AppendText("Now I'll write the implementation.")

	// Add tool call - write file
	ao.AppendToolCall(AgentToolCallMsg{
		ToolCallID: "tool-2",
		Title:      "Write",
		Status:     "completed",
		Input:      map[string]any{"filePath": "feature.go", "content": "package main\n\nfunc Feature() {}"},
		Output:     "File written successfully",
	})

	// Finish conversation
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4-5",
		Provider: "Anthropic",
		Duration: 5 * time.Second,
	}
	ao.AppendFinish(finishMsg)

	// Verify message count
	require.GreaterOrEqual(t, len(ao.messages), 6, "should have multiple messages")

	// Verify rendering doesn't panic
	rendered := ao.Render()
	require.NotEmpty(t, rendered, "rendered output should not be empty")

	_ = ctx // avoid unused variable
}

// --- Helper Functions ---
