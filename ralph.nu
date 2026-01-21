#!/usr/bin/env nu

# ralph.nu - AI coding agent in a while loop. Named after Ralph Wiggum from The Simpsons.

# ─────────────────────────────────────────────────────────────────────────────
# Template constants for custom prompt templates
# ─────────────────────────────────────────────────────────────────────────────

const DEFAULT_TEMPLATE_PATH = ".ralph.template"

const DEFAULT_TEMPLATE = '## Context
Session: {{session}} | Iteration: #{{iteration}}
Spec: {{spec}}
{{inbox}}{{notes}}
## Task State
{{tasks}}

## Tools - all require session_name="{{session}}"
- inbox_list / inbox_mark_read(id) - check/ack messages
- task_add(content, status?) / task_status(id, status) / task_list - manage tasks
- note_add(content, type) / note_list(type?) - record learnings/tips/blockers/decisions
- session_complete - call when ALL tasks done

## Workflow
1. Check inbox, mark read after processing
2. Ensure all spec tasks exist in task list
3. Pick ONE task, mark in_progress, do work, mark completed
4. Run tests, commit with clear message
5. If stuck/learned something: note_add
6. When ALL done: session_complete("{{session}}")

Rules: ONE task/iteration. Test before commit. Call session_complete to end - do NOT just print a message.
{{extra}}'

# Substitute {{variable}} placeholders in template
def apply-template [
  template: string
  vars: record  # {session, iteration, spec, inbox, notes, tasks, extra}
] {
  $template
    | str replace --all "{{session}}" $vars.session
    | str replace --all "{{iteration}}" ($vars.iteration | into string)
    | str replace --all "{{spec}}" $vars.spec
    | str replace --all "{{inbox}}" $vars.inbox
    | str replace --all "{{notes}}" $vars.notes
    | str replace --all "{{tasks}}" $vars.tasks
    | str replace --all "{{extra}}" $vars.extra
}

# Resolve template: --template flag > .ralph.template > hardcoded default
def resolve-template [
  template_flag?: string  # Explicit --template path
] {
  if ($template_flag | is-not-empty) {
    if not ($template_flag | path exists) {
      error make {msg: $"Template file not found: ($template_flag)"}
    }
    return (open $template_flag)
  }
  
  if ($DEFAULT_TEMPLATE_PATH | path exists) {
    return (open $DEFAULT_TEMPLATE_PATH)
  }
  
  $DEFAULT_TEMPLATE
}

# ─────────────────────────────────────────────────────────────────────────────
# Styling helpers - consistent colored output throughout the script
# ─────────────────────────────────────────────────────────────────────────────

# Style definitions using ansi codes
def "style reset" [] { ansi reset }
def "style bold" [] { ansi attr_bold }
def "style dim" [] { ansi grey }
def "style header" [] { ansi cyan_bold }
def "style success" [] { ansi green_bold }
def "style error" [] { ansi red_bold }
def "style warn" [] { ansi yellow }
def "style info" [] { ansi blue }
def "style url" [] { ansi cyan_underline }
def "style label" [] { ansi white_dimmed }
def "style value" [] { ansi white_bold }
def "style section" [] { ansi magenta_bold }

# Print a styled header banner
def print-banner [] {
  print $"(style header)╭─────────────────────────────────────╮(style reset)"
  print $"(style header)│(style reset)              (style bold)ralph.nu(style reset)               (style header)│(style reset)"
  print $"(style header)│(style reset)      (style dim)AI agent in a while loop(style reset)       (style header)│(style reset)"
  print $"(style header)╰─────────────────────────────────────╯(style reset)"
}

# Print key-value info line
def print-kv [key: string, value: string] {
  print $"  (style label)($key):(style reset) (style value)($value)(style reset)"
}

# Print success message
def print-ok [msg: string] {
  print $"  (style success)✓(style reset) ($msg)"
}

# Print error message  
def print-err [msg: string] {
  print $"  (style error)✗(style reset) ($msg)"
}

# Print info/status message
def print-status [msg: string] {
  print $"  (style info)→(style reset) ($msg)"
}

# Print section header
def print-section [title: string] {
  print $"\n(style section)── ($title) ──(style reset)"
}

# ─────────────────────────────────────────────────────────────────────────────
# Requirements checking
# ─────────────────────────────────────────────────────────────────────────────

# Check if a command exists
def cmd-exists [cmd: string] {
  (which $cmd | is-not-empty)
}

# Check required tools, returns list of missing tools
def check-requirements [
  --include-ngrok  # Also check for ngrok
] {
  let required = ["xs", "opencode", "bun"]
  let optional = if $include_ngrok { ["ngrok"] } else { [] }
  let all_tools = ($required | append $optional)
  
  $all_tools | where {|cmd| not (cmd-exists $cmd) }
}

