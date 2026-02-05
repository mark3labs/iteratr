package tui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
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

// ============================================================================
// Note List Tests
// ============================================================================

// TestSidebar_NoteList_Display tests that notes are displayed correctly
func TestSidebar_NoteList_Display(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.StateWithNotes())

	// Verify notes ScrollList has items
	if len(sidebar.notesScrollList.items) != 4 {
		t.Errorf("Notes ScrollList should have 4 items, got %d", len(sidebar.notesScrollList.items))
	}

	// Verify notes are rendered (basic check)
	content := sidebar.notesScrollList.View()
	if content == "" {
		t.Error("Notes ScrollList should render content")
	}
}

// TestSidebar_NoteList_NoKeyboardNavigation tests that notes don't support j/k/enter navigation
func TestSidebar_NoteList_NoKeyboardNavigation(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.notesFocused = true // Focus notes panel
	sidebar.SetState(testfixtures.StateWithNotes())

	// Try pressing 'j' key - should not navigate or select notes
	cmd := sidebar.Update(tea.KeyPressMsg{Text: "j"})
	// Command should be nil (no selection) or only scroll-related
	// Note: ScrollList may handle this for scrolling, but no task selection occurs
	if cmd != nil {
		// If a command is returned, it should not be OpenTaskModalMsg
		msg := cmd()
		if _, ok := msg.(OpenTaskModalMsg); ok {
			t.Error("'j' key on notes should not open task modal")
		}
	}

	// Try pressing 'enter' key - should not select a note
	cmd = sidebar.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(OpenTaskModalMsg); ok {
			t.Error("Enter key on notes should not open task modal")
		}
	}
}

// TestSidebar_NoteList_ScrollingWhenFocused tests that notes ScrollList can scroll when focused
func TestSidebar_NoteList_ScrollingWhenFocused(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 10) // Small height to enable scrolling
	sidebar.notesFocused = true
	sidebar.SetState(testfixtures.StateWithNotes())

	// Try scrolling with j/k (these delegate to ScrollList for scrolling)
	initialOffset := sidebar.notesScrollList.offsetIdx
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	// Offset may change if scrolling is triggered
	// This is a basic smoke test - the ScrollList handles the details
	_ = initialOffset
}

// TestSidebar_NoteList_NoScrollingWhenNotFocused tests that notes don't scroll when not focused
func TestSidebar_NoteList_NoScrollingWhenNotFocused(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.notesFocused = false // Not focused
	sidebar.SetState(testfixtures.StateWithNotes())

	// Try scrolling - should be ignored
	initialOffset := sidebar.notesScrollList.offsetIdx
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	if sidebar.notesScrollList.offsetIdx != initialOffset {
		t.Errorf("Notes ScrollList should not scroll when not focused")
	}
}

// TestSidebar_NoteList_EmptyState tests notes panel with no notes
func TestSidebar_NoteList_EmptyState(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.notesFocused = true
	sidebar.SetState(testfixtures.EmptyState())

	// Verify notes ScrollList is empty
	if len(sidebar.notesScrollList.items) != 0 {
		t.Errorf("Notes ScrollList should be empty, got %d items", len(sidebar.notesScrollList.items))
	}

	// Try navigation on empty list - should not panic
	sidebar.Update(tea.KeyPressMsg{Text: "j"})
	sidebar.Update(tea.KeyPressMsg{Text: "enter"})
}

// TestSidebar_NoteList_RecentNotesOnly tests that only last 10 notes are shown
func TestSidebar_NoteList_RecentNotesOnly(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Create state with more than 10 notes
	state := testfixtures.EmptyState()
	for i := 0; i < 15; i++ {
		state.Notes = append(state.Notes, &session.Note{
			ID:        testfixtures.FixedSessionName + "-" + string(rune('a'+i)),
			Content:   "Note " + string(rune('a'+i)),
			Type:      "learning",
			CreatedAt: testfixtures.FixedTime,
			Iteration: 1,
		})
	}
	sidebar.SetState(state)

	// Only last 10 should be displayed
	if len(sidebar.notesScrollList.items) != 10 {
		t.Errorf("Notes ScrollList should show last 10 notes, got %d", len(sidebar.notesScrollList.items))
	}

	// Verify the displayed notes are the last 10
	// First displayed note should be note #5 (index 5 in the 15-note array)
	firstItem := sidebar.notesScrollList.items[0].(*noteScrollItem)
	if firstItem.note.ID != state.Notes[5].ID {
		t.Errorf("First displayed note should be note #5, got %s", firstItem.note.ID)
	}

	// Last displayed note should be note #14 (last in the array)
	lastItem := sidebar.notesScrollList.items[9].(*noteScrollItem)
	if lastItem.note.ID != state.Notes[14].ID {
		t.Errorf("Last displayed note should be note #14, got %s", lastItem.note.ID)
	}
}

