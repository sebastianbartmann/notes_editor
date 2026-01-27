package vault

import (
	"errors"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{"valid simple path", "daily/2024-01-15.md", nil},
		{"valid nested path", "notes/work/project.md", nil},
		{"valid single file", "readme.md", nil},
		{"empty path", "", ErrEmptyPath},
		{"absolute path unix", "/etc/passwd", ErrAbsolutePath},
		{"path traversal simple", "../secret.txt", ErrPathEscape},
		{"path traversal nested", "daily/../../../etc/passwd", ErrPathEscape},
		{"path traversal at start", "../../..", ErrPathEscape},
		{"current directory", ".", nil},
		{"hidden file", ".gitignore", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidatePath(%q) = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name         string
		vaultRoot    string
		person       string
		relativePath string
		want         string
		wantErr      error
	}{
		{
			name:         "simple file",
			vaultRoot:    "/vault",
			person:       "sebastian",
			relativePath: "daily/2024-01-15.md",
			want:         "/vault/sebastian/daily/2024-01-15.md",
			wantErr:      nil,
		},
		{
			name:         "nested path",
			vaultRoot:    "/vault",
			person:       "petra",
			relativePath: "notes/work/project.md",
			want:         "/vault/petra/notes/work/project.md",
			wantErr:      nil,
		},
		{
			name:         "root of person vault",
			vaultRoot:    "/vault",
			person:       "sebastian",
			relativePath: ".",
			want:         "/vault/sebastian",
			wantErr:      nil,
		},
		{
			name:         "empty path rejected",
			vaultRoot:    "/vault",
			person:       "sebastian",
			relativePath: "",
			want:         "",
			wantErr:      ErrEmptyPath,
		},
		{
			name:         "absolute path rejected",
			vaultRoot:    "/vault",
			person:       "sebastian",
			relativePath: "/etc/passwd",
			want:         "",
			wantErr:      ErrAbsolutePath,
		},
		{
			name:         "traversal rejected",
			vaultRoot:    "/vault",
			person:       "sebastian",
			relativePath: "../petra/secret.md",
			want:         "",
			wantErr:      ErrPathEscape,
		},
		{
			name:         "traversal out of vault rejected",
			vaultRoot:    "/vault",
			person:       "sebastian",
			relativePath: "../../etc/passwd",
			want:         "",
			wantErr:      ErrPathEscape,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePath(tt.vaultRoot, tt.person, tt.relativePath)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolvePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveRootPath(t *testing.T) {
	tests := []struct {
		name         string
		vaultRoot    string
		relativePath string
		want         string
		wantErr      error
	}{
		{
			name:         "simple file",
			vaultRoot:    "/vault",
			relativePath: "sleep_times.md",
			want:         "/vault/sleep_times.md",
			wantErr:      nil,
		},
		{
			name:         "traversal rejected",
			vaultRoot:    "/vault",
			relativePath: "../etc/passwd",
			want:         "",
			wantErr:      ErrPathEscape,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveRootPath(tt.vaultRoot, tt.relativePath)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ResolveRootPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveRootPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
