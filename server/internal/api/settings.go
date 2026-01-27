package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// handleGetEnv returns the contents of the .env file.
func (s *Server) handleGetEnv(w http.ResponseWriter, r *http.Request) {
	envPath := filepath.Join(s.config.NotesRoot, "..", ".env")

	content, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]string{
				"content": "",
			})
			return
		}
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"content": string(content),
	})
}

// SetEnvRequest represents a request to update the .env file.
type SetEnvRequest struct {
	Content string `json:"content"`
}

// handleSetEnv updates the .env file contents.
func (s *Server) handleSetEnv(w http.ResponseWriter, r *http.Request) {
	var req SetEnvRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	envPath := filepath.Join(s.config.NotesRoot, "..", ".env")

	if err := os.WriteFile(envPath, []byte(req.Content), 0644); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeSuccess(w, "Settings saved")
}
