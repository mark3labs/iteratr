package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/agent"
	ierr "github.com/mark3labs/iteratr/internal/errors"
	"github.com/mark3labs/iteratr/internal/hooks"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/mcpserver"
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
	cfg               Config
	ns                *natsserver.Server // Embedded NATS server (nil if node mode)
	natsPort          int                // NATS server port
	nc                *natsgo.Conn       // NATS connection
	store             *session.Store     // Session store
	mcpServer         *mcpserver.Server  // MCP tools server
	runner            *agent.Runner      // Agent runner for opencode subprocess
	tuiApp            *tui.App           // TUI application (nil if headless)
	tuiProgram        *tea.Program       // Bubbletea program
	tuiDone           chan struct{}      // TUI completion signal
	sendChan          chan string        // Channel for user input messages from TUI to orchestrator
	ctx               context.Context    // Context for cancellation
	cancel            context.CancelFunc // Cancel function
	stopped           bool               // Track if Stop() was already called
	isPrimary         bool               // True if this instance owns the NATS server
	hooksConfig       *hooks.Config      // Hooks configuration (nil if no hooks file)
	fileTracker       *agent.FileTracker // Tracks files modified during iteration
	autoCommit        bool               // Auto-commit modified files after iteration
	pendingHookOutput string             // Buffer for hook output to be sent in next iteration
	pendingMu         sync.Mutex         // Protects pendingHookOutput (needed for NATS callback)
	paused            atomic.Bool        // Pause state (atomic for thread-safe access)
	resumeChan        chan struct{}      // Signals resume from pause
	hookCounter       atomic.Int64       // Counter for generating unique hook IDs
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
		resumeChan:  make(chan struct{}, 1), // Buffered to prevent blocking on Resume()
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

	// 3.25. Record model in session state (for resume default)
	if o.cfg.Model != "" {
		if err := o.store.SetSessionModel(o.ctx, o.cfg.SessionName, o.cfg.Model); err != nil {
			logger.Warn("Failed to record session model: %v", err)
			// Non-fatal - continue without model persistence
		}
	}

	// 3.5. Start MCP tools server
	logger.Debug("Starting MCP tools server")
	o.mcpServer = mcpserver.New(o.store, o.cfg.SessionName)
	port, err := o.mcpServer.Start(o.ctx)
	if err != nil {
		logger.Error("Failed to start MCP server: %v", err)
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	logger.Info("MCP tools server started on port %d", port)

	// 3.6. Reset session data if requested
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
	// Fresh sessions start at iteration 0 (planning phase); resumed sessions skip to next iteration
	startIteration := len(state.Iterations)
	if startIteration == 0 {
		logger.Debug("Fresh session, will run Iteration #0 (planning phase)")
	} else {
		startIteration++ // Resume at next iteration after last completed
	}
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
			Model:        o.cfg.Model,
			WorkDir:      o.cfg.WorkDir,
			SessionName:  o.cfg.SessionName,
			NATSPort:     o.natsPort,
			MCPServerURL: o.mcpServer.URL(),
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
					SessionID:  event.SessionID,
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
				// Send message to TUI
				o.tuiProgram.Send(tui.FileChangeMsg{
					Path:      change.Path,
					IsNew:     change.IsNew,
					Additions: change.Additions,
					Deletions: change.Deletions,
				})
			},
		})
	} else {
		// Headless mode - print to stdout
		o.runner = agent.NewRunner(agent.RunnerConfig{
			Model:        o.cfg.Model,
			WorkDir:      o.cfg.WorkDir,
			SessionName:  o.cfg.SessionName,
			NATSPort:     o.natsPort,
			MCPServerURL: o.mcpServer.URL(),
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
	defer func() {
		if o.runner != nil {
			o.runner.Stop()
		}
	}()

	// Run Iteration #0 (planning phase) for fresh sessions
	// No hooks are executed during iteration #0 — hooks are set up after.
	if startIteration == 0 {
		if err := o.runIteration0(); err != nil {
			if o.ctx.Err() != nil {
				logger.Info("Context cancelled during Iteration #0")
				return nil
			}
			return fmt.Errorf("iteration #0 failed: %w", err)
		}
		startIteration = 1 // Main loop starts at iteration 1
	}

	// Subscribe to task completion events for on_task_complete hooks
	// (after iteration #0 so hooks don't fire during planning phase)
	var taskCompleteSub *natsgo.Subscription
	if o.hooksConfig != nil && len(o.hooksConfig.Hooks.OnTaskComplete) > 0 {
		logger.Debug("Subscribing to task completion events for on_task_complete hooks")
		subject := fmt.Sprintf("iteratr.%s.task", o.cfg.SessionName)
		sub, err := o.nc.Subscribe(subject, func(msg *natsgo.Msg) {
			// Parse event to check if it's a status=completed action
			var event struct {
				Action string          `json:"action"`
				Meta   json.RawMessage `json:"meta"`
			}
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				logger.Warn("Failed to parse task event for on_task_complete: %v", err)
				return
			}

			// Only process status=completed events
			if event.Action != "status" {
				return
			}

			var meta struct {
				TaskID string `json:"task_id"`
				Status string `json:"status"`
			}
			if err := json.Unmarshal(event.Meta, &meta); err != nil {
				logger.Warn("Failed to parse task event metadata: %v", err)
				return
			}

			if meta.Status != "completed" {
				return
			}

			// Load current state to get task content
			state, err := o.store.LoadState(o.ctx, o.cfg.SessionName)
			if err != nil {
				logger.Warn("Failed to load state for on_task_complete: %v", err)
				return
			}

			task, exists := state.Tasks[meta.TaskID]
			if !exists {
				logger.Warn("Task %s not found in state for on_task_complete", meta.TaskID)
				return
			}

			// Execute on_task_complete hooks
			logger.Info("Task %s completed, executing on_task_complete hooks", meta.TaskID)
			hookVars := hooks.Variables{
				Session:     o.cfg.SessionName,
				TaskID:      meta.TaskID,
				TaskContent: task.Content,
			}
			onStart, onComplete, _ := o.hookCallbacks("on_task_complete")
			output, err := hooks.ExecuteAllPipedWithCallbacks(o.ctx, o.hooksConfig.Hooks.OnTaskComplete, o.cfg.WorkDir, hookVars, onStart, onComplete)
			if err != nil {
				// Context cancelled or error - just log
				if o.ctx.Err() != nil {
					logger.Debug("Context cancelled during on_task_complete hook execution")
				} else {
					logger.Error("on_task_complete hook execution failed: %v", err)
				}
				return
			}

			if output != "" {
				// Append piped output to pending buffer (FIFO order)
				logger.Debug("on_task_complete hook output: %d bytes (appending to pending buffer)", len(output))
				o.appendPendingOutput(output)
			}
		})
		if err != nil {
			logger.Warn("Failed to subscribe to task completion events: %v", err)
			// Don't fail - hooks are optional
		} else {
			taskCompleteSub = sub
			logger.Debug("Subscribed to task completion events")
		}
	}
	// Unsubscribe on exit
	if taskCompleteSub != nil {
		defer func() {
			if err := taskCompleteSub.Unsubscribe(); err != nil {
				logger.Debug("Failed to unsubscribe from task complete: %v", err)
			}
		}()
	}

	// Execute session_start hooks if configured (after iteration #0, before main loop)
	if o.hooksConfig != nil && len(o.hooksConfig.Hooks.SessionStart) > 0 {
		logger.Debug("Executing %d session_start hook(s)", len(o.hooksConfig.Hooks.SessionStart))
		hookVars := hooks.Variables{
			Session: o.cfg.SessionName,
		}
		onStart, onComplete, _ := o.hookCallbacks("session_start")
		output, err := hooks.ExecuteAllPipedWithCallbacks(o.ctx, o.hooksConfig.Hooks.SessionStart, o.cfg.WorkDir, hookVars, onStart, onComplete)
		if err != nil {
			// Context cancelled - propagate
			if o.ctx.Err() != nil {
				logger.Info("Context cancelled during session_start hook execution")
				return nil
			}
			logger.Error("Session_start hook execution failed: %v", err)
		} else if output != "" {
			// Store piped output in pending buffer for first iteration
			logger.Debug("Session_start hook output: %d bytes (storing in pending buffer)", len(output))
			o.appendPendingOutput(output)
		}
	}

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

		// Drain pending hook output from previous iterations (session_start, post_iteration, on_task_complete)
		pendingOutput := o.drainPendingOutput()
		if len(pendingOutput) > 0 {
			logger.Debug("Drained pending hook output: %d bytes", len(pendingOutput))
		}

		// Execute pre-iteration hooks if configured
		var hookOutput string
		if o.hooksConfig != nil && len(o.hooksConfig.Hooks.PreIteration) > 0 {
			logger.Debug("Executing %d pre-iteration hook(s)", len(o.hooksConfig.Hooks.PreIteration))
			hookVars := hooks.Variables{
				Session:   o.cfg.SessionName,
				Iteration: strconv.Itoa(currentIteration),
			}
			onStart, onComplete, _ := o.hookCallbacks("pre_iteration")
			output, err := hooks.ExecuteAllPipedWithCallbacks(o.ctx, o.hooksConfig.Hooks.PreIteration, o.cfg.WorkDir, hookVars, onStart, onComplete)
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
					logger.Debug("Pre-iteration hook output: %d bytes", len(output))
				}
			}
		}

		// Combine pending output and pre-iteration hook output
		// Pending output comes first (FIFO order from session_start, post_iteration, on_task_complete)
		if len(pendingOutput) > 0 {
			if len(hookOutput) > 0 {
				hookOutput = pendingOutput + "\n" + hookOutput
			} else {
				hookOutput = pendingOutput
			}
			logger.Debug("Combined hook output: %d bytes", len(hookOutput))
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
			}

			// Execute on_error hooks if configured
			if o.hooksConfig != nil && len(o.hooksConfig.Hooks.OnError) > 0 {
				logger.Info("Executing on_error hooks for iteration #%d", currentIteration)
				hookVars := hooks.Variables{
					Session:   o.cfg.SessionName,
					Iteration: strconv.Itoa(currentIteration),
					Error:     err.Error(),
				}
				onStart, onComplete, _ := o.hookCallbacks("on_error")
				hookOutput, hookErr := hooks.ExecuteAllPipedWithCallbacks(o.ctx, o.hooksConfig.Hooks.OnError, o.cfg.WorkDir, hookVars, onStart, onComplete)
				if hookErr != nil {
					// Context cancelled - propagate
					if o.ctx.Err() != nil {
						logger.Info("Context cancelled during on_error hook execution")
						return nil
					}
					logger.Error("on_error hook execution failed: %v", hookErr)
					// Continue despite hook failure
				}

				// If hooks produced piped output, send to agent for immediate recovery
				if hookOutput != "" {
					logger.Info("Sending on_error hook output to agent for recovery attempt")

					// Check session state to determine recovery instructions.
					// If the agent was mid-task (no summary yet or task still in_progress),
					// allow it to continue working. Otherwise tell it to just fix and stop.
					canContinue := false
					if errState, loadErr := o.store.LoadState(o.ctx, o.cfg.SessionName); loadErr == nil {
						// Check if current iteration has no summary yet
						hasSummary := false
						for _, iter := range errState.Iterations {
							if iter.Number == currentIteration && iter.Summary != "" {
								hasSummary = true
								break
							}
						}
						// Check if any task is still in_progress
						hasInProgress := false
						for _, task := range errState.Tasks {
							if task.Status == "in_progress" {
								hasInProgress = true
								break
							}
						}
						canContinue = !hasSummary || hasInProgress
					}

					var recoveryPrompt string
					if canContinue {
						recoveryPrompt = fmt.Sprintf(
							"[ON-ERROR HOOKS - iteration #%d]\n"+
								"The iteration encountered an error: %s\n\n"+
								"Diagnostic output from error hooks:\n%s\n\n"+
								"Fix the issue and continue completing your current task.",
							currentIteration, err.Error(), hookOutput,
						)
					} else {
						recoveryPrompt = fmt.Sprintf(
							"[ON-ERROR HOOKS - iteration #%d]\n"+
								"The iteration failed with error: %s\n\n"+
								"Diagnostic output from error hooks:\n%s\n\n"+
								"Fix the issue if possible, then STOP. Do NOT pick a new task.",
							currentIteration, err.Error(), hookOutput,
						)
					}

					// Send recovery prompt and wait for response
					recoveryErr := o.runner.SendMessages(o.ctx, []string{recoveryPrompt})
					if recoveryErr != nil {
						if o.ctx.Err() != nil {
							logger.Info("Context cancelled during recovery prompt")
							return nil
						}
						logger.Error("Failed to send recovery prompt to agent: %v", recoveryErr)
						// Continue to next iteration despite recovery failure
					} else {
						logger.Info("Recovery prompt sent successfully (canContinue=%v)", canContinue)
					}
				}

				// Continue to next iteration (don't exit session when hooks configured)
				logger.Info("Continuing to next iteration after error")
				continue
			}

			// No on_error hooks configured - return error (backward compatible)
			// Check if it's a panic error - these are critical
			if errors.As(err, &panicErr) {
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

		// Execute post-iteration hooks if configured
		if o.hooksConfig != nil && len(o.hooksConfig.Hooks.PostIteration) > 0 {
			logger.Debug("Executing %d post-iteration hook(s)", len(o.hooksConfig.Hooks.PostIteration))
			hookVars := hooks.Variables{
				Session:   o.cfg.SessionName,
				Iteration: strconv.Itoa(currentIteration),
			}
			onStart, onComplete, _ := o.hookCallbacks("post_iteration")
			output, err := hooks.ExecuteAllPipedWithCallbacks(o.ctx, o.hooksConfig.Hooks.PostIteration, o.cfg.WorkDir, hookVars, onStart, onComplete)
			if err != nil {
				// Context cancelled - propagate
				if o.ctx.Err() != nil {
					logger.Info("Context cancelled during post-iteration hook execution")
					return nil
				}
				logger.Error("Post-iteration hook execution failed: %v", err)
			} else if output != "" {
				// Send hook output to model with clear framing so the agent knows
				// this is post-iteration verification, not a new task prompt.
				logger.Debug("Post-iteration hook output: %d bytes (sending to model)", len(output))
				framedOutput := fmt.Sprintf(
					"[POST-ITERATION HOOKS - iteration #%d]\n"+
						"The following output is from post-iteration hooks (linting, vetting, etc.).\n"+
						"If there are errors or issues, fix them now. Do NOT pick a new task.\n"+
						"When done fixing (or if no issues), STOP immediately.\n\n%s",
					currentIteration, output,
				)
				if err := o.runner.SendMessages(o.ctx, []string{framedOutput}); err != nil {
					logger.Error("Failed to send post-iteration hook output to model: %v", err)
					// Continue anyway - don't fail the iteration
				}
			}
		}

		// Run auto-commit if enabled and files were modified
		if o.autoCommit && o.fileTracker.HasChanges() {
			logger.Info("Auto-commit enabled with %d modified files, running commit", o.fileTracker.Count())
			if err := o.runAutoCommit(o.ctx); err != nil {
				logger.Warn("Auto-commit failed: %v", err)
				// Don't fail the iteration - just log the warning
			}
		}

		// Print completion message in headless mode
		if o.cfg.Headless {
			fmt.Printf("\n✓ Iteration #%d complete\n\n", currentIteration)
		}

		// Check if session_complete was signaled by checking session state
		state, err = o.store.LoadState(o.ctx, o.cfg.SessionName)
		if err != nil {
			logger.Error("Failed to load session state: %v", err)
			return fmt.Errorf("failed to load session state: %w", err)
		}
		if state.Complete {
			logger.Info("Session '%s' marked as complete by agent", o.cfg.SessionName)
			// Send completion message to TUI to show dialog
			if o.tuiProgram != nil {
				o.tuiProgram.Send(tui.SessionCompleteMsg{})
			}
			// Continue processing user messages after completion
			// If agent restarts session, resume normal iteration
		postCompletionLoop:
			for {
				select {
				case <-o.tuiDone:
					break postCompletionLoop
				case <-o.ctx.Done():
					return nil
				case userMsg := <-o.sendChan:
					logger.Info("Processing user message after completion")
					if o.tuiProgram != nil {
						o.tuiProgram.Send(tui.QueuedMessageProcessingMsg{Text: userMsg})
					}
					if err := o.runner.SendMessages(o.ctx, []string{userMsg}); err != nil {
						logger.Error("Failed to send user message: %v", err)
					}
					// Reload state and refresh TUI (task list may have changed)
					state, err = o.store.LoadState(o.ctx, o.cfg.SessionName)
					if err == nil {
						if o.tuiProgram != nil {
							o.tuiProgram.Send(tui.StateUpdateMsg{State: state})
						}
						// Check if session was restarted (agent added tasks and marked incomplete)
						if !state.Complete {
							logger.Info("Session restarted, resuming iterations")
							break postCompletionLoop
						}
					}
				}
			}
			// Only exit main loop if session is still complete
			// If restarted, continue iterating
			state, err = o.store.LoadState(o.ctx, o.cfg.SessionName)
			if err != nil {
				logger.Error("Failed to load session state after post-completion processing: %v", err)
				break
			}
			if state.Complete {
				break
			}
		}

		// After iteration completes, process ALL queued user messages
		if err := o.processUserMessages(); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("Context cancelled while processing user messages")
				return nil
			}
			return err
		}

		// Check if paused - block until resumed or context cancelled
		if err := o.waitIfPaused(); err != nil {
			// Context cancelled during pause
			logger.Info("Context cancelled during pause, stopping iteration loop")
			return nil
		}

		iterationCount++
	}

	logger.Info("Iteration loop finished for session '%s'", o.cfg.SessionName)

	// Final delivery: if pending buffer has content, send to agent before session_end
	// This gives the agent a chance to address test failures discovered in final post_iteration
	if o.hasPendingOutput() {
		pendingOutput := o.drainPendingOutput()
		logger.Info("Final delivery: sending pending hook output to agent (%d bytes)", len(pendingOutput))

		// Send pending output to agent and wait for response
		if err := o.runner.SendMessages(o.ctx, []string{pendingOutput}); err != nil {
			// Check if context was cancelled
			if o.ctx.Err() != nil {
				logger.Info("Context cancelled during final delivery")
				return nil
			}
			logger.Error("Final delivery failed: %v", err)
			// Don't fail - continue to session_end hooks
		} else {
			logger.Info("Final delivery completed successfully")
		}
	}

	// Execute session_end hooks if configured
	// These run after final delivery. pipe_output is ignored - output is not piped anywhere (no more iterations)
	if o.hooksConfig != nil && len(o.hooksConfig.Hooks.SessionEnd) > 0 {
		logger.Info("Executing %d session_end hook(s)", len(o.hooksConfig.Hooks.SessionEnd))
		hookVars := hooks.Variables{
			Session: o.cfg.SessionName,
			// Iteration is not set for session_end hooks (session-level, not iteration-level)
		}
		onStart, onComplete, _ := o.hookCallbacks("session_end")
		_, err := hooks.ExecuteAllWithCallbacks(o.ctx, o.hooksConfig.Hooks.SessionEnd, o.cfg.WorkDir, hookVars, onStart, onComplete)
		if err != nil {
			// Context cancelled - just log and exit gracefully
			if o.ctx.Err() != nil {
				logger.Info("Context cancelled during session_end hook execution")
				return nil
			}
			logger.Error("Session_end hook execution failed: %v", err)
			// Don't fail the session - hooks are best-effort
		} else {
			logger.Info("Session_end hooks completed successfully")
		}
	}

	return nil
}

