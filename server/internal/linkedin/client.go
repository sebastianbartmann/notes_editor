package linkedin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"notes-editor/internal/config"
)

// Service provides LinkedIn API operations.
type Service struct {
	config    *config.LinkedInConfig
	vaultRoot string
	client    *http.Client
}

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
	req, err := http.NewRequest("GET", "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Sub string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return "urn:li:person:" + result.Sub, nil
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
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
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
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
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
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
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

