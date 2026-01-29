// Package linkedin provides LinkedIn API integration for posting and comments.
package linkedin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"notes-editor/internal/config"
)

// TokenResponse represents the OAuth token exchange response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// DefaultTokenURL is the LinkedIn OAuth token exchange endpoint.
const DefaultTokenURL = "https://www.linkedin.com/oauth/v2/accessToken"

// ExchangeCodeForToken exchanges an authorization code for an access token.
func ExchangeCodeForToken(cfg *config.LinkedInConfig, code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", cfg.RedirectURI)
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)

	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// PersistAccessToken saves the access token to the .env file.
func PersistAccessToken(envPath, token string) error {
	// Read existing .env file
	file, err := os.Open(envPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to open .env file: %w", err)
	}

	var lines []string
	tokenFound := false

	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "LINKEDIN_ACCESS_TOKEN=") {
				lines = append(lines, "LINKEDIN_ACCESS_TOKEN="+token)
				tokenFound = true
			} else {
				lines = append(lines, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read .env file: %w", err)
		}
	}

	if !tokenFound {
		lines = append(lines, "LINKEDIN_ACCESS_TOKEN="+token)
	}

	// Write back
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}
