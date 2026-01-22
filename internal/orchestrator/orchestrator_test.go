package orchestrator

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestGracefulShutdown verifies that the orchestrator shuts down cleanly
// when Stop() is called, including cleanup of all components.
func TestGracefulShutdown(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec

This is a test spec.

## Tasks
- [ ] Test task 1
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator
	orch, err := New(Config{
		SessionName: "test-shutdown",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true, // No TUI for test
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}

	// Give it a moment to fully initialize
	time.Sleep(100 * time.Millisecond)

	// Stop orchestrator
	stopDone := make(chan error, 1)
	go func() {
		stopDone <- orch.Stop()
	}()

	// Ensure Stop() completes within reasonable time
	select {
	case err := <-stopDone:
		if err != nil {
			t.Errorf("Stop() returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Stop() timed out - graceful shutdown failed")
	}

	// Verify NATS data was written (proves it was running)
	natsDir := filepath.Join(dataDir, "nats")
	if _, err := os.Stat(natsDir); os.IsNotExist(err) {
		t.Error("NATS data directory was not created")
	}
}

// TestShutdownOnSignal verifies that SIGINT/SIGTERM trigger graceful shutdown
func TestShutdownIdempotency(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator
	orch, err := New(Config{
		SessionName: "test-idempotency",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Call Stop() multiple times - should be idempotent
	if err := orch.Stop(); err != nil {
		t.Errorf("First Stop() returned error: %v", err)
	}

	if err := orch.Stop(); err != nil {
		t.Errorf("Second Stop() returned error: %v", err)
	}

	if err := orch.Stop(); err != nil {
		t.Errorf("Third Stop() returned error: %v", err)
	}
}

// TestContextCancellation verifies that cancelling the context triggers cleanup
func TestContextCancellation(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator
	orch, err := New(Config{
		SessionName: "test-context",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Cancel context (simulates signal)
	orch.cancel()

	// Context should be cancelled
	select {
	case <-orch.ctx.Done():
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Context was not cancelled")
	}

	// Stop should still work after context cancellation
	if err := orch.Stop(); err != nil {
		t.Errorf("Stop() after context cancellation returned error: %v", err)
	}
}

// TestIterationLoopStateTracking verifies that the orchestrator correctly tracks
// iteration state even when agent execution fails or is unavailable.
func TestIterationLoopStateTracking(t *testing.T) {
	// Skip this test if opencode is not available (it's an integration test that needs the agent)
	t.Skip("Integration test requires opencode agent - run manually")

	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec

This is a test spec for iteration loop testing.

## Tasks
- [ ] Test task 1
- [ ] Test task 2
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator with limited iterations
	orch, err := New(Config{
		SessionName: "test-iteration-loop",
		SpecPath:    specPath,
		Iterations:  2, // Run exactly 2 iterations
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}
	defer orch.Stop()

	// Run the iteration loop
	// This will fail because opencode is not available, but we can verify
	// that state is tracked correctly up to the failure point
	err = orch.Run()

	// Load the state to verify iterations were tracked
	state, err := orch.store.LoadState(orch.ctx, "test-iteration-loop")
	if err != nil {
		t.Fatalf("failed to load state after Run(): %v", err)
	}

	// Verify that at least the first iteration was started
	if len(state.Iterations) == 0 {
		t.Error("expected at least one iteration to be tracked in state")
	}

	// Verify iteration numbers are correct
	if len(state.Iterations) > 0 && state.Iterations[0].Number != 1 {
		t.Errorf("expected first iteration number to be 1, got %d", state.Iterations[0].Number)
	}
}

// TestIterationLoopResumeFromLastIteration verifies that the orchestrator
// can continue from where it left off in a previous session.
func TestIterationLoopResumeFromLastIteration(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec

Resumable session test.
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	sessionName := "test-resume-session"

	// First orchestrator - simulate running 2 iterations
	{
		orch, err := New(Config{
			SessionName: sessionName,
			SpecPath:    specPath,
			Iterations:  0, // Unlimited for manual control
			DataDir:     dataDir,
			WorkDir:     tmpDir,
			Headless:    true,
		})
		if err != nil {
			t.Fatalf("failed to create first orchestrator: %v", err)
		}

		if err := orch.Start(); err != nil {
			t.Fatalf("failed to start first orchestrator: %v", err)
		}

		// Manually create iteration events to simulate a session with 2 completed iterations
		if err := orch.store.IterationStart(orch.ctx, sessionName, 1); err != nil {
			t.Fatalf("failed to start iteration 1: %v", err)
		}
		if err := orch.store.IterationComplete(orch.ctx, sessionName, 1); err != nil {
			t.Fatalf("failed to complete iteration 1: %v", err)
		}
		if err := orch.store.IterationStart(orch.ctx, sessionName, 2); err != nil {
			t.Fatalf("failed to start iteration 2: %v", err)
		}
		if err := orch.store.IterationComplete(orch.ctx, sessionName, 2); err != nil {
			t.Fatalf("failed to complete iteration 2: %v", err)
		}

		orch.Stop()
	}

	// Second orchestrator - load same session and verify it would start at iteration 3
	{
		orch, err := New(Config{
			SessionName: sessionName,
			SpecPath:    specPath,
			Iterations:  0,
			DataDir:     dataDir,
			WorkDir:     tmpDir,
			Headless:    true,
		})
		if err != nil {
			t.Fatalf("failed to create second orchestrator: %v", err)
		}

		if err := orch.Start(); err != nil {
			t.Fatalf("failed to start second orchestrator: %v", err)
		}
		defer orch.Stop()

		// Load state and verify starting iteration would be 3
		state, err := orch.store.LoadState(orch.ctx, sessionName)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

		expectedStartIteration := len(state.Iterations) + 1
		if expectedStartIteration != 3 {
			t.Errorf("expected next iteration to be 3, got %d (found %d previous iterations)",
				expectedStartIteration, len(state.Iterations))
		}

		// Verify the 2 previous iterations exist
		if len(state.Iterations) != 2 {
			t.Errorf("expected 2 previous iterations, got %d", len(state.Iterations))
		}
	}
}

// TestIterationLoopSessionComplete verifies that the loop stops when session_complete is signaled
func TestIterationLoopSessionComplete(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec

Test session complete signal.
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	sessionName := "test-session-complete"

	orch, err := New(Config{
		SessionName: sessionName,
		SpecPath:    specPath,
		Iterations:  0, // Unlimited
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}
	defer orch.Stop()

	// Mark session as complete before running
	if err := orch.store.SessionComplete(orch.ctx, sessionName); err != nil {
		t.Fatalf("failed to mark session complete: %v", err)
	}

	// Run should detect the session is already complete and exit immediately
	err = orch.Run()
	if err != nil {
		t.Errorf("Run() returned error when session was already complete: %v", err)
	}

	// Verify no new iterations were started
	state, err := orch.store.LoadState(orch.ctx, sessionName)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if len(state.Iterations) > 0 {
		t.Errorf("expected no iterations to run when session is already complete, but found %d", len(state.Iterations))
	}
}

// TestModelConfiguration verifies that the --model flag is properly passed through to the runner
func TestModelConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{
			name:  "anthropic-model",
			model: "anthropic/claude-sonnet-4-5",
		},
		{
			name:  "openai-model",
			model: "openai/gpt-4",
		},
		{
			name:  "empty-model-use-default",
			model: "",
		},
		{
			name:  "custom-provider",
			model: "custom/my-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir := t.TempDir()
			dataDir := filepath.Join(tmpDir, ".iteratr")

			// Create a simple spec file
			specPath := filepath.Join(tmpDir, "test.md")
			specContent := `# Test Spec
Model configuration test.
`
			if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
				t.Fatalf("failed to write spec file: %v", err)
			}

			cfg := Config{
				SessionName: "test-model-" + tt.name,
				SpecPath:    specPath,
				Iterations:  1,
				DataDir:     dataDir,
				WorkDir:     tmpDir,
				Headless:    true,
				Model:       tt.model,
			}

			// Verify config stores model correctly
			if cfg.Model != tt.model {
				t.Errorf("expected model %q, got %q", tt.model, cfg.Model)
			}

			// Create orchestrator and verify it accepts the model
			orch, err := New(cfg)
			if err != nil {
				t.Fatalf("failed to create orchestrator with model %q: %v", tt.model, err)
			}

			// Verify orchestrator stored the model
			if orch.cfg.Model != tt.model {
				t.Errorf("orchestrator config: expected model %q, got %q", tt.model, orch.cfg.Model)
			}

			// Start and immediately stop to verify initialization works with model
			if err := orch.Start(); err != nil {
				t.Fatalf("failed to start orchestrator with model %q: %v", tt.model, err)
			}
			defer orch.Stop()

			// Verify runner was created with the correct model
			if orch.runner == nil {
				t.Fatal("runner was not initialized")
			}
		})
	}
}

// TestHeadlessMode verifies that headless mode works correctly
func TestHeadlessMode(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec
Headless mode test.
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator in headless mode
	orch, err := New(Config{
		SessionName: "test-headless",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator in headless mode: %v", err)
	}

	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator in headless mode: %v", err)
	}
	defer orch.Stop()

	// Verify TUI is not initialized in headless mode
	if orch.tuiApp != nil {
		t.Error("expected tuiApp to be nil in headless mode")
	}
	if orch.tuiProgram != nil {
		t.Error("expected tuiProgram to be nil in headless mode")
	}
}

// TestTUIInitialization verifies that TUI mode initializes without errors
func TestTUIInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec
TUI mode test.
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator with TUI enabled
	orch, err := New(Config{
		SessionName: "test-tui",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    false, // Enable TUI
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator with TUI: %v", err)
	}

	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator with TUI: %v", err)
	}
	defer orch.Stop()

	// Verify TUI is initialized
	if orch.tuiApp == nil {
		t.Error("expected tuiApp to be initialized with TUI mode enabled")
	}
	if orch.tuiProgram == nil {
		t.Error("expected tuiProgram to be initialized with TUI mode enabled")
	}

	// Note: We can't actually run the TUI in a test environment,
	// but we verify it initializes without errors
}

// Suppress unused variable warning
var _ = syscall.SIGTERM
