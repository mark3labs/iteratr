package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

// TestModalPriority_Dialog_OverTaskModal tests that Dialog has highest priority over TaskModal
func TestModalPriority_Dialog_OverTaskModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open task modal first
	task := &session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1}
	app.taskModal.SetTask(task)

	// Verify task modal is visible
	if !app.taskModal.IsVisible() {
		t.Fatal("Task modal should be visible")
	}

	// Open dialog (higher priority)
	app.dialog.Show("Test Title", "Test message", nil)

	// Verify dialog is visible
	if !app.dialog.IsVisible() {
		t.Fatal("Dialog should be visible")
	}

	// Send ESC key - should close dialog, not task modal
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	// Dialog should be closed
	if app.dialog.IsVisible() {
		t.Error("Dialog should be closed after ESC")
	}

	// Task modal should still be open
	if !app.taskModal.IsVisible() {
		t.Error("Task modal should still be visible after dialog closes")
	}
}

// TestModalPriority_Dialog_OverNoteModal tests that Dialog has priority over NoteModal
func TestModalPriority_Dialog_OverNoteModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open note modal first
	note := &session.Note{ID: "note1", Content: "Test note", Type: "learning", Iteration: 1}
	app.noteModal.SetNote(note)

	if !app.noteModal.IsVisible() {
		t.Fatal("Note modal should be visible")
	}

	// Open dialog
	app.dialog.Show("Test Title", "Test message", nil)

	if !app.dialog.IsVisible() {
		t.Fatal("Dialog should be visible")
	}

	// Send ESC - should close dialog, not note modal
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	if app.dialog.IsVisible() {
		t.Error("Dialog should be closed")
	}
	if !app.noteModal.IsVisible() {
		t.Error("Note modal should still be visible")
	}
}

// TestModalPriority_Dialog_OverNoteInputModal tests Dialog priority over NoteInputModal
func TestModalPriority_Dialog_OverNoteInputModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40
	app.iteration = 1 // Required for note input modal

	// Open note input modal
	app.noteInputModal.Show()

	if !app.noteInputModal.IsVisible() {
		t.Fatal("Note input modal should be visible")
	}

	// Open dialog
	app.dialog.Show("Test Title", "Test message", nil)

	if !app.dialog.IsVisible() {
		t.Fatal("Dialog should be visible")
	}

	// Send any key - dialog should consume it, not note input modal
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "a"})
	app = updatedModel.(*App)

	// Dialog should still be visible (key consumed)
	if !app.dialog.IsVisible() {
		t.Error("Dialog should still be visible after key press")
	}
	if !app.noteInputModal.IsVisible() {
		t.Error("Note input modal should still be visible behind dialog")
	}
}

// TestModalPriority_Dialog_OverTaskInputModal tests Dialog priority over TaskInputModal
func TestModalPriority_Dialog_OverTaskInputModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40
	app.iteration = 1

	// Open task input modal
	app.taskInputModal.Show()

	if !app.taskInputModal.IsVisible() {
		t.Fatal("Task input modal should be visible")
	}

	// Open dialog
	app.dialog.Show("Test Title", "Test message", nil)

	// Send any key - dialog consumes it
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "a"})
	app = updatedModel.(*App)

	if !app.dialog.IsVisible() {
		t.Error("Dialog should still be visible")
	}
	if !app.taskInputModal.IsVisible() {
		t.Error("Task input modal should still be visible behind dialog")
	}
}

// TestModalPriority_Dialog_OverSubagentModal tests Dialog priority over SubagentModal
func TestModalPriority_Dialog_OverSubagentModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Create subagent modal
	app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")

	if app.subagentModal == nil {
		t.Fatal("Subagent modal should not be nil")
	}

	// Open dialog
	app.dialog.Show("Test Title", "Test message", nil)

	// Send ESC - should close dialog, not subagent modal
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	if app.dialog.IsVisible() {
		t.Error("Dialog should be closed")
	}
	if app.subagentModal == nil {
		t.Error("Subagent modal should still exist")
	}
}

