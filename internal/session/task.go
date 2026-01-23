package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/iteratr/internal/nats"
)

// TaskAddParams represents the parameters for adding a task.
type TaskAddParams struct {
	Content   string `json:"content"`
	Status    string `json:"status,omitempty"` // Optional: remaining, in_progress, completed, blocked
	Iteration int    `json:"iteration"`
}

// TaskStatusParams represents the parameters for updating task status.
type TaskStatusParams struct {
	ID        string `json:"id"`     // Task ID or prefix (3+ chars)
	Status    string `json:"status"` // remaining, in_progress, completed, blocked
	Iteration int    `json:"iteration"`
}

// TaskListResult represents the result of listing tasks.
type TaskListResult struct {
	Remaining  []*Task `json:"remaining"`
	InProgress []*Task `json:"in_progress"`
	Completed  []*Task `json:"completed"`
	Blocked    []*Task `json:"blocked"`
}

// TaskAdd creates a new task in the session.
// Status defaults to "remaining" if not specified.
func (s *Store) TaskAdd(ctx context.Context, session string, params TaskAddParams) (*Task, error) {
	// Validate required fields
	if params.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	// Default status to "remaining"
	status := params.Status
	if status == "" {
		status = "remaining"
	}

	// Validate status
	if !isValidTaskStatus(status) {
		return nil, fmt.Errorf("invalid status: %s (must be remaining, in_progress, completed, or blocked)", status)
	}

	// Load current state to get next task counter
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for ID generation: %w", err)
	}

	// Generate sequential ID and timestamp
	id := fmt.Sprintf("TAS-%d", state.TaskCounter+1)
	now := time.Now()

	// Create event metadata
	meta, _ := json.Marshal(map[string]any{
		"status":    status,
		"iteration": params.Iteration,
	})

	// Create and publish event
	event := Event{
		ID:        id,
		Timestamp: now,
		Session:   session,
		Type:      nats.EventTypeTask,
		Action:    "add",
		Data:      params.Content,
		Meta:      meta,
	}

	_, err = s.PublishEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	// Build task object to return
	task := &Task{
		ID:        id,
		Content:   params.Content,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
		Iteration: params.Iteration,
	}

	return task, nil
}

// TaskBatchAdd creates multiple tasks in a single operation.
// Loads state once and generates sequential IDs efficiently.
func (s *Store) TaskBatchAdd(ctx context.Context, session string, tasks []TaskAddParams) ([]*Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("at least one task is required")
	}

	// Load state once to get the current counter
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for ID generation: %w", err)
	}

	counter := state.TaskCounter
	now := time.Now()
	result := make([]*Task, 0, len(tasks))

	for _, params := range tasks {
		if params.Content == "" {
			return nil, fmt.Errorf("content is required for all tasks")
		}

		status := params.Status
		if status == "" {
			status = "remaining"
		}
		if !isValidTaskStatus(status) {
			return nil, fmt.Errorf("invalid status: %s (must be remaining, in_progress, completed, or blocked)", status)
		}

		counter++
		id := fmt.Sprintf("TAS-%d", counter)

		meta, _ := json.Marshal(map[string]any{
			"status":    status,
			"iteration": params.Iteration,
		})

		event := Event{
			ID:        id,
			Timestamp: now,
			Session:   session,
			Type:      nats.EventTypeTask,
			Action:    "add",
			Data:      params.Content,
			Meta:      meta,
		}

		_, err := s.PublishEvent(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("failed to publish task %q: %w", params.Content, err)
		}

		result = append(result, &Task{
			ID:        id,
			Content:   params.Content,
			Status:    status,
			CreatedAt: now,
			UpdatedAt: now,
			Iteration: params.Iteration,
		})
	}

	return result, nil
}

// TaskStatus updates the status of an existing task.
// The ID parameter supports prefix matching (minimum 3 characters).
func (s *Store) TaskStatus(ctx context.Context, session string, params TaskStatusParams) error {
	// Validate required fields
	if params.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if params.Status == "" {
		return fmt.Errorf("status is required")
	}

	// Validate status
	if !isValidTaskStatus(params.Status) {
		return fmt.Errorf("invalid status: %s (must be remaining, in_progress, completed, or blocked)", params.Status)
	}

	// Load current state to resolve task ID prefix
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Resolve task ID (supports prefix matching)
	taskID, err := resolveTaskID(state, params.ID)
	if err != nil {
		return err
	}

	// Create event metadata
	meta, _ := json.Marshal(map[string]any{
		"task_id":   taskID,
		"status":    params.Status,
		"iteration": params.Iteration,
	})

	// Create and publish event
	event := Event{
		Session: session,
		Type:    nats.EventTypeTask,
		Action:  "status",
		Data:    params.Status, // Store new status in data field for convenience
		Meta:    meta,
	}

	_, err = s.PublishEvent(ctx, event)
	return err
}

// TaskPriorityParams represents the parameters for updating task priority.
type TaskPriorityParams struct {
	ID        string `json:"id"`       // Task ID or prefix (8+ chars)
	Priority  int    `json:"priority"` // 0-4 (0=critical, 1=high, 2=medium, 3=low, 4=backlog)
	Iteration int    `json:"iteration"`
}

