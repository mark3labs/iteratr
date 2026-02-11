package wizard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// SessionItem represents a session in the session list.
// Implements ScrollItem interface for lazy rendering.
type SessionItem struct {
	info   session.SessionInfo // Session metadata
	isNew  bool                // True for "New Session" item
	height int                 // Cached height after rendering
}

// ID returns a unique identifier for this item (required by ScrollItem interface).
func (s *SessionItem) ID() string {
	if s.isNew {
		return "__new_session__"
	}
	return s.info.Name
}

// Render returns the rendered string representation (required by ScrollItem interface).
func (s *SessionItem) Render(width int) string {
	if s.isNew {
		// Special rendering for "New Session" entry
		display := "+ New Session"
		s.height = 1
		return display
	}

	// Format: "session-name  [Status]  X/Y tasks  relative-time"
	var parts []string

	// Session name (left-aligned)
	parts = append(parts, s.info.Name)

	// Status badge
	statusText := "[In Progress]"
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")) // Yellow
	if s.info.Complete {
		statusText = "[Complete]"
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")) // Green
	}
	parts = append(parts, statusStyle.Render(statusText))

	// Task progress
	progressText := fmt.Sprintf("%d/%d tasks", s.info.TasksCompleted, s.info.TasksTotal)
	parts = append(parts, progressText)

	// Relative time
	relativeTime := formatRelativeTime(s.info.LastActivity)
	parts = append(parts, relativeTime)

	display := strings.Join(parts, "  ")

	// Truncate if too long
	if len(display) > width-2 {
		display = display[:width-5] + "..."
	}

	s.height = 1
	return display
}

// Height returns the number of lines this item occupies (required by ScrollItem interface).
func (s *SessionItem) Height() int {
	return s.height
}

// formatRelativeTime formats a time.Time as a human-readable relative time string.
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

// SessionSelectorStep manages the session selector UI step.
type SessionSelectorStep struct {
	sessionStore    *session.Store  // Session store for loading sessions
	sessions        []SessionItem   // All session items ("New Session" + existing sessions)
	scrollList      *tui.ScrollList // Lazy-rendering scroll list
	selectedIdx     int             // Index of selected item
	loading         bool            // Whether sessions are being fetched
	error           string          // Error message if fetch failed
	spinner         spinner.Model   // Loading spinner
	width           int             // Available width
	height          int             // Available height
	state           string          // State machine: "listing", "confirm_continue", "confirm_reset"
	confirmInput    string          // User input for confirmation prompts
	selectedSession *SessionItem    // Session selected for confirmation flow
}

// NewSessionSelectorStep creates a new session selector step.
func NewSessionSelectorStep(sessionStore *session.Store) *SessionSelectorStep {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	scrollList := tui.NewScrollList(60, 10)
	scrollList.SetAutoScroll(false) // Manual navigation
	scrollList.SetFocused(true)

	return &SessionSelectorStep{
		sessionStore: sessionStore,
		scrollList:   scrollList,
		spinner:      s,
		loading:      true,
		selectedIdx:  0,
		width:        60,
		height:       10,
		state:        "listing",
	}
}

// Init initializes the session selector and starts fetching sessions.
func (s *SessionSelectorStep) Init() tea.Cmd {
	return tea.Batch(
		s.fetchSessions(),
		s.spinner.Tick,
	)
}

// fetchSessions loads all sessions from the session store.
func (s *SessionSelectorStep) fetchSessions() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sessions, err := s.sessionStore.ListSessions(ctx)
		if err != nil {
			return SessionsErrorMsg{err: err}
		}

		return SessionsLoadedMsg{sessions: sessions}
	}
}

// SetSize updates the dimensions for the session selector.
func (s *SessionSelectorStep) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.scrollList.SetWidth(width)
	s.scrollList.SetHeight(height)
}

