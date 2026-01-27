package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AddTodoRequest represents a request to add a task.
type AddTodoRequest struct {
	Category string `json:"category"` // "work" or "priv"
	Text     string `json:"text"`     // optional - creates blank task if empty
}

// handleAddTodo adds a task to a category in today's daily note.
// Accepts both JSON and form-encoded requests for Android/React compatibility.
func (s *Server) handleAddTodo(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var category, text string

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var req AddTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "Invalid request body")
			return
		}
		category = req.Category
		text = req.Text
	} else {
		// Parse form data (application/x-www-form-urlencoded)
		if err := r.ParseForm(); err != nil {
			writeBadRequest(w, "Invalid form data")
			return
		}
		category = r.FormValue("category")
		text = r.FormValue("text")
	}

	if category == "" {
		writeBadRequest(w, "Category is required")
		return
	}
	if category != "work" && category != "priv" {
		writeBadRequest(w, "Invalid category")
		return
	}

	// Get or create today's daily note
	_, path, err := s.daily.GetOrCreateDaily(person, time.Now())
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	if err := s.daily.AddTask(person, path, category, text); err != nil {
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
// Accepts both JSON and form-encoded requests for Android/React compatibility.
func (s *Server) handleToggleTodo(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	var path string
	var line int

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var req ToggleTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "Invalid request body")
			return
		}
		path = req.Path
		line = req.Line
	} else {
		// Parse form data (application/x-www-form-urlencoded)
		if err := r.ParseForm(); err != nil {
			writeBadRequest(w, "Invalid form data")
			return
		}
		path = r.FormValue("path")
		lineStr := r.FormValue("line")
		var err error
		line, err = strconv.Atoi(lineStr)
		if err != nil {
			writeBadRequest(w, "Invalid line number")
			return
		}
	}

	if path == "" {
		writeBadRequest(w, "Path is required")
		return
	}
	if line < 1 {
		writeBadRequest(w, "Line must be positive")
		return
	}

	if err := s.daily.ToggleTask(person, path, line); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Commit changes
	_ = s.git.CommitAndPush("Toggle task")

	writeSuccess(w, "Task toggled")
}
