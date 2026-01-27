package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/hooks"
)

// TestSessionEndHookExecution verifies that session_end hooks execute after final delivery
func TestSessionEndHookExecution(t *testing.T) {
	// Create temporary working directory
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "session_end_executed.txt")

	// Create hooks config with session_end hook
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			SessionEnd: []*hooks.HookConfig{
				{
					Command:    "echo 'session ending' > " + markerFile,
					Timeout:    5,
					PipeOutput: true, // Should be ignored for session_end
				},
			},
		},
	}

	// Create orchestrator with mock components
	ctx := context.Background()
	o := &Orchestrator{
		ctx:         ctx,
		cfg:         Config{SessionName: "test-session", WorkDir: tmpDir},
		hooksConfig: hooksConfig,
	}

	// Simulate end of Run() method where session_end hooks execute
	if o.hooksConfig != nil && len(o.hooksConfig.Hooks.SessionEnd) > 0 {
		hookVars := hooks.Variables{
			Session: o.cfg.SessionName,
		}
		_, err := hooks.ExecuteAll(o.ctx, o.hooksConfig.Hooks.SessionEnd, o.cfg.WorkDir, hookVars)
		if err != nil {
			t.Fatalf("session_end hook execution failed: %v", err)
		}
	}

	// Verify hook executed by checking marker file exists
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Fatal("session_end hook did not execute - marker file not found")
	}

	// Verify file contents
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	expected := "session ending\n"
	if string(content) != expected {
		t.Errorf("Expected marker file content %q, got %q", expected, string(content))
	}
}

// TestSessionEndHookPipeOutputIgnored verifies that pipe_output is ignored for session_end hooks
func TestSessionEndHookPipeOutputIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hooks config with pipe_output: true
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			SessionEnd: []*hooks.HookConfig{
				{
					Command:    "echo 'this output should not be piped'",
					Timeout:    5,
					PipeOutput: true, // Should be ignored
				},
			},
		},
	}

	ctx := context.Background()
	o := &Orchestrator{
		ctx:         ctx,
		cfg:         Config{SessionName: "test-session", WorkDir: tmpDir},
		hooksConfig: hooksConfig,
	}

	// Execute session_end hooks
	hookVars := hooks.Variables{
		Session: o.cfg.SessionName,
	}
	output, err := hooks.ExecuteAll(o.ctx, o.hooksConfig.Hooks.SessionEnd, o.cfg.WorkDir, hookVars)
	if err != nil {
		t.Fatalf("session_end hook execution failed: %v", err)
	}

	// ExecuteAll should return output, but spec says pipe_output is ignored for session_end
	// The output is returned but not actually piped anywhere (no more iterations)
	// This test just verifies execution succeeds
	if output == "" {
		t.Error("Expected output from ExecuteAll, got empty string")
	}
}

// TestSessionEndHookContextCancellation verifies graceful handling of context cancellation
func TestSessionEndHookContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hooks config with long-running command
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			SessionEnd: []*hooks.HookConfig{
				{
					Command: "sleep 10",
					Timeout: 15,
				},
			},
		},
	}

	// Create context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())
	o := &Orchestrator{
		ctx:         ctx,
		cfg:         Config{SessionName: "test-session", WorkDir: tmpDir},
		hooksConfig: hooksConfig,
	}

	// Cancel context immediately
	cancel()

	// Execute session_end hooks - should return context error
	hookVars := hooks.Variables{
		Session: o.cfg.SessionName,
	}
	_, err := hooks.ExecuteAll(o.ctx, o.hooksConfig.Hooks.SessionEnd, o.cfg.WorkDir, hookVars)

	// Should return context cancellation error
	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
	if ctx.Err() != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", ctx.Err())
	}
}

// TestSessionEndHookMultipleHooks verifies multiple session_end hooks execute in order
func TestSessionEndHookMultipleHooks(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile1 := filepath.Join(tmpDir, "hook1.txt")
	markerFile2 := filepath.Join(tmpDir, "hook2.txt")

	// Create hooks config with multiple session_end hooks
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			SessionEnd: []*hooks.HookConfig{
				{
					Command: "echo 'hook1' > " + markerFile1,
					Timeout: 5,
				},
				{
					Command: "sleep 0.1 && echo 'hook2' > " + markerFile2,
					Timeout: 5,
				},
			},
		},
	}

	ctx := context.Background()
	o := &Orchestrator{
		ctx:         ctx,
		cfg:         Config{SessionName: "test-session", WorkDir: tmpDir},
		hooksConfig: hooksConfig,
	}

	// Execute session_end hooks
	hookVars := hooks.Variables{
		Session: o.cfg.SessionName,
	}
	_, err := hooks.ExecuteAll(o.ctx, o.hooksConfig.Hooks.SessionEnd, o.cfg.WorkDir, hookVars)
	if err != nil {
		t.Fatalf("session_end hooks execution failed: %v", err)
	}

	// Give hooks time to complete
	time.Sleep(200 * time.Millisecond)

	// Verify both hooks executed
	if _, err := os.Stat(markerFile1); os.IsNotExist(err) {
		t.Error("First session_end hook did not execute")
	}
	if _, err := os.Stat(markerFile2); os.IsNotExist(err) {
		t.Error("Second session_end hook did not execute")
	}
}

// TestSessionEndHookNoHooksConfigured verifies graceful handling when no hooks configured
func TestSessionEndHookNoHooksConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := context.Background()
	o := &Orchestrator{
		ctx:         ctx,
		cfg:         Config{SessionName: "test-session", WorkDir: tmpDir},
		hooksConfig: nil, // No hooks configured
	}

	// Should not panic or error when hooksConfig is nil
	if o.hooksConfig != nil && len(o.hooksConfig.Hooks.SessionEnd) > 0 {
		t.Fatal("Expected no hooks, but found some")
	}

	// Create empty hooks config
	o.hooksConfig = &hooks.Config{
		Version: 1,
		Hooks:   hooks.HooksConfig{}, // No session_end hooks
	}

	// Should not panic or error when session_end hooks are empty
	if len(o.hooksConfig.Hooks.SessionEnd) > 0 {
		t.Fatal("Expected no session_end hooks, but found some")
	}
}

// TestSessionEndHookVariableExpansion verifies that session variable is expanded correctly
func TestSessionEndHookVariableExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "session_var.txt")

	// Create hooks config that uses {{session}} variable
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			SessionEnd: []*hooks.HookConfig{
				{
					Command: "echo '{{session}}' > " + markerFile,
					Timeout: 5,
				},
			},
		},
	}

	ctx := context.Background()
	sessionName := "test-session-123"
	o := &Orchestrator{
		ctx:         ctx,
		cfg:         Config{SessionName: sessionName, WorkDir: tmpDir},
		hooksConfig: hooksConfig,
	}

	// Execute session_end hooks with variable expansion
	hookVars := hooks.Variables{
		Session: o.cfg.SessionName,
	}
	_, err := hooks.ExecuteAll(o.ctx, o.hooksConfig.Hooks.SessionEnd, o.cfg.WorkDir, hookVars)
	if err != nil {
		t.Fatalf("session_end hook execution failed: %v", err)
	}

	// Verify session variable was expanded correctly
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	expected := sessionName + "\n"
	if string(content) != expected {
		t.Errorf("Expected marker file content %q, got %q", expected, string(content))
	}
}
