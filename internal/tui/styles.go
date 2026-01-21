package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary colors
	colorPrimary   = lipgloss.Color("99")  // Bright purple
	colorSecondary = lipgloss.Color("39")  // Bright blue
	colorSuccess   = lipgloss.Color("42")  // Green
	colorWarning   = lipgloss.Color("214") // Orange
	colorError     = lipgloss.Color("196") // Red
	colorMuted     = lipgloss.Color("240") // Gray

	// Background colors
	colorBgHeader = lipgloss.Color("235") // Dark gray
	colorBgFooter = lipgloss.Color("235") // Dark gray

	// Text colors
	colorText       = lipgloss.Color("252") // Light gray
	colorTextBright = lipgloss.Color("255") // White
	colorTextDim    = lipgloss.Color("243") // Dim gray
)

// Style definitions
var (
	// Header styles
	styleHeader = lipgloss.NewStyle().
			Foreground(colorTextBright).
			Background(colorBgHeader).
			Bold(true).
			Padding(0, 1)

	styleHeaderTitle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	styleHeaderSeparator = lipgloss.NewStyle().
				Foreground(colorMuted)

	styleHeaderInfo = lipgloss.NewStyle().
			Foreground(colorText)

	// Footer styles
	styleFooter = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBgFooter).
			Padding(0, 1)

	styleFooterKey = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleFooterLabel = lipgloss.NewStyle().
				Foreground(colorText)

	styleFooterActive = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	// View status indicators
	styleStatusRemaining  = lipgloss.NewStyle().Foreground(colorMuted)
	styleStatusInProgress = lipgloss.NewStyle().Foreground(colorWarning)
	styleStatusCompleted  = lipgloss.NewStyle().Foreground(colorSuccess)
	styleStatusBlocked    = lipgloss.NewStyle().Foreground(colorError)

	// General styles
	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Underline(true)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleHighlight = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleDim = lipgloss.NewStyle().
			Foreground(colorTextDim)
)
