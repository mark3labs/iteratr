package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestInfoMessageItemRender(t *testing.T) {
	tests := []struct {
		name     string
		item     InfoMessageItem
		width    int
		contains []string
	}{
		{
			name: "full info with model, provider, and duration",
			item: InfoMessageItem{
				id:       "test1",
				model:    "claude-3-5-sonnet",
				provider: "Anthropic",
				duration: 2*time.Second + 500*time.Millisecond,
			},
			width: 80,
			contains: []string{
				"◇",
				"claude-3-5-sonnet",
				"via",
				"Anthropic",
				"⏱",
				"2.5s",
			},
		},
		{
			name: "model only without provider",
			item: InfoMessageItem{
				id:       "test2",
				model:    "gpt-4",
				provider: "",
				duration: 1 * time.Second,
			},
			width: 80,
			contains: []string{
				"◇",
				"gpt-4",
				"⏱",
				"1s",
			},
		},
		{
			name: "duration only",
			item: InfoMessageItem{
				id:       "test3",
				model:    "",
				provider: "",
				duration: 500 * time.Millisecond,
			},
			width: 80,
			contains: []string{
				"◇",
				"⏱",
				"500ms",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.Render(tt.width)

			// Check that all expected substrings are present
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Render() result missing expected substring %q\nGot: %s", expected, result)
				}
			}

			// Verify caching works
			cachedResult := tt.item.Render(tt.width)
			if result != cachedResult {
				t.Errorf("Cached render differs from original render")
			}

			// Verify cache invalidates with different width
			differentWidth := tt.item.Render(tt.width + 10)
			if differentWidth == "" {
				t.Errorf("Render with different width returned empty string")
			}
		})
	}
}

func TestTextMessageItemRender(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		width        int
		expectCapped bool // whether width should be capped at 120
	}{
		{
			name:         "short text",
			content:      "Hello world",
			width:        80,
			expectCapped: false,
		},
		{
			name:         "text requiring word wrap",
			content:      "This is a very long line that should wrap at the appropriate width boundary when rendered",
			width:        50, // Wider to avoid edge case wrap issues
			expectCapped: false,
		},
		{
			name:         "wide terminal caps at 120",
			content:      "Content",
			width:        200, // Should be capped at 120
			expectCapped: true,
		},
		{
			name:         "multi-line content",
			content:      "Line 1\nLine 2\nLine 3",
			width:        80,
			expectCapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &TextMessageItem{
				id:      "test",
				content: tt.content,
			}

			result := item.Render(tt.width)

			// Verify result is not empty
			if result == "" {
				t.Errorf("Render() returned empty string")
			}

			// Verify caching works - second call with same width should return cached
			cachedResult := item.Render(tt.width)
			if result != cachedResult {
				t.Errorf("Cached render differs from original render")
			}

			// Verify cachedWidth is set correctly
			if item.cachedWidth != tt.width {
				t.Errorf("cachedWidth = %d, want %d", item.cachedWidth, tt.width)
			}

			// Verify cache invalidates with different width
			differentResult := item.Render(tt.width + 10)
			if differentResult == result && tt.width+10 != tt.width {
				// Result might be the same if both widths are above cap, but cache should still refresh
				if item.cachedWidth != tt.width+10 {
					t.Errorf("Cache not invalidated: cachedWidth = %d, want %d", item.cachedWidth, tt.width+10)
				}
			}

			// Verify effective width capping at 120
			if tt.expectCapped {
				// When width > 122 (120 + 2 for border), effective width should be 120
				// The content should be wrapped at 120, not at the full width
				// We can test this by checking that the result doesn't use the full width
				if strings.Contains(result, strings.Repeat(" ", 150)) {
					t.Errorf("Result appears to use uncapped width when it should be capped at 120")
				}
			}

			// Verify Height() returns a positive line count
			height := item.Height()
			if height <= 0 {
				t.Errorf("Height() = %d, want > 0", height)
			}

			// Height should match the newline count in the cached render
			expectedHeight := strings.Count(item.cachedRender, "\n") + 1
			if height != expectedHeight {
				t.Errorf("Height() = %d, want %d (based on cachedRender newlines)", height, expectedHeight)
			}
		})
	}
}

