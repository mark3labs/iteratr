package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestTaskList_GetFilteredTasks(t *testing.T) {
	state := &session.State{
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
			"t2": {ID: "t2", Content: "Task 2", Status: "in_progress"},
			"t3": {ID: "t3", Content: "Task 3", Status: "completed"},
			"t4": {ID: "t4", Content: "Task 4", Status: "blocked"},
		},
	}

	tests := []struct {
		name         string
		filterStatus string
		wantCount    int
	}{
		{
			name:         "filter all",
			filterStatus: "all",
			wantCount:    4,
		},
		{
			name:         "filter remaining",
			filterStatus: "remaining",
			wantCount:    1,
		},
		{
			name:         "filter in_progress",
			filterStatus: "in_progress",
			wantCount:    1,
		},
		{
			name:         "filter completed",
			filterStatus: "completed",
			wantCount:    1,
		},
		{
			name:         "filter blocked",
			filterStatus: "blocked",
			wantCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := &TaskList{
				state:        state,
				filterStatus: tt.filterStatus,
			}

			filtered := tl.getFilteredTasks()
			if len(filtered) != tt.wantCount {
				t.Errorf("got %d tasks, want %d", len(filtered), tt.wantCount)
			}
		})
	}
}

func TestTaskList_CycleFilter(t *testing.T) {
	tl := NewTaskList()

	// Verify initial filter
	if tl.filterStatus != "all" {
		t.Errorf("initial filter: got %s, want all", tl.filterStatus)
	}

	// Cycle through filters
	expectedFilters := []string{"remaining", "in_progress", "completed", "blocked", "all"}

	for i, expected := range expectedFilters {
		tl.cycleFilter()
		if tl.filterStatus != expected {
			t.Errorf("cycle %d: got %s, want %s", i+1, tl.filterStatus, expected)
		}
	}
}

func TestTaskList_HandleKeyPress(t *testing.T) {
	state := &session.State{
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
			"t2": {ID: "t2", Content: "Task 2", Status: "remaining"},
			"t3": {ID: "t3", Content: "Task 3", Status: "remaining"},
		},
	}

	tests := []struct {
		name           string
		key            string
		initialCursor  int
		expectedCursor int
	}{
		{
			name:           "j moves cursor down",
			key:            "j",
			initialCursor:  0,
			expectedCursor: 1,
		},
		{
			name:           "k moves cursor up",
			key:            "k",
			initialCursor:  2,
			expectedCursor: 1,
		},
		{
			name:           "g goes to top",
			key:            "g",
			initialCursor:  2,
			expectedCursor: 0,
		},
		{
			name:           "G goes to bottom",
			key:            "G",
			initialCursor:  0,
			expectedCursor: 2,
		},
		{
			name:           "j at end stays at end",
			key:            "j",
			initialCursor:  2,
			expectedCursor: 2,
		},
		{
			name:           "k at start stays at start",
			key:            "k",
			initialCursor:  0,
			expectedCursor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := &TaskList{
				state:        state,
				filterStatus: "all",
				cursor:       tt.initialCursor,
			}

			msg := tea.KeyPressMsg{Code: rune(tt.key[0]), Text: tt.key}
			tl.handleKeyPress(msg)

			if tl.cursor != tt.expectedCursor {
				t.Errorf("cursor: got %d, want %d", tl.cursor, tt.expectedCursor)
			}
		})
	}
}

func TestTaskList_HandleKeyPress_Filter(t *testing.T) {
	state := &session.State{
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
		},
	}

	tl := &TaskList{
		state:        state,
		filterStatus: "all",
		cursor:       0,
	}

	// Press 'f' to cycle filter
	msg := tea.KeyPressMsg{Code: 'f', Text: "f"}
	tl.handleKeyPress(msg)

	if tl.filterStatus != "remaining" {
		t.Errorf("filter status: got %s, want remaining", tl.filterStatus)
	}

	// Verify cursor and scroll are reset
	if tl.cursor != 0 {
		t.Errorf("cursor: got %d, want 0", tl.cursor)
	}
	if tl.scrollOffset != 0 {
		t.Errorf("scroll offset: got %d, want 0", tl.scrollOffset)
	}
}

