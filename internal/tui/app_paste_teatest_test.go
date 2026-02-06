package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// generateLongString generates a string of specified length using repeated pattern
func generateLongString(length int) string {
	pattern := "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]rune, length)
	for i := 0; i < length; i++ {
		result[i] = rune(pattern[i%len(pattern)])
	}
	return string(result)
}

// TestApp_PasteExceedsCharLimit_TruncatesAndShowsToast verifies that when
// pasting content exceeding the 500 char limit into TaskInputModal,
// the content is truncated and a toast notification is shown.
func TestApp_PasteExceedsCharLimit_TruncatesAndShowsToast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the task input modal
	app.taskInputModal.Show()

	// Create paste content of 600 chars (exceeds 500 limit)
	pasteContent := generateLongString(600)

	// Send paste message through App's handlePaste
	pasteMsg := tea.PasteMsg{Content: pasteContent}
	_, cmd := app.handlePaste(pasteMsg)

	// The command should contain a batch with textarea update + toast
	if cmd == nil {
		t.Fatal("Expected command from paste handling, got nil")
	}

	// Execute the command to get messages
	msg := cmd()
	if msg == nil {
		t.Fatal("Expected message from paste command, got nil")
	}

	// Check for batch message containing both textarea update and toast
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Expected BatchMsg from paste handling, got %T", msg)
	}

	// Look for ShowToastMsg in the batch
	var toastFound bool
	var toastText string
	for _, c := range batchMsg {
		if c != nil {
			innerMsg := c()
			if tm, ok := innerMsg.(ShowToastMsg); ok {
				toastFound = true
				toastText = tm.Text
				break
			}
		}
	}

	if !toastFound {
		t.Error("Expected ShowToastMsg in batch, but not found")
	}

	expectedToast := "100 chars truncated"
	if toastText != expectedToast {
		t.Errorf("Expected toast message '%s', got: %s", expectedToast, toastText)
	}

	// Verify content was truncated to 500 chars
	value := app.taskInputModal.textarea.Value()
	runeCount := len([]rune(value))
	if runeCount != 500 {
		t.Errorf("Expected textarea to contain 500 chars (truncated), got: %d", runeCount)
	}

	// Content should be first 500 chars of original
	expectedContent := string([]rune(pasteContent)[:500])
	if value != expectedContent {
		t.Error("Truncated content doesn't match expected")
	}
}

// TestApp_PasteIntoPartiallyFilled_TruncatesCorrectly verifies that when
// textarea already has content, paste is truncated correctly based on remaining space.
func TestApp_PasteIntoPartiallyFilled_TruncatesCorrectly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the task input modal
	app.taskInputModal.Show()

	// Set initial content to 400 chars (100 chars remaining space)
	initialContent := generateLongString(400)
	app.taskInputModal.textarea.SetValue(initialContent)

	// Paste 300 chars - only 100 should fit, 200 truncated
	pasteContent := generateLongString(300)
	pasteMsg := tea.PasteMsg{Content: pasteContent}
	_, cmd := app.handlePaste(pasteMsg)

	// Execute the command
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Expected BatchMsg, got %T", msg)
	}

	// Look for toast message
	var toastFound bool
	var toastText string
	for _, c := range batchMsg {
		if c != nil {
			innerMsg := c()
			if tm, ok := innerMsg.(ShowToastMsg); ok {
				toastFound = true
				toastText = tm.Text
				break
			}
		}
	}

	if !toastFound {
		t.Error("Expected ShowToastMsg for truncated paste")
	}

	expectedToast := "200 chars truncated"
	if toastText != expectedToast {
		t.Errorf("Expected toast message '%s', got: %s", expectedToast, toastText)
	}

	// Content should be at limit (500 chars total)
	value := app.taskInputModal.textarea.Value()
	runeCount := len([]rune(value))
	if runeCount != 500 {
		t.Errorf("Expected textarea to contain 500 chars total, got: %d", runeCount)
	}
}

// TestApp_PasteWhenFull_ShowsToastOnly verifies that when textarea is at the
// limit, paste shows toast but doesn't add content.
func TestApp_PasteWhenFull_ShowsToastOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the task input modal
	app.taskInputModal.Show()

	// Fill textarea to exactly 500 chars
	fullContent := generateLongString(500)
	app.taskInputModal.textarea.SetValue(fullContent)

	// Try to paste more content
	pasteContent := "This should not be added"
	pasteMsg := tea.PasteMsg{Content: pasteContent}
	_, cmd := app.handlePaste(pasteMsg)

	// Execute the command
	msg := cmd()

	// When textarea is full, we should get a ShowToastMsg directly (not a batch)
	toastMsg, ok := msg.(ShowToastMsg)
	if !ok {
		t.Fatalf("Expected ShowToastMsg when pasting into full textarea, got %T", msg)
	}

	expectedToast := "24 chars truncated"
	if toastMsg.Text != expectedToast {
		t.Errorf("Expected toast message '%s', got: %s", expectedToast, toastMsg.Text)
	}

	// Content should still be exactly 500 chars
	value := app.taskInputModal.textarea.Value()
	if value != fullContent {
		t.Error("Content should not have changed when pasting into full textarea")
	}
}

