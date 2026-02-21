package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"notes-editor/internal/claude"
	"notes-editor/internal/vault"
)

// helper to create a service with a controllable upstream channel per call.
type multiStreamRuntime struct {
	mode      string
	available bool
	calls     chan chan StreamEvent // each ChatStream call receives from here
}

func (r *multiStreamRuntime) Mode() string    { return r.mode }
func (r *multiStreamRuntime) Available() bool  { return r.available }

func (r *multiStreamRuntime) Chat(_ string, _ RuntimeChatRequest) (*RuntimeChatResponse, error) {
	return nil, nil
}

func (r *multiStreamRuntime) ChatStream(_ context.Context, _ string, _ RuntimeChatRequest) (*RuntimeStream, error) {
	ch := <-r.calls
	return &RuntimeStream{Events: ch}, nil
}

func (r *multiStreamRuntime) ClearSession(_ string) error { return nil }

func (r *multiStreamRuntime) GetHistory(_ string) ([]claude.ChatMessage, error) {
	return nil, nil
}

func newTestService(t *testing.T, rt *multiStreamRuntime) *Service {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	store := vault.NewStore("/tmp/test-vault-" + time.Now().Format("20060102150405.000"))
	svc := NewServiceWithRuntimes(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     rt,
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: false},
	})
	svc.maxRunDuration = 30 * time.Second
	return svc
}

func TestConcurrentSessionsDifferentSessionIDs(t *testing.T) {
	rt := &multiStreamRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		calls:     make(chan chan StreamEvent, 2),
	}

	upstream1 := make(chan StreamEvent, 10)
	upstream2 := make(chan StreamEvent, 10)
	rt.calls <- upstream1
	rt.calls <- upstream2

	svc := newTestService(t, rt)

	// Start two concurrent streams on DIFFERENT sessions for the same person.
	run1, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "session-A",
		Message:   "hello from A",
	})
	if err != nil {
		t.Fatalf("run1 failed: %v", err)
	}

	run2, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "session-B",
		Message:   "hello from B",
	})
	if err != nil {
		t.Fatalf("run2 failed: %v", err)
	}

	// Both should appear as active runs.
	runs := svc.ListActiveRuns("sebastian")
	if len(runs) != 2 {
		t.Fatalf("expected 2 active runs, got %d", len(runs))
	}

	// Complete session A.
	upstream1 <- StreamEvent{Type: "text", Delta: "response A"}
	upstream1 <- StreamEvent{Type: "done", SessionID: "session-A"}
	close(upstream1)

	for range run1.Events {
	}

	// Session A done, session B still running.
	runs = svc.ListActiveRuns("sebastian")
	if len(runs) != 1 {
		t.Fatalf("expected 1 active run after A done, got %d", len(runs))
	}
	if runs[0].SessionID != "session-B" {
		t.Fatalf("expected remaining run on session-B, got %q", runs[0].SessionID)
	}

	// Complete session B.
	upstream2 <- StreamEvent{Type: "text", Delta: "response B"}
	upstream2 <- StreamEvent{Type: "done", SessionID: "session-B"}
	close(upstream2)

	for range run2.Events {
	}

	runs = svc.ListActiveRuns("sebastian")
	if len(runs) != 0 {
		t.Fatalf("expected 0 active runs after both done, got %d", len(runs))
	}

	// Both sessions should have conversation history.
	histA, err := svc.GetConversationHistory("sebastian", "session-A")
	if err != nil {
		t.Fatalf("get history A: %v", err)
	}
	histB, err := svc.GetConversationHistory("sebastian", "session-B")
	if err != nil {
		t.Fatalf("get history B: %v", err)
	}

	if len(histA) == 0 {
		t.Fatal("session-A history empty")
	}
	if len(histB) == 0 {
		t.Fatal("session-B history empty")
	}

	// Verify each session got its own content.
	var foundA, foundB bool
	for _, item := range histA {
		if item.Role == "assistant" && strings.Contains(item.Content, "response A") {
			foundA = true
		}
	}
	for _, item := range histB {
		if item.Role == "assistant" && strings.Contains(item.Content, "response B") {
			foundB = true
		}
	}
	if !foundA {
		t.Fatalf("session-A missing assistant response, history=%v", histA)
	}
	if !foundB {
		t.Fatalf("session-B missing assistant response, history=%v", histB)
	}
}

