package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestAgentOutput_Paste_CollapsesNewlines(t *testing.T) {
	agent := NewAgentOutput()
	agent.UpdateSize(80, 24)
	agent.SetInputFocused(true)

	// Paste multi-line text
	pasteContent := "line1\nline2\n\nline3"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	agent.Update(pasteMsg)

	// Verify newlines were collapsed to single space
	assert.Equal(t, "line1 line2 line3", agent.InputValue())
}

func TestAgentOutput_Paste_SanitizesContent(t *testing.T) {
	agent := NewAgentOutput()
	agent.UpdateSize(80, 24)
	agent.SetInputFocused(true)

	// Paste content with ANSI codes and null bytes
	pasteContent := "\x1b[31mred\x1b[0m text\x00"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	agent.Update(pasteMsg)

	// Verify ANSI codes and null bytes were removed
	assert.Equal(t, "red text", agent.InputValue())
}

func TestAgentOutput_Paste_WhenNotFocused_NoOp(t *testing.T) {
	agent := NewAgentOutput()
	agent.UpdateSize(80, 24)
	agent.SetInputFocused(false)

	// Paste content while input is not focused
	pasteContent := "pasted text"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	agent.Update(pasteMsg)

	// Input should remain empty since it's not focused
	assert.Equal(t, "", agent.InputValue())
}

func TestAgentOutput_Paste_CombinedSanitizationAndNewlineCollapse(t *testing.T) {
	agent := NewAgentOutput()
	agent.UpdateSize(80, 24)
	agent.SetInputFocused(true)

	// Paste content with newlines, ANSI codes, and CRLF
	pasteContent := "\x1b[1mbold\x1b[0m\r\nline1\n\nline2\x00"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	agent.Update(pasteMsg)

	// Verify all transformations: ANSI stripped, CRLF->LF then newlines collapsed, null bytes removed
	assert.Equal(t, "bold line1 line2", agent.InputValue())
}
