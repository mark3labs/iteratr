package testfixtures

import (
	"testing"

	"github.com/mark3labs/iteratr/internal/session"
)

func TestEmptyState(t *testing.T) {
	t.Parallel()

	state := EmptyState()

	// Verify basic fields
	if state.Session != FixedSessionName {
		t.Errorf("Expected session %q, got %q", FixedSessionName, state.Session)
	}
	if state.Complete {
		t.Error("Expected empty state to not be complete")
	}

	// Verify empty collections
	if len(state.Tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(state.Tasks))
	}
	if state.TaskCounter != 0 {
		t.Errorf("Expected TaskCounter 0, got %d", state.TaskCounter)
	}
	if len(state.Notes) != 0 {
		t.Errorf("Expected 0 notes, got %d", len(state.Notes))
	}
	if state.NoteCounter != 0 {
		t.Errorf("Expected NoteCounter 0, got %d", state.NoteCounter)
	}
	if len(state.Iterations) != 0 {
		t.Errorf("Expected 0 iterations, got %d", len(state.Iterations))
	}
}

func TestStateWithTasks(t *testing.T) {
	t.Parallel()

	state := StateWithTasks()

	// Verify basic fields
	if state.Session != FixedSessionName {
		t.Errorf("Expected session %q, got %q", FixedSessionName, state.Session)
	}
	if state.Complete {
		t.Error("Expected state to not be complete")
	}

	// Verify task count and counter
	if len(state.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(state.Tasks))
	}
	if state.TaskCounter != 3 {
		t.Errorf("Expected TaskCounter 3, got %d", state.TaskCounter)
	}

	// Verify task states
	testCases := []struct {
		id       string
		status   string
		priority int
	}{
		{"TAS-1", "completed", 0},
		{"TAS-2", "in_progress", 1},
		{"TAS-3", "remaining", 2},
	}

	for _, tc := range testCases {
		task, ok := state.Tasks[tc.id]
		if !ok {
			t.Errorf("Expected task %s to exist", tc.id)
			continue
		}
		if task.Status != tc.status {
			t.Errorf("Task %s: expected status %q, got %q", tc.id, tc.status, task.Status)
		}
		if task.Priority != tc.priority {
			t.Errorf("Task %s: expected priority %d, got %d", tc.id, tc.priority, task.Priority)
		}
	}

	// Verify task dependency
	if len(state.Tasks["TAS-3"].DependsOn) != 1 || state.Tasks["TAS-3"].DependsOn[0] != "TAS-2" {
		t.Errorf("Task TAS-3 should depend on TAS-2")
	}

	// Verify empty notes
	if len(state.Notes) != 0 {
		t.Errorf("Expected 0 notes, got %d", len(state.Notes))
	}

	// Verify iteration
	if len(state.Iterations) != 1 {
		t.Errorf("Expected 1 iteration, got %d", len(state.Iterations))
	}
	if state.Iterations[0].Number != 1 {
		t.Errorf("Expected iteration 1, got %d", state.Iterations[0].Number)
	}
}

func TestStateWithNotes(t *testing.T) {
	t.Parallel()

	state := StateWithNotes()

	// Verify basic fields
	if state.Session != FixedSessionName {
		t.Errorf("Expected session %q, got %q", FixedSessionName, state.Session)
	}

	// Verify note count and counter
	if len(state.Notes) != 4 {
		t.Errorf("Expected 4 notes, got %d", len(state.Notes))
	}
	if state.NoteCounter != 4 {
		t.Errorf("Expected NoteCounter 4, got %d", state.NoteCounter)
	}

	// Verify note types
	testCases := []struct {
		id  string
		typ string
	}{
		{"NOT-1", "learning"},
		{"NOT-2", "stuck"},
		{"NOT-3", "tip"},
		{"NOT-4", "decision"},
	}

	for i, tc := range testCases {
		if i >= len(state.Notes) {
			t.Errorf("Expected note %s to exist at index %d", tc.id, i)
			continue
		}
		note := state.Notes[i]
		if note.ID != tc.id {
			t.Errorf("Note %d: expected ID %q, got %q", i, tc.id, note.ID)
		}
		if note.Type != tc.typ {
			t.Errorf("Note %s: expected type %q, got %q", tc.id, tc.typ, note.Type)
		}
	}

	// Verify empty tasks
	if len(state.Tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(state.Tasks))
	}

	// Verify iteration
	if len(state.Iterations) != 1 {
		t.Errorf("Expected 1 iteration, got %d", len(state.Iterations))
	}
}

