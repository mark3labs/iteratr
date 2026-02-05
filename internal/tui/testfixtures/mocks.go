// Package testfixtures provides mock implementations and test utilities for TUI testing.
//
// This file contains mock implementations for common dependencies used in TUI tests:
//   - MockStore: Mock implementation of session.Store for event sourcing operations
//   - MockEventChannel: Controllable event channel for testing NATS event consumption
//   - MockGit: Mock implementation for git repository operations
//   - MockOrchestrator: Mock implementation of the Orchestrator interface for pause/resume control
//
// All mocks are thread-safe and provide verification methods for assertions in tests.
//
// Example usage:
//
//	func TestMyComponent(t *testing.T) {
//	    store := testfixtures.NewMockStore()
//	    store.State = testfixtures.StateWithTasks()
//
//	    git := testfixtures.NewMockGit()
//	    git.SetDirty(true)
//
//	    orch := testfixtures.NewMockOrchestrator()
//
//	    // Use mocks in your test...
//	    // Later verify calls:
//	    require.Equal(t, 1, store.LoadStateCalls)
//	    require.True(t, orch.WasPauseRequested())
//	}
package testfixtures

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/iteratr/internal/git"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/nats-io/nats.go/jetstream"
)

// MockStore is a mock implementation of session.Store for testing.
// It provides controllable state loading and event publishing without requiring NATS.
type MockStore struct {
	mu sync.RWMutex

	// State to return from LoadState
	State *session.State
	// Error to return from LoadState
	LoadError error

	// Published events are recorded here
	PublishedEvents []session.Event
	// Error to return from PublishEvent
	PublishError error

	// Sessions to return from ListSessions
	Sessions []session.SessionInfo
	// Error to return from ListSessions
	ListError error

	// Error to return from ResetSession
	ResetError error

	// Counters for verification
	LoadStateCalls  int
	PublishCalls    int
	ListCalls       int
	ResetCalls      int
	LastSessionName string
}

// NewMockStore creates a new MockStore with default empty state.
func NewMockStore() *MockStore {
	return &MockStore{
		State:           EmptyState(),
		PublishedEvents: []session.Event{},
		Sessions:        []session.SessionInfo{},
	}
}

// LoadState returns the configured state or error.
func (m *MockStore) LoadState(ctx context.Context, sessionName string) (*session.State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LoadStateCalls++
	m.LastSessionName = sessionName

	if m.LoadError != nil {
		return nil, m.LoadError
	}

	// Return copy of state to prevent test modifications from affecting mock
	return m.State, nil
}

// PublishEvent records the event and returns configured error.
func (m *MockStore) PublishEvent(ctx context.Context, event session.Event) (*jetstream.PubAck, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PublishCalls++
	m.PublishedEvents = append(m.PublishedEvents, event)

	if m.PublishError != nil {
		return nil, m.PublishError
	}

	// Return mock PubAck
	return &jetstream.PubAck{
		Stream:   "iteratr_events",
		Sequence: uint64(len(m.PublishedEvents)),
	}, nil
}

// ListSessions returns the configured session list or error.
func (m *MockStore) ListSessions(ctx context.Context) ([]session.SessionInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ListCalls++

	if m.ListError != nil {
		return nil, m.ListError
	}

	return m.Sessions, nil
}

// ResetSession returns the configured error.
func (m *MockStore) ResetSession(ctx context.Context, sessionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ResetCalls++
	m.LastSessionName = sessionName

	return m.ResetError
}

// GetPublishedEvents returns a copy of published events (thread-safe).
func (m *MockStore) GetPublishedEvents() []session.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]session.Event, len(m.PublishedEvents))
	copy(events, m.PublishedEvents)
	return events
}

// Reset clears all recorded calls and events.
func (m *MockStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LoadStateCalls = 0
	m.PublishCalls = 0
	m.ListCalls = 0
	m.ResetCalls = 0
	m.LastSessionName = ""
	m.PublishedEvents = []session.Event{}
}

// MockEventChannel provides a controllable event channel for testing.
// It allows tests to send events and verify event consumption.
type MockEventChannel struct {
	mu sync.Mutex

	// Channel for sending events to the app
	Ch chan session.Event

	// Received events are recorded here for verification
	ReceivedEvents []session.Event

	// Closed tracks if channel was closed
	Closed bool
}

