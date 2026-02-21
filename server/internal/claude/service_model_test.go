package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCallAPIUsesConfiguredModel(t *testing.T) {
	var capturedModel string
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		capturedModel, _ = payload["model"].(string)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id":"msg_1",
				"type":"message",
				"role":"assistant",
				"content":[{"type":"text","text":"ok"}],
				"stop_reason":"end_turn"
			}`)),
			Header: make(http.Header),
		}, nil
	})
	defer func() { http.DefaultTransport = origTransport }()

	svc := &Service{apiKey: "test-key", model: "claude-test-model"}
	if _, err := svc.callAPI([]anthropicMsg{{Role: "user", Content: "hi"}}, false, "system"); err != nil {
		t.Fatalf("callAPI failed: %v", err)
	}

	if capturedModel != "claude-test-model" {
		t.Fatalf("unexpected model in request: got %q want %q", capturedModel, "claude-test-model")
	}
}

func TestCallAPIUsesDefaultModelWhenConfiguredModelEmpty(t *testing.T) {
	var capturedModel string
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		capturedModel, _ = payload["model"].(string)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id":"msg_1",
				"type":"message",
				"role":"assistant",
				"content":[{"type":"text","text":"ok"}],
				"stop_reason":"end_turn"
			}`)),
			Header: make(http.Header),
		}, nil
	})
	defer func() { http.DefaultTransport = origTransport }()

	svc := &Service{apiKey: "test-key"}
	if _, err := svc.callAPI([]anthropicMsg{{Role: "user", Content: "hi"}}, false, "system"); err != nil {
		t.Fatalf("callAPI failed: %v", err)
	}

	if capturedModel != defaultModel {
		t.Fatalf("unexpected model in request: got %q want %q", capturedModel, defaultModel)
	}
}

func TestStreamAPIUsesConfiguredModel(t *testing.T) {
	var capturedModel string
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		capturedModel, _ = payload["model"].(string)

		sse := strings.Join([]string{
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
			`data: {"type":"message_stop"}`,
		}, "\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(sse)),
			Header:     make(http.Header),
		}, nil
	})
	defer func() { http.DefaultTransport = origTransport }()

	svc := &Service{apiKey: "test-key", model: "claude-stream-model"}
	events := make(chan StreamEvent, 4)
	if _, _, err := svc.streamAPI([]anthropicMsg{{Role: "user", Content: "hi"}}, events, "system"); err != nil {
		t.Fatalf("streamAPI failed: %v", err)
	}

	if capturedModel != "claude-stream-model" {
		t.Fatalf("unexpected model in stream request: got %q want %q", capturedModel, "claude-stream-model")
	}
}
