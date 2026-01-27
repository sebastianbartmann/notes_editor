package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupDailyTest(t *testing.T) (*Daily, *Store, string) {
	t.Helper()
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	daily := NewDaily(store)

	// Create person directories
	if err := os.MkdirAll(filepath.Join(tmpDir, "sebastian", "daily"), 0755); err != nil {
		t.Fatal(err)
	}

	return daily, store, tmpDir
}

func TestDaily_GetOrCreateDaily_NewNote(t *testing.T) {
	daily, _, _ := setupDailyTest(t)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	content, path, err := daily.GetOrCreateDaily("sebastian", date)
	if err != nil {
		t.Fatalf("GetOrCreateDaily() error = %v", err)
	}

	if path != "daily/2024-01-15.md" {
		t.Errorf("path = %q, want %q", path, "daily/2024-01-15.md")
	}

	if !strings.Contains(content, "# 2024-01-15") {
		t.Error("content missing date header")
	}
	if !strings.Contains(content, "## todos") {
		t.Error("content missing todos section")
	}
	if !strings.Contains(content, "## custom notes") {
		t.Error("content missing custom notes section")
	}
}

func TestDaily_GetOrCreateDaily_ExistingNote(t *testing.T) {
	daily, store, _ := setupDailyTest(t)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Create existing note
	existingContent := "# Existing Note\n\nCustom content."
	err := store.WriteFile("sebastian", "daily/2024-01-15.md", existingContent)
	if err != nil {
		t.Fatal(err)
	}

	content, _, err := daily.GetOrCreateDaily("sebastian", date)
	if err != nil {
		t.Fatalf("GetOrCreateDaily() error = %v", err)
	}

	if content != existingContent {
		t.Errorf("content = %q, want existing content", content)
	}
}

func TestDaily_TodoInheritance(t *testing.T) {
	daily, store, _ := setupDailyTest(t)

	// Create previous day's note with todos
	prevContent := `# 2024-01-14

## todos

### work
- [ ] incomplete task 1
- [x] completed task
- [ ] incomplete task 2

### priv
- [ ] personal task
- [x] done personal task

## custom notes

### 10:00
Some notes here.
`
	err := store.WriteFile("sebastian", "daily/2024-01-14.md", prevContent)
	if err != nil {
		t.Fatal(err)
	}

	// Create today's note
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	content, _, err := daily.GetOrCreateDaily("sebastian", date)
	if err != nil {
		t.Fatalf("GetOrCreateDaily() error = %v", err)
	}

	// Should have inherited incomplete todos
	if !strings.Contains(content, "- [ ] incomplete task 1") {
		t.Error("missing inherited incomplete task 1")
	}
	if !strings.Contains(content, "- [ ] incomplete task 2") {
		t.Error("missing inherited incomplete task 2")
	}
	if !strings.Contains(content, "- [ ] personal task") {
		t.Error("missing inherited personal task")
	}

	// Should NOT have completed tasks
	if strings.Contains(content, "completed task") {
		t.Error("should not inherit completed tasks")
	}
	if strings.Contains(content, "done personal task") {
		t.Error("should not inherit done personal task")
	}
}

func TestDaily_PinnedInheritance(t *testing.T) {
	daily, store, _ := setupDailyTest(t)

	// Create previous day's note with pinned entries
	prevContent := `# 2024-01-14

## todos

## custom notes

### 10:00 <pinned>
This is a pinned note that should carry over.
Multiple lines here.

### 11:00
Regular note - not pinned.

### 12:00 <pinned>
Another pinned entry.
`
	err := store.WriteFile("sebastian", "daily/2024-01-14.md", prevContent)
	if err != nil {
		t.Fatal(err)
	}

	// Create today's note
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	content, _, err := daily.GetOrCreateDaily("sebastian", date)
	if err != nil {
		t.Fatalf("GetOrCreateDaily() error = %v", err)
	}

	// Should have inherited pinned entries
	if !strings.Contains(content, "### 10:00 <pinned>") {
		t.Error("missing first pinned entry header")
	}
	if !strings.Contains(content, "This is a pinned note") {
		t.Error("missing first pinned entry content")
	}
	if !strings.Contains(content, "### 12:00 <pinned>") {
		t.Error("missing second pinned entry header")
	}

	// Should NOT have non-pinned entries
	if strings.Contains(content, "### 11:00") {
		t.Error("should not inherit non-pinned entries")
	}
	if strings.Contains(content, "Regular note") {
		t.Error("should not inherit non-pinned content")
	}
}

