package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mark3labs/iteratr/internal/logger"
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

// ResetSession removes all events for a session, resetting it to a fresh state.
func (s *Store) ResetSession(ctx context.Context, session string) error {
	return nats.PurgeSession(ctx, s.stream, session)
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
		logger.Error("Failed to marshal event: %v", err)
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	// Build subject: iteratr.{session}.{type}
	subject := nats.SubjectForEvent(event.Session, event.Type)

	logger.Debug("Publishing event: session=%s type=%s action=%s", event.Session, event.Type, event.Action)

	// Publish to JetStream
	ack, err := s.js.Publish(ctx, subject, data)
	if err != nil {
		logger.Error("Failed to publish event to subject %s: %v", subject, err)
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Debug("Event published successfully: seq=%d", ack.Sequence)
	return ack, nil
}

// State represents the current state of a session, reconstructed from events.
// It implements the reduce pattern by applying events to build up the current state.
type State struct {
	Session     string           `json:"session"`
	Tasks       map[string]*Task `json:"tasks"`        // Task ID -> Task
	TaskCounter int              `json:"task_counter"` // Incrementing counter for TAS-N IDs
	Notes       []*Note          `json:"notes"`        // Chronological list of notes
	NoteCounter int              `json:"note_counter"` // Incrementing counter for NOT-N IDs
	Iterations  []*Iteration     `json:"iterations"`   // Iteration history
	Complete    bool             `json:"complete"`     // Session marked complete
}

// Task represents a task in the task system.
type Task struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`     // remaining, in_progress, completed, blocked
	Priority  int       `json:"priority"`   // 0-4, default 2 (0=critical, 1=high, 2=medium, 3=low, 4=backlog)
	DependsOn []string  `json:"depends_on"` // Task IDs this task is blocked by
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Iteration int       `json:"iteration"` // Iteration that last modified this task
}

// Note represents a note recorded during a session.
type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // learning, stuck, tip, decision
	CreatedAt time.Time `json:"created_at"`
	Iteration int       `json:"iteration"` // Iteration that created this note
}

// Iteration represents a single iteration execution.
type Iteration struct {
	Number      int       `json:"number"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at,omitempty"`
	Complete    bool      `json:"complete"`
	Summary     string    `json:"summary,omitempty"`      // What was accomplished
	TasksWorked []string  `json:"tasks_worked,omitempty"` // Task IDs touched
}

// Apply applies an event to the state, implementing the reduce pattern.
// This method mutates the state based on the event type and action.
func (st *State) Apply(event Event) {
	switch event.Type {
	case nats.EventTypeTask:
		st.applyTaskEvent(event)
	case nats.EventTypeNote:
		st.applyNoteEvent(event)
	case nats.EventTypeIteration:
		st.applyIterationEvent(event)
	case nats.EventTypeControl:
		st.applyControlEvent(event)
	}
}

