package testfixtures

import (
	"time"

	"github.com/mark3labs/iteratr/internal/session"
)

// Fixed test values for consistent golden files
const (
	FixedSessionName = "test-session"
	FixedGitHash     = "abc1234"
	FixedIteration   = 1
)

var (
	FixedTime = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
)

// EmptyState returns a minimal empty session state
func EmptyState() *session.State {
	return &session.State{
		Session:     FixedSessionName,
		Tasks:       map[string]*session.Task{},
		TaskCounter: 0,
		Notes:       []*session.Note{},
		NoteCounter: 0,
		Iterations:  []*session.Iteration{},
		Complete:    false,
	}
}

// StateWithTasks returns a state with various tasks in different states
func StateWithTasks() *session.State {
	return &session.State{
		Session: FixedSessionName,
		Tasks: map[string]*session.Task{
			"TAS-1": {
				ID:        "TAS-1",
				Content:   "[P0] Create test infrastructure",
				Status:    "completed",
				Priority:  0,
				DependsOn: []string{},
				CreatedAt: FixedTime,
				UpdatedAt: FixedTime.Add(10 * time.Minute),
				Iteration: 1,
			},
			"TAS-2": {
				ID:        "TAS-2",
				Content:   "[P1] Implement feature X",
				Status:    "in_progress",
				Priority:  1,
				DependsOn: []string{},
				CreatedAt: FixedTime.Add(5 * time.Minute),
				UpdatedAt: FixedTime.Add(15 * time.Minute),
				Iteration: 1,
			},
			"TAS-3": {
				ID:        "TAS-3",
				Content:   "[P2] Add documentation",
				Status:    "remaining",
				Priority:  2,
				DependsOn: []string{"TAS-2"},
				CreatedAt: FixedTime.Add(10 * time.Minute),
				UpdatedAt: FixedTime.Add(10 * time.Minute),
				Iteration: 1,
			},
		},
		TaskCounter: 3,
		Notes:       []*session.Note{},
		NoteCounter: 0,
		Iterations: []*session.Iteration{
			{
				Number:    1,
				StartedAt: FixedTime,
				Complete:  false,
			},
		},
		Complete: false,
	}
}

// StateWithNotes returns a state with various notes
func StateWithNotes() *session.State {
	return &session.State{
		Session:     FixedSessionName,
		Tasks:       map[string]*session.Task{},
		TaskCounter: 0,
		Notes: []*session.Note{
			{
				ID:        "NOT-1",
				Content:   "Learned about event sourcing pattern",
				Type:      "learning",
				CreatedAt: FixedTime,
				Iteration: 1,
			},
			{
				ID:        "NOT-2",
				Content:   "Blocked on missing dependencies",
				Type:      "stuck",
				CreatedAt: FixedTime.Add(5 * time.Minute),
				Iteration: 1,
			},
			{
				ID:        "NOT-3",
				Content:   "Use teatest for testing TUI components",
				Type:      "tip",
				CreatedAt: FixedTime.Add(10 * time.Minute),
				Iteration: 1,
			},
			{
				ID:        "NOT-4",
				Content:   "Decided to use golden files for visual regression",
				Type:      "decision",
				CreatedAt: FixedTime.Add(15 * time.Minute),
				Iteration: 1,
			},
		},
		NoteCounter: 4,
		Iterations: []*session.Iteration{
			{
				Number:    1,
				StartedAt: FixedTime,
				Complete:  false,
			},
		},
		Complete: false,
	}
}

