package tui

import (
	"fmt"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// TestSubagentModal_LoadingState tests the modal in loading state
func TestSubagentModal_LoadingState(t *testing.T) {
	modal := NewSubagentModal("session-123", "codebase-analyzer", "/test/workdir")

	// Modal should start in loading state
	if !modal.loading {
		t.Errorf("Modal should start in loading state")
	}

	// Render loading state
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "subagent_modal_loading.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)

	// Cleanup
	modal.Close()
}

// TestSubagentModal_ErrorState tests the modal in error state
func TestSubagentModal_ErrorState(t *testing.T) {
	modal := NewSubagentModal("session-999", "missing-session", "/test/workdir")

	// Set error state
	modal.loading = false
	modal.err = testfixtures.ErrSessionNotFound("session-999")

	// Render error state
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "subagent_modal_error.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)

	// Cleanup
	modal.Close()
}

// TestSubagentModal_ErrorStateStreamError tests the modal with stream error
func TestSubagentModal_ErrorStateStreamError(t *testing.T) {
	modal := NewSubagentModal("session-456", "test-agent", "/test/workdir")

	// Set stream error
	modal.loading = false
	modal.err = testfixtures.ErrStreamError()

	// Render error state
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "subagent_modal_error_stream.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)

	// Cleanup
	modal.Close()
}

// TestSubagentModal_ContentStateEmpty tests the modal with empty content
func TestSubagentModal_ContentStateEmpty(t *testing.T) {
	modal := NewSubagentModal("session-empty", "test-agent", "/test/workdir")

	// Set to content state with no messages
	modal.loading = false
	modal.err = nil

	// Render content state
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "subagent_modal_content_empty.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)

	// Cleanup
	modal.Close()
}

// TestSubagentModal_ContentStateWithMessages tests the modal with message content
func TestSubagentModal_ContentStateWithMessages(t *testing.T) {
	modal := NewSubagentModal("session-abc", "codebase-analyzer", "/test/workdir")

	// Set to content state
	modal.loading = false
	modal.err = nil

	// Add various message types
	modal.appendText("Analyzing codebase structure...")
	modal.appendThinking("Need to search for test files and patterns.")
	modal.appendToolCall(agent.ToolCallEvent{
		ToolCallID: "tool-1",
		Title:      "Grep",
		Status:     "success",
		Kind:       "grep",
		RawInput: map[string]any{
			"pattern": "func Test",
			"include": "**/*_test.go",
		},
		Output: "Found 42 test files",
	})
	modal.appendText("Found comprehensive test coverage across the codebase.")

	// Render content state
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "subagent_modal_content_messages.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)

	// Cleanup
	modal.Close()
}

// TestSubagentModal_ContentStateWithToolUpdate tests updating an existing tool call
func TestSubagentModal_ContentStateWithToolUpdate(t *testing.T) {
	modal := NewSubagentModal("session-update", "test-agent", "/test/workdir")

	// Set to content state
	modal.loading = false
	modal.err = nil

	// Add initial tool call
	modal.appendToolCall(agent.ToolCallEvent{
		ToolCallID: "tool-2",
		Title:      "Read",
		Status:     "running",
		Kind:       "read",
		RawInput: map[string]any{
			"filePath": "/path/to/file.go",
		},
	})

	// Update tool call with output
	modal.appendToolCall(agent.ToolCallEvent{
		ToolCallID: "tool-2",
		Title:      "Read",
		Status:     "success",
		Kind:       "read",
		Output:     "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n",
	})

	// Render content state
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "subagent_modal_content_tool_update.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)

	// Cleanup
	modal.Close()
}

// TestSubagentModal_ScrollKeysHandling tests keyboard scroll handling
func TestSubagentModal_ScrollKeysHandling(t *testing.T) {
	modal := NewSubagentModal("session-scroll", "test-agent", "/test/workdir")

	// Set to content state with many messages to enable scrolling
	modal.loading = false
	modal.err = nil

	// Add enough content to require scrolling
	for i := 0; i < 30; i++ {
		modal.appendText("Line of content for scroll testing\n")
	}

	// Send down arrow key
	modal.Update(tea.KeyPressMsg{Text: "down"})

	// Update should have been forwarded to scrollList
	// (We can't easily verify scroll position changed without exposing internals,
	// but we verify Update() doesn't panic and returns successfully)

	// Cleanup
	modal.Close()
}

// TestSubagentModal_HandleUpdateMessages tests HandleUpdate message processing
func TestSubagentModal_HandleUpdateMessages(t *testing.T) {
	modal := NewSubagentModal("session-update-msg", "test-agent", "/test/workdir")
	modal.loading = false
	modal.err = nil

	// Test SubagentTextMsg
	cmd := modal.HandleUpdate(SubagentTextMsg{Text: "Test text", Continue: true})
	if cmd == nil {
		t.Error("Expected command for continued stream")
	}
	if len(modal.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(modal.messages))
	}

	// Test SubagentThinkingMsg
	cmd = modal.HandleUpdate(SubagentThinkingMsg{Content: "Thinking...", Continue: true})
	if cmd == nil {
		t.Error("Expected command for continued stream")
	}
	if len(modal.messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(modal.messages))
	}

	// Test SubagentToolCallMsg
	cmd = modal.HandleUpdate(SubagentToolCallMsg{
		Event: agent.ToolCallEvent{
			ToolCallID: "tool-3",
			Title:      "TestTool",
			Status:     "success",
			Kind:       "test",
		},
		Continue: true,
	})
	if cmd == nil {
		t.Error("Expected command for continued stream")
	}
	if len(modal.messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(modal.messages))
	}

	// Test SubagentUserMsg (first one should be skipped)
	cmd = modal.HandleUpdate(SubagentUserMsg{Text: "First user message (skipped)", Continue: true})
	if cmd == nil {
		t.Error("Expected command for continued stream")
	}
	if len(modal.messages) != 3 {
		t.Errorf("First user message should be skipped, expected 3 messages, got %d", len(modal.messages))
	}

	// Second user message should be added
	cmd = modal.HandleUpdate(SubagentUserMsg{Text: "Second user message", Continue: true})
	if cmd == nil {
		t.Error("Expected command for continued stream")
	}
	if len(modal.messages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(modal.messages))
	}

	// Test SubagentStreamMsg with Continue=false
	cmd = modal.HandleUpdate(SubagentStreamMsg{Continue: false})
	if cmd != nil {
		t.Error("Expected nil command when Continue=false")
	}

	// Cleanup
	modal.Close()
}