func TestTextMessageItemRenderBorderAndPadding(t *testing.T) {
	// Test that styleAssistantBorder is applied (left border + padding)
	item := &TextMessageItem{
		id:      "test",
		content: "Test content",
	}

	result := item.Render(80)

	// The result should contain ANSI codes for border/padding from lipgloss
	// With markdown rendering, the text may be split across ANSI codes,
	// so we check for both words separately
	if !strings.Contains(result, "Test") || !strings.Contains(result, "content") {
		t.Errorf("Render() result does not contain original content words: %s", result)
	}

	// Verify the result is longer than the original content due to styling
	if len(result) <= len("Test content") {
		t.Errorf("Render() result should be longer than content due to styling")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			want:     "500ms",
		},
		{
			name:     "seconds",
			duration: 2*time.Second + 500*time.Millisecond,
			want:     "2.5s",
		},
		{
			name:     "minutes",
			duration: 2*time.Minute + 30*time.Second,
			want:     "2m30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestThinkingMessageItemRender(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		collapsed         bool
		duration          time.Duration
		finished          bool
		width             int
		expectTruncation  bool
		expectFooter      bool
		expectedLineCount int // Approximate expected lines (for collapsed)
	}{
		{
			name:              "short content expanded",
			content:           "Line 1\nLine 2\nLine 3",
			collapsed:         false,
			duration:          0,
			finished:          false,
			width:             80,
			expectTruncation:  false,
			expectFooter:      false,
			expectedLineCount: 3,
		},
		{
			name:              "short content collapsed (<=10 lines, no truncation)",
			content:           "Line 1\nLine 2\nLine 3",
			collapsed:         true,
			duration:          0,
			finished:          false,
			width:             80,
			expectTruncation:  false,
			expectFooter:      false,
			expectedLineCount: 3,
		},
		{
			name:              "long content collapsed (>10 lines, shows last 10)",
			content:           "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15",
			collapsed:         true,
			duration:          0,
			finished:          false,
			width:             80,
			expectTruncation:  true, // Should show "… (5 lines hidden)"
			expectFooter:      false,
			expectedLineCount: 11, // 1 truncation line + 10 visible lines
		},
		{
			name:              "long content expanded (>10 lines, shows all)",
			content:           "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15",
			collapsed:         false,
			duration:          0,
			finished:          false,
			width:             80,
			expectTruncation:  false,
			expectFooter:      false,
			expectedLineCount: 15,
		},
		{
			name:              "finished with duration",
			content:           "Thinking...",
			collapsed:         false,
			duration:          2 * time.Second,
			finished:          true,
			width:             80,
			expectTruncation:  false,
			expectFooter:      true, // Should show "Thought for 2s"
			expectedLineCount: 2,    // 1 content line + 1 footer line
		},
		{
			name:              "finished with duration and truncation",
			content:           "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12",
			collapsed:         true,
			duration:          1500 * time.Millisecond,
			finished:          true,
			width:             80,
			expectTruncation:  true,
			expectFooter:      true,
			expectedLineCount: 12, // 1 truncation + 10 visible + 1 footer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &ThinkingMessageItem{
				id:        "test",
				content:   tt.content,
				collapsed: tt.collapsed,
				duration:  tt.duration,
				finished:  tt.finished,
			}

			result := item.Render(tt.width)

			// Verify result is not empty
			if result == "" {
				t.Errorf("Render() returned empty string")
			}

			// Check for truncation hint
			if tt.expectTruncation {
				if !strings.Contains(result, "lines hidden") {
					t.Errorf("Expected truncation hint in result, but not found\nGot: %s", result)
				}
			} else {
				if strings.Contains(result, "lines hidden") {
					t.Errorf("Did not expect truncation hint in result, but found it\nGot: %s", result)
				}
			}

			// Check for footer
			if tt.expectFooter {
				if !strings.Contains(result, "Thought for") {
					t.Errorf("Expected footer 'Thought for' in result, but not found\nGot: %s", result)
				}
			} else {
				if strings.Contains(result, "Thought for") {
					t.Errorf("Did not expect footer in result, but found it\nGot: %s", result)
				}
			}

			// Verify caching works
			cachedResult := item.Render(tt.width)
			if result != cachedResult {
				t.Errorf("Cached render differs from original render")
			}

			// Verify cache invalidates with different width
			differentResult := item.Render(tt.width + 10)
			if differentResult == "" {
				t.Errorf("Render with different width returned empty string")
			}

			// Verify Height() returns a positive line count
			height := item.Height()
			if height <= 0 {
				t.Errorf("Height() = %d, want > 0", height)
			}
		})
	}
}

func TestThinkingMessageItemToggleExpanded(t *testing.T) {
	content := "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12"
	item := &ThinkingMessageItem{
		id:        "test",
		content:   content,
		collapsed: true,
	}

	// Initial render (collapsed)
	result1 := item.Render(80)
	if !strings.Contains(result1, "lines hidden") {
		t.Errorf("Expected truncation in collapsed state")
	}

	// Toggle to expanded
	item.ToggleExpanded()
	if item.IsExpanded() != true {
		t.Errorf("IsExpanded() = false after toggle, want true")
	}

	// Verify cache was invalidated
	if item.cachedWidth != 0 {
		t.Errorf("cachedWidth should be 0 after toggle, got %d", item.cachedWidth)
	}

	// Render expanded
	result2 := item.Render(80)
	if strings.Contains(result2, "lines hidden") {
		t.Errorf("Did not expect truncation in expanded state")
	}

	// Results should differ
	if result1 == result2 {
		t.Errorf("Collapsed and expanded renders should differ")
	}

	// Toggle back to collapsed
	item.ToggleExpanded()
	if item.IsExpanded() != false {
		t.Errorf("IsExpanded() = true after second toggle, want false")
	}

	result3 := item.Render(80)
	if !strings.Contains(result3, "lines hidden") {
		t.Errorf("Expected truncation after toggling back to collapsed")
	}
}

