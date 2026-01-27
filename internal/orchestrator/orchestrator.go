package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/agent"
	ierr "github.com/mark3labs/iteratr/internal/errors"
	"github.com/mark3labs/iteratr/internal/hooks"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/template"
	"github.com/mark3labs/iteratr/internal/tui"
	natsserver "github.com/nats-io/nats-server/v2/server"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config holds configuration for the orchestrator.
type Config struct {
	SessionName       string // Name of the session
	SpecPath          string // Path to spec file
	TemplatePath      string // Path to custom template (optional)
	ExtraInstructions string // Extra instructions (optional)
	Iterations        int    // Max iterations (0 = infinite)
	DataDir           string // Data directory for persistent storage
	WorkDir           string // Working directory for agent
	Headless          bool   // Run without TUI
	Model             string // Model to use (e.g., anthropic/claude-sonnet-4-5)
	Reset             bool   // Reset session data before starting
	AutoCommit        bool   // Auto-commit modified files after iteration
}

// Orchestrator manages the iteration loop with embedded NATS, agent runner, and TUI.
type Orchestrator struct {
	cfg         Config
	ns          *natsserver.Server // Embedded NATS server (nil if node mode)
	natsPort    int                // NATS server port
	nc          *natsgo.Conn       // NATS connection
	store       *session.Store     // Session store
	runner      *agent.Runner      // Agent runner for opencode subprocess
	tuiApp      *tui.App           // TUI application (nil if headless)
	tuiProgram  *tea.Program       // Bubbletea program
	tuiDone     chan struct{}      // TUI completion signal
	sendChan    chan string        // Channel for user input messages from TUI to orchestrator
	ctx         context.Context    // Context for cancellation
	cancel      context.CancelFunc // Cancel function
	stopped     bool               // Track if Stop() was already called
	isPrimary   bool               // True if this instance owns the NATS server
	hooksConfig *hooks.Config      // Hooks configuration (nil if no hooks file)
	fileTracker *agent.FileTracker // Tracks files modified during iteration
	autoCommit  bool               // Auto-commit modified files after iteration
}

// New creates a new Orchestrator with the given configuration.
func New(cfg Config) (*Orchestrator, error) {
	// Set defaults
	if cfg.DataDir == "" {
		cfg.DataDir = ".iteratr"
	}
	if cfg.WorkDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		cfg.WorkDir = wd
	}

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	return &Orchestrator{
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
		tuiDone:     make(chan struct{}),
		sendChan:    make(chan string, 10), // Buffered channel for user input messages
		fileTracker: agent.NewFileTracker(cfg.WorkDir),
		autoCommit:  cfg.AutoCommit,
	}, nil
}

