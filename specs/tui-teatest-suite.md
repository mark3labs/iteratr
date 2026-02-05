# TUI Teatest Suite

## Overview

Comprehensive test suite for all TUI components using teatest v2. Refactor existing tests to teatest patterns, add golden file visual regression, behavioral WaitFor testing, and full mock infrastructure.

## User Story

As a developer, I need confidence that TUI changes don't break existing behavior. Golden files catch visual regressions, WaitFor patterns verify async flows, and component tests ensure keyboard/mouse interactions work correctly.

## Requirements

### Testing Approach
- Golden file visual regression via `RequireEqualOutput`
- Behavioral WaitFor patterns for async streaming, animations, user flows
- Full sequence testing for prefix keys (ctrl+x → action)
- Verify state propagation timing with WaitFor cascade checks
- Execute returned commands and verify resulting messages

### Coverage Scope
- All TUI components (no exclusions)
- All 7 message types in all states (collapsed + expanded)
- All modal priority combinations (dialog > prefix > modal > component)
- Keyboard navigation flows (focus cycling, prefix keys, modal dismissal)
- Key mouse interactions (click-to-expand, click-outside-to-close)
- All error paths (SubagentError, connection failures, etc)
- Theme switching verification
- Textarea edge cases (long content, unicode, newlines, paste)
- Pause state edge cases (PAUSING→PAUSED transitions)
- Auto-scroll position verification

### Animation Handling
- Disable animations: `lipgloss.SetColorProfile(termenv.Ascii)`
- GradientSpinner renders static character when disabled
- Consistent output for golden file comparison

### Test Infrastructure
- Shared fixtures package: `internal/tui/testfixtures/`
- Full mock suite: MockStore, MockEvents, MockGit, MockSessionLoader
- Mock event channel for NATS subscription
- Mock callback injection for Dialog testing
- Fixed test values for dynamic content (time, git hash, iteration)
- Conservative 5s timeout for WaitFor (CI compatibility)
- Canonical terminal size: 120x40

### Configuration
- Add `*.golden -text` to `.gitattributes`
- Golden file updates via `go test -update`
- Use existing `go test ./...` for CI (no new make target)
- No build tag separation - all tests run together
- Golden files include ANSI escape sequences (catches styling regressions)

### Flaky Test Handling
- Custom retry helper: `RetryTest(t, 3, func() error {...})`
- Located in `testfixtures/helpers.go`
- Max 3 attempts before failing

### Test Organization
- Same directory pattern: `component_teatest_test.go` alongside `component.go`
- Refactor existing tests to teatest patterns
- Component-level focus (integration_test.go stays as-is)
- Minimal documentation (self-explanatory code)

### Bug Handling
- Fix any bugs discovered during implementation

## Technical Implementation

### Directory Structure
```
internal/tui/
├── testfixtures/
│   ├── fixtures.go       # Common session.State builders
│   ├── mocks.go          # MockStore, MockEvents, MockGit, MockLoader
│   └── helpers.go        # Shared test utilities
├── app_teatest_test.go
├── dashboard_teatest_test.go
├── agent_teatest_test.go
├── sidebar_teatest_test.go
├── status_teatest_test.go
├── modal_teatest_test.go
├── note_modal_teatest_test.go
├── note_input_modal_teatest_test.go
├── task_input_modal_teatest_test.go
├── subagent_modal_teatest_test.go
├── dialog_teatest_test.go
├── logs_teatest_test.go
├── scrolllist_teatest_test.go
├── messages_teatest_test.go
├── anim_teatest_test.go
├── testdata/
│   └── *.golden          # Golden files per component/test
└── ...
```

### Test Pattern Template
```go
import "charm.land/x/exp/teatest/v2"

func init() {
    lipgloss.SetColorProfile(termenv.Ascii)
}

func TestComponent_Scenario(t *testing.T) {
    tm := teatest.NewTestModel(t, 
        NewComponent(testfixtures.MockDeps()),
        teatest.WithInitialTermSize(120, 40),
    )
    
    // Wait for initial render
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("ready"))
    }, teatest.WithDuration(5*time.Second))
    
    // Send input
    tm.Send(tea.KeyPressMsg{Type: tea.KeyTab})
    
    // Verify golden
    out, _ := io.ReadAll(tm.FinalOutput(t))
    teatest.RequireEqualOutput(t, out)
}
```

### Prefix Key Test Pattern
```go
func TestApp_PrefixKeySequence(t *testing.T) {
    tm := teatest.NewTestModel(t, NewApp(...), teatest.WithInitialTermSize(120, 40))
    
    // Enter prefix mode
    tm.Send(tea.KeyPressMsg{Type: tea.KeyCtrlX})
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("C-x"))  // prefix indicator
    }, teatest.WithDuration(5*time.Second))
    
    // Execute action
    tm.Send(tea.KeyPressMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Event Log"))
    }, teatest.WithDuration(5*time.Second))
    
    tm.Quit()
    out, _ := io.ReadAll(tm.FinalOutput(t))
    teatest.RequireEqualOutput(t, out)
}
```

