package agent

import (
	"strings"
	"testing"

	"notes-editor/internal/vault"
)

func TestExportSessionsMarkdownWritesMarkdownFiles(t *testing.T) {
	t.Setenv("PI_GATEWAY_PI_SESSION_DIR", t.TempDir())
	store := vault.NewStore(t.TempDir())
	svc := NewServiceWithRuntimes(store, map[string]Runtime{})

	svc.touchSession("petra", "session-abc", "Trip ideas", RuntimeModeAnthropicAPIKey)
	svc.replaceStoredConversation("petra", "session-abc", []ConversationItem{
		{Type: ConversationItemMessage, Role: "user", Content: "Please summarize last week."},
		{Type: ConversationItemToolCall, Tool: "read_file"},
		{Type: ConversationItemMessage, Role: "assistant", Content: "Here is the summary."},
		{Type: ConversationItemUsage, Usage: &UsageSnapshot{TotalTokens: 123}},
	})

	exported, err := svc.ExportSessionsMarkdown("petra")
	if err != nil {
		t.Fatalf("ExportSessionsMarkdown failed: %v", err)
	}
	if exported == nil {
		t.Fatal("expected non-nil export result")
	}
	if len(exported.Files) < 2 {
		t.Fatalf("expected at least README + one session file, got %d", len(exported.Files))
	}
	if !strings.HasPrefix(exported.Directory, "agent/session_exports/") {
		t.Fatalf("unexpected export directory: %q", exported.Directory)
	}

	readme, err := store.ReadFile("petra", exported.Files[0])
	if err != nil {
		t.Fatalf("failed reading README export: %v", err)
	}
	if !strings.Contains(readme, "# Agent session export") {
		t.Fatalf("unexpected README content: %q", readme)
	}

	sessionDoc, err := store.ReadFile("petra", exported.Files[1])
	if err != nil {
		t.Fatalf("failed reading session export: %v", err)
	}
	if !strings.Contains(sessionDoc, "### User") || !strings.Contains(sessionDoc, "### Assistant") {
		t.Fatalf("session export missing message sections: %q", sessionDoc)
	}
	if strings.Contains(sessionDoc, "tool_call") || strings.Contains(sessionDoc, "Usage:") {
		t.Fatalf("session export should include only message content, got: %q", sessionDoc)
	}
}
