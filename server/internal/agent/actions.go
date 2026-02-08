package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"notes-editor/internal/vault"
)

const (
	defaultActionsPath = "agent/actions"
	maxPromptBytes     = 64 * 1024
)

var multiDash = regexp.MustCompile(`-+`)

// ActionMetadata represents parsed metadata from action front matter.
type ActionMetadata struct {
	RequiresConfirmation bool `json:"requires_confirmation"`
	MaxSteps             int  `json:"max_steps,omitempty"`
}

// Action represents one action file available to run.
type Action struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Path     string         `json:"path"`
	Metadata ActionMetadata `json:"metadata"`
}

type resolvedAction struct {
	Action
	Prompt string
}

func (s *Service) listActions(person string) ([]Action, error) {
	dirPath, err := vault.ResolvePath(s.store.RootPath(), person, defaultActionsPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Action{}, nil
		}
		return nil, err
	}

	actions := make([]Action, 0, len(entries))
	seenIDs := make(map[string]int)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isActionFile(name) {
			continue
		}

		filePath := filepath.Join(defaultActionsPath, name)
		content, err := s.store.ReadFile(person, filePath)
		if err != nil {
			return nil, err
		}
		meta, _, err := parseActionContent(content)
		if err != nil {
			return nil, fmt.Errorf("invalid action %q: %w", name, err)
		}

		label := actionLabel(name)
		id := slugify(label)
		if id == "" {
			continue
		}
		if seenIDs[id] > 0 {
			id = fmt.Sprintf("%s-%d", id, seenIDs[id]+1)
		}
		seenIDs[slugify(label)]++

		actions = append(actions, Action{
			ID:       id,
			Label:    label,
			Path:     filePath,
			Metadata: meta,
		})
	}

	return actions, nil
}

func (s *Service) resolveAction(person, actionID string) (*resolvedAction, error) {
	actions, err := s.listActions(person)
	if err != nil {
		return nil, err
	}
	for _, action := range actions {
		if action.ID != actionID {
			continue
		}
		content, err := s.store.ReadFile(person, action.Path)
		if err != nil {
			return nil, err
		}
		meta, prompt, err := parseActionContent(content)
		if err != nil {
			return nil, fmt.Errorf("invalid action %q: %w", action.Label, err)
		}
		if len(prompt) > maxPromptBytes {
			return nil, fmt.Errorf("action prompt exceeds max size (%d bytes)", maxPromptBytes)
		}
		action.Metadata = meta
		return &resolvedAction{
			Action: action,
			Prompt: prompt,
		}, nil
	}
	return nil, fmt.Errorf("action not found")
}

func parseActionContent(content string) (ActionMetadata, string, error) {
	meta := ActionMetadata{}
	if !strings.HasPrefix(content, "---\n") {
		return meta, strings.TrimSpace(content), nil
	}

	end := strings.Index(content[4:], "\n---\n")
	if end < 0 {
		return meta, "", fmt.Errorf("front matter not terminated")
	}
	end += 4

	frontMatter := content[4:end]
	body := content[end+5:]

	lines := strings.Split(frontMatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return meta, "", fmt.Errorf("invalid front matter line: %q", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "requires_confirmation":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return meta, "", fmt.Errorf("invalid requires_confirmation value")
			}
			meta.RequiresConfirmation = parsed
		case "max_steps":
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed <= 0 {
				return meta, "", fmt.Errorf("invalid max_steps value")
			}
			meta.MaxSteps = parsed
		}
	}

	return meta, strings.TrimSpace(body), nil
}

func isActionFile(name string) bool {
	return strings.HasSuffix(name, ".prompt.md") || strings.HasSuffix(name, ".md")
}

func actionLabel(name string) string {
	if strings.HasSuffix(name, ".prompt.md") {
		return strings.TrimSuffix(name, ".prompt.md")
	}
	return strings.TrimSuffix(name, ".md")
}

func slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return ""
	}

	var b strings.Builder
	for _, ch := range input {
		switch {
		case ch >= 'a' && ch <= 'z':
			b.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		case ch == '-' || ch == ' ' || ch == '_':
			b.WriteRune('-')
		}
	}
	out := multiDash.ReplaceAllString(b.String(), "-")
	return strings.Trim(out, "-")
}
