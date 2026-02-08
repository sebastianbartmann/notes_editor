# Notes Editor

A personal notes app with a Go backend, React web client, and native Android client. Data is stored in a git-synced vault with per-person subfolders.

## Features

- **Daily Notes** - Date-based markdown files with tasks, pinned entries, and quick append
- **File Browser** - Tree view with create/edit/delete, scoped per person
- **Sleep Tracking** - Per-child sleep time logging with status tracking
- **White Noise** - Procedural audio (web) / MP3 playback with media controls (Android)
- **Claude Chat** - Streaming AI chat with tool use (WIP)
- **Git Sync** - Auto-commits and pushes on every change, pulls on load

## Architecture

```
server/          Go REST API (serves web client + API for Android)
clients/web/     React/TypeScript SPA (Vite)
app/android/     Kotlin/Compose native Android client
```

## Requirements

- Go 1.22+
- Node.js 18+ (for web client)
- Git repository for notes storage
- JDK 17 + Android SDK (for Android builds only)

## Configuration

Create `server/.env`:
```
NOTES_TOKEN=your-auth-token
NOTES_ROOT=/path/to/notes/repo
ANTHROPIC_API_KEY=optional-for-claude
SERVER_ADDR=:80
```

Required: `NOTES_TOKEN`, `NOTES_ROOT`. Optional: `ANTHROPIC_API_KEY`, LinkedIn OAuth vars.

## Development

```bash
# Go server (port 8080)
make build-server && make server

# React dev server (port 5173, proxies API to 8080)
make install-client
make client

# Run all tests
make test
```

## Production

```bash
# Build everything (web client + Go server)
make build

# Install systemd service
make install-systemd
make status-systemd
```

## Android

```bash
# Build debug APK
make build-android

# Build and install via adb
make deploy-android

# Run Maestro UI tests
make android-test
```

The Android app uses a bearer token configured in `AppConfig.kt`.

## Testing

```bash
make test              # All tests (server + web client)
make test-server       # Go tests only
make test-client       # React tests (TypeScript check + Vitest)
make test-coverage     # Go tests with HTML coverage report
make android-test      # Maestro UI tests (requires emulator)
```

## License

MIT
