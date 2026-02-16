package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestVault(t *testing.T) (*Store, string) {
	t.Helper()
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create person directories
	if err := os.MkdirAll(filepath.Join(tmpDir, "sebastian", "daily"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "petra"), 0755); err != nil {
		t.Fatal(err)
	}

	return store, tmpDir
}

func TestStore_ReadWriteFile(t *testing.T) {
	store, _ := setupTestVault(t)

	// Write a file
	content := "# Test Note\n\nThis is test content."
	err := store.WriteFile("sebastian", "daily/test.md", content)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Read it back
	got, err := store.ReadFile("sebastian", "daily/test.md")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got != content {
		t.Errorf("ReadFile() = %q, want %q", got, content)
	}

	// Read non-existent file
	_, err = store.ReadFile("sebastian", "nonexistent.md")
	if !os.IsNotExist(err) {
		t.Errorf("ReadFile(nonexistent) error = %v, want os.IsNotExist", err)
	}
}

func TestStore_WriteFile_CreatesDirectories(t *testing.T) {
	store, _ := setupTestVault(t)

	content := "nested content"
	err := store.WriteFile("sebastian", "deep/nested/path/file.md", content)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := store.ReadFile("sebastian", "deep/nested/path/file.md")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got != content {
		t.Errorf("ReadFile() = %q, want %q", got, content)
	}
}

func TestStore_AppendFile(t *testing.T) {
	store, _ := setupTestVault(t)

	// Write initial content
	err := store.WriteFile("sebastian", "append.md", "line1\n")
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Append content
	err = store.AppendFile("sebastian", "append.md", "line2\n")
	if err != nil {
		t.Fatalf("AppendFile() error = %v", err)
	}

	got, err := store.ReadFile("sebastian", "append.md")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := "line1\nline2\n"
	if got != want {
		t.Errorf("After append, content = %q, want %q", got, want)
	}
}

func TestStore_AppendFile_CreatesFile(t *testing.T) {
	store, _ := setupTestVault(t)

	// Append to non-existent file
	err := store.AppendFile("sebastian", "new/file.md", "content")
	if err != nil {
		t.Fatalf("AppendFile() error = %v", err)
	}

	got, err := store.ReadFile("sebastian", "new/file.md")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got != "content" {
		t.Errorf("ReadFile() = %q, want %q", got, "content")
	}
}

func TestStore_DeleteFile(t *testing.T) {
	store, _ := setupTestVault(t)

	// Create a file
	err := store.WriteFile("sebastian", "todelete.md", "content")
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Delete it
	err = store.DeleteFile("sebastian", "todelete.md")
	if err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}

	// Verify it's gone
	exists, err := store.FileExists("sebastian", "todelete.md")
	if err != nil {
		t.Fatalf("FileExists() error = %v", err)
	}
	if exists {
		t.Error("File still exists after delete")
	}
}

func TestStore_DeleteFile_Idempotent(t *testing.T) {
	store, _ := setupTestVault(t)

	// Delete non-existent file should not error
	err := store.DeleteFile("sebastian", "nonexistent.md")
	if err != nil {
		t.Errorf("DeleteFile(nonexistent) error = %v, want nil", err)
	}
}

func TestStore_ListDir(t *testing.T) {
	store, tmpDir := setupTestVault(t)

	// Create test files and directories
	testDir := filepath.Join(tmpDir, "sebastian", "testdir")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file1.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file2.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(testDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	// Hidden file should be filtered
	if err := os.WriteFile(filepath.Join(testDir, ".hidden"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := store.ListDir("sebastian", "testdir")
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	// Should have 2 files + 1 directory, no hidden file
	if len(entries) != 3 {
		t.Errorf("ListDir() returned %d entries, want 3", len(entries))
	}

	// Files should come first, sorted alphabetically
	if entries[0].Name != "file1.md" || entries[0].IsDir {
		t.Errorf("First entry = %+v, want file1.md (file)", entries[0])
	}
	if entries[1].Name != "file2.md" || entries[1].IsDir {
		t.Errorf("Second entry = %+v, want file2.md (file)", entries[1])
	}
	if entries[2].Name != "subdir" || !entries[2].IsDir {
		t.Errorf("Third entry = %+v, want subdir (dir)", entries[2])
	}
}

func TestStore_ListDir_EmptyDirectoryReturnsEmptySlice(t *testing.T) {
	store, tmpDir := setupTestVault(t)

	emptyDir := filepath.Join(tmpDir, "sebastian", "emptydir")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	entries, err := store.ListDir("sebastian", "emptydir")
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if entries == nil {
		t.Fatal("ListDir() returned nil, want empty slice")
	}
	if len(entries) != 0 {
		t.Errorf("ListDir() returned %d entries, want 0", len(entries))
	}
}

func TestStore_PathTraversal(t *testing.T) {
	store, _ := setupTestVault(t)

	// Attempt path traversal on read
	_, err := store.ReadFile("sebastian", "../petra/secret.md")
	if err == nil {
		t.Error("ReadFile() with traversal should error")
	}

	// Attempt path traversal on write
	err = store.WriteFile("sebastian", "../petra/evil.md", "malicious")
	if err == nil {
		t.Error("WriteFile() with traversal should error")
	}

	// Attempt path traversal on delete
	err = store.DeleteFile("sebastian", "../../etc/passwd")
	if err == nil {
		t.Error("DeleteFile() with traversal should error")
	}
}

func TestStore_PersonIsolation(t *testing.T) {
	store, _ := setupTestVault(t)

	// Write file for sebastian
	err := store.WriteFile("sebastian", "private.md", "sebastian's data")
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Petra should not see sebastian's file
	_, err = store.ReadFile("petra", "private.md")
	if !os.IsNotExist(err) {
		t.Errorf("Petra can read sebastian's file: error = %v", err)
	}
}
