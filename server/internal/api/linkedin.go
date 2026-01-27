package api

import (
	"net/http"
	"path/filepath"

	"notes-editor/internal/linkedin"
)

// handleLinkedInCallback handles the OAuth callback from LinkedIn.
func (s *Server) handleLinkedInCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeBadRequest(w, "Authorization code is required")
		return
	}

	// Check if LinkedIn is configured
	if s.config.LinkedIn.ClientID == "" || s.config.LinkedIn.ClientSecret == "" {
		writeBadRequest(w, "LinkedIn OAuth not configured")
		return
	}

	// Exchange code for token
	tokenResp, err := linkedin.ExchangeCodeForToken(&s.config.LinkedIn, code)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	// Persist the token
	envPath := filepath.Join(s.config.NotesRoot, "..", ".env")
	if err := linkedin.PersistAccessToken(envPath, tokenResp.AccessToken); err != nil {
		writeBadRequest(w, "Failed to save token: "+err.Error())
		return
	}

	// Update runtime config
	s.config.LinkedIn.AccessToken = tokenResp.AccessToken

	// Reinitialize LinkedIn service
	s.linkedin = linkedin.NewService(&s.config.LinkedIn, s.config.NotesRoot)

	writeSuccess(w, "LinkedIn connected successfully")
}
