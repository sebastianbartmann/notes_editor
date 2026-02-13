package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"notes-editor/internal/claude"
	"notes-editor/internal/linkedin"
	"notes-editor/internal/textnorm"
	"notes-editor/internal/vault"
)

type piGatewayStreamEvent struct {
	Type      string         `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	RunID     string         `json:"run_id,omitempty"`
	Delta     string         `json:"delta,omitempty"`
	Tool      string         `json:"tool,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	OK        bool           `json:"ok,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Message   string         `json:"message,omitempty"`
}

const piRuntimeSessionMapPath = ".notes-editor/runtime-session-map.json"

// PiGatewayRuntime bridges the Go server to the local Pi sidecar.
type PiGatewayRuntime struct {
	baseURL string
	client  *http.Client
	store   *vault.Store

	sessions                   *claude.SessionStore
	mu                         sync.Mutex
	runtimeSessionByAppSession map[string]string
	sessionMapLoaded           bool
}

// NewPiGatewayRuntime creates a Pi gateway runtime.
func NewPiGatewayRuntime(baseURL string) *PiGatewayRuntime {
	return &PiGatewayRuntime{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			// Streaming requests must not have a global client timeout.
			Timeout: 0,
		},
		sessions:                   claude.NewSessionStore(),
		runtimeSessionByAppSession: make(map[string]string),
	}
}

// WithDependencies injects vault dependencies for prompt/historical context.
func (r *PiGatewayRuntime) WithDependencies(store *vault.Store, _ *linkedin.Service) *PiGatewayRuntime {
	r.store = store
	r.ensureSessionMapLoaded()
	return r
}

// Mode returns the runtime mode key.
func (r *PiGatewayRuntime) Mode() string {
	return RuntimeModeGatewaySubscription
}

// Available reports current availability.
func (r *PiGatewayRuntime) Available() bool {
	return r != nil && r.baseURL != ""
}

// Chat executes a non-streaming request.
func (r *PiGatewayRuntime) Chat(ctxPerson string, req RuntimeChatRequest) (*RuntimeChatResponse, error) {
	stream, err := r.ChatStream(context.Background(), ctxPerson, req)
	if err != nil {
		return nil, err
	}

	var response strings.Builder
	sessionID := req.SessionID
	for event := range stream.Events {
		switch event.Type {
		case "text":
			response.WriteString(event.Delta)
		case "done":
			if event.SessionID != "" {
				sessionID = event.SessionID
			}
		case "error":
			return nil, fmt.Errorf(event.Message)
		}
	}

	return &RuntimeChatResponse{
		Response:  response.String(),
		SessionID: sessionID,
	}, nil
}

// ChatStream executes a streaming request.
func (r *PiGatewayRuntime) ChatStream(ctx context.Context, person string, req RuntimeChatRequest) (*RuntimeStream, error) {
	if !r.Available() {
		return nil, &RuntimeUnavailableError{
			Mode:   RuntimeModeGatewaySubscription,
			Reason: "Gateway URL not configured",
		}
	}

	appSession := r.sessions.GetOrCreate(req.SessionID, person)
	appSession.AddMessage("user", req.Message)

	runtimeSessionID := r.getRuntimeSessionID(person, appSession.ID)

	systemPrompt := claude.SystemPrompt
	if r.store != nil {
		// Prefer person-scoped agent prompt under agent/ folder; fall back to legacy root prompt.
		if prompt, err := r.store.ReadFile(person, "agent/agents.md"); err == nil {
			if strings.TrimSpace(prompt) != "" {
				systemPrompt = prompt
			}
		} else if os.IsNotExist(err) {
			if prompt, err := r.store.ReadFile(person, "agents.md"); err == nil {
				if strings.TrimSpace(prompt) != "" {
					systemPrompt = prompt
				}
			} else if !os.IsNotExist(err) {
				// Non-fatal: fall back to default system prompt.
			}
		} else {
			// Non-fatal: fall back to default system prompt.
		}

		systemPrompt += claude.BuildAvailableSkillsPromptAddon(r.store, person)
	}

	payload := map[string]any{
		"person":        person,
		"session_id":    runtimeSessionID,
		"message":       req.Message,
		"system_prompt": systemPrompt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/v1/chat-stream", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, mapPiTransportError(err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, mapPiStatusError(resp.StatusCode, string(responseBody))
	}

	out := make(chan StreamEvent, 100)

	go func() {
		defer close(out)
		defer resp.Body.Close()

		var fullResponse strings.Builder
		finalSessionID := appSession.ID
		sawDone := false
		var trimmer textnorm.LeadingBlankLineTrimmer

		scanner := bufio.NewScanner(resp.Body)
		buffer := make([]byte, 0, 64*1024)
		scanner.Buffer(buffer, 2*1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var event piGatewayStreamEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			switch event.Type {
			case "start":
				if event.SessionID != "" {
					r.setRuntimeSessionID(person, appSession.ID, event.SessionID)
				}
			case "text":
				normalized := trimmer.Push(event.Delta)
				if normalized == "" {
					continue
				}
				fullResponse.WriteString(normalized)
				out <- StreamEvent{
					Type:  "text",
					Delta: normalized,
				}
			case "tool_call":
				out <- StreamEvent{Type: "tool_call", Tool: event.Tool, Args: event.Args}
			case "status":
				out <- StreamEvent{Type: "status", Message: event.Message}
			case "tool_result":
				// The gateway now executes tools by delegating back to the Go server, so
				// tool results are streamed directly from the sidecar.
				out <- StreamEvent{Type: "tool_result", Tool: event.Tool, OK: event.OK, Summary: event.Summary}
			case "error":
				out <- StreamEvent{Type: "error", Message: event.Message}
			case "done":
				sawDone = true
				out <- StreamEvent{Type: "done", SessionID: finalSessionID}
			}
		}

		if scanner.Err() != nil {
			out <- StreamEvent{
				Type:    "error",
				Message: scanner.Err().Error(),
			}
		}

		appSession.AddMessage("assistant", fullResponse.String())
		if !sawDone {
			out <- StreamEvent{
				Type:      "done",
				SessionID: finalSessionID,
			}
		}
	}()

	return &RuntimeStream{
		Events: out,
	}, nil
}

