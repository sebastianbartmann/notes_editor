package linkedin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"notes-editor/internal/config"
)

// Service provides LinkedIn API operations.
type Service struct {
	config    *config.LinkedInConfig
	vaultRoot string
	client    *http.Client
}

const (
	linkedinRetryAttempts = 3
	linkedinRetryDelay    = 300 * time.Millisecond
)

// NewService creates a new LinkedIn service.
func NewService(cfg *config.LinkedInConfig, vaultRoot string) *Service {
	return &Service{
		config:    cfg,
		vaultRoot: vaultRoot,
		client:    &http.Client{},
	}
}

// IsConfigured returns true if LinkedIn is configured with an access token.
func (s *Service) IsConfigured() bool {
	return s.config.AccessToken != ""
}

// GetPersonURN retrieves the authenticated user's URN.
func (s *Service) GetPersonURN() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.getPersonURNWithRetry(ctx)
}

// CreatePost creates a new LinkedIn post.
func (s *Service) CreatePost(text, person string) (string, error) {
	personURN, err := s.GetPersonURN()
	if err != nil {
		return "", fmt.Errorf("failed to get person URN: %w", err)
	}

	payload := map[string]any{
		"author":         personURN,
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]any{
			"com.linkedin.ugc.ShareContent": map[string]any{
				"shareCommentary": map[string]any{
					"text": text,
				},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]any{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.linkedin.com/v2/ugcPosts", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return "", mapLinkedInAPIError(resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Log the post
	if err := s.LogPost(person, result.ID, text, string(respBody)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log post: %v\n", err)
	}

	return result.ID, nil
}

// ReadComments retrieves comments on a LinkedIn post.
func (s *Service) ReadComments(postURN string) (string, error) {
	encodedURN := url.PathEscape(postURN)
	apiURL := fmt.Sprintf("https://api.linkedin.com/v2/socialActions/%s/comments", encodedURN)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", mapLinkedInAPIError(resp.StatusCode, string(body))
	}

	return string(body), nil
}

// CreateComment posts a comment on a LinkedIn post.
// If parentURN is provided, it creates a reply to that comment.
func (s *Service) CreateComment(postURN, text, parentURN, person string) (string, error) {
	personURN, err := s.GetPersonURN()
	if err != nil {
		return "", fmt.Errorf("failed to get person URN: %w", err)
	}

	encodedURN := url.PathEscape(postURN)
	apiURL := fmt.Sprintf("https://api.linkedin.com/v2/socialActions/%s/comments", encodedURN)

	payload := map[string]any{
		"actor": personURN,
		"message": map[string]any{
			"text": text,
		},
	}

	if parentURN != "" {
		payload["parentComment"] = parentURN
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return "", mapLinkedInAPIError(resp.StatusCode, string(respBody))
	}

	// Parse response to get comment URN
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	commentURN, _ := result["$URN"].(string)
	if commentURN == "" {
		commentURN = "unknown"
	}

	// Log the comment
	action := "comment"
	if parentURN != "" {
		action = "reply"
	}
	if err := s.LogComment(person, action, postURN, commentURN, text, string(respBody)); err != nil {
		fmt.Printf("Failed to log comment: %v\n", err)
	}

	return string(respBody), nil
}

// Validate checks whether the configured access token is currently usable.
func (s *Service) Validate(ctx context.Context) error {
	_, err := s.getPersonURNWithRetry(ctx)
	return err
}

func (s *Service) getPersonURNWithRetry(ctx context.Context) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= linkedinRetryAttempts; attempt++ {
		urn, retry, err := s.getPersonURNOnce(ctx)
		if err == nil {
			return urn, nil
		}
		lastErr = err
		if !retry || attempt == linkedinRetryAttempts {
			break
		}

		select {
		case <-time.After(linkedinRetryDelay * time.Duration(attempt)):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if lastErr == nil {
		lastErr = errors.New("unknown linkedin validation failure")
	}
	return "", lastErr
}

func (s *Service) getPersonURNOnce(ctx context.Context) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return "", true, fmt.Errorf("linkedin network error: %w", err)
		}
		return "", false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return "", retry, mapLinkedInAPIError(resp.StatusCode, string(body))
	}

	var result struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	return "urn:li:person:" + result.Sub, false, nil
}

func mapLinkedInAPIError(status int, body string) error {
	body = strings.TrimSpace(body)
	switch status {
	case http.StatusUnauthorized:
		return fmt.Errorf("linkedin authentication failed (401): reconnect LinkedIn in Settings")
	case http.StatusForbidden:
		return fmt.Errorf("linkedin authorization failed (403): token missing required scopes or expired")
	case http.StatusTooManyRequests:
		return fmt.Errorf("linkedin rate limit reached (429)")
	default:
		if body == "" {
			return fmt.Errorf("linkedin API error (status %d)", status)
		}
		return fmt.Errorf("linkedin API error (status %d): %s", status, body)
	}
}
