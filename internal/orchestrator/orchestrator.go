package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/acp"
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
}

// Orchestrator manages the iteration loop with embedded NATS, ACP client, and TUI.
type Orchestrator struct {
	cfg        Config
	ns         *natsserver.Server // Embedded NATS server
	nc         *natsgo.Conn       // NATS connection
	store      *session.Store     // Session store
	acpClient  *acp.ACPClient     // ACP client for agent communication
	tuiApp     *tui.App           // TUI application (nil if headless)
	tuiProgram *tea.Program       // Bubbletea program
	ctx        context.Context    // Context for cancellation
	cancel     context.CancelFunc // Cancel function
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
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start initializes all components and starts the orchestrator.
func (o *Orchestrator) Start() error {
	// 1. Start embedded NATS server
	if err := o.startNATS(); err != nil {
		return fmt.Errorf("failed to start NATS: %w", err)
	}

	// 2. Connect to NATS in-process
	if err := o.connectNATS(); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// 3. Setup JetStream stream
	if err := o.setupJetStream(); err != nil {
		return fmt.Errorf("failed to setup JetStream: %w", err)
	}

	// 4. Create ACP client
	o.acpClient = acp.NewACPClient(o.store, o.cfg.SessionName, o.cfg.WorkDir)

	// 5. Start TUI if not headless
	if !o.cfg.Headless {
		if err := o.startTUI(); err != nil {
			return fmt.Errorf("failed to start TUI: %w", err)
		}
	}

	return nil
}

// Run executes the main iteration loop.
func (o *Orchestrator) Run() error {
	// Load current session state to determine starting iteration
	state, err := o.store.LoadState(o.ctx, o.cfg.SessionName)
	if err != nil {
		return fmt.Errorf("failed to load session state: %w", err)
	}

	// Determine starting iteration number
	startIteration := len(state.Iterations) + 1

	// Check if session is already complete
	if state.Complete {
		fmt.Printf("Session '%s' is already marked as complete\n", o.cfg.SessionName)
		return nil
	}

	// Run iteration loop
	iterationCount := 0
	for {
		currentIteration := startIteration + iterationCount

		// Check iteration limit (0 = infinite)
		if o.cfg.Iterations > 0 && iterationCount >= o.cfg.Iterations {
			fmt.Printf("Reached iteration limit of %d\n", o.cfg.Iterations)
			break
		}

		// Log iteration start
		if err := o.store.IterationStart(o.ctx, o.cfg.SessionName, currentIteration); err != nil {
			return fmt.Errorf("failed to log iteration start: %w", err)
		}

		// Send iteration start message to TUI
		if o.tuiProgram != nil {
			o.tuiProgram.Send(tui.IterationStartMsg{Number: currentIteration})
		}

		// Build prompt with current state
		prompt, err := template.BuildPrompt(o.ctx, template.BuildConfig{
			SessionName:       o.cfg.SessionName,
			Store:             o.store,
			IterationNumber:   currentIteration,
			SpecPath:          o.cfg.SpecPath,
			TemplatePath:      o.cfg.TemplatePath,
			ExtraInstructions: o.cfg.ExtraInstructions,
		})
		if err != nil {
			return fmt.Errorf("failed to build prompt: %w", err)
		}

		// Setup callbacks for streaming output to TUI
		if o.tuiProgram != nil {
			o.acpClient.SetOutputCallback(func(content string) {
				o.tuiProgram.Send(tui.AgentOutputMsg{Content: content})
			})
		}

		// Run agent iteration
		fmt.Printf("Running iteration #%d...\n", currentIteration)
		if err := o.acpClient.RunIteration(o.ctx, prompt); err != nil {
			return fmt.Errorf("iteration #%d failed: %w", currentIteration, err)
		}

		// Log iteration complete
		if err := o.store.IterationComplete(o.ctx, o.cfg.SessionName, currentIteration); err != nil {
			return fmt.Errorf("failed to log iteration complete: %w", err)
		}

		// Check if session_complete was signaled
		if o.acpClient.IsSessionComplete() {
			fmt.Printf("Session '%s' marked as complete by agent\n", o.cfg.SessionName)
			break
		}

		iterationCount++
	}

	return nil
}

// Stop gracefully shuts down all components.
func (o *Orchestrator) Stop() error {
	// Cancel context
	o.cancel()

	// Stop TUI
	if o.tuiProgram != nil {
		o.tuiProgram.Quit()
	}

	// Close NATS connection
	if o.nc != nil {
		o.nc.Close()
	}

	// Shutdown NATS server
	if o.ns != nil {
		o.ns.Shutdown()
		o.ns.WaitForShutdown()
	}

	return nil
}

// startNATS starts the embedded NATS server.
func (o *Orchestrator) startNATS() error {
	// Ensure data directory exists
	dataDir := filepath.Join(o.cfg.DataDir, "nats")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create NATS data directory: %w", err)
	}

	// Configure embedded NATS with JetStream
	opts := &natsserver.Options{
		JetStream:  true,
		StoreDir:   dataDir,
		DontListen: true, // No network ports - in-process only
	}

	// Create server
	ns, err := natsserver.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create NATS server: %w", err)
	}

	// Start server in background
	go ns.Start()

	// Wait for server to be ready
	if !ns.ReadyForConnections(4 * 1e9) { // 4 seconds in nanoseconds
		return fmt.Errorf("NATS server failed to start in time")
	}

	o.ns = ns
	return nil
}

// connectNATS creates an in-process connection to the embedded NATS server.
func (o *Orchestrator) connectNATS() error {
	nc, err := natsgo.Connect("", natsgo.InProcessServer(o.ns))
	if err != nil {
		return fmt.Errorf("failed to connect to embedded NATS: %w", err)
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

	// Start TUI in background
	go func() {
		if _, err := o.tuiProgram.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		}
	}()

	return nil
}
