package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// OpenTaskModalMsg is sent when a task should be opened in the modal.
type OpenTaskModalMsg struct {
	Task *session.Task
}

// taskScrollItem wraps a task for use in ScrollList.
type taskScrollItem struct {
	task       *session.Task
	isSelected bool
	width      int
	rendered   string
	height     int
}

func (t *taskScrollItem) ID() string {
	return t.task.ID
}

func (t *taskScrollItem) Render(width int) string {
	if t.width != width || t.rendered == "" {
		t.width = width
		t.rendered = t.renderTask()
		t.height = strings.Count(t.rendered, "\n") + 1
	}
	return t.rendered
}

func (t *taskScrollItem) Height() int {
	if t.height == 0 {
		t.Render(t.width)
	}
	return t.height
}

func (t *taskScrollItem) renderTask() string {
	task := t.task
	// Status indicator
	var indicator string
	var indicatorStyle lipgloss.Style
	s := theme.Current().S()

	switch task.Status {
	case "in_progress":
		indicator = "►"
		indicatorStyle = s.StatusInProgress
	case "remaining":
		indicator = "○"
		indicatorStyle = s.StatusRemaining
	case "completed":
		indicator = "✓"
		indicatorStyle = s.StatusCompleted
	case "blocked":
		indicator = "⊘"
		indicatorStyle = s.StatusBlocked
	case "cancelled":
		indicator = "⊗"
		indicatorStyle = s.StatusBlocked
	default:
		indicator = "○"
		indicatorStyle = s.StatusRemaining
	}

	// Truncate content to fit width (leave room for indicator and padding)
	maxContentWidth := t.width - 6 // 2 for indicator+space, 2 padding, 2 for selection arrow
	if maxContentWidth < 10 {
		maxContentWidth = 10
	}

	content := task.Content
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}

	// Build line (selection arrow handled by ScrollList)
	styledIndicator := indicatorStyle.Render(indicator)
	line := fmt.Sprintf(" %s %s", styledIndicator, content)

	return line
}

// noteScrollItem wraps a note for use in ScrollList.
type noteScrollItem struct {
	note       *session.Note
	isSelected bool
	width      int
	rendered   string
	height     int
}

func (n *noteScrollItem) ID() string {
	return n.note.ID
}

func (n *noteScrollItem) Render(width int) string {
	if n.width != width || n.rendered == "" {
		n.width = width
		n.rendered = n.renderNote()
		n.height = strings.Count(n.rendered, "\n") + 1
	}
	return n.rendered
}

func (n *noteScrollItem) Height() int {
	if n.height == 0 {
		n.Render(n.width)
	}
	return n.height
}

func (n *noteScrollItem) renderNote() string {
	note := n.note
	// Type indicator
	var indicator string
	var indicatorStyle lipgloss.Style
	s := theme.Current().S()

	switch note.Type {
	case "learning":
		indicator = "*"
		indicatorStyle = s.StatusCompleted // Green
	case "stuck":
		indicator = "!"
		indicatorStyle = s.StatusBlocked // Red
	case "tip":
		indicator = "›"
		indicatorStyle = s.StatusInProgress // Yellow
	case "decision":
		indicator = "◇"
		indicatorStyle = s.StatusRemaining // Blue
	default:
		indicator = "≡"
		indicatorStyle = s.Muted
	}

	// Truncate content to fit width
	maxContentWidth := n.width - 8
	if maxContentWidth < 10 {
		maxContentWidth = 10
	}

	content := note.Content
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}

	// Build line (selection arrow handled by ScrollList)
	styledIndicator := indicatorStyle.Render(indicator)
	line := fmt.Sprintf(" %s %s", styledIndicator, content)

	return line
}

// Logo constants
const (
	logoHeight = 6 // Height of the logo box including borders and padding
	logoText1  = "▀ ▀█▀ █▀▀ █▀█ ▄▀█ ▀█▀ █▀█"
	logoText2  = "█  █  ██▄ █▀▄ █▀█  █  █▀▄"
)

// Sidebar displays tasks and notes in two sections.
type Sidebar struct {
	state            *session.State
	width            int
	height           int
	tasksScrollList  *ScrollList
	notesScrollList  *ScrollList
	cursor           int            // Selected task index (for keyboard navigation)
	activeTaskID     string         // Currently active task (shown in modal)
	focused          bool           // Deprecated: use tasksFocused/notesFocused instead
	tasksFocused     bool           // Whether the tasks panel has focus (accent border)
	notesFocused     bool           // Whether the notes panel has focus (accent border)
	taskIndex        map[string]int // O(1) lookup: task ID -> index in ordered task list
	noteIndex        map[string]int // O(1) lookup: note ID -> index in state.Notes
	pulse            Pulse
	pulsedTaskIDs    map[string]string // Track task ID -> last status to detect changes
	needsPulse       bool              // Flag indicating pulse should start on next Update
	tasksContentArea uv.Rectangle      // Screen area where task lines are drawn (for mouse hit detection)
	notesContentArea uv.Rectangle      // Screen area where note lines are drawn (for mouse hit detection)
	activeNoteID     string            // Currently active note (shown in modal)
}

