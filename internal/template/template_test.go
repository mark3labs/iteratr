package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/session"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     Variables
		want     string
	}{
		{
			name:     "simple substitution",
			template: "Session: {{session}}, Iteration: {{iteration}}",
			vars: Variables{
				Session:   "test-session",
				Iteration: "42",
			},
			want: "Session: test-session, Iteration: 42",
		},
		{
			name:     "all variables",
			template: "{{session}}|{{iteration}}|{{spec}}|{{inbox}}|{{notes}}|{{tasks}}|{{extra}}",
			vars: Variables{
				Session:   "s1",
				Iteration: "1",
				Spec:      "spec content",
				Inbox:     "inbox",
				Notes:     "notes",
				Tasks:     "tasks",
				Extra:     "extra",
			},
			want: "s1|1|spec content|inbox|notes|tasks|extra",
		},
		{
			name:     "empty values",
			template: "Session: {{session}}{{inbox}}{{extra}}",
			vars: Variables{
				Session: "test",
				Inbox:   "",
				Extra:   "",
			},
			want: "Session: test",
		},
		{
			name:     "multiline template",
			template: "## Context\nSession: {{session}} | Iteration: #{{iteration}}\n{{inbox}}{{notes}}",
			vars: Variables{
				Session:   "my-session",
				Iteration: "3",
				Inbox:     "## Inbox\n- Message 1\n",
				Notes:     "## Notes\n- Note 1\n",
			},
			want: "## Context\nSession: my-session | Iteration: #3\n## Inbox\n- Message 1\n## Notes\n- Note 1\n",
		},
		{
			name:     "placeholder not replaced if variable missing",
			template: "{{session}} {{unknown}}",
			vars: Variables{
				Session: "test",
			},
			want: "test {{unknown}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.template, tt.vars)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderWithDefaultTemplate(t *testing.T) {
	vars := Variables{
		Session:   "iteratr",
		Iteration: "20",
		Spec:      "# Test Spec\nThis is a test spec.",
		Inbox:     "",
		Notes:     "LEARNING:\n  - [#1] Something learned\n",
		Tasks:     "REMAINING:\n  - [abc123] Task 1\nCOMPLETED: 5 tasks\n",
		Extra:     "",
		Port:      "4222",
	}

	result := Render(DefaultTemplate, vars)

	// Check that placeholders were replaced
	if strings.Contains(result, "{{session}}") {
		t.Error("{{session}} placeholder not replaced")
	}
	if strings.Contains(result, "{{iteration}}") {
		t.Error("{{iteration}} placeholder not replaced")
	}
	if strings.Contains(result, "{{spec}}") {
		t.Error("{{spec}} placeholder not replaced")
	}
	if strings.Contains(result, "{{tasks}}") {
		t.Error("{{tasks}} placeholder not replaced")
	}
	if strings.Contains(result, "{{notes}}") {
		t.Error("{{notes}} placeholder not replaced")
	}
	if strings.Contains(result, "{{port}}") {
		t.Error("{{port}} placeholder not replaced")
	}

	// Check that expected content is present
	if !strings.Contains(result, "Session: iteratr | Iteration: #20") {
		t.Error("Session/iteration not properly formatted")
	}
	if !strings.Contains(result, "# Test Spec") {
		t.Error("Spec content not included")
	}
	if !strings.Contains(result, "LEARNING:") {
		t.Error("Notes not included")
	}
	if !strings.Contains(result, "REMAINING:") {
		t.Error("Tasks not included")
	}
	if !strings.Contains(result, `--name iteratr`) {
		t.Error("Session name not in tools section")
	}
}

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string // Returns file path
		wantErr     bool
		wantContent string
	}{
		{
			name: "load existing file",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "template.txt")
				content := "Custom template with {{session}} and {{iteration}}"
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr:     false,
			wantContent: "Custom template with {{session}} and {{iteration}}",
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/template.txt"
			},
			wantErr: true,
		},
		{
			name: "empty file",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "empty.txt")
				if err := os.WriteFile(path, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr:     false,
			wantContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got, err := LoadFromFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantContent {
				t.Errorf("LoadFromFile() = %q, want %q", got, tt.wantContent)
			}
		})
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name       string
		customPath string
		setup      func(t *testing.T) string // Returns custom path if needed
		wantErr    bool
		checkFunc  func(t *testing.T, result string)
	}{
		{
			name:       "default template when no custom path",
			customPath: "",
			setup:      func(t *testing.T) string { return "" },
			wantErr:    false,
			checkFunc: func(t *testing.T, result string) {
				if result != DefaultTemplate {
					t.Error("Expected default template")
				}
				if !strings.Contains(result, "{{session}}") {
					t.Error("Default template should contain placeholders")
				}
			},
		},
		{
			name: "custom template from file",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "custom.template")
				content := "## My Custom Template\nSession: {{session}}\n"
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "## My Custom Template") {
					t.Error("Expected custom template content")
				}
			},
		},
		{
			name: "custom template file not found",
			setup: func(t *testing.T) string {
				return "/nonexistent/template.txt"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got, err := GetTemplate(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func TestFormatInbox(t *testing.T) {
	tests := []struct {
		name  string
		state *session.State
		want  string
	}{
		{
			name: "no messages",
			state: &session.State{
				Inbox: []*session.Message{},
			},
			want: "No messages",
		},
		{
			name: "all messages read",
			state: &session.State{
				Inbox: []*session.Message{
					{ID: "msg001", Content: "Test", Read: true, CreatedAt: time.Now()},
				},
			},
			want: "No unread messages",
		},
		{
			name: "unread messages",
			state: &session.State{
				Inbox: []*session.Message{
					{ID: "msg001abc", Content: "Message 1", Read: false, CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
					{ID: "msg002xyz", Content: "Message 2", Read: false, CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
					{ID: "msg003def", Content: "Read message", Read: true, CreatedAt: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)},
				},
			},
			want: "2 unread message(s):",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatInbox(tt.state)
			if !strings.Contains(got, tt.want) {
				t.Errorf("formatInbox() = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func TestFormatNotes(t *testing.T) {
	tests := []struct {
		name  string
		state *session.State
		want  []string // Strings that should be present
	}{
		{
			name: "no notes",
			state: &session.State{
				Notes: []*session.Note{},
			},
			want: []string{"No notes recorded"},
		},
		{
			name: "notes by type",
			state: &session.State{
				Notes: []*session.Note{
					{ID: "n1", Content: "Learned something", Type: "learning", Iteration: 5},
					{ID: "n2", Content: "Made a choice", Type: "decision", Iteration: 7},
					{ID: "n3", Content: "Hit a blocker", Type: "stuck", Iteration: 10},
				},
			},
			want: []string{"Learning:", "[#5] Learned something", "Decision:", "[#7] Made a choice", "Stuck:", "[#10] Hit a blocker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNotes(tt.state)
			for _, expected := range tt.want {
				if !strings.Contains(got, expected) {
					t.Errorf("formatNotes() = %q, want to contain %q", got, expected)
				}
			}
		})
	}
}

func TestFormatTasks(t *testing.T) {
	tests := []struct {
		name  string
		state *session.State
		want  []string // Strings that should be present
	}{
		{
			name: "no tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{},
			},
			want: []string{"No tasks"},
		},
		{
			name: "tasks by status with default priority",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"task001": {ID: "task001abc", Content: "Do thing 1", Status: "remaining", Priority: 2, Iteration: 0},
					"task002": {ID: "task002xyz", Content: "Do thing 2", Status: "in_progress", Priority: 2, Iteration: 5},
					"task003": {ID: "task003def", Content: "Done thing", Status: "completed", Priority: 2, Iteration: 3},
					"task004": {ID: "task004ghi", Content: "Blocked thing", Status: "blocked", Priority: 2, Iteration: 0},
				},
			},
			want: []string{
				"Remaining:",
				"[P2] [task001a] Do thing 1",
				"In progress:",
				"[P2] [task002x] Do thing 2",
				"[iteration #5]",
				"Completed:",
				"[P2] [task003d] Done thing",
				"[iteration #3]",
				"Blocked:",
				"[P2] [task004g] Blocked thing",
			},
		},
		{
			name: "tasks with various priorities",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"task001": {ID: "task001abc", Content: "Critical task", Status: "remaining", Priority: 0, Iteration: 0},
					"task002": {ID: "task002xyz", Content: "High priority", Status: "remaining", Priority: 1, Iteration: 0},
					"task003": {ID: "task003def", Content: "Medium task", Status: "remaining", Priority: 2, Iteration: 0},
					"task004": {ID: "task004ghi", Content: "Low priority", Status: "remaining", Priority: 3, Iteration: 0},
					"task005": {ID: "task005jkl", Content: "Backlog item", Status: "remaining", Priority: 4, Iteration: 0},
				},
			},
			want: []string{
				"[P0] [task001a] Critical task",
				"[P1] [task002x] High priority",
				"[P2] [task003d] Medium task",
				"[P3] [task004g] Low priority",
				"[P4] [task005j] Backlog item",
			},
		},
		{
			name: "tasks with dependencies",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"task001": {ID: "task001abc", Content: "Base task", Status: "remaining", Priority: 2, DependsOn: []string{}, Iteration: 0},
					"task002": {ID: "task002xyz", Content: "Dependent task", Status: "remaining", Priority: 2, DependsOn: []string{"task001abc"}, Iteration: 0},
					"task003": {ID: "task003def", Content: "Multi-dependent", Status: "remaining", Priority: 2, DependsOn: []string{"task001abc", "task002xyz"}, Iteration: 0},
				},
			},
			want: []string{
				"[P2] [task001a] Base task",
				"[P2] [task002x] Dependent task (depends on: task001a)",
				"[P2] [task003d] Multi-dependent (depends on: task001a, task002x)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTasks(tt.state)
			for _, expected := range tt.want {
				if !strings.Contains(got, expected) {
					t.Errorf("formatTasks() = %q, want to contain %q", got, expected)
				}
			}
		})
	}
}

func TestFormatIterationHistory(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		state *session.State
		want  []string // Strings that should be present
	}{
		{
			name: "no iterations",
			state: &session.State{
				Iterations: []*session.Iteration{},
			},
			want: []string{"No iteration history yet"},
		},
		{
			name: "no summaries",
			state: &session.State{
				Iterations: []*session.Iteration{
					{Number: 1, StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-50 * time.Minute), Complete: true, Summary: ""},
				},
			},
			want: []string{"No iteration summaries recorded yet"},
		},
		{
			name: "one iteration with summary",
			state: &session.State{
				Iterations: []*session.Iteration{
					{Number: 1, StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-30 * time.Minute), Complete: true, Summary: "Added auth middleware"},
				},
			},
			want: []string{"- #1 (30min ago): Added auth middleware"},
		},
		{
			name: "multiple iterations with summaries - shows last 5",
			state: &session.State{
				Iterations: []*session.Iteration{
					{Number: 1, StartedAt: now.Add(-10 * time.Hour), EndedAt: now.Add(-9 * time.Hour), Complete: true, Summary: "Setup project"},
					{Number: 2, StartedAt: now.Add(-8 * time.Hour), EndedAt: now.Add(-7 * time.Hour), Complete: true, Summary: "Added database models"},
					{Number: 3, StartedAt: now.Add(-6 * time.Hour), EndedAt: now.Add(-5 * time.Hour), Complete: true, Summary: "Implemented API routes"},
					{Number: 4, StartedAt: now.Add(-4 * time.Hour), EndedAt: now.Add(-3 * time.Hour), Complete: true, Summary: "Added validation"},
					{Number: 5, StartedAt: now.Add(-2 * time.Hour), EndedAt: now.Add(-1 * time.Hour), Complete: true, Summary: "Fixed auth bug"},
					{Number: 6, StartedAt: now.Add(-30 * time.Minute), EndedAt: now.Add(-15 * time.Minute), Complete: true, Summary: "Added tests"},
				},
			},
			want: []string{
				"- #2 (7hr ago): Added database models",
				"- #3 (5hr ago): Implemented API routes",
				"- #4 (3hr ago): Added validation",
				"- #5 (1hr ago): Fixed auth bug",
				"- #6 (15min ago): Added tests",
			},
		},
		{
			name: "iterations with and without summaries",
			state: &session.State{
				Iterations: []*session.Iteration{
					{Number: 1, StartedAt: now.Add(-2 * time.Hour), EndedAt: now.Add(-90 * time.Minute), Complete: true, Summary: ""},
					{Number: 2, StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-30 * time.Minute), Complete: true, Summary: "Completed feature X"},
					{Number: 3, StartedAt: now.Add(-20 * time.Minute), EndedAt: now.Add(-10 * time.Minute), Complete: true, Summary: "Fixed bug Y"},
				},
			},
			want: []string{
				"- #2 (30min ago): Completed feature X",
				"- #3 (10min ago): Fixed bug Y",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatIterationHistory(tt.state)
			for _, expected := range tt.want {
				if !strings.Contains(got, expected) {
					t.Errorf("formatIterationHistory() = %q, want to contain %q", got, expected)
				}
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"one minute", 1 * time.Minute, "1min ago"},
		{"multiple minutes", 15 * time.Minute, "15min ago"},
		{"one hour", 1 * time.Hour, "1hr ago"},
		{"multiple hours", 5 * time.Hour, "5hr ago"},
		{"one day", 24 * time.Hour, "1 day ago"},
		{"multiple days", 3 * 24 * time.Hour, "3 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimeAgo(tt.duration)
			if got != tt.want {
				t.Errorf("formatTimeAgo(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	// This is an integration test - requires actual NATS setup
	// For now, test the formatting functions independently above
	// Full BuildPrompt testing will be done in integration tests
	t.Skip("Integration test - requires NATS setup")
}
