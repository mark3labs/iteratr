# iteratr

[![CI](https://github.com/mark3labs/iteratr/actions/workflows/ci.yml/badge.svg)](https://github.com/mark3labs/iteratr/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/mark3labs/iteratr)](https://github.com/mark3labs/iteratr/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/mark3labs/iteratr)](https://goreportcard.com/report/github.com/mark3labs/iteratr)
[![Go Reference](https://pkg.go.dev/badge/github.com/mark3labs/iteratr.svg)](https://pkg.go.dev/github.com/mark3labs/iteratr)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<p align="center">
  <img src="iteratr.gif" alt="iteratr demo" />
</p>

Orchestrates AI coding agents in an iterative loop.

> **Warning:** This project is under active development. APIs, commands, and configuration formats may change without notice. Expect breaking changes between versions until a stable release is announced.
>
> **Warning:** iteratr runs opencode with auto-approve permissions enabled. The agent can execute commands, modify files, and make changes without manual confirmation. Use in trusted environments and review changes carefully.

## Features

- **Session Management**: Named sessions with persistent state across iterations
- **Task System**: Track tasks with status, priority (0-4), and dependencies
- **Notes System**: Record learnings, tips, blockers, and decisions across iterations
- **Full-Screen TUI**: Real-time dashboard with agent output, task sidebar, logs, and notes
- **User Input via TUI**: Send messages directly to the agent through the TUI interface
- **Embedded NATS**: In-process persistence with JetStream (no external database needed)
- **ACP Integration**: Control opencode agents via Agent Control Protocol with persistent sessions
- **Headless Mode**: Run without TUI for CI/CD environments
- **Model Selection**: Choose which LLM model to use per session
- **Interactive Wizard**: Guided setup when no spec file provided
- **Lifecycle Hooks**: Run custom scripts at session start/end, before/after iterations, on task completion, and on errors

## Installation

### Prerequisites

- [opencode](https://opencode.coder.com) installed and in PATH

### Quick Install (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/mark3labs/iteratr/refs/heads/master/install.sh | sh
```

Install a specific version:

```bash
curl -sSL https://raw.githubusercontent.com/mark3labs/iteratr/refs/heads/master/install.sh | sh -s v1.0.0
```

Install to a custom directory:

```bash
curl -sSL https://raw.githubusercontent.com/mark3labs/iteratr/refs/heads/master/install.sh | INSTALL_DIR=~/.local/bin sh
```

### bun/npm/pnpm

```bash
bun add -g iteratr
```

### Go Install

Requires Go 1.25+.

```bash
go install github.com/mark3labs/iteratr/cmd/iteratr@latest
```

### Build from Source

```bash
git clone https://github.com/mark3labs/iteratr.git
cd iteratr
task build
```

Or without the task runner:

```bash
go build -o iteratr ./cmd/iteratr
```

### Verify Installation

```bash
iteratr doctor
```

This checks that opencode and other dependencies are available.

## Quick Start

### 1. Initial Setup

First-time setup creates a config file with your preferred model:

```bash
iteratr setup
```

This launches an interactive wizard that saves settings to `~/.config/iteratr/iteratr.yml`. For project-specific config, use `--project` to create `./iteratr.yml` instead.

### 2. Create a Spec File

Create a spec file at `specs/myfeature.md`:

```markdown
# My Feature

## Overview
Build a user authentication system.

## Requirements
- User login/logout
- Password hashing
- Session management

## Tasks
- [ ] Create user model
- [ ] Implement login endpoint
- [ ] Add session middleware
- [ ] Write tests
```

### 3. Run the Build Loop

```bash
iteratr build --spec specs/myfeature.md
```

This will:
- Start an embedded NATS server for persistence
- Launch a full-screen TUI
- Load the spec and create tasks
- Run opencode agent in iterative loops
- Track progress and state across iterations

### 4. Interact via TUI

While iteratr is running, type messages directly in the TUI to send guidance or feedback to the agent. The agent will receive the message in its next iteration.

### Alternative: Interactive Wizard

Run without `--spec` to launch the interactive build wizard:

```bash
iteratr build
```

The wizard guides you through 4 steps:
1. **File Picker** - Browse and select a spec file
2. **Model Selector** - Choose an LLM model (fuzzy search supported)
3. **Template Editor** - Customize the prompt template
4. **Config** - Set session name and max iterations

## Configuration

iteratr uses layered configuration with files, environment variables, and CLI flags.

### Config Files

- **Global**: `~/.config/iteratr/iteratr.yml` (or `$XDG_CONFIG_HOME/iteratr/iteratr.yml`)
- **Project**: `./iteratr.yml` (current directory)

### Precedence

CLI flags > ENV vars > project config > global config > defaults

### Setup Command

Create initial configuration:

```bash
# Create global config at ~/.config/iteratr/iteratr.yml
iteratr setup

# Create project config at ./iteratr.yml
iteratr setup --project

# Overwrite existing config
iteratr setup --force
```

The wizard prompts for:
1. **Model selection** - Choose from opencode models or enter custom model ID
2. **Auto-commit** - Whether to automatically commit changes after iterations

### Config Schema

```yaml
# iteratr.yml
model: ""              # required (or ITERATR_MODEL env var)
auto_commit: true      # auto-commit after iterations
data_dir: .iteratr     # NATS/session storage
log_level: info        # debug, info, warn, error
log_file: ""           # empty = no file logging
iterations: 0          # 0 = infinite
headless: false        # run without TUI
template: ""           # path to template file, empty = embedded default
```

### View Current Config

```bash
iteratr config
```

Shows resolved configuration with all sources merged.

## Usage

### Commands

#### `iteratr setup`

Create or update configuration file.

```bash
iteratr setup [flags]
```

**Flags:**

- `-p, --project`: Create config in current directory (./iteratr.yml)
- `-f, --force`: Overwrite existing config

**Examples:**

```bash
# Interactive setup (global config)
iteratr setup

# Create project-specific config
iteratr setup --project

# Overwrite existing config
iteratr setup --force
```

#### `iteratr config`

Display current configuration with all sources merged.

```bash
iteratr config
```

Shows the resolved config values from all layers (files, env vars, defaults).

#### `iteratr build`

Run the iterative agent build loop.

```bash
iteratr build [flags]
```

**Flags:**

- `-n, --name <name>`: Session name (default: spec filename stem)
- `-s, --spec <path>`: Spec file path (default: `./specs/SPEC.md`)
- `-t, --template <path>`: Custom prompt template file (overrides config)
- `-e, --extra-instructions <text>`: Extra instructions for the prompt
- `-i, --iterations <count>`: Max iterations, 0=infinite (overrides config)
- `-m, --model <model>`: Model to use (overrides config, required if not in config/env)
- `--headless`: Run without TUI (overrides config)
- `--auto-commit`: Auto-commit changes after iterations (overrides config)
- `--reset`: Reset session data before starting
- `--data-dir <path>`: Data directory for NATS storage (overrides config)

**Examples:**

```bash
# Basic usage with default spec
iteratr build

# Specify a custom spec
iteratr build --spec specs/myfeature.md

# Run with custom session name
iteratr build --name my-session --spec specs/myfeature.md

# Run 5 iterations then stop
iteratr build --iterations 5

# Use a specific model
iteratr build --model anthropic/claude-sonnet-4-5

# Run in headless mode (no TUI)
iteratr build --headless

# Reset session and start fresh
iteratr build --reset

# Add extra instructions
iteratr build --extra-instructions "Focus on error handling"
```

#### `iteratr tool`

Session management subcommands used by the agent during execution. These are invoked as opencode tools.

```bash
iteratr tool <subcommand> [flags]
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `task-add` | Add a single task |
| `task-batch-add` | Add multiple tasks at once |
| `task-status` | Update task status |
| `task-priority` | Set task priority (0-4) |
| `task-depends` | Add task dependency |
| `task-list` | List all tasks grouped by status |
| `task-next` | Get next highest priority unblocked task |
| `note-add` | Record a note |
| `note-list` | List notes |
| `iteration-summary` | Record an iteration summary |
| `session-complete` | Signal all tasks done, end loop |

#### `iteratr gen-template`

Export the default prompt template to a file for customization.

```bash
iteratr gen-template [flags]
```

**Flags:**

- `-o, --output <path>`: Output file (default: `.iteratr.template`)

**Example:**

```bash
# Generate template
iteratr gen-template

# Customize the template
vim .iteratr.template

# Use custom template in build
iteratr build --template .iteratr.template
```

#### `iteratr doctor`

Check dependencies and environment.

```bash
iteratr doctor
```

Verifies:
- opencode is installed and in PATH
- Go version
- Environment requirements

#### `iteratr version`

Show version information.

```bash
iteratr version
```

Displays version, commit hash, and build date.

## TUI Navigation

When running with the TUI (default), use these keys:

- **`Ctrl+C`**: Quit
- **`Ctrl+L`**: Toggle logs overlay
- **`Ctrl+S`**: Toggle sidebar (compact mode)
- **`Tab`**: Cycle focus between Agent → Tasks → Notes panes
- **`i`**: Focus input field (type messages to the agent)
- **`Enter`**: Submit input message (when input focused)
- **`Esc`**: Exit input field / close modal
- **`j/k`**: Navigate lists (when sidebar focused)

Footer buttons (mouse-clickable) switch between Dashboard, Logs, and Notes views.

## Session State

iteratr maintains session state in the `.iteratr/` directory using embedded NATS JetStream:

```
.iteratr/
├── jetstream/
│   ├── _js_/         # JetStream metadata
│   └── iteratr_events/  # Event stream data
```

All session data (tasks, notes, iterations) is stored as events in a NATS stream. This provides:

- **Persistence**: State survives across runs
- **Resume capability**: Continue from the last iteration
- **Event history**: Full audit trail of all changes
- **Concurrency**: Multiple tools can interact with session data

### Session Tools

The agent has access to these tools during execution (via `iteratr tool` subcommands):

**Task Management:**
- `task-add` - Create a task with content and optional status
- `task-batch-add` - Create multiple tasks at once
- `task-status` - Update task status (remaining, in_progress, completed, blocked)
- `task-priority` - Set task priority (0=lowest, 4=highest)
- `task-depends` - Add a dependency between tasks
- `task-list` - List all tasks grouped by status
- `task-next` - Get next highest priority unblocked task

**Notes:**
- `note-add` - Record a note (type: learning|stuck|tip|decision)
- `note-list` - List notes, optionally filtered by type

**Iteration:**
- `iteration-summary` - Record a summary of what was accomplished

**Session Control:**
- `session-complete` - Signal all tasks done, end iteration loop (validates all tasks are complete)

## Prompt Templates

iteratr uses Go template syntax with `{{variable}}` placeholders.

### Available Variables

- `{{session}}` - Session name
- `{{iteration}}` - Current iteration number
- `{{spec}}` - Spec file contents
- `{{notes}}` - Notes from previous iterations
- `{{tasks}}` - Current task state
- `{{history}}` - Iteration history/summaries
- `{{extra}}` - Extra instructions from `--extra-instructions` flag
- `{{port}}` - NATS server port
- `{{binary}}` - Path to iteratr binary

### Custom Templates

Generate the default template:

```bash
iteratr gen-template -o my-template.txt
```

Edit the template, then use it:

```bash
iteratr build --template my-template.txt
```

## Lifecycle Hooks

Run custom scripts at different points in the session lifecycle to inject dynamic context, run validations, send notifications, or handle errors.

### Configuration

Create `.iteratr.hooks.yml` in your working directory:

```yaml
version: 1

hooks:
  session_start:
    - command: "git pull --rebase"
      timeout: 30
    - command: "go build ./..."
      timeout: 60
      pipe_output: true  # Send build errors to agent

  pre_iteration:
    - command: "golangci-lint run ./..."
      timeout: 30
      pipe_output: true  # Agent sees lint errors and can fix them

  post_iteration:
    - command: "go test ./... -short"
      timeout: 120
      pipe_output: true  # Agent sees test failures next iteration
    - command: 'curl -X POST $SLACK_WEBHOOK -d "{\"text\":\"Iteration done\"}"'
      timeout: 5
      # pipe_output: false (default) - just notification

  session_end:
    - command: "git push origin HEAD"
      timeout: 30
    - command: "./scripts/notify-complete.sh {{session}}"
      timeout: 10

  on_task_complete:
    - command: "./scripts/validate-task.sh {{task_id}}"
      timeout: 30
      pipe_output: true  # Send validation results to agent

  on_error:
    - command: "git diff HEAD"
      timeout: 10
      pipe_output: true  # Show agent what changed before error
```

### Hook Types

| Hook | When | Use Case |
|------|------|----------|
| `session_start` | Once, before first iteration | Pull latest code, verify dependencies |
| `pre_iteration` | Before each iteration | Run linters, formatters, checks |
| `post_iteration` | After each iteration completes | Run tests, send notifications |
| `session_end` | Once, after session completes | Push code, send completion alerts |
| `on_task_complete` | When task status → completed | Validate task completion |
| `on_error` | On any iteration failure | Gather diagnostics, show diff |

### Hook Options

- `command` - Shell command to execute (supports template variables)
- `timeout` - Timeout in seconds (default: 30)
- `pipe_output` - Send output to agent (default: false)

### Template Variables

Available in hook commands:

- `{{session}}` - Session name (all hooks)
- `{{iteration}}` - Current iteration number (pre_iteration, post_iteration, on_error)
- `{{task_id}}` - Completed task ID (on_task_complete)
- `{{task_content}}` - Completed task content (on_task_complete)
- `{{error}}` - Error message (on_error)

### Output Piping

When `pipe_output: true`, hook output is sent to the agent:

- **session_start**: Output held until first iteration starts
- **pre_iteration**: Output prepended to iteration prompt
- **post_iteration**: Output held for next iteration
- **on_task_complete**: Output accumulated and sent at next iteration
- **on_error**: Output sent immediately in recovery prompt
- **session_end**: Output not piped (no more iterations)

This allows the agent to see test failures, lint errors, or build issues and fix them automatically.

### Error Handling

- Config not found: hooks skipped, iteration continues
- Command failure/timeout: error included in output, iteration continues
- Hook failures never stop the session

## Environment Variables

All config keys can be set via environment variables with the `ITERATR_` prefix:

| Config Key | ENV Var | Type | Default |
|------------|---------|------|---------|
| `model` | `ITERATR_MODEL` | string | (required) |
| `auto_commit` | `ITERATR_AUTO_COMMIT` | bool | `true` |
| `data_dir` | `ITERATR_DATA_DIR` | string | `.iteratr` |
| `log_level` | `ITERATR_LOG_LEVEL` | string | `info` |
| `log_file` | `ITERATR_LOG_FILE` | string | `""` |
| `iterations` | `ITERATR_ITERATIONS` | int | `0` |
| `headless` | `ITERATR_HEADLESS` | bool | `false` |
| `template` | `ITERATR_TEMPLATE` | string | `""` |

Environment variables override config file values but are overridden by CLI flags.

**Examples:**

```bash
# Set model via environment (useful for CI/CD)
export ITERATR_MODEL=anthropic/claude-opus-4
iteratr build

# Use custom data directory
export ITERATR_DATA_DIR=/var/lib/iteratr
iteratr build

# Enable debug logging
export ITERATR_LOG_LEVEL=debug
export ITERATR_LOG_FILE=iteratr.log
iteratr build

# Run without config file (CI/CD mode)
export ITERATR_MODEL=anthropic/claude-sonnet-4-5
export ITERATR_HEADLESS=true
export ITERATR_ITERATIONS=5
iteratr build --spec specs/myfeature.md
```

## Architecture

```
+------------------+       ACP/stdio        +------------------+
|     iteratr      | <-------------------> |     opencode     |
|                  |                       |                  |
|  +------------+  |                       |  +------------+  |
|  | Bubbletea  |  |                       |  |   Agent    |  |
|  |    TUI     |  |                       |  +------------+  |
|  +------------+  |                       +------------------+
|        |         |
|  +------------+  |
|  |    ACP     |  |
|  |   Client   |  |
|  +------------+  |
|        |         |
|  +------------+  |
|  |   NATS     |  |
|  | JetStream  |  |
|  | (embedded) |  |
|  +------------+  |
+------------------+
```

### Key Components

- **Orchestrator**: Manages iteration loop and coordinates components
- **ACP Client**: Communicates with opencode agent via stdio (persistent sessions)
- **Session Store**: Event-sourced state persisted to NATS JetStream
- **TUI**: Full-screen Bubbletea v2 interface with Ultraviolet layouts and Glamour markdown rendering
- **Template Engine**: Renders prompts with session state variables

## Examples

### Example 1: Basic Feature Development

```bash
# Create a spec
cat > specs/user-auth.md <<EOF
# User Authentication

## Tasks
- [ ] Create User model
- [ ] Add login endpoint
- [ ] Add logout endpoint
- [ ] Write integration tests
EOF

# Run the build loop
iteratr build --spec specs/user-auth.md --iterations 10
```

### Example 2: Resume a Session

```bash
# Initial run (stops after 3 iterations)
iteratr build --spec specs/myfeature.md --iterations 3

# Resume from iteration 4
iteratr build --spec specs/myfeature.md
```

The session automatically resumes from where it left off.

### Example 3: Fresh Start with Reset

```bash
# Reset and start over
iteratr build --spec specs/myfeature.md --reset
```

### Example 4: Headless Mode for CI/CD

```bash
# Run in headless mode (useful for CI/CD)
iteratr build --headless --iterations 5 --spec specs/myfeature.md > build.log 2>&1
```

### Example 5: Custom Template with Extra Instructions

```bash
# Generate template
iteratr gen-template -o team-template.txt

# Edit template to add team-specific guidelines
vim team-template.txt

# Configure template in config file
cat >> iteratr.yml <<EOF
template: team-template.txt
EOF

# Or use --template flag to override config
iteratr build \
  --template team-template.txt \
  --extra-instructions "Follow the error handling patterns in internal/errors/" \
  --spec specs/myfeature.md
```

### Example 6: Project-Specific Configuration

```bash
# Create project config with team settings
iteratr setup --project

# Edit to customize for this project
cat >> ./iteratr.yml <<EOF
model: anthropic/claude-opus-4
auto_commit: false
iterations: 10
template: .team-template
EOF

# All team members use the same config
git add iteratr.yml
git commit -m "Add iteratr project config"

# Builds use project config automatically
iteratr build --spec specs/myfeature.md
```

## Workflow

The recommended workflow with iteratr:

1. **Create a spec** with clear requirements and tasks
2. **Run `iteratr build`** to start the iteration loop
3. **Monitor progress** in the TUI dashboard
4. **Send messages** via TUI if you need to provide guidance
5. **Review notes** to see what the agent learned
6. **Agent completes** by calling `session-complete` when all tasks are done

Each iteration:
1. Agent reviews task list and notes from previous iterations
2. Agent picks next highest priority unblocked task
3. Agent marks task in_progress
4. Agent works on the task (writes code, runs tests)
5. Agent commits changes if successful
6. Agent marks task completed and records any learnings
7. Agent records an iteration summary
8. Repeat until all tasks are done

## Troubleshooting

### opencode not found

```bash
# Check if opencode is installed
which opencode

# Install opencode
# Visit https://opencode.coder.com for installation instructions
```

### Session won't start

```bash
# Check doctor output
iteratr doctor

# Reset session data
iteratr build --reset

# Or clean data directory manually (CAUTION: loses session state)
rm -rf .iteratr
```

### Agent not responding

```bash
# Check if opencode is working
opencode --version

# Enable debug logging
export ITERATR_LOG_LEVEL=debug
export ITERATR_LOG_FILE=debug.log
iteratr build
tail -f debug.log
```

### TUI rendering issues

```bash
# Try headless mode
iteratr build --headless

# Check terminal size
echo $TERM
tput cols
tput lines
```

## Development

### Building

```bash
# Using task runner (recommended)
task build

# Or directly with go
go build -o iteratr ./cmd/iteratr

# Run tests
task test

# Run tests with coverage
task test-coverage

# Lint
task lint

# Full CI check
task ci
```

### Project Structure

```
.
├── cmd/iteratr/          # CLI commands
│   ├── main.go           # Entry point with Cobra root command
│   ├── build.go          # Build command
│   ├── tool.go           # Tool subcommands (task, note, session)
│   ├── doctor.go         # Doctor command
│   ├── gen_template.go   # Template generation
│   └── version.go        # Version command
├── internal/
│   ├── agent/            # ACP client and agent runner
│   ├── hooks/            # Pre-iteration hook execution
│   ├── nats/             # Embedded NATS server and stream management
│   ├── session/          # Event-sourced session state
│   ├── template/         # Prompt template engine
│   ├── tui/              # Bubbletea v2 TUI components
│   │   ├── theme/        # Theme system (Catppuccin Mocha)
│   │   └── wizard/       # Interactive build wizard
│   ├── orchestrator/     # Iteration loop orchestration
│   ├── logger/           # Structured logging
│   └── errors/           # Error handling and retry
├── specs/                # Feature specifications
├── Taskfile.yml          # Task runner configuration
├── .iteratr/             # Session data (gitignored)
├── .iteratr.hooks.yml    # Pre-iteration hooks config (optional)
└── README.md
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Links

- **Repository**: https://github.com/mark3labs/iteratr
- **opencode**: https://opencode.coder.com
- **ACP Protocol**: https://github.com/coder/acp
- **Bubbletea**: https://github.com/charmbracelet/bubbletea
- **NATS**: https://nats.io


