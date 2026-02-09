package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const sleepTimesFile = "sleep_times.md"

// SleepEntry represents a sleep time entry.
type SleepEntry struct {
	Line   int    `json:"line"`
	Date   string `json:"date"`
	Child  string `json:"child"`
	Time   string `json:"time"`
	Status string `json:"status"`
}

// handleGetSleepTimes returns recent sleep time entries.
func (s *Server) handleGetSleepTimes(w http.ResponseWriter, r *http.Request) {
	s.syncMgr.TriggerPullIfStale(30 * time.Second)

	s.mu.RLock()
	content, err := s.store.ReadRootFile(sleepTimesFile)
	s.mu.RUnlock()
	if err != nil {
		// File doesn't exist yet - return empty list
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": []SleepEntry{},
		})
		return
	}

	entries := parseSleepEntries(content)

	// Sort by date descending (most recent first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date > entries[j].Date
	})

	// Limit to 20 entries
	if len(entries) > 20 {
		entries = entries[:20]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
	})
}

// parseSleepEntries parses sleep entries from file content.
func parseSleepEntries(content string) []SleepEntry {
	var entries []SleepEntry
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse format: YYYY-MM-DD | ChildName | TimeEntry | Status
		parts := strings.Split(line, " | ")
		if len(parts) != 4 {
			continue
		}

		entries = append(entries, SleepEntry{
			Line:   i + 1, // 1-indexed
			Date:   strings.TrimSpace(parts[0]),
			Child:  strings.TrimSpace(parts[1]),
			Time:   strings.TrimSpace(parts[2]),
			Status: strings.TrimSpace(parts[3]),
		})
	}

	return entries
}

// AppendSleepRequest represents a request to add a sleep entry.
type AppendSleepRequest struct {
	Child  string `json:"child"`  // "Thomas" or "Fabian"
	Time   string `json:"time"`   // Time string
	Status string `json:"status"` // "eingeschlafen" or "aufgewacht"
}

// handleAppendSleepTime adds a new sleep time entry.
func (s *Server) handleAppendSleepTime(w http.ResponseWriter, r *http.Request) {
	var req AppendSleepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Child == "" {
		writeBadRequest(w, "Child is required")
		return
	}
	if req.Time == "" {
		writeBadRequest(w, "Time is required")
		return
	}
	if req.Status == "" {
		writeBadRequest(w, "Status is required")
		return
	}

	// Validate child
	if req.Child != "Thomas" && req.Child != "Fabian" {
		writeBadRequest(w, "Invalid child name")
		return
	}

	// Validate status
	if req.Status != "eingeschlafen" && req.Status != "aufgewacht" {
		writeBadRequest(w, "Invalid status")
		return
	}

	// Build entry line
	date := time.Now().Format("2006-01-02")
	entry := fmt.Sprintf("%s | %s | %s | %s\n", date, req.Child, req.Time, req.Status)

	s.mu.Lock()
	if err := s.store.AppendRootFile(sleepTimesFile, entry); err != nil {
		s.mu.Unlock()
		writeBadRequest(w, err.Error())
		return
	}
	s.mu.Unlock()

	s.syncMgr.TriggerPush("Add sleep entry")

	writeSuccess(w, "Entry added")
}

// DeleteSleepRequest represents a request to delete a sleep entry.
type DeleteSleepRequest struct {
	Line int `json:"line"` // 1-indexed line number
}

// handleDeleteSleepTime deletes a sleep time entry by line number.
func (s *Server) handleDeleteSleepTime(w http.ResponseWriter, r *http.Request) {
	var req DeleteSleepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

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

	// Remove the line (convert to 0-indexed)
	lines = append(lines[:req.Line-1], lines[req.Line:]...)

	// Write back
	newContent := strings.Join(lines, "\n")
	if err := s.store.WriteRootFile(sleepTimesFile, newContent); err != nil {
		s.mu.Unlock()
		writeBadRequest(w, err.Error())
		return
	}
	s.mu.Unlock()

	s.syncMgr.TriggerPush("Delete sleep entry")

	writeSuccess(w, "Entry deleted")
}
