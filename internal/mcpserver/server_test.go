package mcpserver

import (
	"context"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestNew(t *testing.T) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream
	js, err := nats.CreateJetStream(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream: %v", err)
	}

	// Setup stream
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("failed to setup stream: %v", err)
	}

	// Create store
	store := session.NewStore(js, stream)

	// Create server
	sessionName := "test-session"
	srv := New(store, sessionName)

	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.store != store {
		t.Error("expected store to be set")
	}
	if srv.sessName != sessionName {
		t.Errorf("expected session name %s, got %s", sessionName, srv.sessName)
	}
	if srv.mcpServer != nil {
		t.Error("expected mcpServer to be nil before Start")
	}
	if srv.httpServer != nil {
		t.Error("expected httpServer to be nil before Start")
	}
	if srv.port != 0 {
		t.Errorf("expected port to be 0 before Start, got %d", srv.port)
	}
}

func TestURL_BeforeStart(t *testing.T) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream
	js, err := nats.CreateJetStream(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream: %v", err)
	}

	// Setup stream
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("failed to setup stream: %v", err)
	}

	// Create store
	store := session.NewStore(js, stream)

	// Create server
	srv := New(store, "test-session")

	// URL before start should return port 0
	url := srv.URL()
	expectedURL := "http://localhost:0/mcp"
	if url != expectedURL {
		t.Errorf("expected URL %s before start, got %s", expectedURL, url)
	}
}

func TestStop_NotStarted(t *testing.T) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream
	js, err := nats.CreateJetStream(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream: %v", err)
	}

	// Setup stream
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("failed to setup stream: %v", err)
	}

	// Create store
	store := session.NewStore(js, stream)

	// Create server
	srv := New(store, "test-session")

	// Stop should be safe even if never started
	if err := srv.Stop(); err != nil {
		t.Errorf("Stop on unstarted server should be safe, got error: %v", err)
	}
}

func TestStartPort_RandomAssignment(t *testing.T) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream
	js, err := nats.CreateJetStream(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream: %v", err)
	}

	// Setup stream
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("failed to setup stream: %v", err)
	}

	// Create store
	store := session.NewStore(js, stream)

	// Start multiple servers and ensure they get different ports
	srv1 := New(store, "test-session-1")
	port1, err := srv1.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server 1: %v", err)
	}
	defer func() {
		if err := srv1.Stop(); err != nil {
			t.Errorf("failed to stop server 1: %v", err)
		}
	}()

	srv2 := New(store, "test-session-2")
	port2, err := srv2.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server 2: %v", err)
	}
	defer func() {
		if err := srv2.Stop(); err != nil {
			t.Errorf("failed to stop server 2: %v", err)
		}
	}()

	// Verify different ports
	if port1 == port2 {
		t.Errorf("expected different ports, got %d for both servers", port1)
	}

	// Verify ports are valid
	if port1 <= 0 || port1 > 65535 {
		t.Errorf("port1 %d is out of valid range", port1)
	}
	if port2 <= 0 || port2 > 65535 {
		t.Errorf("port2 %d is out of valid range", port2)
	}
}