# Run doctor checks and print status for each tool
def run-doctor [
  --include-ngrok  # Also check for ngrok
] {
  let tools = [
    {name: "xs", desc: "Event store", url: "https://github.com/cablehead/xs"}
    {name: "opencode", desc: "AI coding agent", url: "https://opencode.ai"}
    {name: "bun", desc: "JavaScript runtime", url: "https://bun.sh"}
  ]
  
  let tools = if $include_ngrok {
    $tools | append {name: "ngrok", desc: "Remote tunnel (optional)", url: "https://ngrok.com"}
  } else {
    $tools
  }
  
  mut all_ok = true
  
  for tool in $tools {
    if (cmd-exists $tool.name) {
      let version = try {
        let raw = (run-external $tool.name "--version" | complete | get stdout | str trim | split row "\n" | first)
        # Strip leading tool name and "version" word if present (e.g., "ngrok version 3.31.0" -> "3.31.0")
        $raw | str replace -r '^[\w-]+ +(version +)?' ''
      } catch { "unknown" }
      print-ok $"($tool.name) ($version)"
    } else {
      print-err $"($tool.name) not found - ($tool.desc)"
      print $"     (style url)($tool.url)(style reset)"
      $all_ok = false
    }
  }
  
  $all_ok
}

# ─────────────────────────────────────────────────────────────────────────────

# Kill processes matching a pattern (helper to reduce duplication)
def kill-matching [
  pattern: string     # Pattern to match with pgrep/pkill -f
  description: string # Human-readable description for status message
] {
  let result = (pgrep -f $pattern | complete)
  if $result.exit_code == 0 and ($result.stdout | str trim | is-not-empty) {
    print-status $"Killing existing ($description)..."
    pkill -f $pattern
    sleep 100ms
  }
}

# Poll until condition is met or timeout (generic retry helper)
# Returns: result from check_fn on success, or null on timeout
def poll-until [
  check_fn: closure   # Closure that returns {ok: bool, value: any}
  --attempts: int = 30
  --delay: duration = 100ms
  --error-msg: string = "Operation timed out"
] {
  for attempt in 0..($attempts) {
    let result = (do $check_fn)
    if $result.ok { return $result.value }
    sleep $delay
  }
  error make {msg: $error_msg}
}

# Kill any existing processes for this session
def kill-existing [
  store_path: string  # Path to the store directory
  port: int           # Web server port
] {
  kill-matching $"xs serve ($store_path)" $"xs serve for (style value)($store_path)(style reset)"
  kill-matching $"opencode serve --port ($port)" $"opencode serve on port (style value)($port)(style reset)"
  kill-matching $"ngrok http ($port)" $"ngrok on port (style value)($port)(style reset)"
}

# Start xs store server as background job
def start-store [
  store_path: string  # Path to the store directory
] {
  mkdir $store_path
  print-status $"Starting xs store at (style value)($store_path)(style reset)..."
  
  let job_id = (job spawn { xs serve $store_path })
  
  poll-until {|| 
    let r = (xs version $store_path | complete)
    {ok: ($r.exit_code == 0), value: $job_id}
  } --error-msg "xs store failed to start after 3 seconds"
  
  print-ok "xs store is ready"
  $job_id
}

# Start opencode serve server as background job
def start-web [
  port: int  # Port for the web server
] {
  print-status $"Starting opencode serve on port (style value)($port)(style reset)..."
  
  let job_id = (job spawn { opencode serve --port $port })
  let url = $"http://localhost:($port)"
  
  poll-until {||
    let r = (curl -s -o /dev/null -w "%{http_code}" $url | complete)
    {ok: ($r.exit_code == 0 and ($r.stdout | into int) < 500), value: {job_id: $job_id, url: $url}}
  } --error-msg "opencode serve failed to start after 3 seconds"
  
  print-ok $"opencode serve ready at (style url)($url)(style reset)"
  {job_id: $job_id, url: $url}
}

