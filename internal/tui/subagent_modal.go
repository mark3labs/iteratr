package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// SubagentModal displays a full-screen modal that loads and replays a subagent session.
// It reuses the existing ScrollList and MessageItem infrastructure from AgentOutput.
type SubagentModal struct {
	// Content display (reuses AgentOutput infrastructure)
	scrollList *ScrollList    // For scrolling and rendering
	messages   []MessageItem  // Message accumulation
	toolIndex  map[string]int // toolCallId → message index

	// Session metadata
	sessionID    string
	subagentType string
	workDir      string

	// ACP subprocess (populated by Start())
	loader *agent.SessionLoader

	// State
	loading bool
	err     error // Non-nil shows error message in modal

	// Spinner for loading state (created lazily when needed)
	spinner *GradientSpinner

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSubagentModal creates a new SubagentModal.
// Initial dimensions are placeholder - will be updated on first Draw().
func NewSubagentModal(sessionID, subagentType, workDir string) *SubagentModal {
	ctx, cancel := context.WithCancel(context.Background())
	spinner := NewDefaultGradientSpinner("Loading session...")
	return &SubagentModal{
		sessionID:    sessionID,
		subagentType: subagentType,
		workDir:      workDir,
		scrollList:   NewScrollList(80, 20), // Placeholder dimensions
		messages:     make([]MessageItem, 0),
		toolIndex:    make(map[string]int),
		loading:      true,
		ctx:          ctx,
		cancel:       cancel,
		spinner:      &spinner,
	}
}

// Start spawns the ACP subprocess, initializes it, and begins loading the session.
// Returns a command that will start the session loading process.
func (m *SubagentModal) Start() tea.Cmd {
	return func() tea.Msg {
		// Spawn SessionLoader subprocess
		loader, err := agent.NewSessionLoader(m.ctx, m.workDir)
		if err != nil {
			logger.Warn("Failed to start ACP subprocess for subagent modal: %v", err)
			return SubagentErrorMsg{Err: fmt.Errorf("failed to start ACP: %w", err)}
		}
		m.loader = loader

		// Load the session (triggers replay)
		if err := loader.LoadAndStream(m.ctx, m.sessionID, m.workDir); err != nil {
			logger.Warn("Failed to load session %s: %v", m.sessionID, err)
			return SubagentErrorMsg{Err: fmt.Errorf("session not found: %s", m.sessionID)}
		}

		// Session loading started - modal no longer in loading state
		m.loading = false

		// Start streaming notifications
		return m.streamNext()
	}
}

// Draw renders the modal as a full-screen overlay.
// Handles three states: loading (spinner), error (message), and content (scroll list).
func (m *SubagentModal) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Calculate modal dimensions (full-screen with small margins)
	modalWidth := area.Dx() - 4
	modalHeight := area.Dy() - 4
	if modalWidth < 40 {
		modalWidth = area.Dx()
	}
	if modalHeight < 10 {
		modalHeight = area.Dy()
	}

	// Calculate content area dimensions
	contentWidth := modalWidth - 6   // Account for border (2) + padding (4)
	contentHeight := modalHeight - 5 // Account for padding (2) + title (1) + separator (1) + hint (1)
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	s := theme.Current().S()

	// Build title with subagent type
	titleText := fmt.Sprintf("Subagent: %s", m.subagentType)
	title := renderModalTitle(titleText, contentWidth)
	separator := s.ModalSeparator.Render(strings.Repeat("─", contentWidth))

	// Build content based on state
	var content string
	var hint string

	if m.err != nil {
		// Error state: show error message with ESC hint
		errorMsg := s.Error.Render(fmt.Sprintf("× %s", m.err.Error()))

		// Center error message vertically
		padding := strings.Repeat("\n", (contentHeight-1)/2)
		errorContent := padding + errorMsg

		content = strings.Join([]string{
			title,
			separator,
			errorContent,
		}, "\n")
		hint = RenderHint(KeyEsc, "close")

	} else if m.loading {
		// Loading state: show spinner
		spinnerView := ""
		if m.spinner != nil {
			spinnerView = m.spinner.View()
		}

		// Center spinner vertically
		padding := strings.Repeat("\n", (contentHeight-1)/2)
		spinnerContent := padding + spinnerView

		content = strings.Join([]string{
			title,
			separator,
			spinnerContent,
		}, "\n")
		hint = RenderHint(KeyEsc, "close")

	} else {
		// Content state: show session history via scrollList
		// Update scrollList dimensions to match content area
		m.scrollList.SetWidth(contentWidth)
		m.scrollList.SetHeight(contentHeight)

		// Get scrollList view
		listContent := m.scrollList.View()

		content = strings.Join([]string{
			title,
			separator,
			listContent,
		}, "\n")
		hint = RenderHintBar(KeyEsc, "close", KeyUpDown, "scroll")
	}

	// Add hint at bottom
	content = strings.Join([]string{content, hint}, "\n")

	// Style the modal
	modalStyle := s.ModalContainer.
		Width(modalWidth).
		Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Center on screen
	renderedWidth := lipgloss.Width(modalContent)
	renderedHeight := lipgloss.Height(modalContent)
	x := (area.Dx() - renderedWidth) / 2
	y := (area.Dy() - renderedHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	modalArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X + x, Y: area.Min.Y + y},
		Max: uv.Position{X: area.Min.X + x + renderedWidth, Y: area.Min.Y + y + renderedHeight},
	}
	uv.NewStyledString(modalContent).Draw(scr, modalArea)

	return nil
}

// Update handles keyboard input for scrolling.
// Forwards scroll key events (up/down/pgup/pgdown/home/end) to the internal scrollList.
func (m *SubagentModal) Update(msg tea.Msg) tea.Cmd {
	if m.scrollList == nil {
		return nil
	}

	// Set scrollList as focused to enable keyboard handling
	m.scrollList.SetFocused(true)
	defer m.scrollList.SetFocused(false)

	// Forward message to scrollList
	return m.scrollList.Update(msg)
}

// HandleUpdate processes streaming messages from the subagent session.
// Returns a command to continue streaming if Continue is true.
func (m *SubagentModal) HandleUpdate(msg tea.Msg) tea.Cmd {
	// This will be implemented in task TAS-17 (continuous streaming)
	return nil
}

// streamNext reads the next message from the session stream and returns a tea.Cmd.
// This will be implemented in task TAS-17 (continuous streaming).
func (m *SubagentModal) streamNext() tea.Msg {
	// Placeholder - will be implemented in TAS-17
	return SubagentDoneMsg{}
}

// Close terminates the ACP subprocess and cleans up resources.
// Safe to call multiple times or if Start() was never called.
func (m *SubagentModal) Close() {
	// Cancel context to stop any ongoing operations
	if m.cancel != nil {
		m.cancel()
	}

	// Close SessionLoader if established
	if m.loader != nil {
		if err := m.loader.Close(); err != nil {
			logger.Warn("Failed to close session loader: %v", err)
		}
		m.loader = nil
	}
}
