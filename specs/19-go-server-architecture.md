# Go Server Architecture

> Status: Draft
> Version: 1.1
> Last Updated: 2026-01-28

## Overview

The Go server provides all backend functionality for the Notes Editor application. It exposes a REST API consumed by both the Android and React web clients.

## Project Structure

```
server/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, config loading, server startup
│
├── internal/
│   ├── api/
│   │   ├── router.go            # Route definitions
│   │   ├── middleware.go        # Auth, logging, CORS
│   │   ├── daily.go             # Daily note handlers
│   │   ├── todos.go             # Todo handlers
│   │   ├── files.go             # File operations handlers
│   │   ├── sleep.go             # Sleep tracking handlers
│   │   ├── claude.go            # Claude chat handlers
│   │   ├── linkedin.go          # LinkedIn OAuth handler
│   │   └── settings.go          # Settings handlers
│   │
│   ├── vault/
│   │   ├── store.go             # File read/write operations
│   │   ├── store_test.go
│   │   ├── git.go               # Git pull/commit/push
│   │   ├── git_test.go
│   │   ├── daily.go             # Daily note logic (create, inherit todos)
│   │   ├── daily_test.go
│   │   └── paths.go             # Path resolution and validation
│   │
│   ├── claude/
│   │   ├── service.go           # Claude API client
│   │   ├── service_test.go
│   │   ├── session.go           # In-memory session management
│   │   ├── session_test.go
│   │   ├── tools.go             # Tool definitions for Claude
│   │   └── stream.go            # Streaming response handling
│   │
│   ├── linkedin/
│   │   ├── oauth.go             # OAuth token exchange
│   │   ├── client.go            # LinkedIn API client
│   │   ├── client_test.go
│   │   └── logging.go           # Activity CSV logging
│   │
│   ├── auth/
│   │   ├── auth.go              # Token validation
│   │   ├── auth_test.go
│   │   └── person.go            # Person context management
│   │
│   └── config/
│       └── config.go            # Environment/config loading
│
├── go.mod
├── go.sum
└── Makefile                     # Build, test, lint commands
```

## Dependencies

```go
// go.mod
module notes-editor

go 1.22

require (
    github.com/go-chi/chi/v5 v5.x.x    // HTTP router
    github.com/go-chi/cors v1.x.x      // CORS middleware
    github.com/joho/godotenv v1.x.x    // .env loading
)
```

Minimal dependencies. No ORM (filesystem storage), no heavy frameworks.

## Package Responsibilities

### `internal/api`

HTTP handlers that:
1. Parse and validate requests
2. Call business logic in other packages
3. Format JSON responses

Handlers are thin - they delegate to vault/claude/linkedin packages.

```go
// Example handler structure
func (h *Handler) GetDaily(w http.ResponseWriter, r *http.Request) {
    person := auth.PersonFromContext(r.Context())

    note, err := h.vault.GetOrCreateDaily(person, time.Now())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(note)
}
```

### `internal/vault`

File operations and git sync:
- `Store`: Read/write/delete files with path validation
- `Git`: Pull, commit, push operations
- `Daily`: Daily note creation with todo inheritance

All file paths are validated to prevent directory traversal.

```go
type Store struct {
    rootPath string  // ~/notes
}

func (s *Store) ReadFile(person, path string) ([]byte, error)
func (s *Store) WriteFile(person, path string, content []byte) error
func (s *Store) DeleteFile(person, path string) error
func (s *Store) ListDir(person, path string) ([]FileEntry, error)
```

### `internal/claude`

Claude AI integration:
- `Service`: Manages Claude API calls with tool use
- `Session`: In-memory conversation history (map[string]*Session)
- `Tools`: File read/write/search tools for Claude

```go
type Service struct {
    apiKey   string
    sessions *SessionStore
    vault    *vault.Store
}

func (s *Service) Chat(ctx context.Context, sessionID, message string) (*Response, error)
func (s *Service) ChatStream(ctx context.Context, sessionID, message string) (<-chan Event, error)
```

### `internal/linkedin`

LinkedIn API integration:
- OAuth token exchange
- Post creation
- Comment operations
- Activity logging to CSV

### `internal/auth`

Authentication and person context:
- Bearer token validation (constant-time comparison)
- Person context extraction from headers
- Context helpers for handlers

```go
func ValidateToken(token, expected string) bool
func PersonFromContext(ctx context.Context) string
func WithPerson(ctx context.Context, person string) context.Context
```

## Testing Strategy

### Unit Tests

Each package has `*_test.go` files with table-driven tests.

