package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// handleGetEnv returns the contents of the .env file.
func (s *Server) handleGetEnv(w http.ResponseWriter, r *http.Request) {
	envPath := filepath.Join(s.config.NotesRoot, "..", ".env")

	content, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]string{
				"content": "",
			})
			return
		}
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"content": string(content),
	})
}

// SetEnvRequest represents a request to update the .env file.
type SetEnvRequest struct {
	Content string `json:"content"`
}

// handleSetEnv updates the .env file contents.
func (s *Server) handleSetEnv(w http.ResponseWriter, r *http.Request) {
	var req SetEnvRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid request body")
		return
	}

	envPath := filepath.Join(s.config.NotesRoot, "..", ".env")

	if err := os.WriteFile(envPath, []byte(req.Content), 0644); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.reloadRuntimeServices(); err != nil {
		writeBadRequest(w, "Settings saved but reload failed: "+err.Error())
		return
	}

	writeSuccess(w, "Settings saved")
}

// handleDownloadVaultBackup streams a ZIP backup for the selected person's vault.
func (s *Server) handleDownloadVaultBackup(w http.ResponseWriter, r *http.Request) {
	person, ok := requirePerson(w, r)
	if !ok {
		return
	}

	s.syncMgr.TriggerPullIfStale(30 * time.Second)

	personRoot := filepath.Join(s.config.NotesRoot, person)
	info, err := os.Stat(personRoot)
	if err != nil {
		if os.IsNotExist(err) {
			writeNotFound(w, "Person vault not found")
			return
		}
		writeBadRequest(w, err.Error())
		return
	}
	if !info.IsDir() {
		writeBadRequest(w, "Person vault is not a directory")
		return
	}

	filename := fmt.Sprintf("%s-vault-%s.zip", person, time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-store")

	zw := zip.NewWriter(w)
	if err := writeVaultZip(zw, personRoot); err != nil {
		log.Printf("backup zip failed for person=%s: %v", person, err)
	}
	if err := zw.Close(); err != nil {
		log.Printf("backup zip close failed for person=%s: %v", person, err)
	}
}

func writeVaultZip(zw *zip.Writer, personRoot string) error {
	return filepath.WalkDir(personRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == personRoot {
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath, err := filepath.Rel(personRoot, path)
		if err != nil {
			return err
		}
		zipPath := filepath.ToSlash(relativePath)

		if d.IsDir() {
			_, err := zw.Create(zipPath + "/")
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = zipPath
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
