package template

// DefaultTemplate is the embedded default prompt template.
// It uses {{variable}} placeholders for dynamic content injection.
const DefaultTemplate = `# iteratr Session
Session: {{session}} | Iteration: #{{iteration}}

{{history}}

## Spec
{{spec}}

{{tasks}}

{{notes}}

## Rules
- ONE task per iteration - complete fully, then STOP
- Use ` + "`" + `{{binary}} tool` + "`" + ` for ALL task management - never built-in todo tools
- Test changes before marking complete
- Write iteration-summary before stopping
- Call session-complete only when ALL tasks done
- Respect user-added tasks even if not in spec

## Workflow
1. If no tasks: sync from spec using task-add tool
2. Pick ONE ready task (highest priority, no blockers) using task-next tool
3. Mark task as in_progress using task-update tool
4. Implement + test
5. Mark task as completed using task-update tool
6. Write iteration-summary using iteration-summary tool
7. STOP (do not pick another task)

## Commands
` + "`" + `{{binary}} tool COMMAND --data-dir .iteratr --name {{session}} [args]` + "`" + `

| Command | Args | Notes |
|---------|------|-------|
| task-add | --content "..." | Add single task |
| task-batch-add | --tasks '[{...}]' | Add multiple tasks |
| task-status | --id X --status S | S: remaining, in_progress, completed, blocked |
| task-priority | --id X --priority N | 0=critical, 1=high, 2=medium, 3=low, 4=backlog |
| task-depends | --id X --depends-on Y | Set dependency |
| note-add | --content "..." --type T | T: learning, stuck, tip, decision |
| iteration-summary | --summary "..." | Record what you did |
| session-complete | | Only when ALL tasks done |

## If Stuck
- ` + "`" + `note-add --type stuck --content "describe issue"` + "`" + `
- Mark task blocked or fix before completing
- If blocked by another task: ` + "`" + `task-depends --id X --depends-on Y` + "`" + `

## Subagents
Spin up subagents (via Task tool) to parallelize work. Each subagent has fresh context, so "one task per agent" is preserved.

**DO parallelize when:**
- Tasks are independent (no shared files)
- Tasks have no uncommitted dependencies between them
- Read-only research while you implement

**DO NOT parallelize when:**
- Tasks modify the same files (causes conflicts)
- Task B depends on Task A's uncommitted changes
- Uncertain about conflicts - err sequential

Mark all delegated tasks in_progress, then completed when subagents finish.
{{extra}}`
