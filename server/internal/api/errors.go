// Package api provides HTTP handlers and middleware for the notes server.
package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an error response body.
type ErrorResponse struct {
	Detail string `json:"detail"`
}

// SuccessResponse represents a success response body.
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response with the given status code.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Detail: message})
}

// writeSuccess writes a success response.
func writeSuccess(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: message})
}

// writeBadRequest writes a 400 Bad Request error.
func writeBadRequest(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, message)
}

// writeUnauthorized writes a 401 Unauthorized error.
func writeUnauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "Unauthorized")
}

// writeNotFound writes a 404 Not Found error.
func writeNotFound(w http.ResponseWriter, message string) {
	writeError(w, http.StatusNotFound, message)
}

// StreamError represents an error in an NDJSON stream.
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// writeStreamError writes an error event to an NDJSON stream.
func writeStreamError(w http.ResponseWriter, message string) {
	data, _ := json.Marshal(StreamError{Type: "error", Message: message})
	w.Write(data)
	w.Write([]byte("\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
