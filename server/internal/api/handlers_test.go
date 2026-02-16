package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDailyHandlers tests the daily note endpoints.
func TestDailyHandlers(t *testing.T) {
	srv, vaultRoot, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	t.Run("GET /api/daily creates daily note if not exists", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/daily", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		// Verify response fields
		if resp["date"] == nil {
			t.Error("response missing 'date' field")
		}
		if resp["content"] == nil {
			t.Error("response missing 'content' field")
		}
		if resp["path"] == nil {
			t.Error("response missing 'path' field")
		}

		// Verify date format
		date, _ := resp["date"].(string)
		if _, err := time.Parse("2006-01-02", date); err != nil {
			t.Errorf("invalid date format: %s", date)
		}
	})

	t.Run("POST /api/save saves daily note", func(t *testing.T) {
		// First create a file
		today := time.Now().Format("2006-01-02")
		dailyPath := filepath.Join(vaultRoot, "sebastian", "daily", today+".md")
		os.MkdirAll(filepath.Dir(dailyPath), 0755)
		os.WriteFile(dailyPath, []byte("# Original"), 0644)

		body := `{"path":"daily/` + today + `.md","content":"# Updated Content"}`
		req := makeRequest(t, "POST", "/api/save", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp SuccessResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if !resp.Success {
			t.Error("response should have success=true")
		}

		// Verify file was updated
		content, _ := os.ReadFile(dailyPath)
		if !strings.Contains(string(content), "# Updated Content") {
			t.Error("file content was not updated")
		}
	})

	t.Run("POST /api/save missing path returns 400", func(t *testing.T) {
		body := `{"content":"test"}`
		req := makeRequest(t, "POST", "/api/save", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})

	t.Run("POST /api/append appends entry", func(t *testing.T) {
		// Create daily note with custom notes section
		today := time.Now().Format("2006-01-02")
		dailyPath := filepath.Join(vaultRoot, "sebastian", "daily", today+".md")
		os.WriteFile(dailyPath, []byte("# daily "+today+"\n\n## custom notes\n"), 0644)

		body := `{"path":"daily/` + today + `.md","text":"Test entry","pinned":false}`
		req := makeRequest(t, "POST", "/api/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("POST /api/append missing text returns 400", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		body := `{"path":"daily/` + today + `.md"}`
		req := makeRequest(t, "POST", "/api/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Text is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Text is required")
		}
	})

	t.Run("POST /api/clear-pinned clears markers", func(t *testing.T) {
		// Create file with pinned marker
		today := time.Now().Format("2006-01-02")
		dailyPath := filepath.Join(vaultRoot, "sebastian", "daily", today+".md")
		os.WriteFile(dailyPath, []byte("# daily "+today+"\n### Entry <pinned>\nContent\n"), 0644)

		body := `{"path":"daily/` + today + `.md"}`
		req := makeRequest(t, "POST", "/api/clear-pinned", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("POST /api/clear-pinned missing path returns 400", func(t *testing.T) {
		body := `{}`
		req := makeRequest(t, "POST", "/api/clear-pinned", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})
}

// TestTodoHandlers tests the todo endpoints.
func TestTodoHandlers(t *testing.T) {
	srv, vaultRoot, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	t.Run("POST /api/todos/add adds task to category", func(t *testing.T) {
		// Create daily note with todos section and work category
		today := time.Now().Format("2006-01-02")
		dailyPath := filepath.Join(vaultRoot, "sebastian", "daily", today+".md")
		os.MkdirAll(filepath.Dir(dailyPath), 0755)
		os.WriteFile(dailyPath, []byte("# daily "+today+"\n\n## todos\n\n### work\n- [ ] Existing task\n\n### priv\n"), 0644)

		body := `{"category":"work","text":"New task"}`
		req := makeRequest(t, "POST", "/api/todos/add", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify task was added
		content, _ := os.ReadFile(dailyPath)
		if !strings.Contains(string(content), "New task") {
			t.Error("task was not added to file")
		}
	})

	t.Run("POST /api/todos/add with empty text creates blank task", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		dailyPath := filepath.Join(vaultRoot, "sebastian", "daily", today+".md")
		os.WriteFile(dailyPath, []byte("# daily "+today+"\n\n## todos\n\n### priv\n"), 0644)

		body := `{"category":"priv"}`
		req := makeRequest(t, "POST", "/api/todos/add", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify blank task was added
		content, _ := os.ReadFile(dailyPath)
		if !strings.Contains(string(content), "- [ ]") {
			t.Error("blank task was not added")
		}
	})

	t.Run("POST /api/todos/add missing category returns 400", func(t *testing.T) {
		body := `{"text":"test"}`
		req := makeRequest(t, "POST", "/api/todos/add", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Category is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Category is required")
		}
	})

	t.Run("POST /api/todos/add invalid category returns 400", func(t *testing.T) {
		body := `{"category":"invalid"}`
		req := makeRequest(t, "POST", "/api/todos/add", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Invalid category" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Invalid category")
		}
	})

	t.Run("POST /api/todos/toggle toggles task", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		dailyPath := filepath.Join(vaultRoot, "sebastian", "daily", today+".md")
		os.WriteFile(dailyPath, []byte("# daily "+today+"\n- [ ] Task to toggle\n"), 0644)

		body := `{"path":"daily/` + today + `.md","line":2}`
		req := makeRequest(t, "POST", "/api/todos/toggle", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify task was toggled
		content, _ := os.ReadFile(dailyPath)
		if !strings.Contains(string(content), "- [x]") {
			t.Error("task was not toggled to checked")
		}
	})

	t.Run("POST /api/todos/toggle missing path returns 400", func(t *testing.T) {
		body := `{"line":1}`
		req := makeRequest(t, "POST", "/api/todos/toggle", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})

	t.Run("POST /api/todos/toggle invalid line returns 400", func(t *testing.T) {
		body := `{"path":"daily/test.md","line":0}`
		req := makeRequest(t, "POST", "/api/todos/toggle", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Line must be positive" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Line must be positive")
		}
	})
}

// TestFileHandlers tests the file management endpoints.
func TestFileHandlers(t *testing.T) {
	srv, vaultRoot, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	t.Run("GET /api/files/list returns directory entries", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/list?path=notes", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["entries"] == nil {
			t.Error("response missing 'entries' field")
		}
	})

	t.Run("GET /api/files/list defaults to root", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/list", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("GET /api/files/list empty directory returns empty entries array", func(t *testing.T) {
		emptyDir := filepath.Join(vaultRoot, "sebastian", "empty")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatalf("failed to create empty dir: %v", err)
		}

		req := makeRequest(t, "GET", "/api/files/list?path=empty", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		entries, ok := resp["entries"].([]interface{})
		if !ok {
			t.Fatalf("entries should be array, got %T (%v)", resp["entries"], resp["entries"])
		}
		if len(entries) != 0 {
			t.Errorf("expected empty entries, got %d", len(entries))
		}
	})

	t.Run("GET /api/files/list nonexistent directory returns 404", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/list?path=nonexistent", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Directory not found" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Directory not found")
		}
	})

	t.Run("GET /api/files/read returns file content", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/read?path=notes/secret.md", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["content"] == nil {
			t.Error("response missing 'content' field")
		}
		if resp["path"] == nil {
			t.Error("response missing 'path' field")
		}
	})

	t.Run("GET /api/files/read missing path returns 400", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/read", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})

	t.Run("GET /api/files/read nonexistent file returns 404", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/files/read?path=nonexistent.md", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "File not found" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "File not found")
		}
	})

	t.Run("POST /api/files/create creates file", func(t *testing.T) {
		body := `{"path":"notes/newfile.md"}`
		req := makeRequest(t, "POST", "/api/files/create", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify file was created
		filePath := filepath.Join(vaultRoot, "sebastian", "notes", "newfile.md")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("file was not created")
		}
	})

	t.Run("POST /api/files/create duplicate returns 400", func(t *testing.T) {
		body := `{"path":"notes/secret.md"}`
		req := makeRequest(t, "POST", "/api/files/create", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "File already exists" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "File already exists")
		}
	})

	t.Run("POST /api/files/create missing path returns 400", func(t *testing.T) {
		body := `{}`
		req := makeRequest(t, "POST", "/api/files/create", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})

	t.Run("POST /api/files/save saves file", func(t *testing.T) {
		body := `{"path":"notes/secret.md","content":"# Updated Secret"}`
		req := makeRequest(t, "POST", "/api/files/save", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify content was updated
		filePath := filepath.Join(vaultRoot, "sebastian", "notes", "secret.md")
		content, _ := os.ReadFile(filePath)
		if !strings.Contains(string(content), "# Updated Secret") {
			t.Error("file content was not updated")
		}
	})

	t.Run("POST /api/files/save missing path returns 400", func(t *testing.T) {
		body := `{"content":"test"}`
		req := makeRequest(t, "POST", "/api/files/save", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})

	t.Run("POST /api/files/delete deletes file", func(t *testing.T) {
		// Create a file to delete
		filePath := filepath.Join(vaultRoot, "sebastian", "notes", "todelete.md")
		os.WriteFile(filePath, []byte("to delete"), 0644)

		body := `{"path":"notes/todelete.md"}`
		req := makeRequest(t, "POST", "/api/files/delete", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify file was deleted
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("file was not deleted")
		}
	})

	t.Run("POST /api/files/delete missing path returns 400", func(t *testing.T) {
		body := `{}`
		req := makeRequest(t, "POST", "/api/files/delete", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})

	t.Run("POST /api/files/unpin unpins entry", func(t *testing.T) {
		// Create file with pinned entry - must be format "### HH:MM <pinned>"
		filePath := filepath.Join(vaultRoot, "sebastian", "notes", "pinned.md")
		os.WriteFile(filePath, []byte("# Title\n### 14:30 <pinned>\nContent\n"), 0644)

		body := `{"path":"notes/pinned.md","line":2}`
		req := makeRequest(t, "POST", "/api/files/unpin", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify the pinned marker was removed
		content, _ := os.ReadFile(filePath)
		if strings.Contains(string(content), "<pinned>") {
			t.Error("pinned marker should have been removed")
		}
	})

	t.Run("POST /api/files/unpin missing path returns 400", func(t *testing.T) {
		body := `{"line":1}`
		req := makeRequest(t, "POST", "/api/files/unpin", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Path is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Path is required")
		}
	})

	t.Run("POST /api/files/unpin invalid line returns 400", func(t *testing.T) {
		body := `{"path":"notes/test.md","line":0}`
		req := makeRequest(t, "POST", "/api/files/unpin", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Line must be positive" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Line must be positive")
		}
	})
}

// TestSleepHandlers tests the sleep tracking endpoints.
func TestSleepHandlers(t *testing.T) {
	t.Run("GET /api/sleep-times returns empty list when no file", func(t *testing.T) {
		srv, vaultRoot, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		// Remove the sleep_times.md file created by setup
		os.Remove(filepath.Join(vaultRoot, "sleep_times.md"))

		req := makeRequest(t, "GET", "/api/sleep-times", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		entries, ok := resp["entries"].([]interface{})
		if !ok {
			t.Fatal("response missing 'entries' field")
		}
		if len(entries) != 0 {
			t.Errorf("expected empty entries, got %d", len(entries))
		}
	})

	t.Run("GET /api/sleep-times returns parsed entries", func(t *testing.T) {
		srv, vaultRoot, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		// Create sleep times file with entries
		content := "2024-01-15 | Thomas | 19:30 | eingeschlafen\n2024-01-15 | Thomas | 07:00 | aufgewacht\n"
		os.WriteFile(filepath.Join(vaultRoot, "sleep_times.md"), []byte(content), 0644)

		req := makeRequest(t, "GET", "/api/sleep-times", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		entries, ok := resp["entries"].([]interface{})
		if !ok {
			t.Fatal("response missing 'entries' field")
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("POST /api/sleep-times/append adds entry", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"child":"Thomas","time":"20:00","status":"eingeschlafen"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp SuccessResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if !resp.Success {
			t.Error("response should have success=true")
		}
	})

	t.Run("POST /api/sleep-times/append missing child returns 400", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"time":"20:00","status":"eingeschlafen"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Child is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Child is required")
		}
	})

	t.Run("POST /api/sleep-times/append missing time returns 400", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"child":"Thomas","status":"eingeschlafen"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Time is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Time is required")
		}
	})

	t.Run("POST /api/sleep-times/append missing status returns 400", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"child":"Thomas","time":"20:00"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Status is required" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Status is required")
		}
	})

	t.Run("POST /api/sleep-times/append invalid child returns 400", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"child":"Unknown","time":"20:00","status":"eingeschlafen"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Invalid child name" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Invalid child name")
		}
	})

	t.Run("POST /api/sleep-times/append invalid status returns 400", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"child":"Thomas","time":"20:00","status":"invalid"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Invalid status" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Invalid status")
		}
	})

	t.Run("POST /api/sleep-times/delete deletes entry", func(t *testing.T) {
		srv, vaultRoot, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		// Create sleep times file
		content := "2024-01-15 | Thomas | 19:30 | eingeschlafen\n2024-01-15 | Thomas | 07:00 | aufgewacht\n"
		os.WriteFile(filepath.Join(vaultRoot, "sleep_times.md"), []byte(content), 0644)

		body := `{"line":1}`
		req := makeRequest(t, "POST", "/api/sleep-times/delete", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// Verify line was deleted
		newContent, _ := os.ReadFile(filepath.Join(vaultRoot, "sleep_times.md"))
		if strings.Contains(string(newContent), "19:30") {
			t.Error("entry was not deleted")
		}
	})

	t.Run("POST /api/sleep-times/delete invalid line returns 400", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"line":0}`
		req := makeRequest(t, "POST", "/api/sleep-times/delete", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Line must be positive" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Line must be positive")
		}
	})

	t.Run("POST /api/sleep-times/delete line out of range returns 400", func(t *testing.T) {
		srv, vaultRoot, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		// Create sleep times file with 1 entry
		content := "2024-01-15 | Thomas | 19:30 | eingeschlafen\n"
		os.WriteFile(filepath.Join(vaultRoot, "sleep_times.md"), []byte(content), 0644)

		body := `{"line":999}`
		req := makeRequest(t, "POST", "/api/sleep-times/delete", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Line number out of range" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Line number out of range")
		}
	})

	t.Run("POST /api/sleep-times/delete no file returns 404", func(t *testing.T) {
		srv, vaultRoot, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		// Remove the sleep_times.md file
		os.Remove(filepath.Join(vaultRoot, "sleep_times.md"))

		body := `{"line":1}`
		req := makeRequest(t, "POST", "/api/sleep-times/delete", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}

		var errResp ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Detail != "Sleep times file not found" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Sleep times file not found")
		}
	})
}

// TestCORSHeaders tests that CORS headers are properly set.
func TestCORSHeaders(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	t.Run("OPTIONS preflight request returns CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/daily", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Authorization, X-Notes-Person")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		// Preflight should return 200 with CORS headers
		// Note: Chi's CORS middleware may return different status codes
		allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin == "" {
			t.Logf("Response headers: %v", rec.Header())
			t.Logf("Response status: %d", rec.Code)
			t.Error("missing Access-Control-Allow-Origin header")
		}
	})

	t.Run("regular request includes CORS origin header", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/daily", "", "sebastian")
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin == "" {
			t.Logf("Response headers: %v", rec.Header())
			t.Error("missing Access-Control-Allow-Origin header")
		}
	})

	t.Run("CORS allows all origins", func(t *testing.T) {
		req := makeRequest(t, "GET", "/api/daily", "", "sebastian")
		req.Header.Set("Origin", "http://some-other-domain.com")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
		// With AllowedOrigins: ["*"], the response should echo back the origin or "*"
		if allowOrigin != "*" && allowOrigin != "http://some-other-domain.com" {
			t.Errorf("Access-Control-Allow-Origin = %q, want '*' or request origin", allowOrigin)
		}
	})
}

// TestInvalidJSON tests handling of invalid JSON in request bodies.
func TestInvalidJSON(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/api/save"},
		{"POST", "/api/append"},
		{"POST", "/api/clear-pinned"},
		{"POST", "/api/todos/add"},
		{"POST", "/api/todos/toggle"},
		{"POST", "/api/sleep-times/append"},
		{"POST", "/api/sleep-times/delete"},
		{"POST", "/api/files/create"},
		{"POST", "/api/files/save"},
		{"POST", "/api/files/delete"},
		{"POST", "/api/files/unpin"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			req := makeRequest(t, ep.method, ep.path, "{invalid json", "sebastian")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for invalid JSON", rec.Code, http.StatusBadRequest)
			}

			var errResp ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			if errResp.Detail != "Invalid request body" {
				t.Errorf("error detail = %q, want %q", errResp.Detail, "Invalid request body")
			}
		})
	}
}

// TestContentTypeJSON tests that responses have correct Content-Type.
func TestContentTypeJSON(t *testing.T) {
	t.Run("GET response has JSON content type", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		req := makeRequest(t, "GET", "/api/daily", "", "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
	})

	t.Run("POST success response has JSON content type", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"child":"Thomas","time":"20:00","status":"eingeschlafen"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
	})

	t.Run("error response has JSON content type", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		req := makeRequest(t, "GET", "/api/files/read", "", "sebastian") // missing path
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
	})
}

// TestSuccessResponseFormat tests that success responses have correct format.
func TestSuccessResponseFormat(t *testing.T) {
	t.Run("success response has success field", func(t *testing.T) {
		srv, _, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		body := `{"child":"Thomas","time":"20:00","status":"eingeschlafen"}`
		req := makeRequest(t, "POST", "/api/sleep-times/append", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var resp SuccessResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if !resp.Success {
			t.Error("response should have success=true")
		}
		if resp.Message == "" {
			t.Error("response should have non-empty message")
		}
	})

	t.Run("save success returns proper response", func(t *testing.T) {
		srv, vaultRoot, cleanup := setupTestServer(t)
		defer cleanup()
		router := NewRouter(srv)

		today := time.Now().Format("2006-01-02")
		dailyPath := filepath.Join(vaultRoot, "sebastian", "daily", today+".md")
		os.MkdirAll(filepath.Dir(dailyPath), 0755)
		os.WriteFile(dailyPath, []byte("# Test"), 0644)

		body := `{"path":"daily/` + today + `.md","content":"# Updated"}`
		req := makeRequest(t, "POST", "/api/save", body, "sebastian")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var resp SuccessResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if !resp.Success {
			t.Error("response should have success=true")
		}
	})
}

// TestErrorResponseFormat tests that all error types have consistent format.
func TestErrorResponseFormat(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	tests := []struct {
		name           string
		setup          func()
		method         string
		path           string
		body           string
		person         string
		authHeader     string
		expectedStatus int
		expectedDetail string
	}{
		{
			name:           "400 missing required field",
			method:         "POST",
			path:           "/api/save",
			body:           `{"content":"test"}`,
			person:         "sebastian",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusBadRequest,
			expectedDetail: "Path is required",
		},
		{
			name:           "400 invalid category",
			method:         "POST",
			path:           "/api/todos/add",
			body:           `{"category":"invalid"}`,
			person:         "sebastian",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusBadRequest,
			expectedDetail: "Invalid category",
		},
		{
			name:           "400 invalid JSON",
			method:         "POST",
			path:           "/api/save",
			body:           `{invalid`,
			person:         "sebastian",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusBadRequest,
			expectedDetail: "Invalid request body",
		},
		{
			name:           "401 missing auth",
			method:         "GET",
			path:           "/api/daily",
			body:           "",
			person:         "sebastian",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedDetail: "Unauthorized",
		},
		{
			name:           "401 invalid token",
			method:         "GET",
			path:           "/api/daily",
			body:           "",
			person:         "sebastian",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
			expectedDetail: "Unauthorized",
		},
		{
			name:           "400 missing person",
			method:         "GET",
			path:           "/api/daily",
			body:           "",
			person:         "",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusBadRequest,
			expectedDetail: "Person not selected",
		},
		{
			name:           "400 invalid person",
			method:         "GET",
			path:           "/api/daily",
			body:           "",
			person:         "hacker",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusBadRequest,
			expectedDetail: "Invalid person",
		},
		{
			name:           "404 file not found",
			method:         "GET",
			path:           "/api/files/read?path=nonexistent.md",
			body:           "",
			person:         "sebastian",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusNotFound,
			expectedDetail: "File not found",
		},
		{
			name:           "404 directory not found",
			method:         "GET",
			path:           "/api/files/list?path=nonexistent",
			body:           "",
			person:         "sebastian",
			authHeader:     "Bearer test-token-123",
			expectedStatus: http.StatusNotFound,
			expectedDetail: "Directory not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.person != "" {
				req.Header.Set("X-Notes-Person", tt.person)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tt.expectedStatus, rec.Body.String())
			}

			var errResp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Errorf("failed to parse error response: %v; body = %s", err, rec.Body.String())
			}
			if errResp.Detail != tt.expectedDetail {
				t.Errorf("error detail = %q, want %q", errResp.Detail, tt.expectedDetail)
			}
		})
	}
}
