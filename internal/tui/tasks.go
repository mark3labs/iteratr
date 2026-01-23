package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/iteratr/internal/session"
)

// TaskList displays tasks grouped by status with filtering and navigation.
type TaskList struct {
	state        *session.State
	width        int
	height       int
	filterStatus string // "all", "remaining", "in_progress", "completed", "blocked"
	cursor       int    // Current selected task index
	scrollOffset int    // For scrolling when list exceeds screen height
}

// NewTaskList creates a new TaskList component.
func NewTaskList() *TaskList {
	return &TaskList{
		filterStatus: "all",
		cursor:       0,
		scrollOffset: 0,
	}
}

// Update handles messages for the task list.
func (t *TaskList) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return t.handleKeyPress(msg)
	}
	return nil
}

// handleKeyPress handles keyboard input for task list navigation and filtering.
func (t *TaskList) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	tasks := t.getFilteredTasks()
	maxIndex := len(tasks) - 1

	switch msg.String() {
	case "j", "down":
		// Move cursor down
		if t.cursor < maxIndex {
			t.cursor++
			t.adjustScroll()
		}
	case "k", "up":
		// Move cursor up
		if t.cursor > 0 {
			t.cursor--
			t.adjustScroll()
		}
	case "g":
		// Go to top
		t.cursor = 0
		t.scrollOffset = 0
	case "G":
		// Go to bottom
		t.cursor = maxIndex
		t.adjustScroll()
	case "f":
		// Cycle through filter statuses
		t.cycleFilter()
		// Reset cursor when filter changes
		t.cursor = 0
		t.scrollOffset = 0
	}

	return nil
}

// cycleFilter cycles through available filter statuses.
func (t *TaskList) cycleFilter() {
	filters := []string{"all", "remaining", "in_progress", "completed", "blocked"}
	for i, f := range filters {
		if f == t.filterStatus {
			t.filterStatus = filters[(i+1)%len(filters)]
			return
		}
	}
}

// adjustScroll adjusts the scroll offset to keep the cursor visible.
func (t *TaskList) adjustScroll() {
	// Simple implementation: keep cursor in view
	// Assumes roughly 3 lines per task (bullet + content)
	visibleLines := t.height / 3
	if t.cursor >= t.scrollOffset+visibleLines {
		t.scrollOffset = t.cursor - visibleLines + 1
	} else if t.cursor < t.scrollOffset {
		t.scrollOffset = t.cursor
	}
}

// statusOrder defines the sort order for task statuses.
// Order: completed, in_progress, remaining, blocked
var statusOrder = map[string]int{
	"completed":   0,
	"in_progress": 1,
	"remaining":   2,
	"blocked":     3,
}

// getFilteredTasks returns tasks matching the current filter, sorted by status and ID.
func (t *TaskList) getFilteredTasks() []*session.Task {
	if t.state == nil {
		return nil
	}

	var filtered []*session.Task
	for _, task := range t.state.Tasks {
		if t.filterStatus == "all" || task.Status == t.filterStatus {
			filtered = append(filtered, task)
		}
	}

	// Sort by status order, then by ID
	sort.Slice(filtered, func(i, j int) bool {
		statusI := statusOrder[filtered[i].Status]
		statusJ := statusOrder[filtered[j].Status]
		if statusI != statusJ {
			return statusI < statusJ
		}
		return filtered[i].ID < filtered[j].ID
	})

	return filtered
}

// Render returns the task list view as a string.
func (t *TaskList) Render() string {
	if t.state == nil {
		return styleEmptyState.Render("No session loaded")
	}

	// Add filter indicator
	filterLabel := t.getFilterLabel()
	filterBar := styleSubtitle.Render(fmt.Sprintf("Filter: %s", filterLabel)) +
		styleDim.Render(" (f to cycle, j/k to navigate)")

	// Get filtered tasks
	filteredTasks := t.getFilteredTasks()
	if len(filteredTasks) == 0 {
		return filterBar + "\n\n" + styleEmptyState.Render("No tasks match current filter")
	}

	// Build task list with cursor
	var sections []string
	sections = append(sections, filterBar)

	// If showing all, group by status
	if t.filterStatus == "all" {
		sections = append(sections, t.renderAllGroups(filteredTasks))
	} else {
		// Show flat list for filtered view
		sections = append(sections, t.renderFlatList(filteredTasks))
	}

	return strings.Join(sections, "\n\n")
}