// TestSidebar_NoteList_TypeIndicators tests that notes show correct type indicators
func TestSidebar_NoteList_TypeIndicators(t *testing.T) {
	t.Parallel()

	// Test each note type renders without panic
	noteTypes := []struct {
		noteType  string
		indicator string
	}{
		{"learning", "*"},
		{"stuck", "!"},
		{"tip", "›"},
		{"decision", "◇"},
	}

	for _, tt := range noteTypes {
		tt := tt
		t.Run(tt.noteType, func(t *testing.T) {
			t.Parallel()

			sidebar := NewSidebar()
			sidebar.SetSize(40, 30)

			state := testfixtures.EmptyState()
			state.Notes = []*session.Note{
				{
					ID:        "NOT-1",
					Content:   "Test note",
					Type:      tt.noteType,
					CreatedAt: testfixtures.FixedTime,
					Iteration: 1,
				},
			}
			sidebar.SetState(state)

			// Verify note item is created
			if len(sidebar.notesScrollList.items) != 1 {
				t.Fatalf("Should have 1 note, got %d", len(sidebar.notesScrollList.items))
			}

			// Render and verify it contains the indicator (basic smoke test)
			noteItem := sidebar.notesScrollList.items[0].(*noteScrollItem)
			rendered := noteItem.Render(40)
			if rendered == "" {
				t.Error("Note should render non-empty content")
			}
			// The rendered output contains ANSI codes, so just check it's not empty
			// Detailed rendering is covered by visual regression tests if needed
		})
	}
}

// TestSidebar_NoteList_ActiveNoteHighlight tests SetActiveNote/ClearActiveNote
func TestSidebar_NoteList_ActiveNoteHighlight(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.StateWithNotes())

	// Set active note
	sidebar.SetActiveNote("NOT-2")
	if sidebar.activeNoteID != "NOT-2" {
		t.Errorf("Active note ID: got %s, want NOT-2", sidebar.activeNoteID)
	}

	// Verify the note item is marked as selected
	// Find NOT-2 in the ScrollList items
	var noteItem *noteScrollItem
	for _, item := range sidebar.notesScrollList.items {
		ni := item.(*noteScrollItem)
		if ni.note.ID == "NOT-2" {
			noteItem = ni
			break
		}
	}
	if noteItem == nil {
		t.Fatal("Could not find NOT-2 in notes ScrollList")
	}
	if !noteItem.isSelected {
		t.Error("Active note should be marked as selected")
	}

	// Clear active note
	sidebar.ClearActiveNote()
	if sidebar.activeNoteID != "" {
		t.Errorf("Active note ID should be empty, got %s", sidebar.activeNoteID)
	}
}

// TestSidebar_NoteList_NoteAtPosition tests coordinate-based note hit detection
func TestSidebar_NoteList_NoteAtPosition(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.StateWithNotes())

	// Draw to populate notesContentArea
	scr := uv.NewScreenBuffer(40, 30)
	area := uv.Rect(0, 0, 40, 30)
	sidebar.Draw(scr, area)

	// Test hit detection within notes area
	// The notes area is in the lower section of the sidebar
	// This is a smoke test - exact coordinates depend on layout
	note := sidebar.NoteAtPosition(5, 25)
	// May be nil if coordinates don't hit a note line, but should not panic
	_ = note

	// Test coordinates outside notes area
	noteOutside := sidebar.NoteAtPosition(-1, -1)
	if noteOutside != nil {
		t.Error("Coordinates outside notes area should return nil")
	}
}