// Start initializes all components and starts the orchestrator.
func (o *Orchestrator) Start() error {
	logger.Info("Starting orchestrator for session '%s'", o.cfg.SessionName)

	// 1. Connect to existing NATS server or start a new one
	logger.Debug("Ensuring NATS connection")
	if err := o.ensureNATS(); err != nil {
		logger.Error("Failed to ensure NATS: %v", err)
		return fmt.Errorf("failed to ensure NATS: %w", err)
	}
	if o.isPrimary {
		logger.Debug("Running as primary (owns NATS server)")
	} else {
		logger.Debug("Running as node (connected to existing server)")
	}

	// 3. Setup JetStream stream
	logger.Debug("Setting up JetStream")
	if err := o.setupJetStream(); err != nil {
		logger.Error("Failed to setup JetStream: %v", err)
		return fmt.Errorf("failed to setup JetStream: %w", err)
	}
	logger.Debug("JetStream setup complete")

	// 3.5. Reset session data if requested
	if o.cfg.Reset {
		logger.Info("Resetting session data for '%s'", o.cfg.SessionName)
		if err := o.store.ResetSession(o.ctx, o.cfg.SessionName); err != nil {
			logger.Error("Failed to reset session: %v", err)
			return fmt.Errorf("failed to reset session: %w", err)
		}
		logger.Info("Session '%s' reset successfully", o.cfg.SessionName)
		fmt.Printf("Session '%s' reset successfully.\n", o.cfg.SessionName)
	}

	// 4. Check if session is already complete (before TUI starts)
	logger.Debug("Checking session state")
	state, err := o.store.LoadState(o.ctx, o.cfg.SessionName)
	if err != nil {
		logger.Error("Failed to load session state: %v", err)
		return fmt.Errorf("failed to load session state: %w", err)
	}

	if state.Complete {
		logger.Info("Session '%s' is already marked as complete", o.cfg.SessionName)
		fmt.Printf("Session '%s' is already marked as complete.\n", o.cfg.SessionName)
		fmt.Print("Do you want to restart it? [y/N]: ")

		var response string
		_, _ = fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			fmt.Println("Session not restarted.")
			return fmt.Errorf("session already complete")
		}

		// Restart the session
		if err := o.store.SessionRestart(o.ctx, o.cfg.SessionName); err != nil {
			logger.Error("Failed to restart session: %v", err)
			return fmt.Errorf("failed to restart session: %w", err)
		}
		logger.Info("Session '%s' restarted", o.cfg.SessionName)
		fmt.Println("Session restarted.")
	}

	// 5. Create agent runner (don't start yet - will start in Run())
	logger.Debug("Creating agent runner")
	// Runner will be initialized in Run() with proper callbacks after TUI is ready

	// 6. Start TUI if not headless
	if !o.cfg.Headless {
		logger.Debug("Starting TUI")
		if err := o.startTUI(); err != nil {
			logger.Error("Failed to start TUI: %v", err)
			return fmt.Errorf("failed to start TUI: %w", err)
		}
		logger.Debug("TUI started")
	} else {
		logger.Info("Running in headless mode")
	}

	// 7. Load hooks configuration (optional)
	logger.Debug("Loading hooks configuration")
	hooksConfig, err := hooks.LoadConfig(o.cfg.WorkDir)
	if err != nil {
		// Log warning but continue - hooks are optional
		logger.Warn("Failed to load hooks config: %v", err)
	} else if hooksConfig != nil {
		o.hooksConfig = hooksConfig
		logger.Info("Hooks configuration loaded")
	}

	logger.Info("Orchestrator started successfully")
	return nil
}

