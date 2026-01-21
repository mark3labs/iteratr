package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/nats-io/nats.go/jetstream"
)

// Event represents a generic event stored in the JetStream event log.
// All session operations (tasks, notes, inbox, iterations) are stored as events
// following an append-only event sourcing pattern.
type Event struct {
	ID        string          `json:"id"`        // NATS message sequence ID
	Timestamp time.Time       `json:"timestamp"` // When the event occurred
	Session   string          `json:"session"`   // Session name
	Type      string          `json:"type"`      // Event type: task, note, inbox, iteration, control
	Action    string          `json:"action"`    // Action type: add, status, mark_read, start, complete, etc.
	Meta      json.RawMessage `json:"meta"`      // Action-specific metadata
	Data      string          `json:"data"`      // Primary content (task text, note text, etc.)
}

// Store manages session state through JetStream event sourcing.
// It provides methods for publishing events and loading state from the event stream.
type Store struct {
	js     jetstream.JetStream // JetStream context for operations
	stream jetstream.Stream    // The iteratr_events stream
}

// NewStore creates a new Store instance with the given JetStream context and stream.
func NewStore(js jetstream.JetStream, stream jetstream.Stream) *Store {
	return &Store{
		js:     js,
		stream: stream,
	}
}

// PublishEvent appends an event to the JetStream event log.
// Events are published to subjects following the pattern: iteratr.{session}.{type}
// Returns the published ACK or an error if publishing fails.
func (s *Store) PublishEvent(ctx context.Context, event Event) (*jetstream.PubAck, error) {
	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	// Build subject: iteratr.{session}.{type}
	subject := nats.SubjectForEvent(event.Session, event.Type)

	// Publish to JetStream
	ack, err := s.js.Publish(ctx, subject, data)
	if err != nil {
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	return ack, nil
}
