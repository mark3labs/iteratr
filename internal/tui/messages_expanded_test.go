package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// TestMessageExpanded_Text tests TextMessageItem in expanded state (always expanded)
func TestMessageExpanded_Text(t *testing.T) {
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
	goldenFile := filepath.Join("testdata", "message_expanded_text.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageExpanded_User tests UserMessageItem in expanded state (always expanded)
func TestMessageExpanded_User(t *testing.T) {
	msg := &UserMessageItem{
		id:      "user-1",
		content: "This is a user message. It should be right-aligned with a different border style.",
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_expanded_user.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageExpanded_Thinking tests ThinkingMessageItem in expanded state
func TestMessageExpanded_Thinking(t *testing.T) {
	// Create thinking message with many lines
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
		collapsed: false, // EXPANDED
		duration:  3500 * time.Millisecond,
		finished:  true,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_expanded_thinking.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageExpanded_Tool tests ToolMessageItem in expanded state
func TestMessageExpanded_Tool(t *testing.T) {
	// Create tool message with long output
	var output strings.Builder
	for i := 1; i <= 20; i++ {
		if i > 1 {
			output.WriteString("\n")
		}
		output.WriteString("Output line ")
		output.WriteString(string(rune('0' + (i % 10))))
		output.WriteString(": This is a line of tool output that demonstrates the expanded view.")
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
		expanded: true, // EXPANDED
		maxLines: 10,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_expanded_tool.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageExpanded_ToolWithDiff tests ToolMessageItem with file diff in expanded state
func TestMessageExpanded_ToolWithDiff(t *testing.T) {
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
		expanded: true, // EXPANDED
		maxLines: 10,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_expanded_tool_diff.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageExpanded_Info tests InfoMessageItem in expanded state (always expanded)
func TestMessageExpanded_Info(t *testing.T) {
	msg := &InfoMessageItem{
		id:       "info-1",
		model:    "claude-sonnet-4-5",
		provider: "Anthropic",
		duration: 5250 * time.Millisecond,
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_expanded_info.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageExpanded_Subagent tests SubagentMessageItem in expanded state
func TestMessageExpanded_Subagent(t *testing.T) {
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
	goldenFile := filepath.Join("testdata", "message_expanded_subagent.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestMessageExpanded_Divider tests DividerMessageItem in expanded state (always expanded)
func TestMessageExpanded_Divider(t *testing.T) {
	msg := &DividerMessageItem{
		id:        "divider-1",
		iteration: 3, // Show "Iteration #3" divider
	}

	// Render at test width
	rendered := msg.Render(testfixtures.TestTermWidth)

	// Verify golden file
	goldenFile := filepath.Join("testdata", "message_expanded_divider.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// compareGolden is deprecated - use testfixtures.CompareGolden instead
// Kept for backwards compatibility with old test files