func TestDaily_AddTask(t *testing.T) {
	daily, store, _ := setupDailyTest(t)

	// Create a daily note
	content := `# 2024-01-15

## todos

### work
- [ ] existing task

## custom notes
`
	err := store.WriteFile("sebastian", "daily/test.md", content)
	if err != nil {
		t.Fatal(err)
	}

	// Add a task
	err = daily.AddTask("sebastian", "daily/test.md", "work", "new task")
	if err != nil {
		t.Fatalf("AddTask() error = %v", err)
	}

	// Verify task was added
	result, _ := store.ReadFile("sebastian", "daily/test.md")
	if !strings.Contains(result, "- [ ] new task") {
		t.Error("new task not found in content")
	}
}

func TestDaily_ToggleTask(t *testing.T) {
	daily, store, _ := setupDailyTest(t)

	// Create a daily note with tasks
	content := `# 2024-01-15

## todos

- [ ] unchecked task
- [x] checked task
`
	err := store.WriteFile("sebastian", "daily/test.md", content)
	if err != nil {
		t.Fatal(err)
	}

	// Toggle unchecked task (line 5)
	err = daily.ToggleTask("sebastian", "daily/test.md", 5)
	if err != nil {
		t.Fatalf("ToggleTask() error = %v", err)
	}

	result, _ := store.ReadFile("sebastian", "daily/test.md")
	if !strings.Contains(result, "- [x] unchecked task") {
		t.Error("task was not checked")
	}

	// Toggle it back
	err = daily.ToggleTask("sebastian", "daily/test.md", 5)
	if err != nil {
		t.Fatalf("ToggleTask() error = %v", err)
	}

	result, _ = store.ReadFile("sebastian", "daily/test.md")
	if !strings.Contains(result, "- [ ] unchecked task") {
		t.Error("task was not unchecked")
	}
}

func TestDaily_ClearAllPinned(t *testing.T) {
	daily, store, _ := setupDailyTest(t)

	content := `# 2024-01-15

## custom notes

### 10:00 <pinned>
Pinned note 1

### 11:00
Regular note

### 12:00 <pinned>
Pinned note 2
`
	err := store.WriteFile("sebastian", "daily/test.md", content)
	if err != nil {
		t.Fatal(err)
	}

	err = daily.ClearAllPinned("sebastian", "daily/test.md")
	if err != nil {
		t.Fatalf("ClearAllPinned() error = %v", err)
	}

	result, _ := store.ReadFile("sebastian", "daily/test.md")
	if strings.Contains(result, "<pinned>") {
		t.Error("pinned markers still present")
	}
	if !strings.Contains(result, "### 10:00") {
		t.Error("time header missing")
	}
}

func TestDaily_UnpinEntry(t *testing.T) {
	daily, store, _ := setupDailyTest(t)

	content := `# 2024-01-15

## custom notes

### 10:00 <pinned>
Pinned note 1

### 12:00 <pinned>
Pinned note 2
`
	err := store.WriteFile("sebastian", "daily/test.md", content)
	if err != nil {
		t.Fatal(err)
	}

	// Unpin line 5 (### 10:00 <pinned>)
	err = daily.UnpinEntry("sebastian", "daily/test.md", 5)
	if err != nil {
		t.Fatalf("UnpinEntry() error = %v", err)
	}

	result, _ := store.ReadFile("sebastian", "daily/test.md")
	lines := strings.Split(result, "\n")

	if strings.Contains(lines[4], "<pinned>") {
		t.Error("first entry still pinned")
	}
	if !strings.Contains(result, "### 12:00 <pinned>") {
		t.Error("second entry should still be pinned")
	}
}

func TestDaily_AppendEntry(t *testing.T) {
	daily, store, _ := setupDailyTest(t)

	content := `# 2024-01-15

## custom notes
`
	err := store.WriteFile("sebastian", "daily/test.md", content)
	if err != nil {
		t.Fatal(err)
	}

	// Append regular entry
	err = daily.AppendEntry("sebastian", "daily/test.md", "Regular note content", false)
	if err != nil {
		t.Fatalf("AppendEntry() error = %v", err)
	}

	result, _ := store.ReadFile("sebastian", "daily/test.md")
	if !strings.Contains(result, "Regular note content") {
		t.Error("entry content not found")
	}
	if strings.Contains(result, "<pinned>") {
		t.Error("should not be pinned")
	}

	// Append pinned entry
	err = daily.AppendEntry("sebastian", "daily/test.md", "Pinned content", true)
	if err != nil {
		t.Fatalf("AppendEntry() error = %v", err)
	}

	result, _ = store.ReadFile("sebastian", "daily/test.md")
	if !strings.Contains(result, "<pinned>") {
		t.Error("pinned marker not found")
	}
	if !strings.Contains(result, "Pinned content") {
		t.Error("pinned content not found")
	}
}