func TestDividerMessageItemRender(t *testing.T) {
	tests := []struct {
		name              string
		iteration         int
		width             int
		expectCentered    bool
		expectIterationNo bool
	}{
		{
			name:              "standard width",
			iteration:         1,
			width:             80,
			expectCentered:    true,
			expectIterationNo: true,
		},
		{
			name:              "narrow width",
			iteration:         5,
			width:             30,
			expectCentered:    true,
			expectIterationNo: true,
		},
		{
			name:              "wide width",
			iteration:         10,
			width:             120,
			expectCentered:    true,
			expectIterationNo: true,
		},
		{
			name:              "very narrow width",
			iteration:         2,
			width:             20,
			expectCentered:    true,
			expectIterationNo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &DividerMessageItem{
				id:        "test",
				iteration: tt.iteration,
			}

			result := item.Render(tt.width)

			// Verify result is not empty
			if result == "" {
				t.Errorf("Render() returned empty string")
			}

			// Check for iteration number in label
			if tt.expectIterationNo {
				expectedLabel := " Iteration #"
				if !strings.Contains(result, expectedLabel) {
					t.Errorf("Expected iteration label %q in result, but not found\nGot: %s", expectedLabel, result)
				}
			}

			// Check for horizontal rule characters
			if !strings.Contains(result, "─") {
				t.Errorf("Expected horizontal rule character '─' in result, but not found\nGot: %s", result)
			}

			// Verify caching works
			cachedResult := item.Render(tt.width)
			if result != cachedResult {
				t.Errorf("Cached render differs from original render")
			}

			// Verify cachedWidth is set correctly
			if item.cachedWidth != tt.width {
				t.Errorf("cachedWidth = %d, want %d", item.cachedWidth, tt.width)
			}

			// Verify cache invalidates with different width
			differentResult := item.Render(tt.width + 10)
			if differentResult == "" {
				t.Errorf("Render with different width returned empty string")
			}

			// Verify cachedWidth updates on re-render
			if item.cachedWidth != tt.width+10 {
				t.Errorf("After re-render, cachedWidth = %d, want %d", item.cachedWidth, tt.width+10)
			}

			// Verify Height() returns a positive line count
			height := item.Height()
			if height <= 0 {
				t.Errorf("Height() = %d, want > 0", height)
			}
		})
	}
}

func TestDividerMessageItemRenderCentering(t *testing.T) {
	// Test that the divider has roughly equal line lengths on each side
	item := &DividerMessageItem{
		id:        "test",
		iteration: 5,
	}

	result := item.Render(80)

	// The label is " Iteration #5 " (15 chars)
	// With width 80, we should have (80-15)/2 = 32.5 -> 32 chars on each side
	// Total line should have horizontal rules on both sides

	// Count the number of horizontal rule characters before and after the label
	// We can't precisely test due to ANSI codes, but we can verify the label is present
	if !strings.Contains(result, " Iteration #5 ") {
		t.Errorf("Expected centered label ' Iteration #5 ' in result")
	}

	// Verify the pattern: horizontal rules, then label, then horizontal rules
	// The result should have at least 2 "─" sequences (before and after label)
	ruleCount := strings.Count(result, "─")
	if ruleCount < 10 {
		t.Errorf("Expected multiple horizontal rule characters, got only %d", ruleCount)
	}
}

func TestDividerMessageItemRenderMinLineWidth(t *testing.T) {
	// Test that minimum line width of 3 is enforced even for very narrow widths
	item := &DividerMessageItem{
		id:        "test",
		iteration: 100, // Long iteration number
	}

	// Very narrow width that would make lineWidth < 3
	result := item.Render(15)

	// Should still render something valid
	if result == "" {
		t.Errorf("Render() returned empty string for narrow width")
	}

	// Should still contain the iteration label
	if !strings.Contains(result, "Iteration") {
		t.Errorf("Expected iteration label even with narrow width")
	}

	// Should still have horizontal rules
	if !strings.Contains(result, "─") {
		t.Errorf("Expected horizontal rules even with narrow width")
	}
}

