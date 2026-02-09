package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleGetDaily returns today's daily note, creating it if necessary.
func (s *Server) handleGetDaily(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	// Keep the daily view fresh. We only block for sync when the last pull is stale,
	// so repeated navigations stay fast.
	st := s.syncMgr.Status()
	if st.LastPullAt.IsZero() || time.Since(st.LastPullAt) >= 30*time.Second {
		_ = s.syncMgr.SyncNow(true, 2*time.Second)
	} else {
		s.syncMgr.TriggerPullIfStale(30 * time.Second)
	}

	s.mu.Lock()
	content, path, created, err := s.daily.GetOrCreateDaily(person, time.Now())
	s.mu.Unlock()
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit only if a new file was created. (Avoid expensive git work on the read path.)
	if created {
		s.syncMgr.TriggerPush("Daily note created")
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"date":    time.Now().Format("2006-01-02"),
		"content": content,
		"path":    path,
	})
}

// SaveRequest represents a request to save content.
type SaveRequest struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// handleSaveDaily saves the daily note content.
func (s *Server) handleSaveDaily(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Path == "" {
		writeBadRequest(w, "Path is required")
		return
	}

	s.mu.Lock()
	if err := s.store.WriteFile(person, req.Path, req.Content); err != nil {
		s.mu.Unlock()
		writeBadRequest(w, err.Error())
		return
	}
	s.mu.Unlock()

	// Commit/push in the background.
	s.syncMgr.TriggerPush("Save note")

	writeSuccess(w, "Saved")
}

// AppendRequest represents a request to append an entry.
type AppendRequest struct {
	Path   string `json:"path"`
	Text   string `json:"text"`
	Pinned bool   `json:"pinned"`
}

// handleAppendDaily appends a timestamped entry to the daily note.
func (s *Server) handleAppendDaily(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req AppendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Path == "" {
		writeBadRequest(w, "Path is required")
		return
	}
	if req.Text == "" {
		writeBadRequest(w, "Text is required")
		return
	}

	s.mu.Lock()
	if err := s.daily.AppendEntry(person, req.Path, req.Text, req.Pinned); err != nil {
		s.mu.Unlock()
		writeBadRequest(w, err.Error())
		return
	}
	s.mu.Unlock()

	s.syncMgr.TriggerPush("Append entry")

	writeSuccess(w, "Appended")
}

// ClearPinnedRequest represents a request to clear pinned markers.
type ClearPinnedRequest struct {
	Path string `json:"path"`
}

// handleClearPinned removes all pinned markers from a note.
func (s *Server) handleClearPinned(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req ClearPinnedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Path == "" {
		writeBadRequest(w, "Path is required")
		return
	}

	s.mu.Lock()
	if err := s.daily.ClearAllPinned(person, req.Path); err != nil {
		s.mu.Unlock()
		writeBadRequest(w, err.Error())
		return
	}
	s.mu.Unlock()

	s.syncMgr.TriggerPush("Clear pinned")

	writeSuccess(w, "Cleared")
}
