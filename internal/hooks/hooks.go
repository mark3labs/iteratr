package hooks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/iteratr/internal/logger"
	"gopkg.in/yaml.v3"
)

// ConfigFileName is the name of the hooks configuration file.
const ConfigFileName = ".iteratr.hooks.yml"

// LoadConfig loads the hooks configuration from the working directory.
// Returns nil if the config file doesn't exist (hooks are optional).
// Returns an error only if the file exists but cannot be parsed.
func LoadConfig(workDir string) (*Config, error) {
	configPath := filepath.Join(workDir, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("No hooks config found at %s", configPath)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read hooks config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse hooks config: %w", err)
	}

	logger.Debug("Loaded hooks config from %s (version: %d)", configPath, cfg.Version)
	return &cfg, nil
}

// Variables holds template variables that can be expanded in hook commands.
type Variables struct {
	Session     string
	Iteration   string
	TaskID      string
	TaskContent string
	Error       string
}

// Execute runs a hook command and returns its output.
// Template variables in the command ({{session}}, {{iteration}}) are expanded before execution.
// On error, returns an error message as output and nil error (graceful degradation).
// Only returns error for context cancellation.
func Execute(ctx context.Context, hook *HookConfig, workDir string, vars Variables) (string, error) {
	if hook == nil || hook.Command == "" {
		return "", nil
	}

	// Expand template variables in command
	command := expandVariables(hook.Command, vars)
	logger.Debug("Executing hook command: %s", command)

	// Determine timeout
	timeout := hook.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Execute command via shell
	cmd := exec.CommandContext(execCtx, "sh", "-c", command)
	cmd.Dir = workDir

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Check for context cancellation (propagate this)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Handle timeout
	if execCtx.Err() == context.DeadlineExceeded {
		logger.Warn("Hook command timed out after %ds: %s", timeout, command)
		return fmt.Sprintf("[Hook timed out after %ds]\nPartial output:\n%s", timeout, stdout.String()), nil
	}

	// Handle command failure (graceful degradation - include error in output)
	if err != nil {
		logger.Warn("Hook command failed: %v", err)
		output := stdout.String()
		if stderr.Len() > 0 {
			output += "\n[stderr]\n" + stderr.String()
		}
		return fmt.Sprintf("[Hook command failed: %v]\n%s", err, output), nil
	}

	// Success - return stdout (include stderr if present)
	output := stdout.String()
	if stderr.Len() > 0 {
		logger.Debug("Hook stderr: %s", stderr.String())
		// Include stderr in output so agent has full context
		output += "\n[stderr]\n" + stderr.String()
	}

	logger.Debug("Hook executed successfully, output length: %d bytes", len(output))
	return output, nil
}

// ExecuteAll runs multiple hook commands and concatenates their output.
// Returns combined output from all hooks separated by newlines.
// Only returns error for context cancellation.
func ExecuteAll(ctx context.Context, hooks []*HookConfig, workDir string, vars Variables) (string, error) {
	if len(hooks) == 0 {
		return "", nil
	}

	var outputs []string
	for _, hook := range hooks {
		output, err := Execute(ctx, hook, workDir, vars)
		if err != nil {
			// Context cancelled - propagate
			return "", err
		}
		if output != "" {
			outputs = append(outputs, output)
		}
	}

	return strings.Join(outputs, "\n"), nil
}

// ExecuteAllPiped runs multiple hook commands and returns only output from hooks with pipe_output: true.
// All hooks are executed (for side effects), but only output from hooks with PipeOutput=true is returned.
// Returns combined piped output separated by newlines.
// Only returns error for context cancellation.
func ExecuteAllPiped(ctx context.Context, hooks []*HookConfig, workDir string, vars Variables) (string, error) {
	if len(hooks) == 0 {
		return "", nil
	}

	var pipedOutputs []string
	for _, hook := range hooks {
		output, err := Execute(ctx, hook, workDir, vars)
		if err != nil {
			// Context cancelled - propagate
			return "", err
		}
		// Only include output if pipe_output is true
		if hook.PipeOutput && output != "" {
			pipedOutputs = append(pipedOutputs, output)
		}
	}

	return strings.Join(pipedOutputs, "\n"), nil
}

