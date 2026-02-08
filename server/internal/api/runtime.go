package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"notes-editor/internal/agent"
	"notes-editor/internal/auth"
	"notes-editor/internal/linkedin"
)

type linkedinHealthSnapshot struct {
	TokenPresent bool      `json:"token_present"`
	Configured   bool      `json:"configured"`
	Healthy      bool      `json:"healthy"`
	LastChecked  time.Time `json:"last_checked,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
}

type gatewayHealthSnapshot struct {
	URL         string    `json:"url"`
	Configured  bool      `json:"configured"`
	Reachable   bool      `json:"reachable"`
	Healthy     bool      `json:"healthy"`
	Mode        string    `json:"mode,omitempty"`
	LastChecked time.Time `json:"last_checked,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
}

func (s *Server) getAgent() *agent.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agent
}

func (s *Server) getLinkedIn() *linkedin.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.linkedin
}

func (s *Server) reloadRuntimeServices() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.config.ReloadRuntimeSettings(); err != nil {
		return err
	}
	auth.SetValidPersons(s.config.ValidPersons)

	s.linkedin, s.claude, s.agent = buildRuntimeServices(s.config, s.store)
	return nil
}

func (s *Server) linkedinHealth(ctx context.Context) linkedinHealthSnapshot {
	snapshot := linkedinHealthSnapshot{}

	s.mu.RLock()
	token := s.config.LinkedIn.AccessToken
	service := s.linkedin
	s.mu.RUnlock()

	snapshot.TokenPresent = token != ""
	snapshot.Configured = service != nil
	if service == nil {
		snapshot.Healthy = false
		if snapshot.TokenPresent {
			snapshot.LastError = "LinkedIn service not initialized"
		}
		return snapshot
	}

	start := time.Now()
	snapshot.LastChecked = start
	if err := service.Validate(ctx); err != nil {
		snapshot.Healthy = false
		snapshot.LastError = err.Error()
		return snapshot
	}
	snapshot.Healthy = true
	return snapshot
}

func (s *Server) gatewayHealth(ctx context.Context) gatewayHealthSnapshot {
	snapshot := gatewayHealthSnapshot{}

	s.mu.RLock()
	baseURL := strings.TrimSpace(s.config.PiGatewayURL)
	s.mu.RUnlock()

	snapshot.URL = baseURL
	snapshot.Configured = baseURL != ""
	if !snapshot.Configured {
		snapshot.Healthy = false
		snapshot.LastError = "PI_GATEWAY_URL not configured"
		return snapshot
	}

	snapshot.LastChecked = time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		snapshot.LastError = fmt.Sprintf("invalid gateway URL: %v", err)
		return snapshot
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		snapshot.LastError = err.Error()
		return snapshot
	}
	defer resp.Body.Close()

	snapshot.Reachable = true
	if resp.StatusCode != http.StatusOK {
		snapshot.LastError = fmt.Sprintf("gateway /health returned status %d", resp.StatusCode)
		return snapshot
	}

	var payload struct {
		OK   bool   `json:"ok"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		snapshot.LastError = "invalid gateway health response"
		return snapshot
	}

	snapshot.Mode = payload.Mode
	snapshot.Healthy = payload.OK
	if !payload.OK {
		snapshot.LastError = "gateway reported unhealthy"
	}

	return snapshot
}
