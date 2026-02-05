package testfixtures

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/git"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/stretchr/testify/require"
)

// --- MockStore Tests ---

func TestMockStore_LoadState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Set custom state
	customState := StateWithTasks()
	store.State = customState

	// Load state
	state, err := store.LoadState(ctx, "test-session")
	require.NoError(t, err)
	require.Equal(t, customState, state)
	require.Equal(t, 1, store.LoadStateCalls)
	require.Equal(t, "test-session", store.LastSessionName)
}

func TestMockStore_LoadStateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Set error
	expectedErr := fmt.Errorf("load failed")
	store.LoadError = expectedErr

	// Load state
	state, err := store.LoadState(ctx, "test-session")
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Nil(t, state)
	require.Equal(t, 1, store.LoadStateCalls)
}

func TestMockStore_PublishEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	event := session.Event{
		ID:        "TAS-1",
		Timestamp: FixedTime,
		Session:   "test-session",
		Type:      "task",
		Action:    "add",
		Data:      "Test task",
	}

	// Publish event
	ack, err := store.PublishEvent(ctx, event)
	require.NoError(t, err)
	require.NotNil(t, ack)
	require.Equal(t, uint64(1), ack.Sequence)
	require.Equal(t, 1, store.PublishCalls)

	// Verify event was recorded
	events := store.GetPublishedEvents()
	require.Len(t, events, 1)
	require.Equal(t, event, events[0])
}

func TestMockStore_PublishEventError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Set error
	expectedErr := fmt.Errorf("publish failed")
	store.PublishError = expectedErr

	event := session.Event{
		ID:      "TAS-1",
		Session: "test-session",
		Type:    "task",
		Action:  "add",
		Data:    "Test task",
	}

	// Publish event
	ack, err := store.PublishEvent(ctx, event)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Nil(t, ack)
	require.Equal(t, 1, store.PublishCalls)

	// Event should still be recorded even on error
	events := store.GetPublishedEvents()
	require.Len(t, events, 1)
}

func TestMockStore_PublishMultipleEvents(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Publish multiple events
	for i := 0; i < 5; i++ {
		event := session.Event{
			ID:      fmt.Sprintf("TAS-%d", i+1),
			Session: "test-session",
			Type:    "task",
			Action:  "add",
			Data:    fmt.Sprintf("Task %d", i+1),
		}
		_, err := store.PublishEvent(ctx, event)
		require.NoError(t, err)
	}

	require.Equal(t, 5, store.PublishCalls)
	events := store.GetPublishedEvents()
	require.Len(t, events, 5)

	// Verify sequence
	for i, event := range events {
		require.Equal(t, fmt.Sprintf("TAS-%d", i+1), event.ID)
	}
}

func TestMockStore_ListSessions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Set sessions
	sessions := []session.SessionInfo{
		{
			Name:           "session-1",
			Complete:       false,
			TasksTotal:     10,
			TasksCompleted: 5,
			LastActivity:   FixedTime,
		},
		{
			Name:           "session-2",
			Complete:       true,
			TasksTotal:     5,
			TasksCompleted: 5,
			LastActivity:   FixedTime.Add(10 * time.Minute),
		},
	}
	store.Sessions = sessions

	// List sessions
	result, err := store.ListSessions(ctx)
	require.NoError(t, err)
	require.Equal(t, sessions, result)
	require.Equal(t, 1, store.ListCalls)
}

func TestMockStore_ListSessionsError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Set error
	expectedErr := fmt.Errorf("list failed")
	store.ListError = expectedErr

	// List sessions
	result, err := store.ListSessions(ctx)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Nil(t, result)
	require.Equal(t, 1, store.ListCalls)
}

func TestMockStore_ResetSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Reset session
	err := store.ResetSession(ctx, "test-session")
	require.NoError(t, err)
	require.Equal(t, 1, store.ResetCalls)
	require.Equal(t, "test-session", store.LastSessionName)
}

func TestMockStore_ResetSessionError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Set error
	expectedErr := fmt.Errorf("reset failed")
	store.ResetError = expectedErr

	// Reset session
	err := store.ResetSession(ctx, "test-session")
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 1, store.ResetCalls)
}

func TestMockStore_Reset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	// Perform operations
	_, _ = store.LoadState(ctx, "session-1")
	_, _ = store.PublishEvent(ctx, session.Event{ID: "TAS-1", Session: "session-1", Type: "task", Action: "add"})
	_, _ = store.ListSessions(ctx)
	_ = store.ResetSession(ctx, "session-2")

	// Verify counters
	require.Equal(t, 1, store.LoadStateCalls)
	require.Equal(t, 1, store.PublishCalls)
	require.Equal(t, 1, store.ListCalls)
	require.Equal(t, 1, store.ResetCalls)
	require.Len(t, store.PublishedEvents, 1)

	// Reset
	store.Reset()

	// Verify reset
	require.Equal(t, 0, store.LoadStateCalls)
	require.Equal(t, 0, store.PublishCalls)
	require.Equal(t, 0, store.ListCalls)
	require.Equal(t, 0, store.ResetCalls)
	require.Empty(t, store.LastSessionName)
	require.Empty(t, store.PublishedEvents)
}

