package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// extractToastMsg executes a command and returns any ShowToastMsg found.
// Handles both single commands and batched commands.
func extractToastMsg(cmd tea.Cmd) *ShowToastMsg {
	if cmd == nil {
		return nil
	}

	msg := cmd()
	if msg == nil {
		return nil
	}

	// Check if it's a batch message
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		// BatchMsg is a slice of commands, execute each to find ShowToastMsg
		for _, c := range batchMsg {
			if c != nil {
				innerMsg := c()
				if tm, ok := innerMsg.(ShowToastMsg); ok {
					return &tm
				}
			}
		}
	} else if tm, ok := msg.(ShowToastMsg); ok {
		return &tm
	}

	return nil
}

// TestNoteInputModal_PasteWithinLimit_NoTruncation tests that paste content
// within the char limit is forwarded without modification.
func TestNoteInputModal_PasteWithinLimit_NoTruncation(t *testing.T) {
	modal := NewNoteInputModal()
	modal.Show()

	// Initial content is empty (0 chars)
	// Paste 100 chars - should fit within 500 limit
	pasteContent := "This is a test note content that is well within the character limit of 500 characters."

	// Create paste message
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	// Handle the paste
	cmd := modal.Update(pasteMsg)

	// Execute command to get any messages
	toastMsg := extractToastMsg(cmd)

	// No toast should be shown (no truncation)
	if toastMsg != nil {
		t.Errorf("Expected no toast for paste within limit, got: %s", toastMsg.Text)
	}

	// Content should be inserted
	if modal.textarea.Value() != pasteContent {
		t.Errorf("Expected textarea to contain pasted content, got: %s", modal.textarea.Value())
	}
}

// TestNoteInputModal_PasteExceedsLimit_Truncates tests that paste content
// exceeding the char limit is truncated and a toast is shown.
func TestNoteInputModal_PasteExceedsLimit_Truncates(t *testing.T) {
	modal := NewNoteInputModal()
	modal.Show()

	// Initial content is empty (0 chars)
	// Create paste content of 600 chars (exceeds 500 limit)
	pasteContent := generateString(600)

	// Create paste message
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	// Handle the paste
	cmd := modal.Update(pasteMsg)

	// Extract toast message
	toastMsg := extractToastMsg(cmd)

	// Toast should be shown with correct truncation count
	if toastMsg == nil {
		t.Fatal("Expected toast for truncated paste, got none")
	}
	expectedToast := "100 chars truncated"
	if toastMsg.Text != expectedToast {
		t.Errorf("Expected toast message '%s', got: %s", expectedToast, toastMsg.Text)
	}

	// Content should be truncated to 500 chars
	value := modal.textarea.Value()
	if len([]rune(value)) != 500 {
		t.Errorf("Expected textarea to contain 500 chars (truncated), got: %d", len([]rune(value)))
	}

	// Content should be first 500 chars of original
	expectedContent := string([]rune(pasteContent)[:500])
	if value != expectedContent {
		t.Error("Truncated content doesn't match expected")
	}
}

// TestNoteInputModal_PasteIntoPartiallyFilled_TruncatesCorrectly tests
// that when textarea already has content, paste is truncated correctly.
func TestNoteInputModal_PasteIntoPartiallyFilled_TruncatesCorrectly(t *testing.T) {
	modal := NewNoteInputModal()
	modal.Show()

	// Set initial content to 400 chars (100 chars remaining space)
	initialContent := generateString(400)
	modal.textarea.SetValue(initialContent)

	// Paste 300 chars - only 100 should fit, 200 truncated
	pasteContent := generateString(300)
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	// Handle the paste
	cmd := modal.Update(pasteMsg)

	// Extract toast message
	toastMsg := extractToastMsg(cmd)

	// Toast should show 200 chars truncated
	if toastMsg == nil {
		t.Fatal("Expected toast for truncated paste, got none")
	}
	expectedToast := "200 chars truncated"
	if toastMsg.Text != expectedToast {
		t.Errorf("Expected toast message '%s', got: %s", expectedToast, toastMsg.Text)
	}

	// Content should be at limit (500 chars total)
	value := modal.textarea.Value()
	runeCount := len([]rune(value))
	if runeCount != 500 {
		t.Errorf("Expected textarea to contain 500 chars total, got: %d", runeCount)
	}
}

// TestNoteInputModal_PasteWhenFull_ShowsToastNoPaste tests that when
// textarea is at the limit, paste shows toast but doesn't add content.
func TestNoteInputModal_PasteWhenFull_ShowsToastNoPaste(t *testing.T) {
	modal := NewNoteInputModal()
	modal.Show()

	// Fill textarea to exactly 500 chars
	fullContent := generateString(500)
	modal.textarea.SetValue(fullContent)

	// Try to paste more content
	pasteContent := "This should not be added"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	// Handle the paste
	cmd := modal.Update(pasteMsg)

	// Extract toast message
	toastMsg := extractToastMsg(cmd)

	// Toast should show all chars truncated
	if toastMsg == nil {
		t.Fatal("Expected toast when pasting into full textarea, got none")
	}
	expectedToast := "24 chars truncated"
	if toastMsg.Text != expectedToast {
		t.Errorf("Expected toast message '%s', got: %s", expectedToast, toastMsg.Text)
	}

	// Content should still be exactly 500 chars
	value := modal.textarea.Value()
	if value != fullContent {
		t.Error("Content should not have changed when pasting into full textarea")
	}
}

// TestNoteInputModal_PasteWhenNotFocused_NoOp tests that paste is ignored
// when the textarea is not focused.
func TestNoteInputModal_PasteWhenNotFocused_NoOp(t *testing.T) {
	modal := NewNoteInputModal()
	modal.Show()

	// Move focus to submit button (away from textarea)
	modal.focus = focusSubmitButton

	// Try to paste content
	pasteContent := "This should not be added"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	// Handle the paste
	cmd := modal.Update(pasteMsg)

	// No command should be returned
	if cmd != nil {
		t.Error("Expected no command when pasting with textarea not focused")
	}

	// Content should be empty
	if modal.textarea.Value() != "" {
		t.Error("Content should be empty when paste is ignored")
	}
}

// TestNoteInputModal_PasteInvisibleModal_NoOp tests that paste is ignored
// when the modal is not visible.
func TestNoteInputModal_PasteInvisibleModal_NoOp(t *testing.T) {
	modal := NewNoteInputModal()
	// Modal is not shown (visible = false)

	// Try to paste content
	pasteContent := "This should not be added"
	pasteMsg := tea.PasteMsg{Content: pasteContent}

	// Handle the paste
	cmd := modal.Update(pasteMsg)

	// No command should be returned
	if cmd != nil {
		t.Error("Expected no command when pasting into invisible modal")
	}
}

// generateString generates a string of specified length using repeated pattern
func generateString(length int) string {
	pattern := "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]rune, length)
	for i := 0; i < length; i++ {
		result[i] = rune(pattern[i%len(pattern)])
	}
	return string(result)
}
