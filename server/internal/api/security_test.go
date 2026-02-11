package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"notes-editor/internal/config"
)

// setupTestServer creates a test server with a temporary vault directory.
func setupTestServer(t *testing.T) (*Server, string, func()) {
	t.Helper()

	// Create temp directory for vault
	vaultRoot := t.TempDir()

	// Create person directories
	os.MkdirAll(filepath.Join(vaultRoot, "sebastian", "daily"), 0755)
	os.MkdirAll(filepath.Join(vaultRoot, "sebastian", "notes"), 0755)
	os.MkdirAll(filepath.Join(vaultRoot, "petra", "daily"), 0755)
	os.MkdirAll(filepath.Join(vaultRoot, "petra", "notes"), 0755)

	// Create test files
	os.WriteFile(filepath.Join(vaultRoot, "sebastian", "notes", "secret.md"),
		[]byte("# Sebastian's Secret\nThis is private."), 0644)
	os.WriteFile(filepath.Join(vaultRoot, "petra", "notes", "secret.md"),
		[]byte("# Petra's Secret\nThis is also private."), 0644)

	// Create a file outside person directories (at vault root)
	os.WriteFile(filepath.Join(vaultRoot, "sleep_times.md"),
		[]byte("# Sleep Times\n2024-01-15 | Thomas | 19:30 | eingeschlafen"), 0644)

	cfg := &config.Config{
		NotesRoot:  vaultRoot,
		NotesToken: "test-token-123",
	}

	srv := NewServer(cfg)

	cleanup := func() {
		// Temp dir is automatically cleaned up by t.TempDir()
	}

	return srv, vaultRoot, cleanup
}

// makeRequest creates an authenticated request with person header.
func makeRequest(t *testing.T, method, path string, body string, person string) *http.Request {
	t.Helper()

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	req.Header.Set("Authorization", "Bearer test-token-123")
	if person != "" {
		req.Header.Set("X-Notes-Person", person)
	}

	return req
}

func TestPathTraversalPrevention(t *testing.T) {
	srv, vaultRoot, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	// Create a sensitive file outside the vault to test traversal
	sensitiveFile := filepath.Join(vaultRoot, "..", "sensitive.txt")
	os.WriteFile(sensitiveFile, []byte("SENSITIVE DATA"), 0644)
	defer os.Remove(sensitiveFile)

	traversalPaths := []string{
		"../sensitive.txt",
		"../../etc/passwd",
		"notes/../../../etc/passwd",
		"..%2F..%2Fetc%2Fpasswd", // URL encoded
		"....//....//etc/passwd",
		"notes/../../petra/notes/secret.md", // Cross-person access
	}

	for _, path := range traversalPaths {
		t.Run("read_"+path, func(t *testing.T) {
			req := makeRequest(t, "GET", "/api/files/read?path="+path, "", "sebastian")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			// Should be 400 Bad Request or 404 Not Found, never 200 with content
			if rec.Code == http.StatusOK {
				t.Errorf("path traversal should be blocked: %s", path)
			}
		})
	}

	// Test path traversal in write operations
	for _, path := range traversalPaths[:3] {
		t.Run("create_"+path, func(t *testing.T) {
			body := `{"path":"` + path + `"}`
			req := makeRequest(t, "POST", "/api/files/create", body, "sebastian")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				t.Errorf("path traversal in create should be blocked: %s", path)
			}
		})

		t.Run("save_"+path, func(t *testing.T) {
			body := `{"path":"` + path + `","content":"malicious"}`
			req := makeRequest(t, "POST", "/api/files/save", body, "sebastian")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				t.Errorf("path traversal in save should be blocked: %s", path)
			}
		})

		t.Run("delete_"+path, func(t *testing.T) {
			body := `{"path":"` + path + `"}`
			req := makeRequest(t, "POST", "/api/files/delete", body, "sebastian")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				t.Errorf("path traversal in delete should be blocked: %s", path)
			}
		})
	}
}

func TestPersonContextIsolation(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	// Sebastian should be able to read his own files
	t.Run("sebastian can read own file", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/read?path=notes/secret.md", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		content, _ := resp["content"].(string)
		if !strings.Contains(content, "Sebastian's Secret") {
			t.Error("should read Sebastian's file content")
		}
	})

	// Petra should be able to read her own files
	t.Run("petra can read own file", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/read?path=notes/secret.md", "", "petra")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		content, _ := resp["content"].(string)
		if !strings.Contains(content, "Petra's Secret") {
			t.Error("should read Petra's file content")
		}
	})

	// Sebastian should NOT be able to read Petra's files via path traversal
	t.Run("sebastian cannot access petra via traversal", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/read?path=../petra/notes/secret.md", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			content, _ := resp["content"].(string)
			if strings.Contains(content, "Petra's Secret") {
				t.Error("sebastian should NOT be able to read petra's files")
			}
		}
	})

	// Sebastian cannot write to Petra's directory
	t.Run("sebastian cannot write to petra", func(t *testing.T) {
		body := `{"path":"../petra/notes/hacked.md","content":"pwned"}`
		req := makeRequest(t, "POST", "/api/files/save", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Error("sebastian should NOT be able to write to petra's directory")
		}
	})

	// Sebastian cannot delete Petra's files
	t.Run("sebastian cannot delete petra files", func(t *testing.T) {
		body := `{"path":"../petra/notes/secret.md"}`
		req := makeRequest(t, "POST", "/api/files/delete", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Error("sebastian should NOT be able to delete petra's files")
		}
	})
}