// NewSidebar creates a new Sidebar component.
func NewSidebar() *Sidebar {
	return &Sidebar{
		tasksScrollList: NewScrollList(0, 0),
		notesScrollList: NewScrollList(0, 0),
		cursor:          0,
		focused:         false,
		taskIndex:       make(map[string]int),
		noteIndex:       make(map[string]int),
		pulse:           NewPulse(),
		pulsedTaskIDs:   make(map[string]string),
	}
}

// Update handles messages for the sidebar.
func (s *Sidebar) Update(msg tea.Msg) tea.Cmd {
	// Start pulse if needed
	if s.needsPulse && !s.pulse.IsActive() {
		s.needsPulse = false
		return s.pulse.Start()
	}

	switch msg := msg.(type) {
	case PulseMsg:
		// Handle pulse animation (even when not focused)
		return s.pulse.Update(msg)
	case tea.KeyPressMsg:
		if !s.tasksFocused && !s.notesFocused {
			return nil
		}
		return s.handleKeyPress(msg)
	}
	return nil
}

// handleKeyPress handles keyboard input for task navigation and viewport scrolling.
func (s *Sidebar) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	tasks := s.getTasks()

	switch msg.String() {
	case "j", "down":
		// Move cursor down
		if len(tasks) > 0 && s.cursor < len(tasks)-1 {
			s.cursor++
			s.updateContent() // Rebuild content with new cursor position
			s.tasksScrollList.ScrollToItem(s.cursor)
		}
		return nil
	case "k", "up":
		// Move cursor up
		if s.cursor > 0 {
			s.cursor--
			s.updateContent() // Rebuild content with new cursor position
			s.tasksScrollList.ScrollToItem(s.cursor)
		}
		return nil
	case "enter":
		// Return OpenTaskModalMsg for the selected task
		if len(tasks) > 0 && s.cursor < len(tasks) {
			return func() tea.Msg {
				return OpenTaskModalMsg{Task: tasks[s.cursor]}
			}
		}
		return nil
	}

	// Delegate to ScrollLists for scrolling (pgup/pgdown, etc.)
	var cmds []tea.Cmd

	var cmd tea.Cmd
	cmd = s.tasksScrollList.Update(msg)
	cmds = append(cmds, cmd)

	cmd = s.notesScrollList.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

// drawTasksSection renders the tasks section with header and viewport content.
func (s *Sidebar) drawTasksSection(scr uv.Screen, area uv.Rectangle) {
	// Apply pulse effect to title if task status changed
	title := "Tasks"
	if s.pulse.IsActive() {
		// Add visual indicator when pulse is active
		intensity := s.pulse.Intensity()
		if intensity > 0.5 {
			title = "Tasks ●" // Add dot indicator during pulse
		}
	}

	// Draw panel with "Tasks" title
	inner := DrawPanel(scr, area, title, s.tasksFocused)

	// Store the inner content area for coordinate-based mouse hit detection
	s.tasksContentArea = inner

	// Render ScrollList content
	content := s.tasksScrollList.View()
	DrawText(scr, inner, content)

	// Draw scroll indicator (always visible)
	pct := s.tasksScrollList.ScrollPercent()
	indicator := fmt.Sprintf(" %d%% ", int(pct*100))
	indicatorArea := uv.Rect(
		area.Max.X-len(indicator),
		area.Max.Y-1,
		len(indicator),
		1,
	)
	styledIndicator := theme.Current().S().ScrollIndicator.Render(indicator)
	uv.NewStyledString(styledIndicator).Draw(scr, indicatorArea)
}

// drawNotesSection renders the notes section with header and viewport content.
func (s *Sidebar) drawNotesSection(scr uv.Screen, area uv.Rectangle) {
	// Draw panel with "Notes" title
	inner := DrawPanel(scr, area, "Notes", s.notesFocused)

	// Store the inner content area for coordinate-based mouse hit detection
	s.notesContentArea = inner

	// Render ScrollList content
	content := s.notesScrollList.View()
	DrawText(scr, inner, content)

	// Draw scroll indicator (always visible)
	pct := s.notesScrollList.ScrollPercent()
	indicator := fmt.Sprintf(" %d%% ", int(pct*100))
	indicatorArea := uv.Rect(
		area.Max.X-len(indicator),
		area.Max.Y-1,
		len(indicator),
		1,
	)
	styledIndicator := theme.Current().S().ScrollIndicator.Render(indicator)
	uv.NewStyledString(styledIndicator).Draw(scr, indicatorArea)
}

