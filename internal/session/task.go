package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	ID        string `json:"id"`     // Task ID or prefix (8+ chars)
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

	// Create event metadata
	meta, _ := json.Marshal(map[string]any{
		"status":    status,
		"iteration": params.Iteration,
	})

	// Create and publish event
	event := Event{
		Session: session,
		Type:    nats.EventTypeTask,
		Action:  "add",
		Data:    params.Content,
		Meta:    meta,
	}

	ack, err := s.PublishEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	// Build task object to return, using NATS sequence as ID
	task := &Task{
		ID:        fmt.Sprintf("%d", ack.Sequence),
		Content:   params.Content,
		Status:    status,
		CreatedAt: event.Timestamp,
		UpdatedAt: event.Timestamp,
		Iteration: params.Iteration,
	}

	return task, nil
}

// TaskStatus updates the status of an existing task.
// The ID parameter supports prefix matching (minimum 8 characters).
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

// resolveTaskID resolves a task ID or prefix to a full task ID.
// Supports prefix matching with minimum 8 characters.
// Returns an error if the prefix is ambiguous or not found.
func resolveTaskID(state *State, idOrPrefix string) (string, error) {
	// If exact match exists, return it
	if _, exists := state.Tasks[idOrPrefix]; exists {
		return idOrPrefix, nil
	}

	// Check prefix length requirement
	if len(idOrPrefix) < 8 {
		return "", fmt.Errorf("task ID prefix must be at least 8 characters (got %d)", len(idOrPrefix))
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
		return "", fmt.Errorf("ambiguous task ID prefix: %s (matches %d tasks)", idOrPrefix, len(matches))
	}
}