// TestModalPriority_Dialog_OverLogs tests Dialog priority over LogViewer
func TestModalPriority_Dialog_OverLogs(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open logs
	app.logsVisible = true

	// Open dialog
	app.dialog.Show("Test Title", "Test message", nil)

	// Send ESC - should close dialog, not logs
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	if app.dialog.IsVisible() {
		t.Error("Dialog should be closed")
	}
	if !app.logsVisible {
		t.Error("Logs should still be visible")
	}
}

// TestModalPriority_PrefixMode_AfterGlobalKeys tests that prefix mode comes after global key handling
func TestModalPriority_PrefixMode_AfterGlobalKeys(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Enter prefix mode
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+x"})
	app = updatedModel.(*App)

	if !app.awaitingPrefixKey {
		t.Fatal("Should be in prefix mode")
	}

	// ctrl+c should still quit even in prefix mode
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+c"})
	app = updatedModel.(*App)

	if !app.quitting {
		t.Error("ctrl+c should quit even in prefix mode")
	}
}

// TestModalPriority_PrefixMode_BlocksModals tests that prefix key sequences work when modals could open
func TestModalPriority_PrefixMode_BlocksModals(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40
	app.iteration = 1 // Enable note/task creation

	// Enter prefix mode
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+x"})
	app = updatedModel.(*App)

	// Press 'l' to toggle logs (prefix sequence)
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "l"})
	app = updatedModel.(*App)

	if !app.logsVisible {
		t.Error("Logs should be visible after ctrl+x l")
	}
	if app.awaitingPrefixKey {
		t.Error("Should have exited prefix mode")
	}
}

// TestModalPriority_TaskModal_OverNoteModal tests TaskModal priority over NoteModal (shouldn't happen but verify)
func TestModalPriority_TaskModal_OverNoteModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open note modal first
	note := &session.Note{ID: "note1", Content: "Test note", Type: "learning", Iteration: 1}
	app.noteModal.SetNote(note)

	// Then open task modal (should be blocked by note modal in practice, but test priority)
	task := &session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1}
	app.taskModal.SetTask(task)

	// Both are visible
	if !app.taskModal.IsVisible() || !app.noteModal.IsVisible() {
		t.Fatal("Both modals should be visible for priority test")
	}

	// ESC should close task modal first (higher priority in routing)
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	if app.taskModal.IsVisible() {
		t.Error("Task modal should be closed")
	}
	if !app.noteModal.IsVisible() {
		t.Error("Note modal should still be visible")
	}
}

// TestModalPriority_NoteInputModal_OverTaskInputModal tests NoteInputModal priority over TaskInputModal
func TestModalPriority_NoteInputModal_OverTaskInputModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open task input modal
	app.taskInputModal.Show()

	// Then open note input modal (should be blocked but test priority)
	app.noteInputModal.Show()

	if !app.noteInputModal.IsVisible() || !app.taskInputModal.IsVisible() {
		t.Fatal("Both modals should be visible for priority test")
	}

	// ESC should close note input modal first (higher priority)
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	if app.noteInputModal.IsVisible() {
		t.Error("Note input modal should be closed")
	}
	if !app.taskInputModal.IsVisible() {
		t.Error("Task input modal should still be visible")
	}
}

// TestModalPriority_SubagentModal_OverLogs tests SubagentModal priority over LogViewer
func TestModalPriority_SubagentModal_OverLogs(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open logs
	app.logsVisible = true

	// Open subagent modal
	app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")

	if app.subagentModal == nil || !app.logsVisible {
		t.Fatal("Both should be visible for priority test")
	}

	// ESC should close subagent modal first (higher priority)
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	if app.subagentModal != nil {
		t.Error("Subagent modal should be closed")
	}
	if !app.logsVisible {
		t.Error("Logs should still be visible")
	}
}

