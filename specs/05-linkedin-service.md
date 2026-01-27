# LinkedIn Service Specification

> Status: Draft
> Version: 2.0
> Last Updated: 2026-01-27

## Overview

This document specifies the LinkedIn API integration for the Notes Editor application. The service provides:

1. **OAuth Authentication**: Token exchange and persistence for LinkedIn API access
2. **Content Operations**: Create posts, read comments, post comments and replies
3. **Activity Logging**: CSV-based logging of all LinkedIn activity per person

The LinkedIn service is used by Claude's tools to enable AI-assisted LinkedIn posting and engagement.

---

## Architecture

### Package Structure

```
server/internal/linkedin/
├── service.go         # Core LinkedIn service
├── service_test.go
├── oauth.go           # OAuth token exchange
├── oauth_test.go
├── client.go          # LinkedIn API client
├── client_test.go
└── logging.go         # Activity CSV logging
```

---

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `LINKEDIN_CLIENT_ID` | Yes | OAuth application client ID |
| `LINKEDIN_CLIENT_SECRET` | Yes | OAuth application secret |
| `LINKEDIN_REDIRECT_URI` | Yes | OAuth callback URL |
| `LINKEDIN_ACCESS_TOKEN` | Yes* | Current access token (*set after OAuth flow) |

### Config Type

```go
type Config struct {
    ClientID     string
    ClientSecret string
    RedirectURI  string
    AccessToken  string
}

func LoadConfig() (*Config, error) {
    cfg := &Config{
        ClientID:     os.Getenv("LINKEDIN_CLIENT_ID"),
        ClientSecret: os.Getenv("LINKEDIN_CLIENT_SECRET"),
        RedirectURI:  os.Getenv("LINKEDIN_REDIRECT_URI"),
        AccessToken:  os.Getenv("LINKEDIN_ACCESS_TOKEN"),
    }
    // Validate required fields
    return cfg, nil
}
```

### File Locations

| Path | Description |
|------|-------------|
| `.env` | Environment file storing credentials |
| `{VAULT_ROOT}/{person}/linkedin/posts.csv` | Activity log per person |

### Constants

```go
const (
    linkedInAPIBase = "https://api.linkedin.com"
    linkedInOAuthURL = "https://www.linkedin.com/oauth/v2/accessToken"
)
```

---

## Service Type

```go
type Service struct {
    config    *Config
    vaultRoot string
    client    *http.Client
}

func NewService(config *Config, vaultRoot string) *Service {
    return &Service{
        config:    config,
        vaultRoot: vaultRoot,
        client:    &http.Client{Timeout: 30 * time.Second},
    }
}
```

---

## OAuth Flow

### Authorization Flow

1. User initiates OAuth via LinkedIn authorization URL
2. LinkedIn redirects to callback with authorization code
3. Server exchanges code for access token
4. Token is persisted to `.env` and runtime config

### Token Exchange

#### `ExchangeCodeForToken(authCode string) (*TokenResponse, error)`

**Endpoint:** `POST https://www.linkedin.com/oauth/v2/accessToken`

```go
type TokenResponse struct {
    AccessToken string `json:"access_token"`
    ExpiresIn   int    `json:"expires_in"`
    Scope       string `json:"scope"`
}

func (s *Service) ExchangeCodeForToken(authCode string) (*TokenResponse, error) {
    data := url.Values{
        "grant_type":    {"authorization_code"},
        "code":          {authCode},
        "redirect_uri":  {s.config.RedirectURI},
        "client_id":     {s.config.ClientID},
        "client_secret": {s.config.ClientSecret},
    }

    resp, err := s.client.PostForm(linkedInOAuthURL, data)
    // ... parse response
}
```

### Token Persistence

```go
func (s *Service) PersistAccessToken(token string) error {
    // Updates LINKEDIN_ACCESS_TOKEN in .env file
    // Updates s.config.AccessToken
}
```

The `updateEnvValue` helper handles `.env` file modification, preserving comments and other variables.

---

## API Client

### Request Headers

```go
func (s *Service) headers() http.Header {
    return http.Header{
        "Authorization": []string{"Bearer " + s.config.AccessToken},
        "Content-Type":  []string{"application/json"},
    }
}
```