// TaskPriority updates the priority of an existing task.
// The ID parameter supports prefix matching (minimum 8 characters).
// Priority must be 0-4 (0=critical, 1=high, 2=medium, 3=low, 4=backlog).
func (s *Store) TaskPriority(ctx context.Context, session string, params TaskPriorityParams) error {
	// Validate required fields
	if params.ID == "" {
		return fmt.Errorf("task ID is required")
	}

	// Validate priority range
	if params.Priority < 0 || params.Priority > 4 {
		return fmt.Errorf("invalid priority: %d (must be 0-4)", params.Priority)
	}

	// Load current state to resolve task ID prefix
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Resolve task ID (supports prefix matching)
	taskID, err := resolveTaskID(state, params.ID)
	if err != nil {
		return err
	}

	// Create event metadata
	meta, _ := json.Marshal(map[string]any{
		"task_id":   taskID,
		"priority":  params.Priority,
		"iteration": params.Iteration,
	})

	// Create and publish event
	event := Event{
		Session: session,
		Type:    nats.EventTypeTask,
		Action:  "priority",
		Data:    fmt.Sprintf("%d", params.Priority), // Store priority in data field for convenience
		Meta:    meta,
	}

	_, err = s.PublishEvent(ctx, event)
	return err
}

// TaskDependsParams represents the parameters for adding a task dependency.
type TaskDependsParams struct {
	ID        string `json:"id"`         // Task ID or prefix (8+ chars)
	DependsOn string `json:"depends_on"` // Task ID or prefix that this task depends on
	Iteration int    `json:"iteration"`
}

// TaskDepends adds a dependency to an existing task.
// The ID and DependsOn parameters support prefix matching (minimum 8 characters).
func (s *Store) TaskDepends(ctx context.Context, session string, params TaskDependsParams) error {
	// Validate required fields
	if params.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if params.DependsOn == "" {
		return fmt.Errorf("depends_on is required")
	}

	// Load current state to resolve task ID prefixes
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Resolve task ID (supports prefix matching)
	taskID, err := resolveTaskID(state, params.ID)
	if err != nil {
		return err
	}

	// Resolve dependency task ID (supports prefix matching)
	dependsOnID, err := resolveTaskID(state, params.DependsOn)
	if err != nil {
		return fmt.Errorf("failed to resolve depends_on task: %w", err)
	}

	// Validate that task isn't depending on itself
	if taskID == dependsOnID {
		return fmt.Errorf("task cannot depend on itself")
	}

	// Create event metadata
	meta, _ := json.Marshal(map[string]any{
		"task_id":    taskID,
		"depends_on": dependsOnID,
		"iteration":  params.Iteration,
	})

	// Create and publish event
	event := Event{
		Session: session,
		Type:    nats.EventTypeTask,
		Action:  "depends",
		Data:    dependsOnID, // Store dependency ID in data field for convenience
		Meta:    meta,
	}

	_, err = s.PublishEvent(ctx, event)
	return err
}

// TaskList returns all tasks grouped by status.
func (s *Store) TaskList(ctx context.Context, session string) (*TaskListResult, error) {
	// Load current state
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Group tasks by status
	result := &TaskListResult{
		Remaining:  make([]*Task, 0),
		InProgress: make([]*Task, 0),
		Completed:  make([]*Task, 0),
		Blocked:    make([]*Task, 0),
	}

	for _, task := range state.Tasks {
		switch task.Status {
		case "remaining":
			result.Remaining = append(result.Remaining, task)
		case "in_progress":
			result.InProgress = append(result.InProgress, task)
		case "completed":
			result.Completed = append(result.Completed, task)
		case "blocked":
			result.Blocked = append(result.Blocked, task)
		}
	}

	return result, nil
}

// isValidTaskStatus checks if a status string is valid.
func isValidTaskStatus(status string) bool {
	switch status {
	case "remaining", "in_progress", "completed", "blocked":
		return true
	default:
		return false
	}
}

// TaskNext returns the highest priority unblocked task.
// A task is "ready" if it has status "remaining" and all its dependencies are completed.
// Returns nil if no ready tasks exist.
func (s *Store) TaskNext(ctx context.Context, session string) (*Task, error) {
	// Load current state
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	var bestTask *Task
	for _, task := range state.Tasks {
		// Skip non-remaining tasks
		if task.Status != "remaining" {
			continue
		}

		// Check if all dependencies are completed
		allDepsCompleted := true
		for _, depID := range task.DependsOn {
			if depTask, exists := state.Tasks[depID]; exists {
				if depTask.Status != "completed" {
					allDepsCompleted = false
					break
				}
			} else {
				// Dependency doesn't exist - treat as unresolved
				allDepsCompleted = false
				break
			}
		}

		if !allDepsCompleted {
			continue
		}

		// This task is ready - compare priority (lower is higher priority)
		if bestTask == nil || task.Priority < bestTask.Priority {
			bestTask = task
		}
	}

	return bestTask, nil
}

// resolveTaskID resolves a task ID or prefix to a full task ID.
// Supports prefix matching with minimum 3 characters.
// Returns an error if the prefix is ambiguous or not found.
func resolveTaskID(state *State, idOrPrefix string) (string, error) {
	// If exact match exists, return it
	if _, exists := state.Tasks[idOrPrefix]; exists {
		return idOrPrefix, nil
	}

	// Check prefix length requirement
	if len(idOrPrefix) < 3 {
		return "", fmt.Errorf("task ID prefix must be at least 3 characters (got %d)", len(idOrPrefix))
	}

	// Find matching tasks by prefix
	var matches []string
	for taskID := range state.Tasks {
		if strings.HasPrefix(taskID, idOrPrefix) {
			matches = append(matches, taskID)
		}
	}

	// Handle results
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("task not found: %s", idOrPrefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous task ID prefix: %s (matches: %s)", idOrPrefix, strings.Join(matches, ", "))
	}
}
