package agent

import "time"

// UsageSnapshot represents token/context usage metadata for the current turn.
type UsageSnapshot struct {
	InputTokens      int `json:"input_tokens,omitempty"`
	OutputTokens     int `json:"output_tokens,omitempty"`
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
	ContextWindow    int `json:"context_window,omitempty"`
	RemainingTokens  int `json:"remaining_tokens,omitempty"`
}

// StreamEvent is the canonical v2 NDJSON event schema for agent streaming.
type StreamEvent struct {
	Type      string         `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	RunID     string         `json:"run_id,omitempty"`
	Seq       int            `json:"seq,omitempty"`
	TS        time.Time      `json:"ts,omitempty"`
	Delta     string         `json:"delta,omitempty"`
	Tool      string         `json:"tool,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	OK        bool           `json:"ok,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Message   string         `json:"message,omitempty"`
	Usage     *UsageSnapshot `json:"usage,omitempty"`
}