// Update handles messages for the session selector step.
func (s *SessionSelectorStep) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case SessionsLoadedMsg:
		// Sessions fetched successfully
		s.loading = false
		s.sessions = make([]SessionItem, 0, len(msg.sessions)+1)

		// Add "New Session" at the top
		s.sessions = append(s.sessions, SessionItem{isNew: true})

		// Add existing sessions
		for _, info := range msg.sessions {
			s.sessions = append(s.sessions, SessionItem{
				info:  info,
				isNew: false,
			})
		}

		// Update scroll list with items
		scrollItems := make([]tui.ScrollItem, len(s.sessions))
		for i := range s.sessions {
			scrollItems[i] = &s.sessions[i]
		}
		s.scrollList.SetItems(scrollItems)
		s.scrollList.SetSelected(s.selectedIdx)
		// Notify wizard that content changed (for modal resizing)
		return func() tea.Msg { return ContentChangedMsg{} }

	case SessionsErrorMsg:
		// Error fetching sessions
		s.loading = false
		s.error = msg.err.Error()
		// Notify wizard that content changed (for modal resizing)
		return func() tea.Msg { return ContentChangedMsg{} }

	case spinner.TickMsg:
		if s.loading {
			var cmd tea.Cmd
			s.spinner, cmd = s.spinner.Update(msg)
			return cmd
		}
		return nil
	}

	// If still loading, update spinner and return
	if s.loading {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return cmd
	}

	// Handle keyboard input based on state
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch s.state {
		case "listing":
			return s.handleListingInput(keyMsg)
		case "confirm_continue":
			return s.handleConfirmContinueInput(keyMsg)
		case "confirm_reset":
			return s.handleConfirmResetInput(keyMsg)
		}
	}

	return tea.Batch(cmds...)
}

// handleListingInput handles keyboard input in the listing state.
func (s *SessionSelectorStep) handleListingInput(keyMsg tea.KeyPressMsg) tea.Cmd {
	switch keyMsg.String() {
	case "up", "k":
		if s.selectedIdx > 0 {
			s.selectedIdx--
			s.scrollList.SetSelected(s.selectedIdx)
			s.scrollList.ScrollToItem(s.selectedIdx)
		}
		return nil

	case "down", "j":
		if s.selectedIdx < len(s.sessions)-1 {
			s.selectedIdx++
			s.scrollList.SetSelected(s.selectedIdx)
			s.scrollList.ScrollToItem(s.selectedIdx)
		}
		return nil

	case "enter":
		// Session selected
		if s.selectedIdx >= 0 && s.selectedIdx < len(s.sessions) {
			selected := &s.sessions[s.selectedIdx]
			s.selectedSession = selected

			if selected.isNew {
				// New session selected - proceed to wizard
				return func() tea.Msg {
					return SessionSelectedMsg{Name: "", IsNew: true}
				}
			}

			// Existing session selected
			if selected.info.Complete {
				// Completed session - ask if they want to continue
				s.state = "confirm_continue"
				s.confirmInput = ""
				return nil
			} else {
				// Incomplete session - ask if they want to reset
				s.state = "confirm_reset"
				s.confirmInput = ""
				return nil
			}
		}
		return nil
	}

	return nil
}

// handleConfirmContinueInput handles keyboard input in the confirm_continue state.
func (s *SessionSelectorStep) handleConfirmContinueInput(keyMsg tea.KeyPressMsg) tea.Cmd {
	switch keyMsg.String() {
	case "y", "Y":
		// Yes - proceed to reset confirmation
		s.state = "confirm_reset"
		s.confirmInput = ""
		return nil

	case "n", "N", "esc":
		// No - go back to listing
		s.state = "listing"
		s.confirmInput = ""
		s.selectedSession = nil
		return nil

	case "enter":
		// Default is Yes
		s.state = "confirm_reset"
		s.confirmInput = ""
		return nil
	}

	return nil
}

// handleConfirmResetInput handles keyboard input in the confirm_reset state.
func (s *SessionSelectorStep) handleConfirmResetInput(keyMsg tea.KeyPressMsg) tea.Cmd {
	switch keyMsg.String() {
	case "y", "Y":
		// Yes - reset and resume session
		return s.resetAndResume()

	case "n", "N":
		// No - don't reset, just resume
		return s.resumeWithoutReset()

	case "esc":
		// Cancel - go back to listing
		s.state = "listing"
		s.confirmInput = ""
		s.selectedSession = nil
		return nil

	case "enter":
		// Default is No
		return s.resumeWithoutReset()
	}

	return nil
}

