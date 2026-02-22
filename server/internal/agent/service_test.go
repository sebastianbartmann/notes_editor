package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"notes-editor/internal/claude"
	"notes-editor/internal/vault"
)

type stubRuntime struct {
	mode       string
	available  bool
	chatResp   *RuntimeChatResponse
	chatErr    error
	streamResp *RuntimeStream
	streamErr  error
	history    map[string][]claude.ChatMessage
}

func (r *stubRuntime) Mode() string { return r.mode }

func (r *stubRuntime) Available() bool { return r.available }

func (r *stubRuntime) Chat(_ string, _ RuntimeChatRequest) (*RuntimeChatResponse, error) {
	if r.chatErr != nil {
		return nil, r.chatErr
	}
	return r.chatResp, nil
}

func (r *stubRuntime) ChatStream(_ context.Context, _ string, _ RuntimeChatRequest) (*RuntimeStream, error) {
	if r.streamErr != nil {
		return nil, r.streamErr
	}
	return r.streamResp, nil
}

func (r *stubRuntime) ClearSession(_ string) error { return nil }

func (r *stubRuntime) GetHistory(sessionID string) ([]claude.ChatMessage, error) {
	if r.history == nil {
		return nil, nil
	}
	history := r.history[sessionID]
	if len(history) == 0 {
		return nil, nil
	}
	out := make([]claude.ChatMessage, len(history))
	copy(out, history)
	return out, nil
}

type cancelAwareRuntime struct {
	mode      string
	available bool
	lastCtx   context.Context
	events    chan StreamEvent
}

func (r *cancelAwareRuntime) Mode() string { return r.mode }

func (r *cancelAwareRuntime) Available() bool { return r.available }

func (r *cancelAwareRuntime) Chat(_ string, _ RuntimeChatRequest) (*RuntimeChatResponse, error) {
	return nil, nil
}

func (r *cancelAwareRuntime) ChatStream(ctx context.Context, _ string, _ RuntimeChatRequest) (*RuntimeStream, error) {
	r.lastCtx = ctx
	if r.events == nil {
		r.events = make(chan StreamEvent)
	}
	return &RuntimeStream{Events: r.events}, nil
}

func (r *cancelAwareRuntime) ClearSession(_ string) error { return nil }

func (r *cancelAwareRuntime) GetHistory(_ string) ([]claude.ChatMessage, error) { return nil, nil }

func TestMapClaudeEventToolUseToToolCall(t *testing.T) {
	events := mapClaudeEvent(claude.StreamEvent{
		Type:  "tool_use",
		Name:  "read_file",
		Input: map[string]any{"path": "notes.md"},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "tool_call" {
		t.Fatalf("expected tool_call, got %q", events[0].Type)
	}
	if events[0].Tool != "read_file" {
		t.Fatalf("expected tool read_file, got %q", events[0].Tool)
	}
	if events[0].Args["path"] != "notes.md" {
		t.Fatalf("expected args.path notes.md, got %v", events[0].Args["path"])
	}
}

func TestMapClaudeEventToolStatusToToolResult(t *testing.T) {
	events := mapClaudeEvent(claude.StreamEvent{
		Type:    "status",
		Message: "Tool read_file executed",
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "tool_result" {
		t.Fatalf("expected tool_result, got %q", events[0].Type)
	}
	if events[0].Tool != "read_file" {
		t.Fatalf("expected tool read_file, got %q", events[0].Tool)
	}
	if !events[0].OK {
		t.Fatal("expected OK=true")
	}
}

func TestSelectRuntimePiFallbackToAnthropic(t *testing.T) {
	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     &stubRuntime{mode: RuntimeModeAnthropicAPIKey, available: true},
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})

	runtime, status, err := svc.selectRuntimeForMode(RuntimeModeGatewaySubscription)
	if err != nil {
		t.Fatalf("select runtime failed: %v", err)
	}
	if runtime.Mode() != RuntimeModeAnthropicAPIKey {
		t.Fatalf("expected anthropic runtime, got %q", runtime.Mode())
	}
	if status == "" {
		t.Fatal("expected fallback status message")
	}
}

func TestSelectRuntimePiUnavailableWithoutFallback(t *testing.T) {
	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     &stubRuntime{mode: RuntimeModeAnthropicAPIKey, available: false},
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})
	svc.allowPiFallback = false

	_, _, err := svc.selectRuntimeForMode(RuntimeModeGatewaySubscription)
	if err == nil {
		t.Fatal("expected runtime selection error")
	}
	if !IsRuntimeUnavailable(err) {
		t.Fatalf("expected runtime unavailable error, got %v", err)
	}
}