// drawLogo renders the logo box at the top of the sidebar.
func (s *Sidebar) drawLogo(scr uv.Screen, area uv.Rectangle) {
	t := theme.Current()
	width := area.Dx()
	if width < 10 {
		return
	}

	// Calculate inner width (for content inside border)
	innerWidth := width - 2 // Account for left/right border chars

	// Build the box borders
	topBorder := "╭" + strings.Repeat("─", innerWidth) + "╮"
	bottomBorder := "╰" + strings.Repeat("─", innerWidth) + "╯"
	emptyLine := "│" + strings.Repeat(" ", innerWidth) + "│"

	// Style for borders (primary color)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary))

	// Render logo text with gradient
	renderGradientLine := func(text string) string {
		textLen := len([]rune(text))
		padding := (innerWidth - textLen) / 2
		if padding < 0 {
			padding = 0
		}
		rightPadding := innerWidth - textLen - padding
		if rightPadding < 0 {
			rightPadding = 0
		}

		// Build the line with gradient on the text
		var result strings.Builder
		result.WriteString(borderStyle.Render("│"))
		result.WriteString(strings.Repeat(" ", padding))

		// Apply gradient to each character
		runes := []rune(text)
		for i, r := range runes {
			pos := float64(i) / float64(len(runes)-1)
			if len(runes) == 1 {
				pos = 0
			}
			color := theme.InterpolateColor(t.Primary, t.Secondary, pos)
			charStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			result.WriteString(charStyle.Render(string(r)))
		}

		result.WriteString(strings.Repeat(" ", rightPadding))
		result.WriteString(borderStyle.Render("│"))
		return result.String()
	}

	// Draw each line
	type lineData struct {
		content    string
		isGradient bool
	}
	lines := []lineData{
		{topBorder, false},
		{emptyLine, false},
		{logoText1, true},
		{logoText2, true},
		{emptyLine, false},
		{bottomBorder, false},
	}

	for i, line := range lines {
		if area.Min.Y+i >= area.Max.Y {
			break
		}
		lineArea := uv.Rect(area.Min.X, area.Min.Y+i, width, 1)
		var styledLine string
		if line.isGradient {
			styledLine = renderGradientLine(line.content)
		} else {
			styledLine = borderStyle.Render(line.content)
		}
		uv.NewStyledString(styledLine).Draw(scr, lineArea)
	}
}

// getTasks returns all tasks ordered by ID.
func (s *Sidebar) getTasks() []*session.Task {
	if s.state == nil {
		return nil
	}

	tasks := make([]*session.Task, 0, len(s.state.Tasks))
	for _, task := range s.state.Tasks {
		tasks = append(tasks, task)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	return tasks
}

// Draw renders the sidebar to the screen buffer with logo, tasks, and notes sections.
func (s *Sidebar) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Guard against zero dimensions
	if area.Dx() < 10 || area.Dy() < 5 {
		return nil
	}

	// Split area vertically: Logo (fixed) | Tasks (55%) | Notes (45%)
	// First split off the logo area
	logoArea, remainder := uv.SplitVertical(area, uv.Fixed(logoHeight))

	// Then split remaining space between tasks and notes
	remainingHeight := remainder.Dy()
	tasksHeight := int(float64(remainingHeight) * 0.55)
	if tasksHeight < 3 {
		tasksHeight = 3
	}

	tasksArea, notesArea := uv.SplitVertical(remainder, uv.Fixed(tasksHeight))

	// Draw Logo section
	s.drawLogo(scr, logoArea)

	// Draw Tasks section
	s.drawTasksSection(scr, tasksArea)

	// Draw Notes section
	s.drawNotesSection(scr, notesArea)

	return nil
}

// TaskAtPosition returns the task at the given screen coordinates, or nil if none.
// Uses coordinate-based hit detection against the tasks content area.
func (s *Sidebar) TaskAtPosition(x, y int) *session.Task {
	area := s.tasksContentArea
	// Check if the click is within the tasks content area
	if x < area.Min.X || x >= area.Max.X || y < area.Min.Y || y >= area.Max.Y {
		return nil
	}

	// Calculate which task line was clicked, accounting for scroll offset
	// For now, use simple line index since tasks are one line each
	lineIndex := (y - area.Min.Y) + s.tasksScrollList.offsetIdx

	tasks := s.getTasks()
	if lineIndex < 0 || lineIndex >= len(tasks) {
		return nil
	}

	return tasks[lineIndex]
}

