package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// --- TaskModal Unit Tests ---

func TestTaskModal_Initialization(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()

	require.NotNil(t, modal, "modal should be initialized")
	require.False(t, modal.IsVisible(), "modal should not be visible initially")
	require.Nil(t, modal.task, "modal should not have a task initially")
	require.Equal(t, 60, modal.width, "modal should have default width")
	require.Equal(t, 20, modal.height, "modal should have default height")
}

func TestTaskModal_SetTaskAndClose(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	task := &session.Task{
		ID:       "TAS-123",
		Content:  "Test task content",
		Status:   "in_progress",
		Priority: 1,
	}

	// SetTask should make modal visible
	modal.SetTask(task)
	require.True(t, modal.IsVisible(), "modal should be visible after SetTask")
	require.Equal(t, task, modal.task, "modal should store the provided task")

	// Close should hide modal and clear task
	modal.Close()
	require.False(t, modal.IsVisible(), "modal should not be visible after Close")
	require.Nil(t, modal.task, "modal should clear task after Close")
}

func TestTaskModal_StatusBadges(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()

	tests := []struct {
		status   string
		expected string
	}{
		{"in_progress", "► in_progress"},
		{"remaining", "○ remaining"},
		{"completed", "✓ completed"},
		{"blocked", "⊘ blocked"},
		{"cancelled", "○ cancelled"},
		{"unknown", "○ unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			badge := modal.renderStatusBadge(tt.status)
			// Badge should contain expected text (may have ANSI styling)
			require.Contains(t, badge, tt.expected, "badge should contain expected text for status '%s'", tt.status)
		})
	}
}

func TestTaskModal_PriorityBadges(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()

	tests := []struct {
		priority int
		expected string
	}{
		{0, "critical"},
		{1, "high"},
		{2, "medium"},
		{3, "low"},
		{4, "backlog"},
		{5, "p5"},
		{6, "p6"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			badge := modal.renderPriorityBadge(tt.priority)
			require.Contains(t, badge, tt.expected, "badge should contain expected text for priority %d", tt.priority)
		})
	}
}

func TestTaskModal_FormatTimeHelper(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	testTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)

	result := modal.formatTime(testTime)
	expected := "2024-01-15 14:30:45"

	require.Equal(t, expected, result, "time should be formatted correctly")
}

func TestTaskModal_WordWrapHelper(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()

	tests := []struct {
		name        string
		text        string
		width       int
		minExpected int // Minimum expected number of lines
		maxExpected int // Maximum expected number of lines
	}{
		{
			name:        "short_text",
			text:        "Hello world",
			width:       20,
			minExpected: 1,
			maxExpected: 1,
		},
		{
			name:        "text_requiring_wrap",
			text:        "This is a longer text that should wrap to multiple lines when constrained to a narrow width",
			width:       20,
			minExpected: 5,
			maxExpected: 6,
		},
		{
			name:        "empty_text",
			text:        "",
			width:       20,
			minExpected: 0,
			maxExpected: 1,
		},
		{
			name:        "single_long_word",
			text:        "Supercalifragilisticexpialidocious",
			width:       20,
			minExpected: 1,
			maxExpected: 1,
		},
		{
			name:        "zero_width",
			text:        "Test content",
			width:       0,
			minExpected: 1,
			maxExpected: 1, // Should use default width of 40
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := modal.wordWrap(tt.text, tt.width)
			var lines []string
			if result == "" {
				lines = []string{}
			} else {
				lines = strings.Split(result, "\n")
			}
			require.GreaterOrEqual(t, len(lines), tt.minExpected, "should have at least %d lines", tt.minExpected)
			require.LessOrEqual(t, len(lines), tt.maxExpected, "should have at most %d lines", tt.maxExpected)
		})
	}
}

func TestTaskModal_BuildContentElements(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-456",
		Content:   "This is a test task with some content",
		Status:    "in_progress",
		Priority:  1,
		DependsOn: []string{"TAS-123", "TAS-789"},
		CreatedAt: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 15, 16, 45, 0, 0, time.UTC),
	}
	modal.SetTask(task)

	content := modal.buildContent(50)

	// Verify all key elements are present
	tests := []struct {
		name     string
		contains string
	}{
		{"title", "Task Details"},
		{"task_id", "TAS-456"},
		{"status", "in_progress"},
		{"priority", "high"},
		{"content", "This is a test task"},
		{"dependencies", "TAS-123"},
		{"dependencies", "TAS-789"},
		{"close_hint", "esc"},
		{"close_hint", "close"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Contains(t, content, tt.contains, "content should include %s", tt.name)
		})
	}
}

func TestTaskModal_BuildContentNoDependencies(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-999",
		Content:   "Test content without dependencies",
		Status:    "remaining",
		Priority:  2,
		DependsOn: []string{}, // Empty dependencies
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime,
	}
	modal.SetTask(task)

	content := modal.buildContent(50)

	// Should not include "Depends on:" label when no dependencies
	require.NotContains(t, content, "Depends on:", "content should not include 'Depends on:' when task has no dependencies")
	require.Contains(t, content, "TAS-999", "content should include task ID")
	require.Contains(t, content, "Test content without dependencies", "content should include task content")
}

func TestTaskModal_BuildContentMinimalWidth(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	task := &session.Task{
		ID:      "TAS-MIN",
		Content: "Test content",
		Status:  "completed",
	}
	modal.SetTask(task)

	// Test buildContent with minimal width
	content := modal.buildContent(10)

	// Modal should not crash and should include key elements
	require.NotEmpty(t, content, "modal should render even with minimal width")
	require.Contains(t, content, "TAS-MIN", "content should contain task ID")
}

