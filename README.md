# iteratr

AI coding agent orchestrator with embedded persistence and TUI.

## Overview

iteratr is a Go CLI tool that orchestrates AI coding agents in an iterative loop. It manages session state (tasks, notes, inbox) via embedded NATS JetStream, communicates with opencode via ACP (Agent Control Protocol) over stdio, and presents a full-screen TUI using Bubbletea v2.

**Spiritual successor to ralph.nu** - same concepts, modern Go implementation.

## Features

- **Session Management**: Named sessions with persistent state across iterations
- **Task System**: Track tasks with status (remaining, in_progress, completed, blocked)
- **Notes System**: Record learnings, tips, blockers, and decisions across iterations
- **Inbox System**: Send messages to running sessions for human-in-the-loop interaction
- **Full-Screen TUI**: Real-time dashboard, task list, logs, notes, inbox, and agent output
- **Embedded NATS**: In-process persistence with JetStream (no external database needed)
- **ACP Integration**: Control opencode agents via Agent Control Protocol
- **Headless Mode**: Run without TUI for CI/CD environments

## Installation

### Prerequisites

- Go 1.25.5 or later
- [opencode](https://opencode.coder.com) installed and in PATH

### Build from Source

```bash
go install github.com/mark3labs/iteratr/cmd/iteratr@latest
```

Or clone and build:

```bash
git clone https://github.com/mark3labs/iteratr.git
cd iteratr
go build -o iteratr cmd/iteratr/*.go
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

### 3. Interact with the Session

While iteratr is running in one terminal, send messages from another:

```bash
# Send a message to the running session
iteratr message --name myfeature "Please add logging to all functions"
```

The agent will receive the message in its next iteration and can respond.

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
- `--headless`: Run without TUI (logging only)
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

# Run in headless mode (no TUI)
iteratr build --headless

# Add extra instructions
iteratr build --extra-instructions "Focus on error handling"
```

#### `iteratr message`

Send a message to a running session's inbox.

```bash
iteratr message --name <session> <message>
```

**Flags:**

- `-n, --name <name>`: Session name (required)
- `--data-dir <path>`: Data directory (default: `.iteratr`)

**Examples:**

```bash
# Send a simple message
iteratr message --name myfeature "Please add more tests"

# Multi-word messages
iteratr message --name myfeature "Change the API endpoint from /api/v1 to /api/v2"
```

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

- **`1`**: Dashboard view - session overview, progress, current task
- **`2`**: Tasks view - filterable task list by status
- **`3`**: Logs view - scrollable event history
- **`4`**: Notes view - learnings, tips, blockers, decisions
- **`5`**: Inbox view - messages and input field
- **`j/k`**: Navigate lists (tasks, logs, etc.)
- **`↑/↓`**: Scroll viewports
- **`q` or `Ctrl+C`**: Quit

## Session State

iteratr maintains session state in the `.iteratr/` directory using embedded NATS JetStream:

```
.iteratr/
├── jetstream/
│   ├── _js_/         # JetStream metadata
│   └── iteratr_events/  # Event stream data
```

All session data (tasks, notes, inbox messages, iterations) is stored as events in a NATS stream. This provides:

- **Persistence**: State survives across runs
- **Resume capability**: Continue from the last iteration
- **Event history**: Full audit trail of all changes
- **Concurrency**: Multiple tools can interact with session data

### Session Tools

The agent has access to these tools during execution:

**Task Management:**
- `task_add(content, status?)` - Create a task
- `task_status(id, status)` - Update task status
- `task_list()` - List all tasks grouped by status

**Notes:**
- `note_add(content, type)` - Record a note (type: learning|stuck|tip|decision)
- `note_list(type?)` - List notes, optionally filtered

**Inbox:**
- `inbox_list()` - Get unread messages
- `inbox_mark_read(id)` - Acknowledge a message

**Session Control:**
- `session_complete()` - Signal all tasks done, end iteration loop

## Prompt Templates

iteratr uses Go template syntax with `{{variable}}` placeholders.

### Available Variables

- `{{session}}` - Session name
- `{{iteration}}` - Current iteration number
- `{{spec}}` - Spec file contents
- `{{inbox}}` - Unread inbox messages
- `{{notes}}` - Notes from previous iterations
- `{{tasks}}` - Current task state
- `{{extra}}` - Extra instructions from `--extra-instructions` flag

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
- **ACP Client**: Communicates with opencode agent via stdio
- **Session Store**: Persists state to NATS JetStream
- **TUI**: Full-screen Bubbletea v2 interface with multiple views
- **Template Engine**: Renders prompts with session state

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

# Send feedback during execution (from another terminal)
iteratr message --name user-auth "Make sure to use bcrypt for password hashing"
```

### Example 2: Resume a Session

```bash
# Initial run (stops after 3 iterations)
iteratr build --spec specs/myfeature.md --iterations 3

# Resume from iteration 4
iteratr build --spec specs/myfeature.md
```

The session automatically resumes from where it left off.

### Example 3: Headless Mode for CI/CD

```bash
# Run in headless mode (useful for CI/CD)
iteratr build --headless --iterations 5 --spec specs/myfeature.md > build.log 2>&1

# Check if session completed
if grep -q "session_complete" build.log; then
  echo "Build successful!"
else
  echo "Build incomplete or failed"
  exit 1
fi
```

### Example 4: Custom Template with Extra Instructions

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
4. **Send messages** if you need to provide guidance or feedback
5. **Review notes** to see what the agent learned
6. **Agent completes** by calling `session_complete()` when all tasks are done

Each iteration:
1. Agent checks inbox for new messages
2. Agent reviews task list and notes from previous iterations
3. Agent picks ONE task, marks it in_progress
4. Agent works on the task (writes code, runs tests)
5. Agent commits changes if successful
6. Agent marks task completed and records any learnings
7. Repeat until all tasks are done

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

# Clean data directory (CAUTION: loses session state)
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
# Build binary
go build -o iteratr cmd/iteratr/*.go

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Project Structure

```
.
├── cmd/iteratr/          # CLI commands
│   ├── main.go           # Entry point
│   ├── build.go          # Build command
│   ├── message.go        # Message command
│   ├── doctor.go         # Doctor command
│   ├── gen_template.go   # Template generation
│   └── version.go        # Version command
├── internal/
│   ├── acp/              # ACP client and tools
│   ├── nats/             # Embedded NATS server
│   ├── session/          # Session state management
│   ├── template/         # Prompt templates
│   ├── tui/              # Bubbletea TUI components
│   ├── orchestrator/     # Iteration loop orchestration
│   ├── logger/           # Logging utilities
│   └── errors/           # Error handling
├── specs/                # Feature specifications
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
