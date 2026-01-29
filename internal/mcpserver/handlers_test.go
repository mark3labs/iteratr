package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/mcp-go/mcp"
)

// setupTestServer creates a server with a test store
func setupTestServer(t *testing.T) (*Server, func()) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}

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

	cleanup := func() {
		nc.Close()
		ns.Shutdown()
	}

	return srv, cleanup
}

// extractText extracts text from CallToolResult.Content[0]
func extractText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		return textContent.Text
	}
	return ""
}

func TestHandleTaskAdd_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create request with tasks array
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content":  "Test task 1",
						"status":   "remaining",
						"priority": float64(2),
					},
					map[string]any{
						"content": "Test task 2",
					},
				},
			},
		},
	}

	// Call handler
	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	// Check result
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}

	text := extractText(result)
	if !strings.Contains(text, "Added 2 task(s)") {
		t.Errorf("unexpected result: %s", text)
	}
	if !strings.Contains(text, "TAS-1") {
		t.Errorf("missing TAS-1 in result: %s", text)
	}
	if !strings.Contains(text, "TAS-2") {
		t.Errorf("missing TAS-2 in result: %s", text)
	}
}

func TestHandleTaskAdd_MissingTasksParam(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task-add",
			Arguments: map[string]any{},
		},
	}

	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error: missing 'tasks' parameter") {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestHandleTaskAdd_EmptyContent(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "",
					},
				},
			},
		},
	}

	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") && !strings.Contains(text, "content") {
		t.Errorf("expected content validation error, got: %s", text)
	}
}

func TestHandleTaskAdd_DuplicateContent(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add first task
	req1 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Duplicate task",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, req1)
	if err != nil {
		t.Fatalf("first handleTaskAdd failed: %v", err)
	}

	// Try to add duplicate
	req2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Duplicate task",
					},
				},
			},
		},
	}

	result, err := srv.handleTaskAdd(ctx, req2)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") && !strings.Contains(text, "already exists") {
		t.Errorf("expected duplicate error, got: %s", text)
	}
}

func TestHandleTaskUpdate_StatusOnly(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task first
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Test task for update",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Update status
	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":     "TAS-1",
				"status": "in_progress",
			},
		},
	}

	result, err := srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "Updated task TAS-1") {
		t.Errorf("expected success message, got: %s", text)
	}
	if !strings.Contains(text, "status=in_progress") {
		t.Errorf("expected status update in message, got: %s", text)
	}
}

func TestHandleTaskUpdate_PriorityOnly(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task first
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Test task for priority update",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Update priority
	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":       "TAS-1",
				"priority": float64(1), // JSON numbers come as float64
			},
		},
	}

	result, err := srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "Updated task TAS-1") {
		t.Errorf("expected success message, got: %s", text)
	}
	if !strings.Contains(text, "priority=1") {
		t.Errorf("expected priority update in message, got: %s", text)
	}
}

func TestHandleTaskUpdate_DependencyOnly(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add two tasks
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Dependency task",
					},
					map[string]any{
						"content": "Dependent task",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add tasks: %v", err)
	}

	// Update depends_on
	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":         "TAS-2",
				"depends_on": "TAS-1",
			},
		},
	}

	result, err := srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "Updated task TAS-2") {
		t.Errorf("expected success message, got: %s", text)
	}
	if !strings.Contains(text, "depends_on=TAS-1") {
		t.Errorf("expected dependency update in message, got: %s", text)
	}
}

func TestHandleTaskUpdate_MultipleFields(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Test task for multiple updates",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Update multiple fields
	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":       "TAS-1",
				"status":   "in_progress",
				"priority": float64(0),
			},
		},
	}

	result, err := srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "Updated task TAS-1") {
		t.Errorf("expected success message, got: %s", text)
	}
	if !strings.Contains(text, "status=in_progress") {
		t.Errorf("expected status update in message, got: %s", text)
	}
	if !strings.Contains(text, "priority=0") {
		t.Errorf("expected priority update in message, got: %s", text)
	}
}

func TestHandleTaskUpdate_MissingID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"status": "completed",
			},
		},
	}

	result, err := srv.handleTaskUpdate(context.Background(), updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "id") {
		t.Errorf("expected missing ID error, got: %s", text)
	}
}

