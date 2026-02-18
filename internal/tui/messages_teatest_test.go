package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// --- InfoMessageItem Tests ---

func TestInfoMessageItemTeatest_RenderFullInfo(t *testing.T) {
	t.Parallel()

	item := InfoMessageItem{
		id:       "test1",
		model:    "claude-3-5-sonnet",
		provider: "Anthropic",
		duration: 2*time.Second + 500*time.Millisecond,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "◇", "should contain diamond icon")
	require.Contains(t, result, "claude-3-5-sonnet", "should contain model")
	require.Contains(t, result, "via", "should contain 'via'")
	require.Contains(t, result, "Anthropic", "should contain provider")
	require.Contains(t, result, "⏱", "should contain timer icon")
	require.Contains(t, result, "2.5s", "should contain formatted duration")
}

func TestInfoMessageItemTeatest_RenderModelOnly(t *testing.T) {
	t.Parallel()

	item := InfoMessageItem{
		id:       "test2",
		model:    "gpt-4",
		provider: "",
		duration: 1 * time.Second,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "gpt-4", "should contain model")
	require.NotContains(t, result, "via", "should not contain 'via' when no provider")
	require.Contains(t, result, "⏱", "should contain timer icon")
	require.Contains(t, result, "1s", "should contain duration")
}

func TestInfoMessageItemTeatest_RenderCaching(t *testing.T) {
	t.Parallel()

	item := InfoMessageItem{
		id:       "test3",
		model:    "claude-3-5-sonnet",
		provider: "Anthropic",
		duration: 500 * time.Millisecond,
	}

	result1 := item.Render(80)
	result2 := item.Render(80)

	require.Equal(t, result1, result2, "cached render should match original")

	result3 := item.Render(100)
	require.NotEmpty(t, result3, "render with different width should not be empty")
}

// --- TextMessageItem Tests ---

func TestTextMessageItemTeatest_RenderShortText(t *testing.T) {
	t.Parallel()

	item := &TextMessageItem{
		id:      "test",
		content: "Hello world",
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "Hello", "should contain content")
	require.Contains(t, result, "world", "should contain content")
}

