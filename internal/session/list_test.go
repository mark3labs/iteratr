package session

import (
	"context"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
)

func TestListSessions(t *testing.T) {
	// Setup: Create embedded NATS and store
	ctx := context.Background()
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nats.CreateJetStream(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream: %v", err)
	}

	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("failed to setup stream: %v", err)
	}

	store := NewStore(js, stream)

	t.Run("ListSessions returns empty when no sessions exist", func(t *testing.T) {
		infos, err := store.ListSessions(ctx)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}
		if len(infos) != 0 {
			t.Errorf("expected 0 sessions, got %d", len(infos))
		}
	})

	t.Run("ListSessions returns session info", func(t *testing.T) {
		// Create a few sessions with tasks
		session1 := "test-session-1"
		session2 := "test-session-2"
		session3 := "test-session-3"

		// Session 1: 3 tasks, 2 completed
		_, _ = store.TaskAdd(ctx, session1, TaskAddParams{
			Content:   "Task 1.1",
			Status:    "completed",
			Iteration: 1,
		})
		_, _ = store.TaskAdd(ctx, session1, TaskAddParams{
			Content:   "Task 1.2",
			Status:    "completed",
			Iteration: 1,
		})
		_, _ = store.TaskAdd(ctx, session1, TaskAddParams{
			Content:   "Task 1.3",
			Status:    "remaining",
			Iteration: 1,
		})

		// Session 2: 5 tasks, 5 completed (fully complete)
		for i := 1; i <= 5; i++ {
			_, _ = store.TaskAdd(ctx, session2, TaskAddParams{
				Content:   "Task 2." + string(rune('0'+i)),
				Status:    "completed",
				Iteration: 1,
			})
		}
		_ = store.SessionComplete(ctx, session2) // Mark session complete

		// Session 3: 2 tasks, 0 completed
		_, _ = store.TaskAdd(ctx, session3, TaskAddParams{
			Content:   "Task 3.1",
			Status:    "remaining",
			Iteration: 1,
		})
		_, _ = store.TaskAdd(ctx, session3, TaskAddParams{
			Content:   "Task 3.2",
			Status:    "in_progress",
			Iteration: 1,
		})

		// List sessions
		infos, err := store.ListSessions(ctx)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		// Should have 3 sessions
		if len(infos) != 3 {
			t.Fatalf("expected 3 sessions, got %d", len(infos))
		}

		// Find each session in the results
		var info1, info2, info3 *SessionInfo
		for i := range infos {
			switch infos[i].Name {
			case session1:
				info1 = &infos[i]
			case session2:
				info2 = &infos[i]
			case session3:
				info3 = &infos[i]
			}
		}

		// Verify session 1
		if info1 == nil {
			t.Fatal("session 1 not found")
		}
		if info1.TasksTotal != 3 {
			t.Errorf("session 1: expected 3 total tasks, got %d", info1.TasksTotal)
		}
		if info1.TasksCompleted != 2 {
			t.Errorf("session 1: expected 2 completed tasks, got %d", info1.TasksCompleted)
		}
		if info1.Complete {
			t.Error("session 1: expected incomplete")
		}
		if info1.LastActivity.IsZero() {
			t.Error("session 1: expected non-zero last activity")
		}

		// Verify session 2
		if info2 == nil {
			t.Fatal("session 2 not found")
		}
		if info2.TasksTotal != 5 {
			t.Errorf("session 2: expected 5 total tasks, got %d", info2.TasksTotal)
		}
		if info2.TasksCompleted != 5 {
			t.Errorf("session 2: expected 5 completed tasks, got %d", info2.TasksCompleted)
		}
		if !info2.Complete {
			t.Error("session 2: expected complete")
		}

		// Verify session 3
		if info3 == nil {
			t.Fatal("session 3 not found")
		}
		if info3.TasksTotal != 2 {
			t.Errorf("session 3: expected 2 total tasks, got %d", info3.TasksTotal)
		}
		if info3.TasksCompleted != 0 {
			t.Errorf("session 3: expected 0 completed tasks, got %d", info3.TasksCompleted)
		}
		if info3.Complete {
			t.Error("session 3: expected incomplete")
		}
	})

	t.Run("ListSessions sorts by last activity", func(t *testing.T) {
		// Create a new session (will have most recent activity)
		recentSession := "test-session-recent"
		_, _ = store.TaskAdd(ctx, recentSession, TaskAddParams{
			Content:   "Recent task",
			Iteration: 1,
		})

		// List sessions
		infos, err := store.ListSessions(ctx)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		// First session should be the most recent
		if infos[0].Name != recentSession {
			t.Errorf("expected first session to be %s, got %s", recentSession, infos[0].Name)
		}

		// Verify they're sorted by descending LastActivity
		for i := 1; i < len(infos); i++ {
			if infos[i].LastActivity.After(infos[i-1].LastActivity) {
				t.Error("sessions not sorted by last activity descending")
				break
			}
		}
	})
}
