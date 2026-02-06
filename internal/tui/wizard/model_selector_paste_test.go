package wizard

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestModelSelectorStep_Paste_CollapsesNewlines tests that multi-line paste
// content has newlines collapsed to spaces in the search input.
func TestModelSelectorStep_Paste_CollapsesNewlines(t *testing.T) {
	step := NewModelSelectorStep()
	step.loading = false     // Ensure not in loading state
	step.searchInput.Focus() // Focus the search input

	// Paste multi-line text
	pasteContent := "line1\nline2\n\nline3"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	step.Update(pasteMsg)

	// Verify newlines were collapsed to single space
	expected := "line1 line2 line3"
	if step.searchInput.Value() != expected {
		t.Errorf("Expected search input to contain '%s', got: '%s'", expected, step.searchInput.Value())
	}
}

// TestModelSelectorStep_Paste_SanitizesContent tests that pasted content
// is sanitized (ANSI codes, null bytes removed).
func TestModelSelectorStep_Paste_SanitizesContent(t *testing.T) {
	step := NewModelSelectorStep()
	step.loading = false
	step.searchInput.Focus() // Focus the search input

	// Paste content with ANSI codes and null bytes
	pasteContent := "\x1b[31mred\x1b[0m text\x00"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	step.Update(pasteMsg)

	// Verify ANSI codes and null bytes were removed
	expected := "red text"
	if step.searchInput.Value() != expected {
		t.Errorf("Expected search input to contain '%s', got: '%s'", expected, step.searchInput.Value())
	}
}

// TestModelSelectorStep_Paste_CombinedSanitizationAndNewlineCollapse tests
// that all transformations work together: ANSI stripped, newlines collapsed.
func TestModelSelectorStep_Paste_CombinedSanitizationAndNewlineCollapse(t *testing.T) {
	step := NewModelSelectorStep()
	step.loading = false
	step.searchInput.Focus() // Focus the search input

	// Paste content with newlines, ANSI codes, and CRLF
	pasteContent := "\x1b[1mbold\x1b[0m\r\nline1\n\nline2\x00"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	step.Update(pasteMsg)

	// Verify all transformations
	expected := "bold line1 line2"
	if step.searchInput.Value() != expected {
		t.Errorf("Expected search input to contain '%s', got: '%s'", expected, step.searchInput.Value())
	}
}

// TestModelSelectorStep_Paste_WhenLoading_NoOp tests that paste is ignored
// when the step is in loading state.
func TestModelSelectorStep_Paste_WhenLoading_NoOp(t *testing.T) {
	step := NewModelSelectorStep()
	step.loading = true

	// Try to paste while loading
	pasteContent := "test content"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// No command should be returned
	if cmd != nil {
		t.Error("Expected no command when pasting during loading state")
	}

	// Search input should be empty
	if step.searchInput.Value() != "" {
		t.Errorf("Expected search input to be empty, got: '%s'", step.searchInput.Value())
	}
}

// TestModelSelectorStep_Paste_WhenError_NoOp tests that paste is ignored
// when the step is in error state.
func TestModelSelectorStep_Paste_WhenError_NoOp(t *testing.T) {
	step := NewModelSelectorStep()
	step.loading = false
	step.error = "some error"

	// Try to paste while in error state
	pasteContent := "test content"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// No command should be returned
	if cmd != nil {
		t.Error("Expected no command when pasting during error state")
	}

	// Search input should be empty
	if step.searchInput.Value() != "" {
		t.Errorf("Expected search input to be empty, got: '%s'", step.searchInput.Value())
	}
}

// TestModelSelectorStep_Paste_SingleLine_NoChange tests that single-line
// paste content without newlines is inserted unchanged (except sanitization).
func TestModelSelectorStep_Paste_SingleLine_NoChange(t *testing.T) {
	step := NewModelSelectorStep()
	step.loading = false
	step.searchInput.Focus() // Focus the search input

	// Paste single-line content
	pasteContent := "single line content"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	step.Update(pasteMsg)

	// Content should be inserted as-is
	if step.searchInput.Value() != pasteContent {
		t.Errorf("Expected search input to contain '%s', got: '%s'", pasteContent, step.searchInput.Value())
	}
}

// TestModelSelectorStep_Paste_EmptyContent tests that empty paste content
// is handled gracefully.
func TestModelSelectorStep_Paste_EmptyContent(t *testing.T) {
	step := NewModelSelectorStep()
	step.loading = false
	step.searchInput.Focus() // Focus the search input

	// Paste empty content
	pasteMsg := tea.PasteMsg{Content: ""}

	step.Update(pasteMsg)

	// Search input should remain empty
	if step.searchInput.Value() != "" {
		t.Errorf("Expected search input to be empty, got: '%s'", step.searchInput.Value())
	}
}
