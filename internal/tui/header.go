package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// Header renders the top header bar with session info and status.
type Header struct {
	width       int
	sessionName string
	state       *session.State
}

// NewHeader creates a new Header component.
func NewHeader(sessionName string) *Header {
	return &Header{
		sessionName: sessionName,
	}
}

// Draw renders the header to the screen at the given area.
// Returns nil cursor since header is non-interactive.
func (h *Header) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	if area.Dy() < 1 {
		return nil
	}

	// Build left side: app title | session | iteration
	title := styleHeaderTitle.Render("iteratr")
	sep := styleHeaderSeparator.Render(" | ")
	sessionInfo := styleHeaderInfo.Render(h.sessionName)

	left := title + sep + sessionInfo

	// Add iteration info if available
	if h.state != nil && len(h.state.Iterations) > 0 {
		// Find current (latest) iteration
		currentIter := h.state.Iterations[len(h.state.Iterations)-1]
		iterInfo := fmt.Sprintf("Iteration #%d", currentIter.Number)
		left += sep + styleHeaderInfo.Render(iterInfo)
	}

	// Build right side: connection status indicator
	right := h.renderStatusIndicator()

	// Calculate spacing to fill width
	leftRendered := left
	rightRendered := right
	totalWidth := area.Dx()

	// Combine left, spacing, and right
	content := h.buildHeader(leftRendered, rightRendered, totalWidth)

	// Render to screen using DrawStyled helper
	DrawStyled(scr, area, styleHeader, content)

	return nil
}

// buildHeader combines left and right content with spacing.
func (h *Header) buildHeader(left, right string, totalWidth int) string {
	// Strip styles to measure actual content width
	leftWidth := len(stripANSI(left))
	rightWidth := len(stripANSI(right))

	// Calculate padding needed
	padding := totalWidth - leftWidth - rightWidth - 2 // -2 for side padding
	if padding < 1 {
		padding = 1
	}

	// Build final content
	var spacer string
	for i := 0; i < padding; i++ {
		spacer += " "
	}

	return left + spacer + right
}

// renderStatusIndicator renders the connection/status indicator on the right.
func (h *Header) renderStatusIndicator() string {
	// For now, show a simple connected indicator
	// In future iterations, this could reflect actual connection status
	indicator := styleHeaderInfo.Render("● connected")
	return indicator
}

// SetSize updates the header width.
func (h *Header) SetSize(width, height int) {
	h.width = width
}

// SetState updates the session state.
func (h *Header) SetState(state *session.State) {
	h.state = state
}

// Update handles messages. Header is mostly static, so this is minimal.
func (h *Header) Update(msg tea.Msg) tea.Cmd {
	return nil
}

// stripANSI removes ANSI escape sequences to measure actual text width.
// This is a simplified version - for production, use a proper ANSI parser.
func stripANSI(s string) string {
	// Simple heuristic: count visible characters
	// Lipgloss provides Width() method which handles this properly
	// For now, we'll use a simple approximation
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}

// Compile-time interface checks
var _ FullComponent = (*Header)(nil)
