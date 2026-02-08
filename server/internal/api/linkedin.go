package api

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

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
	s.mu.Lock()
	s.config.LinkedIn.AccessToken = tokenResp.AccessToken
	s.mu.Unlock()
	if err := s.reloadRuntimeServices(); err != nil {
		writeBadRequest(w, "Token saved but runtime reload failed: "+err.Error())
		return
	}

	writeSuccess(w, "LinkedIn connected successfully")
}

// handleLinkedInHealth reports LinkedIn runtime health and token status.
func (s *Server) handleLinkedInHealth(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePerson(w, r); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	health := s.linkedinHealth(ctx)
	writeJSON(w, http.StatusOK, health)
}
