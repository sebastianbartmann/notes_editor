package agent

import (
	"context"
	"encoding/json"
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
	var postedResult piGatewayToolResult

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/chat-stream":
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = w.Write([]byte(`{"type":"start","run_id":"run-1","session_id":"runtime-session-1"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"text","delta":"hello "}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"tool_call","id":"call-1","tool":"read_file","args":{"path":"notes/test.md"}}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"text","delta":"world"}` + "\n"))
			_, _ = w.Write([]byte(`{"type":"done"}` + "\n"))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/tool-result":
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&postedResult); err != nil {
				t.Fatalf("decode tool result: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
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
	if postedResult.ID != "call-1" {
		t.Fatalf("unexpected tool result id: %q", postedResult.ID)
	}
	if !postedResult.OK {
		t.Fatal("expected posted tool result OK=true")
	}
	if !strings.Contains(postedResult.Content, "vault content") {
		t.Fatalf("expected posted result content to contain file content, got %q", postedResult.Content)
	}
}
