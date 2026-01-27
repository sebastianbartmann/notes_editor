package claude

import (
	"sync"
	"testing"
)

func TestSession_AddAndGetMessages(t *testing.T) {
	session := &Session{
		ID:       "test-session",
		Person:   "sebastian",
		Messages: make([]ChatMessage, 0),
	}

	session.AddMessage("user", "Hello")
	session.AddMessage("assistant", "Hi there!")

	messages := session.GetMessages()
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Hello" {
		t.Errorf("unexpected first message: %+v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "Hi there!" {
		t.Errorf("unexpected second message: %+v", messages[1])
	}
}

func TestSession_ThreadSafety(t *testing.T) {
	session := &Session{
		ID:       "test-session",
		Person:   "sebastian",
		Messages: make([]ChatMessage, 0),
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session.AddMessage("user", "concurrent message")
		}()
	}
	wg.Wait()

	messages := session.GetMessages()
	if len(messages) != 100 {
		t.Errorf("expected 100 messages, got %d", len(messages))
	}
}

func TestSessionStore_GetOrCreate(t *testing.T) {
	store := NewSessionStore()

	// Create new session
	session1 := store.GetOrCreate("", "sebastian")
	if session1.ID == "" {
		t.Error("session ID should not be empty")
	}
	if session1.Person != "sebastian" {
		t.Errorf("expected person 'sebastian', got %q", session1.Person)
	}

	// Get existing session
	session2 := store.GetOrCreate(session1.ID, "sebastian")
	if session2.ID != session1.ID {
		t.Error("should return same session for same ID and person")
	}

	// Different person should create new session
	session3 := store.GetOrCreate(session1.ID, "petra")
	if session3.ID == session1.ID {
		t.Error("should create new session for different person")
	}
}

func TestSessionStore_Clear(t *testing.T) {
	store := NewSessionStore()

	session := store.GetOrCreate("", "sebastian")
	sessionID := session.ID

	// Verify session exists
	if store.Get(sessionID) == nil {
		t.Error("session should exist")
	}

	// Clear session
	store.Clear(sessionID)

	// Verify session is gone
	if store.Get(sessionID) != nil {
		t.Error("session should be cleared")
	}
}

func TestSessionStore_GetHistory(t *testing.T) {
	store := NewSessionStore()

	// Empty history for non-existent session
	history := store.GetHistory("non-existent")
	if history != nil {
		t.Errorf("expected nil history, got %v", history)
	}

	// Add messages and get history
	session := store.GetOrCreate("", "sebastian")
	session.AddMessage("user", "Hello")
	session.AddMessage("assistant", "Hi!")

	history = store.GetHistory(session.ID)
	if len(history) != 2 {
		t.Errorf("expected 2 messages in history, got %d", len(history))
	}
}
