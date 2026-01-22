package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/mark3labs/iteratr/internal/logger"
)

// Runner manages the execution of opencode run subprocess for each iteration.
type Runner struct {
	model       string
	workDir     string
	sessionName string
	natsPort    int
	onText      func(text string)
	onToolUse   func(name string, input map[string]any)
	onError     func(err error)
}

// RunnerConfig holds configuration for creating a new Runner.
type RunnerConfig struct {
	Model       string                                  // LLM model to use (e.g., "anthropic/claude-sonnet-4-5")
	WorkDir     string                                  // Working directory for agent
	SessionName string                                  // Session name
	NATSPort    int                                     // NATS server port for tool CLI
	OnText      func(text string)                       // Callback for text output
	OnToolUse   func(name string, input map[string]any) // Callback for tool use
	OnError     func(err error)                         // Callback for errors
}

// NewRunner creates a new Runner instance.
func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{
		model:       cfg.Model,
		workDir:     cfg.WorkDir,
		sessionName: cfg.SessionName,
		natsPort:    cfg.NATSPort,
		onText:      cfg.OnText,
		onToolUse:   cfg.OnToolUse,
		onError:     cfg.OnError,
	}
}

// RunIteration executes a single iteration by spawning opencode run subprocess.
// It sends the prompt via stdin and parses JSON events from stdout.
func (r *Runner) RunIteration(ctx context.Context, prompt string) error {
	logger.Debug("Starting opencode run iteration")

	// Build command arguments
	args := []string{"run", "--format", "json"}
	if r.model != "" {
		args = append(args, "--model", r.model)
		logger.Debug("Using model: %s", r.model)
	}

	// Create command
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = r.workDir
	cmd.Env = os.Environ()

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

	// Stderr goes to our stderr
	cmd.Stderr = os.Stderr

	// Start the command
	logger.Debug("Starting opencode subprocess")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Send prompt to stdin
	logger.Debug("Sending prompt to opencode (length: %d)", len(prompt))
	if _, err := io.WriteString(stdin, prompt); err != nil {
		logger.Error("Failed to write prompt: %v", err)
		return fmt.Errorf("failed to write prompt: %w", err)
	}
	stdin.Close()

	// Parse JSON events from stdout
	logger.Debug("Parsing JSON events from opencode")
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		r.parseEvent(line)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Scanner error: %v", err)
		return fmt.Errorf("failed to read output: %w", err)
	}

	// Wait for process to complete
	logger.Debug("Waiting for opencode process to exit")
	if err := cmd.Wait(); err != nil {
		logger.Error("opencode exited with error: %v", err)
		return fmt.Errorf("opencode failed: %w", err)
	}

	logger.Debug("opencode iteration completed successfully")
	return nil
}

// parseEvent parses a JSON event line and dispatches to appropriate callback.
func (r *Runner) parseEvent(line string) {
	var event struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}

	if err := json.Unmarshal([]byte(line), &event); err != nil {
		logger.Warn("Failed to parse event JSON: %v", err)
		return
	}

	switch event.Type {
	case "text":
		var text string
		if err := json.Unmarshal(event.Content, &text); err != nil {
			logger.Warn("Failed to parse text content: %v", err)
			return
		}
		if r.onText != nil {
			r.onText(text)
		}

	case "tool_use":
		var tu struct {
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		}
		if err := json.Unmarshal(event.Content, &tu); err != nil {
			logger.Warn("Failed to parse tool_use content: %v", err)
			return
		}
		if r.onToolUse != nil {
			r.onToolUse(tu.Name, tu.Input)
		}

	case "error":
		var errMsg string
		if err := json.Unmarshal(event.Content, &errMsg); err != nil {
			logger.Warn("Failed to parse error content: %v", err)
			return
		}
		if r.onError != nil {
			r.onError(fmt.Errorf("%s", errMsg))
		}

	default:
		logger.Debug("Unknown event type: %s", event.Type)
	}
}
