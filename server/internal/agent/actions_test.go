package agent

import (
	"os"
	"path/filepath"
	"testing"

	"notes-editor/internal/vault"
)

func TestParseActionContent(t *testing.T) {
	content := `---
requires_confirmation: true
max_steps: 4
---
Run this action.`

	meta, prompt, err := parseActionContent(content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !meta.RequiresConfirmation {
		t.Fatal("expected requires_confirmation=true")
	}
	if meta.MaxSteps != 4 {
		t.Fatalf("expected max_steps=4, got %d", meta.MaxSteps)
	}
	if prompt != "Run this action." {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestListActionsReturnsFileBackedActions(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"
	actionsDir := filepath.Join(root, person, "agent", "actions")
	if err := os.MkdirAll(actionsDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(actionsDir, "Extract URLs.prompt.md"), []byte("Do it"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	svc := NewService(nil, vault.NewStore(root))
	actions, err := svc.ListActions(person)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].ID != "extract-urls" {
		t.Fatalf("unexpected id: %q", actions[0].ID)
	}
}

func TestResolveMessageRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"
	actionsDir := filepath.Join(root, person, "agent", "actions")
	if err := os.MkdirAll(actionsDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	action := `---
requires_confirmation: true
---
Sensitive action`
	if err := os.WriteFile(filepath.Join(actionsDir, "Sensitive.prompt.md"), []byte(action), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	svc := NewService(nil, vault.NewStore(root))
	_, err := svc.resolveMessage(person, ChatRequest{ActionID: "sensitive"})
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}
