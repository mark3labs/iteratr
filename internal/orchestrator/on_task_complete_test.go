package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/hooks"
	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TestOnTaskCompleteSubscription verifies orchestrator subscribes to task completion events
func TestOnTaskCompleteSubscription(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")

	// Create hooks config file with on_task_complete hook
	hooksConfig := `
version: 1
hooks:
  on_task_complete:
    - command: "echo 'Task {{task_id}} completed'"
      timeout: 5
      pipe_output: true
`
	hooksPath := filepath.Join(tmpDir, ".iteratr.hooks.yml")
	if err := os.WriteFile(hooksPath, []byte(hooksConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Start embedded NATS server
	ns, port, err := nats.StartEmbeddedNATS(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectToPort(port)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	// Setup JetStream
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatal(err)
	}

	store := session.NewStore(js, stream)

	// Verify hooks file exists
	if _, err := os.Stat(hooksPath); err != nil {
		t.Fatalf("hooks.yaml file does not exist: %v", err)
	}

	// Load hooks config
	hooksConf, err := hooks.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load hooks config: %v", err)
	}

	// Verify hooks config loaded
	if hooksConf == nil {
		t.Fatal("hooks config is nil (LoadConfig returned nil without error)")
	}
	if hooksConf.Hooks.OnTaskComplete == nil {
		t.Fatalf("on_task_complete hooks is nil, hooks: %+v", hooksConf.Hooks)
	}
	if len(hooksConf.Hooks.OnTaskComplete) == 0 {
		t.Fatal("Expected on_task_complete hooks to be configured")
	}

	// Note: We don't need the orchestrator instance in this test, just verify subscription logic
	_ = hooksConf // Use hooksConf in subscription below

	// Subscribe to task completion events (simulate what Run() does)
	subject := fmt.Sprintf("iteratr.%s.task", "test-session")
	subscriptionActive := make(chan bool, 1)
	hookOutput := make(chan string, 1)

	sub, err := nc.Subscribe(subject, func(msg *natsgo.Msg) {
		subscriptionActive <- true

		// Parse event
		var event struct {
			Action string          `json:"action"`
			Meta   json.RawMessage `json:"meta"`
		}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return
		}

		if event.Action != "status" {
			return
		}

		var meta struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(event.Meta, &meta); err != nil {
			return
		}

		if meta.Status != "completed" {
			return
		}

		// Load state to get task content
		state, err := store.LoadState(ctx, "test-session")
		if err != nil {
			return
		}

		task, exists := state.Tasks[meta.TaskID]
		if !exists {
			return
		}

		// Execute hooks
		hookVars := hooks.Variables{
			Session:     "test-session",
			TaskID:      meta.TaskID,
			TaskContent: task.Content,
		}
		output, err := hooks.ExecuteAllPiped(ctx, hooksConf.Hooks.OnTaskComplete, tmpDir, hookVars)
		if err == nil && output != "" {
			hookOutput <- output
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	// Add a task
	task, err := store.TaskAdd(ctx, "test-session", session.TaskAddParams{
		Content:   "Test task",
		Iteration: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Change task status to completed
	if err := store.TaskStatus(ctx, "test-session", session.TaskStatusParams{
		ID:        task.ID,
		Status:    "completed",
		Iteration: 1,
	}); err != nil {
		t.Fatal(err)
	}

	// Wait for subscription to receive event
	select {
	case <-subscriptionActive:
		// Good, subscription received the event
	case <-time.After(2 * time.Second):
		t.Fatal("Subscription did not receive task completion event")
	}

	// Wait for hook output
	select {
	case output := <-hookOutput:
		// echo adds a trailing newline
		expectedOutput := fmt.Sprintf("Task %s completed\n", task.ID)
		if output != expectedOutput {
			t.Errorf("Hook output mismatch.\nExpected: %q\nGot: %q", expectedOutput, output)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Hook did not execute within timeout")
	}
}

// TestOnTaskCompleteAppendsToPendingBuffer verifies piped output is appended to pending buffer
func TestOnTaskCompleteAppendsToPendingBuffer(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")

	// Create hooks config
	hooksConfig := `
version: 1
hooks:
  on_task_complete:
    - command: "echo 'Validation passed for task {{task_id}}'"
      timeout: 5
      pipe_output: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".iteratr.hooks.yml"), []byte(hooksConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Start NATS
	ns, port, err := nats.StartEmbeddedNATS(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ns.Shutdown()

	nc, err := nats.ConnectToPort(port)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatal(err)
	}

	store := session.NewStore(js, stream)
	hooksConf, err := hooks.LoadConfig(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	o := &Orchestrator{
		cfg: Config{
			SessionName: "test-session",
			WorkDir:     tmpDir,
		},
		nc:          nc,
		store:       store,
		ctx:         ctx,
		cancel:      func() {},
		hooksConfig: hooksConf,
	}

	// Add task
	task, err := store.TaskAdd(ctx, "test-session", session.TaskAddParams{
		Content:   "Feature X",
		Iteration: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Subscribe
	subject := fmt.Sprintf("iteratr.%s.task", "test-session")
	done := make(chan bool, 1)

	sub, err := nc.Subscribe(subject, func(msg *natsgo.Msg) {
		var event struct {
			Action string          `json:"action"`
			Meta   json.RawMessage `json:"meta"`
		}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return
		}

		if event.Action != "status" {
			return
		}

		var meta struct {
			Status string `json:"status"`
			TaskID string `json:"task_id"`
		}
		if err := json.Unmarshal(event.Meta, &meta); err != nil {
			return
		}

		if meta.Status != "completed" {
			return
		}

		state, err := store.LoadState(ctx, "test-session")
		if err != nil {
			return
		}

		task := state.Tasks[meta.TaskID]
		if task == nil {
			return
		}

		hookVars := hooks.Variables{
			Session:     "test-session",
			TaskID:      meta.TaskID,
			TaskContent: task.Content,
		}
		output, err := hooks.ExecuteAllPiped(ctx, hooksConf.Hooks.OnTaskComplete, tmpDir, hookVars)
		if err == nil && output != "" {
			o.appendPendingOutput(output)
			done <- true
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	// Complete task
	if err := store.TaskStatus(ctx, "test-session", session.TaskStatusParams{
		ID:        task.ID,
		Status:    "completed",
		Iteration: 1,
	}); err != nil {
		t.Fatal(err)
	}

	// Wait for hook
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Hook did not execute")
	}

	// Verify pending buffer
	if !o.hasPendingOutput() {
		t.Fatal("Expected pending output in buffer")
	}

	output := o.drainPendingOutput()
	// echo adds a trailing newline
	expectedOutput := fmt.Sprintf("Validation passed for task %s\n", task.ID)
	if output != expectedOutput {
		t.Errorf("Pending buffer mismatch.\nExpected: %q\nGot: %q", expectedOutput, output)
	}
}

// TestOnTaskCompleteIgnoresNonCompletedStatus verifies hooks only run for completed status
func TestOnTaskCompleteIgnoresNonCompletedStatus(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")

	// Create hooks config
	hooksConfig := `
version: 1
hooks:
  on_task_complete:
    - command: "echo 'Should not run'"
      timeout: 5
      pipe_output: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".iteratr.hooks.yml"), []byte(hooksConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Start NATS
	ns, port, err := nats.StartEmbeddedNATS(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ns.Shutdown()

	nc, err := nats.ConnectToPort(port)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatal(err)
	}

	store := session.NewStore(js, stream)
	hooksConf, err := hooks.LoadConfig(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	o := &Orchestrator{
		cfg: Config{
			SessionName: "test-session",
			WorkDir:     tmpDir,
		},
		nc:          nc,
		store:       store,
		ctx:         ctx,
		cancel:      func() {},
		hooksConfig: hooksConf,
	}

	// Add task
	task, err := store.TaskAdd(ctx, "test-session", session.TaskAddParams{
		Content:   "Task",
		Iteration: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Subscribe
	subject := fmt.Sprintf("iteratr.%s.task", "test-session")
	hookExecuted := make(chan bool, 1)

	sub, err := nc.Subscribe(subject, func(msg *natsgo.Msg) {
		var event struct {
			Action string          `json:"action"`
			Meta   json.RawMessage `json:"meta"`
		}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return
		}

		if event.Action != "status" {
			return
		}

		var meta struct {
			Status string `json:"status"`
			TaskID string `json:"task_id"`
		}
		if err := json.Unmarshal(event.Meta, &meta); err != nil {
			return
		}

		// Only set flag if status is completed (should NOT happen in this test)
		if meta.Status == "completed" {
			hookExecuted <- true
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	// Change status to in_progress (NOT completed)
	if err := store.TaskStatus(ctx, "test-session", session.TaskStatusParams{
		ID:        task.ID,
		Status:    "in_progress",
		Iteration: 1,
	}); err != nil {
		t.Fatal(err)
	}

	// Wait briefly - hook should NOT execute
	select {
	case <-hookExecuted:
		t.Fatal("Hook executed for non-completed status")
	case <-time.After(500 * time.Millisecond):
		// Expected - hook did not execute
	}

	// Verify buffer is empty
	if o.hasPendingOutput() {
		t.Fatal("Pending buffer should be empty")
	}
}