// Run executes the main iteration loop.
func (o *Orchestrator) Run() error {
	logger.Info("Starting iteration loop for session '%s'", o.cfg.SessionName)

	// Load current session state to determine starting iteration
	logger.Debug("Loading session state")
	state, err := o.store.LoadState(o.ctx, o.cfg.SessionName)
	if err != nil {
		logger.Error("Failed to load session state: %v", err)
		return fmt.Errorf("failed to load session state: %w", err)
	}

	// Determine starting iteration number
	startIteration := len(state.Iterations) + 1
	logger.Debug("Starting from iteration %d (found %d previous iterations)", startIteration, len(state.Iterations))

	// Check if session was marked complete (e.g., by external process or previous run)
	// Note: The interactive restart prompt happens in Start(), this is just a safety check
	if state.Complete {
		logger.Info("Session '%s' is already marked as complete", o.cfg.SessionName)
		return nil
	}

	// Print session info in headless mode
	if o.cfg.Headless {
		// Count tasks by status
		remainingCount := 0
		completedCount := 0
		for _, task := range state.Tasks {
			switch task.Status {
			case "remaining":
				remainingCount++
			case "completed":
				completedCount++
			}
		}

		fmt.Printf("=== Session: %s ===\n", o.cfg.SessionName)
		fmt.Printf("Starting at iteration #%d\n", startIteration)
		if o.cfg.Iterations > 0 {
			fmt.Printf("Max iterations: %d\n", o.cfg.Iterations)
		} else {
			fmt.Println("Max iterations: unlimited")
		}
		fmt.Printf("Tasks: %d remaining, %d completed\n\n", remainingCount, completedCount)
	}

	// Setup runner with callbacks based on headless mode
	logger.Debug("Setting up agent runner with callbacks")
	if o.tuiProgram != nil {
		// TUI mode - send output to TUI
		o.runner = agent.NewRunner(agent.RunnerConfig{
			Model:       o.cfg.Model,
			WorkDir:     o.cfg.WorkDir,
			SessionName: o.cfg.SessionName,
			NATSPort:    o.natsPort,
			OnText: func(content string) {
				o.tuiProgram.Send(tui.AgentOutputMsg{Content: content})
			},
			OnToolCall: func(event agent.ToolCallEvent) {
				msg := tui.AgentToolCallMsg{
					ToolCallID: event.ToolCallID,
					Title:      event.Title,
					Status:     event.Status,
					Kind:       event.Kind,
					Input:      event.RawInput,
					Output:     event.Output,
				}
				if event.FileDiff != nil {
					msg.FileDiff = &tui.FileDiff{
						File:      event.FileDiff.File,
						Before:    event.FileDiff.Before,
						After:     event.FileDiff.After,
						Additions: event.FileDiff.Additions,
						Deletions: event.FileDiff.Deletions,
					}
				}
				o.tuiProgram.Send(msg)
			},
			OnThinking: func(content string) {
				o.tuiProgram.Send(tui.AgentThinkingMsg{Content: content})
			},
			OnFinish: func(event agent.FinishEvent) {
				o.tuiProgram.Send(tui.AgentFinishMsg{
					Reason:   event.StopReason,
					Error:    event.Error,
					Model:    event.Model,
					Provider: event.Provider,
					Duration: event.Duration,
				})
			},
			OnFileChange: func(change agent.FileChange) {
				// Record change in tracker
				o.fileTracker.RecordChange(change.AbsPath, change.IsNew, change.Additions, change.Deletions)
			},
		})
	} else {
		// Headless mode - print to stdout
		o.runner = agent.NewRunner(agent.RunnerConfig{
			Model:       o.cfg.Model,
			WorkDir:     o.cfg.WorkDir,
			SessionName: o.cfg.SessionName,
			NATSPort:    o.natsPort,
			OnText: func(content string) {
				fmt.Print(content)
			},
			OnToolCall: func(event agent.ToolCallEvent) {
				// Simple tool lifecycle output for headless mode
				switch event.Status {
				case "pending":
					fmt.Printf("\n[tool: %s] ...\n", event.Title)
				case "in_progress":
					if cmd, ok := event.RawInput["command"].(string); ok {
						fmt.Printf("[tool: %s] command: %s\n", event.Title, cmd)
					}
				case "completed":
					outputLines := len(event.Output)
					if outputLines > 0 {
						fmt.Printf("[tool: %s] ✓ (output: %d bytes)\n", event.Title, len(event.Output))
					} else {
						fmt.Printf("[tool: %s] ✓\n", event.Title)
					}
				}
			},
			OnThinking: func(content string) {
				// Print thinking content dimmed in headless mode
				fmt.Printf("\033[2m%s\033[0m", content)
			},
			OnFinish: func(event agent.FinishEvent) {
				// Print finish summary in headless mode
				fmt.Printf("\n--- Agent finished: %s", event.StopReason)
				if event.Error != "" {
					fmt.Printf(" (error: %s)", event.Error)
				}
				fmt.Printf(" | Duration: %s", event.Duration.Round(time.Millisecond))
				if event.Model != "" {
					fmt.Printf(" | Model: %s", event.Model)
				}
				fmt.Println(" ---")
			},
			OnFileChange: func(change agent.FileChange) {
				// Record change in tracker
				o.fileTracker.RecordChange(change.AbsPath, change.IsNew, change.Additions, change.Deletions)
			},
		})
	}

	// Start the persistent ACP session
	logger.Debug("Starting persistent ACP session")
	if err := o.runner.Start(o.ctx); err != nil {
		logger.Error("Failed to start ACP session: %v", err)
		return fmt.Errorf("failed to start ACP session: %w", err)
	}
	logger.Debug("ACP session started successfully")
	// Ensure runner is stopped on exit
	defer o.runner.Stop()

	// Run iteration loop
	iterationCount := 0
	for {
		// Check for context cancellation (TUI quit, signal, etc.)
		select {
		case <-o.ctx.Done():
			logger.Info("Context cancelled, stopping iteration loop")
			return nil
		default:
		}

		currentIteration := startIteration + iterationCount

		// Check iteration limit (0 = infinite)
		if o.cfg.Iterations > 0 && iterationCount >= o.cfg.Iterations {
			logger.Info("Reached iteration limit of %d", o.cfg.Iterations)
			fmt.Printf("Reached iteration limit of %d\n", o.cfg.Iterations)
			break
		}

		logger.Info("=== Starting iteration #%d ===", currentIteration)

		// Clear file tracker for new iteration
		o.fileTracker.Clear()
		logger.Debug("File tracker cleared for iteration #%d", currentIteration)

		// Log iteration start
		if err := o.store.IterationStart(o.ctx, o.cfg.SessionName, currentIteration); err != nil {
			logger.Error("Failed to log iteration start: %v", err)
			return fmt.Errorf("failed to log iteration start: %w", err)
		}

		// Send iteration start message to TUI
		if o.tuiProgram != nil {
			o.tuiProgram.Send(tui.IterationStartMsg{Number: currentIteration})
		}

		// Execute pre-iteration hooks if configured
		var hookOutput string
		if o.hooksConfig != nil && len(o.hooksConfig.Hooks.PreIteration) > 0 {
			logger.Debug("Executing %d pre-iteration hook(s)", len(o.hooksConfig.Hooks.PreIteration))
			hookVars := hooks.Variables{
				Session:   o.cfg.SessionName,
				Iteration: strconv.Itoa(currentIteration),
			}
			output, err := hooks.ExecuteAll(o.ctx, o.hooksConfig.Hooks.PreIteration, o.cfg.WorkDir, hookVars)
			if err != nil {
				// Context cancelled - propagate
				if o.ctx.Err() != nil {
					logger.Info("Context cancelled during hook execution")
					return nil
				}
				logger.Error("Hook execution failed: %v", err)
			} else {
				hookOutput = output
				if len(output) > 0 {
					logger.Debug("Hook output: %d bytes", len(output))
				}
			}
		}

		// Build prompt with current state
		logger.Debug("Building prompt for iteration #%d", currentIteration)
		prompt, err := template.BuildPrompt(o.ctx, template.BuildConfig{
			SessionName:       o.cfg.SessionName,
			Store:             o.store,
			IterationNumber:   currentIteration,
			SpecPath:          o.cfg.SpecPath,
			TemplatePath:      o.cfg.TemplatePath,
			ExtraInstructions: o.cfg.ExtraInstructions,
			NATSPort:          o.natsPort,
		})
		if err != nil {
			logger.Error("Failed to build prompt: %v", err)
			return fmt.Errorf("failed to build prompt: %w", err)
		}
		logger.Debug("Prompt built, length: %d characters", len(prompt))

		// Run agent iteration with panic recovery (reusing persistent ACP session)
		// Hook output is sent as a separate content block before the main prompt
		logger.Info("Running agent for iteration #%d", currentIteration)
		err = ierr.Recover(func() error {
			return o.runner.RunIteration(o.ctx, prompt, hookOutput)
		})
		if err != nil {
			// Check if context was cancelled (TUI quit, signal, etc.) - exit gracefully
			if o.ctx.Err() != nil {
				logger.Info("Context cancelled during iteration, stopping gracefully")
				return nil
			}

			// Log the error (don't write to stderr - corrupts terminal during TUI shutdown)
			logger.Error("Iteration #%d failed: %v", currentIteration, err)

			// Check if it's a panic error - these are critical
			var panicErr *ierr.PanicError
			if errors.As(err, &panicErr) {
				logger.Error("Iteration #%d panicked with stack trace: %s", currentIteration, panicErr.StackTrace)
				return fmt.Errorf("iteration #%d panicked: %w", currentIteration, err)
			}

			// For other errors, return immediately
			return fmt.Errorf("iteration #%d failed: %w", currentIteration, err)
		}
		logger.Info("Iteration #%d agent execution completed", currentIteration)

		// Log iteration complete
		if err := o.store.IterationComplete(o.ctx, o.cfg.SessionName, currentIteration); err != nil {
			logger.Error("Failed to log iteration complete: %v", err)
			return fmt.Errorf("failed to log iteration complete: %w", err)
		}

		logger.Info("=== Iteration #%d completed successfully ===", currentIteration)

		// Print completion message in headless mode
		if o.cfg.Headless {
			fmt.Printf("\n✓ Iteration #%d complete\n\n", currentIteration)
		}

		// Check if session_complete was signaled by checking session state
		state, err := o.store.LoadState(o.ctx, o.cfg.SessionName)
		if err != nil {
			logger.Error("Failed to load session state: %v", err)
			return fmt.Errorf("failed to load session state: %w", err)
		}
		if state.Complete {
			logger.Info("Session '%s' marked as complete by agent", o.cfg.SessionName)
			// Send completion message to TUI to show dialog
			if o.tuiProgram != nil {
				o.tuiProgram.Send(tui.SessionCompleteMsg{})
				// Wait for TUI to quit (user dismisses dialog)
				<-o.tuiDone
			}
			break
		}

		// After iteration completes, process ALL queued user messages
		if err := o.processUserMessages(); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("Context cancelled while processing user messages")
				return nil
			}
			return err
		}

		iterationCount++
	}

	logger.Info("Iteration loop finished for session '%s'", o.cfg.SessionName)
	return nil
}

