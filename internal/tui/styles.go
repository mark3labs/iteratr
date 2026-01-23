package tui

import (
	"strings"

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
	colorGutterBg = lipgloss.Color("#282839") // Gutter well background (between base and surface0)
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
	// Session info styles (used in status bar)
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

	// Status bar style (session info + connection status)
	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBgHeader).
			Padding(0, 1)

	// View status indicators
	styleStatusRemaining  = lipgloss.NewStyle().Foreground(colorMuted)
	styleStatusInProgress = lipgloss.NewStyle().Foreground(colorWarning)
	styleStatusCompleted  = lipgloss.NewStyle().Foreground(colorSuccess)
	styleStatusBlocked    = lipgloss.NewStyle().Foreground(colorError)

	// General styles
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
	styleTaskSelected = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Background(colorSurface0). // Slightly lighter than background
				Bold(true)

	// Log viewer styles
	styleLogTimestamp = lipgloss.NewStyle().
				Foreground(colorTextDim).
				Bold(false)

	styleLogTask = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleLogNote = lipgloss.NewStyle().
			Foreground(colorWarning)

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

	styleInputField = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorSecondary).
			Padding(0, 1).
			Foreground(colorText)

	styleInputPrompt = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	styleInputSeparator = lipgloss.NewStyle().
				Foreground(colorMuted)

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

	// Thinking block styles
	styleThinkingBox = lipgloss.NewStyle().
				Background(colorSurface0).
				PaddingLeft(1).
				MarginBottom(1)

	styleThinkingContent = lipgloss.NewStyle().
				Foreground(colorSubtext1).
				Italic(true)

	styleThinkingTruncationHint = lipgloss.NewStyle().
					Foreground(colorSubtext0).
					Italic(true)

	styleThinkingFooter = lipgloss.NewStyle().
				Foreground(colorSubtext0)

	styleThinkingDuration = lipgloss.NewStyle().
				Foreground(colorSecondary)

	// Tool call styles
	styleToolIconPending = lipgloss.NewStyle().
				Foreground(colorWarning)

	styleToolIconSuccess = lipgloss.NewStyle().
				Foreground(colorSuccess)

	styleToolIconError = lipgloss.NewStyle().
				Foreground(colorError)

	styleToolIconCanceled = lipgloss.NewStyle().
				Foreground(colorMuted)

	styleToolName = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleToolParams = lipgloss.NewStyle().
			Foreground(colorSubtext0)

	styleToolTruncation = lipgloss.NewStyle().
				Foreground(colorSubtext0).
				Background(colorSurface0).
				Italic(true).
				MarginLeft(2)

	styleToolError = lipgloss.NewStyle().
			Foreground(colorError)

	styleToolOutput = lipgloss.NewStyle().
			Background(colorSurface0).
			MarginLeft(2).
			PaddingLeft(1)

	// Code block styles (for file view/read tool output)
	styleCodeLineNum = lipgloss.NewStyle().
				Foreground(colorOverlay0).
				Background(colorGutterBg).
				PaddingRight(1)

	styleCodeLineNumZero = lipgloss.NewStyle().
				Foreground(colorGutterBg).
				Background(colorGutterBg)

	styleCodeContent = lipgloss.NewStyle().
				Background(colorSurface0).
				PaddingLeft(1)

	styleCodeTruncation = lipgloss.NewStyle().
				Foreground(colorSubtext0).
				Background(colorSurface0).
				Italic(true).
				MarginLeft(2)

	// Diff view styles (for Edit tool output)
	colorDiffInsertBg = lipgloss.Color("#303a30") // Green-tinted background for insertions
	colorDiffDeleteBg = lipgloss.Color("#3a3030") // Red-tinted background for deletions

	styleDiffLineNumInsert = lipgloss.NewStyle().
				Foreground(colorOverlay0).
				Background(colorDiffInsertBg)

	styleDiffLineNumDelete = lipgloss.NewStyle().
				Foreground(colorOverlay0).
				Background(colorDiffDeleteBg)

	styleDiffContentInsert = lipgloss.NewStyle().
				Background(colorDiffInsertBg)

	styleDiffContentDelete = lipgloss.NewStyle().
				Background(colorDiffDeleteBg).
				Strikethrough(true)

	styleDiffDivider = lipgloss.NewStyle().
				Foreground(colorSurface2)

	// Diagnostic styles (for compiler errors/warnings)
	styleDiagFile = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true).
			MarginLeft(2)

	styleDiagError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	styleDiagWarning = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	styleDiagPosition = lipgloss.NewStyle().
				Foreground(colorSubtext0)

	styleDiagMessage = lipgloss.NewStyle().
				Foreground(colorText)

	// Info message styles
	styleInfoIcon = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleInfoModel = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleInfoProvider = lipgloss.NewStyle().
				Foreground(colorSubtext0)

	styleInfoDuration = lipgloss.NewStyle().
				Foreground(colorInfo)

	styleInfoRule = lipgloss.NewStyle().
			Foreground(colorSurface2)

	// Finish state styles
	styleFinishError = lipgloss.NewStyle().
				Foreground(colorError)

	styleFinishCanceled = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	// Assistant message border
	styleAssistantBorder = lipgloss.NewStyle().
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

	// Panel styles (no borders - focus indicated by header color)
	stylePanelTitle = lipgloss.NewStyle().
			Foreground(colorSubtext0).
			Bold(true)

	stylePanelTitleFocused = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	stylePanelRule = lipgloss.NewStyle().
			Foreground(colorSurface2)

	stylePanelRuleFocused = lipgloss.NewStyle().
				Foreground(colorPrimary)

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

	// Modal styles
	styleModalContainer = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorderFocused).
				Background(colorBase).
				Padding(1, 2)

	styleModalTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Align(lipgloss.Center)

	styleModalSection = lipgloss.NewStyle().
				Foreground(colorText)

	styleModalSeparator = lipgloss.NewStyle().
				Foreground(colorSurface2)

	styleModalLabel = lipgloss.NewStyle().
			Foreground(colorSubtext0)

	styleModalValue = lipgloss.NewStyle().
			Foreground(colorText)

	styleModalHint = lipgloss.NewStyle().
			Foreground(colorSubtext0).
			Italic(true).
			Align(lipgloss.Center)

	// Modal hint key/description styles (crush pattern)
	styleHintKey = lipgloss.NewStyle().
			Foreground(colorSubtext1).
			Bold(true)

	styleHintDesc = lipgloss.NewStyle().
			Foreground(colorSubtext0)

	styleHintSeparator = lipgloss.NewStyle().
				Foreground(colorSurface2)
)

// renderModalTitle renders a title with diagonal hatching decoration.
// Creates format: "Title ╱╱╱╱╱╱╱╱╱" with a gradient from primary to secondary.
func renderModalTitle(title string, width int) string {
	styledTitle := styleModalTitle.Render(title)
	titleWidth := lipgloss.Width(styledTitle)

	remainingWidth := width - titleWidth - 1 // -1 for space before hatching
	if remainingWidth <= 0 {
		return styledTitle
	}

	// Build gradient hatching in segments for performance
	// Use ~8 color stops across the width instead of per-character
	const maxStops = 8
	segmentSize := remainingWidth / maxStops
	if segmentSize < 1 {
		segmentSize = 1
	}

	var hatching strings.Builder
	for i := 0; i < remainingWidth; i += segmentSize {
		pos := float64(i) / float64(remainingWidth)
		color := interpolateColor(string(colorPrimary), string(colorSecondary), pos)
		charStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

		// Render a segment of characters with the same color
		end := i + segmentSize
		if end > remainingWidth {
			end = remainingWidth
		}
		hatching.WriteString(charStyle.Render(strings.Repeat("╱", end-i)))
	}

	return styledTitle + " " + hatching.String()
}