func TestAuthenticationRequired(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/api/daily", ""},
		{"POST", "/api/save", `{"content":"test"}`},
		{"POST", "/api/append", `{"content":"test"}`},
		{"POST", "/api/clear-pinned", ""},
		{"POST", "/api/todos/add", `{"category":"work"}`},
		{"POST", "/api/todos/toggle", `{"path":"daily/2024-01-15.md","line":5}`},
		{"GET", "/api/sleep-times", ""},
		{"POST", "/api/sleep-times/append", `{"child":"Thomas","entry":"19:30"}`},
		{"POST", "/api/sleep-times/delete", `{"line":1}`},
		{"GET", "/api/files/list?path=.", ""},
		{"GET", "/api/files/read?path=notes/test.md", ""},
		{"POST", "/api/files/create", `{"path":"test.md"}`},
		{"POST", "/api/files/save", `{"path":"test.md","content":"test"}`},
		{"POST", "/api/files/delete", `{"path":"test.md"}`},
		{"POST", "/api/claude/chat", `{"message":"hi"}`},
		{"POST", "/api/claude/chat-stream", `{"message":"hi"}`},
		{"POST", "/api/claude/clear", `{"session_id":"abc"}`},
		{"GET", "/api/claude/history?session_id=abc", ""},
		{"POST", "/api/agent/chat", `{"message":"hi"}`},
		{"POST", "/api/agent/chat-stream", `{"message":"hi"}`},
		{"POST", "/api/agent/session/clear", `{"session_id":"abc"}`},
		{"GET", "/api/agent/session/history?session_id=abc", ""},
		{"POST", "/api/agent/stop", `{"run_id":"abc"}`},
		{"GET", "/api/agent/config", ""},
		{"POST", "/api/agent/config", `{"runtime_mode":"anthropic_api_key"}`},
		{"GET", "/api/agent/actions", ""},
		{"POST", "/api/agent/actions/test/run", `{}`},
		{"POST", "/api/agent/tools/execute", `{"tool":"read_file","args":{"path":"notes/test.md"}}`},
		{"GET", "/api/agent/gateway/health", ""},
		{"GET", "/api/settings/env", ""},
		{"POST", "/api/settings/env", `{"content":"KEY=value"}`},
		{"GET", "/api/linkedin/health", ""},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			// No Authorization header
			req.Header.Set("X-Notes-Person", "sebastian")

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("endpoint without auth should return 401, got %d", rec.Code)
			}
		})
	}
}

func TestLinkedInCallbackSkipsAuth(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	// LinkedIn callback should not require auth (OAuth flow)
	req := httptest.NewRequest("GET", "/api/linkedin/oauth/callback?code=test-code&state=test", nil)
	// No Authorization header
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should not be 401 (might be other error due to invalid OAuth, but not auth error)
	if rec.Code == http.StatusUnauthorized {
		t.Error("LinkedIn callback should not require authentication")
	}
}

func TestInvalidPersonRejected(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	invalidPersons := []string{
		"hacker",
		"admin",
		"root",
		"Sebastian",  // Case sensitive
		"PETRA",      // Case sensitive
		"sebastian ", // Trailing space
		" sebastian", // Leading space
	}

	for _, person := range invalidPersons {
		t.Run(person, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/daily", nil)
			req.Header.Set("Authorization", "Bearer test-token-123")
			req.Header.Set("X-Notes-Person", person)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("invalid person %q should return 400, got %d", person, rec.Code)
			}

			var errResp ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			if errResp.Detail != "Invalid person" {
				t.Errorf("error detail = %q, want %q", errResp.Detail, "Invalid person")
			}
		})
	}
}

func TestPersonRequiredForProtectedEndpoints(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	// Endpoints that require a person to be selected
	personRequiredEndpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/api/daily", ""},
		{"POST", "/api/save", `{"content":"test"}`},
		{"POST", "/api/append", `{"content":"test"}`},
		{"POST", "/api/clear-pinned", ""},
		{"POST", "/api/todos/add", `{"category":"work"}`},
		{"GET", "/api/files/list?path=.", ""},
		{"GET", "/api/files/read?path=notes/test.md", ""},
		{"POST", "/api/files/create", `{"path":"test.md"}`},
		{"POST", "/api/files/save", `{"path":"test.md","content":"test"}`},
		{"POST", "/api/files/delete", `{"path":"test.md"}`},
		{"POST", "/api/agent/chat", `{"message":"hi"}`},
		{"POST", "/api/agent/chat-stream", `{"message":"hi"}`},
		{"GET", "/api/agent/sessions", ""},
		{"POST", "/api/agent/sessions/clear", `{}`},
		{"POST", "/api/agent/session/clear", `{"session_id":"test"}`},
		{"GET", "/api/agent/session/history?session_id=test", ""},
		{"GET", "/api/agent/config", ""},
		{"POST", "/api/agent/config", `{"runtime_mode":"anthropic_api_key"}`},
		{"GET", "/api/agent/actions", ""},
		{"POST", "/api/agent/actions/test/run", `{}`},
		{"POST", "/api/agent/tools/execute", `{"tool":"read_file","args":{"path":"notes/test.md"}}`},
		{"GET", "/api/agent/gateway/health", ""},
		{"GET", "/api/linkedin/health", ""},
	}

	for _, ep := range personRequiredEndpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			req.Header.Set("Authorization", "Bearer test-token-123")
			// No X-Notes-Person header

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("endpoint without person should return 400, got %d", rec.Code)
			}

			var errResp ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			if errResp.Detail != "Person not selected" {
				t.Errorf("error detail = %q, want %q", errResp.Detail, "Person not selected")
			}
		})
	}
}