// processUserMessages drains sendChan and sends all queued messages as a single ACP request.
// Each message becomes a separate content block, but appears as separate messages in the TUI.
// Called after each agent response (iteration or user message).
// Returns when channel is empty and all messages processed.
func (o *Orchestrator) processUserMessages() error {
	// Collect all queued messages
	var messages []string
	for {
		select {
		case <-o.ctx.Done():
			return o.ctx.Err()
		case userMsg := <-o.sendChan:
			messages = append(messages, userMsg)
		default:
			// Channel empty, done collecting
			goto send
		}
	}

send:
	if len(messages) == 0 {
		return nil
	}

	logger.Info("Processing %d queued user message(s)", len(messages))

	// Notify TUI for each message (so they appear as separate messages in UI)
	if o.tuiProgram != nil {
		for _, msg := range messages {
			o.tuiProgram.Send(tui.QueuedMessageProcessingMsg{Text: msg})
		}
	}

	// Send all messages as separate content blocks in a single ACP request
	if err := o.runner.SendMessages(o.ctx, messages); err != nil {
		logger.Error("Failed to send user messages: %v", err)
		if o.tuiProgram != nil {
			o.tuiProgram.Send(tui.AgentOutputMsg{
				Content: fmt.Sprintf("\n[Error sending messages: %v]\n", err),
			})
		}
		return nil // Don't fail the iteration loop
	}

	return nil
}

