package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
