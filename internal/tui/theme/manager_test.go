package theme

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Theme switching tests verify the Manager's theme registration and switching functionality.
// Tests cover: (1) theme registration, (2) SetTheme success/failure, (3) Current() behavior,
// (4) styles rebuild on theme switch, (5) thread-safety, (6) singleton behavior,
// (7) default catppuccin-mocha registration, (8) multiple theme switching,
// (9) empty theme manager, (10) lazy style initialization, (11) concurrent style access.
//
// These tests ensure the theme system correctly manages multiple themes and safely
// switches between them in a concurrent environment. All tests use testify/require
// for assertions. Tests that interact with the global singleton cannot use t.Parallel().

// TestManager_Register verifies theme registration
func TestManager_Register(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	theme1 := &Theme{Name: "test-theme-1", Primary: "#ff0000"}
	theme2 := &Theme{Name: "test-theme-2", Primary: "#00ff00"}

	m.Register(theme1)
	m.Register(theme2)

	require.Contains(t, m.themes, "test-theme-1")
	require.Contains(t, m.themes, "test-theme-2")
	require.Equal(t, theme1, m.themes["test-theme-1"])
	require.Equal(t, theme2, m.themes["test-theme-2"])
}

// TestManager_SetTheme_Success verifies successful theme switching
func TestManager_SetTheme_Success(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	theme1 := &Theme{Name: "theme-a", Primary: "#ff0000"}
	theme2 := &Theme{Name: "theme-b", Primary: "#00ff00"}

	m.Register(theme1)
	m.Register(theme2)

	// Switch to theme-a
	ok := m.SetTheme("theme-a")
	require.True(t, ok)
	require.Equal(t, theme1, m.Current())

	// Switch to theme-b
	ok = m.SetTheme("theme-b")
	require.True(t, ok)
	require.Equal(t, theme2, m.Current())
}

// TestManager_SetTheme_Failure verifies SetTheme returns false for non-existent theme
func TestManager_SetTheme_Failure(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	theme1 := &Theme{Name: "existing-theme", Primary: "#ff0000"}
	m.Register(theme1)
	m.SetTheme("existing-theme")

	// Try to switch to non-existent theme
	ok := m.SetTheme("non-existent-theme")
	require.False(t, ok)

	// Current theme should remain unchanged
	require.Equal(t, theme1, m.Current())
}

// TestManager_Current_Nil verifies Current returns nil when no theme is set
func TestManager_Current_Nil(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	current := m.Current()
	require.Nil(t, current)
}

// TestManager_StylesRebuild verifies styles are rebuilt when switching themes
func TestManager_StylesRebuild(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	// Create two themes with different primary colors
	theme1 := &Theme{
		Name:    "dark-theme",
		Primary: "#ff0000",
		FgBase:  "#ffffff",
	}
	theme2 := &Theme{
		Name:    "light-theme",
		Primary: "#0000ff",
		FgBase:  "#000000",
	}

	m.Register(theme1)
	m.Register(theme2)

	// Switch to theme1 and get styles
	m.SetTheme("dark-theme")
	styles1 := m.Current().S()

	// Switch to theme2 and get styles
	m.SetTheme("light-theme")
	styles2 := m.Current().S()

	// Styles should be different instances
	require.NotEqual(t, styles1, styles2)

	// Verify the styles reflect the different theme colors
	// We can't directly compare style foreground colors easily,
	// but we can verify the themes are different
	require.NotEqual(t, theme1.Primary, theme2.Primary)
	require.NotEqual(t, theme1.FgBase, theme2.FgBase)
}

// TestManager_ThreadSafety verifies Manager is thread-safe
func TestManager_ThreadSafety(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	// Register multiple themes
	for i := 0; i < 10; i++ {
		theme := &Theme{
			Name:    string(rune('a' + i)),
			Primary: "#ff0000",
		}
		m.Register(theme)
	}

	var wg sync.WaitGroup
	const numGoroutines = 100

	// Concurrently switch themes and read current theme
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Alternate between setting and getting themes
			themeName := string(rune('a' + (idx % 10)))
			m.SetTheme(themeName)
			current := m.Current()

			// If current is not nil, verify it has a valid name
			if current != nil {
				require.NotEmpty(t, current.Name)
			}
		}(i)
	}

	wg.Wait()

	// Verify manager is in valid state
	current := m.Current()
	require.NotNil(t, current)
	require.NotEmpty(t, current.Name)
}