// Stop gracefully shuts down all components.
// It collects errors from each component and returns a combined error if any fail.
// Multiple calls to Stop() are safe and idempotent.
func (o *Orchestrator) Stop() error {
	// Make Stop() idempotent - only run once
	if o.stopped {
		return nil
	}
	o.stopped = true

	logger.Info("Stopping orchestrator for session '%s'", o.cfg.SessionName)

	// Use MultiError to collect all shutdown errors
	multiErr := &ierr.MultiError{}

	// Cancel context to signal all goroutines to stop
	if o.cancel != nil {
		o.cancel()
	}

	// Wait for TUI to finish (context cancellation signals Bubbletea to shutdown)
	if o.tuiProgram != nil {
		logger.Debug("Waiting for TUI to finish")
		select {
		case <-o.tuiDone:
			logger.Debug("TUI stopped successfully")
		case <-time.After(2 * time.Second):
			// TUI didn't finish in time, force quit and continue
			logger.Warn("TUI shutdown timed out after 2s, forcing quit")
			o.tuiProgram.Quit()
			multiErr.Append(ierr.NewTransientError("TUI shutdown", fmt.Errorf("timed out after 2s")))
		}
		o.tuiProgram = nil
	}

	// Stop agent runner (closes ACP connection and subprocess)
	if o.runner != nil {
		logger.Debug("Stopping agent runner")
		o.runner.Stop()
		o.runner = nil
	}

	// Close NATS connection (and server if primary)
	if o.isPrimary {
		// Primary mode: shut down the server we own
		logger.Debug("Shutting down NATS server (primary mode)")
		if err := nats.Shutdown(o.nc, o.ns); err != nil {
			logger.Error("NATS shutdown failed: %v", err)
			multiErr.Append(fmt.Errorf("NATS shutdown failed: %w", err))
		} else {
			logger.Debug("NATS shut down successfully")
		}
	} else {
		// Node mode: just close the connection, don't kill the server
		logger.Debug("Closing NATS connection (node mode)")
		if o.nc != nil {
			o.nc.Close()
		}
	}

	// Clear references
	o.nc = nil
	o.ns = nil

	logger.Info("Orchestrator stopped")

	// Return combined errors if any
	return multiErr.ErrorOrNil()
}

