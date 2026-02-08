package agent

import (
	"context"

	"notes-editor/internal/claude"
)

// AnthropicRuntime adapts the existing Claude service into the Runtime contract.
type AnthropicRuntime struct {
	claude *claude.Service
}

// NewAnthropicRuntime creates an Anthropic API key runtime adapter.
func NewAnthropicRuntime(claudeSvc *claude.Service) *AnthropicRuntime {
	return &AnthropicRuntime{claude: claudeSvc}
}

// Mode returns the runtime mode key.
func (r *AnthropicRuntime) Mode() string {
	return RuntimeModeAnthropicAPIKey
}

// Available returns true when Claude service is configured.
func (r *AnthropicRuntime) Available() bool {
	return r != nil && r.claude != nil
}

// Chat executes a non-streaming request.
func (r *AnthropicRuntime) Chat(person string, req RuntimeChatRequest) (*RuntimeChatResponse, error) {
	if !r.Available() {
		return nil, &RuntimeUnavailableError{
			Mode:   RuntimeModeAnthropicAPIKey,
			Reason: "Claude service not configured",
		}
	}

	resp, err := r.claude.Chat(person, claude.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
	})
	if err != nil {
		return nil, err
	}

	return &RuntimeChatResponse{
		Response:  resp.Response,
		SessionID: resp.SessionID,
	}, nil
}

// ChatStream executes a streaming request.
func (r *AnthropicRuntime) ChatStream(_ context.Context, person string, req RuntimeChatRequest) (*RuntimeStream, error) {
	if !r.Available() {
		return nil, &RuntimeUnavailableError{
			Mode:   RuntimeModeAnthropicAPIKey,
			Reason: "Claude service not configured",
		}
	}

	upstream, err := r.claude.ChatStream(person, claude.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
	})
	if err != nil {
		return nil, err
	}

	out := make(chan StreamEvent, 100)
	go func() {
		defer close(out)
		for event := range upstream {
			mapped := mapClaudeEvent(event)
			for _, e := range mapped {
				out <- e
			}
		}
	}()

	return &RuntimeStream{Events: out}, nil
}

// ClearSession removes a session from Anthropic runtime state.
func (r *AnthropicRuntime) ClearSession(sessionID string) error {
	if !r.Available() {
		return &RuntimeUnavailableError{
			Mode:   RuntimeModeAnthropicAPIKey,
			Reason: "Claude service not configured",
		}
	}
	r.claude.Sessions().Clear(sessionID)
	return nil
}

// GetHistory returns session history from Anthropic runtime state.
func (r *AnthropicRuntime) GetHistory(sessionID string) ([]claude.ChatMessage, error) {
	if !r.Available() {
		return nil, &RuntimeUnavailableError{
			Mode:   RuntimeModeAnthropicAPIKey,
			Reason: "Claude service not configured",
		}
	}
	history := r.claude.Sessions().GetHistory(sessionID)
	if history == nil {
		return []claude.ChatMessage{}, nil
	}
	return history, nil
}

var _ Runtime = (*AnthropicRuntime)(nil)