// TestSubagentModal_AppendTextConcatenation tests that consecutive text appends are concatenated
func TestSubagentModal_AppendTextConcatenation(t *testing.T) {
	modal := NewSubagentModal("session-concat", "test-agent", "/test/workdir")
	modal.loading = false

	// Append text in chunks
	modal.appendText("Hello ")
	modal.appendText("world ")
	modal.appendText("test")

	// Should result in single TextMessageItem
	if len(modal.messages) != 1 {
		t.Errorf("Expected 1 message from concatenation, got %d", len(modal.messages))
	}

	textMsg, ok := modal.messages[0].(*TextMessageItem)
	if !ok {
		t.Fatalf("Expected TextMessageItem, got %T", modal.messages[0])
	}

	expected := "Hello world test"
	if textMsg.content != expected {
		t.Errorf("Expected content %q, got %q", expected, textMsg.content)
	}

	// Cleanup
	modal.Close()
}

// TestSubagentModal_AppendThinkingConcatenation tests that consecutive thinking appends are concatenated
func TestSubagentModal_AppendThinkingConcatenation(t *testing.T) {
	modal := NewSubagentModal("session-thinking", "test-agent", "/test/workdir")
	modal.loading = false

	// Append thinking in chunks
	modal.appendThinking("First thought. ")
	modal.appendThinking("Second thought. ")
	modal.appendThinking("Third thought.")

	// Should result in single ThinkingMessageItem
	if len(modal.messages) != 1 {
		t.Errorf("Expected 1 message from concatenation, got %d", len(modal.messages))
	}

	thinkingMsg, ok := modal.messages[0].(*ThinkingMessageItem)
	if !ok {
		t.Fatalf("Expected ThinkingMessageItem, got %T", modal.messages[0])
	}

	expected := "First thought. Second thought. Third thought."
	if thinkingMsg.content != expected {
		t.Errorf("Expected content %q, got %q", expected, thinkingMsg.content)
	}

	// Cleanup
	modal.Close()
}

// TestSubagentModal_HandleClickToggle tests click-to-toggle expandable messages
func TestSubagentModal_HandleClickToggle(t *testing.T) {
	modal := NewSubagentModal("session-click", "test-agent", "/test/workdir")
	modal.loading = false

	// Add a thinking message (expandable)
	modal.appendThinking("This is thinking content that can be toggled.")

	// Render to calculate viewport area
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Get initial expanded state
	thinkingMsg := modal.messages[0].(*ThinkingMessageItem)
	initialState := thinkingMsg.collapsed

	// Simulate click within viewport
	// Modal is centered, so calculate click position within content area
	clickX := modal.viewportArea.Min.X + 10 // Click 10 chars into content
	clickY := modal.viewportArea.Min.Y + 1  // Click on first content line

	modal.HandleClick(clickX, clickY)

	// State should have toggled
	if thinkingMsg.collapsed == initialState {
		t.Error("Expected message expanded state to toggle after click")
	}

	// Cleanup
	modal.Close()
}