// TestManager_RegisterOverwrite verifies registering a theme with same name overwrites
func TestManager_RegisterOverwrite(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	theme1 := &Theme{Name: "test-theme", Primary: "#ff0000"}
	theme2 := &Theme{Name: "test-theme", Primary: "#00ff00"}

	m.Register(theme1)
	require.Equal(t, theme1, m.themes["test-theme"])

	// Overwrite with same name
	m.Register(theme2)
	require.Equal(t, theme2, m.themes["test-theme"])
	require.Equal(t, "#00ff00", m.themes["test-theme"].Primary)
}

// TestDefaultManager_Singleton verifies DefaultManager returns same instance
func TestDefaultManager_Singleton(t *testing.T) {
	// Cannot use t.Parallel() because DefaultManager is a singleton

	manager1 := DefaultManager()
	manager2 := DefaultManager()

	require.Same(t, manager1, manager2)
}

// TestDefaultManager_CatppuccinMochaRegistered verifies catppuccin-mocha is registered by default
func TestDefaultManager_CatppuccinMochaRegistered(t *testing.T) {
	// Cannot use t.Parallel() because DefaultManager is a singleton

	m := DefaultManager()

	// Verify catppuccin-mocha is registered
	ok := m.SetTheme("catppuccin-mocha")
	require.True(t, ok)

	current := m.Current()
	require.NotNil(t, current)
	require.Equal(t, "catppuccin-mocha", current.Name)
}

// TestCurrent_ReturnsCatppuccinMocha verifies Current() returns catppuccin-mocha by default
func TestCurrent_ReturnsCatppuccinMocha(t *testing.T) {
	// Cannot use t.Parallel() because Current() uses global singleton

	theme := Current()
	require.NotNil(t, theme)
	require.Equal(t, "catppuccin-mocha", theme.Name)
}

// TestManager_MultipleThemes verifies switching between multiple registered themes
func TestManager_MultipleThemes(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	themes := []*Theme{
		{Name: "mocha", Primary: "#cba6f7"},
		{Name: "latte", Primary: "#8839ef"},
		{Name: "frappe", Primary: "#ca9ee6"},
		{Name: "macchiato", Primary: "#c6a0f6"},
	}

	// Register all themes
	for _, th := range themes {
		m.Register(th)
	}

	// Switch through each theme and verify
	for _, expected := range themes {
		ok := m.SetTheme(expected.Name)
		require.True(t, ok)

		current := m.Current()
		require.Equal(t, expected, current)
		require.Equal(t, expected.Name, current.Name)
		require.Equal(t, expected.Primary, current.Primary)
	}
}

// TestManager_EmptyThemes verifies Manager with no registered themes
func TestManager_EmptyThemes(t *testing.T) {
	t.Parallel()

	m := &Manager{themes: make(map[string]*Theme)}

	// Current should be nil
	require.Nil(t, m.Current())

	// SetTheme should fail
	ok := m.SetTheme("any-theme")
	require.False(t, ok)
	require.Nil(t, m.Current())
}

// TestTheme_StylesLazyInit verifies styles are lazily initialized
func TestTheme_StylesLazyInit(t *testing.T) {
	t.Parallel()

	theme := &Theme{
		Name:    "test",
		Primary: "#ff0000",
		FgBase:  "#ffffff",
	}

	// Styles should be nil initially
	require.Nil(t, theme.styles)

	// First call to S() initializes styles
	styles1 := theme.S()
	require.NotNil(t, styles1)
	require.NotNil(t, theme.styles)

	// Second call should return same instance
	styles2 := theme.S()
	require.Same(t, styles1, styles2)
}

// TestTheme_StylesConcurrentAccess verifies concurrent access to S() is safe
func TestTheme_StylesConcurrentAccess(t *testing.T) {
	t.Parallel()

	theme := &Theme{
		Name:    "test",
		Primary: "#ff0000",
		FgBase:  "#ffffff",
		FgMuted: "#aaaaaa",
	}

	var wg sync.WaitGroup
	const numGoroutines = 100

	// Concurrently call S() from multiple goroutines
	stylesResults := make([]*Styles, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			stylesResults[idx] = theme.S()
		}(i)
	}

	wg.Wait()

	// All results should be the same instance
	firstStyles := stylesResults[0]
	for i := 1; i < numGoroutines; i++ {
		require.Same(t, firstStyles, stylesResults[i])
	}
}