func TestFullState(t *testing.T) {
	t.Parallel()

	state := FullState()

	// Verify basic fields
	if state.Session != FixedSessionName {
		t.Errorf("Expected session %q, got %q", FixedSessionName, state.Session)
	}
	if state.Complete {
		t.Error("Expected state to not be complete")
	}

	// Verify task count and counter
	if len(state.Tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(state.Tasks))
	}
	if state.TaskCounter != 4 {
		t.Errorf("Expected TaskCounter 4, got %d", state.TaskCounter)
	}

	// Verify task states
	testCases := []struct {
		id       string
		status   string
		priority int
	}{
		{"TAS-1", "completed", 0},
		{"TAS-2", "in_progress", 1},
		{"TAS-3", "remaining", 2},
		{"TAS-4", "blocked", 3},
	}

	for _, tc := range testCases {
		task, ok := state.Tasks[tc.id]
		if !ok {
			t.Errorf("Expected task %s to exist", tc.id)
			continue
		}
		if task.Status != tc.status {
			t.Errorf("Task %s: expected status %q, got %q", tc.id, tc.status, task.Status)
		}
		if task.Priority != tc.priority {
			t.Errorf("Task %s: expected priority %d, got %d", tc.id, tc.priority, task.Priority)
		}
	}

	// Verify task with multiple dependencies
	if len(state.Tasks["TAS-4"].DependsOn) != 2 {
		t.Errorf("Task TAS-4 should have 2 dependencies, got %d", len(state.Tasks["TAS-4"].DependsOn))
	}

	// Verify note count and counter
	if len(state.Notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(state.Notes))
	}
	if state.NoteCounter != 2 {
		t.Errorf("Expected NoteCounter 2, got %d", state.NoteCounter)
	}

	// Verify iterations
	if len(state.Iterations) != 2 {
		t.Errorf("Expected 2 iterations, got %d", len(state.Iterations))
	}

	// Verify first iteration is complete
	if !state.Iterations[0].Complete {
		t.Error("Expected first iteration to be complete")
	}
	if state.Iterations[0].Summary != "Created test infrastructure" {
		t.Errorf("Unexpected summary for iteration 1: %q", state.Iterations[0].Summary)
	}

	// Verify second iteration is in progress
	if state.Iterations[1].Complete {
		t.Error("Expected second iteration to not be complete")
	}
	if state.Iterations[1].Number != 2 {
		t.Errorf("Expected iteration 2, got %d", state.Iterations[1].Number)
	}
}

func TestStateWithBlockedTasks(t *testing.T) {
	t.Parallel()

	state := StateWithBlockedTasks()

	// Verify basic fields
	if state.Session != FixedSessionName {
		t.Errorf("Expected session %q, got %q", FixedSessionName, state.Session)
	}

	// Verify task count
	if len(state.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(state.Tasks))
	}

	// Verify first task is blocked
	task1, ok := state.Tasks["TAS-1"]
	if !ok {
		t.Fatal("Expected task TAS-1 to exist")
	}
	if task1.Status != "blocked" {
		t.Errorf("Expected TAS-1 status %q, got %q", "blocked", task1.Status)
	}

	// Verify second task depends on first
	task2, ok := state.Tasks["TAS-2"]
	if !ok {
		t.Fatal("Expected task TAS-2 to exist")
	}
	if len(task2.DependsOn) != 1 || task2.DependsOn[0] != "TAS-1" {
		t.Errorf("Task TAS-2 should depend on TAS-1")
	}
}

