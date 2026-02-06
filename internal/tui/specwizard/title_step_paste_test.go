package specwizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/tui"
)

func TestTitleStep_Paste_CollapsesNewlines(t *testing.T) {
	step := NewTitleStep()

	// Paste multi-line content
	pasteContent := "line1\nline2\nline3"
	msg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(msg)
	_ = cmd // Command may be non-nil (e.g., for cursor blink)

	// Check that content was pasted and newlines were collapsed to spaces
	value := step.GetTitle()
	if value != "line1 line2 line3" {
		t.Errorf("Expected 'line1 line2 line3', got '%s'", value)
	}
}

func TestTitleStep_Paste_CollapsesCarriageReturns(t *testing.T) {
	step := NewTitleStep()

	// Paste content with CRLF line endings
	pasteContent := "line1\r\nline2\r\nline3"
	msg := tea.PasteMsg{Content: pasteContent}

	step.Update(msg)

	// Check that CRLF was collapsed to spaces
	value := step.GetTitle()
	if value != "line1 line2 line3" {
		t.Errorf("Expected 'line1 line2 line3', got '%s'", value)
	}
}

func TestTitleStep_Paste_SanitizesAndCollapses(t *testing.T) {
	step := NewTitleStep()

	// Paste content with ANSI codes and newlines
	pasteContent := "\x1b[31mred\x1b[0m\nline2\n\x00nullbyte"
	msg := tea.PasteMsg{Content: pasteContent}

	step.Update(msg)

	// Check that ANSI codes and null bytes are removed, newlines collapsed
	value := step.GetTitle()
	expected := "red line2 nullbyte"
	if value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, value)
	}
}

func TestTitleStep_Paste_CollapsesConsecutiveWhitespace(t *testing.T) {
	step := NewTitleStep()

	// Paste content with multiple consecutive newlines and spaces
	pasteContent := "line1\n\n\n  line2   \n\nline3"
	msg := tea.PasteMsg{Content: pasteContent}

	step.Update(msg)

	// Check that all consecutive whitespace is collapsed to single spaces
	value := step.GetTitle()
	expected := "line1 line2 line3"
	if value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, value)
	}
}

func TestTitleStep_Paste_EmptyContent(t *testing.T) {
	step := NewTitleStep()

	// Paste empty content
	msg := tea.PasteMsg{Content: ""}

	step.Update(msg)

	// Check that nothing was pasted
	value := step.GetTitle()
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}
}

func TestTitleStep_Paste_RespectsCharLimit(t *testing.T) {
	step := NewTitleStep()

	// Create content that exceeds the 100 char limit
	pasteContent := strings.Repeat("a", 150)
	msg := tea.PasteMsg{Content: pasteContent}

	step.Update(msg)

	// Check that textinput enforces its CharLimit of 100
	value := step.input.Value()
	if len(value) > 100 {
		t.Errorf("Expected length <= 100, got %d", len(value))
	}
}

func TestCollapseNewlines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single newline",
			input:    "line1\nline2",
			expected: "line1 line2",
		},
		{
			name:     "multiple newlines",
			input:    "line1\nline2\nline3",
			expected: "line1 line2 line3",
		},
		{
			name:     "CRLF line endings",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1 line2 line3",
		},
		{
			name:     "mixed line endings",
			input:    "line1\nline2\r\nline3\rline4",
			expected: "line1 line2 line3 line4",
		},
		{
			name:     "consecutive newlines",
			input:    "line1\n\n\nline2",
			expected: "line1 line2",
		},
		{
			name:     "consecutive spaces and newlines",
			input:    "line1   \n\n  line2",
			expected: "line1 line2",
		},
		{
			name:     "single line no change",
			input:    "single line text",
			expected: "single line text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \n\t\r   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collapseNewlines(tt.input)
			if result != tt.expected {
				t.Errorf("collapseNewlines(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTitleStep_Paste_ClearsError(t *testing.T) {
	step := NewTitleStep()

	// First trigger an error by submitting empty title
	step.Submit()

	if step.err == "" {
		t.Error("Expected error to be set")
	}

	// Now paste some content
	msg := tea.PasteMsg{Content: "valid title"}
	step.Update(msg)

	// Error should be cleared
	if step.err != "" {
		t.Errorf("Expected error to be cleared, got: %s", step.err)
	}
}

func TestTitleStep_Paste_NoErrorWhenValid(t *testing.T) {
	step := NewTitleStep()

	// Paste valid content when there's no error
	msg := tea.PasteMsg{Content: "some title"}
	step.Update(msg)

	// Error should remain empty
	if step.err != "" {
		t.Errorf("Expected no error, got: %s", step.err)
	}
}

func TestTitleStep_SanitizePasteIntegration(t *testing.T) {
	// Test that SanitizePaste from tui package works correctly with TitleStep
	input := "text\x1b[31m with ANSI\x1b[0m and\x00null\nnewlines"
	sanitized := tui.SanitizePaste(input)

	// ANSI codes and null bytes should be removed, newlines preserved at this stage
	if strings.Contains(sanitized, "\x1b") {
		t.Error("SanitizePaste should remove ANSI escape codes")
	}
	if strings.Contains(sanitized, "\x00") {
		t.Error("SanitizePaste should remove null bytes")
	}
	if !strings.Contains(sanitized, "\n") {
		t.Error("SanitizePaste should preserve newlines for collapseNewlines to handle")
	}

	// Now test through TitleStep
	step := NewTitleStep()
	msg := tea.PasteMsg{Content: input}
	step.Update(msg)

	value := step.GetTitle()
	// Note: null byte is removed without replacement, so "and" and "null" become "andnull"
	expected := "text with ANSI andnull newlines"
	if value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, value)
	}
}
