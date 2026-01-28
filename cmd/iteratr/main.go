package main

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/spf13/cobra"
)

const (
	logoText1 = "▀ ▀█▀ █▀▀ █▀█ ▄▀█ ▀█▀ █▀█"
	logoText2 = "█  █  ██▄ █▀▄ █▀█  █  █▀▄"
)

// Version set via ldflags during build
var version = "dev"

func main() {
	// Ensure logger is closed on exit
	defer func() { _ = logger.Close() }()

	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(version)); err != nil {
		logger.Error("Command execution failed: %v", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "iteratr",
	Short: "AI coding agent orchestrator with embedded persistence and TUI",
}

// renderLogo creates the logo with gradient colors
func renderLogo() string {
	t := theme.NewCatppuccinMocha()
	line1 := theme.ApplyGradient(logoText1, t.Primary, t.Secondary)
	line2 := theme.ApplyGradient(logoText2, t.Primary, t.Secondary)
	return strings.Join([]string{line1, line2}, "\n")
}

func init() {
	// Set Long description with logo
	rootCmd.Long = renderLogo() + `

iteratr is a Go CLI tool that orchestrates AI coding agents in an iterative loop.
It manages session state (tasks, notes, inbox) via embedded NATS JetStream,
communicates with opencode via ACP (Agent Control Protocol) over stdio,
and presents a full-screen TUI using Bubbletea v2.

Spiritual successor to ralph.nu - same concepts, modern Go implementation.

Getting Started:
  1. Run 'iteratr setup' to create your config file
  2. Run 'iteratr build' to start a session
  3. Run 'iteratr config' to view current settings`

	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(genTemplateCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configCmd)
}