// NewMockEventChannel creates a new mock event channel with the given buffer size.
func NewMockEventChannel(bufferSize int) *MockEventChannel {
	return &MockEventChannel{
		Ch:             make(chan session.Event, bufferSize),
		ReceivedEvents: []session.Event{},
	}
}

// Send sends an event to the channel (non-blocking if channel is buffered).
func (m *MockEventChannel) Send(event session.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Closed {
		return fmt.Errorf("cannot send to closed channel")
	}

	select {
	case m.Ch <- event:
		m.ReceivedEvents = append(m.ReceivedEvents, event)
		return nil
	default:
		return fmt.Errorf("channel full")
	}
}

// Close closes the channel.
func (m *MockEventChannel) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.Closed {
		close(m.Ch)
		m.Closed = true
	}
}

// GetReceivedEvents returns a copy of received events (thread-safe).
func (m *MockEventChannel) GetReceivedEvents() []session.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	events := make([]session.Event, len(m.ReceivedEvents))
	copy(events, m.ReceivedEvents)
	return events
}

// Reset clears received events.
func (m *MockEventChannel) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReceivedEvents = []session.Event{}
}

// MockGit is a mock implementation for git operations.
// It provides controllable git information without requiring a real git repository.
type MockGit struct {
	mu sync.RWMutex

	// Info to return from GetInfo
	Info *git.Info
	// Error to return from GetInfo
	GetInfoError error

	// Counters for verification
	GetInfoCalls int
	LastDir      string
}

// NewMockGit creates a new MockGit with default test values.
func NewMockGit() *MockGit {
	return &MockGit{
		Info: &git.Info{
			Branch: "main",
			Hash:   FixedGitHash,
			Dirty:  false,
			Ahead:  0,
			Behind: 0,
		},
	}
}

// GetInfo returns the configured git info or error.
func (m *MockGit) GetInfo(dir string) (*git.Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetInfoCalls++
	m.LastDir = dir

	if m.GetInfoError != nil {
		return nil, m.GetInfoError
	}

	return m.Info, nil
}

// SetDirty sets the dirty flag in the git info.
func (m *MockGit) SetDirty(dirty bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Info != nil {
		m.Info.Dirty = dirty
	}
}

// SetBranch sets the branch name in the git info.
func (m *MockGit) SetBranch(branch string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Info != nil {
		m.Info.Branch = branch
	}
}

// SetAheadBehind sets the ahead/behind counts in the git info.
func (m *MockGit) SetAheadBehind(ahead, behind int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Info != nil {
		m.Info.Ahead = ahead
		m.Info.Behind = behind
	}
}

// Reset clears all recorded calls.
func (m *MockGit) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetInfoCalls = 0
	m.LastDir = ""
}

// MockOrchestrator is a mock implementation of the Orchestrator interface for testing.
// It provides controllable pause/resume state without requiring a real orchestrator.
type MockOrchestrator struct {
	mu sync.RWMutex

	paused         bool
	pauseRequested bool
	pauseCancelled bool
	resumed        bool
}

// NewMockOrchestrator creates a new MockOrchestrator.
func NewMockOrchestrator() *MockOrchestrator {
	return &MockOrchestrator{}
}

// RequestPause marks pause as requested.
func (m *MockOrchestrator) RequestPause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pauseRequested = true
}

// CancelPause marks pause as cancelled.
func (m *MockOrchestrator) CancelPause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pauseCancelled = true
}

// Resume marks orchestrator as resumed.
func (m *MockOrchestrator) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resumed = true
}

// IsPaused returns the paused state.
func (m *MockOrchestrator) IsPaused() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.paused
}

// SetPaused sets the paused state (for testing).
func (m *MockOrchestrator) SetPaused(paused bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paused = paused
}

// WasPauseRequested returns true if RequestPause was called.
func (m *MockOrchestrator) WasPauseRequested() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pauseRequested
}

// WasPauseCancelled returns true if CancelPause was called.
func (m *MockOrchestrator) WasPauseCancelled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pauseCancelled
}

// WasResumed returns true if Resume was called.
func (m *MockOrchestrator) WasResumed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resumed
}

// Reset clears all state.
func (m *MockOrchestrator) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.paused = false
	m.pauseRequested = false
	m.pauseCancelled = false
	m.resumed = false
}