func TestChatStreamRejectsConcurrentRunsForSameSession(t *testing.T) {
	upstream := make(chan StreamEvent)
	anthropic := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	store := vault.NewStore(t.TempDir())
	svc := NewServiceWithRuntimes(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     anthropic,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})
	svc.maxRunDuration = 5 * time.Second

	run1, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "shared-session",
		Message:   "first",
	})
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for range run1.Events {
		}
		close(done)
	}()

	_, err = svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "shared-session",
		Message:   "second",
	})
	if !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("expected ErrSessionBusy, got %v", err)
	}

	close(upstream)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first stream to finish")
	}
}

func TestChatStreamStopsWhenToolCallLimitExceeded(t *testing.T) {
	upstream := make(chan StreamEvent, 4)
	upstream <- StreamEvent{Type: "tool_call", Tool: "read_file", Args: map[string]any{"path": "a.md"}}
	upstream <- StreamEvent{Type: "tool_call", Tool: "read_file", Args: map[string]any{"path": "b.md"}}
	close(upstream)

	anthropic := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	store := vault.NewStore(t.TempDir())
	svc := NewServiceWithRuntimesAndOptions(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     anthropic,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	}, ServiceOptions{MaxToolCalls: 1})

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		Message: "limit test",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	var sawLimitError bool
	for event := range run.Events {
		if event.Type == "error" && strings.Contains(event.Message, "max tool calls") {
			sawLimitError = true
		}
	}
	if !sawLimitError {
		t.Fatal("expected max tool calls error event")
	}
}

func TestChatStreamRegistersNewSessionFromStreamEvent(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	upstream := make(chan StreamEvent, 3)
	upstream <- StreamEvent{Type: "start", SessionID: "new-session-1"}
	upstream <- StreamEvent{Type: "text", Delta: "hello"}
	upstream <- StreamEvent{Type: "done", SessionID: "new-session-1"}
	close(upstream)

	runtime := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})
	mode := RuntimeModeAnthropicAPIKey
	if _, err := svc.SaveConfig("sebastian", ConfigUpdate{RuntimeMode: &mode}); err != nil {
		t.Fatalf("failed to set runtime mode: %v", err)
	}

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		Message: "first message from android",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}
	for range run.Events {
	}

	sessions, err := svc.ListSessions("sebastian")
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].SessionID != "new-session-1" {
		t.Fatalf("expected session_id new-session-1, got %q", sessions[0].SessionID)
	}
}

func TestListActiveRunsForPerson(t *testing.T) {
	upstream := make(chan StreamEvent)
	runtime := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "s-active",
		Message:   "hello",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	runs := svc.ListActiveRuns("sebastian")
	if len(runs) != 1 {
		t.Fatalf("expected one active run, got %d", len(runs))
	}
	if runs[0].RunID != run.RunID {
		t.Fatalf("expected run id %q, got %q", run.RunID, runs[0].RunID)
	}
	if runs[0].SessionID != "s-active" {
		t.Fatalf("expected session id s-active, got %q", runs[0].SessionID)
	}

	close(upstream)
	for range run.Events {
	}

	runs = svc.ListActiveRuns("sebastian")
	if len(runs) != 0 {
		t.Fatalf("expected zero active runs after completion, got %d", len(runs))
	}
}

func TestChatStreamEmitsErrorWhenDoneArrivesWithoutOutput(t *testing.T) {
	upstream := make(chan StreamEvent, 1)
	upstream <- StreamEvent{Type: "done", SessionID: "s-empty"}
	close(upstream)

	runtime := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		Message: "empty output",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	var got []StreamEvent
	for event := range run.Events {
		got = append(got, event)
	}
	if len(got) < 2 {
		t.Fatalf("expected at least error+done events, got %d", len(got))
	}

	var sawError bool
	var sawDone bool
	for _, event := range got {
		if event.Type == "error" && strings.Contains(event.Message, "No assistant output received") {
			sawError = true
		}
		if event.Type == "done" {
			sawDone = true
		}
	}
	if !sawError {
		t.Fatalf("expected empty stream error, events=%v", got)
	}
	if !sawDone {
		t.Fatalf("expected done event, events=%v", got)
	}
}

func TestChatStreamEmitsErrorWhenStreamClosesWithoutDoneOrOutput(t *testing.T) {
	upstream := make(chan StreamEvent)
	close(upstream)

	runtime := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		Message: "empty close",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	var sawError bool
	var sawDone bool
	for event := range run.Events {
		if event.Type == "error" && strings.Contains(event.Message, "No assistant output received") {
			sawError = true
		}
		if event.Type == "done" {
			sawDone = true
		}
	}
	if !sawError || !sawDone {
		t.Fatalf("expected empty-stream error and done, got error=%v done=%v", sawError, sawDone)
	}
}