func TestTaskList_GetFilterLabel(t *testing.T) {
	tests := []struct {
		filterStatus string
		expected     string
	}{
		{"all", "All Tasks"},
		{"remaining", "Remaining"},
		{"in_progress", "In Progress"},
		{"completed", "Completed"},
		{"blocked", "Blocked"},
		{"unknown", "All Tasks"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.filterStatus, func(t *testing.T) {
			tl := &TaskList{filterStatus: tt.filterStatus}
			got := tl.getFilterLabel()
			if got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestTaskList_Render(t *testing.T) {
	tests := []struct {
		name      string
		state     *session.State
		wantEmpty bool
	}{
		{
			name: "renders with tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
				},
			},
			wantEmpty: false,
		},
		{
			name:      "renders without state",
			state:     nil,
			wantEmpty: false, // Should render "No session loaded"
		},
		{
			name: "renders with empty task list",
			state: &session.State{
				Tasks: map[string]*session.Task{},
			},
			wantEmpty: false, // Should render "No tasks match current filter"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := &TaskList{state: tt.state, filterStatus: "all"}
			output := tl.Render()
			if output == "" {
				t.Error("expected non-empty output")
			}
		})
	}
}

func TestTaskList_RenderTask(t *testing.T) {
	task := &session.Task{
		ID:      "abcdef1234567890",
		Content: "Test task content",
		Status:  "remaining",
	}

	tl := NewTaskList()

	// Render unselected
	output := tl.renderTask(task, styleStatusRemaining, false)
	if output == "" {
		t.Error("expected non-empty output")
	}

	// Render selected
	outputSelected := tl.renderTask(task, styleStatusRemaining, true)
	if outputSelected == "" {
		t.Error("expected non-empty output for selected task")
	}
	if output == outputSelected {
		t.Error("selected and unselected task should render differently")
	}
}

func TestTaskList_UpdateSize(t *testing.T) {
	tl := NewTaskList()
	tl.UpdateSize(100, 50)

	if tl.width != 100 {
		t.Errorf("width: got %d, want 100", tl.width)
	}
	if tl.height != 50 {
		t.Errorf("height: got %d, want 50", tl.height)
	}
}

func TestTaskList_UpdateState(t *testing.T) {
	tl := NewTaskList()
	state := &session.State{
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
		},
	}

	tl.UpdateState(state)

	if tl.state != state {
		t.Error("state was not updated")
	}
}

func TestNewTaskList(t *testing.T) {
	tl := NewTaskList()

	if tl == nil {
		t.Fatal("expected non-nil task list")
	}
	if tl.filterStatus != "all" {
		t.Errorf("filter status: got %s, want all", tl.filterStatus)
	}
	if tl.cursor != 0 {
		t.Errorf("cursor: got %d, want 0", tl.cursor)
	}
	if tl.scrollOffset != 0 {
		t.Errorf("scroll offset: got %d, want 0", tl.scrollOffset)
	}
}

func TestTaskList_AdjustScroll(t *testing.T) {
	tl := &TaskList{
		height:       30, // ~10 tasks visible (30/3)
		cursor:       15,
		scrollOffset: 0,
	}

	// Scrolling down should adjust offset
	tl.adjustScroll()
	if tl.scrollOffset == 0 {
		t.Error("expected scroll offset to be adjusted when cursor moves down")
	}

	// Scrolling up
	tl.cursor = 0
	tl.adjustScroll()
	if tl.scrollOffset != 0 {
		t.Errorf("scroll offset: got %d, want 0 when cursor at top", tl.scrollOffset)
	}
}

func TestTaskList_GetStatusStyle(t *testing.T) {
	tl := NewTaskList()

	tests := []struct {
		status string
	}{
		{"remaining"},
		{"in_progress"},
		{"completed"},
		{"blocked"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			style := tl.getStatusStyle(tt.status)
			// Just verify it returns a style without panicking
			_ = style
		})
	}
}

func TestTaskList_RenderAllGroups(t *testing.T) {
	tests := []struct {
		name  string
		tasks []*session.Task
	}{
		{
			name: "renders tasks by status groups",
			tasks: []*session.Task{
				{ID: "t1", Content: "Task 1", Status: "remaining"},
				{ID: "t2", Content: "Task 2", Status: "in_progress"},
				{ID: "t3", Content: "Task 3", Status: "completed"},
				{ID: "t4", Content: "Task 4", Status: "blocked"},
			},
		},
		{
			name:  "renders empty task list",
			tasks: []*session.Task{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := NewTaskList()
			output := tl.renderAllGroups(tt.tasks)
			if output == "" && len(tt.tasks) > 0 {
				t.Error("expected non-empty output for non-empty task list")
			}
		})
	}
}

func TestTaskList_RenderFlatList(t *testing.T) {
	tasks := []*session.Task{
		{ID: "t1", Content: "Task 1", Status: "remaining"},
		{ID: "t2", Content: "Task 2", Status: "remaining"},
	}

	tl := NewTaskList()
	output := tl.renderFlatList(tasks)

	if output == "" {
		t.Error("expected non-empty output")
	}
}
