package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestToast_ShowDisplaysMessage(t *testing.T) {
	toast := NewToast()

	cmd := toast.Show("test message")

	if !toast.IsVisible() {
		t.Error("expected toast to be visible after Show()")
	}

	if toast.GetMessage() != "test message" {
		t.Errorf("expected message 'test message', got %q", toast.GetMessage())
	}

	// Verify command is returned (for dismissal)
	if cmd == nil {
		t.Error("expected Show() to return a command for dismissal")
	}
}

func TestToast_ViewReturnsEmptyWhenNotVisible(t *testing.T) {
	toast := NewToast()

	view := toast.View(80, 24)

	if view != "" {
		t.Errorf("expected empty view when not visible, got %q", view)
	}
}

func TestToast_ViewRendersMessageWhenVisible(t *testing.T) {
	toast := NewToast()
	toast.Show("142 chars truncated")

	view := toast.View(80, 24)

	if view == "" {
		t.Error("expected non-empty view when visible")
	}

	// The view should contain the message
	if !strings.Contains(view, "142 chars truncated") {
		t.Errorf("expected view to contain message, got %q", view)
	}
}

func TestToast_DismissMsgHidesToast(t *testing.T) {
	toast := NewToast()
	toast.Show("test message")

	// Send dismiss message with matching generation
	cmd := toast.Update(ToastDismissMsg{Generation: toast.generation})

	if toast.IsVisible() {
		t.Error("expected toast to be hidden after ToastDismissMsg")
	}

	if toast.GetMessage() != "" {
		t.Error("expected message to be cleared after dismiss")
	}

	if cmd != nil {
		t.Error("expected no command after dismiss")
	}
}

func TestToast_ShowClearsPreviousMessage(t *testing.T) {
	toast := NewToast()
	toast.Show("first message")
	toast.Show("second message")

	if toast.GetMessage() != "second message" {
		t.Errorf("expected 'second message', got %q", toast.GetMessage())
	}
}

func TestToast_ShowUpdatesDismissTime(t *testing.T) {
	toast := NewToast()

	// Show first message
	toast.Show("first")
	firstDismissAt := toast.dismissAt

	// Wait a tiny bit
	time.Sleep(10 * time.Millisecond)

	// Show second message - should reset dismiss time
	toast.Show("second")
	secondDismissAt := toast.dismissAt

	if !secondDismissAt.After(firstDismissAt) {
		t.Error("expected second dismiss time to be after first")
	}
}

func TestToast_ViewReturnsContentWithoutPositioning(t *testing.T) {
	toast := NewToast()
	toast.Show("test")

	view := toast.View(80, 24)

	// View() should return only the styled content without positioning
	// (positioning is handled by app.go Draw() method)
	if view == "" {
		t.Error("expected non-empty view when visible")
	}

	// Should contain the message
	if !strings.Contains(view, "test") {
		t.Errorf("expected view to contain message, got %q", view)
	}

	// Should NOT have vertical positioning (newlines) - that's app.go's job
	lines := strings.Split(view, "\n")
	if len(lines) != 1 {
		t.Errorf("expected single line (no vertical positioning), got %d lines", len(lines))
	}
}

func TestToast_ViewHandlesNarrowWidth(t *testing.T) {
	toast := NewToast()
	toast.Show("very long message that might exceed narrow width")

	// Test with very narrow width
	view := toast.View(10, 24)

	// Should still render something
	if view == "" {
		t.Error("expected view even with narrow width")
	}
}

func TestToast_ViewHandlesZeroDimensions(t *testing.T) {
	toast := NewToast()
	toast.Show("test")

	view := toast.View(0, 0)

	// Should handle gracefully
	if view == "" {
		// This is acceptable for zero dimensions
		t.Log("view is empty for zero dimensions (acceptable)")
	}
}

func TestToast_DismissCmdCalculatesRemainingTime(t *testing.T) {
	toast := NewToast()
	toast.Show("test")

	// Get the dismiss command
	cmd := toast.dismissCmd()

	if cmd == nil {
		t.Error("expected non-nil dismiss command")
	}

	// Execute the command to verify it returns a message
	msg := cmd()

	// Should return a ToastDismissMsg
	if _, ok := msg.(ToastDismissMsg); !ok {
		t.Errorf("expected ToastDismissMsg, got %T", msg)
	}
}

func TestToast_UpdateIgnoresOtherMessages(t *testing.T) {
	toast := NewToast()
	toast.Show("test")

	// Send unrelated message
	cmd := toast.Update(tea.KeyPressMsg{})

	// Toast should still be visible
	if !toast.IsVisible() {
		t.Error("expected toast to remain visible after unrelated message")
	}

	if cmd != nil {
		t.Error("expected no command for unrelated message")
	}
}

func TestToast_ShowToastMsgType(t *testing.T) {
	// Verify the message type exists and has the expected structure
	msg := ShowToastMsg{Text: "test notification"}

	if msg.Text != "test notification" {
		t.Errorf("expected text 'test notification', got %q", msg.Text)
	}
}

func TestToast_MultipleShowCalls(t *testing.T) {
	toast := NewToast()

	// Show multiple messages in sequence
	messages := []string{
		"first notification",
		"second notification",
		"third notification",
	}

	for _, msg := range messages {
		toast.Show(msg)

		if toast.GetMessage() != msg {
			t.Errorf("expected message %q, got %q", msg, toast.GetMessage())
		}

		if !toast.IsVisible() {
			t.Error("expected toast to be visible")
		}
	}
}

func TestToast_DismissSequence(t *testing.T) {
	toast := NewToast()

	// Show toast
	cmd := toast.Show("test")
	if cmd == nil {
		t.Fatal("expected Show() to return dismiss command")
	}

	// Execute the command to get the dismiss message
	msg := cmd()
	dismissMsg, ok := msg.(ToastDismissMsg)
	if !ok {
		t.Fatalf("expected ToastDismissMsg from command, got %T", msg)
	}

	// Send the dismiss message to the toast
	toast.Update(dismissMsg)

	// Toast should now be hidden
	if toast.IsVisible() {
		t.Error("expected toast to be hidden after complete dismiss sequence")
	}

	// View should be empty
	view := toast.View(80, 24)
	if view != "" {
		t.Errorf("expected empty view after dismiss, got %q", view)
	}
}

func TestToast_StaleDismissDoesNotHideNewToast(t *testing.T) {
	toast := NewToast()

	// Show first toast
	cmd1 := toast.Show("first message")
	if cmd1 == nil {
		t.Fatal("expected Show() to return dismiss command")
	}

	// Get the dismiss message for the first toast
	msg1 := cmd1()
	dismissMsg1, ok := msg1.(ToastDismissMsg)
	if !ok {
		t.Fatalf("expected ToastDismissMsg from command, got %T", msg1)
	}

	// Show second toast (this increments generation)
	toast.Show("second message")

	// Send the stale dismiss message from the first toast
	toast.Update(dismissMsg1)

	// Second toast should still be visible (stale dismiss ignored)
	if !toast.IsVisible() {
		t.Error("expected second toast to remain visible after stale dismiss")
	}

	if toast.GetMessage() != "second message" {
		t.Errorf("expected message 'second message', got %q", toast.GetMessage())
	}
}
