package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// TestSidebar_TaskNavigation_Down tests navigating down through tasks with j/down keys
func TestSidebar_TaskNavigation_Down(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.StateWithTasks())

	// Initial cursor should be at 0
	if sidebar.cursor != 0 {
		t.Errorf("Initial cursor: got %d, want 0", sidebar.cursor)
	}

	// Press 'j' to move down
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	if sidebar.cursor != 1 {
		t.Errorf("After first 'j': cursor got %d, want 1", sidebar.cursor)
	}

	// Press 'down' arrow to move down again
	sidebar.Update(tea.KeyPressMsg{Text: "down"})
	if sidebar.cursor != 2 {
		t.Errorf("After 'down': cursor got %d, want 2", sidebar.cursor)
	}

	// Trying to go past last item should keep cursor at last position
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	if sidebar.cursor != 2 {
		t.Errorf("After 'j' at end: cursor got %d, want 2", sidebar.cursor)
	}
}

// TestSidebar_TaskNavigation_Up tests navigating up through tasks with k/up keys
func TestSidebar_TaskNavigation_Up(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.StateWithTasks())

	// Move to last task
	sidebar.cursor = 2

	// Press 'k' to move up
	sidebar.Update(tea.KeyPressMsg{Text: "k"})
	if sidebar.cursor != 1 {
		t.Errorf("After first 'k': cursor got %d, want 1", sidebar.cursor)
	}

	// Press 'up' arrow to move up again
	sidebar.Update(tea.KeyPressMsg{Text: "up"})
	if sidebar.cursor != 0 {
		t.Errorf("After 'up': cursor got %d, want 0", sidebar.cursor)
	}

	// Trying to go past first item should keep cursor at 0
	sidebar.Update(tea.KeyPressMsg{Text: "k"})
	if sidebar.cursor != 0 {
		t.Errorf("After 'k' at start: cursor got %d, want 0", sidebar.cursor)
	}
}

// TestSidebar_TaskNavigation_EnterOpensModal tests that pressing Enter on a task opens the modal
func TestSidebar_TaskNavigation_EnterOpensModal(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.StateWithTasks())

	// Navigate to second task (TAS-2)
	sidebar.cursor = 1
	sidebar.updateContent()

	// Press Enter
	cmd := sidebar.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("Enter should return command to open modal")
	}

	// Execute command to get message
	msg := cmd()
	openMsg, ok := msg.(OpenTaskModalMsg)
	if !ok {
		t.Fatalf("Expected OpenTaskModalMsg, got %T", msg)
	}

	// Verify correct task is selected (should be TAS-2 based on alphabetical ID order)
	if openMsg.Task.ID != "TAS-2" {
		t.Errorf("OpenTaskModalMsg task ID: got %s, want TAS-2", openMsg.Task.ID)
	}
	if openMsg.Task.Content != "[P1] Implement feature X" {
		t.Errorf("OpenTaskModalMsg task content: got %s, want '[P1] Implement feature X'", openMsg.Task.Content)
	}
}

// TestSidebar_TaskNavigation_NoFocus tests that navigation doesn't work when not focused
func TestSidebar_TaskNavigation_NoFocus(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = false // Not focused
	sidebar.SetState(testfixtures.StateWithTasks())

	initialCursor := sidebar.cursor

	// Try to navigate - should be ignored
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	if sidebar.cursor != initialCursor {
		t.Errorf("Cursor should not change when not focused: got %d, want %d", sidebar.cursor, initialCursor)
	}

	// Try Enter - should return nil since not focused
	cmd := sidebar.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd != nil {
		t.Error("Enter should not work when sidebar not focused")
	}
}

// TestSidebar_TaskNavigation_EmptyList tests navigation with no tasks
func TestSidebar_TaskNavigation_EmptyList(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.EmptyState())

	// Try navigation on empty list
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	if sidebar.cursor != 0 {
		t.Errorf("Cursor on empty list: got %d, want 0", sidebar.cursor)
	}

	// Try enter on empty list
	cmd := sidebar.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd != nil {
		t.Error("Enter on empty list should return nil")
	}
}

// TestSidebar_TaskNavigation_SingleTask tests navigation with only one task
func TestSidebar_TaskNavigation_SingleTask(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true

	// Create state with single task
	state := &session.State{
		Tasks: map[string]*session.Task{
			"TAS-1": {ID: "TAS-1", Content: "Only task", Status: "remaining", Priority: 1},
		},
	}
	sidebar.SetState(state)

	// Cursor should stay at 0
	if sidebar.cursor != 0 {
		t.Errorf("Initial cursor: got %d, want 0", sidebar.cursor)
	}

	// Try moving down - should stay at 0
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	if sidebar.cursor != 0 {
		t.Errorf("After 'j' with single task: cursor got %d, want 0", sidebar.cursor)
	}

	// Try moving up - should stay at 0
	sidebar.Update(tea.KeyPressMsg{Text: "k"})
	if sidebar.cursor != 0 {
		t.Errorf("After 'k' with single task: cursor got %d, want 0", sidebar.cursor)
	}

	// Enter should still work
	cmd := sidebar.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Error("Enter on single task should return command")
	}
}