// getFilterLabel returns a human-readable filter label.
func (t *TaskList) getFilterLabel() string {
	switch t.filterStatus {
	case "all":
		return "All Tasks"
	case "remaining":
		return "Remaining"
	case "in_progress":
		return "In Progress"
	case "completed":
		return "Completed"
	case "blocked":
		return "Blocked"
	default:
		return "All Tasks"
	}
}

// renderAllGroups renders tasks grouped by status (when filter is "all").
func (t *TaskList) renderAllGroups(tasks []*session.Task) string {
	// Group tasks by status
	remaining := []*session.Task{}
	inProgress := []*session.Task{}
	completed := []*session.Task{}
	blocked := []*session.Task{}

	for _, task := range tasks {
		switch task.Status {
		case "remaining":
			remaining = append(remaining, task)
		case "in_progress":
			inProgress = append(inProgress, task)
		case "completed":
			completed = append(completed, task)
		case "blocked":
			blocked = append(blocked, task)
		}
	}

	var sections []string

	// Sort tasks within each group by ID
	sortByID := func(tasks []*session.Task) {
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].ID < tasks[j].ID
		})
	}
	sortByID(completed)
	sortByID(inProgress)
	sortByID(remaining)
	sortByID(blocked)

	// Track global index for cursor highlighting
	globalIdx := 0

	// Render each status group in order: completed, in_progress, remaining, blocked
	if len(completed) > 0 {
		sections = append(sections, t.renderGroup("COMPLETED", completed, styleStatusCompleted, &globalIdx))
	}
	if len(inProgress) > 0 {
		sections = append(sections, t.renderGroup("IN PROGRESS", inProgress, styleStatusInProgress, &globalIdx))
	}
	if len(remaining) > 0 {
		sections = append(sections, t.renderGroup("REMAINING", remaining, styleStatusRemaining, &globalIdx))
	}
	if len(blocked) > 0 {
		sections = append(sections, t.renderGroup("BLOCKED", blocked, styleStatusBlocked, &globalIdx))
	}

	if len(sections) == 0 {
		return styleEmptyState.Render("No tasks yet")
	}

	return strings.Join(sections, "\n\n")
}

// renderFlatList renders a flat list of tasks (when filtered by status).
func (t *TaskList) renderFlatList(tasks []*session.Task) string {
	var taskLines []string
	statusStyle := t.getStatusStyle(t.filterStatus)

	for i, task := range tasks {
		isSelected := i == t.cursor
		taskLines = append(taskLines, t.renderTask(task, statusStyle, isSelected))
	}

	return strings.Join(taskLines, "\n")
}

// getStatusStyle returns the appropriate style for a status.
func (t *TaskList) getStatusStyle(status string) lipgloss.Style {
	switch status {
	case "remaining":
		return styleStatusRemaining
	case "in_progress":
		return styleStatusInProgress
	case "completed":
		return styleStatusCompleted
	case "blocked":
		return styleStatusBlocked
	default:
		return styleStatusRemaining
	}
}

// renderGroup renders a group of tasks with a status header.
func (t *TaskList) renderGroup(title string, tasks []*session.Task, statusStyle lipgloss.Style, globalIdx *int) string {
	// Render header with count
	header := styleGroupHeader.Render(fmt.Sprintf("%s (%d)", title, len(tasks)))

	// Render tasks
	var taskLines []string
	for _, task := range tasks {
		isSelected := *globalIdx == t.cursor
		taskLines = append(taskLines, t.renderTask(task, statusStyle, isSelected))
		*globalIdx++
	}

	return header + "\n" + strings.Join(taskLines, "\n")
}

// renderTask renders a single task with ID prefix and content.
func (t *TaskList) renderTask(task *session.Task, statusStyle lipgloss.Style, isSelected bool) string {
	idPrefix := task.ID

	// Render ID and content
	id := styleTaskID.Render(fmt.Sprintf("[%s]", idPrefix))
	content := styleTaskContent.Render(task.Content)

	// Combine with status indicator
	bullet := statusStyle.Render("●")
	line := fmt.Sprintf("  %s %s %s", bullet, id, content)

	// Highlight if selected
	if isSelected {
		// Add selection indicator and apply selected style
		line = styleTaskSelected.Render("▶ " + strings.TrimPrefix(line, "  "))
	}

	return line
}

// UpdateSize updates the task list dimensions.
func (t *TaskList) UpdateSize(width, height int) tea.Cmd {
	t.width = width
	t.height = height
	return nil
}

// UpdateState updates the task list with new session state.
func (t *TaskList) UpdateState(state *session.State) tea.Cmd {
	t.state = state
	return nil
}
