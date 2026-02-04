# Specs Index

<!-- SPECS -->
| Name | Description | Date |
|------|-------------|------|
| [iteratr](./iteratr.md) | AI coding agent orchestrator with embedded NATS and TUI | 2026-02-02 |
| [managed-process-migration](./managed-process-migration.md) | Replace ACP with managed process for model selection | 2026-02-02 |
| [tui-beautification](./tui-beautification.md) | Modernize TUI with Crush patterns, Ultraviolet layouts, component interfaces | 2026-02-02 |
| [tui-beautification-cleanup](./tui-beautification-cleanup.md) | Minor cleanup: legacy Render() removal, taskIndex consistency | 2026-02-02 |
| [test-spec](./test-spec.md) | E2E test spec - validates TUI/task workflow with temp files | 2026-02-02 |
| [session-task-enhancements](./session-task-enhancements.md) | Task deps/priority, iteration history, snapshots, prompt improvements | 2026-02-02 |
| [state-snapshots](./state-snapshots.md) | KV snapshots for faster state loading (deferred from session-task-enhancements) | 2026-02-02 |
| [acp-migration](./acp-migration.md) | Migrate back to ACP protocol for opencode communication | 2026-02-02 |
| [clickable-task-modal](./clickable-task-modal.md) | Click task in sidebar to open detail modal with bubblezone | 2026-02-02 |
| [agent-message-display](./agent-message-display.md) | Rich agent message rendering: thinking blocks, tool status, markdown, animations | 2026-02-02 |
| [user-input-acp](./user-input-acp.md) | Replace inbox with Bubbles textinput, send user messages via persistent ACP session | 2026-02-02 |
| [lint-fixes](./lint-fixes.md) | Run golangci-lint and fix all reported issues | 2026-02-02 |
| [user-note-creation](./user-note-creation.md) | Ctrl+N modal to create user notes via NATS (same persistence as agent notes) | 2026-02-02 |
| [user-task-creation](./user-task-creation.md) | Ctrl+T modal to create user tasks via NATS (mirrors notes pattern) | 2026-02-02 |
| [message-queue-enhancement](./message-queue-enhancement.md) | Replace UI-layer single-message queue with orchestrator-layer FIFO queue (crush pattern) | 2026-02-02 |
| [build-wizard](./build-wizard.md) | Interactive wizard for `iteratr build` when no spec provided | 2026-02-02 |
| [wizard-session-selector](./wizard-session-selector.md) | Resume existing sessions or start new from wizard step 0 | 2026-02-02 |
| [hooks](./hooks.md) | Pre-iteration hooks for injecting dynamic context via shell commands | 2026-02-02 |
| [extended-hooks](./extended-hooks.md) | Full lifecycle hooks: session_start/end, post_iteration, on_task_complete, on_error, pipe_output | 2026-02-02 |
| [theme-system](./theme-system.md) | Centralized theme package for consistent colors/styles (crush pattern) | 2026-02-02 |
| [file-tracking](./file-tracking.md) | Track modified files during iteration for auto-commit | 2026-02-02 |
| [subagent-viewer](./subagent-viewer.md) | Display subagent calls as clickable messages with session viewer modal | 2026-02-02 |
| [config-management](./config-management.md) | Viper-based config with `iteratr setup` TUI command | 2026-02-02 |
| [mcp-tools-server](./mcp-tools-server.md) | Embedded MCP HTTP server replacing CLI tool injection | 2026-02-02 |
| [session-pause](./session-pause.md) | Pause/resume sessions via ctrl+x p, status bar indicator | 2026-02-02 |
| [git-status-bar](./git-status-bar.md) | Display git branch, hash, dirty state, ahead/behind in status bar | 2026-02-02 |
| [2026-02-02-dead-code-cleanup](./2026-02-02-dead-code-cleanup.md) | Remove ~720 lines of dead code: unused files, legacy methods, orphaned helpers | 2026-02-02 |
| [sidebar-toggle](./sidebar-toggle.md) | Ctrl+x b toggles sidebar visibility, persists preference, responsive behavior | 2026-02-02 |
| [spec-command](./spec-command.md) | `iteratr spec` wizard with AI-assisted interview via opencode acp + custom MCP | 2026-02-02 |
| [coderabbit-pr-fixes](./coderabbit-pr-fixes.md) | Fetch unresolved coderabbitai[bot] PR comments, create tasks, fix issues | 2026-02-04 |
