// Package config handles loading and validating configuration from environment variables.
package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	// NotesToken is the bearer token for API authentication.
	NotesToken string
	// NotesRoot is the root path for the notes vault.
	NotesRoot string
	// AnthropicKey is the API key for Claude AI service.
	AnthropicKey string
	// StaticDir is the directory for serving static web UI files.
	StaticDir string
	// LinkedIn configuration for OAuth and API access.
	LinkedIn LinkedInConfig
}

// LinkedInConfig holds LinkedIn OAuth and API configuration.
type LinkedInConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AccessToken  string
	// TokenURL is the OAuth token exchange endpoint. Defaults to LinkedIn's URL if empty.
	TokenURL string
}

// Load reads configuration from environment variables.
// It loads .env file if present, but environment variables take precedence.
func Load() (*Config, error) {
	// Load .env file if present (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		NotesToken:   os.Getenv("NOTES_TOKEN"),
		NotesRoot:    os.Getenv("NOTES_ROOT"),
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		StaticDir:    os.Getenv("STATIC_DIR"),
		LinkedIn: LinkedInConfig{
			ClientID:     os.Getenv("LINKEDIN_CLIENT_ID"),
			ClientSecret: os.Getenv("LINKEDIN_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("LINKEDIN_REDIRECT_URI"),
			AccessToken:  os.Getenv("LINKEDIN_ACCESS_TOKEN"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration fields are set.
func (c *Config) Validate() error {
	if c.NotesToken == "" {
		return errors.New("NOTES_TOKEN is required")
	}
	if c.NotesRoot == "" {
		return errors.New("NOTES_ROOT is required")
	}
	// AnthropicKey is optional - Claude features will be disabled without it
	// LinkedIn config is optional - LinkedIn features will be disabled without it
	return nil
}

// ReloadLinkedInToken reloads the LinkedIn access token from the .env file.
// This is used after OAuth token exchange to pick up the newly saved token.
func (c *Config) ReloadLinkedInToken() error {
	// Force reload .env file
	if err := godotenv.Overload(); err != nil {
		return err
	}
	c.LinkedIn.AccessToken = os.Getenv("LINKEDIN_ACCESS_TOKEN")
	return nil
}