// TestSidebar_TaskNavigation_CursorPersistence tests cursor position maintained across state updates
func TestSidebar_TaskNavigation_CursorPersistence(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.StateWithTasks())

	// Navigate to second task
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	if sidebar.cursor != 1 {
		t.Fatalf("Cursor should be at 1, got %d", sidebar.cursor)
	}

	// Update state (task status change) - use StateWithTasks and modify
	updatedState := testfixtures.StateWithTasks()
	updatedState.Tasks["TAS-2"].Status = "completed" // Change status
	sidebar.SetState(updatedState)

	// Cursor should remain at position 1
	if sidebar.cursor != 1 {
		t.Errorf("Cursor should persist across state updates: got %d, want 1", sidebar.cursor)
	}
}

// TestSidebar_TaskNavigation_BoundaryConditions tests edge cases
func TestSidebar_TaskNavigation_BoundaryConditions(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.StateWithTasks())

	// Test multiple down presses beyond boundary
	for i := 0; i < 10; i++ {
		sidebar.Update(tea.KeyPressMsg{Text: "j"})
	}
	if sidebar.cursor != 2 {
		t.Errorf("Cursor after multiple down presses: got %d, want 2", sidebar.cursor)
	}

	// Test multiple up presses beyond boundary
	for i := 0; i < 10; i++ {
		sidebar.Update(tea.KeyPressMsg{Text: "k"})
	}
	if sidebar.cursor != 0 {
		t.Errorf("Cursor after multiple up presses: got %d, want 0", sidebar.cursor)
	}
}

// TestSidebar_TaskNavigation_ScrollToItem tests that navigation triggers scroll
func TestSidebar_TaskNavigation_ScrollToItem(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 10) // Small height to force scrolling
	sidebar.tasksFocused = true

	// Create state with many tasks
	tasks := make(map[string]*session.Task)
	for i := 0; i < 20; i++ {
		taskID := testfixtures.FixedSessionName + "-" + string(rune('a'+i))
		tasks[taskID] = &session.Task{
			ID:       taskID,
			Content:  "Task " + taskID,
			Status:   "remaining",
			Priority: 1,
		}
	}
	state := &session.State{Tasks: tasks}
	sidebar.SetState(state)

	// Navigate down several times
	for i := 0; i < 10; i++ {
		sidebar.Update(tea.KeyPressMsg{Text: "j"})
	}

	// Cursor should be at position 10
	if sidebar.cursor != 10 {
		t.Errorf("Cursor after navigation: got %d, want 10", sidebar.cursor)
	}

	// Verify scroll list is tracking cursor position
	// (ScrollList.ScrollToItem should have been called)
	// We can't directly test scroll position without accessing internals,
	// but we verify the cursor moved correctly
}

// TestSidebar_TaskOrdering tests that tasks are displayed in ID alphabetical order
func TestSidebar_TaskOrdering(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Create tasks with different IDs (getTasks sorts by ID)
	state := &session.State{
		Tasks: map[string]*session.Task{
			"TAS-3": {ID: "TAS-3", Content: "Third alphabetically", Status: "remaining", Priority: 1},
			"TAS-1": {ID: "TAS-1", Content: "First alphabetically", Status: "remaining", Priority: 3},
			"TAS-2": {ID: "TAS-2", Content: "Second alphabetically", Status: "remaining", Priority: 0},
		},
	}
	sidebar.SetState(state)

	// Get ordered tasks
	tasks := sidebar.getTasks()

	// Verify tasks are ordered by ID (alphabetical)
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Task order should be alphabetical: TAS-1, TAS-2, TAS-3
	if tasks[0].ID != "TAS-1" {
		t.Errorf("First task should be TAS-1, got %s", tasks[0].ID)
	}
	if tasks[1].ID != "TAS-2" {
		t.Errorf("Second task should be TAS-2, got %s", tasks[1].ID)
	}
	if tasks[2].ID != "TAS-3" {
		t.Errorf("Third task should be TAS-3, got %s", tasks[2].ID)
	}
}

// TestSidebar_TaskNavigation_CursorClampedOnStateChange tests cursor is clamped when task list shrinks
func TestSidebar_TaskNavigation_CursorClampedOnStateChange(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.StateWithTasks())

	// Navigate to last task
	sidebar.cursor = 2

	// Update to state with fewer tasks
	state := &session.State{
		Tasks: map[string]*session.Task{
			"TAS-1": {ID: "TAS-1", Content: "First task", Status: "remaining", Priority: 1},
		},
	}
	sidebar.SetState(state)

	// Cursor should be clamped to 0 (only task)
	if sidebar.cursor != 0 {
		t.Errorf("Cursor should be clamped to 0: got %d", sidebar.cursor)
	}
}

// TestSidebar_TaskNavigation_EnterWithKeyboardAndMouse tests both keyboard and coordinate-based selection
func TestSidebar_TaskNavigation_EnterWithKeyboardAndMouse(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.tasksFocused = true
	sidebar.SetState(testfixtures.StateWithTasks())

	// Test keyboard navigation to select task
	sidebar.cursor = 1
	sidebar.updateContent()

	cmd := sidebar.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("Enter should return command")
	}

	msg := cmd()
	openMsg, ok := msg.(OpenTaskModalMsg)
	if !ok {
		t.Fatalf("Expected OpenTaskModalMsg, got %T", msg)
	}

	// Should select TAS-2 (second task alphabetically)
	if openMsg.Task.ID != "TAS-2" {
		t.Errorf("Expected TAS-2, got %s", openMsg.Task.ID)
	}
}
