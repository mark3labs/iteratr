package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/iteratr/internal/nats"
)

// IterationStart logs the start of a new iteration.
// Creates an event of type "iteration" with action "start".
func (s *Store) IterationStart(ctx context.Context, session string, number int) error {
	// Build metadata
	meta, err := json.Marshal(map[string]any{
		"number": number,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal iteration start metadata: %w", err)
	}

	// Create event
	event := Event{
		Session: session,
		Type:    nats.EventTypeIteration,
		Action:  "start",
		Meta:    meta,
		Data:    fmt.Sprintf("Iteration %d started", number),
	}

	// Publish event
	_, err = s.PublishEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to publish iteration start event: %w", err)
	}

	return nil
}

// IterationComplete logs the completion of an iteration.
// Creates an event of type "iteration" with action "complete".
func (s *Store) IterationComplete(ctx context.Context, session string, number int) error {
	// Build metadata
	meta, err := json.Marshal(map[string]any{
		"number": number,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal iteration complete metadata: %w", err)
	}

	// Create event
	event := Event{
		Session: session,
		Type:    nats.EventTypeIteration,
		Action:  "complete",
		Meta:    meta,
		Data:    fmt.Sprintf("Iteration %d completed", number),
	}

	// Publish event
	_, err = s.PublishEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to publish iteration complete event: %w", err)
	}

	return nil
}
