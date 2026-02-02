# Dead Code Cleanup

**Created**: 2026-02-02
**Status**: Draft

## Overview

Remove ~720 lines of dead code identified through codebase analysis. Includes unused files, legacy rendering methods, unused enum values, and orphaned helper functions.

## User Story

As a maintainer, I want to remove dead code so the codebase is easier to understand and maintain.

## Requirements

- Remove all identified dead code without breaking functionality
- Ensure tests pass after each removal
- No functional changes - pure cleanup

## Findings Summary

| Category | Items | Lines |
|----------|-------|-------|
| Unused files | 2 | ~462 |
| Legacy Render() methods | 4 | ~90 |
| Unused Dashboard methods | 3 | ~105 |
| Unused enum values | 2 | ~5 |
| Unused hint functions | 7 | ~40 |
| Legacy wrapper methods | 5 | ~20 |
| Unused animation helper | 1 | ~14 |

## Tasks

### 1. Delete unused files

- [ ] Delete `internal/tui/footer.go` (239 lines, never instantiated)
- [ ] Delete `internal/tui/notes.go` (223 lines, Sidebar handles notes)
- [ ] Run tests to verify no breakage

### 2. Remove legacy Render() methods

- [ ] Remove `Sidebar.Render()` at `sidebar.go:699-784` (86 lines)
- [ ] Remove `AgentOutput.Render()` stub at `agent.go:149`
- [ ] Remove `LogViewer.Render()` stub at `logs.go:113`
- [ ] Remove outdated comment at `sidebar.go:698` ("Phase 12")
- [ ] Run tests

### 3. Remove unused Dashboard methods

- [ ] Remove `Dashboard.renderSessionInfo()` at `dashboard.go:158-173`
- [ ] Remove `Dashboard.renderProgressIndicator()` at `dashboard.go:264-324`
- [ ] Remove `Dashboard.progressStats` type and `getTaskStats()` at `dashboard.go:296-324`
- [ ] Run tests

### 4. Remove unused enum values

- [ ] Remove `ViewNotes` from `interfaces.go:54-63`
- [ ] Remove `ViewInbox` from `interfaces.go:54-63`
- [ ] Run tests

### 5. Remove unused hint functions

- [ ] Remove `HintScroll()` at `hints.go:69-76`
- [ ] Remove `HintModal()` at `hints.go:81-83`
- [ ] Remove `HintLogs()` at `hints.go:86-88`
- [ ] Remove `HintInputFocused()` and `HintInputBlurred()` at `hints.go:92-99`
- [ ] Remove `HintTaskNav()` at `hints.go:103-105`
- [ ] Remove `HintAgentViewport()` at `hints.go:115-117`
- [ ] Remove unused key constants `KeyCtrlEnter`, `KeyBackspace` at `hints.go:24,28`
- [ ] Run tests

### 6. Remove legacy wrapper methods

- [ ] Update `app.go:166` to use `dashboard.SetState()` instead of `UpdateState()`
- [ ] Remove `Dashboard.UpdateSize()` at `dashboard.go:187-193`
- [ ] Remove `Dashboard.UpdateState()` at `dashboard.go:210-216`
- [ ] Remove `Sidebar.UpdateSize()` at `sidebar.go:788`
- [ ] Remove `Sidebar.UpdateState()` at `sidebar.go:789`
- [ ] Remove `Sidebar.SetFocused()` at `sidebar.go:787`
- [ ] Run tests

### 7. Remove unused animation helper

- [ ] Remove `GetPulseStyle()` at `anim.go:140-153`
- [ ] Run tests

### 8. Final verification

- [ ] Run full test suite
- [ ] Run `go build` to verify compilation
- [ ] Manual smoke test of TUI

## Out of Scope

- Refactoring duplicate code (modal sizing, gradient rendering)
- Consolidating message types
- Addressing TODO comments
- Interface consistency improvements

## Open Questions

None - all items confirmed unused via static analysis.
