package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExecuteAllPiped(t *testing.T) {
	ctx := context.Background()
	workDir := t.TempDir()
	vars := Variables{Session: "test", Iteration: "1"}

	tests := []struct {
		name     string
		hooks    []*HookConfig
		expected string
	}{
		{
			name:     "no hooks",
			hooks:    []*HookConfig{},
			expected: "",
		},
		{
			name: "single hook with pipe_output true",
			hooks: []*HookConfig{
				{Command: "echo 'piped'", Timeout: 5, PipeOutput: true},
			},
			expected: "piped\n",
		},
		{
			name: "single hook with pipe_output false",
			hooks: []*HookConfig{
				{Command: "echo 'not piped'", Timeout: 5, PipeOutput: false},
			},
			expected: "",
		},
		{
			name: "multiple hooks mixed pipe_output",
			hooks: []*HookConfig{
				{Command: "echo 'first piped'", Timeout: 5, PipeOutput: true},
				{Command: "echo 'not piped'", Timeout: 5, PipeOutput: false},
				{Command: "echo 'second piped'", Timeout: 5, PipeOutput: true},
			},
			expected: "first piped\n\nsecond piped\n",
		},
		{
			name: "all hooks with pipe_output false",
			hooks: []*HookConfig{
				{Command: "echo 'first'", Timeout: 5, PipeOutput: false},
				{Command: "echo 'second'", Timeout: 5, PipeOutput: false},
			},
			expected: "",
		},
		{
			name: "all hooks with pipe_output true",
			hooks: []*HookConfig{
				{Command: "echo 'first'", Timeout: 5, PipeOutput: true},
				{Command: "echo 'second'", Timeout: 5, PipeOutput: true},
			},
			expected: "first\n\nsecond\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ExecuteAllPiped(ctx, tt.hooks, workDir, vars)
			if err != nil {
				t.Fatalf("ExecuteAllPiped() error = %v", err)
			}
			if output != tt.expected {
				t.Errorf("ExecuteAllPiped() output = %q, expected %q", output, tt.expected)
			}
		})
	}
}

func TestExecuteAllPiped_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	workDir := t.TempDir()
	vars := Variables{Session: "test", Iteration: "1"}
	hooks := []*HookConfig{
		{Command: "echo 'test'", Timeout: 5, PipeOutput: true},
	}

	_, err := ExecuteAllPiped(ctx, hooks, workDir, vars)
	if err == nil {
		t.Error("ExecuteAllPiped() expected error for cancelled context, got nil")
	}
}

func TestConfigParsing(t *testing.T) {
	yamlContent := `
version: 1
hooks:
  session_start:
    - command: "git pull"
      timeout: 30
      pipe_output: true
  pre_iteration:
    - command: "golangci-lint run"
      timeout: 30
  post_iteration:
    - command: "go test ./..."
      timeout: 120
      pipe_output: true
  session_end:
    - command: "git push"
      timeout: 30
  on_task_complete:
    - command: "./validate.sh"
      timeout: 30
      pipe_output: true
  on_error:
    - command: "git diff HEAD"
      timeout: 10
      pipe_output: true
`

	var cfg Config
	err := yaml.Unmarshal([]byte(yamlContent), &cfg)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("Version = %d, expected 1", cfg.Version)
	}

	// Verify session_start
	if len(cfg.Hooks.SessionStart) != 1 {
		t.Errorf("SessionStart length = %d, expected 1", len(cfg.Hooks.SessionStart))
	} else {
		hook := cfg.Hooks.SessionStart[0]
		if hook.Command != "git pull" {
			t.Errorf("SessionStart[0].Command = %q, expected %q", hook.Command, "git pull")
		}
		if hook.Timeout != 30 {
			t.Errorf("SessionStart[0].Timeout = %d, expected 30", hook.Timeout)
		}
		if !hook.PipeOutput {
			t.Error("SessionStart[0].PipeOutput = false, expected true")
		}
	}

	// Verify pre_iteration
	if len(cfg.Hooks.PreIteration) != 1 {
		t.Errorf("PreIteration length = %d, expected 1", len(cfg.Hooks.PreIteration))
	}

	// Verify post_iteration
	if len(cfg.Hooks.PostIteration) != 1 {
		t.Errorf("PostIteration length = %d, expected 1", len(cfg.Hooks.PostIteration))
	} else {
		hook := cfg.Hooks.PostIteration[0]
		if !hook.PipeOutput {
			t.Error("PostIteration[0].PipeOutput = false, expected true")
		}
	}

	// Verify session_end
	if len(cfg.Hooks.SessionEnd) != 1 {
		t.Errorf("SessionEnd length = %d, expected 1", len(cfg.Hooks.SessionEnd))
	} else {
		hook := cfg.Hooks.SessionEnd[0]
		if hook.Command != "git push" {
			t.Errorf("SessionEnd[0].Command = %q, expected %q", hook.Command, "git push")
		}
	}

	// Verify on_task_complete
	if len(cfg.Hooks.OnTaskComplete) != 1 {
		t.Errorf("OnTaskComplete length = %d, expected 1", len(cfg.Hooks.OnTaskComplete))
	} else {
		hook := cfg.Hooks.OnTaskComplete[0]
		if hook.Command != "./validate.sh" {
			t.Errorf("OnTaskComplete[0].Command = %q, expected %q", hook.Command, "./validate.sh")
		}
		if !hook.PipeOutput {
			t.Error("OnTaskComplete[0].PipeOutput = false, expected true")
		}
	}

	// Verify on_error
	if len(cfg.Hooks.OnError) != 1 {
		t.Errorf("OnError length = %d, expected 1", len(cfg.Hooks.OnError))
	} else {
		hook := cfg.Hooks.OnError[0]
		if hook.Command != "git diff HEAD" {
			t.Errorf("OnError[0].Command = %q, expected %q", hook.Command, "git diff HEAD")
		}
		if !hook.PipeOutput {
			t.Error("OnError[0].PipeOutput = false, expected true")
		}
	}
}