func TestMockStore_ThreadSafety(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMockStore()

	var wg sync.WaitGroup
	concurrency := 100

	// Concurrent PublishEvent calls
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			event := session.Event{
				ID:      fmt.Sprintf("TAS-%d", idx),
				Session: "test-session",
				Type:    "task",
				Action:  "add",
				Data:    fmt.Sprintf("Task %d", idx),
			}
			_, _ = store.PublishEvent(ctx, event)
		}(i)
	}

	// Concurrent GetPublishedEvents calls
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_ = store.GetPublishedEvents()
		}()
	}

	wg.Wait()

	// Verify all events were recorded
	require.Equal(t, concurrency, store.PublishCalls)
	events := store.GetPublishedEvents()
	require.Len(t, events, concurrency)
}

// --- MockEventChannel Tests ---

func TestMockEventChannel_Send(t *testing.T) {
	t.Parallel()

	ch := NewMockEventChannel(10)
	defer ch.Close()

	event := session.Event{
		ID:      "TAS-1",
		Session: "test-session",
		Type:    "task",
		Action:  "add",
		Data:    "Test task",
	}

	// Send event
	err := ch.Send(event)
	require.NoError(t, err)

	// Receive event
	received := <-ch.Ch
	require.Equal(t, event, received)

	// Verify recorded
	events := ch.GetReceivedEvents()
	require.Len(t, events, 1)
	require.Equal(t, event, events[0])
}

func TestMockEventChannel_SendMultiple(t *testing.T) {
	t.Parallel()

	ch := NewMockEventChannel(10)
	defer ch.Close()

	// Send multiple events
	for i := 0; i < 5; i++ {
		event := session.Event{
			ID:      fmt.Sprintf("TAS-%d", i+1),
			Session: "test-session",
			Type:    "task",
			Action:  "add",
			Data:    fmt.Sprintf("Task %d", i+1),
		}
		err := ch.Send(event)
		require.NoError(t, err)
	}

	// Receive all events
	for i := 0; i < 5; i++ {
		<-ch.Ch
	}

	// Verify recorded
	events := ch.GetReceivedEvents()
	require.Len(t, events, 5)
}

func TestMockEventChannel_SendToClosedChannel(t *testing.T) {
	t.Parallel()

	ch := NewMockEventChannel(10)
	ch.Close()

	event := session.Event{
		ID:      "TAS-1",
		Session: "test-session",
		Type:    "task",
		Action:  "add",
		Data:    "Test task",
	}

	// Send to closed channel
	err := ch.Send(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed channel")
}

func TestMockEventChannel_SendToFullChannel(t *testing.T) {
	t.Parallel()

	ch := NewMockEventChannel(2) // Small buffer
	defer ch.Close()

	// Fill channel
	for i := 0; i < 2; i++ {
		event := session.Event{ID: fmt.Sprintf("TAS-%d", i+1)}
		err := ch.Send(event)
		require.NoError(t, err)
	}

	// Try to send to full channel
	event := session.Event{ID: "TAS-3"}
	err := ch.Send(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "channel full")
}

func TestMockEventChannel_Reset(t *testing.T) {
	t.Parallel()

	ch := NewMockEventChannel(10)
	defer ch.Close()

	// Send events
	for i := 0; i < 3; i++ {
		event := session.Event{ID: fmt.Sprintf("TAS-%d", i+1)}
		_ = ch.Send(event)
	}

	require.Len(t, ch.GetReceivedEvents(), 3)

	// Reset
	ch.Reset()

	// Verify reset
	events := ch.GetReceivedEvents()
	require.Empty(t, events)
}

// --- MockGit Tests ---

func TestMockGit_GetInfo(t *testing.T) {
	t.Parallel()

	git := NewMockGit()

	// Get info
	info, err := git.GetInfo("/tmp/repo")
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, "main", info.Branch)
	require.Equal(t, FixedGitHash, info.Hash)
	require.False(t, info.Dirty)
	require.Equal(t, 0, info.Ahead)
	require.Equal(t, 0, info.Behind)
	require.Equal(t, 1, git.GetInfoCalls)
	require.Equal(t, "/tmp/repo", git.LastDir)
}

func TestMockGit_GetInfoError(t *testing.T) {
	t.Parallel()

	git := NewMockGit()

	// Set error
	expectedErr := fmt.Errorf("git failed")
	git.GetInfoError = expectedErr

	// Get info
	info, err := git.GetInfo("/tmp/repo")
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Nil(t, info)
	require.Equal(t, 1, git.GetInfoCalls)
}

