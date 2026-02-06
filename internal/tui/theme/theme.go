package theme

import (
	"sync"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Theme defines the color palette for the TUI.
type Theme struct {
	Name   string
	IsDark bool

	// Semantic colors
	Primary   string // lipgloss.Color is a string type
	Secondary string
	Tertiary  string

	// Background hierarchy (dark→light)
	BgCrust    string
	BgBase     string
	BgMantle   string
	BgGutter   string
	BgSurface0 string
	BgSurface1 string
	BgSurface2 string
	BgOverlay  string

	// Foreground hierarchy (dim→bright)
	FgMuted  string
	FgSubtle string
	FgBase   string
	FgBright string

	// Status colors
	Success string
	Warning string
	Error   string
	Info    string

	// Diff colors
	DiffInsertBg  string
	DiffDeleteBg  string
	DiffEqualBg   string
	DiffMissingBg string

	// Border colors
	BorderMuted   string
	BorderDefault string
	BorderFocused string

	// Lazy-built styles
	styles     *Styles
	stylesOnce sync.Once
}

// S returns the pre-built styles for this theme.
// Styles are lazily initialized on first call.
func (t *Theme) S() *Styles {
	t.stylesOnce.Do(func() {
		t.styles = t.buildStyles()
	})
	return t.styles
}

// buildStyles constructs the pre-built styles from theme colors.
func (t *Theme) buildStyles() *Styles {
	s := &Styles{}

	// Base text styles
	s.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase))
	s.Muted = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.Subtle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgSubtle))
	s.Bright = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBright))
	s.Dim = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))

	// Status styles
	s.Success = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
	s.Warning = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
	s.Error = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
	s.Info = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info))

	// Header/Footer styles
	s.HeaderTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	s.HeaderSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.HeaderInfo = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase))
	s.Footer = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase)).Background(lipgloss.Color(t.BgMantle)).Padding(0, 1)
	s.FooterKey = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary)).Bold(true)
	s.FooterLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase))
	s.FooterActive = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)

	// Status bar styles
	s.StatusBar = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase)).Background(lipgloss.Color(t.BgMantle)).Padding(0, 1)
	s.StatusRemaining = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.StatusInProgress = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
	s.StatusCompleted = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
	s.StatusBlocked = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
	s.StatusPausing = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning)).Bold(true)
	s.StatusPaused = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info)).Bold(true)
	s.Highlight = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	s.ProgressFill = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBright)).Background(lipgloss.Color(t.Primary))
	s.StatLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Bold(false)
	s.StatValue = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	s.TaskSelected = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Background(lipgloss.Color(t.BgSurface0)).Bold(true)
	s.ScrollIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.BgSurface0)).Padding(0, 1)
	s.EmptyState = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Italic(true).Align(lipgloss.Center)

	// Tool call styles
	s.ToolIconPending = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
	s.ToolIconSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
	s.ToolIconError = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
	s.ToolIconCanceled = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.ToolName = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary)).Bold(true)
	s.ToolParams = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.ToolTruncation = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.BgSurface0)).Italic(true).MarginLeft(2)
	s.ToolError = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
	s.ToolOutput = lipgloss.NewStyle().Background(lipgloss.Color(t.BgSurface0)).MarginLeft(2).PaddingLeft(1)

	// Code block styles
	s.CodeLineNum = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.BgGutter)).PaddingRight(1)
	s.CodeLineNumZero = lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgGutter)).Background(lipgloss.Color(t.BgGutter))
	s.CodeContent = lipgloss.NewStyle().Background(lipgloss.Color(t.BgSurface0)).PaddingLeft(1)
	s.CodeTruncation = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.BgSurface0)).Italic(true).MarginLeft(2)

	// Write file block styles
	s.WriteLineNum = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.BgGutter)).PaddingRight(1)
	s.WriteLineNumZero = lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgGutter)).Background(lipgloss.Color(t.BgGutter))
	s.WriteContent = lipgloss.NewStyle().Background(lipgloss.Color(t.DiffInsertBg)).PaddingLeft(1)

	// Diff view styles
	s.DiffLineNumInsert = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.DiffInsertBg))
	s.DiffLineNumDelete = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.DiffDeleteBg))
	s.DiffLineNumEqual = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.DiffEqualBg))
	s.DiffLineNumMissing = lipgloss.NewStyle().Background(lipgloss.Color(t.DiffMissingBg))
	s.DiffContentInsert = lipgloss.NewStyle().Background(lipgloss.Color(t.DiffInsertBg))
	s.DiffContentDelete = lipgloss.NewStyle().Background(lipgloss.Color(t.DiffDeleteBg)).Strikethrough(true)
	s.DiffContentEqual = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Background(lipgloss.Color(t.DiffEqualBg))
	s.DiffContentMissing = lipgloss.NewStyle().Background(lipgloss.Color(t.DiffMissingBg))
	s.DiffDivider = lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgSurface2))

	// Diagnostic styles
	s.DiagFile = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary)).Bold(true).MarginLeft(2)
	s.DiagError = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true)
	s.DiagWarning = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning)).Bold(true)
	s.DiagPosition = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.DiagMessage = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase))

	// Info message styles
	s.InfoIcon = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.InfoModel = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary))
	s.InfoProvider = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.InfoDuration = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Info))
	s.InfoRule = lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgSurface2))

	// Finish state styles
	s.FinishError = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
	s.FinishCanceled = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Italic(true)

	// Thinking block styles
	s.ThinkingBox = lipgloss.NewStyle().Background(lipgloss.Color(t.BgSurface0)).PaddingLeft(1).MarginBottom(1)
	s.ThinkingContent = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgSubtle)).Italic(true)
	s.ThinkingTruncationHint = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Italic(true)
	s.ThinkingFooter = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.ThinkingDuration = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary))

	// Notes styles
	s.NoteTypeLearning = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
	s.NoteTypeStuck = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true)
	s.NoteTypeTip = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary)).Bold(true)
	s.NoteTypeDecision = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning)).Bold(true)
	s.NoteIteration = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Bold(false)
	s.NoteContent = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase)).PaddingLeft(2)

	// Log viewer styles
	s.LogTimestamp = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Bold(false)
	s.LogTask = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary))
	s.LogNote = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
	s.LogIteration = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
	s.LogControl = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
	s.LogContent = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase))

	// Modal styles
	s.ModalContainer = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(t.BorderFocused)).Background(lipgloss.Color(t.BgBase)).Padding(1, 2)
	s.ModalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true).Align(lipgloss.Center)
	s.ModalSection = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase))
	s.ModalSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgSurface2))
	s.ModalLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.ModalValue = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase))
	s.ModalHint = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Italic(true).Align(lipgloss.Center)
	s.HintKey = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgSubtle)).Bold(true)
	s.HintDesc = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	s.HintSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgSurface2))

	// Badge styles
	baseBadge := lipgloss.NewStyle().Padding(0, 1).Bold(true)
	s.Badge = baseBadge
	s.BadgeSuccess = baseBadge.Foreground(lipgloss.Color(t.FgBright)).Background(lipgloss.Color(t.Success))
	s.BadgeWarning = baseBadge.Foreground(lipgloss.Color(t.FgBright)).Background(lipgloss.Color(t.Warning))
	s.BadgeError = baseBadge.Foreground(lipgloss.Color(t.FgBright)).Background(lipgloss.Color(t.Error))
	s.BadgeInfo = baseBadge.Foreground(lipgloss.Color(t.FgBright)).Background(lipgloss.Color(t.Secondary))
	s.BadgeMuted = baseBadge.Foreground(lipgloss.Color(t.FgBase)).Background(lipgloss.Color(t.FgMuted))

	// Panel styles
	s.PanelTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)).Bold(true)
	s.PanelTitleFocused = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	s.PanelRule = lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgSurface2))
	s.PanelRuleFocused = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))

	// Message styles
	s.AssistantBorder = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true). // Left border only
		BorderForeground(lipgloss.Color(t.Primary)).
		PaddingLeft(1)
	s.UserBorder = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true). // Left border only
		BorderForeground(lipgloss.Color(t.Secondary)).
		PaddingLeft(1).
		PaddingTop(1).
		PaddingBottom(1)
	s.QueuedBadge = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Secondary)).
		Foreground(lipgloss.Color(t.BgCrust)).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)
	s.IterationDivider = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgMuted)).
		Bold(true).
		MarginTop(1).
		MarginBottom(1)

	// Subagent message style - subtle box with rounded border
	s.SubagentMessageBox = lipgloss.NewStyle().
		Background(lipgloss.Color(t.BgSurface0)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.BorderMuted)).
		Padding(0, 1, 1, 1). // top, right, bottom, left
		MarginLeft(2)

	// Input styles (bubbles textinput)
	s.TextInputStyles = t.buildTextInputStyles()

	// Button styles
	s.ButtonNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgBase)).
		Background(lipgloss.Color(t.BgSurface0)).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.BorderDefault))
	s.ButtonDisabled = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgMuted)).
		Background(lipgloss.Color(t.BgSurface0)).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.BorderMuted))
	s.ButtonFocused = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Background(lipgloss.Color(t.BgSurface0)).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.BorderFocused)).
		Bold(true)

	return s
}

// buildTextInputStyles creates textinput.Styles for bubbles components.
func (t *Theme) buildTextInputStyles() textinput.Styles {
	return textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase)),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color(t.Tertiary)),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted)),
		},
		Cursor: textinput.CursorStyle{
			Color: lipgloss.Color(t.Primary),
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
}
