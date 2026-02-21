package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"notes-editor/internal/sleep"
)

const sleepTimesFile = "sleep_times.md"

// SleepEntry represents a sleep time entry.
type SleepEntry struct {
	ID         string `json:"id"`
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
	for _, e := range entries {
		out = append(out, mapSleepEntry(e))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": out,
	})
}

func mapSleepEntry(e sleep.Entry) SleepEntry {
	timeText := ""
	dateText := ""
	occurredAt := ""
	if e.OccurredAt != nil {
		loc, _ := time.LoadLocation("Europe/Vienna")
		local := e.OccurredAt.In(loc)
		dateText = local.Format("2006-01-02")
		occurredAt = e.OccurredAt.Format(time.RFC3339)
		timeText = local.Format("15:04")
	}
	if dateText == "" {
		dateText = e.CreatedAt.Format("2006-01-02")
	}

	return SleepEntry{
		ID:         e.ID,
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
	occurredAt, err := parseIncomingOccurredAt(req.OccurredAt)
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

	_, err = store.CreateEntry(req.Child, status, occurredAt, req.Notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add sleep entry")
		return
	}

	_ = s.refreshSleepMarkdownBackup()
	s.syncMgr.TriggerPush("Sleep entry added")
	writeSuccess(w, "Entry added")

}

type UpdateSleepRequest struct {
	ID         string `json:"id"`
	Child      string `json:"child"`
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
	occurredAt, err := parseIncomingOccurredAt(req.OccurredAt)
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

	if err := store.UpdateEntry(req.ID, req.Child, status, occurredAt, req.Notes); err != nil {
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
	ID string `json:"id"`
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
		writeBadRequest(w, "ID is required")
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

func parseIncomingOccurredAt(occurredAtRaw string) (*time.Time, error) {
	if strings.TrimSpace(occurredAtRaw) == "" {
		return nil, errors.New("occurred_at is required")
	}
	parsed, err := sleep.ParseOccurredAtISO(occurredAtRaw)
	if err != nil {
		return nil, errors.New("occurred_at must be RFC3339")
	}
	return parsed, nil
}

func sleepDBPath(notesRoot string) string {
	return filepath.Join(notesRoot, "sleep.db")
}

func sleepMarkdownExists(notesRoot string) bool {
	_, err := os.Stat(filepath.Join(notesRoot, sleepTimesFile))
	return err == nil
}
