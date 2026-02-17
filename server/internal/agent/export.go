package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// SessionMarkdownExport describes one markdown export batch.
type SessionMarkdownExport struct {
	Directory string   `json:"directory"`
	Files     []string `json:"files"`
}

// ExportSessionsMarkdown exports all person-scoped sessions to markdown files.
func (s *Service) ExportSessionsMarkdown(person string) (*SessionMarkdownExport, error) {
	if s.store == nil {
		return nil, fmt.Errorf("vault store not configured")
	}

	sessions, err := s.ListSessions(person)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	baseDir := filepath.ToSlash(filepath.Join("agent", "session_exports", timestamp))

	files := make([]string, 0, len(sessions))
	indexBody := strings.Builder{}
	indexBody.WriteString("# Agent session export\n\n")
	indexBody.WriteString(fmt.Sprintf("- Exported at (UTC): %s\n", time.Now().UTC().Format(time.RFC3339)))
	indexBody.WriteString(fmt.Sprintf("- Sessions: %d\n\n", len(sessions)))

	for i, session := range sessions {
		items, historyErr := s.GetConversationHistory(person, session.SessionID)
		if historyErr != nil {
			return nil, historyErr
		}

		filename := exportSessionFilename(i+1, session.Name, session.SessionID)
		relPath := filepath.ToSlash(filepath.Join(baseDir, filename))
		body := renderSessionMarkdown(session, items)
		if err := s.store.WriteFile(person, relPath, body); err != nil {
			return nil, err
		}

		files = append(files, relPath)
		indexBody.WriteString(fmt.Sprintf("- [%s](%s)\n", session.Name, filename))
	}

	indexPath := filepath.ToSlash(filepath.Join(baseDir, "README.md"))
	if err := s.store.WriteFile(person, indexPath, indexBody.String()); err != nil {
		return nil, err
	}

	files = append([]string{indexPath}, files...)
	return &SessionMarkdownExport{
		Directory: baseDir,
		Files:     files,
	}, nil
}

func exportSessionFilename(position int, name, sessionID string) string {
	base := slugify(name)
	if base == "" {
		base = slugify(sessionID)
	}
	if base == "" {
		base = fmt.Sprintf("session-%d", position)
	}
	return fmt.Sprintf("%02d-%s.md", position, base)
}

func renderSessionMarkdown(summary SessionSummary, items []ConversationItem) string {
	var b strings.Builder

	b.WriteString("# ")
	if strings.TrimSpace(summary.Name) == "" {
		b.WriteString("Session")
	} else {
		b.WriteString(summary.Name)
	}
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("- Session ID: `%s`\n", summary.SessionID))
	b.WriteString(fmt.Sprintf("- Created (UTC): %s\n", summary.CreatedAt.UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Last used (UTC): %s\n", summary.LastUsedAt.UTC().Format(time.RFC3339)))
	b.WriteString("\n## Conversation\n\n")

	messageCount := 0
	for _, item := range items {
		if item.Type != ConversationItemMessage {
			continue
		}
		role := strings.TrimSpace(item.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}

		messageCount++
		if role == "user" {
			b.WriteString("### User\n\n")
		} else {
			b.WriteString("### Assistant\n\n")
		}
		b.WriteString(content)
		b.WriteString("\n\n")
	}

	if messageCount == 0 {
		b.WriteString("_No user/assistant messages found in this session._\n")
	}

	return b.String()
}
