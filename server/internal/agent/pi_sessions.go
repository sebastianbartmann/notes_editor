package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"notes-editor/internal/claude"
)

type piSessionLine struct {
	Type    string            `json:"type"`
	Message *piSessionMessage `json:"message,omitempty"`
}

type piSessionMessage struct {
	Role    string                 `json:"role"`
	Content []piSessionContentPart `json:"content,omitempty"`
}

type piSessionContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func gatewaySessionDir() string {
	sessionDir := strings.TrimSpace(os.Getenv("PI_GATEWAY_PI_SESSION_DIR"))
	if sessionDir != "" {
		return sessionDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "notes-editor-sessions")
}

func gatewaySessionFilePath(person, sessionID string) string {
	person = strings.TrimSpace(person)
	sessionID = strings.TrimSpace(sessionID)
	if person == "" || sessionID == "" {
		return ""
	}
	sessionDir := gatewaySessionDir()
	if sessionDir == "" {
		return ""
	}
	return filepath.Join(sessionDir, person+"--"+sessionID+".jsonl")
}

func readGatewaySessionHistory(person, sessionID string) ([]claude.ChatMessage, error) {
	path := gatewaySessionFilePath(person, sessionID)
	if path == "" {
		return []claude.ChatMessage{}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []claude.ChatMessage{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var history []claude.ChatMessage
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event piSessionLine
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type != "message" || event.Message == nil {
			continue
		}

		role := strings.TrimSpace(event.Message.Role)
		if role != "user" && role != "assistant" {
			continue
		}

		var textParts []string
		for _, part := range event.Message.Content {
			if part.Type != "text" || part.Text == "" {
				continue
			}
			textParts = append(textParts, part.Text)
		}
		if len(textParts) == 0 {
			continue
		}

		history = append(history, claude.ChatMessage{
			Role:    role,
			Content: strings.Join(textParts, ""),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return history, nil
}