```go
func TestStore_ReadFile(t *testing.T) {
    tests := []struct {
        name    string
        person  string
        path    string
        setup   func(root string)  // Create test files
        want    string
        wantErr bool
    }{
        {
            name:   "reads existing file",
            person: "sebastian",
            path:   "daily/2026-01-27.md",
            setup: func(root string) {
                // Create test file
            },
            want: "# daily 2026-01-27\n",
        },
        {
            name:    "rejects path traversal",
            person:  "sebastian",
            path:    "../../../etc/passwd",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

Test full request/response cycles:

```go
func TestAPI_GetDaily(t *testing.T) {
    // Setup test server with temp directory
    srv := setupTestServer(t)

    req := httptest.NewRequest("GET", "/api/daily", nil)
    req.Header.Set("Authorization", "Bearer test-token")
    req.Header.Set("X-Notes-Person", "sebastian")

    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    // Verify response body
}
```

### Test Fixtures

Use temporary directories for file system tests:

```go
func setupTestVault(t *testing.T) *vault.Store {
    t.Helper()
    dir := t.TempDir()

    // Create person directories
    os.MkdirAll(filepath.Join(dir, "sebastian", "daily"), 0755)
    os.MkdirAll(filepath.Join(dir, "petra", "daily"), 0755)

    return vault.NewStore(dir)
}
```

### Mocking External Services

Claude and LinkedIn APIs are mocked in tests:

```go
type MockClaudeAPI struct {
    Response string
    Error    error
}

func (m *MockClaudeAPI) CreateMessage(ctx context.Context, req Request) (*Response, error) {
    if m.Error != nil {
        return nil, m.Error
    }
    return &Response{Content: m.Response}, nil
}
```

## Configuration

Environment variables loaded from `.env`:

```
NOTES_TOKEN=secret-bearer-token
NOTES_ROOT=/home/user/notes
STATIC_DIR=./static
ANTHROPIC_API_KEY=sk-ant-...
LINKEDIN_CLIENT_ID=...
LINKEDIN_CLIENT_SECRET=...
LINKEDIN_ACCESS_TOKEN=...
```

## Static File Serving

The Go server serves the React web UI as static files from the `STATIC_DIR` directory (defaults to `./static`).

**Route structure:**
- `/api/*` - API endpoints (protected by auth middleware)
- `/*` - Static files (no auth, login handled client-side)

**SPA routing:** For any request that doesn't match a static file, the server returns `index.html` to support client-side routing.

**Build process:**
```bash
# Build web UI and copy to static directory
cd clients/web && npm run build
cp -r dist ../server/static

# Or use the root Makefile
make build  # Builds both web UI and server
```

**Deployment:** The systemd service runs the Go binary which serves both the API and web UI on port 8080.

## HTTP Server Setup

```go
func main() {
    cfg := config.Load()

    store := vault.NewStore(cfg.NotesRoot)
    claudeSvc := claude.NewService(cfg.AnthropicKey, store)
    linkedinSvc := linkedin.NewService(cfg.LinkedInConfig)

    handler := api.NewHandler(store, claudeSvc, linkedinSvc)
    router := api.NewRouter(handler, cfg.NotesToken)

    log.Printf("Starting server on :8080")
    http.ListenAndServe(":8080", router)
}
```

## Middleware Stack

```go
func NewRouter(srv *Server) http.Handler {
    r := chi.NewRouter()

    // Global middleware (no auth)
    r.Use(RecovererMiddleware)
    r.Use(LoggingMiddleware)
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{"*"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Notes-Person"},
        AllowCredentials: true,
        MaxAge:           300,
    }))

    // API routes with auth
    r.Route("/api", func(r chi.Router) {
        r.Use(AuthMiddleware(srv.config.NotesToken))
        r.Use(PersonMiddleware)

        r.Get("/daily", srv.handleGetDaily)
        r.Post("/save", srv.handleSaveDaily)
        // ... other API routes
    })

    // Static file serving for web UI (no auth)
    r.Get("/*", staticFileHandler(srv.config.StaticDir))

    return r
}
```

## Makefile

```makefile
.PHONY: build test lint run

build:
	go build -o bin/server ./cmd/server

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

run:
	go run ./cmd/server
```

## Error Handling

Errors are returned as JSON with appropriate HTTP status codes:

```go
type APIError struct {
    Detail string `json:"detail"`
}

func writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(APIError{Detail: message})
}
```

## Streaming Responses

Claude chat streaming uses NDJSON:

```go
func (h *Handler) ChatStream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/x-ndjson")
    w.Header().Set("Cache-Control", "no-cache")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", 500)
        return
    }

    events, err := h.claude.ChatStream(r.Context(), sessionID, message)
    if err != nil {
        writeStreamError(w, err)
        return
    }

    for event := range events {
        json.NewEncoder(w).Encode(event)
        flusher.Flush()
    }
}
```

## Related Specifications

- [00-project-overview](./00-project-overview.md) - Overall architecture
- [01-rest-api-contract](./01-rest-api-contract.md) - API endpoints
- [02-vault-storage-git-sync](./02-vault-storage-git-sync.md) - Storage layer
- [04-claude-service](./04-claude-service.md) - Claude integration
- [05-linkedin-service](./05-linkedin-service.md) - LinkedIn integration