func TestHandleTaskUpdate_NoUpdateParams(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task first
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Test task",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Try to update with no update params
	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id": "TAS-1",
			},
		},
	}

	result, err := srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "no valid update parameters") {
		t.Errorf("expected no update params error, got: %s", text)
	}
}

func TestHandleTaskUpdate_InvalidTaskID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":     "TAS-999",
				"status": "completed",
			},
		},
	}

	result, err := srv.handleTaskUpdate(context.Background(), updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") {
		t.Errorf("expected error for invalid task ID, got: %s", text)
	}
}

func TestHandleTaskList_Empty(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-list",
		},
	}

	result, err := srv.handleTaskList(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskList returned error: %v", err)
	}

	text := extractText(result)
	if text != "No tasks" {
		t.Errorf("expected 'No tasks', got: %s", text)
	}
}

func TestHandleTaskList_WithTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add tasks with different statuses
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Remaining task",
						"status":  "remaining",
					},
					map[string]any{
						"content": "In progress task",
						"status":  "in_progress",
					},
					map[string]any{
						"content": "Completed task",
						"status":  "completed",
					},
					map[string]any{
						"content": "Blocked task",
						"status":  "blocked",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add tasks: %v", err)
	}

	// List tasks
	listReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-list",
		},
	}

	result, err := srv.handleTaskList(ctx, listReq)
	if err != nil {
		t.Fatalf("handleTaskList returned error: %v", err)
	}

	text := extractText(result)

	// Check for section headers (formatted with capital first letter and spaces)
	if !strings.Contains(text, "Remaining:") {
		t.Errorf("expected 'Remaining:' section, got: %s", text)
	}
	if !strings.Contains(text, "In progress:") {
		t.Errorf("expected 'In progress:' section, got: %s", text)
	}
	if !strings.Contains(text, "Completed:") {
		t.Errorf("expected 'Completed:' section, got: %s", text)
	}
	if !strings.Contains(text, "Blocked:") {
		t.Errorf("expected 'Blocked:' section, got: %s", text)
	}

	// Check for task IDs and content
	if !strings.Contains(text, "[TAS-1]") || !strings.Contains(text, "Remaining task") {
		t.Errorf("expected TAS-1 in result, got: %s", text)
	}
	if !strings.Contains(text, "[TAS-2]") || !strings.Contains(text, "In progress task") {
		t.Errorf("expected TAS-2 in result, got: %s", text)
	}
	if !strings.Contains(text, "[TAS-3]") || !strings.Contains(text, "Completed task") {
		t.Errorf("expected TAS-3 in result, got: %s", text)
	}
	if !strings.Contains(text, "[TAS-4]") || !strings.Contains(text, "Blocked task") {
		t.Errorf("expected TAS-4 in result, got: %s", text)
	}
}

func TestHandleTaskNext_NoTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-next",
		},
	}

	result, err := srv.handleTaskNext(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskNext returned error: %v", err)
	}

	text := extractText(result)
	if text != "No ready tasks" {
		t.Errorf("expected 'No ready tasks', got: %s", text)
	}
}

func TestHandleTaskNext_WithReadyTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add tasks with different priorities
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content":  "Low priority task",
						"priority": float64(3),
					},
					map[string]any{
						"content":  "High priority task",
						"priority": float64(1),
					},
					map[string]any{
						"content":  "Medium priority task",
						"priority": float64(2),
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add tasks: %v", err)
	}

	// Get next task
	nextReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-next",
		},
	}

	result, err := srv.handleTaskNext(ctx, nextReq)
	if err != nil {
		t.Fatalf("handleTaskNext returned error: %v", err)
	}

	text := extractText(result)

	// Should return JSON format
	var taskData map[string]any
	if err := json.Unmarshal([]byte(text), &taskData); err != nil {
		t.Fatalf("expected JSON output, got: %s", text)
	}

	// Should be the high priority task (TAS-2 with priority 1)
	if taskData["id"] != "TAS-2" {
		t.Errorf("expected TAS-2 (highest priority), got: %v", taskData["id"])
	}
	if taskData["content"] != "High priority task" {
		t.Errorf("expected 'High priority task', got: %v", taskData["content"])
	}
	if taskData["priority"] != float64(1) {
		t.Errorf("expected priority 1, got: %v", taskData["priority"])
	}
}