func TestMockGit_SetDirty(t *testing.T) {
	t.Parallel()

	git := NewMockGit()

	// Initially clean
	info, _ := git.GetInfo("/tmp/repo")
	require.False(t, info.Dirty)

	// Set dirty
	git.SetDirty(true)

	// Verify dirty
	info, _ = git.GetInfo("/tmp/repo")
	require.True(t, info.Dirty)
}

func TestMockGit_SetBranch(t *testing.T) {
	t.Parallel()

	git := NewMockGit()

	// Set branch
	git.SetBranch("feature-branch")

	// Verify branch
	info, _ := git.GetInfo("/tmp/repo")
	require.Equal(t, "feature-branch", info.Branch)
}

func TestMockGit_SetAheadBehind(t *testing.T) {
	t.Parallel()

	git := NewMockGit()

	// Set ahead/behind
	git.SetAheadBehind(5, 3)

	// Verify
	info, _ := git.GetInfo("/tmp/repo")
	require.Equal(t, 5, info.Ahead)
	require.Equal(t, 3, info.Behind)
}

func TestMockGit_Reset(t *testing.T) {
	t.Parallel()

	git := NewMockGit()

	// Perform operations
	_, _ = git.GetInfo("/tmp/repo1")
	_, _ = git.GetInfo("/tmp/repo2")

	require.Equal(t, 2, git.GetInfoCalls)
	require.Equal(t, "/tmp/repo2", git.LastDir)

	// Reset
	git.Reset()

	// Verify reset
	require.Equal(t, 0, git.GetInfoCalls)
	require.Empty(t, git.LastDir)
}

func TestMockGit_NilInfo(t *testing.T) {
	t.Parallel()

	git := NewMockGit()
	git.Info = nil

	// Get info returns nil (repo not found case)
	info, err := git.GetInfo("/tmp/repo")
	require.NoError(t, err)
	require.Nil(t, info)
}

func TestMockGit_CustomInfo(t *testing.T) {
	t.Parallel()

	gitMock := NewMockGit()

	// Set custom info
	customInfo := &git.Info{
		Branch: "develop",
		Hash:   "fedcba9",
		Dirty:  true,
		Ahead:  10,
		Behind: 5,
	}
	gitMock.Info = customInfo

	// Get info
	info, err := gitMock.GetInfo("/tmp/repo")
	require.NoError(t, err)
	require.Equal(t, customInfo, info)
}

// --- MockOrchestrator Tests ---

func TestMockOrchestrator_RequestPause(t *testing.T) {
	t.Parallel()

	orch := NewMockOrchestrator()

	require.False(t, orch.WasPauseRequested())

	// Request pause
	orch.RequestPause()

	require.True(t, orch.WasPauseRequested())
}

func TestMockOrchestrator_CancelPause(t *testing.T) {
	t.Parallel()

	orch := NewMockOrchestrator()

	require.False(t, orch.WasPauseCancelled())

	// Cancel pause
	orch.CancelPause()

	require.True(t, orch.WasPauseCancelled())
}

func TestMockOrchestrator_Resume(t *testing.T) {
	t.Parallel()

	orch := NewMockOrchestrator()

	require.False(t, orch.WasResumed())

	// Resume
	orch.Resume()

	require.True(t, orch.WasResumed())
}

func TestMockOrchestrator_IsPaused(t *testing.T) {
	t.Parallel()

	orch := NewMockOrchestrator()

	// Initially not paused
	require.False(t, orch.IsPaused())

	// Set paused
	orch.SetPaused(true)

	// Verify paused
	require.True(t, orch.IsPaused())

	// Set not paused
	orch.SetPaused(false)

	// Verify not paused
	require.False(t, orch.IsPaused())
}

func TestMockOrchestrator_Reset(t *testing.T) {
	t.Parallel()

	orch := NewMockOrchestrator()

	// Perform operations
	orch.RequestPause()
	orch.CancelPause()
	orch.Resume()
	orch.SetPaused(true)

	require.True(t, orch.WasPauseRequested())
	require.True(t, orch.WasPauseCancelled())
	require.True(t, orch.WasResumed())
	require.True(t, orch.IsPaused())

	// Reset
	orch.Reset()

	// Verify reset
	require.False(t, orch.WasPauseRequested())
	require.False(t, orch.WasPauseCancelled())
	require.False(t, orch.WasResumed())
	require.False(t, orch.IsPaused())
}

func TestMockOrchestrator_ThreadSafety(t *testing.T) {
	t.Parallel()

	orch := NewMockOrchestrator()

	var wg sync.WaitGroup
	concurrency := 100

	// Concurrent operations
	wg.Add(concurrency * 4)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			orch.RequestPause()
		}()
		go func() {
			defer wg.Done()
			orch.CancelPause()
		}()
		go func() {
			defer wg.Done()
			orch.Resume()
		}()
		go func() {
			defer wg.Done()
			_ = orch.IsPaused()
		}()
	}

	wg.Wait()

	// Verify operations completed (no panics)
	require.True(t, orch.WasPauseRequested())
	require.True(t, orch.WasPauseCancelled())
	require.True(t, orch.WasResumed())
}