// FullState returns a state with tasks, notes, and iterations
func FullState() *session.State {
	return &session.State{
		Session: FixedSessionName,
		Tasks: map[string]*session.Task{
			"TAS-1": {
				ID:        "TAS-1",
				Content:   "[P0] Create test infrastructure",
				Status:    "completed",
				Priority:  0,
				DependsOn: []string{},
				CreatedAt: FixedTime,
				UpdatedAt: FixedTime.Add(10 * time.Minute),
				Iteration: 1,
			},
			"TAS-2": {
				ID:        "TAS-2",
				Content:   "[P1] Implement feature X",
				Status:    "in_progress",
				Priority:  1,
				DependsOn: []string{},
				CreatedAt: FixedTime.Add(5 * time.Minute),
				UpdatedAt: FixedTime.Add(15 * time.Minute),
				Iteration: 2,
			},
			"TAS-3": {
				ID:        "TAS-3",
				Content:   "[P2] Add documentation",
				Status:    "remaining",
				Priority:  2,
				DependsOn: []string{"TAS-2"},
				CreatedAt: FixedTime.Add(10 * time.Minute),
				UpdatedAt: FixedTime.Add(10 * time.Minute),
				Iteration: 1,
			},
			"TAS-4": {
				ID:        "TAS-4",
				Content:   "[P3] Refactor legacy code",
				Status:    "blocked",
				Priority:  3,
				DependsOn: []string{"TAS-1", "TAS-2"},
				CreatedAt: FixedTime.Add(15 * time.Minute),
				UpdatedAt: FixedTime.Add(20 * time.Minute),
				Iteration: 1,
			},
		},
		TaskCounter: 4,
		Notes: []*session.Note{
			{
				ID:        "NOT-1",
				Content:   "Learned about event sourcing pattern",
				Type:      "learning",
				CreatedAt: FixedTime,
				Iteration: 1,
			},
			{
				ID:        "NOT-2",
				Content:   "Blocked on missing dependencies",
				Type:      "stuck",
				CreatedAt: FixedTime.Add(5 * time.Minute),
				Iteration: 1,
			},
		},
		NoteCounter: 2,
		Iterations: []*session.Iteration{
			{
				Number:      1,
				StartedAt:   FixedTime,
				EndedAt:     FixedTime.Add(30 * time.Minute),
				Complete:    true,
				Summary:     "Created test infrastructure",
				TasksWorked: []string{"TAS-1"},
			},
			{
				Number:    2,
				StartedAt: FixedTime.Add(35 * time.Minute),
				Complete:  false,
			},
		},
		Complete: false,
	}
}

// StateWithBlockedTasks returns a state with tasks in blocked state
func StateWithBlockedTasks() *session.State {
	return &session.State{
		Session: FixedSessionName,
		Tasks: map[string]*session.Task{
			"TAS-1": {
				ID:        "TAS-1",
				Content:   "[P1] Blocked by external dependency",
				Status:    "blocked",
				Priority:  1,
				DependsOn: []string{},
				CreatedAt: FixedTime,
				UpdatedAt: FixedTime,
				Iteration: 1,
			},
			"TAS-2": {
				ID:        "TAS-2",
				Content:   "[P2] Blocked by TAS-1",
				Status:    "remaining",
				Priority:  2,
				DependsOn: []string{"TAS-1"},
				CreatedAt: FixedTime.Add(5 * time.Minute),
				UpdatedAt: FixedTime.Add(5 * time.Minute),
				Iteration: 1,
			},
		},
		TaskCounter: 2,
		Notes:       []*session.Note{},
		NoteCounter: 0,
		Iterations: []*session.Iteration{
			{
				Number:    1,
				StartedAt: FixedTime,
				Complete:  false,
			},
		},
		Complete: false,
	}
}

// StateWithCompletedSession returns a state with session marked complete
func StateWithCompletedSession() *session.State {
	return &session.State{
		Session: FixedSessionName,
		Tasks: map[string]*session.Task{
			"TAS-1": {
				ID:        "TAS-1",
				Content:   "[P0] Implement feature",
				Status:    "completed",
				Priority:  0,
				DependsOn: []string{},
				CreatedAt: FixedTime,
				UpdatedAt: FixedTime.Add(10 * time.Minute),
				Iteration: 1,
			},
		},
		TaskCounter: 1,
		Notes:       []*session.Note{},
		NoteCounter: 0,
		Iterations: []*session.Iteration{
			{
				Number:      1,
				StartedAt:   FixedTime,
				EndedAt:     FixedTime.Add(15 * time.Minute),
				Complete:    true,
				Summary:     "Completed all tasks",
				TasksWorked: []string{"TAS-1"},
			},
		},
		Complete: true,
	}
}