func TestConfigParsing_EmptyHooks(t *testing.T) {
	yamlContent := `
version: 1
hooks: {}
`

	var cfg Config
	err := yaml.Unmarshal([]byte(yamlContent), &cfg)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if cfg.Hooks.SessionStart != nil {
		t.Error("SessionStart should be nil for empty hooks")
	}
	if cfg.Hooks.SessionEnd != nil {
		t.Error("SessionEnd should be nil for empty hooks")
	}
	if cfg.Hooks.OnTaskComplete != nil {
		t.Error("OnTaskComplete should be nil for empty hooks")
	}
	if cfg.Hooks.OnError != nil {
		t.Error("OnError should be nil for empty hooks")
	}
}

func TestConfigFromFile(t *testing.T) {
	// Test loading from actual file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".iteratr.hooks.yml")

	yamlContent := `version: 1
hooks:
  session_start:
    - command: "echo start"
      timeout: 10
  session_end:
    - command: "echo end"
      timeout: 10
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if len(cfg.Hooks.SessionStart) != 1 {
		t.Errorf("SessionStart length = %d, expected 1", len(cfg.Hooks.SessionStart))
	}
	if len(cfg.Hooks.SessionEnd) != 1 {
		t.Errorf("SessionEnd length = %d, expected 1", len(cfg.Hooks.SessionEnd))
	}
}

func TestExpandVariables(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		vars     Variables
		expected string
	}{
		{
			name:     "session and iteration",
			command:  "echo {{session}} iteration {{iteration}}",
			vars:     Variables{Session: "test-session", Iteration: "5"},
			expected: "echo test-session iteration 5",
		},
		{
			name:     "task_id and task_content",
			command:  "./validate.sh {{task_id}} '{{task_content}}'",
			vars:     Variables{TaskID: "TAS-123", TaskContent: "Fix the bug"},
			expected: "./validate.sh TAS-123 'Fix the bug'",
		},
		{
			name:     "error variable",
			command:  "echo 'Error occurred: {{error}}'",
			vars:     Variables{Error: "connection timeout"},
			expected: "echo 'Error occurred: connection timeout'",
		},
		{
			name:     "all variables",
			command:  "{{session}}/{{iteration}}/{{task_id}}/{{task_content}}/{{error}}",
			vars:     Variables{Session: "s", Iteration: "1", TaskID: "t", TaskContent: "c", Error: "e"},
			expected: "s/1/t/c/e",
		},
		{
			name:     "no variables",
			command:  "echo 'hello world'",
			vars:     Variables{},
			expected: "echo 'hello world'",
		},
		{
			name:     "empty variable values",
			command:  "echo {{session}} {{iteration}} {{task_id}} {{task_content}} {{error}}",
			vars:     Variables{},
			expected: "echo     ",
		},
		{
			name:     "multiple same variable",
			command:  "{{session}}-{{session}}-{{session}}",
			vars:     Variables{Session: "test"},
			expected: "test-test-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandVariables(tt.command, tt.vars)
			if result != tt.expected {
				t.Errorf("expandVariables() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