func TestConcurrentSessionsSameSessionBlocked(t *testing.T) {
	rt := &multiStreamRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		calls:     make(chan chan StreamEvent, 1),
	}

	upstream := make(chan StreamEvent)
	rt.calls <- upstream

	svc := newTestService(t, rt)

	run1, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "session-X",
		Message:   "first",
	})
	if err != nil {
		t.Fatalf("run1 failed: %v", err)
	}

	// Second request to the SAME session should fail with ErrSessionBusy.
	_, err = svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "session-X",
		Message:   "second",
	})
	if err == nil {
		t.Fatal("expected error for concurrent same-session")
	}
	if !IsSessionBusy(err) {
		t.Fatalf("expected ErrSessionBusy, got %v", err)
	}

	close(upstream)
	for range run1.Events {
	}
}

func TestConcurrentSessionsDifferentPersons(t *testing.T) {
	rt := &multiStreamRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		calls:     make(chan chan StreamEvent, 2),
	}

	upstream1 := make(chan StreamEvent, 5)
	upstream2 := make(chan StreamEvent, 5)
	rt.calls <- upstream1
	rt.calls <- upstream2

	svc := newTestService(t, rt)

	// Two different persons can use the same session ID concurrently.
	run1, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "shared-id",
		Message:   "seb msg",
	})
	if err != nil {
		t.Fatalf("run1 failed: %v", err)
	}

	run2, err := svc.ChatStream(context.Background(), "petra", ChatRequest{
		SessionID: "shared-id",
		Message:   "petra msg",
	})
	if err != nil {
		t.Fatalf("run2 failed: %v", err)
	}

	// Each person sees only their own runs.
	if len(svc.ListActiveRuns("sebastian")) != 1 {
		t.Fatal("expected 1 active run for sebastian")
	}
	if len(svc.ListActiveRuns("petra")) != 1 {
		t.Fatal("expected 1 active run for petra")
	}

	upstream1 <- StreamEvent{Type: "text", Delta: "seb response"}
	upstream1 <- StreamEvent{Type: "done", SessionID: "shared-id"}
	close(upstream1)

	upstream2 <- StreamEvent{Type: "text", Delta: "petra response"}
	upstream2 <- StreamEvent{Type: "done", SessionID: "shared-id"}
	close(upstream2)

	for range run1.Events {
	}
	for range run2.Events {
	}

	// Sessions are person-scoped.
	sessionsS, _ := svc.ListSessions("sebastian")
	sessionsP, _ := svc.ListSessions("petra")
	if len(sessionsS) != 1 {
		t.Fatalf("expected 1 session for sebastian, got %d", len(sessionsS))
	}
	if len(sessionsP) != 1 {
		t.Fatalf("expected 1 session for petra, got %d", len(sessionsP))
	}
}

func TestStopRunWhileStreaming(t *testing.T) {
	rt := &multiStreamRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		calls:     make(chan chan StreamEvent, 1),
	}

	upstream := make(chan StreamEvent)
	rt.calls <- upstream

	svc := newTestService(t, rt)

	run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "stop-test",
		Message:   "long running",
	})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}

	// Send some data, then stop the run.
	upstream <- StreamEvent{Type: "text", Delta: "partial "}

	if !svc.StopRun("sebastian", run.RunID) {
		t.Fatal("StopRun returned false")
	}

	// Drain events — should see a "cancelled" error and done.
	var sawCancel, sawDone bool
	for event := range run.Events {
		if event.Type == "error" && strings.Contains(event.Message, "cancelled") {
			sawCancel = true
		}
		if event.Type == "done" {
			sawDone = true
		}
	}
	if !sawCancel {
		t.Fatal("expected cancel error event")
	}
	if !sawDone {
		t.Fatal("expected done event after cancel")
	}

	// Run should be cleaned up.
	if len(svc.ListActiveRuns("sebastian")) != 0 {
		t.Fatal("expected no active runs after stop")
	}

	// History should still have the partial content.
	hist, _ := svc.GetConversationHistory("sebastian", "stop-test")
	if len(hist) == 0 {
		t.Fatal("expected non-empty history after stopped run")
	}
}

