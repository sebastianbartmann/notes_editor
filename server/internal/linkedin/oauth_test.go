package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"notes-editor/internal/config"
)

func TestExchangeCodeForToken_Success(t *testing.T) {
	// Create mock server that returns a valid token response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and content type
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("Expected grant_type 'authorization_code', got %q", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "test-auth-code" {
			t.Errorf("Expected code 'test-auth-code', got %q", r.FormValue("code"))
		}
		if r.FormValue("redirect_uri") != "https://example.com/callback" {
			t.Errorf("Expected redirect_uri 'https://example.com/callback', got %q", r.FormValue("redirect_uri"))
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("Expected client_id 'test-client-id', got %q", r.FormValue("client_id"))
		}
		if r.FormValue("client_secret") != "test-client-secret" {
			t.Errorf("Expected client_secret 'test-client-secret', got %q", r.FormValue("client_secret"))
		}

		// Return successful response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "test-access-token-12345",
			ExpiresIn:   3600,
			Scope:       "r_liteprofile w_member_social",
		})
	}))
	defer server.Close()

	cfg := &config.LinkedInConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
		TokenURL:     server.URL,
	}

	token, err := ExchangeCodeForToken(cfg, "test-auth-code")
	if err != nil {
		t.Fatalf("ExchangeCodeForToken failed: %v", err)
	}

	if token.AccessToken != "test-access-token-12345" {
		t.Errorf("Expected access_token 'test-access-token-12345', got %q", token.AccessToken)
	}
	if token.ExpiresIn != 3600 {
		t.Errorf("Expected expires_in 3600, got %d", token.ExpiresIn)
	}
	if token.Scope != "r_liteprofile w_member_social" {
		t.Errorf("Expected scope 'r_liteprofile w_member_social', got %q", token.Scope)
	}
}

func TestExchangeCodeForToken_HTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"forbidden", http.StatusForbidden},
		{"internal server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"error": "test error"}`))
			}))
			defer server.Close()

			cfg := &config.LinkedInConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURI:  "https://example.com/callback",
				TokenURL:     server.URL,
			}

			_, err := ExchangeCodeForToken(cfg, "test-auth-code")
			if err == nil {
				t.Fatal("Expected error for non-200 status code")
			}
			if !strings.Contains(err.Error(), "token exchange failed with status") {
				t.Errorf("Expected status error, got: %v", err)
			}
		})
	}
}

func TestExchangeCodeForToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	cfg := &config.LinkedInConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
		TokenURL:     server.URL,
	}

	_, err := ExchangeCodeForToken(cfg, "test-auth-code")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to decode token response") {
		t.Errorf("Expected decode error, got: %v", err)
	}
}

func TestExchangeCodeForToken_NetworkError(t *testing.T) {
	cfg := &config.LinkedInConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
		TokenURL:     "http://localhost:1", // Invalid port, should fail to connect
	}

	_, err := ExchangeCodeForToken(cfg, "test-auth-code")
	if err == nil {
		t.Fatal("Expected error for network failure")
	}
	if !strings.Contains(err.Error(), "token exchange request failed") {
		t.Errorf("Expected request error, got: %v", err)
	}
}

func TestExchangeCodeForToken_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{})
	}))
	defer server.Close()

	cfg := &config.LinkedInConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
		TokenURL:     server.URL,
	}

	token, err := ExchangeCodeForToken(cfg, "test-auth-code")
	if err != nil {
		t.Fatalf("ExchangeCodeForToken failed: %v", err)
	}

	// Empty response should parse successfully but with empty values
	if token.AccessToken != "" {
		t.Errorf("Expected empty access_token, got %q", token.AccessToken)
	}
}

func TestExchangeCodeForToken_DefaultURL(t *testing.T) {
	if DefaultTokenURL != "https://www.linkedin.com/oauth/v2/accessToken" {
		t.Errorf("DefaultTokenURL should be LinkedIn's token endpoint, got %q", DefaultTokenURL)
	}
}

func TestPersistAccessToken_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	err := PersistAccessToken(envPath, "new-token-12345")
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	expected := "LINKEDIN_ACCESS_TOKEN=new-token-12345\n"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestPersistAccessToken_UpdateExistingToken(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Create existing .env with token
	initialContent := "NOTES_TOKEN=abc123\nLINKEDIN_ACCESS_TOKEN=old-token\nOTHER_VAR=value\n"
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial .env: %v", err)
	}

	err := PersistAccessToken(envPath, "new-token-67890")
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	expected := "NOTES_TOKEN=abc123\nLINKEDIN_ACCESS_TOKEN=new-token-67890\nOTHER_VAR=value\n"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestPersistAccessToken_AddToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Create existing .env without LinkedIn token
	initialContent := "NOTES_TOKEN=abc123\nOTHER_VAR=value\n"
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial .env: %v", err)
	}

	err := PersistAccessToken(envPath, "added-token")
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	expected := "NOTES_TOKEN=abc123\nOTHER_VAR=value\nLINKEDIN_ACCESS_TOKEN=added-token\n"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestPersistAccessToken_PreservesFileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Create file with various formats
	initialContent := "# Comment line\nNOTES_TOKEN=abc123\n\nLINKEDIN_ACCESS_TOKEN=old\nEMPTY=\n"
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial .env: %v", err)
	}

	err := PersistAccessToken(envPath, "updated")
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	// Should preserve comments, empty lines, and other vars
	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("Expected 5 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "# Comment line" {
		t.Errorf("Expected comment preserved, got %q", lines[0])
	}
	if lines[2] != "" {
		t.Errorf("Expected empty line preserved, got %q", lines[2])
	}
	if lines[3] != "LINKEDIN_ACCESS_TOKEN=updated" {
		t.Errorf("Expected updated token, got %q", lines[3])
	}
}

func TestPersistAccessToken_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Token with special characters (should be handled correctly)
	token := "AQVn-x7YZ_abc+def/ghi=123"
	err := PersistAccessToken(envPath, token)
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	expected := "LINKEDIN_ACCESS_TOKEN=" + token + "\n"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestPersistAccessToken_EmptyToken(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	err := PersistAccessToken(envPath, "")
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	expected := "LINKEDIN_ACCESS_TOKEN=\n"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestPersistAccessToken_FileWithoutTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Create file without trailing newline
	initialContent := "NOTES_TOKEN=abc"
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial .env: %v", err)
	}

	err := PersistAccessToken(envPath, "token123")
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	// Should add trailing newline
	if !strings.HasSuffix(string(content), "\n") {
		t.Error("Expected trailing newline")
	}
	if !strings.Contains(string(content), "LINKEDIN_ACCESS_TOKEN=token123") {
		t.Error("Expected token to be added")
	}
}

func TestPersistAccessToken_MultipleLinkedInTokenLines(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Create file with duplicate LinkedIn token lines (edge case)
	initialContent := "LINKEDIN_ACCESS_TOKEN=first\nOTHER=x\nLINKEDIN_ACCESS_TOKEN=second\n"
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial .env: %v", err)
	}

	err := PersistAccessToken(envPath, "updated")
	if err != nil {
		t.Fatalf("PersistAccessToken failed: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	// Both lines should be updated
	count := strings.Count(string(content), "LINKEDIN_ACCESS_TOKEN=updated")
	if count != 2 {
		t.Errorf("Expected both token lines to be updated, got content: %q", string(content))
	}
}
