package api

import (
	"encoding/json"
	"net/http"
	"os"
)

// handleListFiles lists files in a directory.
func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}

	entries, err := s.store.ListDir(person, path)
	if err != nil {
		if os.IsNotExist(err) {
			writeNotFound(w, "Directory not found")
			return
		}
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
	})
}

// handleReadFile reads a file's content.
func (s *Server) handleReadFile(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeBadRequest(w, "Path is required")
		return
	}

	content, err := s.store.ReadFile(person, path)
	if err != nil {
		if os.IsNotExist(err) {
			writeNotFound(w, "File not found")
			return
		}
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"content": content,
		"path":    path,
	})
}

// CreateFileRequest represents a request to create a file.
type CreateFileRequest struct {
	Path string `json:"path"`
}

// handleCreateFile creates an empty file.
func (s *Server) handleCreateFile(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req CreateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Path == "" {
		writeBadRequest(w, "Path is required")
		return
	}

	// Check if file already exists
	exists, err := s.store.FileExists(person, req.Path)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if exists {
		writeBadRequest(w, "File already exists")
		return
	}

	// Create empty file
	if err := s.store.WriteFile(person, req.Path, ""); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit changes
	_ = s.git.CommitAndPush("Create file")

	writeSuccess(w, "File created")
}

// SaveFileRequest represents a request to save a file.
type SaveFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// handleSaveFile saves content to a file.
func (s *Server) handleSaveFile(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req SaveFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Path == "" {
		writeBadRequest(w, "Path is required")
		return
	}

	if err := s.store.WriteFile(person, req.Path, req.Content); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit changes
	_ = s.git.CommitAndPush("Save file")

	writeSuccess(w, "File saved")
}

// DeleteFileRequest represents a request to delete a file.
type DeleteFileRequest struct {
	Path string `json:"path"`
}

// handleDeleteFile deletes a file.
func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req DeleteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Path == "" {
		writeBadRequest(w, "Path is required")
		return
	}

	if err := s.store.DeleteFile(person, req.Path); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit changes
	_ = s.git.CommitAndPush("Delete file")

	writeSuccess(w, "File deleted")
}

// UnpinEntryRequest represents a request to unpin an entry.
type UnpinEntryRequest struct {
	Path string `json:"path"`
	Line int    `json:"line"` // 1-indexed line number
}

// handleUnpinEntry removes the pinned marker from a specific line.
func (s *Server) handleUnpinEntry(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req UnpinEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Path == "" {
		writeBadRequest(w, "Path is required")
		return
	}
	if req.Line < 1 {
		writeBadRequest(w, "Line must be positive")
		return
	}

	if err := s.daily.UnpinEntry(person, req.Path, req.Line); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit changes
	_ = s.git.CommitAndPush("Unpin entry")

	writeSuccess(w, "Entry unpinned")
}