func TestTextMessageItemTeatest_RenderMultilineContent(t *testing.T) {
	t.Parallel()

	item := &TextMessageItem{
		id:      "test",
		content: "Line 1\nLine 2\nLine 3",
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Equal(t, 80, item.cachedWidth, "cached width should be set")

	height := item.Height()
	require.Greater(t, height, 0, "height should be positive")
}

func TestTextMessageItemTeatest_RenderWidthCapping(t *testing.T) {
	t.Parallel()

	item := &TextMessageItem{
		id:      "test",
		content: "Content",
	}

	// Wide terminal should render with capped width, while cachedWidth tracks the requested width
	result := item.Render(200)

	require.NotEmpty(t, result, "render should not be empty")
	require.Equal(t, 200, item.cachedWidth, "cached width should match requested")
	// Verify effective width capping by ensuring we don't use the full 200 width
	require.NotContains(t, result, strings.Repeat(" ", 150), "should not use uncapped width")
}

func TestTextMessageItemTeatest_CacheInvalidation(t *testing.T) {
	t.Parallel()

	item := &TextMessageItem{
		id:      "test",
		content: "This is test content that will be rendered",
	}

	// First render at width 80
	result1 := item.Render(80)
	require.NotEmpty(t, result1, "first render should not be empty")
	require.Equal(t, 80, item.cachedWidth, "cached width should be 80")

	// Second render at same width - should return cached
	result2 := item.Render(80)
	require.Equal(t, result1, result2, "cached render should match")

	// Third render at different width - should re-render
	result3 := item.Render(100)
	require.NotEmpty(t, result3, "re-render should not be empty")
	require.Equal(t, 100, item.cachedWidth, "cached width should update to 100")
}

func TestTextMessageItemTeatest_HeightCalculation(t *testing.T) {
	t.Parallel()

	item := &TextMessageItem{
		id:      "test",
		content: "Line 1\nLine 2\nLine 3",
	}

	item.Render(80)

	height := item.Height()
	expectedHeight := strings.Count(item.cachedRender, "\n") + 1

	require.Equal(t, expectedHeight, height, "height should match newline count")
}

// --- ThinkingMessageItem Tests ---

func TestThinkingMessageItemTeatest_RenderShortContentExpanded(t *testing.T) {
	t.Parallel()

	item := &ThinkingMessageItem{
		id:        "test",
		content:   "Line 1\nLine 2\nLine 3",
		collapsed: false,
		duration:  0,
		finished:  false,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.NotContains(t, result, "lines hidden", "should not show truncation for short content")
}

func TestThinkingMessageItemTeatest_RenderShortContentCollapsed(t *testing.T) {
	t.Parallel()

	item := &ThinkingMessageItem{
		id:        "test",
		content:   "Line 1\nLine 2\nLine 3",
		collapsed: true,
		duration:  0,
		finished:  false,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.NotContains(t, result, "lines hidden", "should not truncate <=10 lines")
}

func TestThinkingMessageItemTeatest_RenderLongContentCollapsed(t *testing.T) {
	t.Parallel()

	content := "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15"
	item := &ThinkingMessageItem{
		id:        "test",
		content:   content,
		collapsed: true,
		duration:  0,
		finished:  false,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "lines hidden", "should show truncation hint for >10 lines")
}

func TestThinkingMessageItemTeatest_RenderLongContentExpanded(t *testing.T) {
	t.Parallel()

	content := "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15"
	item := &ThinkingMessageItem{
		id:        "test",
		content:   content,
		collapsed: false,
		duration:  0,
		finished:  false,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.NotContains(t, result, "lines hidden", "should not show truncation when expanded")
}

func TestThinkingMessageItemTeatest_RenderFinishedWithDuration(t *testing.T) {
	t.Parallel()

	item := &ThinkingMessageItem{
		id:        "test",
		content:   "Thinking...",
		collapsed: false,
		duration:  2 * time.Second,
		finished:  true,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "Thought for", "should show footer when finished")
	require.Contains(t, result, "2s", "should show duration in footer")
}

func TestThinkingMessageItemTeatest_ToggleExpanded(t *testing.T) {
	t.Parallel()

	content := "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12"
	item := &ThinkingMessageItem{
		id:        "test",
		content:   content,
		collapsed: true,
	}

	// Initial render (collapsed)
	result1 := item.Render(80)
	require.Contains(t, result1, "lines hidden", "should show truncation when collapsed")

	// Toggle to expanded
	item.ToggleExpanded()
	require.True(t, item.IsExpanded(), "should be expanded after toggle")
	require.Equal(t, 0, item.cachedWidth, "cache should be invalidated")

	// Render expanded
	result2 := item.Render(80)
	require.NotContains(t, result2, "lines hidden", "should not show truncation when expanded")
	require.NotEqual(t, result1, result2, "collapsed and expanded renders should differ")

	// Toggle back to collapsed
	item.ToggleExpanded()
	require.False(t, item.IsExpanded(), "should be collapsed after second toggle")

	result3 := item.Render(80)
	require.Contains(t, result3, "lines hidden", "should show truncation when collapsed again")
}

// --- DividerMessageItem Tests ---

func TestDividerMessageItemTeatest_RenderStandardWidth(t *testing.T) {
	t.Parallel()

	item := &DividerMessageItem{
		id:        "test",
		iteration: 1,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, " Iteration #", "should contain iteration label")
	require.Contains(t, result, "─", "should contain horizontal rule character")
	require.Equal(t, 80, item.cachedWidth, "cached width should be set")
}

func TestDividerMessageItemTeatest_RenderVaryingWidths(t *testing.T) {
	t.Parallel()

	item := &DividerMessageItem{
		id:        "test",
		iteration: 5,
	}

	result80 := item.Render(80)
	result120 := item.Render(120)

	require.NotEmpty(t, result80, "render at 80 should not be empty")
	require.NotEmpty(t, result120, "render at 120 should not be empty")
	require.NotEqual(t, result80, result120, "different widths should produce different renders")
}

func TestDividerMessageItemTeatest_RenderCentering(t *testing.T) {
	t.Parallel()

	item := &DividerMessageItem{
		id:        "test",
		iteration: 5,
	}

	result := item.Render(80)

	require.Contains(t, result, " Iteration #5 ", "should contain centered label")
	ruleCount := strings.Count(result, "─")
	require.Greater(t, ruleCount, 10, "should have multiple horizontal rule characters")
}

func TestDividerMessageItemTeatest_RenderNarrowWidth(t *testing.T) {
	t.Parallel()

	item := &DividerMessageItem{
		id:        "test",
		iteration: 100,
	}

	result := item.Render(15)

	require.NotEmpty(t, result, "render should not be empty for narrow width")
	require.Contains(t, result, "Iteration", "should contain iteration label")
	require.Contains(t, result, "─", "should contain horizontal rules")
}

func TestDividerMessageItemTeatest_Height(t *testing.T) {
	t.Parallel()

	item := &DividerMessageItem{
		id:        "test",
		iteration: 3,
	}

	item.Render(80)
	height := item.Height()

	require.Greater(t, height, 0, "height should be positive")
}

// --- formatDuration Tests ---

func TestFormatDurationTeatest_Milliseconds(t *testing.T) {
	t.Parallel()

	got := formatDuration(500 * time.Millisecond)
	require.Equal(t, "500ms", got, "should format milliseconds")
}

func TestFormatDurationTeatest_Seconds(t *testing.T) {
	t.Parallel()

	got := formatDuration(2*time.Second + 500*time.Millisecond)
	require.Equal(t, "2.5s", got, "should format seconds with decimal")
}

func TestFormatDurationTeatest_Minutes(t *testing.T) {
	t.Parallel()

	got := formatDuration(2*time.Minute + 30*time.Second)
	require.Equal(t, "2m30s", got, "should format minutes and seconds")
}

// --- formatToolParams Tests ---

func TestFormatToolParamsTeatest_EmptyParams(t *testing.T) {
	t.Parallel()

	got := formatToolParams(map[string]any{}, 100)
	require.Equal(t, "", got, "empty params should return empty string")
}

func TestFormatToolParamsTeatest_CommandOnly(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"command": "git status",
	}
	got := formatToolParams(input, 100)

	require.Equal(t, "git status", got, "should return command only")
}

func TestFormatToolParamsTeatest_FilePathOnly(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"filePath": "/path/to/file.go",
	}
	got := formatToolParams(input, 100)

	require.Equal(t, "/path/to/file.go", got, "should return filePath only")
}

func TestFormatToolParamsTeatest_CommandWithAdditionalParams(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"command": "npm install",
		"workdir": "/app",
		"timeout": 5000,
	}
	got := formatToolParams(input, 100)

	require.True(t, strings.HasPrefix(got, "npm install ("), "should start with command")
	require.Contains(t, got, "workdir=/app", "should contain workdir param")
	require.Contains(t, got, "timeout=5000", "should contain timeout param")
}

func TestFormatToolParamsTeatest_Truncation(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"command": "very long command that exceeds the maximum width limit",
	}
	got := formatToolParams(input, 30)

	require.LessOrEqual(t, len(got), 30, "result should not exceed maxWidth")
	require.True(t, strings.HasSuffix(got, "..."), "should end with ellipsis when truncated")
}

