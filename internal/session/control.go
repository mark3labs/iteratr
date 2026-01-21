package session

import (
	"context"
	"fmt"

	"github.com/mark3labs/iteratr/internal/nats"
)

// SessionComplete marks a session as complete.
// Creates an event of type "control" with action "session_complete".
// This signals that all tasks are done and the iteration loop should terminate.
func (s *Store) SessionComplete(ctx context.Context, session string) error {
	// Create event
	event := Event{
		Session: session,
		Type:    nats.EventTypeControl,
		Action:  "session_complete",
		Data:    "Session marked as complete",
	}

	// Publish event
	_, err := s.PublishEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to publish session complete event: %w", err)
	}

	return nil
}
