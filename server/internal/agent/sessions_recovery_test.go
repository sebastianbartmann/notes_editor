package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"notes-editor/internal/vault"
)

func TestListGatewayRuntimeSessionFiles(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	dir := os.Getenv("PI_GATEWAY_PI_SESSION_DIR")

	older := filepath.Join(dir, "petra--old-session.jsonl")
	newer := filepath.Join(dir, "petra--new-session.jsonl")
	other := filepath.Join(dir, "sebastian--ignore.jsonl")

	sessionBody := "{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"hi\"}]}}\n"
	if err := os.WriteFile(older, []byte(sessionBody), 0644); err != nil {
		t.Fatalf("write older: %v", err)
	}
	if err := os.WriteFile(newer, []byte(sessionBody), 0644); err != nil {
		t.Fatalf("write newer: %v", err)
	}
	if err := os.WriteFile(other, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write other: %v", err)
	}

	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(older, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes older: %v", err)
	}
	if err := os.Chtimes(newer, newTime, newTime); err != nil {
		t.Fatalf("chtimes newer: %v", err)
	}

	got := listGatewayRuntimeSessionFiles("petra")
	if len(got) != 2 {
		t.Fatalf("expected 2 recovered sessions, got %d", len(got))
	}
	if got[0].SessionID != "old-session" && got[1].SessionID != "old-session" {
		t.Fatalf("missing old-session in %+v", got)
	}
	if got[0].SessionID != "new-session" && got[1].SessionID != "new-session" {
		t.Fatalf("missing new-session in %+v", got)
	}
}

func TestListGatewayRuntimeSessionFilesReturnsAllMatches(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	dir := os.Getenv("PI_GATEWAY_PI_SESSION_DIR")

	body := "{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"hi\"}]}}\n"
	const total = 35
	for i := 0; i < total; i++ {
		name := filepath.Join(dir, fmt.Sprintf("petra--session-%02d.jsonl", i))
		if err := os.WriteFile(name, []byte(body), 0644); err != nil {
			t.Fatalf("write session file %d: %v", i, err)
		}
	}

	got := listGatewayRuntimeSessionFiles("petra")
	if len(got) != total {
		t.Fatalf("expected %d recovered sessions, got %d", total, len(got))
	}
}

func TestListSessionsHydratesGatewayRecoveredSessions(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	dir := os.Getenv("PI_GATEWAY_PI_SESSION_DIR")
	body := "{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"hello\"}]}}\n"
	if err := os.WriteFile(filepath.Join(dir, "petra--recover-me.jsonl"), []byte(body), 0644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	svc := NewServiceWithRuntimes(vault.NewStore(t.TempDir()), map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     &stubRuntime{mode: RuntimeModeAnthropicAPIKey, available: true},
		RuntimeModeGatewaySubscription: &stubRuntime{mode: RuntimeModeGatewaySubscription, available: true},
	})

	sessions, err := svc.ListSessions("petra")
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 recovered session, got %d", len(sessions))
	}
	if sessions[0].SessionID != "recover-me" {
		t.Fatalf("expected recover-me session id, got %q", sessions[0].SessionID)
	}
}

func TestListSessionsDoesNotDuplicateMappedRuntimeSession(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	dir := os.Getenv("PI_GATEWAY_PI_SESSION_DIR")
	body := "{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"hello\"}]}}\n"
	if err := os.WriteFile(filepath.Join(dir, "petra--runtime-123.jsonl"), []byte(body), 0644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	store := vault.NewStore(t.TempDir())
	piRuntime := NewPiGatewayRuntime("http://example.local").WithDependencies(store, nil)
	piRuntime.setRuntimeSessionID("petra", "app-123", "runtime-123")

	svc := NewServiceWithRuntimes(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     &stubRuntime{mode: RuntimeModeAnthropicAPIKey, available: true},
		RuntimeModeGatewaySubscription: piRuntime,
	})
	svc.touchSession("petra", "app-123", "hello", RuntimeModeGatewaySubscription)

	sessions, err := svc.ListSessions("petra")
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].SessionID != "app-123" {
		t.Fatalf("expected app session id app-123, got %q", sessions[0].SessionID)
	}
}

func TestClearAllSessionsRemovesGatewayRecoveredSessionFiles(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	dir := os.Getenv("PI_GATEWAY_PI_SESSION_DIR")
	body := "{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"hello\"}]}}\n"
	sessionFile := filepath.Join(dir, "petra--runtime-123.jsonl")
	if err := os.WriteFile(sessionFile, []byte(body), 0644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	store := vault.NewStore(t.TempDir())
	piRuntime := NewPiGatewayRuntime("http://example.local").WithDependencies(store, nil)
	piRuntime.setRuntimeSessionID("petra", "app-123", "runtime-123")

	svc := NewServiceWithRuntimes(store, map[string]Runtime{
		RuntimeModeAnthropicAPIKey:     &stubRuntime{mode: RuntimeModeAnthropicAPIKey, available: true},
		RuntimeModeGatewaySubscription: piRuntime,
	})
	svc.touchSession("petra", "app-123", "hello", RuntimeModeGatewaySubscription)

	if err := svc.ClearAllSessions("petra"); err != nil {
		t.Fatalf("clear all sessions failed: %v", err)
	}

	sessions, err := svc.ListSessions("petra")
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no sessions after clear-all, got %d", len(sessions))
	}
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Fatalf("expected runtime session file removed, got err=%v", err)
	}
}