func TestFormatToolParamsTeatest_NonPrimaryParams(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"pattern": "*.go",
		"path":    "/src",
	}
	got := formatToolParams(input, 100)

	require.True(t, strings.HasPrefix(got, "("), "should start with parenthesis")
	require.True(t, strings.HasSuffix(got, ")"), "should end with parenthesis")
	require.Contains(t, got, "pattern=*.go", "should contain pattern param")
	require.Contains(t, got, "path=/src", "should contain path param")
}

// --- ToolMessageItem Tests ---

func TestToolMessageItemTeatest_RenderPending(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "test1",
		toolName: "Bash",
		status:   ToolStatusPending,
		input: map[string]any{
			"command": "ls -la",
		},
		output:   "",
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "●", "should contain pending icon")
	require.Contains(t, result, "Bash", "should contain tool name")
}

func TestToolMessageItemTeatest_RenderSuccess(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "test2",
		toolName: "Read",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"filePath": "/path/to/file.go",
		},
		output:   "line 1\nline 2\nline 3",
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "✓", "should contain success icon")
	require.Contains(t, result, "Read", "should contain tool name")
}

func TestToolMessageItemTeatest_RenderError(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "test3",
		toolName: "Bash",
		status:   ToolStatusError,
		input: map[string]any{
			"command": "invalid-command",
		},
		output:   "command not found: invalid-command",
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "×", "should contain error icon")
	require.Contains(t, result, "Bash", "should contain tool name")
}

func TestToolMessageItemTeatest_RenderCollapsedWithTruncation(t *testing.T) {
	t.Parallel()

	output := "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15"
	item := ToolMessageItem{
		id:       "test4",
		toolName: "Bash",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"command": "cat file.txt",
		},
		output:   output,
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "✓", "should contain success icon")
	require.Contains(t, result, "more lines", "should show truncation hint")
}

func TestToolMessageItemTeatest_RenderExpandedNoTruncation(t *testing.T) {
	t.Parallel()

	output := "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15"
	item := ToolMessageItem{
		id:       "test5",
		toolName: "Bash",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"command": "cat file.txt",
		},
		output:   output,
		expanded: true,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.NotContains(t, result, "more lines", "should not show truncation when expanded")
}

