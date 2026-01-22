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
	colorBgSubtle = lipgloss.Color("237") // Slightly lighter than background

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

	// Status bar style (task stats above footer)
	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBgHeader).
			Padding(0, 1).
			Align(lipgloss.Right)

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

	// Dashboard styles
	styleProgressBar = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Background(colorMuted)

	styleProgressFill = lipgloss.NewStyle().
				Foreground(colorTextBright).
				Background(colorPrimary)

	styleStatLabel = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Bold(false)

	styleStatValue = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleCurrentTask = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1).
				Foreground(colorTextBright)

	// Task list styles
	styleTaskID = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(false)

	styleTaskContent = lipgloss.NewStyle().
				Foreground(colorText)

	styleTaskSelected = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Background(lipgloss.Color("237")). // Slightly lighter than background
				Bold(true)

	styleGroupHeader = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				Underline(true).
				MarginTop(1)

	// Log viewer styles
	styleLogTimestamp = lipgloss.NewStyle().
				Foreground(colorTextDim).
				Bold(false)

	styleLogTask = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleLogNote = lipgloss.NewStyle().
			Foreground(colorWarning)

	styleLogInbox = lipgloss.NewStyle().
			Foreground(colorPrimary)

	styleLogIteration = lipgloss.NewStyle().
				Foreground(colorSuccess)

	styleLogControl = lipgloss.NewStyle().
			Foreground(colorError)

	styleLogContent = lipgloss.NewStyle().
			Foreground(colorText)

	// Notes panel styles
	styleNoteTypeLearning = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	styleNoteTypeStuck = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)

	styleNoteTypeTip = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	styleNoteTypeDecision = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	styleNoteIteration = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(false)

	styleNoteContent = lipgloss.NewStyle().
				Foreground(colorText).
				PaddingLeft(2)

	// Inbox styles
	styleMessageUnread = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	styleMessageRead = lipgloss.NewStyle().
				Foreground(colorTextDim)

	styleMessageTimestamp = lipgloss.NewStyle().
				Foreground(colorTextDim).
				Bold(false)

	styleInputField = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorSecondary).
			Padding(0, 1).
			Foreground(colorText)

	styleInputPrompt = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	// Agent output styles
	styleAgentText = lipgloss.NewStyle().
			Foreground(colorText)

	styleAgentCode = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Background(lipgloss.Color("236"))

	styleAgentThinking = lipgloss.NewStyle().
				Foreground(colorTextDim).
				Italic(true)

	styleAgentOutput = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder(), false, false, false, true). // Left border only
				BorderForeground(colorPrimary).
				PaddingLeft(1)

	// List styles
	styleListItem = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(2)

	styleListBullet = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// Panel styles
	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)

	stylePanelTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	// Badge styles
	styleBadge = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true)

	styleBadgeSuccess = styleBadge.Copy().
				Foreground(colorTextBright).
				Background(colorSuccess)

	styleBadgeWarning = styleBadge.Copy().
				Foreground(colorTextBright).
				Background(colorWarning)

	styleBadgeError = styleBadge.Copy().
			Foreground(colorTextBright).
			Background(colorError)

	styleBadgeInfo = styleBadge.Copy().
			Foreground(colorTextBright).
			Background(colorSecondary)

	styleBadgeMuted = styleBadge.Copy().
			Foreground(colorText).
			Background(colorMuted)

	// Scrollbar styles
	styleScrollbar = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleScrollbarThumb = lipgloss.NewStyle().
				Foreground(colorSecondary)

	// Empty state styles
	styleEmptyState = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Italic(true).
			Align(lipgloss.Center)
)
