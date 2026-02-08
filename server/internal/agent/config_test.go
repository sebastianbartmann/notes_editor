package agent

import (
	"testing"

	"notes-editor/internal/vault"
)

func TestConfigRoundTrip(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"
	svc := NewService(nil, vault.NewStore(root))

	cfg, err := svc.GetConfig(person)
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	if cfg.RuntimeMode != defaultRuntimeMode {
		t.Fatalf("expected default runtime mode %q, got %q", defaultRuntimeMode, cfg.RuntimeMode)
	}

	mode := "gateway_subscription"
	prompt := "custom prompt"
	updated, err := svc.SaveConfig(person, ConfigUpdate{
		RuntimeMode: &mode,
		Prompt:      &prompt,
	})
	if err != nil {
		t.Fatalf("save config failed: %v", err)
	}
	if updated.RuntimeMode != mode {
		t.Fatalf("expected runtime mode %q, got %q", mode, updated.RuntimeMode)
	}
	if updated.Prompt != prompt {
		t.Fatalf("expected prompt %q, got %q", prompt, updated.Prompt)
	}
}

func TestSaveConfigRejectsInvalidRuntimeMode(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"
	svc := NewService(nil, vault.NewStore(root))

	mode := "unknown_mode"
	_, err := svc.SaveConfig(person, ConfigUpdate{RuntimeMode: &mode})
	if err == nil {
		t.Fatal("expected error for invalid runtime mode")
	}
}