func TestToolMessageItemTeatest_ToggleExpanded(t *testing.T) {
	t.Parallel()

	output := "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12"
	item := &ToolMessageItem{
		id:       "test",
		toolName: "Bash",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"command": "cat file.txt",
		},
		output:   output,
		expanded: false,
		maxLines: 10,
	}

	// Initial render (collapsed)
	result1 := item.Render(80)
	require.Contains(t, result1, "more lines", "should show truncation when collapsed")

	// Toggle to expanded
	item.ToggleExpanded()
	require.True(t, item.IsExpanded(), "should be expanded after toggle")
	require.Equal(t, 0, item.cachedWidth, "cache should be invalidated")

	// Render expanded
	result2 := item.Render(80)
	require.NotContains(t, result2, "more lines", "should not show truncation when expanded")
	require.NotEqual(t, result1, result2, "collapsed and expanded renders should differ")

	// Toggle back to collapsed
	item.ToggleExpanded()
	require.False(t, item.IsExpanded(), "should be collapsed after second toggle")

	result3 := item.Render(80)
	require.Contains(t, result3, "more lines", "should show truncation when collapsed again")
}

func TestToolMessageItemTeatest_IconPerStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status     ToolStatus
		expectIcon string
	}{
		{ToolStatusPending, "●"},
		{ToolStatusRunning, "●"},
		{ToolStatusSuccess, "✓"},
		{ToolStatusError, "×"},
		{ToolStatusCanceled, "×"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			item := &ToolMessageItem{
				id:       "test",
				toolName: "TestTool",
				status:   tt.status,
				input:    map[string]any{"command": "test"},
			}

			result := item.Render(80)

			require.Contains(t, result, tt.expectIcon, "should contain expected icon for status")
		})
	}
}

func TestToolMessageItemTeatest_OutputTruncationBoundary(t *testing.T) {
	t.Parallel()

	lines := make([]string, 15)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d", i+1)
	}
	output := strings.Join(lines, "\n")

	item := &ToolMessageItem{
		id:       "test",
		toolName: "Bash",
		status:   ToolStatusSuccess,
		input:    map[string]any{"command": "cat"},
		output:   output,
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	// Should show first 10 lines
	for i := 1; i <= 10; i++ {
		expected := fmt.Sprintf("Line %d", i)
		require.Contains(t, result, expected, "should contain line %d", i)
	}

	// Should NOT show lines beyond maxLines
	require.NotContains(t, result, "Line 11", "should not contain line 11")

	// Should show truncation hint with correct count
	require.Contains(t, result, "5 more lines", "should show correct truncation count")
}

// --- renderDiffBlock Tests ---

func TestRenderDiffBlockTeatest_SingleLineChange(t *testing.T) {
	t.Parallel()

	result := renderDiffBlock("old line", "new line", "", 80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "│", "should contain divider")

	lines := strings.Split(result, "\n")
	require.Len(t, lines, 1, "should have 1 row")
}

func TestRenderDiffBlockTeatest_MultiLineOldToSingleNew(t *testing.T) {
	t.Parallel()

	result := renderDiffBlock("line1\nline2\nline3", "replacement", "", 80)

	require.NotEmpty(t, result, "render should not be empty")
	lines := strings.Split(result, "\n")
	require.Len(t, lines, 3, "should have 3 rows (max of old/new line counts)")

	for i, line := range lines {
		require.Contains(t, line, "│", "row %d should contain divider", i)
	}
}

func TestRenderDiffBlockTeatest_SingleOldToMultiLineNew(t *testing.T) {
	t.Parallel()

	result := renderDiffBlock("original", "new1\nnew2\nnew3\nnew4", "", 80)

	require.NotEmpty(t, result, "render should not be empty")
	lines := strings.Split(result, "\n")
	require.Len(t, lines, 4, "should have 4 rows (max of old/new line counts)")
}

// --- renderDiagnostics Tests ---

func TestRenderDiagnosticsTeatest_SingleError(t *testing.T) {
	t.Parallel()

	output := `<diagnostics file="/path/to/file.go">
ERROR [153:21] undefined: conn
</diagnostics>`

	result := renderDiagnostics(output, 100)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "file.go", "should contain file path")
	require.Contains(t, result, "ERROR", "should contain ERROR text")
}

func TestRenderDiagnosticsTeatest_MultipleErrors(t *testing.T) {
	t.Parallel()

	output := `<diagnostics file="/path/to/file.go">
ERROR [153:21] undefined: conn
ERROR [153:38] undefined: sessID
</diagnostics>`

	result := renderDiagnostics(output, 100)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "file.go", "should contain file path")
}

func TestRenderDiagnosticsTeatest_NoDiagnosticsTag(t *testing.T) {
	t.Parallel()

	output := "just some text"
	result := renderDiagnostics(output, 100)

	require.Empty(t, result, "should return empty when no diagnostics tag")
}

