# iteratr

[![CI](https://github.com/mark3labs/iteratr/actions/workflows/ci.yml/badge.svg)](https://github.com/mark3labs/iteratr/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/mark3labs/iteratr)](https://github.com/mark3labs/iteratr/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/mark3labs/iteratr)](https://goreportcard.com/report/github.com/mark3labs/iteratr)
[![Go Reference](https://pkg.go.dev/badge/github.com/mark3labs/iteratr.svg)](https://pkg.go.dev/github.com/mark3labs/iteratr)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<p align="center">
  <img src="iteratr.gif" alt="iteratr demo" />
</p>

AI coding agent orchestrator with embedded persistence and TUI.

> **Warning:** This project is under active development. APIs, commands, and configuration formats may change without notice. Expect breaking changes between versions until a stable release is announced.
>
> **Warning:** iteratr runs opencode with auto-approve permissions enabled. The agent can execute commands, modify files, and make changes without manual confirmation. Use in trusted environments and review changes carefully.

## Overview

iteratr is a Go CLI tool that orchestrates AI coding agents in an iterative loop. It manages session state (tasks, notes, iterations) via embedded NATS JetStream, communicates with opencode via ACP (Agent Control Protocol) over stdio, and presents a full-screen TUI using Bubbletea v2.

**Spiritual successor to ralph.nu** - same concepts, modern Go implementation.

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

## Installation

### Prerequisites

- [opencode](https://opencode.coder.com) installed and in PATH

### Quick Install (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/mark3labs/iteratr/main/install.sh | sh
```

Install a specific version:

```bash
curl -sSL https://raw.githubusercontent.com/mark3labs/iteratr/main/install.sh | sh -s v1.0.0
```

Install to a custom directory:

```bash
curl -sSL https://raw.githubusercontent.com/mark3labs/iteratr/main/install.sh | INSTALL_DIR=~/.local/bin sh
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

### 1. Create a Spec File

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

### 2. Run the Build Loop

```bash
iteratr build --spec specs/myfeature.md
```

This will:
- Start an embedded NATS server for persistence
- Launch a full-screen TUI
- Load the spec and create tasks
- Run opencode agent in iterative loops
- Track progress and state across iterations

### 3. Interact via TUI

While iteratr is running, type messages directly in the TUI to send guidance or feedback to the agent. The agent will receive the message in its next iteration.

## Usage

### Commands

#### `iteratr build`

Run the iterative agent build loop.

```bash
iteratr build [flags]
```

**Flags:**

- `-n, --name <name>`: Session name (default: spec filename stem)
- `-s, --spec <path>`: Spec file path (default: `./specs/SPEC.md`)
- `-t, --template <path>`: Custom prompt template file
- `-e, --extra-instructions <text>`: Extra instructions for the prompt
- `-i, --iterations <count>`: Max iterations, 0=infinite (default: 0)
- `-m, --model <model>`: Model to use (default: `anthropic/claude-sonnet-4-5`)
- `--headless`: Run without TUI (logging only)
- `--reset`: Reset session data before starting
- `--data-dir <path>`: Data directory for NATS storage (default: `.iteratr`)

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
- `session-complete` - Signal all tasks done, end iteration loop

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

## Environment Variables

- `ITERATR_DATA_DIR` - Data directory for NATS storage (default: `.iteratr`)
- `ITERATR_LOG_FILE` - Log file path for debugging
- `ITERATR_LOG_LEVEL` - Log level: debug, info, warn, error

**Example:**

```bash
# Use custom data directory
export ITERATR_DATA_DIR=/var/lib/iteratr
iteratr build

# Enable debug logging
export ITERATR_LOG_LEVEL=debug
export ITERATR_LOG_FILE=iteratr.log
iteratr build
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

# Use custom template with extra instructions
iteratr build \
  --template team-template.txt \
  --extra-instructions "Follow the error handling patterns in internal/errors/" \
  --spec specs/myfeature.md
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
│   ├── nats/             # Embedded NATS server and stream management
│   ├── session/          # Event-sourced session state
│   ├── template/         # Prompt template engine
│   ├── tui/              # Bubbletea v2 TUI components
│   ├── orchestrator/     # Iteration loop orchestration
│   ├── logger/           # Structured logging
│   └── errors/           # Error handling and retry
├── specs/                # Feature specifications
├── Taskfile.yml          # Task runner configuration
├── .iteratr/             # Session data (gitignored)
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

## Credits

Inspired by ralph.nu - the original AI agent orchestrator in Nushell.
