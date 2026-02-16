package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"notes-editor/internal/claude"
)

const (
	defaultSessionNamePrefix = "Session"
	maxSessionNameLen        = 72
	maxSessionPreviewLen     = 140
	maxRecoveredSessions     = 30
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
	s.hydrateGatewayRecoveredSessions(person)

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

		items := s.getStoredConversation(person, rec.SessionID)
		if len(items) > 0 {
			msgCount := 0
			for _, item := range items {
				if item.Type == ConversationItemMessage {
					msgCount++
				}
			}
			summary.MessageCount = msgCount
			summary.LastPreview = historyPreviewItems(items)
			summaries = append(summaries, summary)
			continue
		}

		runtime := s.runtimes[rec.RuntimeMode]
		if runtime == nil || !runtime.Available() {
			summaries = append(summaries, summary)
			continue
		}

		var history []claude.ChatMessage
		var err error
		if rec.RuntimeMode == RuntimeModeGatewaySubscription {
			if piRuntime, ok := runtime.(*PiGatewayRuntime); ok {
				history, err = piRuntime.GetHistoryForPerson(person, rec.SessionID)
			} else {
				history, err = runtime.GetHistory(rec.SessionID)
			}
		} else {
			history, err = runtime.GetHistory(rec.SessionID)
		}
		if err == nil {
			summary.MessageCount = len(history)
			summary.LastPreview = historyPreview(history)
		}

		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (s *Service) hydrateGatewayRecoveredSessions(person string) {
	runtime := s.runtimes[RuntimeModeGatewaySubscription]
	if runtime == nil || !runtime.Available() {
		return
	}

	recovered := listGatewayRuntimeSessionFiles(person)
	if len(recovered) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	personSessions := s.sessionRecordsByPerson[person]
	if personSessions == nil {
		personSessions = make(map[string]*sessionRecord)
		s.sessionRecordsByPerson[person] = personSessions
	}

	for _, rec := range recovered {
		history, err := readGatewaySessionHistory(person, rec.SessionID)
		if err != nil || len(history) == 0 {
			continue
		}
		if _, exists := personSessions[rec.SessionID]; exists {
			continue
		}
		seq := s.sessionSequenceByPerson[person] + 1
		s.sessionSequenceByPerson[person] = seq
		personSessions[rec.SessionID] = &sessionRecord{
			SessionID:   rec.SessionID,
			Person:      person,
			Name:        buildSessionName("", seq),
			RuntimeMode: RuntimeModeGatewaySubscription,
			CreatedAt:   rec.Timestamp,
			LastUsedAt:  rec.Timestamp,
		}
	}
}

type recoveredRuntimeSession struct {
	SessionID string
	Timestamp time.Time
}

func listGatewayRuntimeSessionFiles(person string) []recoveredRuntimeSession {
	person = strings.TrimSpace(person)
	if person == "" {
		return nil
	}

	sessionDir := gatewaySessionDir()
	if sessionDir == "" {
		return nil
	}

	pattern := filepath.Join(sessionDir, person+"--*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}

	out := make([]recoveredRuntimeSession, 0, len(matches))
	for _, fullPath := range matches {
		base := filepath.Base(fullPath)
		if !strings.HasPrefix(base, person+"--") || !strings.HasSuffix(base, ".jsonl") {
			continue
		}
		sessionID := strings.TrimSuffix(strings.TrimPrefix(base, person+"--"), ".jsonl")
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			continue
		}

		ts := time.Now().UTC()
		if info, err := os.Stat(fullPath); err == nil {
			ts = info.ModTime().UTC()
		}
		out = append(out, recoveredRuntimeSession{
			SessionID: sessionID,
			Timestamp: ts,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	if len(out) > maxRecoveredSessions {
		out = out[:maxRecoveredSessions]
	}
	return out
}

func (s *Service) ClearAllSessions(person string) error {
	s.mu.Lock()
	personSessions := s.sessionRecordsByPerson[person]
	records := make([]sessionRecord, 0, len(personSessions))
	for _, rec := range personSessions {
		records = append(records, *rec)
	}
	delete(s.sessionRecordsByPerson, person)
	delete(s.conversationsByPerson, person)
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