func TestRenderDiagnosticsTeatest_EmptyDiagnostics(t *testing.T) {
	t.Parallel()

	output := `<diagnostics file="test.go">
</diagnostics>`

	result := renderDiagnostics(output, 100)

	require.Empty(t, result, "should return empty for empty diagnostics")
}

// --- ToolMessageItem Diff and Diagnostics Integration Tests ---

func TestToolMessageItemTeatest_EditDiffRendering(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "edit-1",
		toolName: "edit",
		kind:     "edit",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"filePath":  "/path/to/file.go",
			"oldString": "func old() {}",
			"newString": "func new() {\n\treturn nil\n}",
		},
		output:   "Edit applied successfully.",
		maxLines: 10,
	}

	result := item.Render(100)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "│", "should contain side-by-side divider")
	require.NotContains(t, result, "Edit applied successfully", "should suppress plain output text")
	require.Contains(t, result, "Edit", "should show tool name in header")
}

func TestToolMessageItemTeatest_DiagnosticsRendering(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "bash-1",
		toolName: "Bash",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"command": "go build ./...",
		},
		output: `<diagnostics file="/home/user/project/main.go">
ERROR [10:5] undefined: myVar
ERROR [15:12] cannot use string as int
</diagnostics>`,
		maxLines: 10,
	}

	result := item.Render(100)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "main.go", "should contain file path")
	require.Contains(t, result, "ERROR", "should contain ERROR text")
	require.Contains(t, result, "×", "should contain error icon")
}

func TestToolMessageItemTeatest_EditWithDiagnostics(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "edit-diag",
		toolName: "edit",
		kind:     "edit",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"filePath":  "/path/to/file.go",
			"oldString": "old code",
			"newString": "new code",
		},
		output: `Edit applied successfully.
<diagnostics file="/path/to/file.go">
ERROR [10:5] undefined: newVar
</diagnostics>`,
		maxLines: 10,
	}

	result := item.Render(120)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "│", "should contain diff divider")
	require.Contains(t, result, "ERROR", "should contain diagnostics")
	require.Contains(t, result, "file.go", "should contain file path from diagnostics")
}

// --- truncateLine Tests ---

func TestTruncateLineTeatest_ShortLine(t *testing.T) {
	t.Parallel()

	got := truncateLine("hello", 10)
	require.Equal(t, "hello", got, "short line should not be truncated")
}

func TestTruncateLineTeatest_ExactWidth(t *testing.T) {
	t.Parallel()

	got := truncateLine("hello", 5)
	require.Equal(t, "hello", got, "line at exact width should not be truncated")
}

func TestTruncateLineTeatest_NeedsTruncation(t *testing.T) {
	t.Parallel()

	got := truncateLine("hello world", 8)
	require.Equal(t, "hello w…", got, "line exceeding width should be truncated")
}

func TestTruncateLineTeatest_VeryNarrow(t *testing.T) {
	t.Parallel()

	got := truncateLine("hello", 3)
	require.Equal(t, "hel", got, "very narrow width should truncate without ellipsis")
}

func TestTruncateLineTeatest_Empty(t *testing.T) {
	t.Parallel()

	got := truncateLine("", 10)
	require.Equal(t, "", got, "empty line should remain empty")
}

// --- Width Caching Tests (All Message Types) ---

