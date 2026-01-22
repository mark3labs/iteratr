package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/agent"
	ierr "github.com/mark3labs/iteratr/internal/errors"
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
	DataDir           string // Data directory for NATS storage
	WorkDir           string // Working directory for agent
	Headless          bool   // Run without TUI
	Model             string // Model to use (e.g., anthropic/claude-sonnet-4-5)
}

// Orchestrator manages the iteration loop with embedded NATS, agent runner, and TUI.
type Orchestrator struct {
	cfg        Config
	ns         *natsserver.Server // Embedded NATS server
	natsPort   int                // NATS server port
	nc         *natsgo.Conn       // NATS connection
	store      *session.Store     // Session store
	runner     *agent.Runner      // Agent runner for opencode subprocess
	tuiApp     *tui.App           // TUI application (nil if headless)
	tuiProgram *tea.Program       // Bubbletea program
	tuiDone    chan struct{}      // TUI completion signal
	ctx        context.Context    // Context for cancellation
	cancel     context.CancelFunc // Cancel function
	stopped    bool               // Track if Stop() was already called
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
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
		tuiDone: make(chan struct{}),
	}, nil
}

// Start initializes all components and starts the orchestrator.
func (o *Orchestrator) Start() error {
	logger.Info("Starting orchestrator for session '%s'", o.cfg.SessionName)

	// 1. Start embedded NATS server
	logger.Debug("Starting embedded NATS server")
	if err := o.startNATS(); err != nil {
		logger.Error("Failed to start NATS: %v", err)
		return fmt.Errorf("failed to start NATS: %w", err)
	}
	logger.Debug("NATS server started successfully")

	// 2. Connect to NATS in-process
	logger.Debug("Connecting to NATS in-process")
	if err := o.connectNATS(); err != nil {
		logger.Error("Failed to connect to NATS: %v", err)
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	logger.Debug("Connected to NATS")

	// 3. Setup JetStream stream
	logger.Debug("Setting up JetStream")
	if err := o.setupJetStream(); err != nil {
		logger.Error("Failed to setup JetStream: %v", err)
		return fmt.Errorf("failed to setup JetStream: %w", err)
	}
	logger.Debug("JetStream setup complete")

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
		fmt.Scanln(&response)

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

	// 5. Create agent runner
	logger.Debug("Creating agent runner")
	o.runner = agent.NewRunner(agent.RunnerConfig{
		Model:       o.cfg.Model,
		WorkDir:     o.cfg.WorkDir,
		SessionName: o.cfg.SessionName,
		NATSPort:    o.natsPort,
		OnText:      nil, // Set later based on headless mode
		OnToolUse:   nil, // Not needed - tools called via CLI
		OnError:     nil, // Errors returned from RunIteration
	})

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

	// Run iteration loop
	iterationCount := 0
	for {
		currentIteration := startIteration + iterationCount

		// Check iteration limit (0 = infinite)
		if o.cfg.Iterations > 0 && iterationCount >= o.cfg.Iterations {
			logger.Info("Reached iteration limit of %d", o.cfg.Iterations)
			fmt.Printf("Reached iteration limit of %d\n", o.cfg.Iterations)
			break
		}

		logger.Info("=== Starting iteration #%d ===", currentIteration)

		// Log iteration start
		if err := o.store.IterationStart(o.ctx, o.cfg.SessionName, currentIteration); err != nil {
			logger.Error("Failed to log iteration start: %v", err)
			return fmt.Errorf("failed to log iteration start: %w", err)
		}

		// Send iteration start message to TUI
		if o.tuiProgram != nil {
			o.tuiProgram.Send(tui.IterationStartMsg{Number: currentIteration})
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

		// Setup output callback for runner
		if o.tuiProgram != nil {
			// Send to TUI
			o.runner = agent.NewRunner(agent.RunnerConfig{
				Model:       o.cfg.Model,
				WorkDir:     o.cfg.WorkDir,
				SessionName: o.cfg.SessionName,
				NATSPort:    o.natsPort,
				OnText: func(content string) {
					o.tuiProgram.Send(tui.AgentOutputMsg{Content: content})
				},
				OnToolUse: func(name string, input map[string]any) {
					o.tuiProgram.Send(tui.AgentToolMsg{Tool: name, Input: input})
				},
				OnError: nil, // Errors returned from RunIteration
			})
		} else {
			// Print to stdout in headless mode
			o.runner = agent.NewRunner(agent.RunnerConfig{
				Model:       o.cfg.Model,
				WorkDir:     o.cfg.WorkDir,
				SessionName: o.cfg.SessionName,
				NATSPort:    o.natsPort,
				OnText: func(content string) {
					fmt.Print(content)
				},
				OnToolUse: func(name string, input map[string]any) {
					fmt.Printf("\n[tool: %s]\n", name)
				},
				OnError: nil, // Errors returned from RunIteration
			})
		}

		// Run agent iteration with panic recovery
		logger.Info("Running agent for iteration #%d", currentIteration)
		fmt.Printf("Running iteration #%d...\n", currentIteration)
		err = ierr.Recover(func() error {
			return o.runner.RunIteration(o.ctx, prompt)
		})
		if err != nil {
			// Log the error but continue with graceful handling
			logger.Error("Iteration #%d failed: %v", currentIteration, err)
			fmt.Fprintf(os.Stderr, "Iteration #%d failed: %v\n", currentIteration, err)

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
			fmt.Printf("Session '%s' marked as complete by agent\n", o.cfg.SessionName)
			break
		}

		iterationCount++
	}

	logger.Info("Iteration loop finished for session '%s'", o.cfg.SessionName)
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

	// Stop TUI and wait for it to finish
	if o.tuiProgram != nil {
		logger.Debug("Stopping TUI")
		o.tuiProgram.Quit()
		// Wait for TUI to finish with timeout
		select {
		case <-o.tuiDone:
			logger.Debug("TUI stopped successfully")
		case <-time.After(2 * time.Second):
			// TUI didn't finish in time, continue with shutdown
			logger.Warn("TUI shutdown timed out after 2s")
			multiErr.Append(ierr.NewTransientError("TUI shutdown", fmt.Errorf("timed out after 2s")))
		}
		o.tuiProgram = nil
	}

	// Agent runner cleanup is automatic - context cancellation will stop any running subprocess
	logger.Debug("Agent runner will be cleaned up via context cancellation")

	// Close NATS connection and server using helper
	logger.Debug("Shutting down NATS")
	if err := nats.Shutdown(o.nc, o.ns); err != nil {
		logger.Error("NATS shutdown failed: %v", err)
		multiErr.Append(fmt.Errorf("NATS shutdown failed: %w", err))
	} else {
		logger.Debug("NATS shut down successfully")
	}

	// Clear references
	o.nc = nil
	o.ns = nil

	logger.Info("Orchestrator stopped")

	// Return combined errors if any
	return multiErr.ErrorOrNil()
}

// startNATS starts the embedded NATS server.
func (o *Orchestrator) startNATS() error {
	// Ensure data directory exists
	dataDir := filepath.Join(o.cfg.DataDir, "nats")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create NATS data directory: %w", err)
	}

	// Use the shared nats package to start the server
	ns, port, err := nats.StartEmbeddedNATS(dataDir)
	if err != nil {
		return fmt.Errorf("failed to start NATS server: %w", err)
	}

	o.ns = ns
	o.natsPort = port
	return nil
}

// connectNATS creates a connection to the embedded NATS server.
func (o *Orchestrator) connectNATS() error {
	// Read the port from the port file
	dataDir := filepath.Join(o.cfg.DataDir, "nats")
	port, err := nats.ReadPort(dataDir)
	if err != nil {
		return fmt.Errorf("failed to read NATS port: %w", err)
	}

	// Connect via TCP
	nc, err := nats.ConnectToPort(port)
	if err != nil {
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
	o.tuiApp = tui.NewApp(o.ctx, o.store, o.cfg.SessionName, o.nc)

	// Create Bubbletea program
	o.tuiProgram = tea.NewProgram(o.tuiApp)

	// Start TUI in background with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "TUI panic: %v\n", r)
			}
			// Signal TUI is done
			close(o.tuiDone)
		}()

		if _, err := o.tuiProgram.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
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
