package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"notes-editor/internal/vault"
)

func TestToolExecutor_SearchFiles_UsesQMD(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"
	if err := os.MkdirAll(filepath.Join(root, person, "notes"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, person, "notes", "today.md"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	origEndpoint := qmdMCPEndpoint
	origClient := qmdMCPClient
	t.Cleanup(func() {
		qmdMCPEndpoint = origEndpoint
		qmdMCPClient = origClient
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     any            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("invalid request JSON: %v", err)
		}

		switch req.Method {
		case "initialize":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"protocolVersion": "2025-06-18",
				},
			})
		case "tools/call":
			if got := req.Params["name"]; got != "deep_search" {
				t.Fatalf("tools/call name=%v want deep_search", got)
			}
			args, _ := req.Params["arguments"].(map[string]any)
			if got := args["collection"]; got != person {
				t.Fatalf("collection=%v want %q", got, person)
			}
			if got := args["query"]; got != "match" {
				t.Fatalf("query=%v want %q", got, "match")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"structuredContent": map[string]any{
						"results": []map[string]any{
							{
								"docid":   "#abc123",
								"score":   0.87,
								"file":    "notes/today.md",
								"title":   "Today",
								"context": "Journal",
								"snippet": "12: first match\n25: second match",
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected method: %s", req.Method)
		}
	}))
	defer srv.Close()
	qmdMCPEndpoint = srv.URL + "/mcp"
	qmdMCPClient = srv.Client()

	store := vault.NewStore(root)
	te := NewToolExecutor(store, nil, person)
	out, err := te.ExecuteTool("search_files", map[string]any{
		"pattern": "match",
		"path":    "notes",
	})
	if err != nil {
		t.Fatalf("ExecuteTool(search_files): %v", err)
	}

	var payload []struct {
		File    string `json:"file"`
		Matches []struct {
			LineNumber int    `json:"line_number"`
			Content    string `json:"content"`
		} `json:"matches"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal response: %v; out=%s", err, out)
	}
	if len(payload) != 1 {
		t.Fatalf("result count=%d want 1; out=%s", len(payload), out)
	}
	if payload[0].File != "notes/today.md" {
		t.Fatalf("file=%q want %q", payload[0].File, "notes/today.md")
	}
	if len(payload[0].Matches) != 2 {
		t.Fatalf("matches count=%d want 2", len(payload[0].Matches))
	}
	if payload[0].Matches[0].LineNumber != 12 || payload[0].Matches[1].LineNumber != 25 {
		t.Fatalf("unexpected line numbers: %+v", payload[0].Matches)
	}
}

func TestToolExecutor_SearchFiles_QMDErrorBubblesUp(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"
	if err := os.MkdirAll(filepath.Join(root, person), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	origEndpoint := qmdMCPEndpoint
	origClient := qmdMCPClient
	t.Cleanup(func() {
		qmdMCPEndpoint = origEndpoint
		qmdMCPClient = origClient
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("invalid request JSON: %v", err)
		}
		if req.Method == "initialize" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  map[string]any{"protocolVersion": "2025-06-18"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]any{
				"code":    -32000,
				"message": "qmd exploded",
			},
		})
	}))
	defer srv.Close()
	qmdMCPEndpoint = srv.URL + "/mcp"
	qmdMCPClient = srv.Client()

	store := vault.NewStore(root)
	te := NewToolExecutor(store, nil, person)
	_, err := te.ExecuteTool("search_files", map[string]any{"pattern": "abc"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "qmd exploded") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolExecutor_GlobFiles(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"

	// Create a small vault fixture.
	mustWrite := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, person, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	mustWrite("a.md", "hello")
	mustWrite("folder/b.prompt.md", "b")
	mustWrite("folder/c.txt", "c")
	mustWrite(".hidden/secret.md", "nope")
	mustWrite("folder/.hidden.md", "nope")

	store := vault.NewStore(root)
	te := NewToolExecutor(store, nil, person)

	out, err := te.ExecuteTool("glob_files", map[string]any{
		"pattern": "**/*.md",
	})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}

	var got []string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v. out=%q", err, out)
	}

	// Hidden paths should not be returned.
	assertNotContains(t, got, ".hidden/secret.md")
	assertNotContains(t, got, "folder/.hidden.md")

	// Expected files should be present.
	assertContains(t, got, "a.md")
	assertContains(t, got, "folder/b.prompt.md")
}

func TestToolExecutor_GlobFiles_PathAndLimit(t *testing.T) {
	root := t.TempDir()
	person := "sebastian"

	if err := os.MkdirAll(filepath.Join(root, person, "folder"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, person, "folder", "a.prompt.md"), []byte("a"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, person, "folder", "b.prompt.md"), []byte("b"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	store := vault.NewStore(root)
	te := NewToolExecutor(store, nil, person)

	out, err := te.ExecuteTool("glob_files", map[string]any{
		"pattern": "*.prompt.md",
		"path":    "folder",
		"limit":   1,
	})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}

	var got []string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v. out=%q", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("expected limit=1 result, got %d: %#v", len(got), got)
	}
}

func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			return
		}
	}
	t.Fatalf("expected to contain %q, got: %#v", needle, haystack)
}

func assertNotContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			t.Fatalf("expected NOT to contain %q, got: %#v", needle, haystack)
		}
	}
}

func TestToolExecutor_WebSearch_WrappedAndCapped(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("X-Subscription-Token") == "" {
			t.Fatalf("missing X-Subscription-Token header")
		}
		if got := r.URL.Query().Get("count"); got != "2" {
			t.Fatalf("count query = %q, want 2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"web":{"results":[
				{"title":"One","url":"https://example.com/1","description":"first"},
				{"title":"Two","url":"https://example.com/2","description":"second"},
				{"title":"Three","url":"https://example.com/3","description":"third"}
			]}
		}`))
	}))
	defer ts.Close()

	origEndpoint := webSearchEndpoint
	webSearchEndpoint = ts.URL
	defer func() { webSearchEndpoint = origEndpoint }()

	origCache := webSearchCache
	webSearchCache = map[string]webSearchCacheEntry{}
	defer func() { webSearchCache = origCache }()

	t.Setenv("BRAVE_API_KEY", "test-key")
	t.Setenv("WEB_SEARCH_MAX_RESULTS", "2")
	t.Setenv("WEB_SEARCH_CACHE_TTL", "5m")

	store := vault.NewStore(t.TempDir())
	te := NewToolExecutor(store, nil, "sebastian")
	out, err := te.ExecuteTool("web_search", map[string]any{"query": "golang tools"})
	if err != nil {
		t.Fatalf("ExecuteTool(web_search): %v", err)
	}
	if !strings.HasPrefix(out, "<web_search_result_json>\n") || !strings.HasSuffix(out, "\n</web_search_result_json>") {
		t.Fatalf("unexpected wrapper: %q", out)
	}
	raw := strings.TrimPrefix(out, "<web_search_result_json>\n")
	raw = strings.TrimSuffix(raw, "\n</web_search_result_json>")

	var payload struct {
		Query   string `json:"query"`
		Count   int    `json:"count"`
		Results []struct {
			Title string `json:"title"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v; out=%q", err, out)
	}
	if payload.Query != "golang tools" {
		t.Fatalf("query = %q, want %q", payload.Query, "golang tools")
	}
	if payload.Count != 2 || len(payload.Results) != 2 {
		t.Fatalf("count/results = %d/%d, want 2/2", payload.Count, len(payload.Results))
	}
	if hits.Load() != 1 {
		t.Fatalf("upstream hits = %d, want 1", hits.Load())
	}

	// second call should come from cache
	if _, err := te.ExecuteTool("web_search", map[string]any{"query": "golang   tools"}); err != nil {
		t.Fatalf("ExecuteTool(web_search) cached: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("upstream hits after cached call = %d, want 1", hits.Load())
	}
}
