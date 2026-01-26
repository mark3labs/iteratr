package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/iteratr/internal/logger"
)

// Runner manages the execution of opencode run subprocess for each iteration.
type Runner struct {
	model       string
	workDir     string
	sessionName string
	natsPort    int
	onText      func(text string)
	onToolCall  func(ToolCallEvent)
	onThinking  func(string)
	onFinish    func(FinishEvent)

	// ACP subprocess (reused) and current session (created fresh per iteration)
	conn      *acpConn
	sessionID string // Current session ID (replaced each iteration for fresh context)
	cmd       *exec.Cmd
}

// RunnerConfig holds configuration for creating a new Runner.
type RunnerConfig struct {
	Model       string              // LLM model to use (e.g., "anthropic/claude-sonnet-4-5")
	WorkDir     string              // Working directory for agent
	SessionName string              // Session name
	NATSPort    int                 // NATS server port for tool CLI
	OnText      func(text string)   // Callback for text output
	OnToolCall  func(ToolCallEvent) // Callback for tool lifecycle events
	OnThinking  func(string)        // Callback for thinking/reasoning output
	OnFinish    func(FinishEvent)   // Callback for iteration finish events
}

// NewRunner creates a new Runner instance.
func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{
		model:       cfg.Model,
		workDir:     cfg.WorkDir,
		sessionName: cfg.SessionName,
		natsPort:    cfg.NATSPort,
		onText:      cfg.OnText,
		onToolCall:  cfg.OnToolCall,
		onThinking:  cfg.OnThinking,
		onFinish:    cfg.OnFinish,
	}
}

// extractProvider parses provider name from model string.
// Model format is typically "provider/model-name" (e.g., "anthropic/claude-sonnet-4-5").
// Returns capitalized provider name (e.g., "Anthropic") or empty string if no slash.
func extractProvider(model string) string {
	if idx := strings.Index(model, "/"); idx >= 0 {
		provider := model[:idx]
		// Capitalize first letter
		if len(provider) > 0 {
			return strings.ToUpper(provider[:1]) + provider[1:]
		}
		return provider
	}
	return ""
}

// Start spawns the opencode acp subprocess and initializes the ACP protocol.
// Sessions are created fresh per iteration for clean context.
// Must be called before RunIteration.
func (r *Runner) Start(ctx context.Context) error {
	logger.Debug("Starting ACP subprocess")

	// Create command - spawn opencode acp
	cmd := exec.CommandContext(ctx, "opencode", "acp")
	cmd.Dir = r.workDir
	cmd.Env = os.Environ()
	// Don't inherit stderr - it corrupts terminal state during TUI shutdown
	// Subprocess errors are captured via the ACP protocol

	// Setup stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Setup stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	logger.Debug("Starting opencode subprocess")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Create acpConn from stdin/stdout pipes
	conn := newACPConn(stdin, stdout)

	// Initialize ACP protocol (handshake only - sessions created per iteration)
	if err := conn.initialize(ctx); err != nil {
		_ = conn.close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("ACP initialize failed: %w", err)
	}

	// Store subprocess state (no session yet - created fresh per iteration)
	r.conn = conn
	r.cmd = cmd

	logger.Debug("ACP subprocess ready")
	return nil
}