func TestWidthCachingTeatest_AllMessageTypes(t *testing.T) {
	t.Parallel()

	t.Run("TextMessageItem", func(t *testing.T) {
		item := &TextMessageItem{
			id:      "test",
			content: "This is test content that will be rendered",
		}

		// First render at width 80
		result1 := item.Render(80)
		require.NotEmpty(t, result1, "first render should not be empty")
		require.Equal(t, 80, item.cachedWidth, "cached width should be 80")
		require.NotEmpty(t, item.cachedRender, "cached render should be populated")

		// Second render at same width - should return cached
		result2 := item.Render(80)
		require.Equal(t, result1, result2, "cached render should match")

		// Third render at different width - should re-render
		result3 := item.Render(100)
		require.NotEmpty(t, result3, "re-render should not be empty")
		require.Equal(t, 100, item.cachedWidth, "cached width should update")
	})

	t.Run("ThinkingMessageItem", func(t *testing.T) {
		item := &ThinkingMessageItem{
			id:        "test",
			content:   "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12",
			collapsed: true,
			finished:  true,
			duration:  2 * time.Second,
		}

		result1 := item.Render(80)
		require.NotEmpty(t, result1, "first render should not be empty")
		require.Equal(t, 80, item.cachedWidth, "cached width should be 80")

		result2 := item.Render(80)
		require.Equal(t, result1, result2, "cached render should match")

		result3 := item.Render(120)
		require.Equal(t, 120, item.cachedWidth, "cached width should update")
		require.NotEmpty(t, result3, "re-render should not be empty")
	})

	t.Run("ToolMessageItem", func(t *testing.T) {
		item := &ToolMessageItem{
			id:       "test",
			toolName: "Bash",
			status:   ToolStatusSuccess,
			input:    map[string]any{"command": "ls -la"},
			output:   "file1.txt\nfile2.txt\nfile3.txt",
			expanded: false,
			maxLines: 10,
		}

		result1 := item.Render(80)
		require.NotEmpty(t, result1, "first render should not be empty")
		require.Equal(t, 80, item.cachedWidth, "cached width should be 80")

		result2 := item.Render(80)
		require.Equal(t, result1, result2, "cached render should match")

		result3 := item.Render(60)
		require.Equal(t, 60, item.cachedWidth, "cached width should update")
		require.NotEmpty(t, result3, "re-render should not be empty")
	})

	t.Run("InfoMessageItem", func(t *testing.T) {
		item := &InfoMessageItem{
			id:       "test",
			model:    "claude-3-5-sonnet",
			provider: "Anthropic",
			duration: 3 * time.Second,
		}

		result1 := item.Render(80)
		require.NotEmpty(t, result1, "first render should not be empty")
		require.Equal(t, 80, item.cachedWidth, "cached width should be 80")

		result2 := item.Render(80)
		require.Equal(t, result1, result2, "cached render should match")

		result3 := item.Render(100)
		require.Equal(t, 100, item.cachedWidth, "cached width should update")
		require.NotEmpty(t, result3, "re-render should not be empty")
	})

	t.Run("DividerMessageItem", func(t *testing.T) {
		item := &DividerMessageItem{
			id:        "test",
			iteration: 5,
		}

		result1 := item.Render(80)
		require.NotEmpty(t, result1, "first render should not be empty")
		require.Equal(t, 80, item.cachedWidth, "cached width should be 80")

		result2 := item.Render(80)
		require.Equal(t, result1, result2, "cached render should match")

		result3 := item.Render(120)
		require.Equal(t, 120, item.cachedWidth, "cached width should update")
		require.NotEmpty(t, result3, "re-render should not be empty")
		require.NotEqual(t, result1, result3, "divider should render differently at different widths")
	})
}

// --- Golden Tests for Error States ---

// TestToolMessageItemTeatest_RenderError_Golden tests tool error message rendering with golden file
func TestToolMessageItemTeatest_RenderError_Golden(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "tool-error-1",
		toolName: "mcp_bash",
		kind:     "bash",
		status:   ToolStatusError,
		input:    map[string]any{"command": "go build ./...", "description": "Run build command"},
		output:   "exit status 1: compilation failed\nundefined: missingFunction",
		expanded: true,
	}

	result := item.Render(testfixtures.TestTermWidth)

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 10)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 10)
	uv.NewStyledString(result).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "tool_message_error.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// TestRenderDiagnosticsTeatest_SingleError_Golden tests single diagnostic error rendering with golden file
func TestRenderDiagnosticsTeatest_SingleError_Golden(t *testing.T) {
	t.Parallel()

	output := `<diagnostics file="/home/user/project/internal/app/handler.go">
ERROR [153:21] undefined: conn
</diagnostics>`

	result := renderDiagnostics(output, testfixtures.TestTermWidth)

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 5)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 5)
	uv.NewStyledString(result).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "diagnostic_single_error.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// TestRenderDiagnosticsTeatest_MultipleErrors_Golden tests multiple diagnostic errors rendering with golden file
func TestRenderDiagnosticsTeatest_MultipleErrors_Golden(t *testing.T) {
	t.Parallel()

	output := `<diagnostics file="/home/user/project/internal/server/server.go">
ERROR [153:21] undefined: conn
ERROR [153:38] undefined: sessID
ERROR [175:15] cannot use result (variable of type *Result) as string value in return statement
</diagnostics>`

	result := renderDiagnostics(output, testfixtures.TestTermWidth)

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 8)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 8)
	uv.NewStyledString(result).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "diagnostic_multiple_errors.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// --- extractReadContent Tests ---

func TestExtractReadContent_TaggedFormat(t *testing.T) {
	t.Parallel()

	output := "<path>/some/file.go</path>\n<type>file</type>\n<content>1: package main\n2: func main() {}\n</content>"
	result := extractReadContent(output)

	require.Equal(t, "1: package main\n2: func main() {}", result)
}

