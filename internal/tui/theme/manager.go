package theme

import "sync"

// Manager manages theme registration and switching.
type Manager struct {
	themes  map[string]*Theme
	current *Theme
	mu      sync.RWMutex
}

// Register adds a theme to the manager.
func (m *Manager) Register(t *Theme) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.themes[t.Name] = t
}

// SetTheme switches to the named theme.
func (m *Manager) SetTheme(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.themes[name]; ok {
		m.current = t
		return true
	}
	return false
}

// Current returns the currently active theme.
func (m *Manager) Current() *Theme {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Package-level singleton
var (
	manager     *Manager
	managerOnce sync.Once
)

// Current returns the currently active theme from the default manager.
func Current() *Theme {
	return DefaultManager().Current()
}

// DefaultManager returns the singleton theme manager.
// On first call, it registers and activates the Catppuccin Mocha theme.
func DefaultManager() *Manager {
	managerOnce.Do(func() {
		manager = &Manager{themes: make(map[string]*Theme)}
		manager.Register(NewCatppuccinMocha())
		manager.SetTheme("catppuccin-mocha")
	})
	return manager
}
