package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/orchestrator"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/spf13/cobra"
)

var buildFlags struct {
	name              string
	spec              string
	template          string
	extraInstructions string
	iterations        int
	headless          bool
	dataDir           string
	model             string
	reset             bool
	autoCommit        bool
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run the iterative agent build loop",
	Long: `Run the iterative agent build loop for a session.

The build command starts an iterative loop where an AI agent works on tasks
defined in a spec file. It uses embedded NATS for persistence and presents
a TUI (unless --headless) to monitor progress.

Configuration is loaded from multiple sources with the following precedence:
  CLI flags > Environment variables > Project config > Global config > Defaults

Project config: ./iteratr.yml
Global config: ~/.config/iteratr/iteratr.yml`,
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().StringVarP(&buildFlags.name, "name", "n", "", "Session name (default: spec filename stem)")
	buildCmd.Flags().StringVarP(&buildFlags.spec, "spec", "s", "", "Spec file path (default: ./specs/SPEC.md)")
	buildCmd.Flags().StringVarP(&buildFlags.template, "template", "t", "", "Custom template file (overrides config file)")
	buildCmd.Flags().StringVarP(&buildFlags.extraInstructions, "extra-instructions", "e", "", "Extra instructions for prompt")
	buildCmd.Flags().IntVarP(&buildFlags.iterations, "iterations", "i", 0, "Max iterations, 0=infinite (overrides config file)")
	buildCmd.Flags().BoolVar(&buildFlags.headless, "headless", false, "Run without TUI (overrides config file)")
	buildCmd.Flags().StringVar(&buildFlags.dataDir, "data-dir", ".iteratr", "Data directory for NATS storage (overrides config file)")
	buildCmd.Flags().StringVarP(&buildFlags.model, "model", "m", "", "Model to use (overrides config file, e.g., anthropic/claude-sonnet-4-5)")
	buildCmd.Flags().BoolVar(&buildFlags.reset, "reset", false, "Reset session data before starting (clears all NATS events for this session)")
	buildCmd.Flags().BoolVar(&buildFlags.autoCommit, "auto-commit", true, "Auto-commit modified files after iteration (overrides config file)")
}

// setupWizardStore creates a temporary NATS connection and session store for the wizard.
// Returns the store and a cleanup function that must be called when done.
func setupWizardStore(dataDir string) (*session.Store, func(), error) {
	// Ensure data directory exists
	fullDataDir := filepath.Join(dataDir, "data")
	if err := os.MkdirAll(fullDataDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Try to connect to existing NATS server first
	nc := nats.TryConnectExisting(fullDataDir)
	var ns interface{} // NATS server (if we started one)

	if nc == nil {
		// No existing server, start one
		logger.Debug("Starting temporary NATS server for wizard")
		server, port, err := nats.StartEmbeddedNATS(fullDataDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to start NATS: %w", err)
		}
		ns = server

		// Connect to the server we just started
		nc, err = nats.ConnectToPort(port)
		if err != nil {
			server.Shutdown()
			return nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
		}
	}

	// Create JetStream context
	js, err := nats.CreateJetStream(nc)
	if err != nil {
		nc.Close()
		if ns != nil {
			ns.(*natsserver.Server).Shutdown()
		}
		return nil, nil, fmt.Errorf("failed to create JetStream: %w", err)
	}

	// Setup stream
	ctx := context.Background()
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		nc.Close()
		if ns != nil {
			ns.(*natsserver.Server).Shutdown()
		}
		return nil, nil, fmt.Errorf("failed to setup stream: %w", err)
	}

	// Create session store
	store := session.NewStore(js, stream)

	// Return cleanup function
	cleanup := func() {
		if nc != nil {
			nc.Close()
		}
		// Don't shutdown server if we didn't start it (connected to existing)
		// The orchestrator will manage the server lifecycle
	}

	return store, cleanup, nil
}

