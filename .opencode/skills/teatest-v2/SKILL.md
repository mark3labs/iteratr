---
name: teatest-v2
description: Test Bubble Tea v2 TUI applications using teatest v2 library
license: MIT
compatibility: opencode
metadata:
  language: go
  domain: testing
---

# teatest v2

Testing library for Bubble Tea v2 applications.

## Import

```go
import "charm.land/x/exp/teatest/v2"
```

Requires `github.com/charmbracelet/bubbletea/v2`.

## Core API

### Create Test Model

```go
tm := teatest.NewTestModel(t, model,
    teatest.WithInitialTermSize(80, 24),
    teatest.WithProgramOptions(tea.WithContext(ctx)),
)
```

### Send Input

```go
// Send any tea.Msg
tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

// Type text (sends individual key messages)
tm.Type("hello")
```

### Get Output

```go
// Current output (non-blocking)
reader := tm.Output()

// Final output (blocks until program exits)
reader := tm.FinalOutput(t, teatest.WithFinalTimeout(5*time.Second))
```

### Get Final Model

```go
// Blocks until program exits, then returns final model state
fm := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second))
m := fm.(MyModel) // type assert to concrete type
```

### Wait for Conditions

```go
// Wait for output to contain specific text
teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
    return bytes.Contains(bts, []byte("expected text"))
}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(100*time.Millisecond))
```

### Control Program

```go
tm.Quit()                                              // quit program
tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second)) // wait for exit
tm.GetProgram()                                        // access underlying *tea.Program
```

## Golden File Testing

```go
func TestOutput(t *testing.T) {
    tm := teatest.NewTestModel(t, initialModel(), teatest.WithInitialTermSize(80, 24))
    out, _ := io.ReadAll(tm.FinalOutput(t))
    teatest.RequireEqualOutput(t, out) // compares to testdata/TestOutput.golden
}
```

Update golden files: `go test -update`

## Test Patterns

### Full Output Test

```go
func TestFullOutput(t *testing.T) {
    tm := teatest.NewTestModel(t, initialModel(), teatest.WithInitialTermSize(80, 24))
    out, err := io.ReadAll(tm.FinalOutput(t))
    require.NoError(t, err)
    teatest.RequireEqualOutput(t, out)
}
```

### Final Model State Test

```go
func TestFinalModel(t *testing.T) {
    tm := teatest.NewTestModel(t, initialModel(), teatest.WithInitialTermSize(80, 24))
    fm := tm.FinalModel(t)
    m, ok := fm.(model)
    require.True(t, ok)
    assert.Equal(t, expected, m.field)
}
```

### Interactive Test

```go
func TestInteraction(t *testing.T) {
    tm := teatest.NewTestModel(t, initialModel(), teatest.WithInitialTermSize(80, 24))

    // Wait for initial render
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("ready"))
    })

    // Send input
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

    // Wait for program to finish
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
```

## CI/Cross-Platform Tips

### Consistent Color Profile

```go
func init() {
    lipgloss.SetColorProfile(termenv.Ascii) // disable colors for reproducible output
}
```

### Golden File Git Attributes

Add to `.gitattributes`:
```text
*.golden -text
```

## Key Types

| Type | Purpose |
|------|---------|
| `TestModel` | Wrapper for testing a tea.Model |
| `TestOption` | Options for NewTestModel |
| `FinalOpt` | Options for Final*/WaitFinished methods |
| `WaitForOption` | Options for WaitFor |

## Options

| Function | Purpose |
|----------|---------|
| `WithInitialTermSize(x, y)` | Set terminal dimensions |
| `WithProgramOptions(...)` | Pass tea.ProgramOptions |
| `WithFinalTimeout(d)` | Timeout for final operations |
| `WithDuration(d)` | Total wait duration |
| `WithCheckInterval(d)` | Check frequency for WaitFor |