# Start ngrok tunnel as background job
def start-ngrok [
  port: int           # Local port to forward
  password: string    # Basic auth password
  domain?: string     # Optional custom domain
] {
  let pw_len = ($password | str length)
  if $pw_len < 8 or $pw_len > 128 {
    error make {msg: $"ngrok password must be 8-128 characters, got ($pw_len)"}
  }
  
  print-status "Starting ngrok tunnel..."
  
  let auth = $"ralph:($password)"
  let port_str = ($port | into string)
  let job_id = if ($domain | is-not-empty) {
    job spawn { ngrok http $port_str --basic-auth $auth --domain $domain }
  } else {
    job spawn { ngrok http $port_str --basic-auth $auth }
  }
  
  let result = (poll-until {||
    try {
      let response = (http get http://localhost:4040/api/tunnels)
      let tunnels = ($response.tunnels? | default [])
      if ($tunnels | is-not-empty) {
        {ok: true, value: {job_id: $job_id, url: $tunnels.0.public_url}}
      } else {
        {ok: false, value: null}
      }
    } catch {
      {ok: false, value: null}
    }
  } --delay 500ms --error-msg "ngrok failed to start after 15 seconds")
  
  print-ok "ngrok tunnel ready"
  print $"     (style url)($result.url)(style reset)"
  print $"     (style dim)auth: (style warn)ralph:($password)(style reset)"
  $result
}

# Cleanup function to kill all spawned jobs
def cleanup [
  job_ids: list<int>  # List of job IDs to kill
] {
  print $"\n(style dim)Cleaning up background jobs...(style reset)"
  
  for job_id in $job_ids {
    try {
      job kill $job_id
      print $"  (style dim)killed job ($job_id)(style reset)"
    } catch {
      # Job may have already exited - ignore errors
    }
  }
}

# Cleanup all currently running jobs
def cleanup-all [] {
  let jobs = (job list | get id)
  if ($jobs | is-not-empty) {
    cleanup $jobs
  }
}

# Log iteration start event to xs store
def log-iteration-start [
  store_path: string  # Path to the store directory
  name: string        # Session name
  iteration: int      # Iteration number
] {
  let topic = $"ralph.($name).iteration"
  let timestamp = (date now | format date "%Y-%m-%dT%H:%M:%S%z")
  let meta = {action: "start", n: $iteration, timestamp: $timestamp} | to json -r
  
  echo "" | xs append $store_path $topic --meta $meta | ignore
}

# Log iteration complete event to xs store
def log-iteration-complete [
  store_path: string  # Path to the store directory
  name: string        # Session name
  iteration: int      # Iteration number
  status: string      # Status: "success" or "failure"
] {
  let topic = $"ralph.($name).iteration"
  let timestamp = (date now | format date "%Y-%m-%dT%H:%M:%S%z")
  let meta = {action: "complete", n: $iteration, status: $status, timestamp: $timestamp} | to json -r
  
  echo "" | xs append $store_path $topic --meta $meta | ignore
}

# Compute current task state from append-only log using reduce pattern
# Events: add (creates task), status (changes task status by ID)
# Returns record with tasks grouped by status
def get-task-state [
  store_path: string  # Path to the store directory
  name: string        # Session name
] {
  let topic = $"ralph.($name).task"
  
  # Get all frames from the topic
  let frames = (xs cat $store_path | from json --objects | where topic == $topic)
  
  if ($frames | is-empty) {
    return {completed: [], in_progress: [], blocked: [], remaining: []}
  }
  
  # Use reduce to build state machine - tasks keyed by ID
  let tasks = ($frames | reduce -f {} {|frame, state|
    let action = ($frame.meta.action? | default "add")
    
    match $action {
      "add" => {
        # New task: store content and initial status
        let content = (xs cas $store_path $frame.hash)
        let status = ($frame.meta.status? | default "remaining")
        let iteration = ($frame.meta.iteration? | default null)
        $state | upsert $frame.id {
          id: $frame.id
          content: $content
          status: $status
          iteration: $iteration
        }
      }
      "status" => {
        # Status change: update existing task by ID (supports prefix matching)
        let target_id = $frame.meta.id
        let new_status = $frame.meta.status
        let iteration = ($frame.meta.iteration? | default null)
        # Find task by exact match or prefix (8+ chars)
        let matching_id = ($state | columns | where {|id| $id == $target_id or ($id | str starts-with $target_id)} | first | default null)
        if ($matching_id | is-not-empty) {
          $state | upsert $matching_id {|task|
            $task | get $matching_id | upsert status $new_status | upsert iteration $iteration
          }
        } else {
          $state
        }
      }
      _ => $state
    }
  })
  
  # Convert from {id: task} record to grouped lists by status
  let task_list = ($tasks | values)
  let grouped = if ($task_list | is-empty) { {} } else { $task_list | group-by status }
  
  # Return with all categories (empty lists for missing)
  {
    completed: ($grouped | get -o completed | default [])
    in_progress: ($grouped | get -o in_progress | default [])
    blocked: ($grouped | get -o blocked | default [])
    remaining: ($grouped | get -o remaining | default [])
  }
}

# Get note state from append-only log
# Notes track learnings, tips, blockers, and decisions across iterations
def get-note-state [
  store_path: string  # Path to the store directory
  name: string        # Session name
] {
  let topic = $"ralph.($name).note"
  let frames = (xs cat $store_path | from json --objects | where topic == $topic)
  
  if ($frames | is-empty) { return [] }
  
  $frames | each {|frame|
    let content = (xs cas $store_path $frame.hash)
    {
      id: $frame.id
      type: ($frame.meta.type? | default "note")
      iteration: ($frame.meta.iteration? | default null)
      content: $content
    }
  }
}

# Get inbox state (unread messages) from append-only log using reduce pattern
# Events: add (creates message with unread status), mark_read (changes status by ID)
def get-inbox-state [
  store_path: string  # Path to the store directory
  name: string        # Session name
] {
  let topic = $"ralph.($name).inbox"
  let frames = (xs cat $store_path | from json --objects | where topic == $topic)
  
  if ($frames | is-empty) { return [] }
  
  # Use reduce to build state machine - messages keyed by ID
  let messages = ($frames | reduce -f {} {|frame, state|
    let action = ($frame.meta.action? | default "add")
    
    if $action == "mark_read" {
      # Status change: mark existing message as read by ID (supports prefix matching)
      let target_id = $frame.meta.id
      # Find message by exact match or prefix (8+ chars)
      let matching_id = ($state | columns | where {|id| $id == $target_id or ($id | str starts-with $target_id)} | first | default null)
      if ($matching_id | is-not-empty) {
        $state | upsert $matching_id {|m|
          $m | get $matching_id | upsert status "read"
        }
      } else {
        $state
      }
    } else if $action == "add" or ($frame.meta.status? == "unread") {
      # New message: store content and status
      let content = (xs cas $store_path $frame.hash)
      $state | upsert $frame.id {
        id: $frame.id
        content: $content
        status: "unread"
        timestamp: ($frame.meta.timestamp? | default "")
      }
    } else {
      $state
    }
  })
  
  # Return only unread messages
  $messages | values | where status == "unread"
}

# Format inbox messages for prompt injection
def format-inbox-for-prompt [messages: list] {
  if ($messages | is-empty) { return "" }
  
  mut lines = ["## INBOX (Unread Messages - Process these first!)"]
  for msg in $messages {
    let id_short = ($msg.id | str substring 0..8)
    $lines = ($lines | append $"- [($id_short)] ($msg.timestamp): ($msg.content)")
  }
  $lines = ($lines | append "")
  $lines = ($lines | append "After reading, call inbox_mark_read(session_name, id) for each message.")
  $lines | str join "\n"
}

# Note type ordering and colors
const NOTE_TYPES = ["stuck", "learning", "tip", "decision"]
const NOTE_COLORS = {stuck: "red", learning: "green", tip: "cyan", decision: "yellow"}

# Format notes for prompt injection
# Only shows notes from previous iterations (not current)
def format-notes-for-prompt [notes: list, current_iteration: int] {
  let prev_notes = ($notes | where {|n| 
    $n.iteration != null and $n.iteration < $current_iteration
  })
  if ($prev_notes | is-empty) { return "" }
  
  let grouped = ($prev_notes | group-by type)
  mut lines = ["## Notes from Previous Iterations"]
  
  for type in $NOTE_TYPES {
    let type_notes = ($grouped | get -o $type | default [])
    if ($type_notes | length) > 0 {
      $lines = ($lines | append $"($type | str upcase):")
      for note in $type_notes {
        $lines = ($lines | append $"  - [#($note.iteration)] ($note.content)")
      }
    }
  }
  $lines | str join "\n"
}

# Show session notes grouped by type
def show-session-notes [
  store_path: string  # Path to the store directory
  name: string        # Session name
] {
  let notes = (get-note-state $store_path $name)
  if ($notes | is-empty) { return }
  
  let grouped = ($notes | group-by type)
  print $"\n(style section)SESSION NOTES(style reset)"
  
  for type in $NOTE_TYPES {
    let type_notes = ($grouped | get -o $type | default [])
    if ($type_notes | length) > 0 {
      let color = ($NOTE_COLORS | get $type)
      print $"\n(ansi $color)($type | str upcase)(ansi reset)"
      for note in $type_notes {
        let iter = if ($note.iteration != null) { $"[#($note.iteration)]" } else { "" }
        print $"  ($iter) ($note.content)"
      }
    }
  }
}

# Show tasks using computed task state (ID-based, reduce pattern)
def show-tasks [
  store_path: string  # Path to the store directory
  name: string        # Session name
] {
  let state = (get-task-state $store_path $name)
  
  # Check if any tasks exist
  let has_tasks = (
    ($state.completed | length) > 0 or
    ($state.in_progress | length) > 0 or
    ($state.blocked | length) > 0 or
    ($state.remaining | length) > 0
  )
  
  if not $has_tasks {
    return
  }
  
  # Category display config: [header_color, item_symbol, item_color]
  let category_styles = {
    completed: ["green_bold", "✓", "green"]
    in_progress: ["yellow_bold", "●", "yellow"]
    blocked: ["red_bold", "✗", "red"]
    remaining: ["white_dimmed", "○", "white_dimmed"]
  }
  
  # Display tasks by category
  for category in ["in_progress", "blocked", "remaining", "completed"] {
    let tasks = ($state | get $category)
    if ($tasks | length) > 0 {
      let style_info = ($category_styles | get $category)
      let header_style = ($style_info | get 0)
      let symbol = ($style_info | get 1)
      let item_style = ($style_info | get 2)
      
      print $"\n(ansi $header_style)($category | str upcase | str replace '_' ' ')(ansi reset)"
      $tasks | each {|task|
        let iter_label = if ($task.iteration != null) {
          $"(style dim)[#($task.iteration)](style reset) "
        } else {
          ""
        }
        let id_short = ($task.id | str substring 0..8)
        print $"  (ansi $item_style)($symbol)(ansi reset) (style dim)[($id_short)](style reset) ($iter_label)($task.content)"
      }
    }
  }
}

# Show iteration history from xs store
def show-iterations [
  store_path: string  # Path to the store directory
  name: string        # Session name
] {
  let topic = $"ralph.($name).iteration"
  
  # Get all iteration events
  let frames = (xs cat $store_path | from json --objects | where topic == $topic)
  
  if ($frames | is-empty) {
    return
  }
  
  # Extract and display iteration events
  let events = ($frames | each {|frame|
    {
      action: $frame.meta.action
      iteration: $frame.meta.n
      status: ($frame.meta.status? | default "")
      timestamp: $frame.meta.timestamp
    }
  })
  
  print $"\n(style section)HISTORY(style reset)"
  $events | each {|event|
    if $event.action == "start" {
      print $"  (style info)▶(style reset) (style dim)#($event.iteration)(style reset) started (style dim)($event.timestamp)(style reset)"
    } else if $event.action == "complete" {
      let status_style = if $event.status == "success" { style success } else { style error }
      let status_symbol = if $event.status == "success" { "✓" } else { "✗" }
      print $"  ($status_style)($status_symbol)(style reset) (style dim)#($event.iteration)(style reset) ($event.status) (style dim)($event.timestamp)(style reset)"
    }
  }
}

# Format task state as text for prompt injection (includes IDs for reference)
def format-task-state [state: record] {
  mut lines = []
  
  for category in ["in_progress", "blocked", "remaining"] {
    let tasks = ($state | get $category)
    if ($tasks | length) > 0 {
      $lines = ($lines | append $"($category | str upcase | str replace '_' ' '):")
      for task in $tasks {
        $lines = ($lines | append $"  - [($task.id)] ($task.content)")
      }
    }
  }
  
  if ($state.completed | length) > 0 {
    $lines = ($lines | append $"COMPLETED: ($state.completed | length) tasks")
  }
  
  if ($lines | is-empty) {
    "No tasks yet - check spec for initial tasks"
  } else {
    $lines | str join "\n"
  }
}

# Generate custom tool definitions for opencode
def generate-tools [
  --force             # Overwrite existing tools
] {
  let tool_path = ".opencode/tool/ralph.ts"
  
  # Skip if tools exist and not forcing regeneration
  if (not $force) and ($tool_path | path exists) {
    return
  }
  
  # Create tool directory
  mkdir .opencode/tool
  
  # Build TypeScript content
  # STORE uses absolute path resolved at tool load time
  # All tools take session_name as argument - agent gets it from prompt context
  let content = '
import { tool } from "@opencode-ai/plugin"
import { resolve, dirname } from "path"

// Resolve absolute path to store - tools run from .opencode/tool/ so we go up 2 levels
const STORE = resolve(dirname(import.meta.path), "../../.ralph/store")

// Retry helper with exponential backoff
async function withRetry<T>(fn: () => Promise<T>, maxRetries = 3): Promise<T> {
  let lastError: Error | null = null
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn()
    } catch (e) {
      lastError = e as Error
      if (i < maxRetries - 1) {
        await new Promise(r => setTimeout(r, 100 * Math.pow(2, i)))
      }
    }
  }
  throw lastError
}

