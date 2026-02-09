package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"notes-editor/internal/vault"
)

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