// NoteAtPosition returns the note at the given screen coordinates, or nil if none.
// Uses coordinate-based hit detection against the notes content area.
func (s *Sidebar) NoteAtPosition(x, y int) *session.Note {
	area := s.notesContentArea
	// Check if the click is within the notes content area
	if x < area.Min.X || x >= area.Max.X || y < area.Min.Y || y >= area.Max.Y {
		return nil
	}

	// Calculate which note line was clicked, accounting for scroll offset
	// For now, use simple line index since notes are one line each
	lineIndex := (y - area.Min.Y) + s.notesScrollList.offsetIdx

	// Notes display the last 10 notes
	if s.state == nil || len(s.state.Notes) == 0 {
		return nil
	}
	startIdx := 0
	if len(s.state.Notes) > 10 {
		startIdx = len(s.state.Notes) - 10
	}
	notes := s.state.Notes[startIdx:]

	if lineIndex < 0 || lineIndex >= len(notes) {
		return nil
	}

	return notes[lineIndex]
}

// SetActiveNote marks a note as active (highlighted) and rebuilds content.
func (s *Sidebar) SetActiveNote(id string) {
	s.activeNoteID = id
	s.updateContent()
}

// ClearActiveNote removes the active note highlight and rebuilds content.
func (s *Sidebar) ClearActiveNote() {
	s.activeNoteID = ""
	s.updateContent()
}

// SetActiveTask marks a task as active (highlighted) and rebuilds content.
func (s *Sidebar) SetActiveTask(id string) {
	s.activeTaskID = id
	s.updateContent()
}

// ClearActiveTask removes the active task highlight and rebuilds content.
func (s *Sidebar) ClearActiveTask() {
	s.activeTaskID = ""
	s.updateContent()
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

	// Calculate section heights (Logo fixed, then Tasks 55%, Notes 45% of remainder)
	remainingHeight := height - logoHeight
	if remainingHeight < 5 {
		remainingHeight = 5
	}

	tasksHeight := int(float64(remainingHeight) * 0.55)
	if tasksHeight < 3 {
		tasksHeight = 3
	}
	notesHeight := remainingHeight - tasksHeight
	if notesHeight < 2 {
		notesHeight = 2
		tasksHeight = remainingHeight - notesHeight
	}

	// Account for panel header (1 line each)
	s.tasksScrollList.SetWidth(width - 4)
	s.tasksScrollList.SetHeight(tasksHeight - 1)
	s.notesScrollList.SetWidth(width - 4)
	s.notesScrollList.SetHeight(notesHeight - 1)
}

// SetState updates the sidebar with new session state.
func (s *Sidebar) SetState(state *session.State) {
	oldState := s.state
	s.state = state

	// Detect task status changes and mark for pulse
	if state != nil {
		statusChanged := false

		// Check for new or changed task statuses
		for id, task := range state.Tasks {
			oldStatus, existed := s.pulsedTaskIDs[id]

			// If task is new or status changed, mark for pulse
			// Only trigger pulse if we had a previous state (not initial load)
			if !existed || oldStatus != task.Status {
				s.pulsedTaskIDs[id] = task.Status
				if oldState != nil {
					statusChanged = true
				}
			}
		}

		// Set flag to start pulse on next Update
		if statusChanged {
			s.needsPulse = true
		}
	}

	s.updateContent()

	// Clamp cursor to valid range after state update
	tasks := s.getTasks()
	if len(tasks) == 0 {
		s.cursor = 0
	} else if s.cursor >= len(tasks) {
		s.cursor = len(tasks) - 1
	}
}

// rebuildIndex rebuilds the ID-based lookup indices for tasks and notes.
// This provides O(1) lookups by ID.
func (s *Sidebar) rebuildIndex() {
	if s.state == nil {
		s.taskIndex = make(map[string]int)
		s.noteIndex = make(map[string]int)
		return
	}

	// Rebuild task index: map task ID -> position in ordered task list
	// This provides O(1) lookup to find a task's position in the display order
	tasks := s.getTasks()
	s.taskIndex = make(map[string]int, len(tasks))
	for idx, task := range tasks {
		s.taskIndex[task.ID] = idx
	}

	// Rebuild note index: map note ID -> position in state.Notes slice
	s.noteIndex = make(map[string]int, len(s.state.Notes))
	for idx := range s.state.Notes {
		s.noteIndex[s.state.Notes[idx].ID] = idx
	}
}