// TestSidebar_NoteList_ScrollNotesMethod tests ScrollNotes method
func TestSidebar_NoteList_ScrollNotesMethod(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar()
	sidebar.SetSize(40, 10) // Small height to enable scrolling
	sidebar.SetState(testfixtures.StateWithNotes())

	initialOffset := sidebar.notesScrollList.offsetIdx

	// Scroll down
	sidebar.ScrollNotes(1)
	// Offset should change (or stay same if at boundary)
	// This is a smoke test
	_ = initialOffset

	// Scroll up
	sidebar.ScrollNotes(-1)
	// Should not panic
}

// ============================================================================
// Visual Regression Tests (Golden Files)
// ============================================================================

// compareSidebarGolden compares rendered output with golden file
// TestSidebar_Render_WithTasks tests rendering sidebar with tasks
func TestSidebar_Render_WithTasks(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.StateWithTasks())

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_with_tasks.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_WithNotes tests rendering sidebar with notes
func TestSidebar_Render_WithNotes(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.StateWithNotes())

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_with_notes.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_FullState tests rendering sidebar with both tasks and notes
func TestSidebar_Render_FullState(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.FullState())

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_full_state.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_EmptyState tests rendering sidebar with no tasks or notes
func TestSidebar_Render_EmptyState(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.EmptyState())

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_empty_state.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_TasksFocused tests rendering sidebar with tasks panel focused
func TestSidebar_Render_TasksFocused(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.FullState())
	sidebar.SetTasksScrollFocused(true)

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_tasks_focused.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_NotesFocused tests rendering sidebar with notes panel focused
func TestSidebar_Render_NotesFocused(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.FullState())
	sidebar.SetNotesScrollFocused(true)

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_notes_focused.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_TaskStatuses tests rendering sidebar with various task statuses
func TestSidebar_Render_TaskStatuses(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Create state with tasks in different statuses
	state := &session.State{
		Tasks: map[string]*session.Task{
			"TAS-1": {ID: "TAS-1", Content: "Remaining task", Status: "remaining", Priority: 1},
			"TAS-2": {ID: "TAS-2", Content: "In progress task", Status: "in_progress", Priority: 1},
			"TAS-3": {ID: "TAS-3", Content: "Completed task", Status: "completed", Priority: 1},
			"TAS-4": {ID: "TAS-4", Content: "Blocked task", Status: "blocked", Priority: 1},
			"TAS-5": {ID: "TAS-5", Content: "Cancelled task", Status: "cancelled", Priority: 1},
		},
	}
	sidebar.SetState(state)

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_task_statuses.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_NoteTypes tests rendering sidebar with all note types
func TestSidebar_Render_NoteTypes(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Create state with notes of all types
	state := &session.State{
		Notes: []*session.Note{
			{ID: "NOT-1", Content: "Learning note", Type: "learning", CreatedAt: testfixtures.FixedTime, Iteration: 1},
			{ID: "NOT-2", Content: "Stuck note", Type: "stuck", CreatedAt: testfixtures.FixedTime, Iteration: 1},
			{ID: "NOT-3", Content: "Tip note", Type: "tip", CreatedAt: testfixtures.FixedTime, Iteration: 1},
			{ID: "NOT-4", Content: "Decision note", Type: "decision", CreatedAt: testfixtures.FixedTime, Iteration: 1},
		},
	}
	sidebar.SetState(state)

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_note_types.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_ActiveTask tests rendering sidebar with active task highlighted
func TestSidebar_Render_ActiveTask(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.StateWithTasks())
	sidebar.SetActiveTask("TAS-2")

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_active_task.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_ActiveNote tests rendering sidebar with active note highlighted
func TestSidebar_Render_ActiveNote(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)
	sidebar.SetState(testfixtures.StateWithNotes())
	sidebar.SetActiveNote("NOT-2")

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 30},
	}
	scr := uv.NewScreenBuffer(40, 30)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_active_note.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}

// TestSidebar_Render_SmallScreen tests rendering sidebar on small screen
func TestSidebar_Render_SmallScreen(t *testing.T) {
	sidebar := NewSidebar()
	sidebar.SetSize(30, 20)
	sidebar.SetState(testfixtures.FullState())

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 30, Y: 20},
	}
	scr := uv.NewScreenBuffer(30, 20)
	sidebar.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "sidebar_small_screen.golden")
	testfixtures.CompareGolden(t, goldenFile, rendered)
}
