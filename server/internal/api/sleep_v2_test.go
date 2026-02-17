package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sleepTimesResp struct {
	Entries []SleepEntry `json:"entries"`
}

type sleepSummaryResp struct {
	Nights []SleepNightSummary `json:"nights"`
}

func TestSleepV2MigrationAndIDCrud(t *testing.T) {
	srv, vaultRoot, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	legacy := strings.Join([]string{
		"2026-02-16 | Fabian | 19:30 | eingeschlafen",
		"2026-02-17 | Fabian | weird raw value | aufgewacht",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(vaultRoot, "sleep_times.md"), []byte(legacy), 0644); err != nil {
		t.Fatalf("write legacy sleep_times.md: %v", err)
	}

	// Trigger migration via read.
	req := makeRequest(t, "GET", "/api/sleep-times", "", "sebastian")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET sleep-times status=%d body=%s", rec.Code, rec.Body.String())
	}

	var got sleepTimesResp
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Entries) == 0 {
		t.Fatalf("expected migrated entries")
	}
	if got.Entries[0].ID == "" {
		t.Fatalf("expected stable id field in response")
	}

	// Verify DB exists after migration.
	if _, err := os.Stat(filepath.Join(vaultRoot, "sleep.db")); err != nil {
		t.Fatalf("sleep.db not created: %v", err)
	}

	// Create a new entry.
	appendBody := `{"child":"Thomas","time":"20:10","status":"eingeschlafen"}`
	req = makeRequest(t, "POST", "/api/sleep-times/append", appendBody, "sebastian")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("append status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Read again to get latest IDs.
	req = makeRequest(t, "GET", "/api/sleep-times", "", "sebastian")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET sleep-times status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var target SleepEntry
	for _, e := range got.Entries {
		if e.Child == "Thomas" && e.Time == "20:10" && e.Status == "eingeschlafen" {
			target = e
			break
		}
	}
	if target.ID == "" {
		t.Fatalf("failed to find appended entry with id")
	}

	// Update by ID.
	updateBody := `{"id":"` + target.ID + `","child":"Thomas","time":"20:15","status":"eingeschlafen","notes":"updated"}`
	req = makeRequest(t, "POST", "/api/sleep-times/update", updateBody, "sebastian")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Delete by ID.
	deleteBody := `{"id":"` + target.ID + `"}`
	req = makeRequest(t, "POST", "/api/sleep-times/delete", deleteBody, "sebastian")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSleepV2SummaryAndExport(t *testing.T) {
	srv, vaultRoot, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	// Two events to form one completed night.
	appendAsleep := `{"child":"Fabian","time":"20:00","status":"eingeschlafen","occurred_at":"2026-02-16T19:00:00Z"}`
	req := makeRequest(t, "POST", "/api/sleep-times/append", appendAsleep, "sebastian")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("append asleep status=%d body=%s", rec.Code, rec.Body.String())
	}

	appendAwake := `{"child":"Fabian","time":"06:30","status":"aufgewacht","occurred_at":"2026-02-17T05:30:00Z"}`
	req = makeRequest(t, "POST", "/api/sleep-times/append", appendAwake, "sebastian")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("append awake status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Summary should contain at least one night.
	req = makeRequest(t, "GET", "/api/sleep-times/summary", "", "sebastian")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", rec.Code, rec.Body.String())
	}

	var summary sleepSummaryResp
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if len(summary.Nights) == 0 {
		t.Fatalf("expected at least one night summary")
	}

	// Export should overwrite markdown file from DB.
	req = makeRequest(t, "POST", "/api/sleep-times/export-markdown", `{}`, "sebastian")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", rec.Code, rec.Body.String())
	}

	content, err := os.ReadFile(filepath.Join(vaultRoot, "sleep_times.md"))
	if err != nil {
		t.Fatalf("read exported markdown: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "| Date | Child | Time | Status | Notes |") {
		t.Fatalf("export missing expected table header")
	}
	if !strings.Contains(text, "Fabian") {
		t.Fatalf("export missing expected entry content")
	}
}
