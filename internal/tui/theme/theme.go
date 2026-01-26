package theme

import (
	"sync"

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
	return &Styles{
		HeaderTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)).
			Bold(true),
	}
}