// ensureNATS connects to an existing NATS server or starts a new one.
// If another iteratr instance is already running with a NATS server,
// this instance runs in "node mode" and connects to the existing server.
// Otherwise, it starts a new embedded server and runs in "primary mode".
func (o *Orchestrator) ensureNATS() error {
	// Ensure data directory exists
	dataDir := filepath.Join(o.cfg.DataDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Try to connect to existing server first
	if nc := nats.TryConnectExisting(dataDir); nc != nil {
		logger.Info("Connected to existing NATS server (node mode)")
		o.nc = nc
		o.isPrimary = false
		return nil
	}

	// No server running, start one (primary mode)
	logger.Info("Starting NATS server (primary mode)")
	ns, port, err := nats.StartEmbeddedNATS(dataDir)
	if err != nil {
		return fmt.Errorf("failed to start NATS server: %w", err)
	}
	o.ns = ns
	o.natsPort = port
	o.isPrimary = true

	// Connect to our own server
	nc, err := nats.ConnectToPort(port)
	if err != nil {
		// Failed to connect to server we just started - shut it down
		ns.Shutdown()
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	o.nc = nc
	return nil
}

// setupJetStream creates the JetStream stream and initializes the session store.
func (o *Orchestrator) setupJetStream() error {
	// Create JetStream context using modern API
	js, err := jetstream.New(o.nc)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Setup stream
	stream, err := nats.SetupStream(o.ctx, js)
	if err != nil {
		return fmt.Errorf("failed to setup stream: %w", err)
	}

	// Create session store
	o.store = session.NewStore(js, stream)
	return nil
}

// startTUI initializes and starts the Bubbletea TUI.
func (o *Orchestrator) startTUI() error {
	// Create TUI app
	o.tuiApp = tui.NewApp(o.ctx, o.store, o.cfg.SessionName, o.nc, o.sendChan)

	// Create Bubbletea program with context for graceful shutdown
	o.tuiProgram = tea.NewProgram(o.tuiApp, tea.WithContext(o.ctx))

	// Start TUI in background with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Log panic but don't write to stderr - corrupts terminal during shutdown
				logger.Error("TUI panic: %v", r)
			}
			// Signal TUI is done
			close(o.tuiDone)
		}()

		if _, err := o.tuiProgram.Run(); err != nil {
			// Ignore expected shutdown errors (context cancelled, user interrupt)
			if o.ctx.Err() == nil && !errors.Is(err, tea.ErrInterrupted) {
				logger.Error("TUI error: %v", err)
			}
		}
	}()

	// Monitor TUI quit and cancel orchestrator context
	go func() {
		<-o.tuiDone
		logger.Debug("TUI quit detected, cancelling orchestrator context")
		if o.cancel != nil {
			o.cancel()
		}
	}()

	return nil
}

// isGitRepo checks if the given directory is inside a git repository.
// Returns true if a .git directory exists in the given path or any parent directory.
func isGitRepo(dir string) bool {
	// Walk up the directory tree looking for .git
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	current := absDir
	for {
		gitPath := filepath.Join(current, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return true
		}

		// Move up to parent directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root directory, no .git found
			return false
		}
		current = parent
	}
}
