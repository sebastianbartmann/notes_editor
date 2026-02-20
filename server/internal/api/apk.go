package api

import (
	"net/http"
	"os"
	"path/filepath"
)

// handleDownloadAPK serves the Android debug APK from the project root.
func (s *Server) handleDownloadAPK(w http.ResponseWriter, r *http.Request) {
	cwd, err := os.Getwd()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to resolve working directory")
		return
	}

	projectRoot := filepath.Clean(filepath.Join(cwd, ".."))
	apkPath := filepath.Join(projectRoot, "app", "android", "app", "build", "outputs", "apk", "debug", "app-debug.apk")

	if _, err := os.Stat(apkPath); err != nil {
		if os.IsNotExist(err) {
			writeNotFound(w, "APK file not found")
			return
		}
		writeBadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", `attachment; filename="app-debug.apk"`)
	http.ServeFile(w, r, apkPath)
}