// applyTaskEvent handles task-related events.
func (st *State) applyTaskEvent(event Event) {
	switch event.Action {
	case "add":
		// Parse metadata for status, priority, and iteration
		var meta struct {
			Status    string `json:"status"`
			Priority  int    `json:"priority"`
			Iteration int    `json:"iteration"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Default status to "remaining"
		if meta.Status == "" {
			meta.Status = "remaining"
		}

		// Default priority to 2 (medium) if not set
		// Note: If priority is explicitly 0 (critical), it will be preserved
		// We check if meta has priority set by seeing if it's non-zero or checking meta itself
		priority := meta.Priority
		if priority == 0 && event.Meta != nil {
			// Check if priority was explicitly set to 0 in metadata
			var checkMeta map[string]interface{}
			_ = json.Unmarshal(event.Meta, &checkMeta)
			if _, hasPriority := checkMeta["priority"]; !hasPriority {
				priority = 2 // Default to medium
			}
		} else if priority == 0 && event.Meta == nil {
			priority = 2 // Default to medium
		}

		// Create new task
		task := &Task{
			ID:        event.ID,
			Content:   event.Data,
			Status:    meta.Status,
			Priority:  priority,
			DependsOn: []string{}, // Initialize empty dependencies
			CreatedAt: event.Timestamp,
			UpdatedAt: event.Timestamp,
			Iteration: meta.Iteration,
		}
		st.Tasks[event.ID] = task
		st.TaskCounter++

	case "status":
		// Parse metadata for task ID and new status
		var meta struct {
			TaskID    string `json:"task_id"`
			Status    string `json:"status"`
			Iteration int    `json:"iteration"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Update task status if it exists
		if task, exists := st.Tasks[meta.TaskID]; exists {
			task.Status = meta.Status
			task.UpdatedAt = event.Timestamp
			task.Iteration = meta.Iteration
		}

	case "priority":
		// Parse metadata for task ID and new priority
		var meta struct {
			TaskID    string `json:"task_id"`
			Priority  int    `json:"priority"`
			Iteration int    `json:"iteration"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Update task priority if it exists
		if task, exists := st.Tasks[meta.TaskID]; exists {
			task.Priority = meta.Priority
			task.UpdatedAt = event.Timestamp
			task.Iteration = meta.Iteration
		}

	case "depends":
		// Parse metadata for task ID and dependency
		var meta struct {
			TaskID    string `json:"task_id"`
			DependsOn string `json:"depends_on"`
			Iteration int    `json:"iteration"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Add dependency if task exists and dependency not already present
		if task, exists := st.Tasks[meta.TaskID]; exists {
			// Check if dependency already exists
			found := false
			for _, dep := range task.DependsOn {
				if dep == meta.DependsOn {
					found = true
					break
				}
			}
			// Add if not found
			if !found {
				task.DependsOn = append(task.DependsOn, meta.DependsOn)
			}
			task.UpdatedAt = event.Timestamp
			task.Iteration = meta.Iteration
		}
	}
}

// applyNoteEvent handles note-related events.
func (st *State) applyNoteEvent(event Event) {
	switch event.Action {
	case "add":
		// Parse metadata for note type and iteration
		var meta struct {
			Type      string `json:"type"`
			Iteration int    `json:"iteration"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Create new note
		note := &Note{
			ID:        event.ID,
			Content:   event.Data,
			Type:      meta.Type,
			CreatedAt: event.Timestamp,
			Iteration: meta.Iteration,
		}
		st.Notes = append(st.Notes, note)
		st.NoteCounter++
	}
}

// applyIterationEvent handles iteration-related events.
func (st *State) applyIterationEvent(event Event) {
	switch event.Action {
	case "start":
		// Parse metadata for iteration number
		var meta struct {
			Number int `json:"number"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Create new iteration
		iter := &Iteration{
			Number:    meta.Number,
			StartedAt: event.Timestamp,
			Complete:  false,
		}
		st.Iterations = append(st.Iterations, iter)

	case "complete":
		// Parse metadata for iteration number
		var meta struct {
			Number int `json:"number"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Mark iteration as complete
		for _, iter := range st.Iterations {
			if iter.Number == meta.Number {
				iter.Complete = true
				iter.EndedAt = event.Timestamp
				break
			}
		}

	case "summary":
		// Parse metadata for iteration number, summary, and tasks worked
		var meta struct {
			Number      int      `json:"number"`
			Summary     string   `json:"summary"`
			TasksWorked []string `json:"tasks_worked"`
		}
		_ = json.Unmarshal(event.Meta, &meta)

		// Update iteration with summary and tasks worked
		for _, iter := range st.Iterations {
			if iter.Number == meta.Number {
				iter.Summary = meta.Summary
				iter.TasksWorked = meta.TasksWorked
				break
			}
		}
	}
}

// applyControlEvent handles control-related events.
func (st *State) applyControlEvent(event Event) {
	switch event.Action {
	case "session_complete":
		st.Complete = true
	case "session_restart":
		st.Complete = false
	}
}

// LoadState reconstructs the current state of a session by reading and reducing
// all events from the JetStream event log. This implements the event sourcing pattern.
func (s *Store) LoadState(ctx context.Context, session string) (*State, error) {
	logger.Debug("Loading state for session: %s", session)

	// Create a consumer filtered to this session's events
	consumer, err := s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		FilterSubject: nats.SubjectForSession(session),
		DeliverPolicy: jetstream.DeliverAllPolicy, // Start from beginning
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		logger.Error("Failed to create consumer for session %s: %v", session, err)
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Initialize empty state
	state := &State{
		Session: session,
		Tasks:   make(map[string]*Task),
	}

	// Fetch events in batches and reduce into state
	// Using a large batch size to minimize round trips
	const batchSize = 1000
	malformedCount := 0
	totalEvents := 0
	for {
		// Fetch with short timeout to avoid blocking forever
		msgs, err := consumer.FetchNoWait(batchSize)
		if err != nil {
			// No more messages or error - we've read everything
			logger.Debug("Finished reading events (batch fetch complete)")
			break
		}

		msgCount := 0
		for msg := range msgs.Messages() {
			msgCount++
			totalEvents++
			// Unmarshal event
			var event Event
			if err := json.Unmarshal(msg.Data(), &event); err != nil {
				// Log malformed event and skip (but acknowledge to prevent redelivery)
				malformedCount++
				meta, _ := msg.Metadata()
				logger.Warn("Skipping malformed event (seq=%d): %v", meta.Sequence.Stream, err)
				fmt.Fprintf(os.Stderr, "Warning: Skipping malformed event (seq=%d): %v\n", meta.Sequence.Stream, err)
				_ = msg.Ack()
				continue
			}

			// Store the message sequence as ID if not set
			if event.ID == "" {
				meta, _ := msg.Metadata()
				event.ID = fmt.Sprintf("%d", meta.Sequence.Stream)
			}

			// Apply event to state (reduce)
			state.Apply(event)

			// Acknowledge message
			_ = msg.Ack()
		}

		logger.Debug("Processed batch: %d events", msgCount)

		// If we got fewer messages than batch size, we've reached the end
		if msgCount < batchSize {
			break
		}
	}

	// Warn if we encountered malformed events
	if malformedCount > 0 {
		logger.Warn("Skipped %d malformed events while loading state", malformedCount)
		fmt.Fprintf(os.Stderr, "Warning: Skipped %d malformed events while loading state\n", malformedCount)
	}

	logger.Debug("State loaded: %d total events, %d tasks, %d notes, %d iterations",
		totalEvents, len(state.Tasks), len(state.Notes), len(state.Iterations))

	return state, nil
}
