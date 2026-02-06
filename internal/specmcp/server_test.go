package specmcp

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	errCh := make(chan error, 1)
	go func() {
		specReq := <-server.SpecContentChan()
		// Verify content was received
		if specReq.Content != "# Test Spec\n\nThis is a test spec." {
			errCh <- fmt.Errorf("Expected content to match, got: %s", specReq.Content)
		} else {
			errCh <- nil
		}
		// Simulate successful save
		specReq.ResultCh <- nil
	}()

	// Call handler
	ctx := context.Background()
	result, err := server.handleFinishSpec(ctx, req)

	if err != nil {
		t.Fatalf("handleFinishSpec returned error: %v", err)
	}

	// Check for errors from goroutine
	if contentErr := <-errCh; contentErr != nil {
		t.Error(contentErr)
	}

	// Verify result
	if result.IsError {
		t.Fatalf("Expected success result, got error: %s", extractText(result))
	}

	text := extractText(result)
	if text != "Spec saved successfully" {
		t.Errorf("Expected 'Spec saved successfully', got '%s'", text)
	}
}

// TestServerStop verifies that Stop() shuts down the server cleanly.
func TestServerStop(t *testing.T) {
	server := New("Test Spec", "./specs")
	ctx := context.Background()

	port, err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if port <= 0 {
		t.Fatalf("Invalid port: %d", port)
	}

	// Stop the server
	err = server.Stop()
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}

	// Verify server state is cleared
	if server.httpServer != nil {
		t.Error("httpServer should be nil after Stop()")
	}
	if server.stdServer != nil {
		t.Error("stdServer should be nil after Stop()")
	}
	if server.mcpServer != nil {
		t.Error("mcpServer should be nil after Stop()")
	}
}

// TestServerStopWithoutStart verifies that Stop() is safe to call without Start().
func TestServerStopWithoutStart(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Stop should be safe even if server was never started
	err := server.Stop()
	if err != nil {
		t.Errorf("Stop() returned error when called without Start(): %v", err)
	}
}

// TestServerDoubleStop verifies that calling Stop() twice is safe.
func TestServerDoubleStop(t *testing.T) {
	server := New("Test Spec", "./specs")
	ctx := context.Background()

	_, err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// First stop
	err = server.Stop()
	if err != nil {
		t.Errorf("First Stop() returned error: %v", err)
	}

	// Second stop should be safe (no-op)
	err = server.Stop()
	if err != nil {
		t.Errorf("Second Stop() returned error: %v", err)
	}
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
	go func() {
		specReq := <-server.SpecContentChan()
		// Simulate save error
		specReq.ResultCh <- fmt.Errorf("failed to write file")
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
}

// TestAskQuestionsHandlerSuccess tests successful ask-questions tool call with single-select.
func TestAskQuestionsHandlerSuccess(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request with single-select question
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask-questions",
			Arguments: map[string]any{
				"questions": []any{
					map[string]any{
						"question": "What type of feature is this?",
						"header":   "Feature Type",
						"options": []any{
							map[string]any{
								"label":       "New Feature",
								"description": "Brand new functionality",
							},
							map[string]any{
								"label":       "Enhancement",
								"description": "Improve existing feature",
							},
						},
						"multiple": false,
					},
				},
			},
		},
	}

	// Start a goroutine to handle the channel request
	validationErrCh := make(chan error, 5) // Buffer for all possible validation errors
	go func() {
		questionReq := <-server.QuestionChan()
		// Verify questions were received correctly
		if len(questionReq.Questions) != 1 {
			validationErrCh <- fmt.Errorf("Expected 1 question, got %d", len(questionReq.Questions))
		}
		q := questionReq.Questions[0]
		if q.Question != "What type of feature is this?" {
			validationErrCh <- fmt.Errorf("Unexpected question text: %s", q.Question)
		}
		if q.Header != "Feature Type" {
			validationErrCh <- fmt.Errorf("Unexpected header: %s", q.Header)
		}
		if len(q.Options) != 2 {
			validationErrCh <- fmt.Errorf("Expected 2 options, got %d", len(q.Options))
		}
		if q.Multiple {
			validationErrCh <- fmt.Errorf("Expected single-select question")
		}
		// Send answer back
		questionReq.ResultCh <- []interface{}{"New Feature"}
		close(validationErrCh)
	}()

	// Call handler
	ctx := context.Background()
	result, err := server.handleAskQuestions(ctx, req)

	if err != nil {
		t.Fatalf("handleAskQuestions returned error: %v", err)
	}

	// Check for validation errors from goroutine
	for valErr := range validationErrCh {
		t.Error(valErr)
	}

	// Verify result
	if result.IsError {
		t.Fatalf("Expected success result, got error: %s", extractText(result))
	}

	// Verify answers are returned as JSON
	text := extractText(result)
	if text != `["New Feature"]` {
		t.Errorf("Expected '[\"New Feature\"]', got '%s'", text)
	}
}

