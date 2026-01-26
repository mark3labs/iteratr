# Hooks

## Overview

Pre-iteration hooks allow users to run shell commands before each iteration, injecting dynamic context into the agent prompt.

## User Story

As a developer, I want to run custom scripts before each iteration so the agent receives up-to-date context (git status, test results, build output, etc.).

## Requirements

- Config file: `.iteratr.hooks.yml` in working directory
- Single hook type: `pre_iteration`
- Shell command execution with stdout capture
- Template variable expansion in commands (`{{iteration}}`, `{{session}}`)
- 30 second default timeout
- Graceful error handling (continue iteration with error in output)
- Raw output (no framing headers)

## Config Format

```yaml
version: 1

hooks:
  pre_iteration:
    - command: "git status --short"
      timeout: 5
    - command: "./scripts/context.sh {{iteration}}"
      timeout: 60
```

Single command shorthand:
```yaml
hooks:
  pre_iteration:
    - command: "./scripts/context.sh"
```

## Technical Implementation

### New Package: `internal/hooks/`

**types.go** - Configuration structs:
```go
type Config struct {
    Version int         `yaml:"version"`
    Hooks   HooksConfig `yaml:"hooks"`
}

type HooksConfig struct {
    PreIteration []*HookConfig `yaml:"pre_iteration"`
}

type HookConfig struct {
    Command string `yaml:"command"`
    Timeout int    `yaml:"timeout"` // seconds, default 30
}
```

**hooks.go** - Loading and execution:
- `LoadConfig(workDir string) (*Config, error)` - Load `.iteratr.hooks.yml`, return nil if not found
- `Execute(ctx, hook, workDir, vars)` - Run single command, capture output
- `ExecuteAll(ctx, hooks, workDir, vars)` - Run multiple hooks, concatenate output

### ACP Changes (`internal/agent/acp.go`)

- `prompt()` accepts `[]string` texts instead of single string
- Multiple texts sent as separate content blocks in same request

### Runner Changes (`internal/agent/runner.go`)

- `RunIteration(ctx, prompt, hookOutput)` accepts optional hook output
- Hook output sent as first content block, main prompt as second

### Orchestrator Changes (`internal/orchestrator/orchestrator.go`)

1. Add `hooksConfig *hooks.Config` field to Orchestrator
2. Load hooks config in `Start()` (optional, log if missing)
3. Execute pre-iteration hooks before `BuildPrompt()` call
4. Pass hook output directly to `runner.RunIteration()`

### Error Handling

- Config not found: Skip hooks, continue normally
- Config parse error: Log warning, continue without hooks
- Command failure/timeout: Include error in output, continue iteration
- Context cancelled: Propagate cancellation

### TUI Safety

- Never write hook stderr to `os.Stderr`
- Capture stderr, include in output or log via logger
- Use `cmd.Output()` or pipe-based capture

## Tasks

### 1. Add YAML dependency
- [x] Add `gopkg.in/yaml.v3` to go.mod

### 2. Create hooks package
- [x] Create `internal/hooks/types.go` with Config structs
- [x] Create `internal/hooks/hooks.go` with LoadConfig, Execute, ExecuteAll

### 3. Modify ACP layer for multiple content blocks
- [x] Change `prompt()` to accept `[]string` texts
- [x] Build content blocks from texts array

### 4. Update runner to accept hook output
- [x] Add `hookOutput` parameter to `RunIteration()`
- [x] Send hook output as separate content block before main prompt

### 5. Integrate into orchestrator
- [x] Add hooksConfig field to Orchestrator struct
- [x] Load hooks config in Start() with graceful fallback
- [x] Execute pre-iteration hooks in Run() before BuildPrompt
- [x] Pass hook output to runner.RunIteration()

## Out of Scope

- Post-iteration hooks
- Environment variable injection in config
- Hook-specific working directories

## Open Questions

None - all requirements clarified.
