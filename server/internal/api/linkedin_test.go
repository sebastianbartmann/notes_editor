package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLinkedInHealthNoToken(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	req := makeRequest(t, http.MethodGet, "/api/linkedin/health", "", "sebastian")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["token_present"] != false {
		t.Fatalf("expected token_present=false, got %v", payload["token_present"])
	}
	if payload["configured"] != false {
		t.Fatalf("expected configured=false, got %v", payload["configured"])
	}
	if payload["healthy"] != false {
		t.Fatalf("expected healthy=false, got %v", payload["healthy"])
	}
}