func TestFormatToolParams(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		maxWidth int
		want     string
	}{
		{
			name:     "empty params",
			input:    map[string]any{},
			maxWidth: 100,
			want:     "",
		},
		{
			name: "command only",
			input: map[string]any{
				"command": "git status",
			},
			maxWidth: 100,
			want:     "git status",
		},
		{
			name: "filePath only",
			input: map[string]any{
				"filePath": "/path/to/file.go",
			},
			maxWidth: 100,
			want:     "/path/to/file.go",
		},
		{
			name: "command with additional params",
			input: map[string]any{
				"command": "npm install",
				"workdir": "/app",
				"timeout": 5000,
			},
			maxWidth: 100,
			want:     "npm install (",
		},
		{
			name: "filePath with additional params",
			input: map[string]any{
				"filePath": "/src/main.go",
				"offset":   10,
				"limit":    50,
			},
			maxWidth: 100,
			want:     "/src/main.go (",
		},
		{
			name: "params without primary key",
			input: map[string]any{
				"pattern": "*.go",
				"path":    "/src",
			},
			maxWidth: 100,
			want:     "(",
		},
		{
			name: "truncation when exceeding maxWidth",
			input: map[string]any{
				"command": "very long command that exceeds the maximum width limit",
			},
			maxWidth: 30,
			want:     "very long command that exce...",
		},
		{
			name: "truncation with params",
			input: map[string]any{
				"command": "short",
				"arg1":    "very long argument value",
				"arg2":    "another very long argument",
			},
			maxWidth: 40,
			want:     "short (",
		},
		{
			name: "very small maxWidth",
			input: map[string]any{
				"command": "test",
			},
			maxWidth: 3,
			want:     "tes",
		},
		{
			name: "maxWidth exactly 3 (edge case)",
			input: map[string]any{
				"command": "longcommand",
			},
			maxWidth: 3,
			want:     "lon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolParams(tt.input, tt.maxWidth)

			// For tests with non-deterministic param order, check structure and contents
			hasCommand := tt.input["command"] != nil
			hasFilePath := tt.input["filePath"] != nil
			hasPrimary := hasCommand || hasFilePath
			hasMultipleParams := len(tt.input) > 1

			if hasPrimary && hasMultipleParams {
				// Case: primary key + additional params (non-deterministic order)
				if !strings.HasPrefix(got, tt.want) {
					t.Errorf("formatToolParams() = %q, want to start with %q", got, tt.want)
					return
				}
				// For non-truncated cases, verify all params are present
				if !strings.Contains(got, "...") {
					for key, val := range tt.input {
						if key == "command" || key == "filePath" {
							continue // Primary key already checked in prefix
						}
						expectedPair := key + "=" + fmt.Sprint(val)
						if !strings.Contains(got, expectedPair) {
							t.Errorf("formatToolParams() = %q, missing param %q", got, expectedPair)
						}
					}
				} else {
					// Truncated case - just verify it ends with "..."
					if !strings.HasSuffix(got, "...") {
						t.Errorf("formatToolParams() = %q, expected to end with '...'", got)
					}
				}
			} else if !hasPrimary && len(tt.input) > 0 && !strings.Contains(got, "...") {
				// Case: only non-primary params (non-deterministic order, not truncated)
				if !strings.HasPrefix(got, "(") || !strings.HasSuffix(got, ")") {
					t.Errorf("formatToolParams() = %q, want format (...)", got)
					return
				}
				// Check each param is present
				for key, val := range tt.input {
					expectedPair := key + "=" + fmt.Sprint(val)
					if !strings.Contains(got, expectedPair) {
						t.Errorf("formatToolParams() = %q, missing param %q", got, expectedPair)
					}
				}
			} else {
				// Deterministic cases (single param, empty, or other edge cases)
				if got != tt.want {
					t.Errorf("formatToolParams() = %q, want %q", got, tt.want)
				}
			}

			// Verify result doesn't exceed maxWidth
			if len(got) > tt.maxWidth {
				t.Errorf("formatToolParams() length = %d, exceeds maxWidth %d (got: %q)", len(got), tt.maxWidth, got)
			}
		})
	}
}

func TestFormatToolParamsOrderInsensitive(t *testing.T) {
	// Test that params without primary key are formatted correctly
	// regardless of map iteration order
	input := map[string]any{
		"arg1": "value1",
		"arg2": "value2",
		"arg3": "value3",
	}

	result := formatToolParams(input, 100)

	// Should start and end with parentheses
	if !strings.HasPrefix(result, "(") {
		t.Errorf("Expected result to start with '(', got: %s", result)
	}
	if !strings.HasSuffix(result, ")") {
		t.Errorf("Expected result to end with ')', got: %s", result)
	}

	// Should contain all key-value pairs
	if !strings.Contains(result, "arg1=value1") {
		t.Errorf("Expected 'arg1=value1' in result, got: %s", result)
	}
	if !strings.Contains(result, "arg2=value2") {
		t.Errorf("Expected 'arg2=value2' in result, got: %s", result)
	}
	if !strings.Contains(result, "arg3=value3") {
		t.Errorf("Expected 'arg3=value3' in result, got: %s", result)
	}

	// Should contain commas as separators
	commaCount := strings.Count(result, ",")
	if commaCount != 2 {
		t.Errorf("Expected 2 commas (3 params), got %d commas in: %s", commaCount, result)
	}
}

