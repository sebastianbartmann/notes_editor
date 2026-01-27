# Notes Editor Overview

Notes Editor is a personal/family "second brain" app with a Go backend REST API, React web client, and native Android client. Data lives in a local vault with per-person subfolders (e.g., `sebastian/`, `petra/`), and each client selects its person root and theme locally while the server remains user-agnostic.

Key features: daily notes with tasks and pinned entries, a file tree editor scoped to the selected person, and shared tools such as sleep tracking, noise playback, and a Claude tool. The Android app mirrors the web layout and adds native conveniences (media-style noise controls, persistent settings).

# Repository Guidelines

## Project Structure & Module Organization

- `server/` contains the Go backend:
  - `cmd/server/main.go` - entry point
  - `internal/api/` - HTTP handlers and middleware
  - `internal/vault/` - file operations and git sync
  - `internal/claude/` - Claude AI service with streaming
  - `internal/linkedin/` - LinkedIn OAuth and API
  - `internal/auth/` - token validation and person context
  - `internal/config/` - environment configuration
- `_archive/python-backend/` contains the archived Python FastAPI backend (superseded by Go)
- `clients/web/` contains the React web client:
  - `src/api/` - API client modules with TypeScript types
  - `src/components/` - Reusable UI components (NoteView, Editor, FileTree, Layout)
  - `src/context/` - React context providers (Auth, Person, Theme)
  - `src/hooks/` - Custom hooks (useAuth, usePerson, useTheme)
  - `src/pages/` - Page components (Daily, Files, Sleep, Claude, Noise, Settings)
- `app/android/` contains the Android client; build tooling lives in `app/gradle-8.7/` and `app/android_sdk/`
- Root `Makefile` and `notes-editor.service` define local workflows and systemd deployment for the Go server

## Build, Test, and Development Commands

### Go Backend (server/)
- `cd server && go mod tidy` downloads dependencies
- `cd server && make build` compiles to `bin/server`
- `cd server && make test` runs all tests
- `cd server && make test-coverage` generates coverage report
- `cd server && make run` starts the server on port 8080
- Requires Go 1.22+

### React Web Client (clients/web/)
- `cd clients/web && npm install` installs dependencies
- `cd clients/web && npm run dev` starts Vite dev server with API proxy to localhost:8080
- `cd clients/web && npm run build` creates production build in `dist/`
- `cd clients/web && npm run preview` previews production build
- `cd clients/web && npm test` runs Vitest tests (61 tests)
- `cd clients/web && npm run test:watch` runs tests in watch mode
- TypeScript strict mode enabled

### Root Makefile Commands
- `make run` starts the Go dev server on port 8080
- `make build` builds the Go binary to `server/bin/server`
- `make test` runs Go backend tests
- `make install-systemd` installs/refreshes the systemd unit
- `make status-systemd` checks the systemd service status

### Android
- `make build-android` builds the Android debug APK
- `make deploy-android` builds and installs the debug APK via adb

## Coding Style & Naming Conventions

- Go: standard `gofmt`, exported functions `PascalCase`, internal `camelCase`, packages lowercase
- Kotlin/Android: follow standard Android conventions; keep resource names lowercase with underscores (e.g., `noise_player.xml`)
- TypeScript/React: strict mode, functional components, CSS modules for styling, PascalCase for components
- Keep modules small and prefer explicit imports over wildcard imports

## Testing Guidelines

### Go Backend
- Tests are in `*_test.go` files alongside source code
- Run `cd server && make test` for all tests
- Key test files: `auth/auth_test.go`, `vault/paths_test.go`, `vault/store_test.go`, `vault/daily_test.go`, `claude/session_test.go`

### React Web Client
- Tests are in `*.test.ts(x)` files alongside source code
- Run `cd clients/web && npm test` for all tests (96 tests)
- Key test files:
  - `NoteView.test.tsx` (line parsing, task toggle)
  - `AuthContext.test.tsx`, `ThemeContext.test.tsx`, `PersonContext.test.tsx`
  - `claude.test.ts` (NDJSON streaming, event parsing, chunked data)
  - `client.test.ts` (API client, auth headers, ApiError)
- Uses Vitest with jsdom and @testing-library/react

### Manual Verification
- Run the server and validate the web UI plus Android flows you touched

## Commit & Pull Request Guidelines

- Commit messages in history are short, imperative, and sentence case (e.g., "Add Claude streaming updates").
- PRs should include a clear summary, steps to verify, and screenshots for UI changes (web or Android).
- Link any related issues and call out config changes (e.g., `notes-editor.service` or env vars).

## Security & Configuration Notes

- The app expects `NOTES_TOKEN` to be set for auth; do not commit secrets.
- The Android client uses a bearer token in `app/android/app/src/main/java/com/bartmann/noteseditor/AppConfig.kt`; update it locally and keep it out of commits when possible.
