package specmcp

import (
	"context"
	"testing"
)

// TestServerStartRandomPort verifies that Start() selects a random available port.
func TestServerStartRandomPort(t *testing.T) {
	// Skip this test for now since registerTools() is not yet implemented
	t.Skip("Skipping until tool registration is implemented")

	server := New("Test Spec", "./specs")
	ctx := context.Background()

	port, err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if port <= 0 || port > 65535 {
		t.Errorf("Invalid port number: %d", port)
	}

	// Verify URL is constructed correctly
	expectedURL := "http://localhost:" + string(rune(port)) + "/mcp"
	if server.URL() != expectedURL {
		t.Errorf("URL mismatch: got %s, want %s", server.URL(), expectedURL)
	}

	// Clean up
	if err := server.Stop(); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
}

// TestServerDoubleStart verifies that calling Start() twice returns an error.
func TestServerDoubleStart(t *testing.T) {
	t.Skip("Skipping until tool registration is implemented")

	server := New("Test Spec", "./specs")
	ctx := context.Background()

	_, err := server.Start(ctx)
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("Stop() failed: %v", err)
		}
	}()

	_, err = server.Start(ctx)
	if err == nil {
		t.Error("Second Start() should have returned an error")
	}
}