### State Propagation Test Pattern
```go
func TestApp_StatePropagation(t *testing.T) {
    tm := teatest.NewTestModel(t, NewApp(...), teatest.WithInitialTermSize(120, 40))
    
    // Send state update
    tm.Send(StateUpdateMsg{State: testfixtures.StateWithTasks()})
    
    // Verify cascade timing - StatusBar updates first
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("1/3 tasks"))
    }, teatest.WithDuration(5*time.Second))
    
    // Then Sidebar
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Task 1"))
    }, teatest.WithDuration(5*time.Second))
    
    tm.Quit()
    out, _ := io.ReadAll(tm.FinalOutput(t))
    teatest.RequireEqualOutput(t, out)
}
```

## UI Mockup

N/A - testing infrastructure, no new UI

## Tasks

### 1. Test Infrastructure Setup
- [ ] [P0] Create `internal/tui/testfixtures/` package structure
- [ ] [P0] Implement `fixtures.go` with State builders (empty, with tasks, with notes, full)
- [ ] [P0] Implement `mocks.go` with MockStore, MockEvents, MockGit, MockSessionLoader
- [ ] [P0] Implement `helpers.go` with shared test utilities, init() for Ascii profile, RetryTest helper
- [ ] [P0] Add `*.golden -text` to `.gitattributes`

### 2. Core Component Tests
- [ ] [P1] Refactor `app_test.go` to teatest patterns (`app_teatest_test.go`)
- [ ] [P1] Add App modal priority matrix tests (all combinations)
- [ ] [P1] Add App prefix key sequence tests with WaitFor
- [ ] [P1] Refactor `dashboard_test.go` to teatest patterns
- [ ] [P1] Add Dashboard focus cycling tests with visual verification

### 3. Agent Output Tests
- [ ] [P1] Refactor `agent_test.go` to teatest patterns
- [ ] [P1] Add golden files for all 7 message types (collapsed state)
- [ ] [P1] Add golden files for all 7 message types (expanded state)
- [ ] [P1] Add scroll position verification tests
- [ ] [P2] Add click-to-expand mouse interaction tests

### 4. Sidebar Tests
- [ ] [P1] Create `sidebar_teatest_test.go` with golden files
- [ ] [P1] Add task list navigation tests
- [ ] [P1] Add note list navigation tests
- [ ] [P2] Add pulse animation state tests (static character)

### 5. Modal Tests
- [ ] [P1] Refactor `modal_test.go` to teatest with goldens
- [ ] [P1] Create `note_modal_teatest_test.go` with goldens
- [ ] [P1] Create `note_input_modal_teatest_test.go` with focus cycling tests
- [ ] [P1] Create `task_input_modal_teatest_test.go` with focus cycling tests
- [ ] [P1] Create `subagent_modal_teatest_test.go` with loading/error/content states
- [ ] [P2] Create `dialog_teatest_test.go` with mock callback verification

### 6. Status Bar Tests
- [ ] [P1] Refactor `status_test.go` to teatest with fixed dynamic values
- [ ] [P2] Add pause state edge case tests (PAUSING→PAUSED transitions)

### 7. Utility Component Tests
- [ ] [P2] Create `logs_teatest_test.go` with auto-scroll position verification
- [ ] [P2] Create `scrolllist_teatest_test.go` with viewport boundary tests
- [ ] [P2] Create `messages_teatest_test.go` refactored from existing
- [ ] [P3] Create `anim_teatest_test.go` for static animation states

### 8. Theme Tests
- [ ] [P2] Create theme switching verification tests
- [ ] [P2] Add golden files for catppuccin_mocha theme
- [ ] [P3] Add golden files for additional themes if present

### 9. Wizard/Setup Tests
- [ ] [P2] Migrate `wizard/model_selector_test.go` to teatest
- [ ] [P2] Migrate `wizard/wizard_config_test.go` to teatest
- [ ] [P2] Create teatest coverage for remaining wizard components
- [ ] [P2] Migrate `setup/model_step_test.go` to teatest
- [ ] [P2] Create teatest coverage for setup flow

### 10. Edge Case Tests
- [ ] [P2] Add textarea edge cases: very long content (>1000 chars)
- [ ] [P2] Add textarea edge cases: unicode, emoji, RTL text
- [ ] [P2] Add textarea edge cases: multi-line paste, newlines
- [ ] [P3] Add error state goldens for all error paths

### 11. Command Verification
- [ ] [P2] Add command execution tests for App (execute cmd, verify Msg)
- [ ] [P2] Add command execution tests for Dashboard
- [ ] [P2] Add command execution tests for modal components

### 12. Cleanup Legacy Tests
- [ ] [P3] Remove old test files replaced by teatest versions
- [ ] [P3] Consolidate duplicate test logic into testfixtures
- [ ] [P3] Verify test coverage maintained after migration

## Out of Scope

- Integration-level teatest (integration_test.go stays as-is)
- Specific coverage percentage targets
- Extensive documentation (minimal docs only)
- New CI job or make target
- hover state mouse testing

## Open Questions

None - all questions resolved during spec interview.
