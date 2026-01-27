// Package claude provides Claude AI service integration with tool use and streaming.
package claude

import (
	"sync"

	"github.com/google/uuid"
)

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // Message text
}

// Session holds conversation state for a single chat session.
type Session struct {
	ID       string
	Person   string
	Messages []ChatMessage
	mu       sync.Mutex
}

// AddMessage adds a message to the session history.
func (s *Session) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, ChatMessage{Role: role, Content: content})
}

// GetMessages returns a copy of the session's message history.
func (s *Session) GetMessages() []ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ChatMessage, len(s.Messages))
	copy(result, s.Messages)
	return result
}

// SessionStore manages chat sessions with thread-safe access.
type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate returns an existing session or creates a new one.
// If the session exists but belongs to a different person, a new session is created.
func (ss *SessionStore) GetOrCreate(sessionID, person string) *Session {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if sessionID != "" {
		if session, ok := ss.sessions[sessionID]; ok {
			if session.Person == person {
				return session
			}
			// Different person - create new session
		}
	}

	// Create new session
	newID := uuid.New().String()
	session := &Session{
		ID:       newID,
		Person:   person,
		Messages: make([]ChatMessage, 0),
	}
	ss.sessions[newID] = session
	return session
}

// Get retrieves a session by ID, returning nil if not found.
func (ss *SessionStore) Get(sessionID string) *Session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.sessions[sessionID]
}

// Clear removes a session from the store.
func (ss *SessionStore) Clear(sessionID string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.sessions, sessionID)
}

// GetHistory returns the message history for a session.
func (ss *SessionStore) GetHistory(sessionID string) []ChatMessage {
	ss.mu.RLock()
	session := ss.sessions[sessionID]
	ss.mu.RUnlock()

	if session == nil {
		return nil
	}
	return session.GetMessages()
}