### Get Person URN

Retrieves the authenticated user's LinkedIn person URN.

**Endpoint:** `GET /v2/userinfo`

```go
func (s *Service) GetPersonURN() (string, error) {
    // Returns: "urn:li:person:{sub}"
}
```

**Response:**
```json
{
  "sub": "abc123xyz",
  "name": "John Doe",
  "email": "john@example.com"
}
```

---

## Content Operations

### Create Post

Creates a new LinkedIn post as the authenticated user.

```go
func (s *Service) CreatePost(text string, person string) (*PostResponse, error)
```

**Endpoint:** `POST /v2/ugcPosts`

**Payload:**
```json
{
  "author": "urn:li:person:{id}",
  "lifecycleState": "PUBLISHED",
  "specificContent": {
    "com.linkedin.ugc.ShareContent": {
      "shareCommentary": {"text": "Post content here"},
      "shareMediaCategory": "NONE"
    }
  },
  "visibility": {"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"}
}
```

**Response:**
```go
type PostResponse struct {
    ID string `json:"id"` // "urn:li:share:123456789"
}
```

### Read Comments

Retrieves comments for a LinkedIn post.

```go
func (s *Service) ReadComments(postURN string) (*CommentsResponse, error)
```

**Endpoint:** `GET /v2/socialActions/{encoded_urn}/comments`

**Note:** The post URN is URL-encoded (e.g., `urn:li:share:123` becomes `urn%3Ali%3Ashare%3A123`).

**Response:**
```go
type CommentsResponse struct {
    Elements []Comment `json:"elements"`
}

type Comment struct {
    ID      string         `json:"id"`
    Actor   string         `json:"actor"`
    Message CommentMessage `json:"message"`
    Created CommentTime    `json:"created"`
}

type CommentMessage struct {
    Text string `json:"text"`
}

type CommentTime struct {
    Time int64 `json:"time"` // Unix timestamp in milliseconds
}
```

### Create Comment

Posts a comment on a LinkedIn post. Supports both top-level comments and replies.

```go
func (s *Service) CreateComment(postURN, text string, parentCommentURN string, person string) (*CommentResponse, error)
```

**Endpoint:** `POST /v2/socialActions/{encoded_urn}/comments`

**Payload (top-level comment):**
```json
{
  "actor": "urn:li:person:{id}",
  "message": {"text": "Comment text here"}
}
```

**Payload (reply to comment):**
```json
{
  "actor": "urn:li:person:{id}",
  "message": {"text": "Reply text here"},
  "parentComment": "urn:li:comment:456"
}
```

**Response:**
```go
type CommentResponse struct {
    ID string `json:"id"` // "urn:li:comment:789"
}
```

---

## Activity Logging

### Log Format

Activity is logged to CSV files at `{VAULT_ROOT}/{person}/linkedin/posts.csv`.

**CSV Headers:**
```
timestamp,action,post_urn,comment_urn,text,response
```

**Fields:**
| Field | Description |
|-------|-------------|
| `timestamp` | ISO format datetime (YYYY-MM-DD HH:MM:SS) |
| `action` | One of: `post`, `comment`, `reply` |
| `post_urn` | LinkedIn post URN |
| `comment_urn` | LinkedIn comment URN (for replies) |
| `text` | Content that was posted |
| `response` | JSON-encoded API response |

### Logging Functions

```go
func (s *Service) LogPost(person, text string, response *PostResponse) error {
    return s.logActivity(person, ActivityEntry{
        Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
        Action:     "post",
        PostURN:    response.ID,
        Text:       text,
        Response:   mustJSON(response),
    })
}

func (s *Service) LogComment(person, text string, response *CommentResponse, postURN, parentURN string) error {
    action := "comment"
    if parentURN != "" {
        action = "reply"
    }
    return s.logActivity(person, ActivityEntry{
        Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
        Action:      action,
        PostURN:     postURN,
        CommentURN:  response.ID,
        Text:        text,
        Response:    mustJSON(response),
    })
}
```

### Log File Creation

