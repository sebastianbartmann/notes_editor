// Package config handles loading and validating configuration from environment variables.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	// ServerAddr is the HTTP listen address (e.g., :80, :8080).
	ServerAddr string
	// ValidPersons defines accepted X-Notes-Person values.
	ValidPersons []string
	// LinkedIn configuration for OAuth and API access.
	LinkedIn LinkedInConfig
	// PiGatewayURL is the local gateway sidecar endpoint base URL.
	PiGatewayURL string
	// AgentEnablePiFallback toggles fallback from gateway_subscription to anthropic_api_key.
	AgentEnablePiFallback bool
	// AgentMaxRunDuration bounds one run's lifetime.
	AgentMaxRunDuration time.Duration
	// AgentMaxToolCallsPerRun bounds tool calls emitted in one run.
	AgentMaxToolCallsPerRun int
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
		ServerAddr:   os.Getenv("SERVER_ADDR"),
		ValidPersons: parseCSV(os.Getenv("VALID_PERSONS")),
		PiGatewayURL: strings.TrimSpace(os.Getenv("PI_GATEWAY_URL")),
		LinkedIn: LinkedInConfig{
			ClientID:     os.Getenv("LINKEDIN_CLIENT_ID"),
			ClientSecret: os.Getenv("LINKEDIN_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("LINKEDIN_REDIRECT_URI"),
			AccessToken:  os.Getenv("LINKEDIN_ACCESS_TOKEN"),
			TokenURL:     os.Getenv("LINKEDIN_TOKEN_URL"),
		},
	}
	cfg.AgentEnablePiFallback = parseBoolEnv("AGENT_ENABLE_PI_FALLBACK", true)
	cfg.AgentMaxRunDuration = parseDurationEnv("AGENT_MAX_RUN_DURATION", 45*time.Minute)
	cfg.AgentMaxToolCallsPerRun = parseIntEnv("AGENT_MAX_TOOL_CALLS_PER_RUN", 40)
	if cfg.PiGatewayURL == "" {
		cfg.PiGatewayURL = "http://127.0.0.1:4317"
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
	if c.ServerAddr == "" {
		c.ServerAddr = ":80"
	}
	if len(c.ValidPersons) == 0 {
		c.ValidPersons = []string{"sebastian", "petra"}
	}
	// AnthropicKey is optional - Claude features will be disabled without it
	// LinkedIn config is optional - LinkedIn features will be disabled without it
	return nil
}

// ReloadLinkedInToken reloads the LinkedIn access token from the .env file.
// This is used after OAuth token exchange to pick up the newly saved token.
func (c *Config) ReloadLinkedInToken() error {
	// Force reload .env file
	if err := godotenv.Overload(c.envPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	c.LinkedIn.AccessToken = os.Getenv("LINKEDIN_ACCESS_TOKEN")
	return nil
}

// ReloadRuntimeSettings reloads runtime-relevant values from the environment/.env.
func (c *Config) ReloadRuntimeSettings() error {
	if err := godotenv.Overload(c.envPath()); err != nil && !os.IsNotExist(err) {
		return err
	}

	c.AnthropicKey = os.Getenv("ANTHROPIC_API_KEY")
	c.PiGatewayURL = strings.TrimSpace(os.Getenv("PI_GATEWAY_URL"))
	if c.PiGatewayURL == "" {
		c.PiGatewayURL = "http://127.0.0.1:4317"
	}
	c.AgentEnablePiFallback = parseBoolEnv("AGENT_ENABLE_PI_FALLBACK", true)
	c.AgentMaxRunDuration = parseDurationEnv("AGENT_MAX_RUN_DURATION", 45*time.Minute)
	c.AgentMaxToolCallsPerRun = parseIntEnv("AGENT_MAX_TOOL_CALLS_PER_RUN", 40)
	c.ValidPersons = parseCSV(os.Getenv("VALID_PERSONS"))
	if len(c.ValidPersons) == 0 {
		c.ValidPersons = []string{"sebastian", "petra"}
	}
	c.LinkedIn.ClientID = os.Getenv("LINKEDIN_CLIENT_ID")
	c.LinkedIn.ClientSecret = os.Getenv("LINKEDIN_CLIENT_SECRET")
	c.LinkedIn.RedirectURI = os.Getenv("LINKEDIN_REDIRECT_URI")
	c.LinkedIn.AccessToken = os.Getenv("LINKEDIN_ACCESS_TOKEN")
	c.LinkedIn.TokenURL = os.Getenv("LINKEDIN_TOKEN_URL")

	return nil
}

func (c *Config) envPath() string {
	return filepath.Join(c.NotesRoot, "..", ".env")
}

func parseCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseBoolEnv(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func parseIntEnv(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func parseDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}
