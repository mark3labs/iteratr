package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentOutput_PasteMultiline_CollapsesNewlines is a teatest integration test
// that verifies multi-line paste content is collapsed to single line in agent chat input.
func TestAgentOutput_PasteMultiline_CollapsesNewlines(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Focus the input field
	ao.SetInputFocused(true)
	require.True(t, ao.input.Focused(), "input should be focused")

	// Paste multi-line content
	multiLineContent := "first line\nsecond line\n\nfourth line"
	pasteMsg := tea.PasteMsg{Content: multiLineContent}

	// Send the paste message
	cmd := ao.Update(pasteMsg)
	_ = cmd // command may be non-nil due to textarea updates

	// Verify newlines were collapsed to single spaces
	expected := "first line second line fourth line"
	actual := ao.InputValue()
	assert.Equal(t, expected, actual, "multi-line paste should be collapsed to single line")
}

// TestAgentOutput_PasteWithCRLF_CollapsesNewlines verifies CRLF line endings are handled correctly.
func TestAgentOutput_PasteWithCRLF_CollapsesNewlines(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Focus the input field
	ao.SetInputFocused(true)

	// Paste content with CRLF line endings (Windows-style)
	crlfContent := "line1\r\nline2\r\nline3"
	pasteMsg := tea.PasteMsg{Content: crlfContent}

	// Send the paste message
	ao.Update(pasteMsg)

	// Verify CRLF sequences were collapsed to single spaces
	expected := "line1 line2 line3"
	actual := ao.InputValue()
	assert.Equal(t, expected, actual, "CRLF line endings should be collapsed to single line")
}

// TestAgentOutput_PasteWithConsecutiveNewlines_CollapsesToSingleSpace verifies
// that multiple consecutive newlines are collapsed to a single space.
func TestAgentOutput_PasteWithConsecutiveNewlines_CollapsesToSingleSpace(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Focus the input field
	ao.SetInputFocused(true)

	// Paste content with multiple consecutive newlines
	content := "line1\n\n\n\nline2"
	pasteMsg := tea.PasteMsg{Content: content}

	// Send the paste message
	ao.Update(pasteMsg)

	// Verify consecutive newlines were collapsed to single space
	expected := "line1 line2"
	actual := ao.InputValue()
	assert.Equal(t, expected, actual, "consecutive newlines should be collapsed to single space")
}

// TestAgentOutput_PasteWithMixedWhitespace_CollapsesCorrectly verifies
// that mixed whitespace (newlines, tabs, spaces) is handled correctly.
func TestAgentOutput_PasteWithMixedWhitespace_CollapsesCorrectly(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Focus the input field
	ao.SetInputFocused(true)

	// Paste content with mixed whitespace - tabs are preserved by SanitizePaste,
	// but textinput may convert them to spaces
	content := "line1\n\tindented\nline2"
	pasteMsg := tea.PasteMsg{Content: content}

	// Send the paste message
	ao.Update(pasteMsg)

	// Verify: newlines collapsed to spaces
	// Tabs may be converted to spaces by textinput, so we check for either format
	actual := ao.InputValue()
	// Should contain "line1" followed by whitespace followed by "indented" followed by whitespace followed by "line2"
	assert.Contains(t, actual, "line1", "should contain line1")
	assert.Contains(t, actual, "indented", "should contain indented")
	assert.Contains(t, actual, "line2", "should contain line2")
	assert.NotContains(t, actual, "\n", "should not contain newlines")
}

// TestAgentOutput_PasteMultiline_WhenNotFocused_NoChange verifies
// that paste is ignored when input is not focused.
func TestAgentOutput_PasteMultiline_WhenNotFocused_NoChange(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Ensure input is NOT focused
	ao.SetInputFocused(false)
	require.False(t, ao.input.Focused(), "input should not be focused")

	// Paste multi-line content
	multiLineContent := "line1\nline2\nline3"
	pasteMsg := tea.PasteMsg{Content: multiLineContent}

	// Send the paste message
	cmd := ao.Update(pasteMsg)

	// Verify no command returned and no change to input
	assert.Nil(t, cmd, "should return nil command when input not focused")
	assert.Equal(t, "", ao.InputValue(), "input should remain empty when not focused")
}

// TestAgentOutput_PasteMultiline_WithSanitization verifies
// that paste content is both sanitized and has newlines collapsed.
func TestAgentOutput_PasteMultiline_WithSanitization(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Focus the input field
	ao.SetInputFocused(true)

	// Paste content with newlines, ANSI codes, and null bytes
	content := "\x1b[31mred\x1b[0m\nline1\x00\nline2"
	pasteMsg := tea.PasteMsg{Content: content}

	// Send the paste message
	ao.Update(pasteMsg)

	// Verify: ANSI codes stripped, null bytes removed, newlines collapsed
	expected := "red line1 line2"
	actual := ao.InputValue()
	assert.Equal(t, expected, actual, "content should be sanitized and newlines collapsed")
}

// TestAgentOutput_PasteEmptyContent_HandlesGracefully verifies
// that empty paste content is handled gracefully.
func TestAgentOutput_PasteEmptyContent_HandlesGracefully(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Focus the input field
	ao.SetInputFocused(true)

	// Paste empty content
	pasteMsg := tea.PasteMsg{Content: ""}

	// Send the paste message - should not panic
	cmd := ao.Update(pasteMsg)

	// Verify empty result
	assert.Equal(t, "", ao.InputValue(), "empty paste should result in empty input")
	_ = cmd
}

// TestAgentOutput_PasteMultiline_LargeContent verifies
// that large multi-line paste content is handled correctly.
func TestAgentOutput_PasteMultiline_LargeContent(t *testing.T) {
	t.Parallel()

	// Create agent output and initialize
	ao := NewAgentOutput()
	ao.UpdateSize(80, 24)

	// Focus the input field
	ao.SetInputFocused(true)

	// Create large multi-line content (100 lines)
	var content string
	for i := 0; i < 100; i++ {
		if i > 0 {
			content += "\n"
		}
		content += "line"
	}

	pasteMsg := tea.PasteMsg{Content: content}

	// Send the paste message - should not panic or timeout
	done := make(chan bool, 1)
	go func() {
		ao.Update(pasteMsg)
		done <- true
	}()

	select {
	case <-done:
		// Success - update completed
	case <-time.After(2 * time.Second):
		t.Fatal("paste handling timed out for large content")
	}

	// Verify all newlines were collapsed to single spaces
	// Result should be "line line line ..." (100 times)
	expected := "line"
	for i := 1; i < 100; i++ {
		expected += " line"
	}
	actual := ao.InputValue()
	assert.Equal(t, expected, actual, "large multi-line paste should be collapsed correctly")
}
