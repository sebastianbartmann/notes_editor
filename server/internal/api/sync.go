package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type SyncRequest struct {
	Wait      bool `json:"wait"`
	TimeoutMs int  `json:"timeout_ms"`
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	var req SyncRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // optional body
	}

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	status := s.syncMgr.SyncNow(req.Wait, timeout)

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.syncMgr.Status())
}

func (s *Server) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	if s.indexMgr == nil {
		writeJSON(w, http.StatusOK, IndexStatus{})
		return
	}
	writeJSON(w, http.StatusOK, s.indexMgr.Status())
}
