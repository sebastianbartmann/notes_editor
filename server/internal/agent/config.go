package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"notes-editor/internal/claude"
)

const (
	defaultPromptPath  = "agents.md"
	internalConfigPath = "agent/config.json"
	defaultRuntimeMode = RuntimeModeGatewaySubscription
)

// Config is per-person agent config.
type Config struct {
	RuntimeMode string `json:"runtime_mode"`
	PromptPath  string `json:"prompt_path"`
	ActionsPath string `json:"actions_path"`
	Prompt      string `json:"prompt"`
}

// ConfigUpdate is the mutable subset of agent config.
type ConfigUpdate struct {
	RuntimeMode *string `json:"runtime_mode,omitempty"`
	Prompt      *string `json:"prompt,omitempty"`
}

type persistedConfig struct {
	RuntimeMode string `json:"runtime_mode"`
}

func (s *Service) getConfig(person string) (*Config, error) {
	pc, err := s.readPersistedConfig(person)
	if err != nil {
		return nil, err
	}

	prompt, err := s.store.ReadFile(person, defaultPromptPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		prompt = claude.SystemPrompt
	}

	return &Config{
		RuntimeMode: pc.RuntimeMode,
		PromptPath:  defaultPromptPath,
		ActionsPath: defaultActionsPath,
		Prompt:      prompt,
	}, nil
}

func (s *Service) saveConfig(person string, update ConfigUpdate) (*Config, error) {
	pc, err := s.readPersistedConfig(person)
	if err != nil {
		return nil, err
	}

	if update.RuntimeMode != nil && *update.RuntimeMode != "" {
		mode := strings.TrimSpace(*update.RuntimeMode)
		if !isValidRuntimeMode(mode) {
			return nil, fmt.Errorf("invalid runtime_mode %q", mode)
		}
		pc.RuntimeMode = mode
	}
	data, err := json.MarshalIndent(pc, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := s.store.WriteFile(person, internalConfigPath, string(data)); err != nil {
		return nil, err
	}

	if update.Prompt != nil {
		if err := s.store.WriteFile(person, defaultPromptPath, *update.Prompt); err != nil {
			return nil, err
		}
	}

	return s.getConfig(person)
}

func (s *Service) readPersistedConfig(person string) (*persistedConfig, error) {
	pc := &persistedConfig{RuntimeMode: defaultRuntimeMode}
	content, err := s.store.ReadFile(person, internalConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return pc, nil
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(content), pc); err != nil {
		return nil, err
	}
	if pc.RuntimeMode == "" || !isValidRuntimeMode(pc.RuntimeMode) {
		pc.RuntimeMode = defaultRuntimeMode
	}
	return pc, nil
}

func isValidRuntimeMode(mode string) bool {
	switch mode {
	case RuntimeModeAnthropicAPIKey, RuntimeModeGatewaySubscription:
		return true
	default:
		return false
	}
}
