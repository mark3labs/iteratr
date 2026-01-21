package session

import (
	"context"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
)

func TestIterationOperations(t *testing.T) {
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

	t.Run("IterationStart creates iteration event", func(t *testing.T) {
		err := store.IterationStart(ctx, session, 1)
		if err != nil {
			t.Fatalf("IterationStart failed: %v", err)
		}

		// Load state and verify iteration was added
		state, err := store.LoadState(ctx, session)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		if len(state.Iterations) == 0 {
			t.Fatal("expected at least one iteration")
		}

		iter := state.Iterations[len(state.Iterations)-1]
		if iter.Number != 1 {
			t.Errorf("expected iteration number 1, got %d", iter.Number)
		}
		if iter.Complete {
			t.Error("expected iteration to not be complete yet")
		}
		if iter.StartedAt.IsZero() {
			t.Error("expected StartedAt to be set")
		}
	})

	t.Run("IterationComplete marks iteration as complete", func(t *testing.T) {
		// Start a new iteration
		err := store.IterationStart(ctx, session, 2)
		if err != nil {
			t.Fatalf("IterationStart failed: %v", err)
		}

		// Complete it
		err = store.IterationComplete(ctx, session, 2)
		if err != nil {
			t.Fatalf("IterationComplete failed: %v", err)
		}

		// Load state and verify
		state, err := store.LoadState(ctx, session)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		// Find iteration 2
		var iter2 *Iteration
		for _, iter := range state.Iterations {
			if iter.Number == 2 {
				iter2 = iter
				break
			}
		}

		if iter2 == nil {
			t.Fatal("expected to find iteration 2")
		}
		if !iter2.Complete {
			t.Error("expected iteration 2 to be complete")
		}
		if iter2.EndedAt.IsZero() {
			t.Error("expected EndedAt to be set")
		}
	})

	t.Run("Multiple iterations are tracked separately", func(t *testing.T) {
		// Start and complete iteration 3
		err := store.IterationStart(ctx, session, 3)
		if err != nil {
			t.Fatalf("IterationStart failed: %v", err)
		}

		err = store.IterationComplete(ctx, session, 3)
		if err != nil {
			t.Fatalf("IterationComplete failed: %v", err)
		}

		// Start iteration 4 but don't complete it
		err = store.IterationStart(ctx, session, 4)
		if err != nil {
			t.Fatalf("IterationStart failed: %v", err)
		}

		// Load state and verify
		state, err := store.LoadState(ctx, session)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		if len(state.Iterations) < 4 {
			t.Fatalf("expected at least 4 iterations, got %d", len(state.Iterations))
		}

		// Verify iteration 3 is complete
		var iter3 *Iteration
		for _, iter := range state.Iterations {
			if iter.Number == 3 {
				iter3 = iter
				break
			}
		}
		if iter3 == nil || !iter3.Complete {
			t.Error("expected iteration 3 to be complete")
		}

		// Verify iteration 4 is not complete
		var iter4 *Iteration
		for _, iter := range state.Iterations {
			if iter.Number == 4 {
				iter4 = iter
				break
			}
		}
		if iter4 == nil {
			t.Fatal("expected to find iteration 4")
		}
		if iter4.Complete {
			t.Error("expected iteration 4 to not be complete")
		}
	})

	t.Run("Iterations persist via event sourcing", func(t *testing.T) {
		// Use a dedicated session
		iterSession := "test-iteration-persistence"

		// Add iterations
		store.IterationStart(ctx, iterSession, 1)
		store.IterationComplete(ctx, iterSession, 1)
		store.IterationStart(ctx, iterSession, 2)

		// Load state multiple times - should get same result from event log
		state1, err := store.LoadState(ctx, iterSession)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		state2, err := store.LoadState(ctx, iterSession)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}

		if len(state1.Iterations) != len(state2.Iterations) {
			t.Errorf("expected same number of iterations, got %d and %d",
				len(state1.Iterations), len(state2.Iterations))
		}
	})
}
