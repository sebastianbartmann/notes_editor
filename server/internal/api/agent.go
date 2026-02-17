package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"notes-editor/internal/agent"
	"notes-editor/internal/claude"
)

// StopRunRequest is the request body for stopping an active agent run.
type StopRunRequest struct {
	RunID string `json:"run_id"`
}

// AgentActionRunRequest controls action execution behavior.
type AgentActionRunRequest struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Confirm   bool   `json:"confirm,omitempty"`
}

type AgentToolExecuteRequest struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

type AgentToolExecuteResponse struct {
	OK      bool   `json:"ok"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

type AgentExportSessionsResponse struct {
	Success   bool     `json:"success"`
	Message   string   `json:"message"`
	Directory string   `json:"directory"`
	Files     []string `json:"files"`
}

// handleAgentChat handles non-streaming agent chat requests.
func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	var req agent.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}
	if req.Message == "" && req.ActionID == "" {
		writeBadRequest(w, "Message or action_id is required")
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

	writeJSON(w, http.StatusOK, resp)
}

// handleAgentChatStream handles streaming chat requests with NDJSON v2 events.
func (s *Server) handleAgentChatStream(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	var req agent.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}
	if req.Message == "" && req.ActionID == "" {
		writeBadRequest(w, "Message or action_id is required")
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
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		if _, err := w.Write(data); err != nil {
			go func() {
				for range run.Events {
				}
			}()
			return
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			go func() {
				for range run.Events {
				}
			}()
			return
		}
		flusher.Flush()
	}
}

// handleAgentSessionClear clears a chat session.
func (s *Server) handleAgentSessionClear(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
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

	if err := agentSvc.ClearSession(person, req.SessionID); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	writeSuccess(w, "Session cleared")
}

// handleAgentSessionHistory returns session history.
func (s *Server) handleAgentSessionHistory(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeBadRequest(w, "Session ID is required")
		return
	}

	items, err := agentSvc.GetConversationHistory(person, sessionID)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	messages := make([]claude.ChatMessage, 0, len(items))
	for _, item := range items {
		if item.Type != agent.ConversationItemMessage {
			continue
		}
		if item.Role != "user" && item.Role != "assistant" {
			continue
		}
		messages = append(messages, claude.ChatMessage{Role: item.Role, Content: item.Content})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":    items,
		"messages": messages,
	})
}

// handleAgentSessionsList returns person-scoped session summaries.
func (s *Server) handleAgentSessionsList(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	sessions, err := agentSvc.ListSessions(person)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessions,
	})
}

// handleAgentActiveRunsList returns person-scoped currently running agent streams.
func (s *Server) handleAgentActiveRunsList(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"runs": agentSvc.ListActiveRuns(person),
	})
}

// handleAgentSessionsClearAll clears all person-scoped session state.
func (s *Server) handleAgentSessionsClearAll(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	if err := agentSvc.ClearAllSessions(person); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	writeSuccess(w, "All sessions cleared")
}

// handleAgentSessionsExportMarkdown exports all person sessions to markdown files.
func (s *Server) handleAgentSessionsExportMarkdown(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	exported, err := agentSvc.ExportSessionsMarkdown(person)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	s.syncMgr.TriggerPush("Export agent sessions markdown")

	writeJSON(w, http.StatusOK, AgentExportSessionsResponse{
		Success:   true,
		Message:   "Exported sessions to markdown",
		Directory: exported.Directory,
		Files:     exported.Files,
	})
}

// handleAgentStopRun stops an active streaming run.
func (s *Server) handleAgentStopRun(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	var req StopRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}
	if req.RunID == "" {
		writeBadRequest(w, "Run ID is required")
		return
	}

	if !agentSvc.StopRun(person, req.RunID) {
		writeNotFound(w, "Run not found")
		return
	}

	writeSuccess(w, "Run stopped")
}

// handleAgentConfigGet returns per-person agent config including prompt content.
func (s *Server) handleAgentConfigGet(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}
	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	cfg, err := agentSvc.GetConfig(person)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handleAgentConfigSave updates per-person agent config.
func (s *Server) handleAgentConfigSave(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}
	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	var req agent.ConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	cfg, err := agentSvc.SaveConfig(person, req)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handleAgentActionsList returns file-backed prompt actions for the active person.
func (s *Server) handleAgentActionsList(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}
	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	actions, err := agentSvc.ListActions(person)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"actions": actions,
	})
}

// handleAgentToolExecute executes a canonical tool call in a person-scoped context.
// Intended for the local gateway sidecar to delegate tool execution back to the Go server.
func (s *Server) handleAgentToolExecute(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req AgentToolExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}
	if req.Tool == "" {
		writeBadRequest(w, "tool is required")
		return
	}
	if req.Args == nil {
		req.Args = map[string]any{}
	}

	toolExec := claude.NewToolExecutor(s.store, s.getLinkedIn(), person)
	content, err := toolExec.ExecuteTool(req.Tool, req.Args)
	if err != nil {
		writeJSON(w, http.StatusOK, AgentToolExecuteResponse{
			OK:    false,
			Error: err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, AgentToolExecuteResponse{
		OK:      true,
		Content: content,
	})
}

// handleAgentActionRun executes one action by ID as a chat request.
func (s *Server) handleAgentActionRun(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}
	agentSvc := s.getAgent()
	if agentSvc == nil {
		writeBadRequest(w, "Agent service not configured")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeBadRequest(w, "Action ID is required")
		return
	}

	var req AgentActionRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			req = AgentActionRunRequest{}
		} else {
			writeBadRequest(w, "Invalid request body")
			return
		}
	}

	resp, err := agentSvc.Chat(person, agent.ChatRequest{
		SessionID: req.SessionID,
		ActionID:  id,
		Message:   req.Message,
		Confirm:   req.Confirm,
	})
	if err != nil {
		if errors.Is(err, agent.ErrSessionBusy) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeBadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleAgentGatewayHealth reports gateway runtime health.
func (s *Server) handleAgentGatewayHealth(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePerson(w, r); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()

	health := s.gatewayHealth(ctx)
	writeJSON(w, http.StatusOK, health)
}
