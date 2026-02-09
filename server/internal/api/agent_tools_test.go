package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentToolExecute(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	reqBody := `{"tool":"read_file","args":{"path":"notes/secret.md"}}`
	req := makeRequest(t, "POST", "/api/agent/tools/execute", reqBody, "sebastian")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Content string `json:"content"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true, got ok=false error=%q", payload.Error)
	}
	if payload.Content == "" {
		t.Fatalf("expected non-empty content")
	}
}

func TestAgentToolExecuteRequiresToolName(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	req := makeRequest(t, "POST", "/api/agent/tools/execute", `{}`, "sebastian")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
