package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// TestMessageCollapsed_Text tests TextMessageItem in collapsed state (always expanded, same as expanded test)
func TestMessageCollapsed_Text(t *testing.T) {
	msg := &TextMessageItem{
		id: "text-1",
		content: `This is a text message from the assistant.

It can contain multiple paragraphs with **markdown** formatting.

Here's a code block:
` + "```go\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```" + `

And some more text after the code.`,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_text.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageCollapsed_User tests UserMessageItem in collapsed state (always expanded, same as expanded test)
func TestMessageCollapsed_User(t *testing.T) {
	msg := &UserMessageItem{
		id:      "user-1",
		content: "This is a user message. It should be right-aligned with a different border style.",
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_user.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageCollapsed_Thinking tests ThinkingMessageItem in collapsed state
func TestMessageCollapsed_Thinking(t *testing.T) {
	// Create thinking message with many lines (>10)
	var content strings.Builder
	for i := 1; i <= 15; i++ {
		if i > 1 {
			content.WriteString("\n")
		}
		content.WriteString("Thinking line ")
		content.WriteString(string(rune('0' + i)))
		content.WriteString(": analyzing the problem and considering various approaches.")
	}

	msg := &ThinkingMessageItem{
		id:        "thinking-1",
		content:   content.String(),
		collapsed: true, // COLLAPSED - should show last 10 lines with truncation hint
		duration:  3500 * time.Millisecond,
		finished:  true,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_thinking.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageCollapsed_Tool tests ToolMessageItem in collapsed state
func TestMessageCollapsed_Tool(t *testing.T) {
	// Create tool message with long output (>10 lines)
	var output strings.Builder
	for i := 1; i <= 20; i++ {
		if i > 1 {
			output.WriteString("\n")
		}
		output.WriteString("Output line ")
		output.WriteString(string(rune('0' + (i % 10))))
		output.WriteString(": This is a line of tool output that demonstrates the collapsed view.")
	}

	msg := &ToolMessageItem{
		id:       "tool-1",
		toolName: "Read",
		kind:     "read",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"filePath": "/path/to/file.go",
		},
		output:   output.String(),
		expanded: false, // COLLAPSED - should show first 10 lines with truncation hint
		maxLines: 10,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_tool.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageCollapsed_ToolWithDiff tests ToolMessageItem with file diff in collapsed state
func TestMessageCollapsed_ToolWithDiff(t *testing.T) {
	before := "package main\n\nfunc main() {\n\tprintln(\"old content\")\n}\n"
	after := "package main\n\nfunc main() {\n\tprintln(\"new content\")\n}\n"

	msg := &ToolMessageItem{
		id:       "tool-2",
		toolName: "Edit",
		kind:     "edit",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"filePath":  "/path/to/modified.go",
			"oldString": "old content",
			"newString": "new content",
		},
		fileDiff: &FileDiff{
			File:      "/path/to/modified.go",
			Before:    before,
			After:     after,
			Additions: 1,
			Deletions: 1,
		},
		expanded: false, // COLLAPSED (though diffs don't actually collapse)
		maxLines: 10,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_tool_diff.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageCollapsed_Info tests InfoMessageItem in collapsed state (always expanded, same as expanded test)
func TestMessageCollapsed_Info(t *testing.T) {
	msg := &InfoMessageItem{
		id:       "info-1",
		model:    "claude-sonnet-4-5",
		provider: "Anthropic",
		duration: 5250 * time.Millisecond,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_info.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageCollapsed_Subagent tests SubagentMessageItem in collapsed state (always fully rendered)
func TestMessageCollapsed_Subagent(t *testing.T) {
	msg := &SubagentMessageItem{
		id:           "subagent-1",
		subagentType: "codebase-analyzer",
		description:  "Analyze the codebase for patterns and usage examples",
		status:       ToolStatusSuccess,
		sessionID:    "session-abc123",
		// Note: SubagentMessageItem doesn't have expanded/maxLines fields
		// It's always fully rendered
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_subagent.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageCollapsed_Divider tests DividerMessageItem in collapsed state (always expanded, same as expanded test)
func TestMessageCollapsed_Divider(t *testing.T) {
	msg := &DividerMessageItem{
		id:        "divider-1",
		iteration: 3, // Show "Iteration #3" divider
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_collapsed_divider.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// compareGoldenCollapsed compares rendered output with golden file