// TestModalPriority_CompleteHierarchy tests the complete priority chain
func TestModalPriority_CompleteHierarchy(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open everything from lowest to highest priority
	app.logsVisible = true
	app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
	app.taskInputModal.Show()
	app.noteInputModal.Show()
	app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test note", Type: "learning", Iteration: 1})
	app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1})
	app.dialog.Show("Test Title", "Test message", nil)

	// Close in priority order (highest to lowest)

	// 1. Dialog should close first
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)
	if app.dialog.IsVisible() {
		t.Error("Dialog should be closed")
	}

	// 2. TaskModal should close next
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)
	if app.taskModal.IsVisible() {
		t.Error("Task modal should be closed")
	}

	// 3. NoteModal should close next
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)
	if app.noteModal.IsVisible() {
		t.Error("Note modal should be closed")
	}

	// 4. NoteInputModal should close next
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)
	if app.noteInputModal.IsVisible() {
		t.Error("Note input modal should be closed")
	}

	// 5. TaskInputModal should close next
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)
	if app.taskInputModal.IsVisible() {
		t.Error("Task input modal should be closed")
	}

	// 6. SubagentModal should close next
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)
	if app.subagentModal != nil {
		t.Error("Subagent modal should be closed")
	}

	// 7. Logs should close last
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)
	if app.logsVisible {
		t.Error("Logs should be closed")
	}
}

// TestModalPriority_GlobalKeys_OverDialog tests that ctrl+c works even with dialog open
func TestModalPriority_GlobalKeys_OverDialog(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open dialog
	app.dialog.Show("Test Title", "Test message", nil)

	// ctrl+c should still quit
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+c"})
	app = updatedModel.(*App)

	if !app.quitting {
		t.Error("ctrl+c should quit even with dialog open")
	}
}

// TestModalPriority_PrefixKeySequence_WithModalsBlocked tests that modals can't open during prefix key actions
func TestModalPriority_PrefixKeySequence_WithModalsBlocked(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40
	app.iteration = 1

	// Open a modal first
	app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1})

	if !app.taskModal.IsVisible() {
		t.Fatal("Task modal should be visible")
	}

	// Try to open note input modal via prefix key - should be blocked by existing modal
	updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+x"})
	app = updatedModel.(*App)
	updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "n"})
	app = updatedModel.(*App)

	// Note input modal should NOT be visible because task modal blocks it
	if app.noteInputModal.IsVisible() {
		t.Error("Note input modal should not open when task modal is visible")
	}
	if !app.taskModal.IsVisible() {
		t.Error("Task modal should still be visible")
	}
}

// TestModalPriority_KeyCapture_DialogConsumesAllKeys tests that dialog captures all keyboard input
func TestModalPriority_KeyCapture_DialogConsumesAllKeys(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open dialog
	app.dialog.Show("Test Title", "Test message", nil)

	// Try various keys - all should be consumed by dialog
	keys := []string{"a", "j", "k", "tab", "enter", "space"}
	for _, key := range keys {
		initialVisible := app.dialog.IsVisible()
		updatedModel, cmd := app.handleKeyPress(tea.KeyPressMsg{Text: key})
		app = updatedModel.(*App)

		// Enter and space close dialog, others don't
		if key == "enter" || key == "space" {
			if app.dialog.IsVisible() {
				t.Errorf("Dialog should be closed after %s", key)
			}
			// Re-open for next test
			app.dialog.Show("Test Title", "Test message", nil)
		} else {
			if !initialVisible || !app.dialog.IsVisible() {
				t.Errorf("Dialog should remain open after %s key", key)
			}
		}

		// Verify command is returned (even if nil)
		_ = cmd
	}
}

// TestModalPriority_MouseCapture_DialogPriority tests that mouse clicks respect modal priority
func TestModalPriority_MouseCapture_DialogPriority(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 120
	app.height = 40

	// Open task modal
	app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1})

	// Open dialog over it
	app.dialog.Show("Test Title", "Test message", nil)

	// Click anywhere - should dismiss dialog
	// Note: We can't easily create a MouseClickMsg in tests since it's created by bubbletea
	// Instead, we verify the priority logic by directly calling dialog.HandleClick
	cmd := app.dialog.HandleClick(50, 20)
	_ = cmd // Dialog.HandleClick returns cmd from onClose callback

	if app.dialog.IsVisible() {
		t.Error("Dialog should be closed after mouse click")
	}
	if !app.taskModal.IsVisible() {
		t.Error("Task modal should still be visible")
	}
}