func TestChatStreamStoresAssistantSegmentsInTimelineOrder(t *testing.T) {
	upstream := make(chan StreamEvent, 8)
	upstream <- StreamEvent{Type: "text", Delta: "First part."}
	upstream <- StreamEvent{Type: "tool_call", Tool: "read_file", Args: map[string]any{"path": "daily.md"}}
	upstream <- StreamEvent{Type: "text", Delta: "Second part."}
	upstream <- StreamEvent{Type: "tool_result", Tool: "read_file", OK: true, Summary: "Tool read_file executed"}
	upstream <- StreamEvent{Type: "done", SessionID: "s-seq"}
	close(upstream)

	runtime := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		Message: "test sequence",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}
	for range run.Events {
	}

	history, err := svc.GetConversationHistory("sebastian", "s-seq")
	if err != nil {
		t.Fatalf("failed to fetch history: %v", err)
	}

	relevant := make([]ConversationItem, 0, len(history))
	for _, item := range history {
		if item.Type == ConversationItemStatus {
			continue
		}
		relevant = append(relevant, item)
	}
	if len(relevant) != 5 {
		t.Fatalf("expected 5 relevant items, got %d", len(relevant))
	}
	if relevant[0].Type != ConversationItemMessage || relevant[0].Role != "user" {
		t.Fatalf("expected first item to be user message, got %#v", relevant[0])
	}
	if relevant[1].Type != ConversationItemMessage || relevant[1].Role != "assistant" || relevant[1].Content != "First part." {
		t.Fatalf("expected assistant segment before tool call, got %#v", relevant[1])
	}
	if relevant[2].Type != ConversationItemToolCall {
		t.Fatalf("expected tool_call at position 3, got %#v", relevant[2])
	}
	if relevant[3].Type != ConversationItemMessage || relevant[3].Role != "assistant" || relevant[3].Content != "Second part." {
		t.Fatalf("expected second assistant segment before tool result, got %#v", relevant[3])
	}
	if relevant[4].Type != ConversationItemToolResult {
		t.Fatalf("expected tool_result at position 5, got %#v", relevant[4])
	}
}

func TestGetConversationHistoryIncludesInFlightItems(t *testing.T) {
	upstream := make(chan StreamEvent, 8)
	runtime := &stubRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		streamResp: &RuntimeStream{
			Events: upstream,
		},
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "s-inflight",
		Message:   "in flight user",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	readEvent := func() StreamEvent {
		t.Helper()
		select {
		case event := <-run.Events:
			return event
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for stream event")
			return StreamEvent{}
		}
	}

	if got := readEvent(); got.Type != "start" {
		t.Fatalf("expected start event, got %#v", got)
	}

	upstream <- StreamEvent{Type: "text", Delta: "Assistant in flight. "}
	upstream <- StreamEvent{Type: "tool_call", Tool: "read_file", Args: map[string]any{"path": "daily.md"}}
	_ = readEvent() // text
	_ = readEvent() // tool_call

	history, err := svc.GetConversationHistory("sebastian", "s-inflight")
	if err != nil {
		t.Fatalf("failed to fetch in-flight history: %v", err)
	}

	var sawUser bool
	var sawAssistant bool
	var sawToolCall bool
	for _, item := range history {
		if item.Type == ConversationItemMessage && item.Role == "user" && item.Content == "in flight user" {
			sawUser = true
		}
		if item.Type == ConversationItemMessage && item.Role == "assistant" && strings.Contains(item.Content, "Assistant in flight.") {
			sawAssistant = true
		}
		if item.Type == ConversationItemToolCall && item.Tool == "read_file" {
			sawToolCall = true
		}
	}
	if !sawUser || !sawAssistant || !sawToolCall {
		t.Fatalf("expected in-flight history to include user/assistant/tool_call, got %#v", history)
	}

	upstream <- StreamEvent{Type: "done", SessionID: "s-inflight"}
	close(upstream)
	for range run.Events {
	}

	finalHistory, err := svc.GetConversationHistory("sebastian", "s-inflight")
	if err != nil {
		t.Fatalf("failed to fetch final history: %v", err)
	}
	if len(finalHistory) < len(history) {
		t.Fatalf("expected final history to include in-flight timeline, before=%d after=%d", len(history), len(finalHistory))
	}
}

func TestStopRunCancelsUpstreamStreamContext(t *testing.T) {
	runtime := &cancelAwareRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "cancel-prop-stop",
		Message:   "hello",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	if runtime.lastCtx == nil {
		t.Fatal("expected runtime stream context to be set")
	}

	if !svc.StopRun("sebastian", run.RunID) {
		t.Fatal("StopRun returned false")
	}

	select {
	case <-runtime.lastCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("expected upstream stream context to be cancelled")
	}

	close(runtime.events)
	for range run.Events {
	}
}

