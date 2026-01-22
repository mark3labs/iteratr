package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestSidebar_PulseOnTaskStatusChange(t *testing.T) {
	// Create sidebar
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Set initial state with one task
	initialState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Do something",
				Status:  "remaining",
			},
		},
	}
	sidebar.SetState(initialState)

	// Pulse should not be active initially
	if sidebar.pulse.IsActive() {
		t.Error("Pulse should not be active initially")
	}

	// Change task status
	updatedState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Do something",
				Status:  "in_progress",
			},
		},
	}
	sidebar.SetState(updatedState)

	// Check that needsPulse flag is set
	if !sidebar.needsPulse {
		t.Error("needsPulse should be true after task status change")
	}

	// Trigger an Update to start the pulse
	cmd := sidebar.Update(tea.KeyPressMsg{})
	if cmd == nil {
		t.Error("Update should return pulse start command")
	}

	// Pulse should now be active
	if !sidebar.pulse.IsActive() {
		t.Error("Pulse should be active after starting")
	}

	// needsPulse should be cleared
	if sidebar.needsPulse {
		t.Error("needsPulse should be false after pulse starts")
	}
}

func TestSidebar_PulseOnNewTask(t *testing.T) {
	// Create sidebar
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Set initial state with no tasks
	initialState := &session.State{
		Tasks: map[string]*session.Task{},
	}
	sidebar.SetState(initialState)

	// Add a new task
	newState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "New task",
				Status:  "remaining",
			},
		},
	}
	sidebar.SetState(newState)

	// Check that needsPulse flag is set
	if !sidebar.needsPulse {
		t.Error("needsPulse should be true after new task")
	}

	// Start pulse
	sidebar.Update(tea.KeyPressMsg{})

	// Pulse should be active
	if !sidebar.pulse.IsActive() {
		t.Error("Pulse should be active for new task")
	}
}

func TestSidebar_NoPulseOnSameStatus(t *testing.T) {
	// Create sidebar
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Set initial empty state
	emptyState := &session.State{
		Tasks: map[string]*session.Task{},
	}
	sidebar.SetState(emptyState)

	// Set state with a task (will trigger pulse as it's "new")
	initialState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Do something",
				Status:  "remaining",
			},
		},
	}
	sidebar.SetState(initialState)

	// Trigger update to start initial pulse (new task)
	sidebar.Update(tea.KeyPressMsg{})

	// Wait for pulse to complete
	for i := 0; i < 15; i++ {
		if sidebar.pulse.IsActive() {
			sidebar.Update(PulseMsg{})
		}
	}

	// Set same state again
	sidebar.SetState(initialState)

	// needsPulse should NOT be set (same status)
	if sidebar.needsPulse {
		t.Error("needsPulse should not be set for same status")
	}
}

func TestSidebar_PulseIntensity(t *testing.T) {
	// Create sidebar
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Set state to trigger pulse
	state := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Task",
				Status:  "in_progress",
			},
		},
	}
	sidebar.SetState(state)

	// Start pulse
	sidebar.Update(tea.KeyPressMsg{})

	// Verify intensity is valid
	intensity := sidebar.pulse.Intensity()
	if intensity < 0 || intensity > 1 {
		t.Errorf("Pulse intensity should be between 0 and 1, got %f", intensity)
	}
}

func TestSidebar_NoPulseOnInitialLoad(t *testing.T) {
	// Create sidebar
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Set initial state directly (no previous state)
	initialState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Do something",
				Status:  "remaining",
			},
		},
	}
	sidebar.SetState(initialState)

	// needsPulse should NOT be set on initial load (no oldState to compare against)
	if sidebar.needsPulse {
		t.Error("needsPulse should not be set on initial state load")
	}

	// But pulsedTaskIDs should be populated for future comparisons
	if len(sidebar.pulsedTaskIDs) != 1 {
		t.Errorf("pulsedTaskIDs should have 1 entry, got %d", len(sidebar.pulsedTaskIDs))
	}
	if sidebar.pulsedTaskIDs["task1"] != "remaining" {
		t.Errorf("pulsedTaskIDs[task1] should be 'remaining', got '%s'", sidebar.pulsedTaskIDs["task1"])
	}
}

func TestSidebar_TaskStatusTrackedAcrossUpdates(t *testing.T) {
	// Create sidebar
	sidebar := NewSidebar()
	sidebar.SetSize(40, 30)

	// Set initial state
	initialState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Do something",
				Status:  "remaining",
			},
		},
	}
	sidebar.SetState(initialState)

	// Verify pulsedTaskIDs is populated
	if sidebar.pulsedTaskIDs["task1"] != "remaining" {
		t.Errorf("After initial load, pulsedTaskIDs[task1] should be 'remaining', got '%s'", sidebar.pulsedTaskIDs["task1"])
	}

	// Update with same status - should not trigger pulse
	sameState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Do something",
				Status:  "remaining",
			},
		},
	}
	sidebar.SetState(sameState)

	if sidebar.needsPulse {
		t.Error("needsPulse should not be set when status hasn't changed")
	}

	// Update with changed status - should trigger pulse
	changedState := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Do something",
				Status:  "in_progress",
			},
		},
	}
	sidebar.SetState(changedState)

	if !sidebar.needsPulse {
		t.Error("needsPulse should be set when status changed")
	}

	// Verify pulsedTaskIDs is updated
	if sidebar.pulsedTaskIDs["task1"] != "in_progress" {
		t.Errorf("After status change, pulsedTaskIDs[task1] should be 'in_progress', got '%s'", sidebar.pulsedTaskIDs["task1"])
	}
}
