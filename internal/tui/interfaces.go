package tui

import (
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// Drawable components render to a screen rectangle
type Drawable interface {
	Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor
}

// Updateable components handle messages
type Updateable interface {
	Update(tea.Msg) tea.Cmd
}

// Component combines Drawable and Updateable
type Component interface {
	Drawable
	Updateable
}

// Sizable components track their dimensions
type Sizable interface {
	SetSize(width, height int)
}

// Focusable components track focus state
type Focusable interface {
	SetFocus(focused bool)
	IsFocused() bool
}

// Stateful components receive state updates
type Stateful interface {
	SetState(state *session.State)
}

// FullComponent combines Drawable, Updateable, Sizable, and Stateful
type FullComponent interface {
	Component
	Sizable
	Stateful
}

// FocusableComponent adds focus to FullComponent
type FocusableComponent interface {
	FullComponent
	Focusable
}

// ViewType represents the different view screens in the TUI
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewLogs
)