// View renders the session selector step.
func (s *SessionSelectorStep) View() string {
	var b strings.Builder

	// Show loading state
	if s.loading {
		b.WriteString(s.spinner.View())
		b.WriteString(" Loading sessions...\n")
		return b.String()
	}

	// Show error state
	if s.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		b.WriteString(errorStyle.Render("Error: " + s.error))
		b.WriteString("\n\n")
		hintBar := renderHintBar("r", "retry", "tab", "buttons", "esc", "cancel")
		b.WriteString(hintBar)
		return b.String()
	}

	// Render based on state
	switch s.state {
	case "listing":
		return s.renderListing()
	case "confirm_continue":
		return s.renderConfirmContinue()
	case "confirm_reset":
		return s.renderConfirmReset()
	}

	return b.String()
}

// renderListing renders the session list.
func (s *SessionSelectorStep) renderListing() string {
	var b strings.Builder
	t := theme.Current()

	// Check if we only have the "New Session" entry (empty state)
	if len(s.sessions) == 1 && s.sessions[0].isNew {
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.BgOverlay)).Italic(true)
		b.WriteString(emptyStyle.Render("No existing sessions"))
		b.WriteString("\n\n")
		b.WriteString(s.scrollList.View())
		b.WriteString("\n")
		hintBar := renderHintBar("enter", "new session", "tab", "buttons", "esc", "cancel")
		b.WriteString(hintBar)
		return b.String()
	}

	// Render scroll list (New Session is at top, followed by existing sessions)
	b.WriteString(s.scrollList.View())

	// Add spacing before hint bar
	b.WriteString("\n")

	// Hint bar
	hintBar := renderHintBar(
		"↑↓/j/k", "navigate",
		"enter", "select",
		"tab", "buttons",
		"esc", "cancel",
	)
	b.WriteString(hintBar)

	return b.String()
}

// renderConfirmContinue renders the "continue anyway?" confirmation prompt.
func (s *SessionSelectorStep) renderConfirmContinue() string {
	var b strings.Builder
	t := theme.Current()

	if s.selectedSession == nil {
		return "Error: no session selected"
	}

	// Show selected session name
	b.WriteString("Selected: ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase)).Bold(true).Render(s.selectedSession.info.Name))
	b.WriteString("\n\n")

	// Show confirmation prompt
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	b.WriteString(promptStyle.Render("Session is complete. Continue anyway? [Y/n]"))
	b.WriteString("\n\n")

	// Hint bar
	hintBar := renderHintBar("y", "yes", "n/esc", "no")
	b.WriteString(hintBar)

	return b.String()
}

// renderConfirmReset renders the "reset tasks and notes?" confirmation prompt.
func (s *SessionSelectorStep) renderConfirmReset() string {
	var b strings.Builder
	t := theme.Current()

	if s.selectedSession == nil {
		return "Error: no session selected"
	}

	// Show selected session name
	b.WriteString("Selected: ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgBase)).Bold(true).Render(s.selectedSession.info.Name))
	b.WriteString("\n\n")

	// Show confirmation prompt
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgMuted))
	b.WriteString(promptStyle.Render("Reset all tasks and notes? [y/N]"))
	b.WriteString("\n\n")

	// Hint bar
	hintBar := renderHintBar("y", "yes", "n/enter/esc", "no")
	b.WriteString(hintBar)

	return b.String()
}

// SelectedSession returns the currently selected session name (empty if "New Session" selected).
func (s *SessionSelectorStep) SelectedSession() string {
	if s.selectedIdx >= 0 && s.selectedIdx < len(s.sessions) {
		item := s.sessions[s.selectedIdx]
		if !item.isNew {
			return item.info.Name
		}
	}
	return ""
}

// IsNewSession returns true if "New Session" is selected.
func (s *SessionSelectorStep) IsNewSession() bool {
	if s.selectedIdx >= 0 && s.selectedIdx < len(s.sessions) {
		return s.sessions[s.selectedIdx].isNew
	}
	return false
}

// IsConfirming returns true if the session selector is in a confirmation state.
func (s *SessionSelectorStep) IsConfirming() bool {
	return s.state == "confirm_continue" || s.state == "confirm_reset"
}

