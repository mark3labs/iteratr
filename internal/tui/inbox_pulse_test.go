package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestInboxPanel_PulseOnNewMessage(t *testing.T) {
	// Create inbox panel
	inbox := NewInboxPanel()
	inbox.SetSize(80, 24)

	// Set initial state with no messages
	initialState := &session.State{
		Inbox: []*session.Message{},
	}
	inbox.SetState(initialState)

	// Pulse should not be active initially
	if inbox.pulse.IsActive() {
		t.Error("Pulse should not be active initially")
	}

	// Add a new message
	newState := &session.State{
		Inbox: []*session.Message{
			{
				ID:        "msg1",
				Content:   "New message",
				Read:      false,
				CreatedAt: time.Now(),
			},
		},
	}
	inbox.SetState(newState)

	// Check that needsPulse flag is set
	if !inbox.needsPulse {
		t.Error("needsPulse should be true after new message")
	}

	// Trigger an Update to start the pulse
	cmd := inbox.Update(tea.KeyPressMsg{}) // Any message will trigger the check
	if cmd == nil {
		t.Error("Update should return pulse start command")
	}

	// Pulse should now be active
	if !inbox.pulse.IsActive() {
		t.Error("Pulse should be active after starting")
	}

	// needsPulse should be cleared
	if inbox.needsPulse {
		t.Error("needsPulse should be false after pulse starts")
	}
}

func TestInboxPanel_NoPulseOnSameMessage(t *testing.T) {
	// Create inbox panel
	inbox := NewInboxPanel()
	inbox.SetSize(80, 24)

	// Set initial empty state
	emptyState := &session.State{
		Inbox: []*session.Message{},
	}
	inbox.SetState(emptyState)

	// Set state with a message (will trigger pulse as it's "new")
	initialState := &session.State{
		Inbox: []*session.Message{
			{
				ID:        "msg1",
				Content:   "Existing message",
				Read:      false,
				CreatedAt: time.Now(),
			},
		},
	}
	inbox.SetState(initialState)

	// Trigger update to start initial pulse
	inbox.Update(tea.KeyPressMsg{})

	// Wait for pulse to complete
	for i := 0; i < 15; i++ {
		if inbox.pulse.IsActive() {
			inbox.Update(PulseMsg{})
		}
	}

	// Set same state again
	inbox.SetState(initialState)

	// needsPulse should NOT be set (same message already pulsed)
	if inbox.needsPulse {
		t.Error("needsPulse should not be set for same message")
	}
}

func TestInboxPanel_PulseVisualIndicator(t *testing.T) {
	// Create inbox panel
	inbox := NewInboxPanel()
	inbox.SetSize(80, 24)

	// Set initial empty state first
	emptyState := &session.State{
		Inbox: []*session.Message{},
	}
	inbox.SetState(emptyState)

	// Now set state with message to trigger pulse
	state := &session.State{
		Inbox: []*session.Message{
			{
				ID:        "msg1",
				Content:   "New message",
				Read:      false,
				CreatedAt: time.Now(),
			},
		},
	}
	inbox.SetState(state)

	// Start pulse
	inbox.Update(tea.KeyPressMsg{})

	// During pulse (high intensity), title should include indicator
	// We can't easily test Draw output, but we can verify pulse is active
	if !inbox.pulse.IsActive() {
		t.Error("Pulse should be active")
	}

	intensity := inbox.pulse.Intensity()
	if intensity < 0 || intensity > 1 {
		t.Errorf("Pulse intensity should be between 0 and 1, got %f", intensity)
	}
}
