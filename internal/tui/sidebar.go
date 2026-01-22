package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// Sidebar displays tasks and notes in two sections.
type Sidebar struct {
	state         *session.State
	width         int
	height        int
	tasksViewport viewport.Model
	notesViewport viewport.Model
	cursor        int // Selected task index (for future interactivity)
	focused       bool
	noteIndex     map[string]int // O(1) lookup: note ID -> index in state.Notes
}

// NewSidebar creates a new Sidebar component.
func NewSidebar() *Sidebar {
	return &Sidebar{
		tasksViewport: viewport.New(),
		notesViewport: viewport.New(),
		cursor:        0,
		focused:       false,
		noteIndex:     make(map[string]int),
	}
}

// Update handles messages for the sidebar.
func (s *Sidebar) Update(msg tea.Msg) tea.Cmd {
	if !s.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return s.handleKeyPress(msg)
	}
	return nil
}

// handleKeyPress handles keyboard input for viewport scrolling.
func (s *Sidebar) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	// Delegate to viewports for scrolling
	var cmds []tea.Cmd

	var cmd tea.Cmd
	s.tasksViewport, cmd = s.tasksViewport.Update(msg)
	cmds = append(cmds, cmd)

	s.notesViewport, cmd = s.notesViewport.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

// drawTasksSection renders the tasks section with header and viewport content.
func (s *Sidebar) drawTasksSection(scr uv.Screen, area uv.Rectangle) {
	// Draw panel with "Tasks" title
	inner := DrawPanel(scr, area, "Tasks", s.focused)

	// Render viewport content
	content := s.tasksViewport.View()
	DrawText(scr, inner, content)

	// Draw scroll indicator if needed
	if s.tasksViewport.TotalLineCount() > s.tasksViewport.Height() {
		pct := s.tasksViewport.ScrollPercent()
		indicator := fmt.Sprintf(" %d%% ", int(pct*100))
		indicatorArea := uv.Rect(
			area.Max.X-len(indicator)-1,
			area.Max.Y-1,
			len(indicator),
			1,
		)
		style := lipgloss.NewStyle().Background(colorBgSubtle).Foreground(colorMuted)
		DrawStyled(scr, indicatorArea, style, indicator)
	}
}

// drawNotesSection renders the notes section with header and viewport content.
func (s *Sidebar) drawNotesSection(scr uv.Screen, area uv.Rectangle) {
	// Draw panel with "Notes" title
	inner := DrawPanel(scr, area, "Notes", false) // Notes section never focused

	// Render viewport content
	content := s.notesViewport.View()
	DrawText(scr, inner, content)

	// Draw scroll indicator if needed
	if s.notesViewport.TotalLineCount() > s.notesViewport.Height() {
		pct := s.notesViewport.ScrollPercent()
		indicator := fmt.Sprintf(" %d%% ", int(pct*100))
		indicatorArea := uv.Rect(
			area.Max.X-len(indicator)-1,
			area.Max.Y-1,
			len(indicator),
			1,
		)
		style := lipgloss.NewStyle().Background(colorBgSubtle).Foreground(colorMuted)
		DrawStyled(scr, indicatorArea, style, indicator)
	}
}

// getTasks returns all tasks in display order (in_progress first, then remaining, blocked, completed).
func (s *Sidebar) getTasks() []*session.Task {
	if s.state == nil {
		return nil
	}

	var inProgress, remaining, blocked, completed []*session.Task
	for _, task := range s.state.Tasks {
		switch task.Status {
		case "in_progress":
			inProgress = append(inProgress, task)
		case "remaining":
			remaining = append(remaining, task)
		case "blocked":
			blocked = append(blocked, task)
		case "completed":
			completed = append(completed, task)
		}
	}

	// Concatenate in order
	var tasks []*session.Task
	tasks = append(tasks, inProgress...)
	tasks = append(tasks, remaining...)
	tasks = append(tasks, blocked...)
	tasks = append(tasks, completed...)
	return tasks
}