func TestHandleTaskNext_SkipsBlocked(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add tasks
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content":  "High priority task",
						"priority": float64(0),
					},
					map[string]any{
						"content":  "Low priority task",
						"priority": float64(2),
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add tasks: %v", err)
	}

	// Mark high priority task as in_progress (not remaining)
	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":     "TAS-1",
				"status": "in_progress",
			},
		},
	}
	_, err = srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	// Get next task
	nextReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-next",
		},
	}

	result, err := srv.handleTaskNext(ctx, nextReq)
	if err != nil {
		t.Fatalf("handleTaskNext returned error: %v", err)
	}

	text := extractText(result)

	var taskData map[string]any
	if err := json.Unmarshal([]byte(text), &taskData); err != nil {
		t.Fatalf("expected JSON output, got: %s", text)
	}

	// Should return TAS-2 (only remaining task)
	if taskData["id"] != "TAS-2" {
		t.Errorf("expected TAS-2 (only remaining task), got: %v", taskData["id"])
	}
}

func TestHandleIterationSummary_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task and mark it in_progress to simulate work
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Test task for summary",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":     "TAS-1",
				"status": "in_progress",
			},
		},
	}
	_, err = srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	// Record iteration summary
	summaryReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "iteration-summary",
			Arguments: map[string]any{
				"summary": "Implemented test task",
			},
		},
	}

	result, err := srv.handleIterationSummary(ctx, summaryReq)
	if err != nil {
		t.Fatalf("handleIterationSummary returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "Summary recorded for iteration #1") {
		t.Errorf("expected success message with iteration number, got: %s", text)
	}
}

func TestHandleIterationSummary_MissingSummary(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "iteration-summary",
			Arguments: map[string]any{},
		},
	}

	result, err := srv.handleIterationSummary(context.Background(), req)
	if err != nil {
		t.Fatalf("handleIterationSummary returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "summary") {
		t.Errorf("expected missing summary error, got: %s", text)
	}
}

func TestHandleSessionComplete_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task and complete it
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Test task to complete",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":     "TAS-1",
				"status": "completed",
			},
		},
	}
	_, err = srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	// Mark session complete
	completeReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "session-complete",
		},
	}

	result, err := srv.handleSessionComplete(ctx, completeReq)
	if err != nil {
		t.Fatalf("handleSessionComplete returned error: %v", err)
	}

	text := extractText(result)
	if text != "Session marked complete" {
		t.Errorf("expected success message, got: %s", text)
	}
}

func TestHandleSessionComplete_IncompleteTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task but leave it in remaining status
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Incomplete task",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Try to mark session complete with incomplete task
	completeReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "session-complete",
		},
	}

	result, err := srv.handleSessionComplete(ctx, completeReq)
	if err != nil {
		t.Fatalf("handleSessionComplete returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") {
		t.Errorf("expected error for incomplete tasks, got: %s", text)
	}
	if !strings.Contains(text, "not in terminal state") {
		t.Errorf("expected 'not in terminal state' in error message, got: %s", text)
	}
}

func TestHandleTaskAdd_InvalidTasksType(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": "not an array", // Invalid type
			},
		},
	}

	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "not an array") {
		t.Errorf("expected type error, got: %s", text)
	}
}

func TestHandleTaskAdd_EmptyArray(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{}, // Empty array
			},
		},
	}

	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "at least one task") {
		t.Errorf("expected empty array error, got: %s", text)
	}
}

func TestHandleTaskAdd_InvalidTaskObject(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					"not an object", // Invalid task type
				},
			},
		},
	}

	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "not an object") {
		t.Errorf("expected object type error, got: %s", text)
	}
}

func TestHandleTaskUpdate_InvalidPriorityType(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add a task first
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Test task",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Try to update with invalid priority type
	updateReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":       "TAS-1",
				"priority": "invalid", // String instead of number
			},
		},
	}

	result, err := srv.handleTaskUpdate(ctx, updateReq)
	if err != nil {
		t.Fatalf("handleTaskUpdate returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "must be a number") {
		t.Errorf("expected priority type error, got: %s", text)
	}
}

