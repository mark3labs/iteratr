package session

import (
	"context"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
)

func TestTaskOperations(t *testing.T) {
	// Setup: Create embedded NATS and store
	ctx := context.Background()
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
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
		// Use a dedicated session to avoid conflicts with other tests
		prefixSession := "test-session-prefix"

		// Create a task
		task, err := store.TaskAdd(ctx, prefixSession, TaskAddParams{
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
		err = store.TaskStatus(ctx, prefixSession, TaskStatusParams{
			ID:        prefix,
			Status:    "in_progress",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskStatus with prefix failed: %v", err)
		}

		// Verify the update
		state, err := store.LoadState(ctx, prefixSession)
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

	t.Run("TaskPriority updates task priority", func(t *testing.T) {
		// Use a dedicated session
		prioritySession := "test-session-priority"

		// Create a task with default priority
		task, err := store.TaskAdd(ctx, prioritySession, TaskAddParams{
			Content:   "Task for priority test",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Default priority should be 2 (medium)
		state, err := store.LoadState(ctx, prioritySession)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}
		if state.Tasks[task.ID].Priority != 2 {
			t.Errorf("expected default priority 2, got %d", state.Tasks[task.ID].Priority)
		}

		// Update priority to high (1)
		err = store.TaskPriority(ctx, prioritySession, TaskPriorityParams{
			ID:        task.ID,
			Priority:  1,
			Iteration: 2,
		})
		if err != nil {
			t.Fatalf("TaskPriority failed: %v", err)
		}

		// Verify the update
		state, err = store.LoadState(ctx, prioritySession)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		if state.Tasks[task.ID].Priority != 1 {
			t.Errorf("expected priority 1, got %d", state.Tasks[task.ID].Priority)
		}
		if state.Tasks[task.ID].Iteration != 2 {
			t.Errorf("expected iteration 2, got %d", state.Tasks[task.ID].Iteration)
		}
	})

	t.Run("TaskPriority rejects invalid priority", func(t *testing.T) {
		// Use a dedicated session
		invalidPrioritySession := "test-session-invalid-priority"

		// Create a task
		task, err := store.TaskAdd(ctx, invalidPrioritySession, TaskAddParams{
			Content:   "Task for invalid priority test",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Try to set priority outside valid range
		err = store.TaskPriority(ctx, invalidPrioritySession, TaskPriorityParams{
			ID:        task.ID,
			Priority:  5, // Invalid - max is 4
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for invalid priority")
		}

		err = store.TaskPriority(ctx, invalidPrioritySession, TaskPriorityParams{
			ID:        task.ID,
			Priority:  -1, // Invalid - min is 0
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for negative priority")
		}
	})

	t.Run("TaskDepends adds dependency", func(t *testing.T) {
		// Use a dedicated session
		dependsSession := "test-session-depends"

		// Create two tasks
		task1, err := store.TaskAdd(ctx, dependsSession, TaskAddParams{
			Content:   "Base task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		task2, err := store.TaskAdd(ctx, dependsSession, TaskAddParams{
			Content:   "Dependent task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Add dependency: task2 depends on task1
		err = store.TaskDepends(ctx, dependsSession, TaskDependsParams{
			ID:        task2.ID,
			DependsOn: task1.ID,
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskDepends failed: %v", err)
		}

		// Verify the dependency
		state, err := store.LoadState(ctx, dependsSession)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		if len(state.Tasks[task2.ID].DependsOn) != 1 {
			t.Errorf("expected 1 dependency, got %d", len(state.Tasks[task2.ID].DependsOn))
		}
		if state.Tasks[task2.ID].DependsOn[0] != task1.ID {
			t.Errorf("expected dependency on %s, got %s", task1.ID, state.Tasks[task2.ID].DependsOn[0])
		}
	})

	t.Run("TaskDepends prevents self-dependency", func(t *testing.T) {
		// Use a dedicated session
		selfDepSession := "test-session-self-dep"

		// Create a task
		task, err := store.TaskAdd(ctx, selfDepSession, TaskAddParams{
			Content:   "Task for self-dep test",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Try to add self-dependency
		err = store.TaskDepends(ctx, selfDepSession, TaskDependsParams{
			ID:        task.ID,
			DependsOn: task.ID,
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for self-dependency")
		}
	})

	t.Run("TaskDepends prevents duplicate dependencies", func(t *testing.T) {
		// Use a dedicated session
		dupDepSession := "test-session-dup-dep"

		// Create two tasks
		task1, err := store.TaskAdd(ctx, dupDepSession, TaskAddParams{
			Content:   "Base task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		task2, err := store.TaskAdd(ctx, dupDepSession, TaskAddParams{
			Content:   "Dependent task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Add dependency twice
		err = store.TaskDepends(ctx, dupDepSession, TaskDependsParams{
			ID:        task2.ID,
			DependsOn: task1.ID,
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("First TaskDepends failed: %v", err)
		}

		// Adding same dependency again should succeed (it's idempotent) but not duplicate
		err = store.TaskDepends(ctx, dupDepSession, TaskDependsParams{
			ID:        task2.ID,
			DependsOn: task1.ID,
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("Second TaskDepends failed: %v", err)
		}

		// Verify only one dependency exists
		state, err := store.LoadState(ctx, dupDepSession)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		if len(state.Tasks[task2.ID].DependsOn) != 1 {
			t.Errorf("expected 1 dependency (no duplicates), got %d", len(state.Tasks[task2.ID].DependsOn))
		}
	})

	t.Run("TaskNext returns highest priority unblocked task", func(t *testing.T) {
		// Use a dedicated session
		nextSession := "test-session-task-next"

		// Create tasks with various priorities
		task1, err := store.TaskAdd(ctx, nextSession, TaskAddParams{
			Content:   "Low priority task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}
		// Set priority to low (3)
		store.TaskPriority(ctx, nextSession, TaskPriorityParams{
			ID:        task1.ID,
			Priority:  3,
			Iteration: 1,
		})

		task2, err := store.TaskAdd(ctx, nextSession, TaskAddParams{
			Content:   "High priority task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}
		// Set priority to high (1)
		store.TaskPriority(ctx, nextSession, TaskPriorityParams{
			ID:        task2.ID,
			Priority:  1,
			Iteration: 1,
		})

		task3, err := store.TaskAdd(ctx, nextSession, TaskAddParams{
			Content:   "Critical task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}
		// Set priority to critical (0)
		store.TaskPriority(ctx, nextSession, TaskPriorityParams{
			ID:        task3.ID,
			Priority:  0,
			Iteration: 1,
		})

		// TaskNext should return the critical priority task
		nextTask, err := store.TaskNext(ctx, nextSession)
		if err != nil {
			t.Fatalf("TaskNext failed: %v", err)
		}

		if nextTask == nil {
			t.Fatal("expected a task, got nil")
		}
		if nextTask.ID != task3.ID {
			t.Errorf("expected critical task %s, got %s", task3.ID, nextTask.ID)
		}
	})

	t.Run("TaskNext skips blocked tasks", func(t *testing.T) {
		// Use a dedicated session
		blockedSession := "test-session-task-next-blocked"

		// Create base task
		task1, err := store.TaskAdd(ctx, blockedSession, TaskAddParams{
			Content:   "Base task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}
		// Set to high priority
		store.TaskPriority(ctx, blockedSession, TaskPriorityParams{
			ID:        task1.ID,
			Priority:  0,
			Iteration: 1,
		})

		// Create dependent task with higher priority number (but would be higher if not blocked)
		task2, err := store.TaskAdd(ctx, blockedSession, TaskAddParams{
			Content:   "Dependent task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}
		// Add dependency
		store.TaskDepends(ctx, blockedSession, TaskDependsParams{
			ID:        task2.ID,
			DependsOn: task1.ID,
			Iteration: 1,
		})

		// TaskNext should return task1 (task2 is blocked by dependency)
		nextTask, err := store.TaskNext(ctx, blockedSession)
		if err != nil {
			t.Fatalf("TaskNext failed: %v", err)
		}

		if nextTask == nil {
			t.Fatal("expected a task, got nil")
		}
		if nextTask.ID != task1.ID {
			t.Errorf("expected task1 %s, got %s", task1.ID, nextTask.ID)
		}
	})

	t.Run("TaskNext returns nil when no ready tasks", func(t *testing.T) {
		// Use a dedicated session
		noReadySession := "test-session-no-ready"

		// Create a completed task
		task1, err := store.TaskAdd(ctx, noReadySession, TaskAddParams{
			Content:   "Completed task",
			Status:    "completed",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}

		// Create a task blocked by a non-existent dependency
		task2, err := store.TaskAdd(ctx, noReadySession, TaskAddParams{
			Content:   "Blocked task",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("TaskAdd failed: %v", err)
		}
		// Add dependency on completed task and a non-existent task
		store.TaskDepends(ctx, noReadySession, TaskDependsParams{
			ID:        task2.ID,
			DependsOn: task1.ID, // This is completed, OK
			Iteration: 1,
		})

		// Now task2 depends on task1 which is completed, so task2 should be ready
		nextTask, err := store.TaskNext(ctx, noReadySession)
		if err != nil {
			t.Fatalf("TaskNext failed: %v", err)
		}
		if nextTask == nil {
			t.Error("expected task2 to be ready since task1 is completed")
		}

		// Now mark task2 as in_progress - should have no ready tasks
		store.TaskStatus(ctx, noReadySession, TaskStatusParams{
			ID:        task2.ID,
			Status:    "in_progress",
			Iteration: 1,
		})

		nextTask, err = store.TaskNext(ctx, noReadySession)
		if err != nil {
			t.Fatalf("TaskNext failed: %v", err)
		}
		if nextTask != nil {
			t.Errorf("expected nil when no ready tasks, got %s", nextTask.ID)
		}
	})
}