// TestAskQuestionsHandlerMultiSelect tests ask-questions with multi-select question.
func TestAskQuestionsHandlerMultiSelect(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request with multi-select question
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask-questions",
			Arguments: map[string]any{
				"questions": []any{
					map[string]any{
						"question": "Which components will this affect?",
						"header":   "Affected Components",
						"options": []any{
							map[string]any{
								"label":       "Frontend",
								"description": "UI components",
							},
							map[string]any{
								"label":       "Backend",
								"description": "API and services",
							},
							map[string]any{
								"label":       "Database",
								"description": "Schema changes",
							},
						},
						"multiple": true,
					},
				},
			},
		},
	}

	// Start a goroutine to handle the channel request
	var multiErr error
	go func() {
		questionReq := <-server.QuestionChan()
		// Verify it's multi-select
		if !questionReq.Questions[0].Multiple {
			multiErr = fmt.Errorf("Expected multi-select question")
		}
		// Send multiple answers back
		questionReq.ResultCh <- []interface{}{[]string{"Frontend", "Backend"}}
	}()

	// Call handler
	ctx := context.Background()
	result, err := server.handleAskQuestions(ctx, req)

	if err != nil {
		t.Fatalf("handleAskQuestions returned error: %v", err)
	}

	// Check for errors from goroutine
	if multiErr != nil {
		t.Error(multiErr)
	}

	// Verify result
	if result.IsError {
		t.Fatalf("Expected success result, got error: %s", extractText(result))
	}

	// Verify answers are returned as JSON with array
	text := extractText(result)
	if text != `[["Frontend","Backend"]]` {
		t.Errorf("Expected '[[\"Frontend\",\"Backend\"]]', got '%s'", text)
	}
}

// TestAskQuestionsHandlerMissingQuestions tests ask-questions with missing questions parameter.
func TestAskQuestionsHandlerMissingQuestions(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request without questions
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "ask-questions",
			Arguments: map[string]any{},
		},
	}

	ctx := context.Background()
	result, err := server.handleAskQuestions(ctx, req)

	if err != nil {
		t.Fatalf("handleAskQuestions returned error: %v", err)
	}

	// Verify error result
	if !result.IsError {
		t.Fatal("Expected error result")
	}

	text := extractText(result)
	if text != "missing 'questions' parameter" {
		t.Errorf("Expected 'missing 'questions' parameter', got '%s'", text)
	}
}

// TestAskQuestionsHandlerEmptyQuestionsArray tests ask-questions with empty questions array.
func TestAskQuestionsHandlerEmptyQuestionsArray(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request with empty questions array
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask-questions",
			Arguments: map[string]any{
				"questions": []any{},
			},
		},
	}

	ctx := context.Background()
	result, err := server.handleAskQuestions(ctx, req)

	if err != nil {
		t.Fatalf("handleAskQuestions returned error: %v", err)
	}

	// Verify error result
	if !result.IsError {
		t.Fatal("Expected error result")
	}

	text := extractText(result)
	if text != "at least one question is required" {
		t.Errorf("Expected 'at least one question is required', got '%s'", text)
	}
}

