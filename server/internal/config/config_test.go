package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadParsesAgentRuntimeSettings(t *testing.T) {
	t.Setenv("NOTES_TOKEN", "token")
	t.Setenv("NOTES_ROOT", "/tmp/notes")
	t.Setenv("PI_GATEWAY_URL", "http://127.0.0.1:4301")
	t.Setenv("AGENT_ENABLE_PI_FALLBACK", "false")
	t.Setenv("AGENT_MAX_RUN_DURATION", "90s")
	t.Setenv("AGENT_MAX_TOOL_CALLS_PER_RUN", "12")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.PiGatewayURL != "http://127.0.0.1:4301" {
		t.Fatalf("unexpected PI_GATEWAY_URL: %q", cfg.PiGatewayURL)
	}
	if cfg.AgentEnablePiFallback {
		t.Fatal("expected AgentEnablePiFallback=false")
	}
	if cfg.AgentMaxRunDuration != 90*time.Second {
		t.Fatalf("unexpected AgentMaxRunDuration: %v", cfg.AgentMaxRunDuration)
	}
	if cfg.AgentMaxToolCallsPerRun != 12 {
		t.Fatalf("unexpected AgentMaxToolCallsPerRun: %d", cfg.AgentMaxToolCallsPerRun)
	}
}

func TestReloadRuntimeSettingsUpdatesValues(t *testing.T) {
	t.Setenv("NOTES_TOKEN", "token")
	t.Setenv("NOTES_ROOT", "/tmp/notes")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if err := os.Setenv("PI_GATEWAY_URL", "http://localhost:4310"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("AGENT_ENABLE_PI_FALLBACK", "false"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("AGENT_MAX_RUN_DURATION", "45s"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("AGENT_MAX_TOOL_CALLS_PER_RUN", "7"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("VALID_PERSONS", "alice,bob"); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	if err := cfg.ReloadRuntimeSettings(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if cfg.PiGatewayURL != "http://localhost:4310" {
		t.Fatalf("unexpected PI_GATEWAY_URL: %q", cfg.PiGatewayURL)
	}
	if cfg.AgentEnablePiFallback {
		t.Fatal("expected AgentEnablePiFallback=false")
	}
	if cfg.AgentMaxRunDuration != 45*time.Second {
		t.Fatalf("unexpected AgentMaxRunDuration: %v", cfg.AgentMaxRunDuration)
	}
	if cfg.AgentMaxToolCallsPerRun != 7 {
		t.Fatalf("unexpected AgentMaxToolCallsPerRun: %d", cfg.AgentMaxToolCallsPerRun)
	}
	if len(cfg.ValidPersons) != 2 || cfg.ValidPersons[0] != "alice" || cfg.ValidPersons[1] != "bob" {
		t.Fatalf("unexpected valid persons: %#v", cfg.ValidPersons)
	}
}
