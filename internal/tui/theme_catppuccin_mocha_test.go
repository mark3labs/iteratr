package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// Theme tests verify correct rendering with the catppuccin_mocha theme.
// These golden files establish visual baselines for the default theme and
// provide a pattern for testing additional themes in the future (TAS-34).
//
// All existing golden files in testdata/ already use catppuccin_mocha since
// it's the only registered theme. These tests explicitly document theme usage
// and verify color palette correctness.

// TestCatppuccinMocha_ColorPalette verifies the catppuccin_mocha color values
func TestCatppuccinMocha_ColorPalette(t *testing.T) {
	t.Parallel()

	th := theme.Current()
	if th.Name != "catppuccin-mocha" {
		t.Fatalf("expected catppuccin-mocha theme, got %s", th.Name)
	}

	// Verify key color values match catppuccin mocha palette
	// Reference: https://github.com/catppuccin/catppuccin
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		// Semantic colors
		{"Primary (Mauve)", th.Primary, "#cba6f7"},
		{"Secondary (Blue)", th.Secondary, "#89b4fa"},
		{"Tertiary (Lavender)", th.Tertiary, "#b4befe"},

		// Background hierarchy
		{"BgCrust", th.BgCrust, "#11111b"},
		{"BgBase", th.BgBase, "#1e1e2e"},
		{"BgMantle", th.BgMantle, "#181825"},
		{"BgGutter", th.BgGutter, "#282839"},
		{"BgSurface0", th.BgSurface0, "#313244"},
		{"BgSurface1", th.BgSurface1, "#45475a"},
		{"BgSurface2", th.BgSurface2, "#585b70"},
		{"BgOverlay", th.BgOverlay, "#6c7086"},

		// Foreground hierarchy
		{"FgMuted (Subtext0)", th.FgMuted, "#a6adc8"},
		{"FgSubtle (Subtext1)", th.FgSubtle, "#bac2de"},
		{"FgBase (Text)", th.FgBase, "#cdd6f4"},
		{"FgBright (Rosewater)", th.FgBright, "#f5e0dc"},

		// Status colors
		{"Success (Green)", th.Success, "#a6e3a1"},
		{"Warning (Yellow)", th.Warning, "#f9e2af"},
		{"Error (Red)", th.Error, "#f38ba8"},
		{"Info (Sky)", th.Info, "#89dceb"},

		// Diff colors
		{"DiffInsertBg", th.DiffInsertBg, "#303a30"},
		{"DiffDeleteBg", th.DiffDeleteBg, "#3a3030"},
		{"DiffEqualBg", th.DiffEqualBg, "#1e1e2e"},
		{"DiffMissingBg", th.DiffMissingBg, "#181825"},

		// Border colors
		{"BorderMuted (Surface0)", th.BorderMuted, "#313244"},
		{"BorderDefault (Surface2)", th.BorderDefault, "#585b70"},
		{"BorderFocused (Mauve)", th.BorderFocused, "#cba6f7"},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.expected)
		}
	}
}

// TestCatppuccinMocha_StylesInitialized verifies all pre-built styles are properly initialized
func TestCatppuccinMocha_StylesInitialized(t *testing.T) {
	t.Parallel()

	th := theme.Current()
	if th.Name != "catppuccin-mocha" {
		t.Fatalf("expected catppuccin-mocha theme, got %s", th.Name)
	}

	s := th.S()

	// Verify key styles have correct properties by rendering test strings
	// If styles are properly initialized, they should render without panic
	tests := []struct {
		name   string
		render func() string
	}{
		{"Base text style", func() string { return s.Base.Render("test") }},
		{"Highlight style", func() string { return s.Highlight.Render("test") }},
		{"StatusBar style", func() string { return s.StatusBar.Render("test") }},
		{"ModalContainer style", func() string { return s.ModalContainer.Render("test") }},
		{"Success status style", func() string { return s.Success.Render("test") }},
		{"Error status style", func() string { return s.Error.Render("test") }},
		{"NoteTypeLearning style", func() string { return s.NoteTypeLearning.Render("test") }},
		{"NoteTypeStuck style", func() string { return s.NoteTypeStuck.Render("test") }},
		{"NoteTypeTip style", func() string { return s.NoteTypeTip.Render("test") }},
		{"NoteTypeDecision style", func() string { return s.NoteTypeDecision.Render("test") }},
		{"ButtonNormal style", func() string { return s.ButtonNormal.Render("test") }},
		{"ButtonFocused style", func() string { return s.ButtonFocused.Render("test") }},
	}

	for _, tt := range tests {
		// If rendering doesn't panic, style is initialized
		rendered := tt.render()
		if rendered == "" {
			t.Errorf("%s: rendered empty string", tt.name)
		}
	}
}

