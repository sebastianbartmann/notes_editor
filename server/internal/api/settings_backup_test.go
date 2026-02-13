package api

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultBackupDownload(t *testing.T) {
	srv, vaultRoot, cleanup := setupTestServer(t)
	defer cleanup()
	router := NewRouter(srv)

	notePath := filepath.Join(vaultRoot, "sebastian", "notes", "backup-test.md")
	if err := os.WriteFile(notePath, []byte("backup content"), 0644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	req := makeRequest(t, "GET", "/api/settings/vault-backup", "", "sebastian")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("content-type = %q, want application/zip", got)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "attachment;") || !strings.Contains(contentDisposition, "sebastian-vault-") {
		t.Fatalf("unexpected content-disposition: %q", contentDisposition)
	}

	body := rec.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("zip.NewReader failed: %v", err)
	}

	found := false
	for _, f := range zr.File {
		if f.Name != "notes/backup-test.md" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zipped file: %v", err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zipped file: %v", err)
		}
		if string(data) != "backup content" {
			t.Fatalf("zipped file content = %q, want %q", string(data), "backup content")
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("zip did not contain notes/backup-test.md")
	}
}
