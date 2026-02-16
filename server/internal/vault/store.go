package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileEntry represents a file or directory in a listing.
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// Store provides file operations for the notes vault.
type Store struct {
	rootPath string
}

// NewStore creates a new Store with the given root path.
func NewStore(rootPath string) *Store {
	return &Store{rootPath: rootPath}
}

// RootPath returns the vault root path.
func (s *Store) RootPath() string {
	return s.rootPath
}

// ReadFile reads the content of a file within a person's vault.
func (s *Store) ReadFile(person, path string) (string, error) {
	fullPath, err := ResolvePath(s.rootPath, person, path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteFile writes content to a file within a person's vault.
// It creates parent directories if they don't exist.
func (s *Store) WriteFile(person, path, content string) error {
	fullPath, err := ResolvePath(s.rootPath, person, path)
	if err != nil {
		return err
	}

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// AppendFile appends content to a file within a person's vault.
// It creates the file and parent directories if they don't exist.
func (s *Store) AppendFile(person, path, content string) error {
	fullPath, err := ResolvePath(s.rootPath, person, path)
	if err != nil {
		return err
	}

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}

// DeleteFile deletes a file within a person's vault.
// It's idempotent - returns no error if the file doesn't exist.
func (s *Store) DeleteFile(person, path string) error {
	fullPath, err := ResolvePath(s.rootPath, person, path)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	if os.IsNotExist(err) {
		return nil // Idempotent delete
	}
	return err
}

// ListDir lists the contents of a directory within a person's vault.
// It filters out hidden files (starting with '.') and sorts entries
// with files first, then directories, both sorted alphabetically.
func (s *Store) ListDir(person, path string) ([]FileEntry, error) {
	fullPath, err := ResolvePath(s.rootPath, person, path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	files := make([]FileEntry, 0)
	dirs := make([]FileEntry, 0)
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		entryPath := path
		if entryPath == "" || entryPath == "." {
			entryPath = name
		} else {
			entryPath = filepath.Join(path, name)
		}

		fe := FileEntry{
			Name:  name,
			Path:  entryPath,
			IsDir: entry.IsDir(),
		}

		if entry.IsDir() {
			dirs = append(dirs, fe)
		} else {
			files = append(files, fe)
		}
	}

	// Sort files and directories case-insensitively
	sortFunc := func(entries []FileEntry) {
		sort.Slice(entries, func(i, j int) bool {
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})
	}
	sortFunc(files)
	sortFunc(dirs)

	// Files first, then directories
	return append(files, dirs...), nil
}

// FileExists checks if a file exists within a person's vault.
func (s *Store) FileExists(person, path string) (bool, error) {
	fullPath, err := ResolvePath(s.rootPath, person, path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReadRootFile reads a file relative to the vault root (not person-scoped).
func (s *Store) ReadRootFile(path string) (string, error) {
	fullPath, err := ResolveRootPath(s.rootPath, path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteRootFile writes a file relative to the vault root (not person-scoped).
func (s *Store) WriteRootFile(path, content string) error {
	fullPath, err := ResolveRootPath(s.rootPath, path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// AppendRootFile appends content to a file relative to the vault root.
func (s *Store) AppendRootFile(path, content string) error {
	fullPath, err := ResolveRootPath(s.rootPath, path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}