// Draw renders the sidebar to the screen buffer with tasks and notes sections.
func (s *Sidebar) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Guard against zero dimensions
	if area.Dx() < 10 || area.Dy() < 5 {
		return nil
	}

	// Split area vertically: Tasks (60%) | Notes (40%)
	tasksHeight := int(float64(area.Dy()) * 0.6)
	if tasksHeight < 3 {
		tasksHeight = 3
	}

	tasksArea, notesArea := uv.SplitVertical(area, uv.Fixed(tasksHeight))

	// Draw Tasks section
	s.drawTasksSection(scr, tasksArea)

	// Draw Notes section
	s.drawNotesSection(scr, notesArea)

	return nil
}

// buildTasksContent builds the content string for the tasks viewport.
func (s *Sidebar) buildTasksContent() string {
	tasks := s.getTasks()
	if len(tasks) == 0 {
		return styleDim.Render("  No tasks")
	}

	var lines []string
	for _, task := range tasks {
		line := s.renderTask(task)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderTask renders a single task line.
func (s *Sidebar) renderTask(task *session.Task) string {
	// Status indicator
	var indicator string
	var indicatorStyle lipgloss.Style

	switch task.Status {
	case "in_progress":
		indicator = "►"
		indicatorStyle = styleStatusInProgress
	case "remaining":
		indicator = "○"
		indicatorStyle = styleStatusRemaining
	case "completed":
		indicator = "✓"
		indicatorStyle = styleStatusCompleted
	case "blocked":
		indicator = "⊘"
		indicatorStyle = styleStatusBlocked
	default:
		indicator = "○"
		indicatorStyle = styleStatusRemaining
	}

	// Truncate content to fit width (leave room for indicator and padding)
	maxContentWidth := s.width - 6 // 2 for border, 2 for indicator+space, 2 padding
	if maxContentWidth < 10 {
		maxContentWidth = 10
	}

	content := task.Content
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}

	// Build line
	styledIndicator := indicatorStyle.Render(indicator)
	line := fmt.Sprintf(" %s %s", styledIndicator, content)

	return line
}

// SetFocus sets whether the sidebar has keyboard focus.
func (s *Sidebar) SetFocus(focused bool) {
	s.focused = focused
}

// IsFocused returns whether the sidebar has keyboard focus.
func (s *Sidebar) IsFocused() bool {
	return s.focused
}

// SetSize updates the sidebar dimensions and viewport sizes.
func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height

	// Calculate section heights (Tasks 60%, Notes 40%)
	tasksHeight := int(float64(height) * 0.6)
	if tasksHeight < 3 {
		tasksHeight = 3
	}
	notesHeight := height - tasksHeight
	if notesHeight < 2 {
		notesHeight = 2
		tasksHeight = height - notesHeight
	}

	// Account for borders and headers (2 chars each side, 2 lines for header/border)
	s.tasksViewport.SetWidth(width - 4)
	s.tasksViewport.SetHeight(tasksHeight - 4)
	s.notesViewport.SetWidth(width - 4)
	s.notesViewport.SetHeight(notesHeight - 4)
}

// SetState updates the sidebar with new session state.
func (s *Sidebar) SetState(state *session.State) {
	s.state = state
	s.updateContent()
}

// rebuildIndex rebuilds the ID-based lookup index for notes.
// This provides O(1) lookups for notes by ID.
// Note: Tasks are already stored as a map in state.Tasks (ID -> Task),
// so they already have O(1) lookup and don't need an additional index.
func (s *Sidebar) rebuildIndex() {
	if s.state == nil {
		s.noteIndex = make(map[string]int)
		return
	}

	// Rebuild note index: map note ID -> position in state.Notes slice
	s.noteIndex = make(map[string]int, len(s.state.Notes))
	for idx := range s.state.Notes {
		s.noteIndex[s.state.Notes[idx].ID] = idx
	}
}

// updateContent rebuilds viewport content from state.
func (s *Sidebar) updateContent() {
	if s.state == nil {
		return
	}

	// Rebuild indices for O(1) lookups
	s.rebuildIndex()

	// Update tasks viewport
	s.tasksViewport.SetContent(s.buildTasksContent())

	// Update notes viewport
	s.notesViewport.SetContent(s.buildNotesContent())
}

