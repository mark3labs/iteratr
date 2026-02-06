package specwizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/tui"
)

// generateLongString creates a string of exactly n 'a' characters
func generateLongString(n int) string {
	return strings.Repeat("a", n)
}

func TestDescriptionStep_PasteWithinLimit_NoTruncation(t *testing.T) {
	step := NewDescriptionStep()

	// Paste 100 chars into empty textarea (limit is 5000)
	pasteContent := generateLongString(100)
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// Command should be from textarea (cursor blink), not a batch with toast
	// We can't easily check the exact command type, but we can verify
	// that executing it doesn't produce a toast message
	if cmd != nil {
		msg := cmd()
		// Should NOT be a toast message
		if _, ok := msg.(tui.ShowToastMsg); ok {
			t.Error("Did not expect ShowToastMsg for within-limit paste")
		}
	}

	// Verify content was added
	if step.textarea.Value() != pasteContent {
		t.Errorf("Expected textarea value to be %q, got %q", pasteContent, step.textarea.Value())
	}
}

func TestDescriptionStep_PasteExceedsLimit_Truncates(t *testing.T) {
	step := NewDescriptionStep()

	// Paste 6000 chars (exceeds 5000 limit)
	pasteContent := generateLongString(6000)
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// Should get a batch command with textarea update + toast
	if cmd == nil {
		t.Fatal("Expected batch command for truncation, got nil")
	}

	// Execute the command and check for toast message
	msg := cmd()
	batchMsgs, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Expected tea.BatchMsg, got %T", msg)
	}

	// Should have 2 commands: textarea update + toast
	if len(batchMsgs) != 2 {
		t.Errorf("Expected 2 commands in batch, got %d", len(batchMsgs))
	}

	// Check for toast message with "1000 chars truncated" (6000 - 5000)
	foundToast := false
	for _, batchCmd := range batchMsgs {
		if batchCmd == nil {
			continue
		}
		batchMsg := batchCmd()
		if toastMsg, ok := batchMsg.(tui.ShowToastMsg); ok {
			if toastMsg.Text != "1000 chars truncated" {
				t.Errorf("Expected toast '1000 chars truncated', got %q", toastMsg.Text)
			}
			foundToast = true
		}
	}
	if !foundToast {
		t.Error("Expected ShowToastMsg in batch, not found")
	}

	// Verify content was truncated to 5000 chars
	if len([]rune(step.textarea.Value())) != 5000 {
		t.Errorf("Expected textarea value to be 5000 chars, got %d", len([]rune(step.textarea.Value())))
	}
}

func TestDescriptionStep_PasteIntoPartiallyFilled_TruncatesCorrectly(t *testing.T) {
	step := NewDescriptionStep()

	// First, add 4000 chars
	step.textarea.SetValue(generateLongString(4000))

	// Now paste 2000 more (would exceed 5000)
	pasteContent := generateLongString(2000)
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// Should get batch command for truncation
	if cmd == nil {
		t.Fatal("Expected batch command for truncation, got nil")
	}

	msg := cmd()
	_, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Expected tea.BatchMsg, got %T", msg)
	}

	// Check for toast message with "1000 chars truncated" (4000 + 2000 - 5000)
	foundToast := false
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		for _, batchCmd := range batchMsg {
			if batchCmd == nil {
				continue
			}
			if toastMsg, ok := batchCmd().(tui.ShowToastMsg); ok {
				if toastMsg.Text != "1000 chars truncated" {
					t.Errorf("Expected toast '1000 chars truncated', got %q", toastMsg.Text)
				}
				foundToast = true
			}
		}
	}
	if !foundToast {
		t.Error("Expected ShowToastMsg in batch, not found")
	}

	// Verify content is 5000 chars (4000 + 1000)
	if len([]rune(step.textarea.Value())) != 5000 {
		t.Errorf("Expected textarea value to be 5000 chars, got %d", len([]rune(step.textarea.Value())))
	}
}

func TestDescriptionStep_PasteWhenFull_ShowsToastOnly(t *testing.T) {
	step := NewDescriptionStep()

	// Fill textarea completely
	step.textarea.SetValue(generateLongString(5000))

	// Try to paste more content
	pasteContent := generateLongString(100)
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// Should get a command (the toast)
	if cmd == nil {
		t.Fatal("Expected command for toast, got nil")
	}

	msg := cmd()
	toastMsg, ok := msg.(tui.ShowToastMsg)
	if !ok {
		t.Fatalf("Expected ShowToastMsg, got %T", msg)
	}

	if toastMsg.Text != "100 chars truncated" {
		t.Errorf("Expected toast '100 chars truncated', got %q", toastMsg.Text)
	}

	// Verify content unchanged
	if len([]rune(step.textarea.Value())) != 5000 {
		t.Errorf("Expected textarea value to remain 5000 chars, got %d", len([]rune(step.textarea.Value())))
	}
}

func TestDescriptionStep_PasteSanitizesContent(t *testing.T) {
	step := NewDescriptionStep()

	// Paste content with ANSI codes and null bytes
	pasteContent := "Hello \x1b[31mRed\x1b[0m World\x00!"
	expectedContent := "Hello Red World!"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// Command should be from textarea (cursor blink), not a batch with toast
	// We can verify by checking that executing it doesn't produce a toast
	if cmd != nil {
		msg := cmd()
		// Check that it's not a toast (no truncation needed for short content)
		if _, ok := msg.(tui.ShowToastMsg); ok {
			t.Error("Did not expect toast for short paste")
		}
	}

	// Verify content was sanitized (ANSI codes and null bytes removed)
	if step.textarea.Value() != expectedContent {
		t.Errorf("Expected sanitized content %q, got %q", expectedContent, step.textarea.Value())
	}
}

func TestDescriptionStep_PasteWithNewlines_Preserved(t *testing.T) {
	step := NewDescriptionStep()

	// Multi-line paste (should be preserved for textarea)
	pasteContent := "Line 1\nLine 2\nLine 3"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	cmd := step.Update(pasteMsg)

	// Command should be from textarea (cursor blink), not a batch with toast
	// We can verify by checking that executing it doesn't produce a toast
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tui.ShowToastMsg); ok {
			t.Error("Did not expect toast for short paste")
		}
	}

	// Verify newlines are preserved (textarea allows newlines)
	if step.textarea.Value() != pasteContent {
		t.Errorf("Expected content with newlines preserved %q, got %q", pasteContent, step.textarea.Value())
	}
}

func TestDescriptionStep_PasteCRLF_Normalized(t *testing.T) {
	step := NewDescriptionStep()

	// Paste with Windows-style line endings
	pasteContent := "Line 1\r\nLine 2\r\nLine 3"
	expectedContent := "Line 1\nLine 2\nLine 3" // Should be normalized to LF
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	step.Update(pasteMsg)

	// Verify CRLF is normalized to LF
	if step.textarea.Value() != expectedContent {
		t.Errorf("Expected CRLF normalized to LF %q, got %q", expectedContent, step.textarea.Value())
	}
}

func TestDescriptionStep_PasteTrailingWhitespace_Trimmed(t *testing.T) {
	step := NewDescriptionStep()

	// Paste with trailing whitespace
	pasteContent := "Hello World   \t\n"
	expectedContent := "Hello World" // Trailing whitespace trimmed
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	step.Update(pasteMsg)

	// Verify trailing whitespace is trimmed
	if step.textarea.Value() != expectedContent {
		t.Errorf("Expected trailing whitespace trimmed to %q, got %q", expectedContent, step.textarea.Value())
	}
}
