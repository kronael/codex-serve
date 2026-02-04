package main

import (
	"testing"
	"time"
)

// Test session lifecycle
func TestSessionLifecycle(t *testing.T) {
	sm := NewSessionManager(1 * time.Hour)
	defer sm.Close()

	id := sm.Create()
	if id == "" {
		t.Fatal("expected non-empty session id")
	}

	session, exists := sm.Get(id)
	if !exists {
		t.Fatal("expected session to exist")
	}

	if session.State != StateCreated {
		t.Errorf("expected state CREATED, got %s", session.State)
	}

	deleted := sm.Delete(id)
	if !deleted {
		t.Error("expected session to be deleted")
	}

	_, exists = sm.Get(id)
	if exists {
		t.Error("expected session to not exist after deletion")
	}
}

// Test session state transitions
func TestSessionStateTransitions(t *testing.T) {
	sm := NewSessionManager(1 * time.Hour)
	defer sm.Close()

	id := sm.Create()
	session, _ := sm.Get(id)

	session.UpdateState(StateRunning)
	if session.GetState() != StateRunning {
		t.Errorf("expected state RUNNING, got %s", session.GetState())
	}

	session.UpdateState(StateCompleted)
	if session.GetState() != StateCompleted {
		t.Errorf("expected state COMPLETED, got %s", session.GetState())
	}
}

// Test adding messages to session
func TestSessionMessages(t *testing.T) {
	sm := NewSessionManager(1 * time.Hour)
	defer sm.Close()

	id := sm.Create()
	session, _ := sm.Get(id)

	msg1 := Message{Role: "user", Content: "hello"}
	msg2 := Message{Role: "assistant", Content: "hi there"}

	session.AddMessage(msg1)
	session.AddMessage(msg2)

	history := session.GetHistory()
	if len(history) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history))
	}

	if history[0].Role != "user" || history[0].Content != "hello" {
		t.Error("first message mismatch")
	}
	if history[1].Role != "assistant" || history[1].Content != "hi there" {
		t.Error("second message mismatch")
	}
}

// Test usage tracking
func TestSessionUsage(t *testing.T) {
	sm := NewSessionManager(1 * time.Hour)
	defer sm.Close()

	id := sm.Create()
	session, _ := sm.Get(id)

	session.UpdateUsage(100, 50)
	session.UpdateUsage(200, 75)

	usage := session.GetUsage()
	if usage.InputTokens != 300 {
		t.Errorf("expected 300 input tokens, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 125 {
		t.Errorf("expected 125 output tokens, got %d", usage.OutputTokens)
	}
}

// Test session list
func TestSessionList(t *testing.T) {
	sm := NewSessionManager(1 * time.Hour)
	defer sm.Close()

	id1 := sm.Create()
	id2 := sm.Create()
	id3 := sm.Create()

	ids := sm.List()
	if len(ids) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(ids))
	}

	found := make(map[string]bool)
	for _, id := range ids {
		found[id] = true
	}

	if !found[id1] || !found[id2] || !found[id3] {
		t.Error("not all session ids found in list")
	}
}

// Test session cleanup
func TestSessionCleanup(t *testing.T) {
	sm := NewSessionManager(100 * time.Millisecond)
	defer sm.Close()

	id := sm.Create()
	session, _ := sm.Get(id)

	session.UpdateState(StateCompleted)

	time.Sleep(200 * time.Millisecond)

	sm.cleanup()

	_, exists := sm.Get(id)
	if exists {
		t.Error("expected stale session to be cleaned up")
	}
}

// Test concurrent session access
func TestSessionConcurrency(t *testing.T) {
	sm := NewSessionManager(1 * time.Hour)
	defer sm.Close()

	id := sm.Create()
	session, _ := sm.Get(id)

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			session.AddMessage(Message{Role: "user", Content: "test"})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			session.UpdateUsage(10, 5)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			session.UpdateState(StateRunning)
		}
		done <- true
	}()

	<-done
	<-done
	<-done

	history := session.GetHistory()
	if len(history) != 100 {
		t.Errorf("expected 100 messages, got %d", len(history))
	}

	usage := session.GetUsage()
	if usage.InputTokens != 1000 || usage.OutputTokens != 500 {
		t.Errorf("expected 1000/500 tokens, got %d/%d", usage.InputTokens, usage.OutputTokens)
	}
}