// ClearSession removes runtime session state.
func (r *PiGatewayRuntime) ClearSession(sessionID string) error {
	if !r.Available() {
		return &RuntimeUnavailableError{
			Mode:   RuntimeModeGatewaySubscription,
			Reason: "Gateway URL not configured",
		}
	}
	r.ensureSessionMapLoaded()
	r.sessions.Clear(sessionID)
	changed := false
	var snapshot map[string]string
	r.mu.Lock()
	for key := range r.runtimeSessionByAppSession {
		if strings.HasSuffix(key, "::"+sessionID) {
			delete(r.runtimeSessionByAppSession, key)
			changed = true
		}
	}
	if changed {
		snapshot = cloneSessionMap(r.runtimeSessionByAppSession)
	}
	r.mu.Unlock()
	if changed {
		r.persistSessionMap(snapshot)
	}
	return nil
}

// GetHistory returns runtime session history.
func (r *PiGatewayRuntime) GetHistory(sessionID string) ([]claude.ChatMessage, error) {
	if !r.Available() {
		return nil, &RuntimeUnavailableError{
			Mode:   RuntimeModeGatewaySubscription,
			Reason: "Gateway URL not configured",
		}
	}
	history := r.sessions.GetHistory(sessionID)
	if history == nil {
		return []claude.ChatMessage{}, nil
	}
	return history, nil
}

func (r *PiGatewayRuntime) getRuntimeSessionID(person, appSessionID string) string {
	r.ensureSessionMapLoaded()
	key := sessionRunKey(person, appSessionID)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runtimeSessionByAppSession[key]
}

func (r *PiGatewayRuntime) setRuntimeSessionID(person, appSessionID, runtimeSessionID string) {
	r.ensureSessionMapLoaded()
	key := sessionRunKey(person, appSessionID)
	var snapshot map[string]string
	r.mu.Lock()
	if existing := r.runtimeSessionByAppSession[key]; existing == runtimeSessionID {
		r.mu.Unlock()
		return
	}
	r.runtimeSessionByAppSession[key] = runtimeSessionID
	snapshot = cloneSessionMap(r.runtimeSessionByAppSession)
	r.mu.Unlock()
	r.persistSessionMap(snapshot)
}

func (r *PiGatewayRuntime) ensureSessionMapLoaded() {
	r.mu.Lock()
	if r.sessionMapLoaded || r.store == nil {
		r.mu.Unlock()
		return
	}
	r.sessionMapLoaded = true
	store := r.store
	r.mu.Unlock()

	raw, err := store.ReadRootFile(piRuntimeSessionMapPath)
	if err != nil {
		return
	}

	var payload struct {
		Mappings map[string]string `json:"mappings"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return
	}

	r.mu.Lock()
	for key, value := range payload.Mappings {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if k == "" || v == "" {
			continue
		}
		if _, exists := r.runtimeSessionByAppSession[k]; !exists {
			r.runtimeSessionByAppSession[k] = v
		}
	}
	r.mu.Unlock()
}

func (r *PiGatewayRuntime) persistSessionMap(snapshot map[string]string) {
	if r.store == nil {
		return
	}
	body, err := json.Marshal(struct {
		Mappings map[string]string `json:"mappings"`
	}{
		Mappings: snapshot,
	})
	if err != nil {
		return
	}
	_ = r.store.WriteRootFile(piRuntimeSessionMapPath, string(body))
}

func cloneSessionMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func mapPiTransportError(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return &RuntimeUnavailableError{
			Mode:   RuntimeModeGatewaySubscription,
			Reason: "Pi gateway network error: " + netErr.Error(),
		}
	}
	return &RuntimeUnavailableError{
		Mode:   RuntimeModeGatewaySubscription,
		Reason: "Pi gateway request failed: " + err.Error(),
	}
}

func mapPiStatusError(status int, body string) error {
	body = strings.TrimSpace(body)
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &RuntimeUnavailableError{
			Mode:   RuntimeModeGatewaySubscription,
			Reason: fmt.Sprintf("Pi gateway auth error (%d): %s", status, body),
		}
	default:
		return &RuntimeUnavailableError{
			Mode:   RuntimeModeGatewaySubscription,
			Reason: fmt.Sprintf("Pi gateway error (%d): %s", status, body),
		}
	}
}

var _ Runtime = (*PiGatewayRuntime)(nil)
