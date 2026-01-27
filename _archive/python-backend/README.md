# Archived Python Backend

> Archived: 2026-01-27
> Replaced by: Go server in `server/`

This directory contains the legacy Python/FastAPI backend that was superseded by the Go server implementation.

## Migration Notes

### Why Archived

The Python backend was replaced with a Go implementation for:
- Better performance and lower memory footprint
- Simpler deployment (single binary)
- Type safety and compile-time checks
- Native concurrency support for streaming

### Equivalent Components

| Python (archived) | Go (active) |
|-------------------|-------------|
| `web_app/main.py` | `server/cmd/server/main.go` + `server/internal/api/` |
| `web_app/services/vault_store.py` | `server/internal/vault/store.go` |
| `web_app/services/git_sync.py` | `server/internal/vault/git.go` |
| `web_app/services/claude_service.py` | `server/internal/claude/` |
| `web_app/services/linkedin_service.py` | `server/internal/linkedin/` |

### Custom Integration Migration

If you had custom integrations with the Python backend:

1. **API Endpoints**: All REST endpoints remain the same (see `specs/01-rest-api-contract.md`)
2. **Port Change**: Default port changed from 8000 to 8080
3. **Authentication**: Bearer token auth unchanged
4. **Claude Tools**: Same tool set available via streaming API

### Restoring (if needed)

To temporarily restore the Python backend:

```bash
# Copy files back
cp -r _archive/python-backend/web_app server/
cp _archive/python-backend/pyproject.toml .

# Install dependencies
uv sync

# Run Python server
uv run uvicorn server.web_app.main:app --host 0.0.0.0 --port 8000
```
