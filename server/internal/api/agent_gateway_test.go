package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentGatewayHealth(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":   true,
			"mode": "claude_cli",
		})
	}))
	defer gateway.Close()

	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	srv.config.PiGatewayURL = gateway.URL
	router := NewRouter(srv)

	req := makeRequest(t, "GET", "/api/agent/gateway/health", "", "sebastian")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload struct {
		Configured bool   `json:"configured"`
		Reachable  bool   `json:"reachable"`
		Healthy    bool   `json:"healthy"`
		Mode       string `json:"mode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.Configured || !payload.Reachable || !payload.Healthy {
		t.Fatalf("unexpected health payload: %+v", payload)
	}
	if payload.Mode != "claude_cli" {
		t.Fatalf("unexpected mode: %q", payload.Mode)
	}
}
