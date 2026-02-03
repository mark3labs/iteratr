package main

import (
	"fmt"

	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/specwizard"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Create a feature spec via AI-assisted interview",
	Long: `Create a feature spec via AI-assisted interview.

The spec command launches a wizard that collects a feature name and description,
then spawns an AI agent to interview you in depth about requirements, edge cases,
and implementation details. After the interview, the agent generates a complete
spec document which you can review and edit before saving.

Configuration is loaded from multiple sources with the following precedence:
  CLI flags > Environment variables > Project config > Global config > Defaults

Project config: ./iteratr.yml
Global config: ~/.config/iteratr/iteratr.yml`,
	RunE: runSpec,
}

func runSpec(cmd *cobra.Command, args []string) error {
	// Load config via Viper
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if config exists or if model is set via ENV var
	// If neither exists, prompt user to run setup
	if !config.Exists() && cfg.Model == "" {
		return fmt.Errorf("no configuration found\n\nRun 'iteratr setup' to create a config file, or set ITERATR_MODEL environment variable")
	}

	// Validate that model is set
	if cfg.Model == "" {
		return fmt.Errorf("model not configured\n\nSet model via:\n  - iteratr setup (creates config file)\n  - ITERATR_MODEL environment variable")
	}

	// Run the spec wizard
	if err := specwizard.Run(cfg); err != nil {
		return fmt.Errorf("spec wizard failed: %w", err)
	}

	return nil
}
