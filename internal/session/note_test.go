package session

import (
	"context"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
)

func TestNoteOperations(t *testing.T) {
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

	t.Run("NoteAdd creates note with valid type", func(t *testing.T) {
		note, err := store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Learned about event sourcing",
			Type:      "learning",
			Iteration: 1,
		})
		if err != nil {
			t.Fatalf("NoteAdd failed: %v", err)
		}

		if note.ID == "" {
			t.Error("expected note ID to be set")
		}
		if note.Content != "Learned about event sourcing" {
			t.Errorf("expected content 'Learned about event sourcing', got '%s'", note.Content)
		}
		if note.Type != "learning" {
			t.Errorf("expected type 'learning', got '%s'", note.Type)
		}
		if note.Iteration != 1 {
			t.Errorf("expected iteration 1, got %d", note.Iteration)
		}
	})

	t.Run("NoteAdd accepts all valid types", func(t *testing.T) {
		validTypes := []string{"learning", "stuck", "tip", "decision"}
		for _, noteType := range validTypes {
			_, err := store.NoteAdd(ctx, session, NoteAddParams{
				Content:   "Test note for " + noteType,
				Type:      noteType,
				Iteration: 1,
			})
			if err != nil {
				t.Errorf("NoteAdd failed for valid type '%s': %v", noteType, err)
			}
		}
	})

	t.Run("NoteAdd validates type", func(t *testing.T) {
		_, err := store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Invalid note",
			Type:      "invalid_type",
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for invalid type")
		}
	})

	t.Run("NoteAdd requires content", func(t *testing.T) {
		_, err := store.NoteAdd(ctx, session, NoteAddParams{
			Type:      "learning",
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for missing content")
		}
	})

	t.Run("NoteAdd requires type", func(t *testing.T) {
		_, err := store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Some content",
			Iteration: 1,
		})
		if err == nil {
			t.Error("expected error for missing type")
		}
	})

	t.Run("NoteList returns all notes", func(t *testing.T) {
		// Add a few notes
		_, _ = store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Note 1",
			Type:      "learning",
			Iteration: 1,
		})
		_, _ = store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Note 2",
			Type:      "tip",
			Iteration: 1,
		})

		notes, err := store.NoteList(ctx, session, NoteListParams{})
		if err != nil {
			t.Fatalf("NoteList failed: %v", err)
		}

		if len(notes) == 0 {
			t.Error("expected at least some notes")
		}
	})

	t.Run("NoteList filters by type", func(t *testing.T) {
		// Add notes of different types
		_, _ = store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Learning note",
			Type:      "learning",
			Iteration: 2,
		})
		_, _ = store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Decision note",
			Type:      "decision",
			Iteration: 2,
		})

		// Filter for learning notes
		notes, err := store.NoteList(ctx, session, NoteListParams{
			Type: "learning",
		})
		if err != nil {
			t.Fatalf("NoteList failed: %v", err)
		}

		// All returned notes should be learning type
		for _, note := range notes {
			if note.Type != "learning" {
				t.Errorf("expected type 'learning', got '%s'", note.Type)
			}
		}
	})

	t.Run("NoteList validates filter type", func(t *testing.T) {
		_, err := store.NoteList(ctx, session, NoteListParams{
			Type: "invalid_type",
		})
		if err == nil {
			t.Error("expected error for invalid filter type")
		}
	})

	t.Run("Notes are persisted via event sourcing", func(t *testing.T) {
		// Add a note
		note1, err := store.NoteAdd(ctx, session, NoteAddParams{
			Content:   "Persistence test",
			Type:      "tip",
			Iteration: 3,
		})
		if err != nil {
			t.Fatalf("NoteAdd failed: %v", err)
		}

		// Load state and verify note is present
		state, err := store.LoadState(ctx, session)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		found := false
		for _, note := range state.Notes {
			if note.ID == note1.ID {
				found = true
				if note.Content != "Persistence test" {
					t.Errorf("expected content 'Persistence test', got '%s'", note.Content)
				}
				if note.Type != "tip" {
					t.Errorf("expected type 'tip', got '%s'", note.Type)
				}
				if note.Iteration != 3 {
					t.Errorf("expected iteration 3, got %d", note.Iteration)
				}
				break
			}
		}

		if !found {
			t.Error("note not found in state after reload")
		}
	})
}