func TestToolMessageItemRender(t *testing.T) {
	tests := []struct {
		name              string
		item              ToolMessageItem
		width             int
		expectIcon        string
		expectTruncation  bool
		expectExpanded    bool
		expectedLineCount int // Approximate
	}{
		{
			name: "pending tool with no output",
			item: ToolMessageItem{
				id:       "test1",
				toolName: "Bash",
				status:   ToolStatusPending,
				input: map[string]any{
					"command": "ls -la",
				},
				output:   "",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "●",
			expectTruncation:  false,
			expectedLineCount: 1,
		},
		{
			name: "success tool with short output",
			item: ToolMessageItem{
				id:       "test2",
				toolName: "Read",
				status:   ToolStatusSuccess,
				input: map[string]any{
					"filePath": "/path/to/file.go",
				},
				output:   "line 1\nline 2\nline 3",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "✓",
			expectTruncation:  false,
			expectedLineCount: 4, // Header + 3 output lines
		},
		{
			name: "error tool with error message",
			item: ToolMessageItem{
				id:       "test3",
				toolName: "Bash",
				status:   ToolStatusError,
				input: map[string]any{
					"command": "invalid-command",
				},
				output:   "command not found: invalid-command",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "×",
			expectTruncation:  false,
			expectedLineCount: 2, // Header + error line
		},
		{
			name: "collapsed tool with >10 lines",
			item: ToolMessageItem{
				id:       "test4",
				toolName: "Bash",
				status:   ToolStatusSuccess,
				input: map[string]any{
					"command": "cat file.txt",
				},
				output:   "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "✓",
			expectTruncation:  true, // Should show truncation hint
			expectedLineCount: 12,   // Header + 10 lines + truncation hint
		},
		{
			name: "expanded tool with >10 lines",
			item: ToolMessageItem{
				id:       "test5",
				toolName: "Bash",
				status:   ToolStatusSuccess,
				input: map[string]any{
					"command": "cat file.txt",
				},
				output:   "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12\nL13\nL14\nL15",
				expanded: true,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "✓",
			expectTruncation:  false, // No truncation when expanded
			expectedLineCount: 16,    // Header + 15 lines
		},
		{
			name: "canceled tool",
			item: ToolMessageItem{
				id:       "test6",
				toolName: "Bash",
				status:   ToolStatusCanceled,
				input: map[string]any{
					"command": "long-running-command",
				},
				output:   "",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "×",
			expectTruncation:  false,
			expectedLineCount: 1,
		},
		{
			name: "code output with filePath",
			item: ToolMessageItem{
				id:       "test7",
				toolName: "Read",
				status:   ToolStatusSuccess,
				input: map[string]any{
					"filePath": "/src/main.go",
				},
				output:   "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "✓",
			expectTruncation:  false,
			expectedLineCount: 2, // Header + code block (treated as single syntaxHighlight call)
		},
		{
			name: "running tool",
			item: ToolMessageItem{
				id:       "test8",
				toolName: "Bash",
				status:   ToolStatusRunning,
				input: map[string]any{
					"command": "npm install",
				},
				output:   "",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "●",
			expectTruncation:  false,
			expectedLineCount: 1,
		},
		{
			name: "tool with complex params",
			item: ToolMessageItem{
				id:       "test9",
				toolName: "Bash",
				status:   ToolStatusSuccess,
				input: map[string]any{
					"command": "go test ./...",
					"workdir": "/app",
					"timeout": 30000,
				},
				output:   "ok",
				expanded: false,
				maxLines: 10,
			},
			width:             80,
			expectIcon:        "✓",
			expectTruncation:  false,
			expectedLineCount: 2, // Header + output
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.Render(tt.width)

			// Verify result is not empty
			if result == "" {
				t.Errorf("Render() returned empty string")
			}

			// Check for icon (looking for the character, not styled version)
			if !strings.Contains(result, tt.expectIcon) {
				t.Errorf("Expected icon %q in result, but not found\nGot: %s", tt.expectIcon, result)
			}

			// Check for tool name
			if !strings.Contains(result, tt.item.toolName) {
				t.Errorf("Expected tool name %q in result, but not found\nGot: %s", tt.item.toolName, result)
			}

			// Check for truncation hint
			if tt.expectTruncation {
				if !strings.Contains(result, "more lines") {
					t.Errorf("Expected truncation hint in result, but not found\nGot: %s", result)
				}
			} else {
				if strings.Contains(result, "more lines") {
					t.Errorf("Did not expect truncation hint in result, but found it\nGot: %s", result)
				}
			}

			// Verify output content is present if not empty
			if tt.item.output != "" {
				// For error status, check that output is styled as error
				if tt.item.status == ToolStatusError {
					// We can't check exact ANSI codes, but output should be present
					if !strings.Contains(result, strings.Split(tt.item.output, "\n")[0]) {
						t.Errorf("Expected error output in result, but not found\nGot: %s", result)
					}
				}
			}

			// Verify caching works
			cachedResult := tt.item.Render(tt.width)
			if result != cachedResult {
				t.Errorf("Cached render differs from original render")
			}

			// Verify cachedWidth is set correctly
			if tt.item.cachedWidth != tt.width {
				t.Errorf("cachedWidth = %d, want %d", tt.item.cachedWidth, tt.width)
			}

			// Verify cache invalidates with different width
			differentResult := tt.item.Render(tt.width + 10)
			if differentResult == "" {
				t.Errorf("Render with different width returned empty string")
			}

			// Verify cachedWidth updates on re-render
			if tt.item.cachedWidth != tt.width+10 {
				t.Errorf("After re-render, cachedWidth = %d, want %d", tt.item.cachedWidth, tt.width+10)
			}

			// Verify Height() returns a positive line count
			height := tt.item.Height()
			if height <= 0 {
				t.Errorf("Height() = %d, want > 0", height)
			}
		})
	}
}

func TestToolMessageItemToggleExpanded(t *testing.T) {
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
	if !strings.Contains(result1, "more lines") {
		t.Errorf("Expected truncation in collapsed state")
	}

	// Toggle to expanded
	item.ToggleExpanded()
	if item.IsExpanded() != true {
		t.Errorf("IsExpanded() = false after toggle, want true")
	}

	// Verify cache was invalidated
	if item.cachedWidth != 0 {
		t.Errorf("cachedWidth should be 0 after toggle, got %d", item.cachedWidth)
	}

	// Render expanded
	result2 := item.Render(80)
	if strings.Contains(result2, "more lines") {
		t.Errorf("Did not expect truncation in expanded state")
	}

	// Results should differ
	if result1 == result2 {
		t.Errorf("Collapsed and expanded renders should differ")
	}

	// Toggle back to collapsed
	item.ToggleExpanded()
	if item.IsExpanded() != false {
		t.Errorf("IsExpanded() = true after second toggle, want false")
	}

	result3 := item.Render(80)
	if !strings.Contains(result3, "more lines") {
		t.Errorf("Expected truncation after toggling back to collapsed")
	}
}

func TestToolMessageItemRenderIconPerStatus(t *testing.T) {
	// Test that each status renders the correct icon
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

			if !strings.Contains(result, tt.expectIcon) {
				t.Errorf("Expected icon %q for status %v, but not found in: %s",
					tt.expectIcon, tt.status, result)
			}
		})
	}
}

func TestToolMessageItemRenderParamFormatting(t *testing.T) {
	// Test that parameters are formatted correctly in the header
	item := &ToolMessageItem{
		id:       "test",
		toolName: "Bash",
		status:   ToolStatusSuccess,
		input: map[string]any{
			"command": "npm test",
			"workdir": "/app",
		},
		output:   "Tests passed",
		expanded: false,
		maxLines: 10,
	}

	result := item.Render(80)

	// Should contain the primary param (command)
	if !strings.Contains(result, "npm test") {
		t.Errorf("Expected command in result, but not found: %s", result)
	}

	// Should contain the additional param in parentheses
	if !strings.Contains(result, "workdir=/app") {
		t.Errorf("Expected workdir param in result, but not found: %s", result)
	}
}

func TestToolMessageItemRenderOutputTruncation(t *testing.T) {
	// Test exact truncation behavior at maxLines boundary
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
		if !strings.Contains(result, expected) {
			t.Errorf("Expected %q in collapsed output, but not found", expected)
		}
	}

	// Should NOT show lines beyond maxLines
	if strings.Contains(result, "Line 11") {
		t.Errorf("Did not expect Line 11 in collapsed output with maxLines=10")
	}

	// Should show truncation hint with correct count
	if !strings.Contains(result, "5 more lines") {
		t.Errorf("Expected '5 more lines' truncation hint, but not found in: %s", result)
	}
}

func TestWidthCaching_AllMessageTypes(t *testing.T) {
	// Test that width caching works correctly across all message types:
	// 1. Second call with same width returns cached result
	// 2. Different width triggers re-render

	t.Run("TextMessageItem", func(t *testing.T) {
		item := &TextMessageItem{
			id:      "test",
			content: "This is test content that will be rendered",
		}

		// First render at width 80
		result1 := item.Render(80)
		if result1 == "" {
			t.Fatal("First render returned empty string")
		}
		if item.cachedWidth != 80 {
			t.Errorf("After first render, cachedWidth = %d, want 80", item.cachedWidth)
		}
		if item.cachedRender == "" {
			t.Error("After first render, cachedRender should be populated")
		}

		// Second render at same width - should return cached
		result2 := item.Render(80)
		if result2 != result1 {
			t.Error("Second render with same width returned different result (cache miss)")
		}
		if item.cachedWidth != 80 {
			t.Errorf("After second render, cachedWidth = %d, want 80", item.cachedWidth)
		}

		// Third render at different width - should re-render
		result3 := item.Render(100)
		if result3 == "" {
			t.Fatal("Third render returned empty string")
		}
		if item.cachedWidth != 100 {
			t.Errorf("After re-render, cachedWidth = %d, want 100", item.cachedWidth)
		}
		// Result may or may not differ depending on content, but cache should update
	})

	t.Run("ThinkingMessageItem", func(t *testing.T) {
		item := &ThinkingMessageItem{
			id:        "test",
			content:   "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11\nL12",
			collapsed: true,
			finished:  true,
			duration:  2 * time.Second,
		}

		// First render
		result1 := item.Render(80)
		if result1 == "" {
			t.Fatal("First render returned empty string")
		}
		if item.cachedWidth != 80 {
			t.Errorf("cachedWidth = %d, want 80", item.cachedWidth)
		}

		// Second render - same width
		result2 := item.Render(80)
		if result2 != result1 {
			t.Error("Cache miss on second render with same width")
		}

		// Third render - different width
		result3 := item.Render(120)
		if item.cachedWidth != 120 {
			t.Errorf("After re-render, cachedWidth = %d, want 120", item.cachedWidth)
		}
		if result3 == "" {
			t.Error("Re-render returned empty string")
		}
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

		// First render
		result1 := item.Render(80)
		if result1 == "" {
			t.Fatal("First render returned empty string")
		}
		if item.cachedWidth != 80 {
			t.Errorf("cachedWidth = %d, want 80", item.cachedWidth)
		}

		// Second render - same width
		result2 := item.Render(80)
		if result2 != result1 {
			t.Error("Cache miss on second render with same width")
		}

		// Third render - different width
		result3 := item.Render(60)
		if item.cachedWidth != 60 {
			t.Errorf("After re-render, cachedWidth = %d, want 60", item.cachedWidth)
		}
		if result3 == "" {
			t.Error("Re-render returned empty string")
		}
	})

	t.Run("InfoMessageItem", func(t *testing.T) {
		item := &InfoMessageItem{
			id:       "test",
			model:    "claude-3-5-sonnet",
			provider: "Anthropic",
			duration: 3 * time.Second,
		}

		// First render
		result1 := item.Render(80)
		if result1 == "" {
			t.Fatal("First render returned empty string")
		}
		if item.cachedWidth != 80 {
			t.Errorf("cachedWidth = %d, want 80", item.cachedWidth)
		}

		// Second render - same width
		result2 := item.Render(80)
		if result2 != result1 {
			t.Error("Cache miss on second render with same width")
		}

		// Third render - different width
		result3 := item.Render(100)
		if item.cachedWidth != 100 {
			t.Errorf("After re-render, cachedWidth = %d, want 100", item.cachedWidth)
		}
		if result3 == "" {
			t.Error("Re-render returned empty string")
		}
	})

	t.Run("DividerMessageItem", func(t *testing.T) {
		item := &DividerMessageItem{
			id:        "test",
			iteration: 5,
		}

		// First render
		result1 := item.Render(80)
		if result1 == "" {
			t.Fatal("First render returned empty string")
		}
		if item.cachedWidth != 80 {
			t.Errorf("cachedWidth = %d, want 80", item.cachedWidth)
		}

		// Second render - same width
		result2 := item.Render(80)
		if result2 != result1 {
			t.Error("Cache miss on second render with same width")
		}

		// Third render - different width
		result3 := item.Render(120)
		if item.cachedWidth != 120 {
			t.Errorf("After re-render, cachedWidth = %d, want 120", item.cachedWidth)
		}
		if result3 == "" {
			t.Error("Re-render returned empty string")
		}
		// Divider should look different at different widths
		if result1 == result3 {
			t.Error("Divider should render differently at different widths")
		}
	})
}

func TestRenderDiffBlock(t *testing.T) {
	tests := []struct {
		name     string
		oldStr   string
		newStr   string
		width    int
		wantRows int // expected number of rows (max of old/new line counts)
	}{
		{
			name:     "single line change",
			oldStr:   "old line",
			newStr:   "new line",
			width:    80,
			wantRows: 1,
		},
		{
			name:     "multi-line old to single new",
			oldStr:   "line1\nline2\nline3",
			newStr:   "replacement",
			width:    80,
			wantRows: 3, // max(3, 1)
		},
		{
			name:     "single old to multi-line new",
			oldStr:   "original",
			newStr:   "new1\nnew2\nnew3\nnew4",
			width:    80,
			wantRows: 4, // max(1, 4)
		},
		{
			name:     "equal line counts",
			oldStr:   "a\nb\nc",
			newStr:   "x\ny\nz",
			width:    120,
			wantRows: 3,
		},
		{
			name:     "narrow width",
			oldStr:   "short",
			newStr:   "also short",
			width:    50,
			wantRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderDiffBlock(tt.oldStr, tt.newStr, "", tt.width)
			if result == "" {
				t.Error("renderDiffBlock returned empty string")
			}

			// Count rows by newlines
			lines := strings.Split(result, "\n")
			if len(lines) != tt.wantRows {
				t.Errorf("expected %d rows, got %d", tt.wantRows, len(lines))
			}

			// Verify divider character is present in each row
			for i, line := range lines {
				if !strings.Contains(line, "│") {
					t.Errorf("row %d missing divider: %q", i, line)
				}
			}
		})
	}
}

func TestRenderDiagnostics(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		width      int
		wantEmpty  bool
		wantErrors int // number of diagnostic lines expected
	}{
		{
			name: "single error",
			output: `<diagnostics file="/path/to/file.go">
ERROR [153:21] undefined: conn
</diagnostics>`,
			width:      100,
			wantEmpty:  false,
			wantErrors: 1,
		},
		{
			name: "multiple errors",
			output: `<diagnostics file="/path/to/file.go">
ERROR [153:21] undefined: conn
ERROR [153:38] undefined: sessID
</diagnostics>`,
			width:      100,
			wantEmpty:  false,
			wantErrors: 2,
		},
		{
			name: "mixed errors and warnings",
			output: `<diagnostics file="main.go">
ERROR [10:5] undeclared name: foo
WARNING [20:10] unused variable: bar
</diagnostics>`,
			width:      100,
			wantEmpty:  false,
			wantErrors: 2,
		},
		{
			name:      "no diagnostics tag",
			output:    "just some text",
			width:     100,
			wantEmpty: true,
		},
		{
			name: "empty diagnostics",
			output: `<diagnostics file="test.go">
</diagnostics>`,
			width:     100,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderDiagnostics(tt.output, tt.width)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expected empty result, got %q", result)
				}
				return
			}

			if result == "" {
				t.Error("expected non-empty result")
				return
			}

			// Check that the file path is present in output
			if strings.Contains(tt.output, `file="`) {
				// Extract file path from input
				start := strings.Index(tt.output, `file="`) + 6
				end := strings.Index(tt.output[start:], `"`)
				filePath := tt.output[start : start+end]
				if !strings.Contains(result, filePath) {
					t.Errorf("result should contain file path %q", filePath)
				}
			}
		})
	}
}

