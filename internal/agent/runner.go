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

	// Monitor context cancellation and ensure process cleanup
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			logger.Debug("Context cancelled, killing opencode process")
			// Kill the process group to ensure all children are terminated
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		case <-done:
			// Normal completion
		}
	}()

	// Send prompt to stdin
	logger.Debug("Sending prompt to opencode (length: %d)", len(prompt))
	if _, err := io.WriteString(stdin, prompt); err != nil {
		close(done)
		logger.Error("Failed to write prompt: %v", err)
		return fmt.Errorf("failed to write prompt: %w", err)
	}
	stdin.Close()

	// Parse JSON events from stdout using bufio.Reader (no line length limit)
	logger.Debug("Parsing JSON events from opencode")
	reader := bufio.NewReader(stdout)
	for {
		// Check context between reads
		if ctx.Err() != nil {
			logger.Debug("Context cancelled during output parsing")
			break
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Process any remaining data before EOF
				if line != "" {
					r.parseEvent(line)
				}
				break
			}
			logger.Error("Read error: %v", err)
			close(done)
			return fmt.Errorf("failed to read output: %w", err)
		}

		// Trim newline and skip empty lines
		line = line[:len(line)-1] // Remove trailing \n
		if line == "" {
			continue
		}
		r.parseEvent(line)
	}

	// Signal the monitor goroutine we're done
	close(done)

	// Check for context cancellation first
	if ctx.Err() != nil {
		// Wait briefly for process to die, then return
		cmd.Wait()
		return ctx.Err()
	}

	// Wait for process to complete
	logger.Debug("Waiting for opencode process to exit")
	if err := cmd.Wait(); err != nil {
		// If context was cancelled, don't treat as error
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("opencode exited with error: %v", err)
		return fmt.Errorf("opencode failed: %w", err)
	}

	logger.Debug("opencode iteration completed successfully")
	return nil
}

// parseEvent parses a JSON event line and dispatches to appropriate callback.
// Event format from opencode --format json:
//
//	{"type":"text","timestamp":...,"sessionID":"...","part":{"type":"text","text":"..."}}
//	{"type":"tool_use","timestamp":...,"sessionID":"...","part":{"type":"tool","tool":"...","state":{...}}}
//	{"type":"error","timestamp":...,"sessionID":"...","error":{"name":"...","data":{...}}}
func (r *Runner) parseEvent(line string) {
	var event struct {
		Type string `json:"type"`
		Part *struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			Tool  string         `json:"tool"`
			State map[string]any `json:"state"`
		} `json:"part"`
		Error *struct {
			Name string `json:"name"`
			Data *struct {
				Message string `json:"message"`
			} `json:"data"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(line), &event); err != nil {
		logger.Warn("Failed to parse event JSON: %v", err)
		return
	}

	switch event.Type {
	case "text":
		if event.Part != nil && event.Part.Text != "" {
			if r.onText != nil {
				r.onText(event.Part.Text)
			}
		}

	case "tool_use":
		if event.Part != nil && event.Part.Tool != "" {
			if r.onToolUse != nil {
				var input map[string]any
				if event.Part.State != nil {
					if i, ok := event.Part.State["input"].(map[string]any); ok {
						input = i
					}
				}
				r.onToolUse(event.Part.Tool, input)
			}
		}

	case "error":
		if event.Error != nil {
			errMsg := event.Error.Name
			if event.Error.Data != nil && event.Error.Data.Message != "" {
				errMsg = event.Error.Data.Message
			}
			if r.onError != nil {
				r.onError(fmt.Errorf("%s", errMsg))
			}
		}

	case "step_start", "step_finish":
		// Informational events - no action needed
		logger.Debug("Step event: %s", event.Type)

	default:
		logger.Debug("Unknown event type: %s", event.Type)
	}
}