// HookResult contains the result of executing a single hook command.
type HookResult struct {
	Command  string        // The expanded command that was run
	Output   string        // Command output (stdout + stderr)
	Failed   bool          // Whether the command failed (non-zero exit or timeout)
	Duration time.Duration // How long the command took
}

// OnHookStart is called before a hook command starts executing.
// hookIndex is the 0-based index within the hook list.
type OnHookStart func(hookIndex int, command string)

// OnHookComplete is called after a hook command finishes executing.
type OnHookComplete func(hookIndex int, result HookResult)

// ExecuteAllPipedWithCallbacks is like ExecuteAllPiped but calls onStart/onComplete
// around each individual hook execution, allowing the caller to update the TUI in real time.
// Returns combined piped output (same as ExecuteAllPiped).
// Only returns error for context cancellation.
func ExecuteAllPipedWithCallbacks(
	ctx context.Context,
	hooks []*HookConfig,
	workDir string,
	vars Variables,
	onStart OnHookStart,
	onComplete OnHookComplete,
) (string, error) {
	if len(hooks) == 0 {
		return "", nil
	}

	var pipedOutputs []string
	for i, hook := range hooks {
		if hook == nil || hook.Command == "" {
			continue
		}

		// Expand variables for the callback (show what's actually being run)
		expandedCmd := expandVariables(hook.Command, vars)

		// Notify start
		if onStart != nil {
			onStart(i, expandedCmd)
		}

		start := time.Now()
		output, err := Execute(ctx, hook, workDir, vars)
		elapsed := time.Since(start)

		if err != nil {
			// Context cancelled - notify and propagate
			if onComplete != nil {
				onComplete(i, HookResult{
					Command:  expandedCmd,
					Output:   output,
					Failed:   true,
					Duration: elapsed,
				})
			}
			return "", err
		}

		// Determine if command failed (output starts with "[Hook" markers)
		failed := strings.HasPrefix(output, "[Hook command failed:") ||
			strings.HasPrefix(output, "[Hook timed out")

		// Notify completion
		if onComplete != nil {
			onComplete(i, HookResult{
				Command:  expandedCmd,
				Output:   output,
				Failed:   failed,
				Duration: elapsed,
			})
		}

		// Only include output if pipe_output is true
		if hook.PipeOutput && output != "" {
			pipedOutputs = append(pipedOutputs, output)
		}
	}

	return strings.Join(pipedOutputs, "\n"), nil
}

// ExecuteAllWithCallbacks is like ExecuteAll but calls onStart/onComplete
// around each individual hook execution.
// Returns combined output from all hooks separated by newlines.
// Only returns error for context cancellation.
func ExecuteAllWithCallbacks(
	ctx context.Context,
	hooks []*HookConfig,
	workDir string,
	vars Variables,
	onStart OnHookStart,
	onComplete OnHookComplete,
) (string, error) {
	if len(hooks) == 0 {
		return "", nil
	}

	var outputs []string
	for i, hook := range hooks {
		if hook == nil || hook.Command == "" {
			continue
		}

		expandedCmd := expandVariables(hook.Command, vars)

		if onStart != nil {
			onStart(i, expandedCmd)
		}

		start := time.Now()
		output, err := Execute(ctx, hook, workDir, vars)
		elapsed := time.Since(start)

		if err != nil {
			if onComplete != nil {
				onComplete(i, HookResult{
					Command:  expandedCmd,
					Output:   output,
					Failed:   true,
					Duration: elapsed,
				})
			}
			return "", err
		}

		failed := strings.HasPrefix(output, "[Hook command failed:") ||
			strings.HasPrefix(output, "[Hook timed out")

		if onComplete != nil {
			onComplete(i, HookResult{
				Command:  expandedCmd,
				Output:   output,
				Failed:   failed,
				Duration: elapsed,
			})
		}

		if output != "" {
			outputs = append(outputs, output)
		}
	}

	return strings.Join(outputs, "\n"), nil
}

// expandVariables replaces {{variable}} placeholders in the command string.
func expandVariables(command string, vars Variables) string {
	replacements := map[string]string{
		"{{session}}":      vars.Session,
		"{{iteration}}":    vars.Iteration,
		"{{task_id}}":      vars.TaskID,
		"{{task_content}}": vars.TaskContent,
		"{{error}}":        vars.Error,
	}

	result := command
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
