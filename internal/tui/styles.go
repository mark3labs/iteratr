package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// COLOR PALETTE - Catppuccin Mocha Inspired
// ============================================================================
//
// This palette provides a modern, cohesive color system with:
// - Excellent contrast for accessibility
// - Soothing pastel tones that reduce eye strain
// - Semantic color meanings for intuitive UI
// - Layered backgrounds for depth and hierarchy
//
// COLOR USAGE GUIDELINES:
//
// BACKGROUNDS (darkest to lightest):
//
//	colorCrust   - Deep backgrounds, rarely used
//	colorMantle  - Header/footer backgrounds
//	colorBase    - Main application background
//	colorSurface0/1/2 - Layered panels, cards, overlays
//
// TEXT (dimmest to brightest):
//
//	colorSubtext0 - Very muted text (timestamps, hints)
//	colorSubtext1 - Muted text (secondary info)
//	colorText     - Primary text content
//	colorTextBright - Emphasized text, titles
//
// ACCENTS:
//
//	colorPrimary  - Brand color, primary actions, focused elements
//	colorSecondary - Secondary actions, links, interactive elements
//	colorTertiary - Subtle highlights, tertiary actions
//
// SEMANTIC:
//
//	colorSuccess - Completed tasks, success states
//	colorWarning - In-progress, warnings, attention needed
//	colorError   - Errors, blocked states, destructive actions
//	colorInfo    - Informational notes, tips
//
// BORDERS:
//
//	colorBorderMuted   - Inactive/unfocused borders
//	colorBorderDefault - Standard borders
//	colorBorderFocused - Focused element borders
//	colorBorderActive  - Active/interactive borders
//
// ============================================================================
var (
	// === BASE LAYER (backgrounds) ===
	// Darkest to lightest background shades for layering UI elements
	colorBase     = lipgloss.Color("#1e1e2e") // Base background (darkest)
	colorMantle   = lipgloss.Color("#181825") // Slightly darker than base
	colorCrust    = lipgloss.Color("#11111b") // Darkest shade for deep backgrounds
	colorSurface0 = lipgloss.Color("#313244") // Surface overlay (light)
	colorSurface1 = lipgloss.Color("#45475a") // Surface overlay (medium)
	colorSurface2 = lipgloss.Color("#585b70") // Surface overlay (dark)
	colorOverlay0 = lipgloss.Color("#6c7086") // Overlay for subtle elements
	colorOverlay1 = lipgloss.Color("#7f849c") // Overlay medium
	colorOverlay2 = lipgloss.Color("#9399b2") // Overlay light

	// === TEXT LAYER (foreground) ===
	// Shades for different text emphasis levels
	colorSubtext0   = lipgloss.Color("#a6adc8") // Subtext (muted)
	colorSubtext1   = lipgloss.Color("#bac2de") // Subtext (less muted)
	colorText       = lipgloss.Color("#cdd6f4") // Main text color
	colorTextBright = lipgloss.Color("#f5e0dc") // Brightest text (rosewater)

	// === ACCENT COLORS (semantic) ===
	// Vibrant colors for UI highlights and status indicators
	colorPrimary   = lipgloss.Color("#cba6f7") // Mauve (primary brand color)
	colorSecondary = lipgloss.Color("#89b4fa") // Blue (secondary actions)
	colorTertiary  = lipgloss.Color("#b4befe") // Lavender (tertiary highlights)

	// Semantic status colors
	colorSuccess = lipgloss.Color("#a6e3a1") // Green (success, completed)
	colorWarning = lipgloss.Color("#f9e2af") // Yellow (warning, in-progress)
	colorError   = lipgloss.Color("#f38ba8") // Red (error, blocked)
	colorInfo    = lipgloss.Color("#89dceb") // Sky (info, notes)

	// Additional accent colors
	colorPeach = lipgloss.Color("#fab387") // Peach (warm accent)
	colorTeal  = lipgloss.Color("#94e2d5") // Teal (cool accent)
	colorPink  = lipgloss.Color("#f5c2e7") // Pink (special highlight)

	// === BORDER COLORS ===
	// Border shades for different focus states
	colorBorderMuted   = colorOverlay0  // Unfocused borders
	colorBorderDefault = colorSurface2  // Default borders
	colorBorderFocused = colorPrimary   // Focused element borders
	colorBorderActive  = colorSecondary // Active/interactive borders

	// === LEGACY ALIASES (for backward compatibility) ===
	// These map old color names to new palette
	colorMuted    = colorOverlay0 // Muted elements
	colorTextDim  = colorSubtext0 // Dim text
	colorBgHeader = colorMantle   // Header background
	colorBgFooter = colorMantle   // Footer background
	colorBgSubtle = colorSurface0 // Subtle background highlights
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
			BorderForeground(colorBorderDefault).
			Padding(0, 1)

	styleBorderFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorderFocused).
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
				Background(colorSurface0). // Slightly lighter than background
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
			Background(colorSurface0)

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
			BorderForeground(colorBorderDefault).
			Padding(1, 2)

	stylePanelFocused = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorBorderFocused).
				Padding(1, 2)

	stylePanelTitle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true).
			MarginBottom(1)

	stylePanelTitleFocused = lipgloss.NewStyle().
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

	// Scroll indicator style (for viewport scroll percentage)
	styleScrollIndicator = lipgloss.NewStyle().
				Foreground(colorSubtext0).
				Background(colorSurface0).
				Padding(0, 1)
)

// borderStyle returns the appropriate border style based on focus state
func borderStyle(focused bool) lipgloss.Style {
	if focused {
		return stylePanelFocused
	}
	return stylePanel
}
