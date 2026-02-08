package agent

import (
	"context"
	"errors"
	"fmt"

	"notes-editor/internal/claude"
)

const (
	// RuntimeModeAnthropicAPIKey uses the local Anthropic API-key runtime.
	RuntimeModeAnthropicAPIKey = "anthropic_api_key"
	// RuntimeModeGatewaySubscription uses the gateway/subscription runtime path.
	RuntimeModeGatewaySubscription = "gateway_subscription"
)

// RuntimeChatRequest is the normalized chat request sent to runtimes.
type RuntimeChatRequest struct {
	SessionID string
	Message   string
	// MaxToolCalls is an optional runtime hint (0 means runtime default).
	MaxToolCalls int
}

// RuntimeChatResponse is the normalized non-stream chat response from runtimes.
type RuntimeChatResponse struct {
	Response  string
	SessionID string
}

// RuntimeStream is the normalized streaming response from runtimes.
type RuntimeStream struct {
	Events <-chan StreamEvent
}

// Runtime is the provider-agnostic contract for agent execution backends.
type Runtime interface {
	Mode() string
	Available() bool
	Chat(person string, req RuntimeChatRequest) (*RuntimeChatResponse, error)
	ChatStream(ctx context.Context, person string, req RuntimeChatRequest) (*RuntimeStream, error)
	ClearSession(sessionID string) error
	GetHistory(sessionID string) ([]claude.ChatMessage, error)
}

// RuntimeUnavailableError indicates a runtime backend cannot currently execute.
type RuntimeUnavailableError struct {
	Mode   string
	Reason string
}

func (e *RuntimeUnavailableError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("runtime %q unavailable", e.Mode)
	}
	return fmt.Sprintf("runtime %q unavailable: %s", e.Mode, e.Reason)
}

// IsRuntimeUnavailable returns whether err indicates an unavailable runtime.
func IsRuntimeUnavailable(err error) bool {
	var target *RuntimeUnavailableError
	return errors.As(err, &target)
}