// TestSubagentModal_HandleClickOutsideViewport tests that clicks outside viewport are ignored
func TestSubagentModal_HandleClickOutsideViewport(t *testing.T) {
	modal := NewSubagentModal("session-click-outside", "test-agent", "/test/workdir")
	modal.loading = false

	// Add a thinking message
	modal.appendThinking("Test content")

	// Render to calculate viewport area
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Render to calculate viewport area
	outsideArea := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	outsideScr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(outsideScr, outsideArea)

	thinkingMsg := modal.messages[0].(*ThinkingMessageItem)
	initialState := thinkingMsg.collapsed

	// Click outside viewport (before modal area)
	modal.HandleClick(0, 0)

	// State should NOT have changed
	if thinkingMsg.collapsed != initialState {
		t.Error("Expected message state to remain unchanged for click outside viewport")
	}

	// Cleanup
	modal.Close()
}

// TestSubagentModal_ScrollViewport tests scroll viewport functionality
func TestSubagentModal_ScrollViewport(t *testing.T) {
	modal := NewSubagentModal("session-scroll-viewport", "test-agent", "/test/workdir")
	modal.loading = false

	// Add enough content to enable scrolling
	for i := 0; i < 50; i++ {
		modal.appendText("Scrollable line of content\n")
	}

	// Enable auto-scroll initially
	modal.scrollList.SetAutoScroll(true)

	// Scroll up should disable auto-scroll
	modal.ScrollViewport(-5)
	if modal.scrollList.autoScroll {
		t.Error("Expected auto-scroll to be disabled after scrolling up")
	}

	// Scroll to bottom should re-enable auto-scroll
	modal.scrollList.GotoBottom()
	modal.ScrollViewport(1) // One more scroll down (already at bottom)
	if !modal.scrollList.autoScroll {
		t.Error("Expected auto-scroll to be re-enabled when at bottom")
	}

	// Cleanup
	modal.Close()
}

// Helper function for creating session not found error
// --- SubagentModal Command Execution Tests ---

func TestSubagentModal_Update_ForwardsToScrollList(t *testing.T) {
	t.Parallel()

	modal := NewSubagentModal("session-update-cmd", "test-agent", "/test/workdir")
	modal.loading = false

	// Add some messages to enable scrolling
	for i := 0; i < 20; i++ {
		modal.appendText(fmt.Sprintf("Message %d", i+1))
	}

	// Test that Update forwards to scrollList and may return commands
	testCases := []struct {
		name string
		msg  tea.Msg
	}{
		{"KeyPress j", tea.KeyPressMsg{Text: "j"}},
		{"KeyPress k", tea.KeyPressMsg{Text: "k"}},
		{"KeyPress down", tea.KeyPressMsg{Text: "down"}},
		{"KeyPress up", tea.KeyPressMsg{Text: "up"}},
		{"KeyPress pgdown", tea.KeyPressMsg{Text: "pgdown"}},
		{"KeyPress pgup", tea.KeyPressMsg{Text: "pgup"}},
		{"KeyPress G", tea.KeyPressMsg{Text: "G"}},
		{"KeyPress g", tea.KeyPressMsg{Text: "g"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := modal.Update(tc.msg)
			// ScrollList may or may not return a command
			// We just verify Update doesn't panic
			if cmd != nil {
				msg := cmd()
				_ = msg // Verify command can be executed
			}
		})
	}
}

func TestSubagentModal_Update_WhenScrollListNil(t *testing.T) {
	t.Parallel()

	modal := NewSubagentModal("session-update-nil", "test-agent", "/test/workdir")
	modal.scrollList = nil // Force nil scrollList

	// Update should return nil without panicking
	cmd := modal.Update(tea.KeyPressMsg{Text: "j"})
	if cmd != nil {
		t.Error("Update should return nil when scrollList is nil")
	}
}

func TestSubagentModal_Update_CommandExecution(t *testing.T) {
	t.Parallel()

	modal := NewSubagentModal("session-update-exec", "test-agent", "/test/workdir")
	modal.loading = false

	// Add messages
	for i := 0; i < 10; i++ {
		modal.appendText(fmt.Sprintf("Line %d", i+1))
	}

	// Send a scroll command and verify it can be executed
	cmd := modal.Update(tea.KeyPressMsg{Text: "j"})

	// Execute command if returned
	if cmd != nil {
		msg := cmd()
		// Verify it doesn't panic
		_ = msg
	}

	// Modal should still be functional
	if modal.scrollList == nil {
		t.Error("scrollList should not be nil after Update")
	}
}