func TestSessionHistoryAvailableDuringActiveRun(t *testing.T) {
	rt := &multiStreamRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		calls:     make(chan chan StreamEvent, 1),
	}

	upstream := make(chan StreamEvent, 10)
	rt.calls <- upstream

	svc := newTestService(t, rt)

	// Pre-populate a session with a completed run.
	upstream <- StreamEvent{Type: "text", Delta: "first run done"}
	upstream <- StreamEvent{Type: "done", SessionID: "hist-test"}
	close(upstream)

	run1, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "hist-test",
		Message:   "first message",
	})
	if err != nil {
		t.Fatalf("run1 failed: %v", err)
	}
	for range run1.Events {
	}

	// Now start a second run that stays open.
	upstream2 := make(chan StreamEvent)
	rt.calls = make(chan chan StreamEvent, 1)
	rt.calls <- upstream2

	run2, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "hist-test",
		Message:   "second message",
	})
	if err != nil {
		t.Fatalf("run2 failed: %v", err)
	}

	// While run2 is active, history should return at least the first run's data.
	hist, err := svc.GetConversationHistory("sebastian", "hist-test")
	if err != nil {
		t.Fatalf("get history during active run: %v", err)
	}

	var foundFirstResponse bool
	for _, item := range hist {
		if item.Role == "assistant" && strings.Contains(item.Content, "first run done") {
			foundFirstResponse = true
		}
	}
	if !foundFirstResponse {
		t.Fatalf("expected first run's response in history during active run, got %v", hist)
	}

	// Clean up.
	close(upstream2)
	for range run2.Events {
	}
}

func TestSessionAfterRunCompletes_IsReusable(t *testing.T) {
	rt := &multiStreamRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		calls:     make(chan chan StreamEvent, 2),
	}

	// First run.
	upstream1 := make(chan StreamEvent, 5)
	upstream1 <- StreamEvent{Type: "text", Delta: "run 1"}
	upstream1 <- StreamEvent{Type: "done", SessionID: "reuse-test"}
	close(upstream1)
	rt.calls <- upstream1

	svc := newTestService(t, rt)

	run1, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "reuse-test",
		Message:   "msg 1",
	})
	if err != nil {
		t.Fatalf("run1 failed: %v", err)
	}
	for range run1.Events {
	}

	// Second run on same session should succeed (not busy).
	upstream2 := make(chan StreamEvent, 5)
	upstream2 <- StreamEvent{Type: "text", Delta: "run 2"}
	upstream2 <- StreamEvent{Type: "done", SessionID: "reuse-test"}
	close(upstream2)
	rt.calls <- upstream2

	run2, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
		SessionID: "reuse-test",
		Message:   "msg 2",
	})
	if err != nil {
		t.Fatalf("run2 failed (session should be reusable): %v", err)
	}
	for range run2.Events {
	}

	// History should contain both runs.
	hist, _ := svc.GetConversationHistory("sebastian", "reuse-test")
	var userMsgs int
	for _, item := range hist {
		if item.Type == ConversationItemMessage && item.Role == "user" {
			userMsgs++
		}
	}
	if userMsgs != 2 {
		t.Fatalf("expected 2 user messages across both runs, got %d", userMsgs)
	}
}

func TestListSessionsShowsAllPersonSessions(t *testing.T) {
	rt := &multiStreamRuntime{
		mode:      RuntimeModeAnthropicAPIKey,
		available: true,
		calls:     make(chan chan StreamEvent, 3),
	}

	svc := newTestService(t, rt)

	// Create 3 sessions with completed runs.
	for i, sid := range []string{"s1", "s2", "s3"} {
		upstream := make(chan StreamEvent, 3)
		upstream <- StreamEvent{Type: "text", Delta: "resp"}
		upstream <- StreamEvent{Type: "done", SessionID: sid}
		close(upstream)
		rt.calls <- upstream

		run, err := svc.ChatStream(context.Background(), "sebastian", ChatRequest{
			SessionID: sid,
			Message:   "msg",
		})
		if err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}
		for range run.Events {
		}
	}

	sessions, err := svc.ListSessions("sebastian")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	// Different person should see zero.
	sessionsP, _ := svc.ListSessions("petra")
	if len(sessionsP) != 0 {
		t.Fatalf("expected 0 sessions for petra, got %d", len(sessionsP))
	}
}