func TestHandleTaskList_AllStatuses(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add tasks with all possible statuses
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Remaining task",
						"status":  "remaining",
					},
					map[string]any{
						"content": "In progress task",
						"status":  "in_progress",
					},
					map[string]any{
						"content": "Completed task",
						"status":  "completed",
					},
					map[string]any{
						"content": "Blocked task",
						"status":  "blocked",
					},
					map[string]any{
						"content": "Cancelled task",
						"status":  "cancelled",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add tasks: %v", err)
	}

	// List all tasks
	listReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-list",
		},
	}

	result, err := srv.handleTaskList(ctx, listReq)
	if err != nil {
		t.Fatalf("handleTaskList returned error: %v", err)
	}

	text := extractText(result)

	// Verify all status sections appear
	if !strings.Contains(text, "Remaining:") {
		t.Errorf("missing 'Remaining:' section")
	}
	if !strings.Contains(text, "In progress:") {
		t.Errorf("missing 'In progress:' section")
	}
	if !strings.Contains(text, "Completed:") {
		t.Errorf("missing 'Completed:' section")
	}
	if !strings.Contains(text, "Blocked:") {
		t.Errorf("missing 'Blocked:' section")
	}
	if !strings.Contains(text, "Cancelled:") {
		t.Errorf("missing 'Cancelled:' section")
	}

	// Verify all task IDs are present
	if !strings.Contains(text, "TAS-1") {
		t.Errorf("missing task ID TAS-1 in output")
	}
	if !strings.Contains(text, "TAS-2") {
		t.Errorf("missing task ID TAS-2 in output")
	}
	if !strings.Contains(text, "TAS-3") {
		t.Errorf("missing task ID TAS-3 in output")
	}
	if !strings.Contains(text, "TAS-4") {
		t.Errorf("missing task ID TAS-4 in output")
	}
	if !strings.Contains(text, "TAS-5") {
		t.Errorf("missing task ID TAS-5 in output")
	}
}

func TestHandleTaskNext_OnlyNonRemainingTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add tasks and mark them all as non-remaining
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Task 1",
					},
					map[string]any{
						"content": "Task 2",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, addReq)
	if err != nil {
		t.Fatalf("failed to add tasks: %v", err)
	}

	// Mark both as completed
	updateReq1 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":     "TAS-1",
				"status": "completed",
			},
		},
	}
	_, err = srv.handleTaskUpdate(ctx, updateReq1)
	if err != nil {
		t.Fatalf("failed to update task 1: %v", err)
	}

	updateReq2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-update",
			Arguments: map[string]any{
				"id":     "TAS-2",
				"status": "completed",
			},
		},
	}
	_, err = srv.handleTaskUpdate(ctx, updateReq2)
	if err != nil {
		t.Fatalf("failed to update task 2: %v", err)
	}

	// Try to get next task
	nextReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-next",
		},
	}

	result, err := srv.handleTaskNext(ctx, nextReq)
	if err != nil {
		t.Fatalf("handleTaskNext returned error: %v", err)
	}

	text := extractText(result)
	if text != "No ready tasks" {
		t.Errorf("expected 'No ready tasks', got: %s", text)
	}
}

func TestHandleIterationSummary_EmptySummary(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "iteration-summary",
			Arguments: map[string]any{
				"summary": "", // Empty string
			},
		},
	}

	result, err := srv.handleIterationSummary(context.Background(), req)
	if err != nil {
		t.Fatalf("handleIterationSummary returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") || !strings.Contains(text, "summary") {
		t.Errorf("expected empty summary error, got: %s", text)
	}
}

func TestHandleSessionComplete_EmptySession(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Try to mark session complete with no tasks added
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "session-complete",
		},
	}

	result, err := srv.handleSessionComplete(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSessionComplete returned error: %v", err)
	}

	text := extractText(result)
	// Empty session should succeed (no tasks to check)
	if text != "Session marked complete" {
		t.Errorf("expected success for empty session, got: %s", text)
	}
}