func TestTaskModal_BuildContentLongContent(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()

	// Create task with very long content
	longContent := strings.Repeat("This is a very long task description that should wrap properly. ", 10)
	task := &session.Task{
		ID:       "TAS-LONG",
		Content:  longContent,
		Status:   "in_progress",
		Priority: 1,
	}
	modal.SetTask(task)

	content := modal.buildContent(50)

	// Should contain wrapped content
	require.Contains(t, content, "This is a very long task description", "content should include long text")

	// Should have multiple lines
	lines := strings.Split(content, "\n")
	require.Greater(t, len(lines), 10, "long content should wrap to multiple lines")
}

func TestTaskModal_CacheInvalidation(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	task1 := &session.Task{
		ID:      "TAS-1",
		Content: "First task",
	}
	task2 := &session.Task{
		ID:      "TAS-2",
		Content: "Second task",
	}

	// Build content for first task
	modal.SetTask(task1)
	content1 := modal.buildContent(50)
	require.Contains(t, content1, "TAS-1", "content should contain first task ID")

	// SetTask should invalidate cache
	modal.SetTask(task2)
	content2 := modal.buildContent(50)
	require.Contains(t, content2, "TAS-2", "content should contain second task ID")
	require.NotContains(t, content2, "TAS-1", "content should not contain first task ID")
}

// --- TaskModal Golden File Tests ---

func TestTaskModalGolden_InProgress(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-001",
		Content:   "Implement feature X with comprehensive test coverage",
		Status:    "in_progress",
		Priority:  1,
		DependsOn: []string{},
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime.Add(10 * time.Minute),
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_in_progress.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

func TestTaskModalGolden_Completed(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-002",
		Content:   "Create test infrastructure",
		Status:    "completed",
		Priority:  0,
		DependsOn: []string{},
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime.Add(30 * time.Minute),
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_completed.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

func TestTaskModalGolden_Blocked(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-003",
		Content:   "Refactor legacy code",
		Status:    "blocked",
		Priority:  3,
		DependsOn: []string{"TAS-001", "TAS-002"},
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime.Add(5 * time.Minute),
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_blocked.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

func TestTaskModalGolden_Remaining(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-004",
		Content:   "Add documentation for new features",
		Status:    "remaining",
		Priority:  2,
		DependsOn: []string{"TAS-001"},
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime,
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_remaining.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

func TestTaskModalGolden_Critical(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-005",
		Content:   "Fix critical security vulnerability",
		Status:    "in_progress",
		Priority:  0,
		DependsOn: []string{},
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime.Add(2 * time.Minute),
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_critical.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

func TestTaskModalGolden_Backlog(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-006",
		Content:   "Explore potential performance optimizations",
		Status:    "remaining",
		Priority:  4,
		DependsOn: []string{},
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime,
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_backlog.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

func TestTaskModalGolden_LongContent(t *testing.T) {
	modal := NewTaskModal()
	longContent := "This is a very long task description that spans multiple lines when wrapped to fit within the modal width. " +
		"It includes multiple sentences to demonstrate how the word wrapping functionality handles extended text content. " +
		"The modal should gracefully wrap this content and display it in a readable format without truncation or overflow issues."
	task := &session.Task{
		ID:        "TAS-007",
		Content:   longContent,
		Status:    "in_progress",
		Priority:  1,
		DependsOn: []string{"TAS-001", "TAS-002", "TAS-003"},
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime.Add(15 * time.Minute),
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_long_content.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

func TestTaskModalGolden_NoDependencies(t *testing.T) {
	modal := NewTaskModal()
	task := &session.Task{
		ID:        "TAS-008",
		Content:   "Standalone task with no dependencies",
		Status:    "remaining",
		Priority:  2,
		DependsOn: []string{}, // Empty dependencies
		CreatedAt: testfixtures.FixedTime,
		UpdatedAt: testfixtures.FixedTime,
	}
	modal.SetTask(task)

	content := modal.buildContent(60)
	require.NotEmpty(t, content, "content should not be empty")

	goldenFile := filepath.Join("testdata", "task_modal_no_dependencies.golden")
	testfixtures.CompareGolden(t, goldenFile, content)
}

// compareGolden compares rendered output with golden file
// Note: compareGolden is defined in messages_expanded_test.go and shared across golden file tests

// --- TaskModal Command Execution Tests ---

func TestTaskModal_Update_ReturnsNil(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	task := &session.Task{
		ID:      "TAS-CMD-1",
		Content: "Test command execution",
		Status:  "in_progress",
	}
	modal.SetTask(task)

	// Test that Update returns nil for all message types
	testCases := []struct {
		name string
		msg  tea.Msg
	}{
		{"KeyPressMsg", tea.KeyPressMsg{Text: "enter"}},
		{"KeyPressMsg esc", tea.KeyPressMsg{Text: "esc"}},
		{"KeyPressMsg space", tea.KeyPressMsg{Text: " "}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := modal.Update(tc.msg)
			require.Nil(t, cmd, "TaskModal.Update should always return nil")
		})
	}
}

func TestTaskModal_Update_WhenNotVisible(t *testing.T) {
	t.Parallel()

	modal := NewTaskModal()
	// Modal not visible, no task set

	cmd := modal.Update(tea.KeyPressMsg{Text: "enter"})
	require.Nil(t, cmd, "Update should return nil when modal not visible")
}