// RunIteration executes a single iteration with fresh context by creating a new ACP session.
// Optional hookOutput is sent as a separate content block before the main prompt.
// Start() must be called first to initialize the subprocess.
func (r *Runner) RunIteration(ctx context.Context, prompt string, hookOutput string) error {
	if r.conn == nil {
		return fmt.Errorf("ACP subprocess not started - call Start() first")
	}

	// Create fresh session for this iteration (clean context)
	logger.Debug("Creating new ACP session for iteration")
	sessID, err := r.conn.newSession(ctx, r.workDir)
	if err != nil {
		return fmt.Errorf("ACP new session failed: %w", err)
	}
	r.sessionID = sessID

	// Set model for the new session
	if r.model != "" {
		logger.Debug("Setting model: %s", r.model)
		if err := r.conn.setModel(ctx, sessID, r.model); err != nil {
			return fmt.Errorf("ACP set model failed: %w", err)
		}
	}

	logger.Debug("Running iteration on fresh ACP session: %s", sessID)

	// Build content blocks: hook output (if any) + main prompt
	var texts []string
	if hookOutput != "" {
		texts = append(texts, hookOutput)
		logger.Debug("Including hook output: %d bytes", len(hookOutput))
	}
	texts = append(texts, prompt)

	// Send prompt and stream notifications to callbacks
	// Wire onText, onToolCall, and onThinking callbacks through to prompt()
	startTime := time.Now()
	stopReason, err := r.conn.prompt(ctx, r.sessionID, texts, r.onText, r.onToolCall, r.onThinking)
	duration := time.Since(startTime)

	if err != nil {
		// Prompt failed - determine if it was cancelled or error
		if r.onFinish != nil {
			finalStopReason := "error"
			if ctx.Err() == context.Canceled {
				finalStopReason = "cancelled"
			}
			r.onFinish(FinishEvent{
				StopReason: finalStopReason,
				Error:      err.Error(),
				Duration:   duration,
				Model:      r.model,
				Provider:   extractProvider(r.model),
			})
		}
		return fmt.Errorf("ACP prompt failed: %w", err)
	}

	// Prompt succeeded - call onFinish with the actual stop reason from ACP
	if r.onFinish != nil {
		r.onFinish(FinishEvent{
			StopReason: stopReason,
			Duration:   duration,
			Model:      r.model,
			Provider:   extractProvider(r.model),
		})
	}

	logger.Debug("opencode iteration completed successfully")
	return nil
}

// SendMessages sends multiple user messages to the current ACP session as a single prompt.
// Each message becomes a separate content block in the request.
// This allows batching queued user input while keeping them logically distinct.
// RunIteration() must be called first to create a session.
func (r *Runner) SendMessages(ctx context.Context, texts []string) error {
	if r.conn == nil {
		return fmt.Errorf("ACP subprocess not started - call Start() first")
	}
	if r.sessionID == "" {
		return fmt.Errorf("no active session - call RunIteration() first")
	}

	if len(texts) == 0 {
		return nil
	}

	logger.Debug("Sending %d user message(s) to ACP session", len(texts))

	// Send prompt with all messages as separate content blocks
	startTime := time.Now()
	stopReason, err := r.conn.prompt(ctx, r.sessionID, texts, r.onText, r.onToolCall, r.onThinking)
	duration := time.Since(startTime)

	if err != nil {
		// Prompt failed - determine if it was cancelled or error
		if r.onFinish != nil {
			finalStopReason := "error"
			if ctx.Err() == context.Canceled {
				finalStopReason = "cancelled"
			}
			r.onFinish(FinishEvent{
				StopReason: finalStopReason,
				Error:      err.Error(),
				Duration:   duration,
				Model:      r.model,
				Provider:   extractProvider(r.model),
			})
		}
		return fmt.Errorf("ACP user message failed: %w", err)
	}

	// Prompt succeeded - call onFinish with the actual stop reason from ACP
	if r.onFinish != nil {
		r.onFinish(FinishEvent{
			StopReason: stopReason,
			Duration:   duration,
			Model:      r.model,
			Provider:   extractProvider(r.model),
		})
	}

	logger.Debug("User message processed successfully")
	return nil
}

// Stop terminates the ACP subprocess and cleans up resources.
// Should be called when done with the runner (e.g., on orchestrator exit).
func (r *Runner) Stop() {
	if r.conn != nil {
		logger.Debug("Closing ACP connection")
		_ = r.conn.close()
		r.conn = nil
	}
	if r.cmd != nil && r.cmd.Process != nil {
		logger.Debug("Terminating opencode subprocess")
		_ = r.cmd.Process.Kill()
		_ = r.cmd.Wait()
		r.cmd = nil
	}
	r.sessionID = ""
	logger.Debug("ACP session stopped")
}
