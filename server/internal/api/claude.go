package api

import (
	"encoding/json"
	"net/http"

	"notes-editor/internal/claude"
)

// handleClaudeChat handles non-streaming chat requests.
func (s *Server) handleClaudeChat(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	if s.claude == nil {
		writeBadRequest(w, "Claude service not configured")
		return
	}

	var req claude.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Message == "" {
		writeBadRequest(w, "Message is required")
		return
	}

	resp, err := s.claude.Chat(person, req)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleClaudeChatStream handles streaming chat requests with NDJSON response.
func (s *Server) handleClaudeChatStream(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	if s.claude == nil {
		writeBadRequest(w, "Claude service not configured")
		return
	}

	var req claude.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Message == "" {
		writeBadRequest(w, "Message is required")
		return
	}

	// Set up streaming response
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeBadRequest(w, "Streaming not supported")
		return
	}

	events, err := s.claude.ChatStream(person, req)
	if err != nil {
		writeStreamError(w, err.Error())
		flusher.Flush()
		return
	}

	for event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		w.Write(data)
		w.Write([]byte("\n"))
		flusher.Flush()
	}
}

// ClearSessionRequest represents a request to clear a chat session.
type ClearSessionRequest struct {
	SessionID string `json:"session_id"`
}

// handleClaudeClear clears a chat session.
func (s *Server) handleClaudeClear(w http.ResponseWriter, r *http.Request) {
	if s.claude == nil {
		writeBadRequest(w, "Claude service not configured")
		return
	}

	var req ClearSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.SessionID == "" {
		writeBadRequest(w, "Session ID is required")
		return
	}

	s.claude.Sessions().Clear(req.SessionID)
	writeSuccess(w, "Session cleared")
}

// handleClaudeHistory returns the message history for a session.
func (s *Server) handleClaudeHistory(w http.ResponseWriter, r *http.Request) {
	if s.claude == nil {
		writeBadRequest(w, "Claude service not configured")
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeBadRequest(w, "Session ID is required")
		return
	}

	history := s.claude.Sessions().GetHistory(sessionID)
	if history == nil {
		history = []claude.ChatMessage{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages": history,
	})
}
