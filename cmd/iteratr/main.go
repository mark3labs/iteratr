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
	Short: "Orchestrates AI coding agents in an iterative loop",
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

Orchestrates AI coding agents in an iterative loop.

Getting Started:
  iteratr setup  - create config
  iteratr build  - start session
  iteratr config - view settings`

	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(specCmd)
	rootCmd.AddCommand(genTemplateCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configCmd)
}
