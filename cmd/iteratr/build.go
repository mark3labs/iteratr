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
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run the iterative agent build loop",
	Long: `Run the iterative agent build loop for a session.

The build command starts an iterative loop where an AI agent works on tasks
defined in a spec file. It uses embedded NATS for persistence and presents
a TUI (unless --headless) to monitor progress.`,
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().StringVarP(&buildFlags.name, "name", "n", "", "Session name (default: spec filename stem)")
	buildCmd.Flags().StringVarP(&buildFlags.spec, "spec", "s", "", "Spec file path (default: ./specs/SPEC.md)")
	buildCmd.Flags().StringVarP(&buildFlags.template, "template", "t", "", "Custom template file")
	buildCmd.Flags().StringVarP(&buildFlags.extraInstructions, "extra-instructions", "e", "", "Extra instructions for prompt")
	buildCmd.Flags().IntVarP(&buildFlags.iterations, "iterations", "i", 0, "Max iterations, 0=infinite (default: 0)")
	buildCmd.Flags().BoolVar(&buildFlags.headless, "headless", false, "Run without TUI (logging only)")
	buildCmd.Flags().StringVar(&buildFlags.dataDir, "data-dir", ".iteratr", "Data directory for NATS storage")
	buildCmd.Flags().StringVarP(&buildFlags.model, "model", "m", config.DefaultModel, "Model to use (e.g., anthropic/claude-sonnet-4-5, openai/gpt-4)")
	buildCmd.Flags().BoolVar(&buildFlags.reset, "reset", false, "Reset session data before starting (clears all NATS events for this session)")
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
	// Track temp template file for cleanup
	var tempTemplatePath string

	// Run wizard if no spec provided and not headless
	if buildFlags.spec == "" && !buildFlags.headless {
		logger.Info("No spec file provided, launching wizard...")

		// Set up NATS for wizard session selector
		dataDir := os.Getenv("ITERATR_DATA_DIR")
		if dataDir == "" {
			dataDir = buildFlags.dataDir
		}
		wizardStore, cleanup, err := setupWizardStore(dataDir)
		if err != nil {
			return fmt.Errorf("failed to setup wizard store: %w", err)
		}
		defer cleanup()

		result, err := wizard.RunWizard(wizardStore)
		if err != nil {
			return fmt.Errorf("wizard failed: %w", err)
		}

		// Check if resuming existing session or creating new one
		if result.ResumeMode {
			// Resume mode: only session name is set from wizard
			// Spec/model/template will fall back to defaults or existing CLI flags
			buildFlags.name = result.SessionName
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
	specPath := buildFlags.spec
	if specPath == "" {
		// Look for SPEC.md in specs/ directory
		defaultSpec := "specs/SPEC.md"
		if _, err := os.Stat(defaultSpec); err == nil {
			specPath = defaultSpec
		} else {
			return fmt.Errorf("no spec file found, use --spec to specify path or run without --headless to use wizard")
		}
	}

	// Check if spec file exists
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		return fmt.Errorf("spec file not found: %s", specPath)
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

	// Get environment-based config
	dataDir := os.Getenv("ITERATR_DATA_DIR")
	if dataDir == "" {
		dataDir = buildFlags.dataDir
	}

	// Determine template path with precedence:
	// 1. Explicit --template flag
	// 2. .iteratr.template in current directory (if exists)
	// 3. Default embedded template (handled by template package)
	templatePath := buildFlags.template
	if templatePath == "" {
		if _, err := os.Stat(".iteratr.template"); err == nil {
			templatePath = ".iteratr.template"
		}
	}

	// Create orchestrator
	orch, err := orchestrator.New(orchestrator.Config{
		SessionName:       sessionName,
		SpecPath:          specPath,
		TemplatePath:      templatePath,
		ExtraInstructions: buildFlags.extraInstructions,
		Iterations:        buildFlags.iterations,
		DataDir:           dataDir,
		Headless:          buildFlags.headless,
		Model:             buildFlags.model,
		Reset:             buildFlags.reset,
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
