# Specs Index

| Spec | Status | Description |
|------|--------|-------------|
| [iteratr](./iteratr.md) | Draft | AI coding agent orchestrator with embedded NATS and TUI |
| [managed-process-migration](./managed-process-migration.md) | Draft | Replace ACP with managed process for model selection |
| [tui-beautification](./tui-beautification.md) | Complete | Modernize TUI with Crush patterns, Ultraviolet layouts, component interfaces |
| [tui-beautification-cleanup](./tui-beautification-cleanup.md) | Optional | Minor cleanup: legacy Render() removal, taskIndex consistency |
| [test-spec](./test-spec.md) | Active | E2E test spec - validates TUI/task workflow with temp files |
| [session-task-enhancements](./session-task-enhancements.md) | Draft | Task deps/priority, iteration history, snapshots, prompt improvements |
| [state-snapshots](./state-snapshots.md) | Draft | KV snapshots for faster state loading (deferred from session-task-enhancements) |
| [acp-migration](./acp-migration.md) | Draft | Migrate back to ACP protocol for opencode communication |
| [clickable-task-modal](./clickable-task-modal.md) | Draft | Click task in sidebar to open detail modal with bubblezone |
| [agent-message-display](./agent-message-display.md) | Draft | Rich agent message rendering: thinking blocks, tool status, markdown, animations |
| [user-input-acp](./user-input-acp.md) | Draft | Replace inbox with Bubbles textinput, send user messages via persistent ACP session |
| [lint-fixes](./lint-fixes.md) | Active | Run golangci-lint and fix all reported issues |
| [user-note-creation](./user-note-creation.md) | Draft | Ctrl+N modal to create user notes via NATS (same persistence as agent notes) |
| [user-task-creation](./user-task-creation.md) | Draft | Ctrl+T modal to create user tasks via NATS (mirrors notes pattern) |
| [message-queue-enhancement](./message-queue-enhancement.md) | Draft | Replace UI-layer single-message queue with orchestrator-layer FIFO queue (crush pattern) |
| [build-wizard](./build-wizard.md) | Draft | Interactive wizard for `iteratr build` when no spec provided |
| [wizard-session-selector](./wizard-session-selector.md) | Draft | Resume existing sessions or start new from wizard step 0 |
| [hooks](./hooks.md) | Active | Pre-iteration hooks for injecting dynamic context via shell commands |
| [theme-system](./theme-system.md) | Draft | Centralized theme package for consistent colors/styles (crush pattern) |
