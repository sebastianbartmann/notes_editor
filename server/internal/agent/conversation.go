package agent

import (
	"strings"
	"time"

	"notes-editor/internal/claude"
)

const (
	ConversationItemMessage    = "message"
	ConversationItemToolCall   = "tool_call"
	ConversationItemToolResult = "tool_result"
	ConversationItemStatus     = "status"
	ConversationItemError      = "error"
	ConversationItemUsage      = "usage"
)

// ConversationItem is the unified persisted chat item schema used by agent history.
type ConversationItem struct {
	Type      string         `json:"type"`
	Role      string         `json:"role,omitempty"`
	Content   string         `json:"content,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	RunID     string         `json:"run_id,omitempty"`
	Seq       int            `json:"seq,omitempty"`
	TS        time.Time      `json:"ts,omitempty"`
	Tool      string         `json:"tool,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	OK        bool           `json:"ok,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Message   string         `json:"message,omitempty"`
	Usage     *UsageSnapshot `json:"usage,omitempty"`
}

func chatMessagesToItems(messages []claude.ChatMessage) []ConversationItem {
	out := make([]ConversationItem, 0, len(messages))
	for _, msg := range messages {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		role := strings.TrimSpace(msg.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		out = append(out, ConversationItem{
			Type:    ConversationItemMessage,
			Role:    role,
			Content: msg.Content,
		})
	}
	return out
}

func historyPreviewItems(items []ConversationItem) string {
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.Type != ConversationItemMessage || item.Role != "assistant" {
			continue
		}
		content := normalizeSessionText(item.Content)
		if content == "" {
			continue
		}
		if len(content) <= maxSessionPreviewLen {
			return content
		}
		trimmed := content[:maxSessionPreviewLen]
		lastSpace := strings.LastIndex(trimmed, " ")
		if lastSpace > maxSessionPreviewLen/2 {
			trimmed = trimmed[:lastSpace]
		}
		return strings.TrimSpace(trimmed) + "…"
	}
	return ""
}
