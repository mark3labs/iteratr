# TUI Test Fixtures

This package provides reusable test fixtures, mocks, and utilities for testing TUI components.

## State Builders

Pre-built session states for consistent testing:

- `EmptyState()` - Minimal empty session state
- `StateWithTasks()` - Session with tasks in various states (completed, in_progress, remaining)
- `StateWithNotes()` - Session with notes of all types (learning, stuck, tip, decision)
- `FullState()` - Complete session with tasks, notes, and iterations
- `StateWithBlockedTasks()` - Session with blocked task dependencies
- `StateWithCompletedSession()` - Session marked as complete

All states use fixed values from `fixtures.go` for deterministic golden file testing.

## Mock Implementations

### MockStore

Mock implementation of `session.Store` for testing without NATS:

```go
store := testfixtures.NewMockStore()
store.State = testfixtures.StateWithTasks()

// Use in app
app := NewApp(ctx, store, ...)

// Verify calls
require.Equal(t, 1, store.LoadStateCalls)
require.Len(t, store.GetPublishedEvents(), 3)
```

**Features:**
- Controllable `LoadState` return values and errors
- Records all published events for verification
- Thread-safe for concurrent access
- Configurable `ListSessions` and `ResetSession` behavior

### MockEventChannel

Controllable event channel for testing NATS event consumption:

```go
events := testfixtures.NewMockEventChannel(100)

// Send test events
events.Send(session.Event{Type: "task", Action: "add"})

// Verify received
require.Len(t, events.GetReceivedEvents(), 1)

// Clean up
events.Close()
```

**Features:**
- Buffered channel with configurable size
- Records all received events
- Thread-safe send/receive operations
- Handles closed channel gracefully

### MockGit

Mock git operations without requiring a real repository:

```go
git := testfixtures.NewMockGit()
git.SetDirty(true)
git.SetBranch("feature-branch")
git.SetAheadBehind(5, 2)

info, err := git.GetInfo("/tmp/repo")
// Returns configured git.Info
```

**Features:**
- Default values: main branch, clean state, no ahead/behind
- Helper methods for common state changes
- Configurable errors for testing error paths
- Thread-safe

### MockOrchestrator

Mock orchestrator for pause/resume control:

```go
orch := testfixtures.NewMockOrchestrator()

// Simulate pause
orch.SetPaused(true)

// Verify calls
orch.RequestPause()
require.True(t, orch.WasPauseRequested())
```

**Features:**
- Tracks `RequestPause`, `CancelPause`, `Resume` calls
- Configurable paused state
- Thread-safe verification methods
- Reset for reuse in subtests

## Test Helpers

### Fixed Values

All fixtures use consistent fixed values for reproducible tests:

```go
const (
    FixedSessionName = "test-session"
    FixedGitHash     = "abc1234"
    FixedIteration   = 1
)

var FixedTime = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
```

### Terminal Dimensions

Standard terminal size for consistent layout testing:

```go
const (
    TestTermWidth  = 120
    TestTermHeight = 40
)
```

### Golden File Comparison

Helper for comparing test output with golden files:

```go
testfixtures.CompareGolden(t, "testdata/golden.txt", actual)
```

Run with `-update` flag to regenerate golden files:
```bash
go test -update
```

### Retry Helper

Retry flaky tests with timeout:

```go
testfixtures.RetryTest(t, 3, func() error {
    // Test logic that may be flaky
    if condition {
        return nil // Success
    }
    return fmt.Errorf("not ready")
})
```

## Usage Patterns

### Basic Component Test

```go
func TestComponent_Render(t *testing.T) {
    t.Parallel()

    store := testfixtures.NewMockStore()
    store.State = testfixtures.StateWithTasks()

    component := NewComponent(store)
    component.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

    output := component.View()
    testfixtures.CompareGolden(t, "testdata/component.golden", output)
}
```

### App Integration Test

```go
func TestApp_WithMocks(t *testing.T) {
    t.Parallel()

    ctx := context.Background()
    store := testfixtures.NewMockStore()
    store.State = testfixtures.FullState()

    git := testfixtures.NewMockGit()
    orch := testfixtures.NewMockOrchestrator()

    app := NewApp(ctx, store, "test-session", "/tmp", t.TempDir(), nil, nil, orch)

    // Send messages, verify behavior
    msg := tea.KeyPressMsg{Code: tea.KeyEnter}
    _, _ = app.Update(msg)

    // Verify mock interactions
    require.Equal(t, 1, store.LoadStateCalls)
}
```

### Event Flow Test

```go
func TestApp_EventProcessing(t *testing.T) {
    store := testfixtures.NewMockStore()
    events := testfixtures.NewMockEventChannel(100)
    defer events.Close()

    // Send test event
    events.Send(session.Event{
        Type:   "task",
        Action: "add",
        Data:   "New task",
    })

    // Verify app processes event
    // ...

    // Verify event was received
    received := events.GetReceivedEvents()
    require.Len(t, received, 1)
}
```

## Thread Safety

All mocks are thread-safe and can be used in concurrent tests. They use `sync.RWMutex` internally and provide `Reset()` methods for cleanup between subtests.

## Best Practices

1. **Use `t.Parallel()`** for all tests that don't modify global state
2. **Use fixed values** from `testfixtures` for deterministic golden files
3. **Reset mocks** when reusing in subtests: `store.Reset()`
4. **Verify calls** using counter fields: `store.LoadStateCalls`
5. **Set errors** to test error paths: `store.LoadError = fmt.Errorf("...")`
6. **Use temp dirs** for file operations: `t.TempDir()`
7. **Use CompareGolden** for visual regression testing