// Get current iteration from store
async function getCurrentIteration(session: string): Promise<number> {
  const cmd = `xs cat ${STORE} | from json --objects | where topic == "ralph.${session}.iteration" | where {|f| $f.meta.action? == "start"} | last | get meta.n`
  try {
    const result = await Bun.$`nu -c ${cmd}`.text()
    return parseInt(result.trim()) || 1
  } catch {
    return 1
  }
}

export const task_add = tool({
  description: "Add a new task to the ralph session task list",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    content: tool.schema.string().describe("Task description"),
    status: tool.schema.enum(["remaining", "blocked"]).default("remaining").describe("Initial status"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.task`
    try {
      const meta = JSON.stringify({ action: "add", status: args.status })
      const result = await withRetry(async () => {
        return await Bun.$`echo ${args.content} | xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return result.trim()
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})

export const task_status = tool({
  description: "Update a task status by ID. Use IDs from task_list output.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    id: tool.schema.string().describe("Task ID (full or 8+ char prefix)"),
    status: tool.schema.enum(["in_progress", "completed", "blocked"]).describe("New status"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.task`
    try {
      const iteration = await getCurrentIteration(session)
      const meta = JSON.stringify({ action: "status", id: args.id, status: args.status, iteration })
      await withRetry(async () => {
        await Bun.$`xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return `Task ${args.id} marked as ${args.status}`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})

export const task_list = tool({
  description: "Get current task list grouped by status. Shows task IDs needed for task_status.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.task`
    const cmd = `
      let topic = "${topic}"
      let frames = (xs cat ${STORE} | from json --objects | where topic == $topic)
      
      if ($frames | is-empty) {
        "No tasks yet"
      } else {
        let tasks = ($frames | reduce -f {} {|frame, state|
          let action = ($frame.meta.action? | default "add")
          
          if $action == "add" {
            let content = (xs cas ${STORE} $frame.hash)
            let status = ($frame.meta.status? | default "remaining")
            let iteration = ($frame.meta.iteration? | default null)
            $state | upsert $frame.id {
              id: $frame.id
              content: $content
              status: $status
              iteration: $iteration
            }
          } else if $action == "status" {
            let target_id = $frame.meta.id
            let new_status = $frame.meta.status
            let iteration = ($frame.meta.iteration? | default null)
            # Find task by exact match or prefix (8+ chars)
            let matching_id = ($state | columns | where {|id| $id == $target_id or ($id | str starts-with $target_id)} | first | default null)
            if ($matching_id | is-not-empty) {
              $state | upsert $matching_id {|task|
                $task | get $matching_id | upsert status $new_status | upsert iteration $iteration
              }
            } else {
              $state
            }
          } else {
            $state
          }
        })
        
        let task_list = ($tasks | values)
        let grouped = if ($task_list | is-empty) { {} } else { $task_list | group-by status }
        
        {
          completed: ($grouped | get -o completed | default [])
          in_progress: ($grouped | get -o in_progress | default [])
          blocked: ($grouped | get -o blocked | default [])
          remaining: ($grouped | get -o remaining | default [])
        } | to json
      }
    `
    const result = await Bun.$`nu -c ${cmd}`.text()
    return result.trim() || "No tasks yet"
  },
})

export const session_complete = tool({
  description: "Signal that ALL tasks are complete and terminate the ralph session. Only call when every task is done.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    
    try {
      // Write timestamp-based marker - ralph.nu will check for any session_complete frame
      const ts = Date.now()
      const meta = JSON.stringify({ action: "session_complete", ts })
      const topic = `ralph.${session}.control`
      const result = await withRetry(async () => {
        return await Bun.$`echo "complete" | xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return `Session "${session}" marked complete (ts=${ts}, store=${STORE}, result=${result.slice(0,50)}...)`
    } catch (e) {
      return `ERROR: ${(e as Error).message} (store=${STORE})`
    }
  },
})

export const note_add = tool({
  description: "Add a note for future iterations (learnings, tips, blockers, decisions)",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    content: tool.schema.string().describe("Note content"),
    type: tool.schema.enum(["learning", "stuck", "tip", "decision"]).describe("Note category"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    try {
      const iteration = await getCurrentIteration(session)
      const meta = JSON.stringify({ action: "add", type: args.type, iteration })
      const topic = `ralph.${session}.note`
      await withRetry(async () => {
        await Bun.$`echo ${args.content} | xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      const preview = args.content.length > 50 ? args.content.slice(0, 50) + "..." : args.content
      return `Note added: [${args.type}] ${preview}`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})

export const note_list = tool({
  description: "List notes from this session",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    type: tool.schema.enum(["learning", "stuck", "tip", "decision"]).optional().describe("Filter by type"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const typeFilter = args.type ? `| where {|n| $n.type == "${args.type}"}` : ""
    const topic = `ralph.${session}.note`
    const cmd = `
      xs cat ${STORE} | from json --objects | where topic == "${topic}" | each {|f|
        {
          id: $f.id
          type: ($f.meta.type? | default "note")
          iteration: ($f.meta.iteration? | default null)
          content: (xs cas ${STORE} $f.hash)
        }
      } ${typeFilter} | to json
    `
    const result = await Bun.$`nu -c ${cmd}`.text()
    return result.trim() || "No notes yet"
  },
})

export const inbox_list = tool({
  description: "Get unread inbox messages. Check this at start of each iteration.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.inbox`
    const cmd = `
      let topic = "${topic}"
      let frames = (xs cat ${STORE} | from json --objects | where topic == $topic)
      
      if ($frames | is-empty) {
        "No unread messages"
      } else {
        let messages = ($frames | reduce -f {} {|frame, state|
          let action = ($frame.meta.action? | default "add")
          
          if $action == "mark_read" {
            let target_id = $frame.meta.id
            let matching_id = ($state | columns | where {|id| $id == $target_id or ($id | str starts-with $target_id)} | first | default null)
            if ($matching_id | is-not-empty) {
              $state | upsert $matching_id {|m|
                $m | get $matching_id | upsert status "read"
              }
            } else {
              $state
            }
          } else if $action == "add" or ($frame.meta.status? == "unread") {
            let content = (xs cas ${STORE} $frame.hash)
            $state | upsert $frame.id {
              id: $frame.id
              content: $content
              status: "unread"
              timestamp: ($frame.meta.timestamp? | default "")
            }
          } else {
            $state
          }
        })
        
        let unread = ($messages | values | where status == "unread")
        if ($unread | is-empty) {
          "No unread messages"
        } else {
          $unread | to json
        }
      }
    `
    const result = await Bun.$`nu -c ${cmd}`.text()
    return result.trim() || "No unread messages"
  },
})

export const inbox_mark_read = tool({
  description: "Mark an inbox message as read after processing",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    id: tool.schema.string().describe("Message ID from inbox_list"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.inbox`
    try {
      const meta = JSON.stringify({ action: "mark_read", id: args.id })
      await withRetry(async () => {
        await Bun.$`xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return `Message ${args.id} marked as read`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})
'
  
  $content | save -f .opencode/tool/ralph.ts
}

# Build prompt template with task state injected
def build-prompt [
  spec_content: string        # Content of the spec file
  store_path: string          # Path to the store directory
  name: string                # Session name
  iteration: int              # Current iteration number
  task_state: record          # Current task state from get-task-state
  extra_instructions?: string # Optional extra instructions to append
  template?: string           # Optional template content (uses DEFAULT_TEMPLATE if not provided)
] {
  let state_text = (format-task-state $task_state)
  
  # Get inbox state and format for prompt
  let inbox_messages = (get-inbox-state $store_path $name)
  let inbox_section = (format-inbox-for-prompt $inbox_messages)
  
  # Get notes and format for prompt
  let notes = (get-note-state $store_path $name)
  let notes_section = (format-notes-for-prompt $notes $iteration)
  
  # Build optional sections only when they have content
  let inbox_block = if ($inbox_section | is-empty) { "" } else { $"\n($inbox_section)\n" }
  let notes_block = if ($notes_section | is-empty) { "" } else { $"\n($notes_section)\n" }
  let extra_block = if ($extra_instructions | default "" | is-not-empty) {
    $"\n\n## Additional Instructions\n($extra_instructions)"
  } else { "" }
  
  # Use provided template or default
  let template_content = $template | default $DEFAULT_TEMPLATE
  
  # Apply template with variable substitution
  apply-template $template_content {
    session: $name
    iteration: $iteration
    spec: $spec_content
    inbox: $inbox_block
    notes: $notes_block
    tasks: $state_text
    extra: $extra_block
  }
}

# Main entry point - shows usage info
def main [] {
  # Exit early if being sourced (not executed directly)
  # When sourced from tests, $env.CURRENT_FILE will be the test file, not ralph.nu
  let current_file = ($env.CURRENT_FILE? | default "")
  if ($current_file | str contains "tests/") {
    return
  }

  print-banner
  print ""
  print $"(style section)USAGE(style reset)"
  print $"  (style value)./ralph.nu build(style reset) (style dim)[OPTIONS](style reset)"
  print $"  (style value)./ralph.nu gen-template(style reset) (style dim)[--output <path>](style reset)"
  print $"  (style value)./ralph.nu message(style reset) (style dim)--name <session> <message>(style reset)"
  print $"  (style value)./ralph.nu doctor(style reset) (style dim)[--ngrok](style reset)"
  print $"  (style value)./ralph.nu update(style reset)"
  print ""
  print $"(style section)SUBCOMMANDS(style reset)"
  print $"  (style value)build(style reset)         Run the AI agent loop"
  print $"  (style value)gen-template(style reset)  Write default prompt template to file"
  print $"  (style value)message(style reset)       Send a message to a running session"
  print $"  (style value)doctor(style reset)        Check required dependencies"
  print $"  (style value)update(style reset)        Update ralph.nu to latest version from GitHub"
  print ""
  print $"(style section)BUILD OPTIONS(style reset)"
  print $"  (style label)--name, -n(style reset)              Session name \(defaults to spec filename\)"
  print $"  (style label)--spec, -s(style reset)              Spec file path \(default: ./specs/SPEC.md\)"
  print $"  (style label)--template, -t(style reset)          Custom template file \(default: .ralph.template if exists\)"
  print $"  (style label)--extra-instructions, -e(style reset)  Extra instructions appended to prompt"
  print $"  (style label)--model, -m(style reset)             Model to use \(default: anthropic/claude-sonnet-4-5\)"
  print $"  (style label)--iterations, -i(style reset)        Number of iterations \(0 means infinite\)"
  print $"  (style label)--port(style reset)                  opencode serve port \(default: 4096\)"
  print $"  (style label)--ngrok(style reset)                 Enable ngrok tunnel with password"
  print $"  (style label)--ngrok-domain(style reset)          Custom ngrok domain \(optional\)"
  print $"  (style label)--regen-tools(style reset)           Regenerate tool definitions"
  print ""
  print $"(style section)MESSAGE OPTIONS(style reset)"
  print $"  (style label)--name, -n(style reset)           Session name \(required\)"
  print ""
  print $"(style section)EXAMPLES(style reset)"
  print $"  (style dim)# Start agent with a spec(style reset)"
  print $"  ./ralph.nu build --spec ./specs/my-feature.md"
  print ""
  print $"  (style dim)# Export default template for customization(style reset)"
  print $"  ./ralph.nu gen-template"
  print ""
  print $"  (style dim)# Use custom template(style reset)"
  print $"  ./ralph.nu build --spec ./specs/my-feature.md --template ./my-custom.template"
  print ""
  print $"  (style dim)# Add extra instructions to the agent(style reset)"
  print "  ./ralph.nu build --spec ./specs/my-feature.md -e \"Focus on error handling first\""
  print ""
  print $"  (style dim)# Send message to running session(style reset)"
  print $"  ./ralph.nu message --name my-feature \"Please prioritize the login feature\""
  print ""
}

# Doctor subcommand - check required dependencies
def "main doctor" [
  --ngrok  # Also check for ngrok
] {
  print-banner
  print ""
  print $"(style section)Checking dependencies...(style reset)"
  print ""
  
  let all_ok = (run-doctor --include-ngrok=$ngrok)
  
  print ""
  if $all_ok {
    print-ok "All dependencies installed"
  } else {
    print-err "Some dependencies missing"
    exit 1
  }
}

# Update subcommand - fetch latest version from GitHub
def "main update" [] {
  let url = "https://raw.githubusercontent.com/mark3labs/ralph.nu/refs/heads/master/ralph.nu"
  let script_path = ($env.CURRENT_FILE? | default "./ralph.nu")
  
  print-status $"Fetching latest version from GitHub..."
  
  try {
    http get $url | save -f $script_path
    chmod +x $script_path
    print-ok $"Updated ($script_path)"
  } catch { |err|
    print-err $"Failed to update: ($err.msg)"
    error make {msg: $err.msg}
  }
}

# gen-template subcommand - write default prompt template to file
def "main gen-template" [
  --output (-o): string = ".ralph.template"  # Output file path
] {
  $DEFAULT_TEMPLATE | save -f $output
  print-ok $"Template written to ($output)"
}

# Message subcommand - send message to running session
def "main message" [
  message: string           # Message to send
  --name (-n): string       # Session name (required)
] {
  # Validate --name flag is provided
  if ($name | is-empty) {
    error make {msg: "--name flag is required"}
  }
  
  # Store path is always relative to project root
  let store = ".ralph/store"
  
  # Verify store is running
  let check = (xs version $store | complete)
  if $check.exit_code != 0 {
    error make {msg: "Store not running. Is session active?"}
  }
  
  # Append message to inbox topic with unread status
  let topic = $"ralph.($name).inbox"
  let timestamp = (date now | format date "%Y-%m-%dT%H:%M:%S%z")
  let meta = {status: "unread", timestamp: $timestamp} | to json -r
  
  let result = (echo $message | xs append $store $topic --meta $meta)
  print $"Message sent: ($result)"
}

# Main entry point - build subcommand runs the agent loop
def "main build" [
  input?: string                                            # Optional piped input for prompt
  --name (-n): string = ""                                  # Session name (defaults to spec filename without extension)
  --prompt (-p): string                                     # Custom prompt (replaces entire template)
  --extra-instructions (-e): string = ""                    # Extra instructions appended to prompt
  --spec (-s): string = "./specs/SPEC.md"                   # Spec file path
  --template (-t): string = ""                              # Custom template file path
  --model (-m): string = "anthropic/claude-sonnet-4-5"      # Model to use
  --iterations (-i): int = 0                                # Number of iterations (0 = infinite)
  --port: int = 4096                                        # opencode serve port
  --ngrok: string = ""                                      # Enable ngrok tunnel with this password
  --ngrok-domain: string = ""                               # Custom ngrok domain (optional)
  --regen-tools                                             # Regenerate tool definitions (overwrites existing)
] {
  # Check required tools before doing anything
  let include_ngrok = ($ngrok | is-not-empty)
  let missing = (check-requirements --include-ngrok=$include_ngrok)
  if ($missing | is-not-empty) {
    print-err $"Missing required tools: ($missing | str join ', ')"
    print $"  (style dim)Run './ralph.nu doctor' for details(style reset)"
    exit 1
  }
  
  # Store path is always relative to project root
  let store = ".ralph/store"
  
  # Derive name from spec filename if not provided
  let name = if ($name | is-empty) {
    $spec | path basename | path parse | get stem
  } else {
    $name
  }

  print-banner
  print ""
  print-kv "Session" $name
  print-kv "Store" $store
  print-kv "Port" ($port | into string)
  print ""
  
  # Run main logic with cleanup handling
  try {
    # Kill any existing processes from previous runs
    kill-existing $store $port
    
    # Start xs store
    let store_job_id = (start-store $store)
    
    # Start opencode serve
    let web_result = (start-web $port)
    
    # Start ngrok tunnel if requested
    mut ngrok_job_id = null
    if ($ngrok | is-not-empty) {
      let ngrok_result = (start-ngrok $port $ngrok $ngrok_domain)
      $ngrok_job_id = $ngrok_result.job_id
    }
    
    # Display current state (if any previous sessions exist)
    try {
      show-iterations $store $name
      show-session-notes $store $name
      show-tasks $store $name
    } catch {
      # Store may not have any data yet - this is fine for first run
    }
    
    # Read spec file
    print-kv "Spec" $spec
    let spec_content = (open $spec)
    
    # Resolve template once at start (only if no --prompt override)
    let template_content = if ($template | is-not-empty) {
      resolve-template $template
    } else {
      resolve-template
    }
    
    # Determine base prompt (priority: --prompt > piped input > default template)
    let base_prompt = if ($prompt | is-not-empty) {
      $prompt
    } else if ($input | is-not-empty) {
      $input
    } else {
      null
    }
    
    # Get last iteration from store to enable continuation
    let last_iter = try {
      let frames = (xs cat $store | from json --objects 
        | where topic == $"ralph.($name).iteration")
      if ($frames | is-empty) { 0 } else { $frames | get meta.n | math max }
    } catch { 0 }
    
    if $last_iter > 0 {
      print-status $"Continuing from iteration #($last_iter)"
    }
    
    # Capture session start ID for shutdown signal filtering
    let session_start_id = try {
      xs cat $store | from json --objects | last | get id
    } catch { "0" }
    
    # Main iteration loop
    mut n = $last_iter + 1
    loop {
      print $"\n(style header)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━(style reset)"
      print $"(style header)  ($name)(style reset) (style dim)·(style reset) (style value)Iteration #($n)(style reset)"
      print $"(style header)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━(style reset)\n"
      
      # Generate custom tools (skip if already exist, unless --regen-tools on first iteration)
      generate-tools --force=($regen_tools and $n == ($last_iter + 1))
      
      # Get current task state for this iteration
      let task_state = (get-task-state $store $name)
      
      # Build prompt for this iteration (with optional extra instructions and template)
      let extra = if ($extra_instructions | is-not-empty) { $extra_instructions } else { null }
      let iteration_prompt = (build-prompt $spec_content $store $name $n $task_state $extra $template_content)
      
      # Use base_prompt if set, otherwise use built template
      let final_prompt = if ($base_prompt | is-not-empty) { $base_prompt } else { $iteration_prompt }
      
      # Log iteration start
      log-iteration-start $store $name $n
      
      # Run opencode attached to web server
      opencode run --attach $web_result.url --title $"($name) - Iteration #($n)" -m $model $final_prompt
      
      # Determine status based on exit code
      let status = if $env.LAST_EXIT_CODE == 0 { "success" } else { "failure" }
      
      # Log iteration complete
      log-iteration-complete $store $name $n $status
      
      if $status == "success" {
        print-ok $"Iteration #($n) complete"
      } else {
        print-err $"Iteration #($n) failed"
      }
      
      # Check for graceful shutdown signal (session_complete called by agent)
      # Check for any session_complete frame created during this iteration (within last 5 minutes)
      let cutoff = ((date now) - 5min | format date "%s" | into int) * 1000
      let shutdown = (xs cat $store 
        | from json --objects 
        | where topic == $"ralph.($name).control"
        | where {|f| $f.meta.action? == "session_complete" }
        | where {|f| ($f.meta.ts? | default 0) > $cutoff }
        | is-not-empty)
      
      if $shutdown {
        print-ok "Session complete - all tasks done"
        break
      }
      
      # Increment counter
      $n += 1
      
      # Check iteration limit (0 = infinite)
      if $iterations > 0 and $n > $iterations {
        print $"\n(style dim)Completed ($iterations) iterations. Exiting.(style reset)"
        break
      }
    }
    
  } catch { |err|
    print-err $"($err.msg)"
    cleanup-all
    error make {msg: $err.msg}
  }
  
  # Normal cleanup on success or Ctrl+C
  cleanup-all
}
