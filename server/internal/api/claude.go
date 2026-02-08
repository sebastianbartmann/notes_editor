package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"notes-editor/internal/agent"
)

// handleClaudeChat handles non-streaming chat requests.
func (s *Server) handleClaudeChat(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Claude service not configured")
		return
	}

	var req agent.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Message == "" {
		writeBadRequest(w, "Message is required")
		return
	}

	resp, err := agentSvc.Chat(person, req)
	if err != nil {
		if errors.Is(err, agent.ErrSessionBusy) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"response":   resp.Response,
		"session_id": resp.SessionID,
	})
}

// handleClaudeChatStream handles streaming chat requests with NDJSON response.
func (s *Server) handleClaudeChatStream(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Claude service not configured")
		return
	}

	var req agent.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Message == "" {
		writeBadRequest(w, "Message is required")
		return
	}

	run, err := agentSvc.ChatStream(r.Context(), person, req)
	if err != nil {
		if errors.Is(err, agent.ErrSessionBusy) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeBadRequest(w, err.Error())
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

	for event := range run.Events {
		data, err := json.Marshal(mapToLegacyStreamEvent(event))
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
	agentSvc := s.getAgent()
	if agentSvc == nil {
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

	if err := agentSvc.ClearSession(req.SessionID); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	writeSuccess(w, "Session cleared")
}

// handleClaudeHistory returns the message history for a session.
func (s *Server) handleClaudeHistory(w http.ResponseWriter, r *http.Request) {
	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Claude service not configured")
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeBadRequest(w, "Session ID is required")
		return
	}

	history, err := agentSvc.GetHistory(sessionID)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages": history,
	})
}

func mapToLegacyStreamEvent(event agent.StreamEvent) map[string]any {
	switch event.Type {
	case "start":
		return map[string]any{
			"type":       "session",
			"session_id": event.SessionID,
		}
	case "tool_call":
		return map[string]any{
			"type":  "tool_use",
			"name":  event.Tool,
			"input": event.Args,
		}
	case "tool_result":
		msg := event.Summary
		if msg == "" {
			msg = "Tool executed"
		}
		return map[string]any{
			"type":    "status",
			"message": msg,
		}
	default:
		out := map[string]any{
			"type": event.Type,
		}
		if event.Delta != "" {
			out["delta"] = event.Delta
		}
		if event.SessionID != "" {
			out["session_id"] = event.SessionID
		}
		if event.Message != "" {
			out["message"] = event.Message
		}
		return out
	}
}