func TestToolMessageEditDiffRendering(t *testing.T) {
	// Edit tool with oldString/newString should render a diff
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
	if result == "" {
		t.Fatal("Edit tool render returned empty")
	}

	// Should contain the divider character (side-by-side diff)
	if !strings.Contains(result, "│") {
		t.Error("Edit diff should contain side-by-side divider")
	}

	// Should NOT render the plain output text since diff takes precedence
	// (the output "Edit applied successfully" is suppressed)
	if strings.Contains(result, "Edit applied successfully") {
		t.Error("Edit diff should suppress plain output text")
	}

	// Should show the tool name in header
	if !strings.Contains(result, "Edit") {
		t.Error("should show tool name in header")
	}
}

func TestToolMessageDiagnosticsRendering(t *testing.T) {
	// Tool with diagnostics in output should render them nicely
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
	if result == "" {
		t.Fatal("Diagnostics render returned empty")
	}

	// Should contain file path
	if !strings.Contains(result, "main.go") {
		t.Error("should contain file path")
	}

	// Should contain ERROR markers
	if !strings.Contains(result, "ERROR") {
		t.Error("should contain ERROR text")
	}

	// Should contain error icon
	if !strings.Contains(result, "×") {
		t.Error("should contain error icon")
	}
}

func TestToolMessageEditWithDiagnostics(t *testing.T) {
	// Edit tool that has both diff data AND diagnostics in output
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
	if result == "" {
		t.Fatal("Edit+diagnostics render returned empty")
	}

	// Should have diff (divider)
	if !strings.Contains(result, "│") {
		t.Error("should contain diff divider")
	}

	// Should have diagnostics
	if !strings.Contains(result, "ERROR") {
		t.Error("should contain diagnostics")
	}

	// Should have file path from diagnostics
	if !strings.Contains(result, "file.go") {
		t.Error("should contain file path from diagnostics")
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxWidth int
		want     string
	}{
		{"short line", "hello", 10, "hello"},
		{"exact width", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello w…"},
		{"very narrow", "hello", 3, "hel"},
		{"empty", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateLine(tt.line, tt.maxWidth)
			if got != tt.want {
				t.Errorf("truncateLine(%q, %d) = %q, want %q", tt.line, tt.maxWidth, got, tt.want)
			}
		})
	}
}
