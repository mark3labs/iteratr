package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

// TestApp_TaskModal_ESCKey tests ESC key closes modal
func TestApp_TaskModal_ESCKey(t *testing.T) {
	// Create minimal app for testing (without full NATS setup)
	app := &App{
		taskModal: NewTaskModal(),
		dialog:    NewDialog(),
	}

	// Open modal
	task := &session.Task{
		ID:      "test123",
		Content: "Test task",
	}
	app.taskModal.SetTask(task)

	if !app.taskModal.IsVisible() {
		t.Fatal("Modal should be visible after SetTask")
	}

	// Create ESC key message
	escMsg := tea.KeyPressMsg{Text: "esc"}

	// Process ESC key through handleKeyPress
	_, _ = app.handleKeyPress(escMsg)

	if app.taskModal.IsVisible() {
		t.Error("Modal should be closed after ESC key")
	}
}

// TestApp_TaskModal_BlocksKeysWhenVisible tests that modal blocks keyboard input
func TestApp_TaskModal_BlocksKeysWhenVisible(t *testing.T) {
	app := &App{
		taskModal: NewTaskModal(),
		dialog:    NewDialog(),
	}

	// Open modal
	task := &session.Task{
		ID:      "test123",
		Content: "Test task",
	}
	app.taskModal.SetTask(task)

	// Try pressing a key (should be blocked by modal)
	keyMsg := tea.KeyPressMsg{Text: "x"}
	_, _ = app.handleKeyPress(keyMsg)

	// App should not quit when modal is open
	if app.quitting {
		t.Error("App should not quit when modal is open")
	}
}

// TestApp_TaskModal_OpenTaskModalMsg tests OpenTaskModalMsg
func TestApp_TaskModal_OpenTaskModalMsg(t *testing.T) {
	app := &App{
		taskModal: NewTaskModal(),
	}

	task := &session.Task{
		ID:      "task1",
		Content: "First task",
		Status:  "remaining",
	}

	// Send OpenTaskModalMsg
	msg := OpenTaskModalMsg{Task: task}
	_, _ = app.Update(msg)

	if !app.taskModal.IsVisible() {
		t.Error("Modal should be visible after OpenTaskModalMsg")
	}

	if app.taskModal.task != task {
		t.Error("Modal should display the correct task")
	}
}

// TestApp_TaskModal_SmallTerminal tests modal rendering on small terminal
func TestApp_TaskModal_SmallTerminal(t *testing.T) {
	task := &session.Task{
		ID:      "test123",
		Content: "Test task with some content that should wrap properly",
		Status:  "in_progress",
	}

	testSizes := []struct {
		name   string
		width  int
		height int
	}{
		{"Minimum", 30, 15},
		{"Very small", 40, 20},
		{"Small", 60, 25},
		{"Large", 100, 40},
	}

	for _, tt := range testSizes {
		t.Run(tt.name, func(t *testing.T) {
			modal := NewTaskModal()
			modal.SetTask(task)

			// Test buildContent directly (doesn't require zone manager)
			if modal.IsVisible() {
				content := modal.buildContent(tt.width - 10)
				if content == "" {
					t.Error("Modal content should not be empty")
				}

				// Verify content contains key elements
				if !strings.Contains(content, task.ID) {
					t.Error("Content should contain task ID")
				}
			}
		})
	}
}

// TestApp_TaskModal_RenderPerformance tests that modal content generation doesn't cause performance issues
func TestApp_TaskModal_RenderPerformance(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "test123",
		Content:   "Test task with moderate length content for performance testing that should wrap across multiple lines when displayed in the modal overlay",
		Status:    "in_progress",
		Priority:  1,
		DependsOn: []string{"dep1", "dep2", "dep3"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	modal.SetTask(task)

	// Measure time to build modal content 1000 times
	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		_ = modal.buildContent(50) // Build content directly without zone manager
	}
	duration := time.Since(start)

	// Should complete 1000 iterations in under 100ms (very generous)
	maxDuration := 100 * time.Millisecond
	if duration > maxDuration {
		t.Errorf("Modal content rendering too slow: %v for %d iterations (max %v)",
			duration, iterations, maxDuration)
	}

	avgPerRender := duration / time.Duration(iterations)
	t.Logf("Average buildContent time: %v per modal", avgPerRender)
}

// TestApp_TaskModal_NoKeysPassThrough tests that no keys pass through when modal is open
func TestApp_TaskModal_NoKeysPassThrough(t *testing.T) {
	app := &App{
		taskModal: NewTaskModal(),
		dialog:    NewDialog(),
		quitting:  false,
	}

	// Open modal
	task := &session.Task{
		ID:      "test123",
		Content: "Test task",
	}
	app.taskModal.SetTask(task)

	// Test various keys that should be blocked
	testKeys := []string{"x", "j", "k", "enter", "tab"}

	for _, key := range testKeys {
		t.Run("Key_"+key, func(t *testing.T) {
			keyMsg := tea.KeyPressMsg{Text: key}
			_, _ = app.handleKeyPress(keyMsg)

			// Should still be in non-quitting state
			if app.quitting {
				t.Error("App should not quit when modal is open")
			}

			// Modal should still be visible
			if !app.taskModal.IsVisible() {
				t.Error("Modal should remain visible")
			}
		})
	}
}

// TestApp_TaskModal_QuitAfterClose tests normal quit behavior after modal is closed
func TestApp_TaskModal_QuitAfterClose(t *testing.T) {
	app := &App{
		taskModal: NewTaskModal(),
		dialog:    NewDialog(),
		quitting:  false,
	}

	// Open and close modal
	task := &session.Task{
		ID:      "test123",
		Content: "Test task",
	}
	app.taskModal.SetTask(task)
	app.taskModal.Close()

	// Now ctrl+c should quit
	qMsg := tea.KeyPressMsg{Text: "ctrl+c"}
	_, _ = app.handleKeyPress(qMsg)

	if !app.quitting {
		t.Error("App should quit when ctrl+c pressed and modal closed")
	}
}
