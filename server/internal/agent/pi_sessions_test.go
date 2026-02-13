package agent

import (
	"os"
	"path/filepath"
	"testing"

	"notes-editor/internal/vault"
)

func TestReadGatewaySessionHistoryParsesUserAndAssistantText(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	dir := os.Getenv("PI_GATEWAY_PI_SESSION_DIR")

	content := "" +
		"{\"type\":\"session\"}\n" +
		"{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"hi\"}]}}\n" +
		"{\"type\":\"message\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"hello\"},{\"type\":\"thinking\",\"thinking\":\"x\"},{\"type\":\"text\",\"text\":\" world\"}]}}\n" +
		"{\"type\":\"message\",\"message\":{\"role\":\"toolResult\",\"content\":[{\"type\":\"text\",\"text\":\"ignore\"}]}}\n"

	path := filepath.Join(dir, "petra--s1.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	history, err := readGatewaySessionHistory("petra", "s1")
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "hi" {
		t.Fatalf("unexpected first message: %+v", history[0])
	}
	if history[1].Role != "assistant" || history[1].Content != "hello world" {
		t.Fatalf("unexpected second message: %+v", history[1])
	}
}

func TestPiGatewayRuntimeGetHistoryForPersonReadsSessionFile(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	dir := os.Getenv("PI_GATEWAY_PI_SESSION_DIR")

	content := "" +
		"{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"u1\"}]}}\n" +
		"{\"type\":\"message\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"a1\"}]}}\n"
	if err := os.WriteFile(filepath.Join(dir, "petra--runtime-1.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	root := t.TempDir()
	store := vault.NewStore(root)
	runtime := NewPiGatewayRuntime("http://example.com").WithDependencies(store, nil)
	runtime.setRuntimeSessionID("petra", "app-1", "runtime-1")

	history, err := runtime.GetHistoryForPerson("petra", "app-1")
	if err != nil {
		t.Fatalf("history error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Content != "u1" || history[1].Content != "a1" {
		t.Fatalf("unexpected history: %+v", history)
	}
}
