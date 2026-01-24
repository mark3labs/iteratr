package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags during build)
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Ensure logger is closed on exit
	defer func() { _ = logger.Close() }()

	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command execution failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "iteratr",
	Short: "AI coding agent orchestrator with embedded persistence and TUI",
	Long: `iteratr is a Go CLI tool that orchestrates AI coding agents in an iterative loop.
It manages session state (tasks, notes, inbox) via embedded NATS JetStream,
communicates with opencode via ACP (Agent Control Protocol) over stdio,
and presents a full-screen TUI using Bubbletea v2.

Spiritual successor to ralph.nu - same concepts, modern Go implementation.`,
}

func init() {
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(genTemplateCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
}
