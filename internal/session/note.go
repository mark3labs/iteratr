package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/iteratr/internal/nats"
)

// NoteAddParams represents the parameters for adding a note.
type NoteAddParams struct {
	Content   string `json:"content"`
	Type      string `json:"type"`      // learning, stuck, tip, decision
	Iteration int    `json:"iteration"` // Iteration number that created this note
}

// NoteListParams represents the parameters for listing notes.
type NoteListParams struct {
	Type string `json:"type,omitempty"` // Optional: filter by type
}

// NoteAdd creates a new note in the session.
// Type must be one of: learning, stuck, tip, decision.
func (s *Store) NoteAdd(ctx context.Context, session string, params NoteAddParams) (*Note, error) {
	// Validate required fields
	if params.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if params.Type == "" {
		return nil, fmt.Errorf("type is required")
	}

	// Validate type
	if !isValidNoteType(params.Type) {
		return nil, fmt.Errorf("invalid type: %s (must be learning, stuck, tip, or decision)", params.Type)
	}

	// Load current state to get next note counter
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for ID generation: %w", err)
	}

	// Generate sequential ID and timestamp
	id := fmt.Sprintf("NOT-%d", state.NoteCounter+1)
	now := time.Now()

	// Create event metadata
	meta, _ := json.Marshal(map[string]any{
		"type":      params.Type,
		"iteration": params.Iteration,
	})

	// Create and publish event
	event := Event{
		ID:        id,
		Timestamp: now,
		Session:   session,
		Type:      nats.EventTypeNote,
		Action:    "add",
		Data:      params.Content,
		Meta:      meta,
	}

	_, err = s.PublishEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	// Build note object to return
	note := &Note{
		ID:        id,
		Content:   params.Content,
		Type:      params.Type,
		CreatedAt: now,
		Iteration: params.Iteration,
	}

	return note, nil
}

// NoteList returns all notes, optionally filtered by type.
func (s *Store) NoteList(ctx context.Context, session string, params NoteListParams) ([]*Note, error) {
	// Load current state
	state, err := s.LoadState(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// If no filter, return all notes
	if params.Type == "" {
		return state.Notes, nil
	}

	// Validate filter type
	if !isValidNoteType(params.Type) {
		return nil, fmt.Errorf("invalid type filter: %s (must be learning, stuck, tip, or decision)", params.Type)
	}

	// Filter notes by type
	var filtered []*Note
	for _, note := range state.Notes {
		if note.Type == params.Type {
			filtered = append(filtered, note)
		}
	}

	return filtered, nil
}

// isValidNoteType checks if a note type string is valid.
func isValidNoteType(noteType string) bool {
	switch noteType {
	case "learning", "stuck", "tip", "decision":
		return true
	default:
		return false
	}
}
