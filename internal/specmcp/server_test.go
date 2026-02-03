package specmcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestServerStartRandomPort verifies that Start() selects a random available port.
func TestServerStartRandomPort(t *testing.T) {
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
	expectedURL := fmt.Sprintf("http://localhost:%d/mcp", port)
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

// extractText is a helper function to extract text from a CallToolResult.
func extractText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		return textContent.Text
	}
	return ""
}

// TestFinishSpecHandlerSuccess tests successful finish-spec tool call.
func TestFinishSpecHandlerSuccess(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "finish-spec",
			Arguments: map[string]any{
				"content": "# Test Spec\n\nThis is a test spec.",
			},
		},
	}

	// Start a goroutine to handle the channel request
	done := make(chan bool)
	go func() {
		select {
		case specReq := <-server.SpecContentChan():
			// Verify content was received
			if specReq.Content != "# Test Spec\n\nThis is a test spec." {
				t.Errorf("Expected content to match, got: %s", specReq.Content)
			}
			// Simulate successful save
			specReq.ResultCh <- nil
			done <- true
		case <-done:
			return
		}
	}()

	// Call handler
	ctx := context.Background()
	result, err := server.handleFinishSpec(ctx, req)

	if err != nil {
		t.Fatalf("handleFinishSpec returned error: %v", err)
	}

	// Verify result
	if result.IsError {
		t.Fatalf("Expected success result, got error: %s", extractText(result))
	}

	text := extractText(result)
	if text != "Spec saved successfully" {
		t.Errorf("Expected 'Spec saved successfully', got '%s'", text)
	}

	close(done)
}

// TestFinishSpecHandlerMissingContent tests finish-spec with missing content parameter.
func TestFinishSpecHandlerMissingContent(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request without content
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "finish-spec",
			Arguments: map[string]any{},
		},
	}

	ctx := context.Background()
	result, err := server.handleFinishSpec(ctx, req)

	if err != nil {
		t.Fatalf("handleFinishSpec returned error: %v", err)
	}

	// Verify error result
	if !result.IsError {
		t.Fatal("Expected error result")
	}

	text := extractText(result)
	if text != "content parameter must be a string" {
		t.Errorf("Expected 'content parameter must be a string', got '%s'", text)
	}
}

// TestFinishSpecHandlerEmptyContent tests finish-spec with empty content string.
func TestFinishSpecHandlerEmptyContent(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request with empty content
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "finish-spec",
			Arguments: map[string]any{
				"content": "",
			},
		},
	}

	ctx := context.Background()
	result, err := server.handleFinishSpec(ctx, req)

	if err != nil {
		t.Fatalf("handleFinishSpec returned error: %v", err)
	}

	// Verify error result
	if !result.IsError {
		t.Fatal("Expected error result")
	}

	text := extractText(result)
	if text != "content cannot be empty" {
		t.Errorf("Expected 'content cannot be empty', got '%s'", text)
	}
}

// TestFinishSpecHandlerSaveError tests finish-spec when UI returns an error.
func TestFinishSpecHandlerSaveError(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "finish-spec",
			Arguments: map[string]any{
				"content": "# Test Spec\n\nTest content.",
			},
		},
	}

	// Start a goroutine to simulate save error
	done := make(chan bool)
	go func() {
		select {
		case specReq := <-server.SpecContentChan():
			// Simulate save error
			specReq.ResultCh <- fmt.Errorf("failed to write file")
			done <- true
		case <-done:
			return
		}
	}()

	// Call handler
	ctx := context.Background()
	result, err := server.handleFinishSpec(ctx, req)

	if err != nil {
		t.Fatalf("handleFinishSpec returned error: %v", err)
	}

	// Verify error result
	if !result.IsError {
		t.Fatal("Expected error result")
	}

	text := extractText(result)
	if text != "failed to write file" {
		t.Errorf("Expected 'failed to write file', got '%s'", text)
	}

	close(done)
}