func TestExtractReadContent_NoTags(t *testing.T) {
	t.Parallel()

	output := "line 1\nline 2\nline 3"
	result := extractReadContent(output)

	require.Equal(t, output, result, "should return input unchanged when no tags present")
}

func TestExtractReadContent_LegacyFileFormat(t *testing.T) {
	t.Parallel()

	output := "<file>\n00001| package main\n00002| func main() {}\n</file>"
	result := extractReadContent(output)

	require.Equal(t, output, result, "should return legacy format unchanged")
}

func TestExtractReadContent_ContentTagOnly(t *testing.T) {
	t.Parallel()

	output := "<content>1: hello\n2: world\n</content>"
	result := extractReadContent(output)

	require.Equal(t, "1: hello\n2: world", result)
}

// --- Tagged Read Format Rendering Tests ---

func TestToolMessageItem_TaggedReadFormat(t *testing.T) {
	t.Parallel()

	output := "<path>/path/to/file.go</path>\n<type>file</type>\n<content>1: package main\n2: \n3: func main() {\n</content>"

	item := ToolMessageItem{
		id:       "tagged-read-1",
		toolName: "Read",
		kind:     "read",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"filePath": "/path/to/file.go",
		},
		output:   output,
		expanded: true,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "Read", "should contain tool name")
	// Should NOT show the raw tags in the rendered output
	require.NotContains(t, result, "<path>", "should not contain raw <path> tag")
	require.NotContains(t, result, "<type>", "should not contain raw <type> tag")
	require.NotContains(t, result, "<content>", "should not contain raw <content> tag")
	require.NotContains(t, result, "</content>", "should not contain raw </content> tag")
}

func TestToolMessageItem_TaggedReadFormat_Truncation(t *testing.T) {
	t.Parallel()

	// Build a tagged output with 20 lines
	var contentLines []string
	for i := 1; i <= 20; i++ {
		contentLines = append(contentLines, fmt.Sprintf("%d: line %d content", i, i))
	}
	output := "<path>/path/to/file.go</path>\n<type>file</type>\n<content>" +
		strings.Join(contentLines, "\n") + "\n</content>"

	item := ToolMessageItem{
		id:       "tagged-read-trunc",
		toolName: "Read",
		kind:     "read",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"filePath": "/path/to/file.go",
		},
		output:   output,
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	// Should show truncation hint based on content lines only, not metadata
	require.Contains(t, result, "more lines", "should show truncation hint")
	require.Contains(t, result, "10 more lines", "should count only content lines for truncation")
}

// --- renderTodoList Tests ---

func TestRenderTodoList_BasicRendering(t *testing.T) {
	t.Parallel()

	todos := []any{
		map[string]any{"content": "First task", "status": "completed", "priority": "high"},
		map[string]any{"content": "Second task", "status": "in_progress", "priority": "high"},
		map[string]any{"content": "Third task", "status": "pending", "priority": "medium"},
	}

	result := renderTodoList(todos, 80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "First task", "should contain first todo content")
	require.Contains(t, result, "Second task", "should contain second todo content")
	require.Contains(t, result, "Third task", "should contain third todo content")
	require.Contains(t, result, "[✓]", "should contain completed checkbox")
	require.Contains(t, result, "[•]", "should contain in_progress checkbox")
	require.Contains(t, result, "[ ]", "should contain pending checkbox")
}

func TestRenderTodoList_CancelledStatus(t *testing.T) {
	t.Parallel()

	todos := []any{
		map[string]any{"content": "Cancelled task", "status": "cancelled", "priority": "low"},
	}

	result := renderTodoList(todos, 80)

	require.Contains(t, result, "[✗]", "should contain cancelled checkbox")
	require.Contains(t, result, "Cancelled task", "should contain task content")
}

func TestRenderTodoList_EmptyList(t *testing.T) {
	t.Parallel()

	result := renderTodoList([]any{}, 80)
	require.Empty(t, result, "empty todo list should produce empty output")
}

func TestRenderTodoList_LongContentWraps(t *testing.T) {
	t.Parallel()

	todos := []any{
		map[string]any{
			"content":  "This is a very long todo item that should wrap to multiple lines when rendered at a narrow width",
			"status":   "pending",
			"priority": "high",
		},
	}

	result := renderTodoList(todos, 40)

	require.NotEmpty(t, result, "render should not be empty")
	lines := strings.Split(result, "\n")
	require.Greater(t, len(lines), 1, "long content should wrap to multiple lines")
}

// --- TodoWrite ToolMessageItem Integration Tests ---