// TestApp_PasteWithinLimit_NoToast verifies that when pasting content
// within the char limit, no toast is shown.
func TestApp_PasteWithinLimit_NoToast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the task input modal
	app.taskInputModal.Show()

	// Paste 100 chars - should fit within 500 limit
	pasteContent := "This is a test task content that is well within the character limit of 500 characters."
	pasteMsg := tea.PasteMsg{Content: pasteContent}
	_, cmd := app.handlePaste(pasteMsg)

	// Execute the command
	msg := cmd()

	// Check for ShowToastMsg (should NOT be present)
	var toastFound bool
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batchMsg {
			if c != nil {
				innerMsg := c()
				if _, ok := innerMsg.(ShowToastMsg); ok {
					toastFound = true
					break
				}
			}
		}
	} else if _, ok := msg.(ShowToastMsg); ok {
		toastFound = true
	}

	if toastFound {
		t.Error("Expected no toast for paste within limit")
	}

	// Content should be inserted unchanged
	value := app.taskInputModal.textarea.Value()
	if !strings.Contains(value, "test task content") {
		t.Errorf("Expected textarea to contain pasted content, got: %s", value)
	}
}

// TestApp_PasteNoteModalExceedsLimit_TruncatesAndShowsToast verifies that
// NoteInputModal also truncates and shows toast when paste exceeds limit.
func TestApp_PasteNoteModalExceedsLimit_TruncatesAndShowsToast(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the note input modal (has priority over task modal)
	app.noteInputModal.Show()

	// Create paste content of 600 chars (exceeds 500 limit)
	pasteContent := generateLongString(600)

	// Send paste message through App's handlePaste
	pasteMsg := tea.PasteMsg{Content: pasteContent}
	_, cmd := app.handlePaste(pasteMsg)

	// Execute the command
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Expected BatchMsg, got %T", msg)
	}

	// Look for toast message
	var toastFound bool
	var toastText string
	for _, c := range batchMsg {
		if c != nil {
			innerMsg := c()
			if tm, ok := innerMsg.(ShowToastMsg); ok {
				toastFound = true
				toastText = tm.Text
				break
			}
		}
	}

	if !toastFound {
		t.Error("Expected ShowToastMsg for truncated paste in note modal")
	}

	expectedToast := "100 chars truncated"
	if toastText != expectedToast {
		t.Errorf("Expected toast message '%s', got: %s", expectedToast, toastText)
	}

	// Content should be truncated to 500 chars
	value := app.noteInputModal.textarea.Value()
	runeCount := len([]rune(value))
	if runeCount != 500 {
		t.Errorf("Expected textarea to contain 500 chars (truncated), got: %d", runeCount)
	}
}

// TestApp_ToastIntegration_ShowsAndDismisses verifies that when ShowToastMsg
// is sent to App.Update, the toast is shown and can be dismissed.
func TestApp_ToastIntegration_ShowsAndDismisses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Initially toast should not be visible
	if app.toast.IsVisible() {
		t.Error("Toast should not be visible initially")
	}

	// Send ShowToastMsg to App.Update
	showMsg := ShowToastMsg{Text: "50 chars truncated"}
	_, cmd := app.Update(showMsg)

	// Command should schedule dismissal
	if cmd == nil {
		t.Error("Expected dismissal command from toast show")
	}

	// Toast should now be visible
	if !app.toast.IsVisible() {
		t.Error("Toast should be visible after ShowToastMsg")
	}

	if app.toast.GetMessage() != "50 chars truncated" {
		t.Errorf("Expected toast message '50 chars truncated', got: %s", app.toast.GetMessage())
	}

	// Simulate dismissal by executing the dismiss command
	// (which will include the correct generation)
	dismissMsg := cmd()
	_, _ = app.Update(dismissMsg)

	// Toast should no longer be visible
	if app.toast.IsVisible() {
		t.Error("Toast should not be visible after dismissal")
	}
}

// TestApp_PasteWithSanitizationAndTruncation verifies that paste content
// is both sanitized and truncated when necessary.
func TestApp_PasteWithSanitizationAndTruncation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the task input modal
	app.taskInputModal.Show()

	// Create paste content with ANSI codes and exceeding limit
	// 500 chars of normal content + ANSI codes + 100 more chars
	normalContent := generateLongString(500)
	pasteContent := "\x1b[31m" + normalContent + "\x1b[0m" + generateLongString(100)

	// Send paste message
	pasteMsg := tea.PasteMsg{Content: pasteContent}
	_, cmd := app.handlePaste(pasteMsg)

	// Execute the command
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Expected BatchMsg, got %T", msg)
	}

	// Look for toast message (100 chars should be truncated)
	var toastFound bool
	for _, c := range batchMsg {
		if c != nil {
			innerMsg := c()
			if tm, ok := innerMsg.(ShowToastMsg); ok {
				toastFound = true
				expectedToast := "100 chars truncated"
				if tm.Text != expectedToast {
					t.Errorf("Expected toast '%s', got: %s", expectedToast, tm.Text)
				}
				break
			}
		}
	}

	if !toastFound {
		t.Error("Expected ShowToastMsg for truncated paste")
	}

	// Verify content is at 500 char limit and ANSI codes are stripped
	value := app.taskInputModal.textarea.Value()
	runeCount := len([]rune(value))
	if runeCount != 500 {
		t.Errorf("Expected textarea to contain 500 chars, got: %d", runeCount)
	}

	// ANSI codes should be stripped
	if strings.Contains(value, "\x1b[") {
		t.Error("ANSI escape codes should be stripped from pasted content")
	}
}