func TestChatStreamTimeoutCancelsUpstreamStreamContext(t *testing.T) {
	runtime := &cancelAwareRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     runtime,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})
	svc.maxRunDuration = 50 * time.Millisecond

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "cancel-prop-timeout",
		Message:   "hello",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}

	if runtime.lastCtx == nil {
		t.Fatal("expected runtime stream context to be set")
	}

	select {
	case <-runtime.lastCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("expected upstream stream context to be cancelled on timeout")
	}

	close(runtime.events)
	for range run.Events {
	}
}

func TestStoredConversationItemsRoundTrip(t *testing.T) {
	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey: &stubRuntime{mode: RuntimeModeAnthropicAPIKey, available: true},
	})

	want := []ConversationItem{
		{Type: ConversationItemMessage, Role: "user", Content: "hello"},
		{Type: ConversationItemToolCall, Tool: "read_file", Args: map[string]any{"path": "daily.md"}},
		{Type: ConversationItemToolResult, Tool: "read_file", OK: true, Summary: "ok"},
	}
	if err := svc.writeStoredConversationFile("sebastian", "roundtrip", want); err != nil {
		t.Fatalf("write stored conversation failed: %v", err)
	}

	got, found, err := svc.readStoredConversationFile("sebastian", "roundtrip")
	if err != nil {
		t.Fatalf("read stored conversation failed: %v", err)
	}
	if !found {
		t.Fatal("expected stored conversation file to be found")
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d", len(want), len(got))
	}
	if got[1].Type != ConversationItemToolCall || got[1].Tool != "read_file" {
		t.Fatalf("expected tool_call item to round-trip, got %#v", got[1])
	}
	if got[2].Type != ConversationItemToolResult || !got[2].OK {
		t.Fatalf("expected tool_result item to round-trip, got %#v", got[2])
	}
}

func TestGetConversationHistoryPrefersDurableTimeline(t *testing.T) {
	root := t.TempDir()
	store := vault.NewStore(root)

	upstream := make(chan StreamEvent, 8)
	upstream <- StreamEvent{Type: "text", Delta: "First part."}
	upstream <- StreamEvent{Type: "tool_call", Tool: "read_file", Args: map[string]any{"path": "daily.md"}}
	upstream <- StreamEvent{Type: "text", Delta: "Second part."}
	upstream <- StreamEvent{Type: "tool_result", Tool: "read_file", OK: true, Summary: "Tool read_file executed"}
	upstream <- StreamEvent{Type: "done", SessionID: "persisted-session"}
	close(upstream)

	writerSvc := NewServiceWithRuntimes(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey: &stubRuntime{
			mode:      RuntimeModeAnthropicAPIKey,
			available: true,
			streamResp: &RuntimeStream{
				Events: upstream,
			},
		},
	})

	run, err := writerSvc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "persisted-session",
		Message:   "persist this timeline",
	})
	if err != nil {
		t.Fatalf("chat stream failed: %v", err)
	}
	for range run.Events {
	}

	readerSvc := NewServiceWithRuntimes(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey: &stubRuntime{
			mode:      RuntimeModeAnthropicAPIKey,
			available: true,
			history: map[string][]claude.ChatMessage{
				"persisted-session": {
					{Role: "user", Content: "persist this timeline"},
					{Role: "assistant", Content: "message-only fallback"},
				},
			},
		},
	})

	history, err := readerSvc.GetConversationHistory("sebastian", "persisted-session")
	if err != nil {
		t.Fatalf("get conversation history failed: %v", err)
	}

	var sawToolCall bool
	var sawToolResult bool
	for _, item := range history {
		if item.Type == ConversationItemToolCall && item.Tool == "read_file" {
			sawToolCall = true
		}
		if item.Type == ConversationItemToolResult && item.Tool == "read_file" {
			sawToolResult = true
		}
	}
	if !sawToolCall || !sawToolResult {
		t.Fatalf("expected durable history with tool_call/tool_result, got %#v", history)
	}
}

func TestGetConversationHistoryFallsBackToLegacyChatMessagesWhenNoDurableFile(t *testing.T) {
	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey: &stubRuntime{
			mode:      RuntimeModeAnthropicAPIKey,
			available: true,
			history: map[string][]claude.ChatMessage{
				"legacy-session": {
					{Role: "user", Content: "hello"},
					{Role: "assistant", Content: "legacy response"},
				},
			},
		},
	})

	items, err := svc.GetConversationHistory("sebastian", "legacy-session")
	if err != nil {
		t.Fatalf("get legacy history failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 message items from legacy fallback, got %d", len(items))
	}
	if items[0].Type != ConversationItemMessage || items[1].Type != ConversationItemMessage {
		t.Fatalf("expected message-only fallback items, got %#v", items)
	}
}