```go
func (s *Service) logActivity(person string, entry ActivityEntry) error {
    logPath := filepath.Join(s.vaultRoot, person, "linkedin", "posts.csv")

    // Create directory if needed
    if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
        return err
    }

    // Check if file exists (for header)
    writeHeader := false
    if _, err := os.Stat(logPath); os.IsNotExist(err) {
        writeHeader = true
    }

    f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    w := csv.NewWriter(f)
    if writeHeader {
        w.Write([]string{"timestamp", "action", "post_urn", "comment_urn", "text", "response"})
    }
    w.Write(entry.ToSlice())
    w.Flush()
    return w.Error()
}
```

---

## Error Handling

### Error Types

```go
var (
    ErrMissingConfig    = errors.New("missing required configuration")
    ErrTokenExchange    = errors.New("token exchange failed")
    ErrAPIRequest       = errors.New("LinkedIn API request failed")
    ErrMissingPersonURN = errors.New("missing person URN in response")
)
```

### API Error Response

```go
type APIError struct {
    StatusCode int
    Body       string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("LinkedIn API error %d: %s", e.StatusCode, e.Body)
}
```

### Error Scenarios

| Error | Cause | Handling |
|-------|-------|----------|
| `ErrMissingConfig` | Missing required env variable | Return error on service creation |
| `ErrTokenExchange` | OAuth token exchange failed | Return to caller (HTTP 400) |
| `ErrAPIRequest` | Non-2xx API response | Return wrapped error |

---

## Integration Notes

### Claude Tools Integration

The LinkedIn service is used by Claude tools in the claude package:

| Claude Tool | Service Function |
|-------------|------------------|
| `linkedin_post` | `CreatePost()` + `LogPost()` |
| `linkedin_read_comments` | `ReadComments()` |
| `linkedin_post_comment` | `CreateComment()` + `LogComment()` |
| `linkedin_reply_comment` | `CreateComment(parentURN=...)` + `LogComment()` |

### Vault Integration

Activity logs are stored in the person's vault directory:
```
~/notes/{person}/linkedin/posts.csv
```

This allows logs to be version-controlled with the user's other notes.

---

## Testing

### Unit Tests

```go
func TestService_CreatePost(t *testing.T) {
    // Create mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/v2/userinfo" {
            json.NewEncoder(w).Encode(map[string]string{"sub": "test123"})
            return
        }
        if r.URL.Path == "/v2/ugcPosts" {
            json.NewEncoder(w).Encode(map[string]string{"id": "urn:li:share:999"})
            return
        }
    }))
    defer server.Close()

    cfg := &Config{AccessToken: "test-token"}
    svc := NewServiceWithBaseURL(cfg, t.TempDir(), server.URL)

    resp, err := svc.CreatePost("Test post", "sebastian")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if resp.ID != "urn:li:share:999" {
        t.Errorf("got ID %q, want %q", resp.ID, "urn:li:share:999")
    }
}

func TestService_LogPost(t *testing.T) {
    root := t.TempDir()
    svc := NewService(&Config{}, root)

    err := svc.LogPost("sebastian", "Test post", &PostResponse{ID: "urn:li:share:123"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Verify CSV was created
    logPath := filepath.Join(root, "sebastian", "linkedin", "posts.csv")
    content, err := os.ReadFile(logPath)
    if err != nil {
        t.Fatalf("failed to read log: %v", err)
    }

    if !strings.Contains(string(content), "post,urn:li:share:123") {
        t.Error("log entry not found")
    }
}
```

### Integration Tests

```go
func TestOAuthCallback_Integration(t *testing.T) {
    if os.Getenv("LINKEDIN_TEST_CODE") == "" {
        t.Skip("LINKEDIN_TEST_CODE not set")
    }

    cfg, _ := LoadConfig()
    svc := NewService(cfg, t.TempDir())

    resp, err := svc.ExchangeCodeForToken(os.Getenv("LINKEDIN_TEST_CODE"))
    if err != nil {
        t.Fatalf("token exchange failed: %v", err)
    }

    if resp.AccessToken == "" {
        t.Error("expected access token")
    }
}
```

---

## Related Specifications

- [01-rest-api-contract.md](./01-rest-api-contract.md) - OAuth callback endpoint
- [02-vault-storage-git-sync.md](./02-vault-storage-git-sync.md) - VAULT_ROOT and storage paths
- [04-claude-service.md](./04-claude-service.md) - Claude tools integration
