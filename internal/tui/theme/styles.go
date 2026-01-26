package theme

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// Styles contains all pre-built lipgloss styles for the TUI.
type Styles struct {
	// Base text styles
	Base   lipgloss.Style
	Muted  lipgloss.Style
	Subtle lipgloss.Style
	Bright lipgloss.Style
	Dim    lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// Header/Footer styles
	HeaderTitle     lipgloss.Style
	HeaderSeparator lipgloss.Style
	HeaderInfo      lipgloss.Style
	Footer          lipgloss.Style
	FooterKey       lipgloss.Style
	FooterLabel     lipgloss.Style
	FooterActive    lipgloss.Style

	// Status bar styles
	StatusBar        lipgloss.Style
	StatusRemaining  lipgloss.Style
	StatusInProgress lipgloss.Style
	StatusCompleted  lipgloss.Style
	StatusBlocked    lipgloss.Style
	Highlight        lipgloss.Style
	ProgressFill     lipgloss.Style
	StatLabel        lipgloss.Style
	StatValue        lipgloss.Style
	TaskSelected     lipgloss.Style
	ScrollIndicator  lipgloss.Style
	EmptyState       lipgloss.Style

	// Tool call styles
	ToolIconPending  lipgloss.Style
	ToolIconSuccess  lipgloss.Style
	ToolIconError    lipgloss.Style
	ToolIconCanceled lipgloss.Style
	ToolName         lipgloss.Style
	ToolParams       lipgloss.Style
	ToolTruncation   lipgloss.Style
	ToolError        lipgloss.Style
	ToolOutput       lipgloss.Style

	// Code block styles
	CodeLineNum     lipgloss.Style
	CodeLineNumZero lipgloss.Style
	CodeContent     lipgloss.Style
	CodeTruncation  lipgloss.Style

	// Write file block styles
	WriteLineNum     lipgloss.Style
	WriteLineNumZero lipgloss.Style
	WriteContent     lipgloss.Style

	// Diff view styles
	DiffLineNumInsert  lipgloss.Style
	DiffLineNumDelete  lipgloss.Style
	DiffLineNumEqual   lipgloss.Style
	DiffLineNumMissing lipgloss.Style
	DiffContentInsert  lipgloss.Style
	DiffContentDelete  lipgloss.Style
	DiffContentEqual   lipgloss.Style
	DiffContentMissing lipgloss.Style
	DiffDivider        lipgloss.Style

	// Diagnostic styles
	DiagFile     lipgloss.Style
	DiagError    lipgloss.Style
	DiagWarning  lipgloss.Style
	DiagPosition lipgloss.Style
	DiagMessage  lipgloss.Style

	// Info message styles
	InfoIcon     lipgloss.Style
	InfoModel    lipgloss.Style
	InfoProvider lipgloss.Style
	InfoDuration lipgloss.Style
	InfoRule     lipgloss.Style

	// Finish state styles
	FinishError    lipgloss.Style
	FinishCanceled lipgloss.Style

	// Thinking block styles
	ThinkingBox            lipgloss.Style
	ThinkingContent        lipgloss.Style
	ThinkingTruncationHint lipgloss.Style
	ThinkingFooter         lipgloss.Style
	ThinkingDuration       lipgloss.Style

	// Notes styles
	NoteTypeLearning lipgloss.Style
	NoteTypeStuck    lipgloss.Style
	NoteTypeTip      lipgloss.Style
	NoteTypeDecision lipgloss.Style
	NoteIteration    lipgloss.Style
	NoteContent      lipgloss.Style

	// Log viewer styles
	LogTimestamp lipgloss.Style
	LogTask      lipgloss.Style
	LogNote      lipgloss.Style
	LogIteration lipgloss.Style
	LogControl   lipgloss.Style
	LogContent   lipgloss.Style

	// Modal styles
	ModalContainer lipgloss.Style
	ModalTitle     lipgloss.Style
	ModalSection   lipgloss.Style
	ModalSeparator lipgloss.Style
	ModalLabel     lipgloss.Style
	ModalValue     lipgloss.Style
	ModalHint      lipgloss.Style
	HintKey        lipgloss.Style
	HintDesc       lipgloss.Style
	HintSeparator  lipgloss.Style

	// Badge styles
	Badge        lipgloss.Style
	BadgeSuccess lipgloss.Style
	BadgeWarning lipgloss.Style
	BadgeError   lipgloss.Style
	BadgeInfo    lipgloss.Style
	BadgeMuted   lipgloss.Style

	// Panel styles
	PanelTitle        lipgloss.Style
	PanelTitleFocused lipgloss.Style
	PanelRule         lipgloss.Style
	PanelRuleFocused  lipgloss.Style

	// Message styles
	AssistantBorder lipgloss.Style
	UserBorder      lipgloss.Style

	// Input styles
	TextInputStyles textinput.Styles

	// Button styles
	ButtonNormal   lipgloss.Style
	ButtonDisabled lipgloss.Style
	ButtonFocused  lipgloss.Style
}
