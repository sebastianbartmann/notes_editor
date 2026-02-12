package claude

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestStreamEvent_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		event    StreamEvent
		expected string
	}{
		{
			name:     "text event",
			event:    StreamEvent{Type: "text", Delta: "Hello"},
			expected: `{"type":"text","delta":"Hello"}`,
		},
		{
			name:     "ping event",
			event:    StreamEvent{Type: "ping"},
			expected: `{"type":"ping"}`,
		},
		{
			name:     "done event with session_id",
			event:    StreamEvent{Type: "done", SessionID: "abc-123"},
			expected: `{"type":"done","session_id":"abc-123"}`,
		},
		{
			name:     "error event",
			event:    StreamEvent{Type: "error", Message: "Something went wrong"},
			expected: `{"type":"error","message":"Something went wrong"}`,
		},
		{
			name:     "tool_use event",
			event:    StreamEvent{Type: "tool_use", Name: "read_file", Input: map[string]any{"path": "test.txt"}},
			expected: `{"type":"tool_use","name":"read_file","input":{"path":"test.txt"}}`,
		},
		{
			name:     "status event",
			event:    StreamEvent{Type: "status", Message: "Tool read_file executed"},
			expected: `{"type":"status","message":"Tool read_file executed"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Compare as JSON objects to ignore field ordering
			var got, want map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &want); err != nil {
				t.Fatalf("Failed to unmarshal expected: %v", err)
			}

			// Check type field
			if got["type"] != want["type"] {
				t.Errorf("type mismatch: got %v, want %v", got["type"], want["type"])
			}

			// Check other non-nil fields by marshaling to JSON for comparison
			for k, v := range want {
				gotVal, _ := json.Marshal(got[k])
				wantVal, _ := json.Marshal(v)
				if string(gotVal) != string(wantVal) {
					t.Errorf("field %q mismatch: got %s, want %s", k, gotVal, wantVal)
				}
			}
		})
	}
}

func TestStreamEvent_OmitsEmptyFields(t *testing.T) {
	// Ping event should not include empty fields
	event := StreamEvent{Type: "ping"}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	str := string(data)
	if strings.Contains(str, "delta") {
		t.Errorf("ping event should not contain delta field: %s", str)
	}
	if strings.Contains(str, "name") {
		t.Errorf("ping event should not contain name field: %s", str)
	}
	if strings.Contains(str, "session_id") {
		t.Errorf("ping event should not contain session_id field: %s", str)
	}
	if strings.Contains(str, "message") {
		t.Errorf("ping event should not contain message field: %s", str)
	}
	if strings.Contains(str, "input") {
		t.Errorf("ping event should not contain input field: %s", str)
	}
}

func TestStreamEvent_JSONDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected StreamEvent
	}{
		{
			name:     "text event",
			json:     `{"type":"text","delta":"Hello world"}`,
			expected: StreamEvent{Type: "text", Delta: "Hello world"},
		},
		{
			name:     "ping event",
			json:     `{"type":"ping"}`,
			expected: StreamEvent{Type: "ping"},
		},
		{
			name:     "done event",
			json:     `{"type":"done","session_id":"session-xyz"}`,
			expected: StreamEvent{Type: "done", SessionID: "session-xyz"},
		},
		{
			name:     "error event",
			json:     `{"type":"error","message":"API rate limited"}`,
			expected: StreamEvent{Type: "error", Message: "API rate limited"},
		},
		{
			name:     "tool_use event",
			json:     `{"type":"tool_use","name":"web_search","input":{"query":"golang testing"}}`,
			expected: StreamEvent{Type: "tool_use", Name: "web_search", Input: map[string]any{"query": "golang testing"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event StreamEvent
			if err := json.Unmarshal([]byte(tt.json), &event); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if event.Type != tt.expected.Type {
				t.Errorf("Type: got %q, want %q", event.Type, tt.expected.Type)
			}
			if event.Delta != tt.expected.Delta {
				t.Errorf("Delta: got %q, want %q", event.Delta, tt.expected.Delta)
			}
			if event.SessionID != tt.expected.SessionID {
				t.Errorf("SessionID: got %q, want %q", event.SessionID, tt.expected.SessionID)
			}
			if event.Message != tt.expected.Message {
				t.Errorf("Message: got %q, want %q", event.Message, tt.expected.Message)
			}
			if event.Name != tt.expected.Name {
				t.Errorf("Name: got %q, want %q", event.Name, tt.expected.Name)
			}
		})
	}
}

func TestStreamEvent_NDJSONFormat(t *testing.T) {
	// Simulate a sequence of events as they would appear in NDJSON
	events := []StreamEvent{
		{Type: "text", Delta: "Hello"},
		{Type: "text", Delta: " world"},
		{Type: "tool_use", Name: "read_file", Input: map[string]any{"path": "notes.md"}},
		{Type: "status", Message: "Tool read_file executed"},
		{Type: "text", Delta: "The file contains..."},
		{Type: "ping"},
		{Type: "done", SessionID: "sess-123"},
	}

	// Build NDJSON string
	var builder strings.Builder
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}
		builder.Write(data)
		builder.WriteString("\n")
	}

	ndjson := builder.String()

	// Parse NDJSON back
	lines := strings.Split(strings.TrimSpace(ndjson), "\n")
	if len(lines) != len(events) {
		t.Fatalf("Expected %d lines, got %d", len(events), len(lines))
	}

	for i, line := range lines {
		var parsed StreamEvent
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("Line %d: failed to parse: %v", i, err)
			continue
		}
		if parsed.Type != events[i].Type {
			t.Errorf("Line %d: type mismatch: got %q, want %q", i, parsed.Type, events[i].Type)
		}
	}
}

func TestStreamEvent_TextDeltaAccumulation(t *testing.T) {
	// Test that text deltas can be accumulated to form complete message
	deltas := []string{"Hello", " ", "world", "!", " How", " are", " you", "?"}

	var accumulated strings.Builder
	for _, delta := range deltas {
		event := StreamEvent{Type: "text", Delta: delta}
		// Serialize and deserialize to ensure format is preserved
		data, _ := json.Marshal(event)
		var parsed StreamEvent
		json.Unmarshal(data, &parsed)

		if parsed.Type == "text" {
			accumulated.WriteString(parsed.Delta)
		}
	}

	expected := "Hello world! How are you?"
	if accumulated.String() != expected {
		t.Errorf("Accumulated text: got %q, want %q", accumulated.String(), expected)
	}
}

func TestStreamEvent_ToolInputTypes(t *testing.T) {
	// Test various input types that tools might receive
	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "string input",
			input: map[string]any{"path": "/home/user/file.txt"},
		},
		{
			name:  "integer input",
			input: map[string]any{"count": float64(10)}, // JSON numbers are float64
		},
		{
			name:  "boolean input",
			input: map[string]any{"recursive": true},
		},
		{
			name:  "nested input",
			input: map[string]any{"options": map[string]any{"verbose": true, "format": "json"}},
		},
		{
			name:  "array input",
			input: map[string]any{"files": []any{"a.txt", "b.txt"}},
		},
		{
			name:  "empty input",
			input: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := StreamEvent{Type: "tool_use", Name: "test_tool", Input: tt.input}
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var parsed StreamEvent
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// The input should round-trip correctly
			parsedInput, ok := parsed.Input.(map[string]any)
			if !ok {
				t.Fatalf("Input is not map[string]any: %T", parsed.Input)
			}

			originalInput := tt.input.(map[string]any)
			for k := range originalInput {
				if _, exists := parsedInput[k]; !exists {
					t.Errorf("Key %q missing from parsed input", k)
				}
			}
		})
	}
}

func TestStreamEvent_SpecialCharactersInText(t *testing.T) {
	tests := []struct {
		name  string
		delta string
	}{
		{name: "newline", delta: "line1\nline2"},
		{name: "tab", delta: "col1\tcol2"},
		{name: "quotes", delta: `He said "Hello"`},
		{name: "backslash", delta: `path\to\file`},
		{name: "unicode", delta: "Hello 世界 🌍"},
		{name: "control chars", delta: "text\r\nwith\r\nCRLF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := StreamEvent{Type: "text", Delta: tt.delta}
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var parsed StreamEvent
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.Delta != tt.delta {
				t.Errorf("Delta not preserved: got %q, want %q", parsed.Delta, tt.delta)
			}
		})
	}
}

func TestStreamEvent_EventTypeValues(t *testing.T) {
	// Verify all expected event types are valid
	validTypes := []string{"text", "tool_use", "status", "ping", "done", "error"}

	for _, eventType := range validTypes {
		t.Run(eventType, func(t *testing.T) {
			event := StreamEvent{Type: eventType}
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal event type %q: %v", eventType, err)
			}

			var parsed StreamEvent
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal event type %q: %v", eventType, err)
			}

			if parsed.Type != eventType {
				t.Errorf("Type not preserved: got %q, want %q", parsed.Type, eventType)
			}
		})
	}
}

func TestStreamEvent_SessionIDFormat(t *testing.T) {
	// Session IDs should be preserved exactly
	sessionIDs := []string{
		"abc-123",
		"550e8400-e29b-41d4-a716-446655440000", // UUID format
		"session_with_underscore",
		"MixedCaseSession123",
	}

	for _, sessionID := range sessionIDs {
		t.Run(sessionID, func(t *testing.T) {
			event := StreamEvent{Type: "done", SessionID: sessionID}
			data, _ := json.Marshal(event)

			var parsed StreamEvent
			json.Unmarshal(data, &parsed)

			if parsed.SessionID != sessionID {
				t.Errorf("SessionID not preserved: got %q, want %q", parsed.SessionID, sessionID)
			}
		})
	}
}

func TestStreamEvent_ErrorMessageFormat(t *testing.T) {
	errorMessages := []string{
		"API error (status 429): Rate limited",
		"Failed to execute tool: file not found",
		"Claude API key not configured",
		"stream read error: connection reset",
	}

	for _, msg := range errorMessages {
		t.Run(msg[:20], func(t *testing.T) {
			event := StreamEvent{Type: "error", Message: msg}
			data, _ := json.Marshal(event)

			var parsed StreamEvent
			json.Unmarshal(data, &parsed)

			if parsed.Message != msg {
				t.Errorf("Error message not preserved: got %q, want %q", parsed.Message, msg)
			}
		})
	}
}

func TestProcessStreamTrimsLeadingBlankLineDeltas(t *testing.T) {
	svc := &Service{}
	events := make(chan StreamEvent, 10)
	payload := strings.Join([]string{
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"\n"}}`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
		`data: {"type":"message_stop"}`,
	}, "\n")

	resp, text, err := svc.processStream(strings.NewReader(payload), events)
	if err != nil {
		t.Fatalf("processStream failed: %v", err)
	}
	if text != "Hello" {
		t.Fatalf("unexpected text: %q", text)
	}
	if resp.StopReason != "end_turn" {
		t.Fatalf("unexpected stop_reason: %q", resp.StopReason)
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != "text" || resp.Content[0].Text != "Hello" {
		t.Fatalf("unexpected response content: %#v", resp.Content)
	}

	close(events)
	var deltas []string
	for event := range events {
		if event.Type == "text" {
			deltas = append(deltas, event.Delta)
		}
	}
	if len(deltas) != 1 || deltas[0] != "Hello" {
		t.Fatalf("unexpected emitted deltas: %#v", deltas)
	}
}

func TestProcessStreamSkipsWhitespaceOnlyText(t *testing.T) {
	svc := &Service{}
	events := make(chan StreamEvent, 10)
	payload := strings.Join([]string{
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"\n"}}`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"\t "}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
		`data: {"type":"message_stop"}`,
	}, "\n")

	resp, text, err := svc.processStream(strings.NewReader(payload), events)
	if err != nil {
		t.Fatalf("processStream failed: %v", err)
	}
	if text != "" {
		t.Fatalf("expected empty text, got %q", text)
	}
	if len(resp.Content) != 0 {
		t.Fatalf("expected no content blocks, got %#v", resp.Content)
	}

	close(events)
	for event := range events {
		t.Fatalf("expected no emitted events, got %#v", event)
	}
}

func TestProcessStreamReturnsScannerError(t *testing.T) {
	svc := &Service{}
	events := make(chan StreamEvent, 1)
	_, _, err := svc.processStream(errReader{}, events)
	if err == nil {
		t.Fatal("expected stream read error")
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
