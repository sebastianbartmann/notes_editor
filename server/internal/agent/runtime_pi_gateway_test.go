package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"notes-editor/internal/vault"
)

func TestPiGatewayRuntimeUnavailableWithoutURL(t *testing.T) {
	runtime := NewPiGatewayRuntime("")
	_, err := runtime.ChatStream(context.Background(), "sebastian", RuntimeChatRequest{
		Message: "hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsRuntimeUnavailable(err) {
		t.Fatalf("expected runtime unavailable, got %v", err)
	}
}

func TestPiGatewayRuntimeExecutesToolCalls(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/chat-stream":
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = w.Write([]byte(`{"type":"start","run_id":"run-1","session_id":"runtime-session-1"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"text","delta":"hello "}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"tool_call","tool":"read_file","args":{"path":"notes/test.md"}}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"tool_result","tool":"read_file","ok":true,"summary":"Tool read_file executed"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"text","delta":"world"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"done"}` + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer gateway.Close()

	root := t.TempDir()
	person := "sebastian"
	notesDir := filepath.Join(root, person, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(notesDir, "test.md"), []byte("vault content"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	store := vault.NewStore(root)
	runtime := NewPiGatewayRuntime(gateway.URL).WithDependencies(store, nil)

	stream, err := runtime.ChatStream(context.Background(), person, RuntimeChatRequest{
		Message: "hello",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	var sawDone bool
	var text strings.Builder
	var sawToolResult bool
	for event := range stream.Events {
		switch event.Type {
		case "text":
			text.WriteString(event.Delta)
		case "tool_result":
			sawToolResult = true
			if !event.OK {
				t.Fatalf("expected tool_result OK=true, got false with %q", event.Summary)
			}
		case "done":
			sawDone = true
		}
	}

	if !sawDone {
		t.Fatal("expected done event")
	}
	if !sawToolResult {
		t.Fatal("expected tool_result event")
	}
	if text.String() != "hello world" {
		t.Fatalf("unexpected text: %q", text.String())
	}
}

func TestPiGatewayRuntimeTrimsLeadingBlankLinesInTextStream(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/chat-stream":
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = w.Write([]byte(`{"type":"start","session_id":"runtime-session-1"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"text","delta":"\n"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"text","delta":"Hello"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"done"}` + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer gateway.Close()

	root := t.TempDir()
	person := "sebastian"
	if err := os.MkdirAll(filepath.Join(root, person), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	runtime := NewPiGatewayRuntime(gateway.URL).WithDependencies(vault.NewStore(root), nil)
	stream, err := runtime.ChatStream(context.Background(), person, RuntimeChatRequest{Message: "hello"})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	var deltas []string
	for event := range stream.Events {
		if event.Type == "text" {
			deltas = append(deltas, event.Delta)
		}
	}

	if len(deltas) != 1 || deltas[0] != "Hello" {
		t.Fatalf("unexpected deltas: %#v", deltas)
	}
}

func TestPiGatewayRuntimePersistsSessionMappingAcrossRestart(t *testing.T) {
	type chatPayload struct {
		Person    string `json:"person"`
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}

	var (
		mu              sync.Mutex
		receivedSession []string
	)

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat-stream" {
			http.NotFound(w, r)
			return
		}

		var payload chatPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		mu.Lock()
		receivedSession = append(receivedSession, payload.SessionID)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/x-ndjson")
		startSessionID := payload.SessionID
		if strings.TrimSpace(startSessionID) == "" {
			startSessionID = "runtime-session-1"
		}
		_, _ = w.Write([]byte(`{"type":"start","session_id":"` + startSessionID + `"}` + "\n"))
		_, _ = w.Write([]byte(`{"type":"text","delta":"ok"}` + "\n"))
		_, _ = w.Write([]byte(`{"type":"done"}` + "\n"))
	}))
	defer gateway.Close()

	root := t.TempDir()
	person := "petra"
	if err := os.MkdirAll(filepath.Join(root, person), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store := vault.NewStore(root)

	const appSessionID = "app-session-1"

	runtime1 := NewPiGatewayRuntime(gateway.URL).WithDependencies(store, nil)
	stream1, err := runtime1.ChatStream(context.Background(), person, RuntimeChatRequest{
		SessionID: appSessionID,
		Message:   "hello",
	})
	if err != nil {
		t.Fatalf("chat stream 1 failed: %v", err)
	}
	for range stream1.Events {
	}

	runtime2 := NewPiGatewayRuntime(gateway.URL).WithDependencies(store, nil)
	stream2, err := runtime2.ChatStream(context.Background(), person, RuntimeChatRequest{
		SessionID: appSessionID,
		Message:   "resume",
	})
	if err != nil {
		t.Fatalf("chat stream 2 failed: %v", err)
	}
	for range stream2.Events {
	}

	mu.Lock()
	defer mu.Unlock()
	if len(receivedSession) != 2 {
		t.Fatalf("expected 2 requests to gateway, got %d", len(receivedSession))
	}
	if receivedSession[0] != "" {
		t.Fatalf("expected first request to omit runtime session id, got %q", receivedSession[0])
	}
	if receivedSession[1] != "runtime-session-1" {
		t.Fatalf("expected second request to reuse persisted runtime session id, got %q", receivedSession[1])
	}
}
