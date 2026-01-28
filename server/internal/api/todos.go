package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// AddTodoRequest represents a request to add a task.
type AddTodoRequest struct {
	Category string `json:"category"` // "work" or "priv"
	Text     string `json:"text"`     // optional - creates blank task if empty
}

// handleAddTodo adds a task to a category in today's daily note.
func (s *Server) handleAddTodo(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req AddTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	if req.Category == "" {
		writeBadRequest(w, "Category is required")
		return
	}
	if req.Category != "work" && req.Category != "priv" {
		writeBadRequest(w, "Invalid category")
		return
	}

	// Get or create today's daily note
	_, path, err := s.daily.GetOrCreateDaily(person, time.Now())
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	if err := s.daily.AddTask(person, path, req.Category, req.Text); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit changes
	_ = s.git.CommitAndPush("Add task")

	writeSuccess(w, "Task added")
}

// ToggleTodoRequest represents a request to toggle a task.
type ToggleTodoRequest struct {
	Path string `json:"path"`
	Line int    `json:"line"` // 1-indexed line number
}

// handleToggleTodo toggles a task's completion status.
func (s *Server) handleToggleTodo(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var req ToggleTodoRequest
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

	if err := s.daily.ToggleTask(person, req.Path, req.Line); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit changes
	_ = s.git.CommitAndPush("Toggle task")

	writeSuccess(w, "Task toggled")
}