// IsReady returns true if the session selector is ready for Next button action.
// Ready means: not loading, no error, in listing state (not confirming).
func (s *SessionSelectorStep) IsReady() bool {
	return !s.loading && s.error == "" && s.state == "listing"
}

// TriggerSelection triggers the currently selected item (same as pressing Enter).
// Returns a command that may emit SessionSelectedMsg or transition to confirmation state.
func (s *SessionSelectorStep) TriggerSelection() tea.Cmd {
	if s.selectedIdx >= 0 && s.selectedIdx < len(s.sessions) {
		selected := &s.sessions[s.selectedIdx]
		s.selectedSession = selected

		if selected.isNew {
			// New session selected - proceed to wizard
			return func() tea.Msg {
				return SessionSelectedMsg{Name: "", IsNew: true}
			}
		}

		// Existing session selected
		if selected.info.Complete {
			// Completed session - ask if they want to continue
			s.state = "confirm_continue"
			s.confirmInput = ""
			return nil
		} else {
			// Incomplete session - ask if they want to reset
			s.state = "confirm_reset"
			s.confirmInput = ""
			return nil
		}
	}
	return nil
}

// ReturnToListing returns to the session listing from a confirmation state.
func (s *SessionSelectorStep) ReturnToListing() {
	s.state = "listing"
	s.confirmInput = ""
	s.selectedSession = nil
}

// SessionsLoadedMsg is sent when sessions are successfully fetched.
type SessionsLoadedMsg struct {
	sessions []session.SessionInfo
}

// SessionsErrorMsg is sent when session fetching fails.
type SessionsErrorMsg struct {
	err error
}

// SessionSelectedMsg is sent when a session is selected.
type SessionSelectedMsg struct {
	Name          string // Session name (empty if IsNew)
	IsNew         bool   // True if "New Session" selected
	ShouldReset   bool   // True if user chose to reset tasks/notes
	PreviousModel string // Model previously used by this session (for pre-selection)
}

// resetAndResume resets the session and returns a command to resume it.
func (s *SessionSelectorStep) resetAndResume() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Capture model before reset (reset purges all events including set_model)
		previousModel := s.selectedSession.info.Model

		// Reset the session (purge all events)
		if err := s.sessionStore.ResetSession(ctx, s.selectedSession.info.Name); err != nil {
			return SessionsErrorMsg{err: err}
		}

		return SessionSelectedMsg{
			Name:          s.selectedSession.info.Name,
			IsNew:         false,
			ShouldReset:   true,
			PreviousModel: previousModel,
		}
	}
}

// resumeWithoutReset resumes the session without resetting.
// If the session was completed, calls SessionRestart() to set Complete = false.
func (s *SessionSelectorStep) resumeWithoutReset() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// If session was complete, restart it
		if s.selectedSession.info.Complete {
			if err := s.sessionStore.SessionRestart(ctx, s.selectedSession.info.Name); err != nil {
				return SessionsErrorMsg{err: err}
			}
		}

		return SessionSelectedMsg{
			Name:          s.selectedSession.info.Name,
			IsNew:         false,
			ShouldReset:   false,
			PreviousModel: s.selectedSession.info.Model,
		}
	}
}

// PreferredHeight returns the preferred height for this step's content.
// This allows the modal to size dynamically based on content.
func (s *SessionSelectorStep) PreferredHeight() int {
	// For confirmation states, return fixed smaller height
	if s.state == "confirm_continue" || s.state == "confirm_reset" {
		// "Selected: name" + blank + "prompt" + blank + hint bar = 5 lines
		return 5
	}

	// For loading state
	if s.loading {
		// "Loading sessions..." = 1 line
		return 1
	}

	// For error state
	if s.error != "" {
		// Error message + blank + hint bar = 3 lines
		return 3
	}

	// For listing state:
	// - Sessions list (number of items, max 20 for reasonable modal size)
	// - blank line + hint bar = 2
	listItems := len(s.sessions)
	if listItems > 20 {
		listItems = 20 // Cap at 20 for scrollable list
	}

	return listItems + 2
}