// buildNotesContent builds the content string for the notes viewport.
func (s *Sidebar) buildNotesContent() string {
	if len(s.state.Notes) == 0 {
		return styleDim.Render("  No notes")
	}

	var lines []string
	// Show recent notes (last 10)
	startIdx := 0
	if len(s.state.Notes) > 10 {
		startIdx = len(s.state.Notes) - 10
	}

	for _, note := range s.state.Notes[startIdx:] {
		line := s.renderNote(note)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderNote renders a single note line with type indicator.
func (s *Sidebar) renderNote(note *session.Note) string {
	// Type indicator
	var indicator string
	var indicatorStyle lipgloss.Style

	switch note.Type {
	case "learning":
		indicator = "💡"
		indicatorStyle = styleStatusCompleted // Green-ish
	case "stuck":
		indicator = "🚫"
		indicatorStyle = styleStatusBlocked // Red
	case "tip":
		indicator = "💬"
		indicatorStyle = styleStatusInProgress // Yellow
	case "decision":
		indicator = "⚡"
		indicatorStyle = styleStatusRemaining // Blue
	default:
		indicator = "📝"
		indicatorStyle = styleDim
	}

	// Truncate content to fit width
	maxContentWidth := s.width - 8
	if maxContentWidth < 10 {
		maxContentWidth = 10
	}

	content := note.Content
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}

	// Build line
	styledIndicator := indicatorStyle.Render(indicator)
	line := fmt.Sprintf(" %s %s", styledIndicator, content)

	return line
}

// Render provides legacy string-based rendering for backward compatibility.
// This method will be removed in Phase 12 once App is refactored to use Screen/Draw pattern.
func (s *Sidebar) Render() string {
	// Guard against zero dimensions
	if s.width < 10 || s.height < 5 {
		return ""
	}

	// Calculate section heights (Tasks 60%, Notes 40%)
	tasksHeight := int(float64(s.height) * 0.6)
	if tasksHeight < 3 {
		tasksHeight = 3
	}
	notesHeight := s.height - tasksHeight
	if notesHeight < 2 {
		notesHeight = 2
		tasksHeight = s.height - notesHeight
	}

	// Render tasks section
	tasksHeader := styleSidebarHeader.Width(s.width - 2).Render("Tasks")
	tasksContent := s.tasksViewport.View()
	tasksSection := lipgloss.JoinVertical(lipgloss.Left, tasksHeader, tasksContent)
	tasksBox := styleSidebarBorder.Width(s.width).Height(tasksHeight).Render(tasksSection)

	// Render notes section
	notesHeader := styleSidebarHeader.Width(s.width - 2).Render("Notes")
	notesContent := s.notesViewport.View()
	notesSection := lipgloss.JoinVertical(lipgloss.Left, notesHeader, notesContent)
	notesBox := styleSidebarBorder.Width(s.width).Height(notesHeight).Render(notesSection)

	// Join sections vertically
	return lipgloss.JoinVertical(lipgloss.Left, tasksBox, notesBox)
}

// Legacy methods for backward compatibility
func (s *Sidebar) SetFocused(focused bool)                  { s.SetFocus(focused) }
func (s *Sidebar) UpdateSize(width, height int) tea.Cmd     { s.SetSize(width, height); return nil }
func (s *Sidebar) UpdateState(state *session.State) tea.Cmd { s.SetState(state); return nil }

// GetTaskByID returns a task by ID using O(1) lookup.
// Returns nil if task not found.
func (s *Sidebar) GetTaskByID(id string) *session.Task {
	if s.state == nil {
		return nil
	}
	// state.Tasks is already a map[string]*Task, so this is O(1)
	return s.state.Tasks[id]
}

// GetNoteByID returns a note by ID using O(1) lookup.
// Returns nil if note not found.
func (s *Sidebar) GetNoteByID(id string) *session.Note {
	if s.state == nil {
		return nil
	}
	idx, ok := s.noteIndex[id]
	if !ok || idx < 0 || idx >= len(s.state.Notes) {
		return nil
	}
	return s.state.Notes[idx]
}

// Compile-time interface check
var _ FocusableComponent = (*Sidebar)(nil)

// Sidebar styles (used by legacy Render method, will be removed in Phase 12)
var (
	styleSidebarBorder = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), true, true, true, false). // No left border
				BorderForeground(colorMuted)

	styleSidebarHeader = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorMuted).
				PaddingLeft(1)
)
