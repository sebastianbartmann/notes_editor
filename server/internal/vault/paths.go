// Package vault provides file operations and git sync for the notes vault.
package vault

import (
	"errors"
	"path/filepath"
	"strings"
)

// Path validation errors.
var (
	ErrEmptyPath    = errors.New("path cannot be empty")
	ErrAbsolutePath = errors.New("absolute paths are not allowed")
	ErrPathEscape   = errors.New("path escapes vault root")
)

// ValidatePath validates a relative path for safety.
// It rejects empty paths, absolute paths, and paths that would escape the vault.
func ValidatePath(path string) error {
	if path == "" {
		return ErrEmptyPath
	}

	if filepath.IsAbs(path) {
		return ErrAbsolutePath
	}

	// Clean the path and check for traversal attempts
	cleaned := filepath.Clean(path)
	if strings.HasPrefix(cleaned, "..") {
		return ErrPathEscape
	}

	return nil
}

// ResolvePath safely joins the vault root, person directory, and relative path.
// It validates the path and ensures the result stays within the person's vault.
func ResolvePath(vaultRoot, person, relativePath string) (string, error) {
	if err := ValidatePath(relativePath); err != nil {
		return "", err
	}

	// Build the full path
	personRoot := filepath.Join(vaultRoot, person)
	fullPath := filepath.Join(personRoot, relativePath)

	// Ensure the resolved path stays within the person's vault
	// Use Clean to normalize both paths before comparison
	cleanedPersonRoot := filepath.Clean(personRoot)
	cleanedFullPath := filepath.Clean(fullPath)

	if !strings.HasPrefix(cleanedFullPath, cleanedPersonRoot+string(filepath.Separator)) &&
		cleanedFullPath != cleanedPersonRoot {
		return "", ErrPathEscape
	}

	return fullPath, nil
}

// ResolveRootPath resolves a path relative to the vault root (not person-scoped).
// Used for shared files like sleep_times.md.
func ResolveRootPath(vaultRoot, relativePath string) (string, error) {
	if err := ValidatePath(relativePath); err != nil {
		return "", err
	}

	fullPath := filepath.Join(vaultRoot, relativePath)

	// Ensure the resolved path stays within the vault root
	cleanedRoot := filepath.Clean(vaultRoot)
	cleanedFullPath := filepath.Clean(fullPath)

	if !strings.HasPrefix(cleanedFullPath, cleanedRoot+string(filepath.Separator)) &&
		cleanedFullPath != cleanedRoot {
		return "", ErrPathEscape
	}

	return fullPath, nil
}
