package session

import (
	"context"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
)

func TestTaskOperations(t *testing.T) {
	// Setup: Create embedded NATS and store
	ctx := context.Background()
	ns, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nats.CreateJetStream(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream: %v", err)
	}

	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("failed to setup stream: %v", err)
	}

	store := NewStore(js, stream)
	session := "test-session"

	t.Run("TaskAdd creates task with default status", func(t *testing.T) {
		task, err := store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Implement feature X",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		if task.ID == "" {
			t.Error("expected task ID to be set")
		}
		if task.Content != "Implement feature X" {
			t.Errorf("expected content 'Implement feature X', got '%s'", task.Content)
		}
		if task.Status != "remaining" {
			t.Errorf("expected default status 'remaining', got '%s'", task.Status)
		}
		if task.Iteration != 1 {
			t.Errorf("expected iteration 1, got %d", task.Iteration)
		}
	})

	t.Run("TaskAdd respects explicit status", func(t *testing.T) {
		task, err := store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Fix bug Y",
			Status:    "in_progress",
			Iteration: 2,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		if task.Status != "in_progress" {
			t.Errorf("expected status 'in_progress', got '%s'", task.Status)
		}
	})

	t.Run("TaskAdd validates status", func(t *testing.T) {
		_, err := store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Invalid task",
			Status:    "invalid_status",
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for invalid status")
		}
	})

	t.Run("TaskList groups tasks by status", func(t *testing.T) {
		// Add a few more tasks
		store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Task 1",
			Status:    "remaining",
			Iteration: 1,
		})
		store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Task 2",
			Status:    "completed",
			Iteration: 1,
		})
		store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Task 3",
			Status:    "blocked",
			Iteration: 1,
		})

		result, err := store.TaskList(ctx, session)
		if err != nil {
			t.Fatalf("TaskList failed: %v", err)
		}

		// We should have tasks in multiple statuses
		if len(result.Remaining) == 0 {
			t.Error("expected at least one remaining task")
		}
		if len(result.InProgress) == 0 {
			t.Error("expected at least one in_progress task")
		}
		if len(result.Completed) == 0 {
			t.Error("expected at least one completed task")
		}
		if len(result.Blocked) == 0 {
			t.Error("expected at least one blocked task")
		}
	})

	t.Run("TaskStatus updates task status", func(t *testing.T) {
		// Create a task
		task, err := store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Task to update",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Update its status
		err = store.TaskStatus(ctx, session, TaskStatusParams{
			ID:        task.ID,
			Status:    "completed",
			Iteration: 2,
		})
		if err != nil {
			t.Fatalf("TaskStatus failed: %v", err)
		}

		// Verify the update
		state, err := store.LoadState(ctx, session)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		updatedTask := state.Tasks[task.ID]
		if updatedTask == nil {
			t.Fatalf("task not found in state")
		}
		if updatedTask.Status != "completed" {
			t.Errorf("expected status 'completed', got '%s'", updatedTask.Status)
		}
		if updatedTask.Iteration != 2 {
			t.Errorf("expected iteration 2, got %d", updatedTask.Iteration)
		}
	})

	t.Run("TaskStatus supports prefix matching", func(t *testing.T) {
		// Create a task
		task, err := store.TaskAdd(ctx, session, TaskAddParams{
			Content:   "Task for prefix test",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Use prefix (at least 8 chars)
		if len(task.ID) < 8 {
			t.Skip("task ID too short for prefix test")
		}
		prefix := task.ID[:8]

		// Update using prefix
		err = store.TaskStatus(ctx, session, TaskStatusParams{
			ID:        prefix,
			Status:    "in_progress",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskStatus with prefix failed: %v", err)
		}

		// Verify the update
		state, err := store.LoadState(ctx, session)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		updatedTask := state.Tasks[task.ID]
		if updatedTask.Status != "in_progress" {
			t.Errorf("expected status 'in_progress', got '%s'", updatedTask.Status)
		}
	})

	t.Run("TaskStatus rejects short prefix", func(t *testing.T) {
		err := store.TaskStatus(ctx, session, TaskStatusParams{
			ID:        "1234567", // Only 7 chars
			Status:    "completed",
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for short prefix")
		}
	})

	t.Run("TaskStatus rejects non-existent ID", func(t *testing.T) {
		err := store.TaskStatus(ctx, session, TaskStatusParams{
			ID:        "99999999",
			Status:    "completed",
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for non-existent task")
		}
	})
}