func runBuild(cmd *cobra.Command, args []string) error {
	// Load config via Viper
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if config exists or if model is set via ENV var or CLI flag
	// If neither exists, prompt user to run setup
	if !config.Exists() && cfg.Model == "" && !cmd.Flags().Changed("model") {
		return fmt.Errorf("no configuration found\n\nRun 'iteratr setup' to create a config file, or set ITERATR_MODEL environment variable")
	}

	// Validate config after merging with CLI flags (done below)
	// Note: We defer validation until after CLI flag merging since flags can override config

	// Apply config defaults to buildFlags if CLI flags were not explicitly set
	// This allows config to provide defaults, but CLI flags and wizard override them
	if !cmd.Flags().Changed("model") {
		buildFlags.model = cfg.Model
	}
	if !cmd.Flags().Changed("iterations") {
		buildFlags.iterations = cfg.Iterations
	}
	if !cmd.Flags().Changed("headless") {
		buildFlags.headless = cfg.Headless
	}
	if !cmd.Flags().Changed("auto-commit") {
		buildFlags.autoCommit = cfg.AutoCommit
	}
	if !cmd.Flags().Changed("data-dir") {
		buildFlags.dataDir = cfg.DataDir
	}
	if !cmd.Flags().Changed("template") {
		buildFlags.template = cfg.Template
	}

	// Validate that model is set after applying config and CLI flags
	// Model can come from config file, ENV var (ITERATR_MODEL), or CLI flag
	if buildFlags.model == "" {
		return fmt.Errorf("model not configured\n\nSet model via:\n  - iteratr setup (creates config file)\n  - ITERATR_MODEL environment variable\n  - --model flag")
	}

	// Track temp template file for cleanup
	var tempTemplatePath string
	// Track if we're resuming an existing session (spec is optional in this case)
	resumeMode := false

	// Run wizard if no spec provided and not headless
	if buildFlags.spec == "" && !buildFlags.headless {
		logger.Info("No spec file provided, launching wizard...")

		// Set up NATS for wizard session selector
		wizardStore, cleanup, err := setupWizardStore(buildFlags.dataDir)
		if err != nil {
			return fmt.Errorf("failed to setup wizard store: %w", err)
		}
		defer cleanup()

		result, err := wizard.RunWizard(cmd.Context(), wizardStore, buildFlags.template)
		if err != nil {
			return fmt.Errorf("wizard failed: %w", err)
		}

		// Check if resuming existing session or creating new one
		if result.ResumeMode {
			// Resume mode: only session name is set from wizard
			// Spec/model/template will fall back to defaults or existing CLI flags
			buildFlags.name = result.SessionName
			resumeMode = true
			logger.Info("Resuming existing session: %s", result.SessionName)
		} else {
			// New session mode: apply all wizard results to buildFlags
			buildFlags.spec = result.SpecPath
			buildFlags.model = result.Model
			buildFlags.name = result.SessionName
			buildFlags.iterations = result.Iterations

			// Write template to temp file if it was edited
			if result.Template != "" {
				// Create temp file with secure permissions
				tmpFile, err := os.CreateTemp("", "iteratr-template-*.txt")
				if err != nil {
					return fmt.Errorf("failed to create temp template file: %w", err)
				}

				// Write template content
				if _, err := tmpFile.WriteString(result.Template); err != nil {
					_ = tmpFile.Close()
					_ = os.Remove(tmpFile.Name())
					return fmt.Errorf("failed to write template to temp file: %w", err)
				}

				// Close file
				if err := tmpFile.Close(); err != nil {
					_ = os.Remove(tmpFile.Name())
					return fmt.Errorf("failed to close temp template file: %w", err)
				}

				// Set template path to temp file and track for cleanup
				tempTemplatePath = tmpFile.Name()
				buildFlags.template = tempTemplatePath
				logger.Debug("Wizard template written to temp file: %s", tempTemplatePath)
			}

			logger.Info("Wizard completed: spec=%s, model=%s, session=%s, iterations=%d",
				result.SpecPath, result.Model, result.SessionName, result.Iterations)
		}
	}

	// Determine spec path
	// In resume mode, spec is optional (session already has tasks)
	specPath := buildFlags.spec
	if specPath == "" {
		// Look for SPEC.md in specs/ directory
		defaultSpec := "specs/SPEC.md"
		if _, err := os.Stat(defaultSpec); err == nil {
			specPath = defaultSpec
		} else if !resumeMode {
			// Require spec file for new sessions (not resume mode)
			return fmt.Errorf("no spec file found, use --spec to specify path or run without --headless to use wizard")
		}
	}

	// Check if spec file exists (only if a spec path was determined)
	if specPath != "" {
		if _, err := os.Stat(specPath); os.IsNotExist(err) {
			return fmt.Errorf("spec file not found: %s", specPath)
		}
	}

	// Determine session name
	sessionName := buildFlags.name
	if sessionName == "" {
		// Derive from spec filename
		base := filepath.Base(specPath)
		ext := filepath.Ext(base)
		sessionName = strings.TrimSuffix(base, ext)

		// Replace dots with hyphens (NATS subject constraint)
		sessionName = strings.ReplaceAll(sessionName, ".", "-")
	}

	// Validate session name (alphanumeric, hyphens, underscores only)
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if len(sessionName) > 64 {
		return fmt.Errorf("session name too long (max 64 characters): %s", sessionName)
	}
	for _, r := range sessionName {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return fmt.Errorf("invalid session name: %s (use only alphanumeric, hyphens, underscores)", sessionName)
		}
	}

	// Validate iteration count
	if buildFlags.iterations < 0 {
		return fmt.Errorf("iterations must be >= 0 (0 means unlimited)")
	}

	// Use template path from config, CLI flag, or wizard
	// If empty, orchestrator will use embedded default template
	templatePath := buildFlags.template

	// Warn if .iteratr.template exists but template config is not set
	// (.iteratr.template auto-detection was deprecated - users should set template: .iteratr.template in config)
	if templatePath == "" && cfg.Template == "" {
		if _, err := os.Stat(".iteratr.template"); err == nil {
			logger.Warn("Found .iteratr.template file but template is not configured.")
			logger.Warn("Auto-detection of .iteratr.template was deprecated.")
			logger.Warn("To use this template, add 'template: .iteratr.template' to your config file.")
			logger.Warn("Run 'iteratr setup' or manually edit iteratr.yml")
		}
	}

	// Create orchestrator
	orch, err := orchestrator.New(orchestrator.Config{
		SessionName:       sessionName,
		SpecPath:          specPath,
		TemplatePath:      templatePath,
		ExtraInstructions: buildFlags.extraInstructions,
		Iterations:        buildFlags.iterations,
		DataDir:           buildFlags.dataDir,
		Headless:          buildFlags.headless,
		Model:             buildFlags.model,
		Reset:             buildFlags.reset,
		AutoCommit:        buildFlags.autoCommit,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		return fmt.Errorf("failed to start orchestrator: %w", err)
	}

	// Ensure cleanup always runs using defer
	defer func() {
		if err := orch.Stop(); err != nil {
			// Log error but don't write to stderr - corrupts terminal during TUI shutdown
			logger.Error("Error during shutdown: %v", err)
		}

		// Clean up temp template file if it was created
		if tempTemplatePath != "" {
			if err := os.Remove(tempTemplatePath); err != nil {
				logger.Warn("Failed to remove temp template file %s: %v", tempTemplatePath, err)
			} else {
				logger.Debug("Removed temp template file: %s", tempTemplatePath)
			}
		}
	}()

	// Run iteration loop (Bubbletea handles SIGINT/SIGTERM internally)
	if err := orch.Run(); err != nil {
		return fmt.Errorf("iteration loop failed: %w", err)
	}

	return nil
}
