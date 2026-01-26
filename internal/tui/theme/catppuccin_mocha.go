package theme

// NewCatppuccinMocha creates the default Catppuccin Mocha theme.
func NewCatppuccinMocha() *Theme {
	return &Theme{
		Name:   "catppuccin-mocha",
		IsDark: true,

		// Semantic colors
		Primary:   "#cba6f7", // Mauve - primary brand color
		Secondary: "#89b4fa", // Blue - secondary actions
		Tertiary:  "#b4befe", // Lavender - tertiary highlights

		// Background hierarchy (dark→light)
		BgCrust:    "#11111b", // Crust - outermost app background
		BgBase:     "#1e1e2e", // Base - main background
		BgMantle:   "#181825", // Mantle - header/footer background
		BgGutter:   "#282839", // Gutter - line number background
		BgSurface0: "#313244", // Surface0 - panel overlays
		BgSurface1: "#45475a", // Surface1 - raised panels (not in original styles.go, using catppuccin value)
		BgSurface2: "#585b70", // Surface2 - highest surface level
		BgOverlay:  "#6c7086", // Overlay0 - subtle overlays

		// Foreground hierarchy (dim→bright)
		FgMuted:  "#a6adc8", // Subtext0 - very muted text
		FgSubtle: "#bac2de", // Subtext1 - muted text
		FgBase:   "#cdd6f4", // Text - main text color
		FgBright: "#f5e0dc", // Rosewater - brightest text

		// Status colors
		Success: "#a6e3a1", // Green - success, completed
		Warning: "#f9e2af", // Yellow - warning, in-progress
		Error:   "#f38ba8", // Red - error, blocked
		Info:    "#89dceb", // Sky - info, notes

		// Diff colors
		DiffInsertBg:  "#303a30", // Green-tinted background for insertions
		DiffDeleteBg:  "#3a3030", // Red-tinted background for deletions
		DiffEqualBg:   "#1e1e2e", // Neutral background for context lines
		DiffMissingBg: "#181825", // Dim background for empty sides

		// Border colors
		BorderMuted:   "#313244", // Surface0 - inactive/unfocused borders
		BorderDefault: "#585b70", // Surface2 - standard borders
		BorderFocused: "#cba6f7", // Mauve - focused element borders
	}
}
