package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"notes-editor/internal/textnorm"
)

// StreamEvent represents an event in the NDJSON stream.
type StreamEvent struct {
	Type      string `json:"type"`
	Delta     string `json:"delta,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Usage     *Usage `json:"usage,omitempty"`
}

// Usage reports token and context-window usage for the active assistant turn.
type Usage struct {
	InputTokens      int `json:"input_tokens,omitempty"`
	OutputTokens     int `json:"output_tokens,omitempty"`
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
	ContextWindow    int `json:"context_window,omitempty"`
	RemainingTokens  int `json:"remaining_tokens,omitempty"`
}

// ChatStream performs a streaming chat request.
// Returns a channel that receives stream events.
func (s *Service) ChatStream(person string, req ChatRequest) (<-chan StreamEvent, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("Claude API key not configured")
	}

	systemPrompt := s.loadSystemPrompt(person)

	session := s.sessions.GetOrCreate(req.SessionID, person)
	session.AddMessage("user", req.Message)

	// Build the message history
	messages := buildAnthropicMessages(session.GetMessages())

	// Create tool executor
	toolExec := NewToolExecutor(s.store, s.linkedin, person)

	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)
		s.streamWithToolLoop(messages, toolExec, session, events, systemPrompt)
	}()

	return events, nil
}

// streamWithToolLoop handles streaming with tool use in a loop.
func (s *Service) streamWithToolLoop(messages []anthropicMsg, toolExec *ToolExecutor, session *Session, events chan<- StreamEvent, systemPrompt string) {
	// Set up ping timer
	pingTicker := time.NewTicker(5 * time.Second)
	defer pingTicker.Stop()

	// Create ping channel
	pingDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-pingTicker.C:
				select {
				case events <- StreamEvent{Type: "ping"}:
				default:
				}
			case <-pingDone:
				return
			}
		}
	}()
	defer close(pingDone)

	var fullResponse strings.Builder

	for {
		resp, textDelta, err := s.streamAPI(messages, events, systemPrompt)
		if err != nil {
			events <- StreamEvent{Type: "error", Message: err.Error()}
			return
		}

		fullResponse.WriteString(textDelta)

		// Check if we need to handle tool use
		if resp.StopReason == "tool_use" {
			// Find tool use blocks and execute them
			var toolResults []any
			for _, block := range resp.Content {
				if block.Type == "tool_use" {
					// Send tool_use event
					events <- StreamEvent{
						Type:  "tool_use",
						Name:  block.Name,
						Input: block.Input,
					}

					input, ok := block.Input.(map[string]any)
					if !ok {
						input = make(map[string]any)
					}

					result, err := toolExec.ExecuteTool(block.Name, input)
					if err != nil {
						result = fmt.Sprintf("Error: %s", err.Error())
					}

					// Send status event
					events <- StreamEvent{
						Type:    "status",
						Message: fmt.Sprintf("Tool %s executed", block.Name),
					}

					toolResults = append(toolResults, map[string]any{
						"type":        "tool_result",
						"tool_use_id": block.ID,
						"content":     result,
					})
				}
			}

			// Add assistant response and tool results to messages
			messages = append(messages, anthropicMsg{
				Role:    "assistant",
				Content: resp.Content,
			})
			messages = append(messages, anthropicMsg{
				Role:    "user",
				Content: toolResults,
			})
			continue
		}

		// Done - save response and send done event
		session.AddMessage("assistant", fullResponse.String())
		events <- StreamEvent{
			Type:      "done",
			SessionID: session.ID,
		}
		return
	}
}

// streamAPI makes a streaming call to the Anthropic API.
func (s *Service) streamAPI(messages []anthropicMsg, events chan<- StreamEvent, systemPrompt string) (*anthropicResponse, string, error) {
	reqBody := struct {
		Model     string           `json:"model"`
		MaxTokens int              `json:"max_tokens"`
		System    string           `json:"system"`
		Messages  []anthropicMsg   `json:"messages"`
		Tools     []map[string]any `json:"tools,omitempty"`
		Stream    bool             `json:"stream"`
	}{
		Model:     s.resolvedModel(),
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  messages,
		Tools:     ToolDefinitions,
		Stream:    true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", anthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return s.processStream(resp.Body, events)
}

// processStream processes the SSE stream from the Anthropic API.
func (s *Service) processStream(body io.Reader, events chan<- StreamEvent) (*anthropicResponse, string, error) {
	scanner := bufio.NewScanner(body)
	var textBuilder strings.Builder
	var finalResponse anthropicResponse
	var trimmer textnorm.LeadingBlankLineTrimmer

	// Track content blocks for tool use
	var contentBlocks []contentBlock
	var currentToolInput strings.Builder
	var currentToolID string
	var currentToolName string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE event
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "content_block_start":
			block, ok := event["content_block"].(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := block["type"].(string)
			if blockType == "tool_use" {
				currentToolID, _ = block["id"].(string)
				currentToolName, _ = block["name"].(string)
				currentToolInput.Reset()
			}

		case "content_block_delta":
			delta, ok := event["delta"].(map[string]any)
			if !ok {
				continue
			}
			deltaType, _ := delta["type"].(string)

			switch deltaType {
			case "text_delta":
				text, _ := delta["text"].(string)
				normalized := trimmer.Push(text)
				if normalized == "" {
					continue
				}
				textBuilder.WriteString(normalized)
				events <- StreamEvent{Type: "text", Delta: normalized}

			case "input_json_delta":
				partial, _ := delta["partial_json"].(string)
				currentToolInput.WriteString(partial)
			}

		case "content_block_stop":
			// If we were building a tool use block, save it
			if currentToolID != "" {
				var input any
				if currentToolInput.Len() > 0 {
					_ = json.Unmarshal([]byte(currentToolInput.String()), &input)
				}
				contentBlocks = append(contentBlocks, contentBlock{
					Type:  "tool_use",
					ID:    currentToolID,
					Name:  currentToolName,
					Input: input,
				})
				currentToolID = ""
				currentToolName = ""
			}

		case "message_delta":
			delta, ok := event["delta"].(map[string]any)
			if ok {
				if stopReason, ok := delta["stop_reason"].(string); ok {
					finalResponse.StopReason = stopReason
				}
			}
			if usageRaw, ok := event["usage"].(map[string]any); ok {
				if usage := parseAnthropicUsage(usageRaw); usage != nil {
					events <- StreamEvent{Type: "usage", Usage: usage}
				}
			}

		case "message_start":
			msg, ok := event["message"].(map[string]any)
			if !ok {
				continue
			}
			usageRaw, ok := msg["usage"].(map[string]any)
			if !ok {
				continue
			}
			if usage := parseAnthropicUsage(usageRaw); usage != nil {
				events <- StreamEvent{Type: "usage", Usage: usage}
			}

		case "message_stop":
			// Message complete
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("stream read error: %w", err)
	}

	// Add text content if any
	if textBuilder.Len() > 0 {
		contentBlocks = append([]contentBlock{{
			Type: "text",
			Text: textBuilder.String(),
		}}, contentBlocks...)
	}

	finalResponse.Content = contentBlocks
	return &finalResponse, textBuilder.String(), nil
}

func parseAnthropicUsage(raw map[string]any) *Usage {
	if len(raw) == 0 {
		return nil
	}
	in := int(readNumber(raw, "input_tokens"))
	out := int(readNumber(raw, "output_tokens"))
	cacheRead := int(readNumber(raw, "cache_read_input_tokens"))
	cacheWrite := int(readNumber(raw, "cache_creation_input_tokens"))
	total := in + out + cacheRead + cacheWrite

	// The configured model in this app uses a 200k context window.
	const contextWindow = 200000
	remaining := contextWindow - total
	if remaining < 0 {
		remaining = 0
	}
	if total == 0 && in == 0 && out == 0 && cacheRead == 0 && cacheWrite == 0 {
		return nil
	}
	return &Usage{
		InputTokens:      in,
		OutputTokens:     out,
		CacheReadTokens:  cacheRead,
		CacheWriteTokens: cacheWrite,
		TotalTokens:      total,
		ContextWindow:    contextWindow,
		RemainingTokens:  remaining,
	}
}

func readNumber(raw map[string]any, key string) float64 {
	v, ok := raw[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return 0
		}
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
