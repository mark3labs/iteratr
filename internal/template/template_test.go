package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(result, `session_name="iteratr"`) {
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
