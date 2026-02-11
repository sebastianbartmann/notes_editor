package agent

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"notes-editor/internal/claude"
)

const (
	defaultSessionNamePrefix = "Session"
	maxSessionNameLen        = 72
	maxSessionPreviewLen     = 140
)

type sessionRecord struct {
	SessionID   string
	Person      string
	Name        string
	RuntimeMode string
	CreatedAt   time.Time
	LastUsedAt  time.Time
}

// SessionSummary contains person-scoped session metadata for session pickers.
type SessionSummary struct {
	SessionID    string    `json:"session_id"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	MessageCount int       `json:"message_count"`
	LastPreview  string    `json:"last_preview,omitempty"`
}

func (s *Service) touchSession(person, sessionID, initialMessage, runtimeMode string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	personSessions := s.sessionRecordsByPerson[person]
	if personSessions == nil {
		personSessions = make(map[string]*sessionRecord)
		s.sessionRecordsByPerson[person] = personSessions
	}

	if existing, ok := personSessions[sessionID]; ok {
		existing.LastUsedAt = now
		if existing.RuntimeMode == "" {
			existing.RuntimeMode = runtimeMode
		}
		return
	}

	seq := s.sessionSequenceByPerson[person] + 1
	s.sessionSequenceByPerson[person] = seq

	personSessions[sessionID] = &sessionRecord{
		SessionID:   sessionID,
		Person:      person,
		Name:        buildSessionName(initialMessage, seq),
		RuntimeMode: runtimeMode,
		CreatedAt:   now,
		LastUsedAt:  now,
	}
}

func (s *Service) removeSessionRecord(person, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	personSessions := s.sessionRecordsByPerson[person]
	if personSessions == nil {
		return
	}
	delete(personSessions, sessionID)
	if len(personSessions) == 0 {
		delete(s.sessionRecordsByPerson, person)
	}
}

func (s *Service) runtimeModeForSession(person, sessionID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	personSessions := s.sessionRecordsByPerson[person]
	if personSessions == nil {
		return "", false
	}
	record, ok := personSessions[sessionID]
	if !ok || record.RuntimeMode == "" {
		return "", false
	}
	return record.RuntimeMode, true
}

func (s *Service) ListSessions(person string) ([]SessionSummary, error) {
	s.mu.Lock()
	personSessions := s.sessionRecordsByPerson[person]
	records := make([]*sessionRecord, 0, len(personSessions))
	for _, rec := range personSessions {
		copyRec := *rec
		records = append(records, &copyRec)
	}
	s.mu.Unlock()

	sort.Slice(records, func(i, j int) bool {
		if records[i].LastUsedAt.Equal(records[j].LastUsedAt) {
			return records[i].CreatedAt.After(records[j].CreatedAt)
		}
		return records[i].LastUsedAt.After(records[j].LastUsedAt)
	})

	summaries := make([]SessionSummary, 0, len(records))
	for _, rec := range records {
		summary := SessionSummary{
			SessionID:  rec.SessionID,
			Name:       rec.Name,
			CreatedAt:  rec.CreatedAt,
			LastUsedAt: rec.LastUsedAt,
		}

		runtime := s.runtimes[rec.RuntimeMode]
		if runtime != nil && runtime.Available() {
			history, err := runtime.GetHistory(rec.SessionID)
			if err == nil {
				summary.MessageCount = len(history)
				summary.LastPreview = historyPreview(history)
			}
		}

		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (s *Service) ClearAllSessions(person string) error {
	s.mu.Lock()
	personSessions := s.sessionRecordsByPerson[person]
	records := make([]sessionRecord, 0, len(personSessions))
	for _, rec := range personSessions {
		records = append(records, *rec)
	}
	delete(s.sessionRecordsByPerson, person)
	s.mu.Unlock()

	var firstErr error
	for _, rec := range records {
		runtime := s.runtimes[rec.RuntimeMode]
		if runtime == nil || !runtime.Available() {
			continue
		}
		if err := runtime.ClearSession(rec.SessionID); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	s.mu.Lock()
	for key := range s.activeSessionRun {
		if strings.HasPrefix(key, person+"::") {
			delete(s.activeSessionRun, key)
		}
	}
	s.mu.Unlock()

	return firstErr
}

func buildSessionName(initialMessage string, seq int) string {
	normalized := normalizeSessionText(initialMessage)
	if normalized == "" {
		return fmt.Sprintf("%s %d", defaultSessionNamePrefix, seq)
	}
	if len(normalized) <= maxSessionNameLen {
		return normalized
	}
	trimmed := normalized[:maxSessionNameLen]
	if cut := strings.LastIndex(trimmed, " "); cut >= 16 {
		trimmed = trimmed[:cut]
	}
	return strings.TrimSpace(trimmed)
}

func normalizeSessionText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func historyPreview(history []claude.ChatMessage) string {
	for i := len(history) - 1; i >= 0; i-- {
		content := normalizeSessionText(history[i].Content)
		if content == "" {
			continue
		}
		if len(content) <= maxSessionPreviewLen {
			return content
		}
		trimmed := content[:maxSessionPreviewLen]
		if cut := strings.LastIndex(trimmed, " "); cut >= 20 {
			trimmed = trimmed[:cut]
		}
		return strings.TrimSpace(trimmed)
	}
	return ""
}