// updateContent rebuilds ScrollList content from state.
func (s *Sidebar) updateContent() {
	if s.state == nil {
		return
	}

	// Rebuild indices for O(1) lookups
	s.rebuildIndex()

	// Update tasks ScrollList
	tasks := s.getTasks()
	taskItems := make([]ScrollItem, 0, len(tasks))
	for idx, task := range tasks {
		isSelected := (s.focused && idx == s.cursor) || task.ID == s.activeTaskID
		taskItems = append(taskItems, &taskScrollItem{
			task:       task,
			isSelected: isSelected,
			width:      s.tasksScrollList.width,
		})
	}
	s.tasksScrollList.SetItems(taskItems)
	// Set selected index for cursor highlighting
	if s.focused && s.cursor >= 0 && s.cursor < len(tasks) {
		s.tasksScrollList.SetSelected(s.cursor)
	} else {
		s.tasksScrollList.SetSelected(-1)
	}

	// Update notes ScrollList
	if len(s.state.Notes) == 0 {
		s.notesScrollList.SetItems([]ScrollItem{})
		return
	}

	// Show recent notes (last 10)
	startIdx := 0
	if len(s.state.Notes) > 10 {
		startIdx = len(s.state.Notes) - 10
	}
	notes := s.state.Notes[startIdx:]

	noteItems := make([]ScrollItem, 0, len(notes))
	for _, note := range notes {
		isSelected := note.ID == s.activeNoteID
		noteItems = append(noteItems, &noteScrollItem{
			note:       note,
			isSelected: isSelected,
			width:      s.notesScrollList.width,
		})
	}
	s.notesScrollList.SetItems(noteItems)
}

// Legacy methods for backward compatibility
func (s *Sidebar) SetFocused(focused bool)                  { s.SetFocus(focused) }
func (s *Sidebar) UpdateSize(width, height int) tea.Cmd     { s.SetSize(width, height); return nil }
func (s *Sidebar) UpdateState(state *session.State) tea.Cmd { s.SetState(state); return nil }

// GetTaskByID returns a task by ID using O(1) lookup via taskIndex.
// Returns nil if task not found.
func (s *Sidebar) GetTaskByID(id string) *session.Task {
	if s.state == nil {
		return nil
	}
	// Use taskIndex to find position in ordered task list
	idx, ok := s.taskIndex[id]
	if !ok {
		return nil
	}
	tasks := s.getTasks()
	if idx < 0 || idx >= len(tasks) {
		return nil
	}
	return tasks[idx]
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

// SetTasksScrollFocused sets the focus state of the tasks ScrollList and panel border.
// When focused, the tasks ScrollList will handle keyboard scroll events and the panel will show an accent border.
func (s *Sidebar) SetTasksScrollFocused(focused bool) {
	s.tasksFocused = focused
	if s.tasksScrollList != nil {
		s.tasksScrollList.SetFocused(focused)
	}
}

// SetNotesScrollFocused sets the focus state of the notes ScrollList and panel border.
// When focused, the notes ScrollList will handle keyboard scroll events and the panel will show an accent border.
func (s *Sidebar) SetNotesScrollFocused(focused bool) {
	s.notesFocused = focused
	if s.notesScrollList != nil {
		s.notesScrollList.SetFocused(focused)
	}
}

// IsTasksArea checks if the given screen coordinates fall within the tasks content area.
func (s *Sidebar) IsTasksArea(x, y int) bool {
	return x >= s.tasksContentArea.Min.X && x < s.tasksContentArea.Max.X &&
		y >= s.tasksContentArea.Min.Y && y < s.tasksContentArea.Max.Y
}

// IsNotesArea checks if the given screen coordinates fall within the notes content area.
func (s *Sidebar) IsNotesArea(x, y int) bool {
	return x >= s.notesContentArea.Min.X && x < s.notesContentArea.Max.X &&
		y >= s.notesContentArea.Min.Y && y < s.notesContentArea.Max.Y
}

// ScrollTasks scrolls the tasks viewport by the given number of lines.
func (s *Sidebar) ScrollTasks(lines int) {
	if s.tasksScrollList != nil {
		s.tasksScrollList.ScrollBy(lines)
	}
}

// ScrollNotes scrolls the notes viewport by the given number of lines.
func (s *Sidebar) ScrollNotes(lines int) {
	if s.notesScrollList != nil {
		s.notesScrollList.ScrollBy(lines)
	}
}

// Compile-time interface check
var _ FocusableComponent = (*Sidebar)(nil)