func TestToolMessageItem_TodoWriteRendering(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "todo-1",
		toolName: "Todowrite",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"todos": []any{
				map[string]any{"content": "Research existing code", "status": "completed", "priority": "high"},
				map[string]any{"content": "Implement feature", "status": "in_progress", "priority": "high"},
				map[string]any{"content": "Write tests", "status": "pending", "priority": "medium"},
			},
		},
		output:   `[{"content":"Research existing code","status":"completed","priority":"high"}]`,
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	require.NotEmpty(t, result, "render should not be empty")
	require.Contains(t, result, "Todos", "should contain friendly display name in header")
	// Should render formatted todos, not raw JSON
	require.Contains(t, result, "[✓]", "should contain completed checkbox")
	require.Contains(t, result, "[•]", "should contain in_progress checkbox")
	require.Contains(t, result, "[ ]", "should contain pending checkbox")
	require.Contains(t, result, "Research existing code", "should contain todo content")
	// Should NOT show raw JSON output
	require.NotContains(t, result, `"content":`, "should not contain raw JSON")
}

func TestToolMessageItem_TodoWriteNoParams(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "todo-2",
		toolName: "Todowrite",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"todos": []any{
				map[string]any{"content": "Do something", "status": "pending", "priority": "high"},
			},
		},
		output:   "[]",
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	// Header should NOT contain raw todos param
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)
	header := lines[0]
	require.NotContains(t, header, "todos=", "header should not show raw todos param")
}

// --- toolDisplayName Tests ---

func TestToolDisplayName_IteratrTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"iteratr-tools_task-add", "Task Add"},
		{"iteratr-tools_task-update", "Task Update"},
		{"iteratr-tools_task-list", "Task List"},
		{"iteratr-tools_task-next", "Next Task"},
		{"iteratr-tools_note-add", "Note Add"},
		{"iteratr-tools_note-list", "Note List"},
		{"iteratr-tools_iteration-summary", "Iteration Summary"},
		{"iteratr-tools_session-complete", "Session Complete"},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expected, toolDisplayName(tt.input), "toolDisplayName(%q)", tt.input)
	}
}

func TestToolDisplayName_SpecTools(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Ask Questions", toolDisplayName("iteratr-spec_ask-questions"))
	require.Equal(t, "Finish Spec", toolDisplayName("iteratr-spec_finish-spec"))
}

func TestToolDisplayName_TodoWrite(t *testing.T) {
	t.Parallel()

	// All casing variants should map to "Todos"
	require.Equal(t, "Todos", toolDisplayName("TodoWrite"))
	require.Equal(t, "Todos", toolDisplayName("Todowrite"))
	require.Equal(t, "Todos", toolDisplayName("todowrite"))
	require.Equal(t, "Todos", toolDisplayName("mcp_todowrite"))
}

func TestToolDisplayName_Fallback(t *testing.T) {
	t.Parallel()

	// Unknown tools should get first-letter capitalization
	require.Equal(t, "Bash", toolDisplayName("bash"))
	require.Equal(t, "Read", toolDisplayName("read"))
	require.Equal(t, "Edit", toolDisplayName("edit"))
	require.Equal(t, "Mcp_bash", toolDisplayName("mcp_bash"))
	require.Equal(t, "", toolDisplayName(""))
}

func TestToolDisplayName_InRender(t *testing.T) {
	t.Parallel()

	item := ToolMessageItem{
		id:       "iteratr-1",
		toolName: "iteratr-tools_task-update",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"id":     "TAS-7",
			"status": "completed",
		},
		output:   "Updated task TAS-7: status=completed",
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	require.Contains(t, result, "Task Update", "should show friendly display name")
	require.NotContains(t, result, "iteratr-tools", "should not show raw server prefix")
}

// --- hookDisplayName Tests ---

func TestHookDisplayName_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"session_start", "Session Start"},
		{"pre_iteration", "Pre Iteration"},
		{"post_iteration", "Post Iteration"},
		{"session_end", "Session End"},
		{"on_task_complete", "Task Complete"},
		{"on_error", "On Error"},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expected, hookDisplayName(tt.input), "hookDisplayName(%q)", tt.input)
	}
}

func TestHookDisplayName_Fallback(t *testing.T) {
	t.Parallel()

	// Unknown hook types should pass through unchanged
	require.Equal(t, "custom_hook", hookDisplayName("custom_hook"))
}

func TestHookDisplayName_InRender(t *testing.T) {
	t.Parallel()

	item := HookMessageItem{
		id:       "hook-1",
		hookType: "pre_iteration",
		command:  "go test ./...",
		status:   HookStatusSuccess,
		maxLines: 10,
	}

	result := item.Render(80)

	require.Contains(t, result, "Pre Iteration", "should show friendly hook display name")
	require.NotContains(t, result, "pre_iteration", "should not show raw hook type")
}
