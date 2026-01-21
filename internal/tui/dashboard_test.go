package tui

import (
	"testing"

	"github.com/mark3labs/iteratr/internal/session"
)

func TestDashboard_Render(t *testing.T) {
	tests := []struct {
		name      string
		dashboard *Dashboard
		wantEmpty bool
	}{
		{
			name: "renders with session name and iteration",
			dashboard: &Dashboard{
				sessionName: "test-session",
				iteration:   5,
			},
			wantEmpty: false,
		},
		{
			name: "renders with state",
			dashboard: &Dashboard{
				sessionName: "test-session",
				iteration:   3,
				state: &session.State{
					Session: "test-session",
					Tasks: map[string]*session.Task{
						"task1": {
							ID:      "task1abc",
							Content: "Test task",
							Status:  "remaining",
						},
					},
				},
			},
			wantEmpty: false,
		},
		{
			name: "renders without state",
			dashboard: &Dashboard{
				sessionName: "test-session",
				iteration:   1,
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.dashboard.Render()
			if tt.wantEmpty && output != "" {
				t.Errorf("expected empty output, got: %s", output)
			}
			if !tt.wantEmpty && output == "" {
				t.Error("expected non-empty output, got empty string")
			}
		})
	}
}

func TestDashboard_RenderSessionInfo(t *testing.T) {
	d := &Dashboard{
		sessionName: "my-session",
		iteration:   42,
	}

	output := d.renderSessionInfo()
	if output == "" {
		t.Error("expected non-empty session info")
	}

	// Basic smoke test - should contain session name
	// We don't want to test lipgloss styling details, just that content is present
	// Note: lipgloss styles are stripped in tests, so we just check structure
}

func TestDashboard_GetTaskStats(t *testing.T) {
	tests := []struct {
		name     string
		state    *session.State
		wantZero bool
		expected taskStats
	}{
		{
			name: "counts tasks correctly",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "remaining"},
					"t2": {ID: "t2", Status: "in_progress"},
					"t3": {ID: "t3", Status: "completed"},
					"t4": {ID: "t4", Status: "blocked"},
					"t5": {ID: "t5", Status: "completed"},
				},
			},
			expected: taskStats{
				Total:      5,
				Remaining:  1,
				InProgress: 1,
				Completed:  2,
				Blocked:    1,
			},
		},
		{
			name:     "handles empty task list",
			state:    &session.State{Tasks: map[string]*session.Task{}},
			wantZero: true,
			expected: taskStats{
				Total:      0,
				Remaining:  0,
				InProgress: 0,
				Completed:  0,
				Blocked:    0,
			},
		},
		{
			name: "handles only completed tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "completed"},
				},
			},
			expected: taskStats{
				Total:      2,
				Remaining:  0,
				InProgress: 0,
				Completed:  2,
				Blocked:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			stats := d.getTaskStats()

			if stats.Total != tt.expected.Total {
				t.Errorf("Total: got %d, want %d", stats.Total, tt.expected.Total)
			}
			if stats.Remaining != tt.expected.Remaining {
				t.Errorf("Remaining: got %d, want %d", stats.Remaining, tt.expected.Remaining)
			}
			if stats.InProgress != tt.expected.InProgress {
				t.Errorf("InProgress: got %d, want %d", stats.InProgress, tt.expected.InProgress)
			}
			if stats.Completed != tt.expected.Completed {
				t.Errorf("Completed: got %d, want %d", stats.Completed, tt.expected.Completed)
			}
			if stats.Blocked != tt.expected.Blocked {
				t.Errorf("Blocked: got %d, want %d", stats.Blocked, tt.expected.Blocked)
			}
		})
	}
}

func TestDashboard_RenderProgressIndicator(t *testing.T) {
	tests := []struct {
		name  string
		state *session.State
	}{
		{
			name: "renders progress with tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "remaining"},
				},
			},
		},
		{
			name: "renders progress with no tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{},
			},
		},
		{
			name: "renders progress with all completed",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "completed"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			output := d.renderProgressIndicator()
			if output == "" {
				t.Error("expected non-empty progress indicator")
			}
		})
	}
}

func TestDashboard_RenderCurrentTask(t *testing.T) {
	tests := []struct {
		name      string
		state     *session.State
		wantEmpty bool
	}{
		{
			name: "renders current in_progress task",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {
						ID:      "abcdef1234567890",
						Content: "Current task",
						Status:  "in_progress",
					},
				},
			},
			wantEmpty: false,
		},
		{
			name: "returns empty when no in_progress task",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Content: "Task 1", Status: "completed"},
				},
			},
			wantEmpty: true,
		},
		{
			name:      "returns empty when no tasks",
			state:     &session.State{Tasks: map[string]*session.Task{}},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			output := d.renderCurrentTask()
			if tt.wantEmpty && output != "" {
				t.Errorf("expected empty output, got: %s", output)
			}
			if !tt.wantEmpty && output == "" {
				t.Error("expected non-empty output, got empty string")
			}
		})
	}
}

func TestDashboard_UpdateState(t *testing.T) {
	d := NewDashboard()

	state := &session.State{
		Session: "new-session",
		Tasks:   map[string]*session.Task{},
	}

	d.UpdateState(state)

	if d.state != state {
		t.Error("state was not updated")
	}
	if d.sessionName != "new-session" {
		t.Errorf("session name: got %s, want new-session", d.sessionName)
	}
}

func TestDashboard_SetIteration(t *testing.T) {
	d := NewDashboard()

	d.SetIteration(10)

	if d.iteration != 10 {
		t.Errorf("iteration: got %d, want 10", d.iteration)
	}
}

func TestDashboard_UpdateSize(t *testing.T) {
	d := NewDashboard()

	d.UpdateSize(100, 50)

	if d.width != 100 {
		t.Errorf("width: got %d, want 100", d.width)
	}
	if d.height != 50 {
		t.Errorf("height: got %d, want 50", d.height)
	}
}

func TestNewDashboard(t *testing.T) {
	d := NewDashboard()

	if d == nil {
		t.Fatal("expected non-nil dashboard")
	}
	if d.sessionName != "" {
		t.Errorf("expected empty session name, got %s", d.sessionName)
	}
	if d.iteration != 0 {
		t.Errorf("expected iteration 0, got %d", d.iteration)
	}
}

func TestDashboard_RenderTaskStats(t *testing.T) {
	tests := []struct {
		name      string
		state     *session.State
		wantEmpty bool
	}{
		{
			name: "renders stats with mixed tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "remaining"},
					"t2": {ID: "t2", Status: "in_progress"},
					"t3": {ID: "t3", Status: "completed"},
				},
			},
			wantEmpty: false,
		},
		{
			name:      "returns empty with no tasks",
			state:     &session.State{Tasks: map[string]*session.Task{}},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			output := d.renderTaskStats()
			if tt.wantEmpty && output != "" {
				t.Errorf("expected empty output, got: %s", output)
			}
			if !tt.wantEmpty && output == "" {
				t.Error("expected non-empty output")
			}
		})
	}
}