func TestStateWithCompletedSession(t *testing.T) {
	t.Parallel()

	state := StateWithCompletedSession()

	// Verify session is complete
	if !state.Complete {
		t.Error("Expected session to be complete")
	}

	// Verify task is completed
	if len(state.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(state.Tasks))
	}

	task, ok := state.Tasks["TAS-1"]
	if !ok {
		t.Fatal("Expected task TAS-1 to exist")
	}
	if task.Status != "completed" {
		t.Errorf("Expected task status %q, got %q", "completed", task.Status)
	}

	// Verify iteration is complete
	if len(state.Iterations) != 1 {
		t.Errorf("Expected 1 iteration, got %d", len(state.Iterations))
	}
	if !state.Iterations[0].Complete {
		t.Error("Expected iteration to be complete")
	}
	if state.Iterations[0].Summary != "Completed all tasks" {
		t.Errorf("Unexpected iteration summary: %q", state.Iterations[0].Summary)
	}
}

func TestFixedValues(t *testing.T) {
	t.Parallel()

	// Verify fixed constants are set
	if FixedSessionName == "" {
		t.Error("FixedSessionName should not be empty")
	}
	if FixedGitHash == "" {
		t.Error("FixedGitHash should not be empty")
	}
	if FixedIteration == 0 {
		t.Error("FixedIteration should not be zero")
	}
	if FixedTime.IsZero() {
		t.Error("FixedTime should not be zero")
	}
}

func TestStateIndependence(t *testing.T) {
	t.Parallel()

	// Verify that multiple calls return independent states
	state1 := EmptyState()
	state2 := EmptyState()

	// Modify first state
	state1.TaskCounter = 99

	// Verify second state is not affected
	if state2.TaskCounter != 0 {
		t.Error("State builders should return independent instances")
	}

	// Test with StateWithTasks
	tasksState1 := StateWithTasks()
	tasksState2 := StateWithTasks()

	// Modify first state's task
	tasksState1.Tasks["TAS-1"].Status = "cancelled"

	// Verify second state is not affected
	if tasksState2.Tasks["TAS-1"].Status != "completed" {
		t.Error("State builders should return independent task instances")
	}
}

func TestAllBuildersCoverDifferentScenarios(t *testing.T) {
	t.Parallel()

	// Verify each builder creates distinct states
	empty := EmptyState()
	withTasks := StateWithTasks()
	withNotes := StateWithNotes()
	full := FullState()
	blocked := StateWithBlockedTasks()
	completed := StateWithCompletedSession()

	// Verify unique characteristics
	testCases := []struct {
		name       string
		state      *session.State
		hasTasks   bool
		hasNotes   bool
		hasIters   bool
		isComplete bool
	}{
		{"Empty", empty, false, false, false, false},
		{"WithTasks", withTasks, true, false, true, false},
		{"WithNotes", withNotes, false, true, true, false},
		{"Full", full, true, true, true, false},
		{"Blocked", blocked, true, false, true, false},
		{"Completed", completed, true, false, true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasTasks := len(tc.state.Tasks) > 0
			hasNotes := len(tc.state.Notes) > 0
			hasIters := len(tc.state.Iterations) > 0

			if hasTasks != tc.hasTasks {
				t.Errorf("%s: expected hasTasks=%v, got %v", tc.name, tc.hasTasks, hasTasks)
			}
			if hasNotes != tc.hasNotes {
				t.Errorf("%s: expected hasNotes=%v, got %v", tc.name, tc.hasNotes, hasNotes)
			}
			if hasIters != tc.hasIters {
				t.Errorf("%s: expected hasIters=%v, got %v", tc.name, tc.hasIters, hasIters)
			}
			if tc.state.Complete != tc.isComplete {
				t.Errorf("%s: expected Complete=%v, got %v", tc.name, tc.isComplete, tc.state.Complete)
			}
		})
	}
}