// TestAskQuestionsHandlerInvalidQuestion tests ask-questions with missing required fields.
func TestAskQuestionsHandlerInvalidQuestion(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request with question missing header
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask-questions",
			Arguments: map[string]any{
				"questions": []any{
					map[string]any{
						"question": "What is this?",
						"options": []any{
							map[string]any{
								"label":       "Option A",
								"description": "First option",
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := server.handleAskQuestions(ctx, req)

	if err != nil {
		t.Fatalf("handleAskQuestions returned error: %v", err)
	}

	// Verify error result
	if !result.IsError {
		t.Fatal("Expected error result")
	}

	text := extractText(result)
	if text != "question 0 missing or empty 'header' field" {
		t.Errorf("Expected error about missing header, got '%s'", text)
	}
}

// TestAskQuestionsHandlerEmptyOptions tests ask-questions with empty options array.
func TestAskQuestionsHandlerEmptyOptions(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request with empty options
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask-questions",
			Arguments: map[string]any{
				"questions": []any{
					map[string]any{
						"question": "What is this?",
						"header":   "Question",
						"options":  []any{},
					},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := server.handleAskQuestions(ctx, req)

	if err != nil {
		t.Fatalf("handleAskQuestions returned error: %v", err)
	}

	// Verify error result
	if !result.IsError {
		t.Fatal("Expected error result")
	}

	text := extractText(result)
	if text != "question 0 must have at least one option" {
		t.Errorf("Expected error about empty options, got '%s'", text)
	}
}

// TestAskQuestionsHandlerMultipleQuestions tests ask-questions with multiple questions.
func TestAskQuestionsHandlerMultipleQuestions(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request with two questions
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask-questions",
			Arguments: map[string]any{
				"questions": []any{
					map[string]any{
						"question": "First question?",
						"header":   "Q1",
						"options": []any{
							map[string]any{
								"label":       "A",
								"description": "Option A",
							},
						},
					},
					map[string]any{
						"question": "Second question?",
						"header":   "Q2",
						"options": []any{
							map[string]any{
								"label":       "B",
								"description": "Option B",
							},
						},
					},
				},
			},
		},
	}

	// Start a goroutine to handle the channel request
	var countErr error
	go func() {
		questionReq := <-server.QuestionChan()
		// Verify both questions were received
		if len(questionReq.Questions) != 2 {
			countErr = fmt.Errorf("Expected 2 questions, got %d", len(questionReq.Questions))
		}
		// Send answers for both questions
		questionReq.ResultCh <- []interface{}{"A", "B"}
	}()

	// Call handler
	ctx := context.Background()
	result, err := server.handleAskQuestions(ctx, req)

	if err != nil {
		t.Fatalf("handleAskQuestions returned error: %v", err)
	}

	// Check for errors from goroutine
	if countErr != nil {
		t.Error(countErr)
	}

	// Verify result
	if result.IsError {
		t.Fatalf("Expected success result, got error: %s", extractText(result))
	}

	// Verify both answers are returned
	text := extractText(result)
	if text != `["A","B"]` {
		t.Errorf("Expected '[\"A\",\"B\"]', got '%s'", text)
	}
}

// TestAskQuestionsHandlerRejectsDuplicateRequests tests that concurrent ask-questions calls are rejected.
// This prevents duplicate questions from appearing in the UI when the agent calls ask-questions multiple times.
func TestAskQuestionsHandlerRejectsDuplicateRequests(t *testing.T) {
	server := New("Test Spec", "./specs")

	// Create request
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask-questions",
			Arguments: map[string]any{
				"questions": []any{
					map[string]any{
						"question": "Test question?",
						"header":   "Test",
						"options": []any{
							map[string]any{
								"label":       "Option",
								"description": "Test option",
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	type handlerResult struct {
		result *mcp.CallToolResult
		err    error
	}
	firstResultCh := make(chan handlerResult, 1)
	secondResultCh := make(chan handlerResult, 1)

	// First request - will block on channel send (unbuffered channel, no receiver yet)
	go func() {
		result, err := server.handleAskQuestions(ctx, req)
		firstResultCh <- handlerResult{result, err}
	}()

	// Wait for first handler to set questionPending flag
	// It will block on the unbuffered channel send, keeping pending=true
	time.Sleep(50 * time.Millisecond)

	// Second request - should be rejected immediately because first is pending
	go func() {
		result, err := server.handleAskQuestions(ctx, req)
		secondResultCh <- handlerResult{result, err}
	}()

	// Wait for second request to complete (should be fast since it's rejected)
	var secondResult handlerResult
	select {
	case secondResult = <-secondResultCh:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for second handler to be rejected")
	}

	// Verify second request was rejected
	if secondResult.err != nil {
		t.Fatalf("Second handler returned unexpected error: %v", secondResult.err)
	}
	if secondResult.result == nil {
		t.Fatal("Second handler returned nil result")
	}
	if !secondResult.result.IsError {
		t.Fatal("Expected second request to be rejected with error")
	}
	text := extractText(secondResult.result)
	expectedErr := "a question request is already pending - please wait for the user to answer the current questions before asking more"
	if text != expectedErr {
		t.Errorf("Unexpected error message: got %q, want %q", text, expectedErr)
	}

	// Now simulate answering the first request
	go func() {
		questionReq := <-server.QuestionChan()
		questionReq.ResultCh <- []interface{}{"Option"}
	}()

	// Wait for first request to complete
	var firstResult handlerResult
	select {
	case firstResult = <-firstResultCh:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for first handler")
	}

	// Verify first request succeeded
	if firstResult.err != nil {
		t.Fatalf("First handler returned unexpected error: %v", firstResult.err)
	}
	if firstResult.result == nil {
		t.Fatal("First handler returned nil result")
	}
	if firstResult.result.IsError {
		t.Fatalf("Expected first request to succeed, got error: %s", extractText(firstResult.result))
	}
}
