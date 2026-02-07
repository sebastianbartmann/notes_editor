# Notes Editor Server

Go backend server for the Notes Editor application.

## Prerequisites

- Go 1.22+ (installed at `~/go-sdk/go/` on dev machine)
- Git (for vault synchronization)

## Setup

1. **Configure environment variables**

   Copy the example file and edit with your values:

   ```bash
   cp .env.example .env
   ```

   Required variables:
   - `NOTES_TOKEN` - Authentication token for API access
   - `NOTES_ROOT` - Path to the vault directory (e.g., `/home/user/notes`)
   - `ANTHROPIC_API_KEY` - API key for Claude AI features

   Optional variables:
   - `STATIC_DIR` - Path to static files (defaults to `./static`)
   - `SERVER_ADDR` - HTTP listen address (defaults to `:80`)
   - `LINKEDIN_*` - LinkedIn OAuth credentials (for LinkedIn integration)

2. **Initialize the vault**

   Create the vault directory with person subdirectories:

   ```bash
   mkdir -p /path/to/notes/{sebastian,petra}
   cd /path/to/notes && git init
   ```

3. **Install dependencies**

   ```bash
   cd server
   go mod tidy
   ```

## Development

**Run the server:**

```bash
cd server && go run ./cmd/server
```

Or using the root Makefile:

```bash
make server
```

The server starts on port 80 by default. For local development, set `SERVER_ADDR=:8080`.

**Run tests:**

```bash
cd server && go test ./...
```

Or:

```bash
make test-server
```

**Build binary:**

```bash
cd server && go build -o bin/server ./cmd/server
```

Or:

```bash
make build-server
```

## API Reference

See [specs/01-rest-api-contract.md](../specs/01-rest-api-contract.md) for complete API documentation.

### Quick Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/daily` | GET | Fetch today's daily note |
| `/api/save` | POST | Save note content |
| `/api/append` | POST | Append timestamped entry |
| `/api/clear-pinned` | POST | Remove pinned markers |
| `/api/todos/add` | POST | Add todo to category |
| `/api/todos/toggle` | POST | Toggle todo checkbox |
| `/api/sleep-times` | GET | Get sleep entries |
| `/api/sleep-times/append` | POST | Add sleep entry |
| `/api/sleep-times/delete` | POST | Delete sleep entry |
| `/api/files/list` | GET | List directory contents |
| `/api/files/read` | GET | Read file content |
| `/api/files/create` | POST | Create new file |
| `/api/files/save` | POST | Save file content |
| `/api/files/delete` | POST | Delete file |
| `/api/files/unpin` | POST | Unpin entry by line |
| `/api/claude/chat` | POST | Chat with Claude |
| `/api/claude/chat-stream` | POST | Streaming chat (NDJSON) |
| `/api/claude/clear` | POST | Clear chat session |
| `/api/claude/history` | GET | Get chat history |
| `/api/settings/env` | GET/POST | Read/write .env file |
| `/api/linkedin/oauth/callback` | GET | LinkedIn OAuth callback |

### Authentication

All endpoints (except LinkedIn OAuth callback) require:

```
Authorization: Bearer <NOTES_TOKEN>
X-Notes-Person: sebastian|petra
```

## Production Deployment

1. **Build the full application:**

   ```bash
   make build
   ```

   This builds the Go binary and copies the React web UI to `server/static/`.

2. **Install systemd service:**

   ```bash
   make install-systemd
   ```

3. **Check service status:**

   ```bash
   make status-systemd
   ```

## Architecture

```
server/
├── cmd/server/main.go    # Entry point
└── internal/
    ├── api/              # HTTP handlers and middleware
    ├── auth/             # Token validation
    ├── claude/           # Claude AI service
    ├── config/           # Environment configuration
    ├── linkedin/         # LinkedIn OAuth and API
    └── vault/            # File operations and git sync
```

See [specs/19-go-server-architecture.md](../specs/19-go-server-architecture.md) for detailed architecture documentation.