// runIteration0 executes Iteration #0 (planning phase) for fresh sessions.
// Uses the same MCP server as the main loop but with a planning-only prompt
// that instructs the agent to load all tasks from the spec.
// No hooks are executed during iteration #0.
func (o *Orchestrator) runIteration0() error {
	logger.Info("=== Starting Iteration #0 (Planning Phase) ===")

	// Log iteration start
	if err := o.store.IterationStart(o.ctx, o.cfg.SessionName, 0); err != nil {
		return fmt.Errorf("failed to log iteration #0 start: %w", err)
	}

	// Send iteration start message to TUI
	if o.tuiProgram != nil {
		o.tuiProgram.Send(tui.IterationStartMsg{Number: 0})
	}

	// Build the planning prompt using the Iteration #0 template
	prompt, err := template.BuildIteration0Prompt(o.ctx, template.BuildConfig{
		SessionName:       o.cfg.SessionName,
		Store:             o.store,
		IterationNumber:   0,
		SpecPath:          o.cfg.SpecPath,
		ExtraInstructions: o.cfg.ExtraInstructions,
		NATSPort:          o.natsPort,
	})
	if err != nil {
		return fmt.Errorf("failed to build iteration #0 prompt: %w", err)
	}
	logger.Debug("Iteration #0 prompt built, length: %d characters", len(prompt))

	// Run the agent using the main MCP server (same as iteration loop)
	logger.Info("Running agent for Iteration #0")
	if err := o.runner.RunIteration(o.ctx, prompt, ""); err != nil {
		return fmt.Errorf("iteration #0 agent execution failed: %w", err)
	}

	// Log iteration complete
	if err := o.store.IterationComplete(o.ctx, o.cfg.SessionName, 0); err != nil {
		return fmt.Errorf("failed to log iteration #0 complete: %w", err)
	}

	logger.Info("=== Iteration #0 (Planning Phase) completed ===")

	if o.cfg.Headless {
		fmt.Printf("\n✓ Iteration #0 (planning) complete\n\n")
	}

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

// runAutoCommit executes auto-commit after iteration completes.
// Checks if in git repo, builds commit prompt with file list and context,
// and reuses existing Runner to send commit request to current ACP session.
func (o *Orchestrator) runAutoCommit(ctx context.Context) error {
	// Check if in git repo
	if !isGitRepo(o.cfg.WorkDir) {
		logger.Debug("Not in git repo, skipping auto-commit")
		return nil
	}

	logger.Info("Running auto-commit for %d modified file(s)", o.fileTracker.Count())

	// Build commit prompt with file list and context
	prompt := o.buildCommitPrompt(ctx)

	// Reuse existing Runner - send commit prompt to current ACP session
	// This is faster than spawning a new subprocess and the session already
	// has context about what work was done
	logger.Debug("Sending commit prompt to existing ACP session")
	if err := o.runner.SendMessages(ctx, []string{prompt}); err != nil {
		return fmt.Errorf("failed to send commit prompt: %w", err)
	}

	logger.Info("Auto-commit request sent successfully")
	return nil
}

// buildCommitPrompt generates a commit prompt with modified file list and context.
// Includes: file paths with +/- counts, current task, iteration summary.
func (o *Orchestrator) buildCommitPrompt(ctx context.Context) string {
	// Get modified files list
	paths := o.fileTracker.ModifiedPaths()

	// Get context from session store
	state, err := o.store.LoadState(ctx, o.cfg.SessionName)
	if err != nil {
		logger.Warn("Failed to load session state for commit prompt: %v", err)
		// Continue without context - still provide file list
	}

	// Find in_progress task (if any) for context
	var currentTask string
	if state != nil {
		for _, t := range state.Tasks {
			if t.Status == "in_progress" {
				currentTask = t.Content
				break
			}
		}
	}

	// Get latest iteration summary (if any)
	var iterationSummary string
	if state != nil && len(state.Iterations) > 0 {
		lastIter := state.Iterations[len(state.Iterations)-1]
		iterationSummary = lastIter.Summary
	}

	// Build prompt
	var sb strings.Builder
	sb.WriteString("Commit the following modified files:\n\n")
	for _, p := range paths {
		change := o.fileTracker.Get(p)
		if change == nil {
			continue
		}
		if change.IsNew {
			sb.WriteString(fmt.Sprintf("- %s (new file)\n", p))
		} else if change.Additions > 0 || change.Deletions > 0 {
			sb.WriteString(fmt.Sprintf("- %s (+%d/-%d)\n", p, change.Additions, change.Deletions))
		} else {
			// No metadata available
			sb.WriteString(fmt.Sprintf("- %s\n", p))
		}
	}

	// Add context if available
	if currentTask != "" || iterationSummary != "" {
		sb.WriteString("\nContext:\n")
		if currentTask != "" {
			sb.WriteString(fmt.Sprintf("- Task: %s\n", currentTask))
		}
		if iterationSummary != "" {
			sb.WriteString(fmt.Sprintf("- Summary: %s\n", iterationSummary))
		}
	}

	sb.WriteString("\nInstructions:\n")
	sb.WriteString("1. Stage only the listed files with `git add`\n")
	sb.WriteString("2. Create a commit with a clear, conventional message\n")
	sb.WriteString("3. Do NOT push\n")

	return sb.String()
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

	// Stop MCP server (after runner, before NATS)
	if o.mcpServer != nil {
		logger.Debug("Stopping MCP server")
		if err := o.mcpServer.Stop(); err != nil {
			logger.Error("MCP server shutdown failed: %v", err)
			multiErr.Append(fmt.Errorf("MCP server shutdown failed: %w", err))
		} else {
			logger.Debug("MCP server stopped successfully")
		}
		o.mcpServer = nil
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
	o.tuiApp = tui.NewApp(o.ctx, o.store, o.cfg.SessionName, o.cfg.WorkDir, o.cfg.DataDir, o.nc, o.sendChan, o)

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

// appendPendingOutput appends hook output to the pending buffer (FIFO order).
// Thread-safe for use from NATS callbacks.
func (o *Orchestrator) appendPendingOutput(output string) {
	if output == "" {
		return
	}
	o.pendingMu.Lock()
	defer o.pendingMu.Unlock()

	if o.pendingHookOutput == "" {
		o.pendingHookOutput = output
	} else {
		o.pendingHookOutput += "\n" + output
	}
}

// drainPendingOutput returns and clears the pending buffer.
// Thread-safe for use with NATS callbacks.
func (o *Orchestrator) drainPendingOutput() string {
	o.pendingMu.Lock()
	defer o.pendingMu.Unlock()

	output := o.pendingHookOutput
	o.pendingHookOutput = ""
	return output
}

// hasPendingOutput checks if there is pending hook output.
// Thread-safe for use with NATS callbacks.
func (o *Orchestrator) hasPendingOutput() bool {
	o.pendingMu.Lock()
	defer o.pendingMu.Unlock()

	return o.pendingHookOutput != ""
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

// RequestPause sets the paused flag to request a pause after current iteration.
func (o *Orchestrator) RequestPause() {
	logger.Debug("Pause requested")
	o.paused.Store(true)
}

// CancelPause clears the pause flag (only effective before waitIfPaused blocks).
func (o *Orchestrator) CancelPause() {
	logger.Debug("Pause cancelled")
	o.paused.Store(false)
}

// Resume clears the pause flag and signals resumeChan to unblock waitIfPaused.
func (o *Orchestrator) Resume() {
	logger.Debug("Resume requested")
	o.paused.Store(false)
	// Send non-blocking signal to resumeChan
	select {
	case o.resumeChan <- struct{}{}:
	default:
		// Channel already has a signal, no need to send another
	}
}

// IsPaused returns the current pause state (for TUI display).
func (o *Orchestrator) IsPaused() bool {
	return o.paused.Load()
}

// waitIfPaused blocks if the orchestrator is paused, waiting for resume or context cancellation.
// Called after each iteration completes and user messages are processed.
// Returns nil on resume, or ctx.Err() if context is cancelled.
func (o *Orchestrator) waitIfPaused() error {
	// Fast path: if not paused, return immediately
	if !o.paused.Load() {
		return nil
	}

	// Paused flag is set - notify TUI that we're now blocking
	logger.Info("Orchestrator paused, waiting for resume signal")
	if o.tuiProgram != nil {
		o.tuiProgram.Send(tui.PauseStateMsg{Paused: true})
	}

	// Block until resume signal or context cancellation
	select {
	case <-o.resumeChan:
		// Drain channel in case of multiple signals (unlikely but safe)
		select {
		case <-o.resumeChan:
		default:
		}
		logger.Info("Orchestrator resumed")
		return nil
	case <-o.ctx.Done():
		logger.Info("Context cancelled during pause")
		return o.ctx.Err()
	}
}

// hookCallbacks returns onStart and onComplete callbacks that send TUI messages.
// hookType is the lifecycle phase (e.g. "session_start", "pre_iteration").
// Returns (onStart, onComplete, hookIDs) where hookIDs maps hook index → hookID.
func (o *Orchestrator) hookCallbacks(hookType string) (hooks.OnHookStart, hooks.OnHookComplete, map[int]string) {
	hookIDs := make(map[int]string)

	if o.tuiProgram == nil {
		// Headless mode - no TUI to notify
		return nil, nil, hookIDs
	}

	onStart := func(hookIndex int, command string) {
		id := fmt.Sprintf("hook-%s-%d", hookType, o.hookCounter.Add(1))
		hookIDs[hookIndex] = id
		o.tuiProgram.Send(tui.HookStartMsg{
			HookID:   id,
			HookType: hookType,
			Command:  command,
		})
	}

	onComplete := func(hookIndex int, result hooks.HookResult) {
		id, ok := hookIDs[hookIndex]
		if !ok {
			return
		}
		status := tui.HookStatusSuccess
		if result.Failed {
			status = tui.HookStatusError
		}
		o.tuiProgram.Send(tui.HookCompleteMsg{
			HookID:   id,
			Status:   status,
			Output:   result.Output,
			Duration: result.Duration,
		})
	}

	return onStart, onComplete, hookIDs
}
