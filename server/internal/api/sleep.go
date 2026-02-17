package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"notes-editor/internal/sleep"
)

const sleepTimesFile = "sleep_times.md"

// SleepEntry represents a sleep time entry.
type SleepEntry struct {
	ID         string `json:"id"`
	Line       int    `json:"line"`
	Date       string `json:"date"`
	Child      string `json:"child"`
	Time       string `json:"time"`
	Status     string `json:"status"`
	Notes      string `json:"notes,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

type SleepNightSummary struct {
	NightDate      string `json:"night_date"`
	Child          string `json:"child"`
	DurationMinute int    `json:"duration_minutes"`
	Bedtime        string `json:"bedtime"`
	WakeTime       string `json:"wake_time"`
}

type SleepAverageSummary struct {
	Days            int    `json:"days"`
	Child           string `json:"child"`
	AverageBedtime  string `json:"average_bedtime"`
	AverageWakeTime string `json:"average_wake_time"`
}

func (s *Server) ensureSleepStoreMigrated() {
	if s.sleepStore == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sleepMigrated {
		return
	}
	content, err := s.store.ReadRootFile(sleepTimesFile)
	if err != nil {
		s.sleepMigrated = true
		return
	}
	if _, err := s.sleepStore.ImportMarkdownIfNeeded(content); err != nil {
		// Keep serving API even if migration fails; clients can still append directly to DB.
	}
	s.sleepMigrated = true
}

// handleGetSleepTimes returns recent sleep time entries.
func (s *Server) handleGetSleepTimes(w http.ResponseWriter, r *http.Request) {
	s.ensureSleepStoreMigrated()

	s.mu.RLock()
	store := s.sleepStore
	s.mu.RUnlock()

	if store == nil {
		writeError(w, http.StatusInternalServerError, "Sleep storage not initialized")
		return
	}

	entries, err := store.ListEntries(200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load sleep entries")
		return
	}

	out := make([]SleepEntry, 0, len(entries))
	for i, e := range entries {
		out = append(out, mapSleepEntry(e, i+1))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": out,
	})
}

func mapSleepEntry(e sleep.Entry, line int) SleepEntry {
	timeText := strings.TrimSpace(e.TimeText)
	dateText := ""
	occurredAt := ""
	if e.OccurredAt != nil {
		loc, _ := time.LoadLocation("Europe/Vienna")
		local := e.OccurredAt.In(loc)
		dateText = local.Format("2006-01-02")
		occurredAt = e.OccurredAt.Format(time.RFC3339)
		if timeText == "" {
			timeText = local.Format("15:04")
		}
	}
	if dateText == "" {
		dateText = e.CreatedAt.Format("2006-01-02")
	}

	return SleepEntry{
		ID:         e.ID,
		Line:       line,
		Date:       dateText,
		Child:      e.Child,
		Time:       timeText,
		Status:     sleep.DisplayStatus(e.Status),
		Notes:      e.Notes,
		OccurredAt: occurredAt,
	}
}

// handleGetSleepSummary returns aggregated nightly durations and average bed/wake times.
func (s *Server) handleGetSleepSummary(w http.ResponseWriter, r *http.Request) {
	s.ensureSleepStoreMigrated()

	s.mu.RLock()
	store := s.sleepStore
	s.mu.RUnlock()
	if store == nil {
		writeError(w, http.StatusInternalServerError, "Sleep storage not initialized")
		return
	}

	summary, err := store.BuildSummary()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to build sleep summary")
		return
	}

	nights := make([]SleepNightSummary, 0, len(summary.Nights))
	for _, n := range summary.Nights {
		nights = append(nights, SleepNightSummary{
			NightDate:      n.NightDate,
			Child:          n.Child,
			DurationMinute: n.DurationMinute,
			Bedtime:        n.Bedtime,
			WakeTime:       n.WakeTime,
		})
	}
	averages := make([]SleepAverageSummary, 0, len(summary.Averages))
	for _, a := range summary.Averages {
		averages = append(averages, SleepAverageSummary{
			Days:            a.Days,
			Child:           a.Child,
			AverageBedtime:  a.AverageBedtime,
			AverageWakeTime: a.AverageWakeTime,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nights":   nights,
		"averages": averages,
	})
}

// AppendSleepRequest represents a request to add a sleep entry.
type AppendSleepRequest struct {
	Child      string `json:"child"`
	Time       string `json:"time"`
	Status     string `json:"status"`
	OccurredAt string `json:"occurred_at"`
	Notes      string `json:"notes"`
}

// handleAppendSleepTime adds a new sleep time entry.
func (s *Server) handleAppendSleepTime(w http.ResponseWriter, r *http.Request) {
	s.ensureSleepStoreMigrated()

	var req AppendSleepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Child == "" {
		writeBadRequest(w, "Child is required")
		return
	}
	if !sleep.IsValidChild(req.Child) {
		writeBadRequest(w, "Invalid child name")
		return
	}
	if req.Status == "" {
		writeBadRequest(w, "Status is required")
		return
	}
	status, err := sleep.NormalizeStatus(req.Status)
	if err != nil {
		writeBadRequest(w, "Invalid status")
		return
	}
	if strings.TrimSpace(req.Time) == "" && strings.TrimSpace(req.OccurredAt) == "" {
		writeBadRequest(w, "Time is required")
		return
	}

	occurredAt, err := parseIncomingOccurredAt(req.OccurredAt, req.Time)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	s.mu.RLock()
	store := s.sleepStore
	s.mu.RUnlock()
	if store == nil {
		writeError(w, http.StatusInternalServerError, "Sleep storage not initialized")
		return
	}

	entry, err := store.CreateEntry(req.Child, status, occurredAt, req.Time, req.Notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add sleep entry")
		return
	}

	_ = s.refreshSleepMarkdownBackup()
	s.syncMgr.TriggerPush("Sleep entry added")
	writeSuccess(w, "Entry added")

	_ = entry
}

type UpdateSleepRequest struct {
	ID         string `json:"id"`
	Child      string `json:"child"`
	Time       string `json:"time"`
	Status     string `json:"status"`
	OccurredAt string `json:"occurred_at"`
	Notes      string `json:"notes"`
}

func (s *Server) handleUpdateSleepTime(w http.ResponseWriter, r *http.Request) {
	s.ensureSleepStoreMigrated()

	var req UpdateSleepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeBadRequest(w, "ID is required")
		return
	}
	if !sleep.IsValidChild(req.Child) {
		writeBadRequest(w, "Invalid child name")
		return
	}
	status, err := sleep.NormalizeStatus(req.Status)
	if err != nil {
		writeBadRequest(w, "Invalid status")
		return
	}
	if strings.TrimSpace(req.Time) == "" && strings.TrimSpace(req.OccurredAt) == "" {
		writeBadRequest(w, "Time is required")
		return
	}

	occurredAt, err := parseIncomingOccurredAt(req.OccurredAt, req.Time)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	s.mu.RLock()
	store := s.sleepStore
	s.mu.RUnlock()
	if store == nil {
		writeError(w, http.StatusInternalServerError, "Sleep storage not initialized")
		return
	}

	if err := store.UpdateEntry(req.ID, req.Child, status, occurredAt, req.Time, req.Notes); err != nil {
		if errors.Is(err, sleep.ErrNotFound) {
			writeNotFound(w, "Sleep entry not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update sleep entry")
		return
	}

	_ = s.refreshSleepMarkdownBackup()
	s.syncMgr.TriggerPush("Sleep entry updated")
	writeSuccess(w, "Entry updated")
}

// DeleteSleepRequest represents a request to delete a sleep entry.
type DeleteSleepRequest struct {
	ID   string `json:"id"`
	Line int    `json:"line"`
}

// handleDeleteSleepTime deletes a sleep time entry by ID.
func (s *Server) handleDeleteSleepTime(w http.ResponseWriter, r *http.Request) {
	s.ensureSleepStoreMigrated()

	var req DeleteSleepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.ID) == "" {
		// Legacy compatibility path: line-number deletion from markdown file.
		if req.Line < 1 {
			writeBadRequest(w, "Line must be positive")
			return
		}
		s.mu.Lock()
		content, err := s.store.ReadRootFile(sleepTimesFile)
		if err != nil {
			s.mu.Unlock()
			writeNotFound(w, "Sleep times file not found")
			return
		}
		lines := strings.Split(content, "\n")
		if req.Line > len(lines) {
			s.mu.Unlock()
			writeBadRequest(w, "Line number out of range")
			return
		}
		lines = append(lines[:req.Line-1], lines[req.Line:]...)
		if err := s.store.WriteRootFile(sleepTimesFile, strings.Join(lines, "\n")); err != nil {
			s.mu.Unlock()
			writeBadRequest(w, err.Error())
			return
		}
		s.sleepMigrated = false
		s.mu.Unlock()
		s.syncMgr.TriggerPush("Sleep entry deleted")
		writeSuccess(w, "Entry deleted")
		return
	}

	s.mu.RLock()
	store := s.sleepStore
	s.mu.RUnlock()
	if store == nil {
		writeError(w, http.StatusInternalServerError, "Sleep storage not initialized")
		return
	}

	if err := store.SoftDeleteEntry(req.ID); err != nil {
		if errors.Is(err, sleep.ErrNotFound) {
			writeNotFound(w, "Sleep entry not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete sleep entry")
		return
	}

	_ = s.refreshSleepMarkdownBackup()
	s.syncMgr.TriggerPush("Sleep entry deleted")
	writeSuccess(w, "Entry deleted")
}

func (s *Server) handleExportSleepMarkdown(w http.ResponseWriter, r *http.Request) {
	s.ensureSleepStoreMigrated()

	s.mu.RLock()
	store := s.sleepStore
	s.mu.RUnlock()
	if store == nil {
		writeError(w, http.StatusInternalServerError, "Sleep storage not initialized")
		return
	}

	content, err := store.ExportMarkdown()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to export sleep markdown")
		return
	}

	s.mu.Lock()
	writeErr := s.store.WriteRootFile(sleepTimesFile, content)
	s.mu.Unlock()
	if writeErr != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write sleep_times.md")
		return
	}

	s.syncMgr.TriggerPush("Export sleep markdown backup")
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Sleep data exported to markdown",
		"path":    sleepTimesFile,
	})
}

func (s *Server) refreshSleepMarkdownBackup() error {
	s.mu.RLock()
	store := s.sleepStore
	s.mu.RUnlock()
	if store == nil {
		return nil
	}
	content, err := store.ExportMarkdown()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.WriteRootFile(sleepTimesFile, content)
}

func parseIncomingOccurredAt(occurredAtRaw, timeRaw string) (*time.Time, error) {
	if strings.TrimSpace(occurredAtRaw) != "" {
		parsed, err := sleep.ParseOccurredAtISO(occurredAtRaw)
		if err != nil {
			return nil, fmt.Errorf("occurred_at must be RFC3339")
		}
		return parsed, nil
	}
	if strings.TrimSpace(timeRaw) == "" {
		return nil, nil
	}
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		return nil, err
	}
	nowLocal := time.Now().In(loc)
	dateText := nowLocal.Format("2006-01-02")
	if parsed, ok := sleep.ParseOccurredAt(dateText, timeRaw, loc); ok {
		return parsed, nil
	}
	return nil, nil
}

// parseSleepEntries parses sleep entries from file content.
// Kept for legacy tests/compatibility; DB now stores canonical data.
func parseSleepEntries(content string) []SleepEntry {
	var entries []SleepEntry
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, " | ")
		if len(parts) != 4 {
			continue
		}
		entries = append(entries, SleepEntry{
			Line:   i + 1,
			Date:   strings.TrimSpace(parts[0]),
			Child:  strings.TrimSpace(parts[1]),
			Time:   strings.TrimSpace(parts[2]),
			Status: strings.TrimSpace(parts[3]),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date > entries[j].Date
	})
	return entries
}

func sleepDBPath(notesRoot string) string {
	return filepath.Join(notesRoot, "sleep.db")
}

func sleepMarkdownExists(notesRoot string) bool {
	_, err := os.Stat(filepath.Join(notesRoot, sleepTimesFile))
	return err == nil
}
