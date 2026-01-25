package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/orchestrator"
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
	buildCmd.Flags().StringVarP(&buildFlags.model, "model", "m", "anthropic/claude-sonnet-4-5", "Model to use (e.g., anthropic/claude-sonnet-4-5, openai/gpt-4)")
	buildCmd.Flags().BoolVar(&buildFlags.reset, "reset", false, "Reset session data before starting (clears all NATS events for this session)")
}

func runBuild(cmd *cobra.Command, args []string) error {
	// Determine spec path
	specPath := buildFlags.spec
	if specPath == "" {
		// Look for SPEC.md in specs/ directory
		defaultSpec := "specs/SPEC.md"
		if _, err := os.Stat(defaultSpec); err == nil {
			specPath = defaultSpec
		} else {
			return fmt.Errorf("no spec file found, use --spec to specify path")
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
	}()

	// Run iteration loop (Bubbletea handles SIGINT/SIGTERM internally)
	if err := orch.Run(); err != nil {
		return fmt.Errorf("iteration loop failed: %w", err)
	}

	return nil
}