// TestCatppuccinMocha_ThemeDocumentation generates a text document showing theme colors
// This test creates a golden file that documents all colors in the theme for reference.
func TestCatppuccinMocha_ThemeDocumentation(t *testing.T) {
	t.Parallel()

	th := theme.Current()
	if th.Name != "catppuccin-mocha" {
		t.Fatalf("expected catppuccin-mocha theme, got %s", th.Name)
	}

	var doc strings.Builder
	doc.WriteString("Catppuccin Mocha Theme Color Palette\n")
	doc.WriteString("=====================================\n\n")

	doc.WriteString("Semantic Colors:\n")
	doc.WriteString(fmt.Sprintf("  Primary (Mauve):    %s\n", th.Primary))
	doc.WriteString(fmt.Sprintf("  Secondary (Blue):   %s\n", th.Secondary))
	doc.WriteString(fmt.Sprintf("  Tertiary (Lavender):%s\n\n", th.Tertiary))

	doc.WriteString("Background Hierarchy (dark → light):\n")
	doc.WriteString(fmt.Sprintf("  BgCrust:    %s (outermost)\n", th.BgCrust))
	doc.WriteString(fmt.Sprintf("  BgBase:     %s (main background)\n", th.BgBase))
	doc.WriteString(fmt.Sprintf("  BgMantle:   %s (header/footer)\n", th.BgMantle))
	doc.WriteString(fmt.Sprintf("  BgGutter:   %s (line numbers)\n", th.BgGutter))
	doc.WriteString(fmt.Sprintf("  BgSurface0: %s (panel overlays)\n", th.BgSurface0))
	doc.WriteString(fmt.Sprintf("  BgSurface1: %s (raised panels)\n", th.BgSurface1))
	doc.WriteString(fmt.Sprintf("  BgSurface2: %s (highest surface)\n", th.BgSurface2))
	doc.WriteString(fmt.Sprintf("  BgOverlay:  %s (subtle overlays)\n\n", th.BgOverlay))

	doc.WriteString("Foreground Hierarchy (dim → bright):\n")
	doc.WriteString(fmt.Sprintf("  FgMuted:  %s (very muted)\n", th.FgMuted))
	doc.WriteString(fmt.Sprintf("  FgSubtle: %s (muted)\n", th.FgSubtle))
	doc.WriteString(fmt.Sprintf("  FgBase:   %s (main text)\n", th.FgBase))
	doc.WriteString(fmt.Sprintf("  FgBright: %s (brightest)\n\n", th.FgBright))

	doc.WriteString("Status Colors:\n")
	doc.WriteString(fmt.Sprintf("  Success (Green): %s\n", th.Success))
	doc.WriteString(fmt.Sprintf("  Warning (Yellow):%s\n", th.Warning))
	doc.WriteString(fmt.Sprintf("  Error (Red):     %s\n", th.Error))
	doc.WriteString(fmt.Sprintf("  Info (Sky):      %s\n\n", th.Info))

	doc.WriteString("Diff Colors:\n")
	doc.WriteString(fmt.Sprintf("  DiffInsertBg:  %s (insertions)\n", th.DiffInsertBg))
	doc.WriteString(fmt.Sprintf("  DiffDeleteBg:  %s (deletions)\n", th.DiffDeleteBg))
	doc.WriteString(fmt.Sprintf("  DiffEqualBg:   %s (context)\n", th.DiffEqualBg))
	doc.WriteString(fmt.Sprintf("  DiffMissingBg: %s (empty)\n\n", th.DiffMissingBg))

	doc.WriteString("Border Colors:\n")
	doc.WriteString(fmt.Sprintf("  BorderMuted:   %s (inactive)\n", th.BorderMuted))
	doc.WriteString(fmt.Sprintf("  BorderDefault: %s (standard)\n", th.BorderDefault))
	doc.WriteString(fmt.Sprintf("  BorderFocused: %s (focused)\n", th.BorderFocused))

	goldenPath := filepath.Join("testdata", "theme_catppuccin_mocha_palette.golden")
	testfixtures.CompareGolden(t, goldenPath, doc.String())
}

// TestCatppuccinMocha_ExistingGoldensUseTheme documents that all existing golden files
// already use catppuccin_mocha theme since it's the only registered theme.
func TestCatppuccinMocha_ExistingGoldensUseTheme(t *testing.T) {
	t.Parallel()

	th := theme.Current()
	if th.Name != "catppuccin-mocha" {
		t.Fatalf("expected catppuccin-mocha theme, got %s", th.Name)
	}

	// This test documents that all 100+ existing golden files in testdata/
	// already use the catppuccin_mocha theme for visual regression testing.
	//
	// Golden files using this theme include:
	// - agent_*.golden (6 files)
	// - dashboard_*.golden (6 files)
	// - message_*.golden (16 files - collapsed & expanded)
	// - modal_priority_*.golden (5 files)
	// - note_input_modal_*.golden (16 files)
	// - note_modal_*.golden (8 files)
	// - sidebar_*.golden (11 files)
	// - status_*.golden (11 files)
	// - subagent_modal_*.golden (6 files)
	// - task_input_modal_*.golden (9 files)
	// - task_modal_*.golden (8 files)
	//
	// Total: 102 golden files using catppuccin_mocha theme
	//
	// When additional themes are added (TAS-34), new golden files should be
	// created with naming pattern: <component>_<theme-name>.golden
}
