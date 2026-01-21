package session

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/nats"
	natsserver "github.com/nats-io/nats-server/v2/server"
	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestSessionComplete(t *testing.T) {
	// Start embedded NATS server
	srv, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start NATS: %v", err)
	}
	defer srv.Shutdown()

	// Connect to NATS in-process
	nc, err := natsclient.Connect("", natsclient.InProcessServer(srv))
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream: %v", err)
	}

	// Setup stream
	ctx := context.Background()
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("Failed to setup stream: %v", err)
	}

	// Create store
	store := NewStore(js, stream)
	sessionName := "test-session"

	// Mark session as complete
	err = store.SessionComplete(ctx, sessionName)
	if err != nil {
		t.Fatalf("Failed to mark session complete: %v", err)
	}

	// Load state to verify
	state, err := store.LoadState(ctx, sessionName)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify session is marked as complete
	if !state.Complete {
		t.Errorf("Expected session to be marked complete, but Complete=false")
	}
}

func TestSessionCompleteMultipleTimes(t *testing.T) {
	// Start embedded NATS server
	srv, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start NATS: %v", err)
	}
	defer srv.Shutdown()

	// Connect to NATS in-process
	nc, err := natsclient.Connect("", natsclient.InProcessServer(srv))
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream: %v", err)
	}

	// Setup stream
	ctx := context.Background()
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("Failed to setup stream: %v", err)
	}

	// Create store
	store := NewStore(js, stream)
	sessionName := "test-session-multi"

	// Mark session as complete multiple times (should be idempotent)
	err = store.SessionComplete(ctx, sessionName)
	if err != nil {
		t.Fatalf("Failed to mark session complete (first): %v", err)
	}

	err = store.SessionComplete(ctx, sessionName)
	if err != nil {
		t.Fatalf("Failed to mark session complete (second): %v", err)
	}

	// Load state to verify
	state, err := store.LoadState(ctx, sessionName)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify session is marked as complete
	if !state.Complete {
		t.Errorf("Expected session to be marked complete, but Complete=false")
	}
}

func TestSessionCompleteWithTasks(t *testing.T) {
	// Start embedded NATS server
	srv, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start NATS: %v", err)
	}
	defer srv.Shutdown()

	// Connect to NATS in-process
	nc, err := natsclient.Connect("", natsclient.InProcessServer(srv))
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream: %v", err)
	}

	// Setup stream
	ctx := context.Background()
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("Failed to setup stream: %v", err)
	}

	// Create store
	store := NewStore(js, stream)
	sessionName := "test-session-tasks"

	// Add some tasks
	_, err = store.TaskAdd(ctx, sessionName, TaskAddParams{
		Content:   "Task 1",
		Status:    "remaining",
		Iteration: 1,
	})
	if err != nil {
		t.Fatalf("Failed to add task 1: %v", err)
	}

	_, err = store.TaskAdd(ctx, sessionName, TaskAddParams{
		Content:   "Task 2",
		Status:    "completed",
		Iteration: 1,
	})
	if err != nil {
		t.Fatalf("Failed to add task 2: %v", err)
	}

	// Mark session as complete
	err = store.SessionComplete(ctx, sessionName)
	if err != nil {
		t.Fatalf("Failed to mark session complete: %v", err)
	}

	// Load state to verify
	state, err := store.LoadState(ctx, sessionName)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify session is marked as complete
	if !state.Complete {
		t.Errorf("Expected session to be marked complete, but Complete=false")
	}

	// Verify tasks are still present
	if len(state.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(state.Tasks))
	}
}

// Helper to ensure NATS server is fully ready
func waitForServer(srv *natsserver.Server) {
	if !srv.ReadyForConnections(4 * time.Second) {
		panic("NATS server failed to start")
	}
}
