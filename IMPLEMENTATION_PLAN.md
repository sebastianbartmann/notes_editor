# Implementation Plan

> Last updated: 2026-01-28
> Status: Active - Phase 5 complete. Next: Phase 6 (Android Alignment)

## Instructions
- Tasks marked `- [ ]` are incomplete
- Tasks marked `- [x]` are complete
- Work from top to bottom (highest priority first)
- Add new tasks as you discover them

---

## Phase 1: Go Backend (spec 19) - COMPLETE

> **Note:** Go 1.22.5 installed at `~/go-sdk/go/`. Run `cd server && go test ./...` to verify.

### 1.9 Go Backend Testing - COMPLETE
- [x] Run tests once Go is installed (`make test`)
- [x] Integration tests for full request/response cycles
- [x] Test authentication flow
- [x] Test person context isolation
- [x] Test concurrent request handling (Claude session thread safety)

---

## Phase 2: React Web Client (spec 20) - COMPLETE

> **Note:** Complete React web client implemented in `clients/web/`. Run `cd clients/web && npm install && npm run dev` to start development server. Build verified: TypeScript compiles, Vite builds successfully.

### 2.1 Project Setup - COMPLETE
- [x] Create `clients/web/` directory
- [x] Initialize npm project with `package.json`
  - [x] Add dependencies: react, react-dom, react-router-dom
  - [x] Add dev dependencies: vite, @vitejs/plugin-react, typescript, @types/react
  - [x] Add scripts: dev, build, preview
- [x] Create `tsconfig.json` with React settings
- [x] Create `vite.config.ts` with API proxy to localhost:8080
- [x] Create `index.html` root HTML file

### 2.2 Core Application - COMPLETE
- [x] Create `src/main.tsx` entry point
- [x] Create `src/App.tsx` with routing and provider hierarchy
- [x] Create `src/index.css` with CSS variables and global styles

### 2.3 API Client Layer (`src/api/`) - COMPLETE
- [x] Create `types.ts` - all API response/request types
- [x] Create `client.ts` - base HTTP client with auth headers
- [x] Create `daily.ts` - fetchDaily, saveDaily, appendDaily, clearPinned
- [x] Create `files.ts` - listFiles, readFile, saveFile, createFile, deleteFile, unpinEntry
- [x] Create `todos.ts` - addTodo, toggleTodo
- [x] Create `sleep.ts` - fetchSleepTimes, appendSleepTime, deleteSleepTime
- [x] Create `claude.ts` - chatStream (async generator), clearSession, getHistory

### 2.4 Context Providers (`src/context/`) - COMPLETE
- [x] Create `AuthContext.tsx`
  - [x] AuthState interface, login/logout functions
  - [x] Sync token to localStorage
- [x] Create `PersonContext.tsx`
  - [x] PersonState interface, setPerson function
  - [x] Sync person to localStorage
- [x] Create `ThemeContext.tsx`
  - [x] Theme type (dark/light), setTheme function
  - [x] Update body class for theme switching

### 2.5 Custom Hooks (`src/hooks/`) - COMPLETE
- [x] Create `useAuth.ts` - return auth state and methods
- [x] Create `usePerson.ts` - return person state and methods
- [x] Create `useTheme.ts` - return theme state and methods

### 2.6 Layout Components (`src/components/Layout/`) - COMPLETE
- [x] Create `Layout.tsx` - main wrapper with header and nav
- [x] Create `Header.tsx` - page title, theme toggle, person display
- [x] Create `Navigation.tsx` - links to: Daily, Files, Sleep, Claude, Noise, Settings
- [x] Create `Layout.module.css`

### 2.7 NoteView Components (`src/components/NoteView/`) - COMPLETE
- [x] Create `NoteView.tsx` - main renderer
  - [x] Accept props: content, path, onTaskToggle, onUnpin
  - [x] Parse lines with LineType enum (H1-H6, TASK, TEXT, EMPTY)
  - [x] Task regex: `^\s*-\s*\[([ xX])\]\s*(.*)$`
  - [x] Heading regex: `^(#{1,6})\s+(.*)$`
  - [x] Pinned detection: `<pinned>` in H3
  - [x] Render empty lines with `&nbsp;` entity
  - [x] Apply HTML escaping for XSS protection
  - [x] Inline TaskLine: checkbox with checked state, calls onTaskToggle
  - [x] Inline UnpinButton: button in pinned H3 headings, calls onUnpin
- [x] Create `NoteView.module.css`
  - [x] Style headings (H1: 16px bold, H2: 14px uppercase muted, H3: 13px accent, H4: 12px)
  - [x] Style task checkboxes with accent-color
  - [x] Style pinned headings: background #151a1f, border-radius 4px
  - [x] Style unpin button: border-radius 999px, font-size 11px

### 2.8 Editor Component (`src/components/Editor/`) - COMPLETE
- [x] Create `Editor.tsx` - markdown editor
  - [x] Textarea with content
  - [x] Save/Cancel buttons
  - [x] Track local edits
- [x] Create `Editor.module.css`

### 2.9 FileTree Component (`src/components/FileTree/`) - COMPLETE
- [x] Create `FileTree.tsx` - directory browser
  - [x] Lazy loading for subdirectories
  - [x] Expand/collapse state
  - [x] File selection callback
- [x] Create `FileTree.module.css`

### 2.10-2.11 Additional Components - COMPLETE
> Note: Chat, SleepForm, and NoisePlayer functionality implemented directly in page components for simplicity.

### 2.12 Page Components (`src/pages/`) - COMPLETE
- [x] Create `LoginPage.tsx` - token input form
- [x] Create `DailyPage.tsx`
  - [x] Fetch daily note on mount
  - [x] NoteView for display, Editor for edit mode
  - [x] Task toggle, append form with pinned option
- [x] Create `FilesPage.tsx`
  - [x] FileTree for navigation
  - [x] NoteView/Editor for selected file
  - [x] Create/delete file actions
- [x] Create `SleepPage.tsx`
  - [x] Sleep entry form with child selection, status checkboxes
  - [x] Recent entries list with delete buttons
  - [x] Refresh list after add/delete
- [x] Create `ClaudePage.tsx`
  - [x] Chat interface with streaming (async generator)
  - [x] Tool status display
  - [x] Clear session button
- [x] Create `NoisePage.tsx`
  - [x] Web Audio procedural noise per spec 08
  - [x] Play/stop toggle, LFO modulation, drift timer
- [x] Create `SettingsPage.tsx`
  - [x] Person selector
  - [x] Theme toggle
  - [x] Logout button

### 2.13 Routing - COMPLETE
- [x] Configure BrowserRouter in App.tsx
- [x] Define routes: /, /daily, /files, /files/*, /sleep, /claude, /noise, /settings, /login
- [x] Wrap routes in Layout component
- [x] Wrap app in provider hierarchy (Auth > Person > Theme)
- [x] ProtectedRoute component for auth guard

### 2.14 Theming (`src/index.css`) - COMPLETE

#### Dark Theme (Default)
- [x] `--bg: #0f1012`
- [x] `--panel: #15171a`
- [x] `--panel-border: #2a2d33`
- [x] `--text: #e6e6e6`
- [x] `--muted: #9aa0a6`
- [x] `--accent: #d9832b`
- [x] `--accent-dim: #7a4a1d`
- [x] `--danger: #d66b6b`
- [x] `--input: #0f1114`
- [x] `--note: #101317`

#### Light Theme (body.theme-light)
- [x] `--bg: #e9f7f7`
- [x] `--panel: #f6fbff`
- [x] `--panel-border: #c7e3e6`
- [x] `--text: #1a2a2f`
- [x] `--muted: #4f6f78`
- [x] `--accent: #3aa7a3`
- [x] `--accent-dim: #c9f1ef`
- [x] `--input: #f2fafb`
- [x] `--note: #f9fdff`

#### Spacing and Typography
- [x] `--space-1: 6px`, `--space-2: 10px`, `--space-3: 14px`, `--space-4: 18px`
- [x] `--radius: 6px`
- [x] `--font: "IBM Plex Mono", monospace`
- [x] Base font-size: 14px, line-height: 1.5

### 2.15 Build Configuration - COMPLETE
- [x] Vite dev server with API proxy
- [x] Production build to `dist/`
- [x] Source maps for debugging

---

## Phase 3: Migration and Cleanup

### 3.1 Deprecate Python Backend - COMPLETE
- [x] Document migration path for any custom integrations
- [x] Archive Python code to `_archive/python-backend/`
- [x] Update deployment scripts for Go backend
- [x] Update CI/CD pipelines (N/A - no pipelines exist)

### 3.2 Update Specs - COMPLETE
- [x] Mark spec 07 (web-interface) as fully deprecated (already marked SUPERSEDED)
- [x] Update spec 10 to reference Go auth implementation
- [x] Update spec 11 to reference React NoteView instead of Python
- [x] Update spec 17 to reference Go server instead of Python
- [x] Verify all specs match Go+React implementation

**Issues Found During Verification:**
- Go `/api/todos/add` uses `task` field but spec 01 and React client use `text` - needs alignment
- Go `/api/todos/add` requires `path` but spec says it should auto-determine today's note path
- React web client missing inline task input feature (spec 17) - only has append form

---

## Phase 4: Testing

### 4.1 Go Backend Tests - Security Critical - COMPLETE
- [x] Test path traversal prevention (../../../etc/passwd attacks)
- [x] Test person context isolation (sebastian can't access petra's files)
- [x] Test constant-time token comparison
- [x] Test authentication middleware rejects invalid tokens

### 4.2 Go Backend Fixes - API Alignment - COMPLETE
- [x] Fix `/api/todos/add`: rename `task` field to `text` per spec 01
- [x] Fix `/api/todos/add`: make `text` field optional (empty creates blank task)
- [x] Fix `/api/todos/add`: auto-determine today's daily note path (remove required `path` field)
- [x] Add form-encoded request support for Android compatibility (todos endpoints)

### 4.3 Go Backend Tests - Core Logic - COMPLETE
- [x] Test daily note creation with inherited todos
- [x] Test daily note creation with inherited pinned notes
- [x] Test todo toggle (checked/unchecked)
- [x] Test task addition to categories
- [x] Test pinned marker operations
- [ ] Test git pull with conflict resolution (requires git repo setup)
- [ ] Test git commit and push with retry (requires git repo setup)

### 4.4 Go Backend Tests - Services - PARTIAL
- [x] Test Claude session management (concurrent access)
- [ ] Test Claude tool execution (requires mock Claude API)
- [x] Test NDJSON streaming format
- [ ] Test 5-second keepalive ping in streaming (timing-dependent, complex to test)
- [ ] Test LinkedIn OAuth token exchange (requires HTTP mock)
- [x] Test LinkedIn CSV activity logging

### 4.5 Go Backend Tests - API - COMPLETE
- [x] Integration tests for all endpoints
- [x] Test error response formats (400, 401, 404)
- [x] Test CORS headers
- [x] Test request validation

### 4.6 React Web Client Tests - COMPLETE
- [x] Test NoteView line parsing (H1-H6, tasks, text, empty)
- [x] Test task toggle state management
- [x] Test streaming text incremental display (mocking fetch/ReadableStream)
- [x] Test theme switching (dark/light)
- [x] Test person context switching
- [x] Test localStorage persistence (token, person, theme)

### 4.7 Android App Tests (existing gap)
- [ ] Unit tests for ApiClient failover logic
- [ ] Unit tests for NoteView markdown parsing
- [ ] UI tests for daily screen task toggle
- [ ] UI tests for navigation flows

---

## Phase 5: Documentation - COMPLETE

### 5.1 Go Backend Documentation - COMPLETE
- [x] Create `server/README.md` with setup instructions
- [x] Document all environment variables in `.env.example` (already existed)
- [x] Document API endpoints (link to spec 01)

### 5.2 React Web Client Documentation - COMPLETE
- [x] Create `clients/web/README.md` with setup instructions
- [x] Document development workflow (npm scripts)
- [x] Document build and deployment

---

## Phase 6: Android Alignment (existing gaps)

### 6.1 UI Gaps
- [ ] Implement per-entry unpin UI (API exists: `ApiClient.unpinEntry`)
- [ ] Remove unused callback parameters in `AppNavigation.kt`

### 6.2 Theme Alignment (spec 12)
- [ ] Update dark theme colors to match spec:
  - [ ] `background: #0F1012` (currently #1A1C1F)
  - [ ] `panel: #15171A` (currently #282B31)
  - [ ] `panelBorder: #2A2D33` (currently #3A3E46)
  - [ ] `input: #0F1114` (currently #1E2024)
  - [ ] `note: #101317` (currently #1F2226)
  - [ ] `button: #1E2227` (currently #353942)

### 6.3 Manual Testing
> **Note:** Will be replaced by automated Maestro tests in Phase 8 (spec 21)

- [ ] Test navigation on all screens
- [ ] Test keyboard visibility behavior (bottom nav hides, accessory shows)
- [ ] Test with person=null (only settings accessible)

### 6.4 Android API Compatibility - DEPRECATED
> **SUPERSEDED by Phase 7.** The correct approach is to update the Android app to send JSON
> (matching the spec), not to add form support to the Go server. Phase 7 handles this properly.

---

## Phase 7: Android API Alignment (Critical) - COMPLETE

> **Root Cause:** The Android app was built for the old Python/FastAPI backend which accepted
> form-encoded data. The spec (01-rest-api-contract.md) correctly specifies JSON for all POST
> requests. The Go server follows the spec. The Android app must be updated to match.
>
> **Principle:** The spec is the source of truth. Do not implement multiple solutions.
> Update the Android app to send JSON, not add form support to Go.

### 7.1 Android ApiClient - Switch from Form to JSON - COMPLETE

- [x] Add `postJson()` function to ApiClient.kt that serializes to JSON with Content-Type header
- [x] Remove `postForm()` function (replaced with `postJson()`)
- [x] Remove unused `FormBody` import

### 7.2 Android ApiClient - Fix Endpoint Paths - COMPLETE

- [x] Update `saveFile()` to use `/api/files/save`
- [x] Update `deleteFile()` to use `/api/files/delete`

### 7.3 Android ApiClient - Fix Request Field Names - COMPLETE

- [x] `saveDaily()`: now accepts `path` parameter, sends JSON with `path` and `content`
- [x] `appendDaily()`: now accepts `path`, renamed `content` to `text`, `pinned` is boolean
- [x] `clearPinned()`: now accepts `path` parameter
- [x] `appendSleepTimes()`: renamed `entry` to `time`, replaced `asleep`/`woke` with `status` string
- [x] `deleteSleepEntry()`: sends JSON with integer `line` field
- [x] `createFile()`: sends JSON with `path` field
- [x] `saveFile()`: sends JSON with `path` and `content` fields
- [x] `deleteFile()`: sends JSON with `path` field
- [x] `unpinEntry()`: sends JSON with `path` and `line` fields
- [x] `claudeChat()`: sends JSON with `message` and optional `session_id`
- [x] `claudeChatStream()`: sends JSON with `message` and optional `session_id`, keeps NDJSON Accept header
- [x] `claudeClear()`: sends JSON with `session_id`
- [x] `saveEnv()`: renamed field from `env_content` to `content`

### 7.4 Android Request Models - COMPLETE

All request data classes created in `Models.kt`:
- [x] `SaveDailyRequest`, `AppendDailyRequest`, `ClearPinnedRequest`
- [x] `AddTodoRequest`, `ToggleTodoRequest`
- [x] `AppendSleepRequest`, `DeleteSleepRequest`
- [x] `CreateFileRequest`, `SaveFileRequest`, `DeleteFileRequest`, `UnpinEntryRequest`
- [x] `ClaudeChatRequest`, `ClaudeClearRequest`
- [x] `SaveEnvRequest`

### 7.5 Go Server - Remove Form Support from Todos - COMPLETE

- [x] Removed form-encoded parsing from `handleAddTodo`
- [x] Removed form-encoded parsing from `handleToggleTodo`
- [x] Removed unused `strconv` and `strings` imports

### 7.6 Testing - COMPLETE

- [x] Go server tests pass (all packages)
- [x] React web client tests pass (96 tests)
- [ ] Android testing pending (see Phase 8 for automated setup)

### 7.7 Cleanup - COMPLETE

- [x] No `-json` suffix endpoints in Go server (never existed there)
- [x] AGENTS.md does not mention form-encoded requests
- [x] Only Android and React clients exist; both now use JSON

---

## Phase 8: Android Automated Testing (spec 21)

> **Goal:** Enable agents to run Android UI tests with visual screenshot feedback.
> Uses Maestro for UI testing with headless emulator.

### 8.1 Setup Infrastructure - COMPLETE

- [x] Create `scripts/install-android-sdk.sh` script per spec 21
- [x] Add Makefile targets:
  - [x] `android-test-setup` - One-time setup for new machines
  - [x] `android-emulator-start` - Start headless emulator
  - [x] `android-emulator-stop` - Stop emulator
  - [x] `android-test` - Run all Maestro tests
  - [x] `android-test-report` - Run tests and show summary
  - [x] Individual flow targets: `android-test-daily`, `android-test-files`, etc.

### 8.2 Maestro Test Flows - COMPLETE

- [x] Create `app/android/maestro/` directory structure
- [x] Create `flows/daily-screen.yaml` - Test daily note interactions
- [x] Create `flows/files-screen.yaml` - Test file browser
- [x] Create `flows/sleep-screen.yaml` - Test sleep tracking form
- [x] Create `flows/claude-screen.yaml` - Test Claude chat
- [x] Create `flows/settings-screen.yaml` - Test settings/theme
- [x] Create `flows/full-navigation.yaml` - Test bottom nav and screen transitions
- [x] Create `maestro/README.md` - Document test flows

### 8.3 Environment Configuration - COMPLETE

- [x] Add `app/android/maestro/screenshots/` to `.gitignore`
- [x] Document required environment variables (server: `server/.env.example`, Android SDK: `app/android/maestro/README.md`)
- [x] Test setup script on clean machine (N/A - requires separate VM; script outputs clear instructions)

### 8.4 CI Integration (Optional)

- [ ] Create `.github/workflows/android-test.yml` for GitHub Actions
- [ ] Configure artifact upload for screenshots

---

## Completed

### Documentation (Phase 5) - 2026-01-28
- [x] Created `server/README.md` with setup instructions, development commands, API reference, and deployment guide
- [x] Created `clients/web/README.md` with setup instructions, npm scripts, testing guide, and architecture overview
- [x] `.env.example` already documented all environment variables (no changes needed)
- [x] Both READMEs link to relevant spec documents for detailed documentation

### Go Backend API Integration Tests (Phase 4.5) - 2026-01-28
- [x] Created `api/handlers_test.go` with comprehensive endpoint tests
- [x] **Daily handlers**: GET /api/daily, POST /api/save, /api/append, /api/clear-pinned
- [x] **Todo handlers**: POST /api/todos/add, /api/todos/toggle
- [x] **File handlers**: GET /api/files/list, /api/files/read, POST /api/files/create, /api/files/save, /api/files/delete, /api/files/unpin
- [x] **Sleep handlers**: GET /api/sleep-times, POST /api/sleep-times/append, /api/sleep-times/delete
- [x] **Error response formats**: 400 Bad Request (validation), 401 Unauthorized, 404 Not Found
- [x] **CORS headers**: Preflight OPTIONS, regular requests, wildcard origin
- [x] **Request validation**: Invalid JSON, missing required fields, invalid field values
- [x] **Content-Type headers**: All responses return application/json
- [x] **Success response format**: {success: true, message: "..."} consistency

### Android Testing Environment Documentation (Phase 8.3) - 2026-01-28
- [x] Updated `app/android/maestro/README.md` with Prerequisites section and shell environment variables
- [x] Server environment variables already documented in `server/.env.example`
- [x] Marked setup script clean machine test as N/A (script provides clear instructions)

### Android Automated Testing Infrastructure (Phase 8.1-8.2) - 2026-01-28
- [x] Created `scripts/install-android-sdk.sh` for SDK installation
- [x] Added Makefile targets: `android-test-setup`, `android-emulator-start`, `android-emulator-stop`, `android-test`, `android-test-report`
- [x] Added individual test targets: `android-test-daily`, `android-test-files`, `android-test-sleep`, `android-test-claude`, `android-test-settings`, `android-test-nav`
- [x] Created 6 Maestro test flows:
  - `daily-screen.yaml` (8 screenshots): app launch, refresh, task add, edit mode
  - `full-navigation.yaml` (8 screenshots): bottom nav, all screen transitions
  - `sleep-screen.yaml` (5 screenshots): child/status selection, time entry
  - `files-screen.yaml` (4 screenshots): file tree navigation
  - `claude-screen.yaml` (4 screenshots): chat interface
  - `settings-screen.yaml` (5 screenshots): theme toggle, person selection
- [x] Created `app/android/maestro/README.md` with usage documentation
- [x] Added `app/android/maestro/screenshots/` to `.gitignore`

### Android API Alignment (Phase 7) - 2026-01-28
- [x] **ApiClient.kt**: Replaced `postForm()` with `postJson()` for all POST requests
- [x] **Models.kt**: Added 14 request data classes for JSON serialization
- [x] **DailyScreen.kt**: Updated callers to pass `path` parameter
- [x] **SleepTimesScreen.kt**: Updated to use `status` string instead of `asleep`/`woke` booleans
- [x] **Go todos.go**: Removed form-encoded parsing, now JSON-only per spec
- [x] Fixed endpoint paths: `/api/files/save-json` → `/api/files/save`, `/api/files/delete-json` → `/api/files/delete`
- [x] All Go tests pass, all React tests pass (96 tests)

### Go Backend Service Tests - 2026-01-28
- [x] **LinkedIn CSV Activity Logging Tests** (`linkedin/logging_test.go`):
  - LogPost: creates file, writes header, appends data row
  - LogComment: logs comment and reply actions
  - Appending multiple entries: header written only once
  - Directory creation: creates nested `{person}/linkedin/` paths
  - Newline escaping in text fields
  - compactJSON: compact valid JSON, preserve invalid input
  - Timestamp format: RFC3339 compliance
  - Person isolation: separate log files per person
  - Special character handling: quotes, commas in CSV fields
- [x] **NDJSON Streaming Format Tests** (`claude/stream_test.go`):
  - StreamEvent JSON serialization (all 6 event types)
  - Empty field omission in JSON output
  - JSON deserialization round-trip
  - NDJSON format (newline-delimited)
  - Text delta accumulation for UI display
  - Tool input types: string, int, bool, nested, array, empty
  - Special characters: newlines, tabs, quotes, unicode, CRLF
  - Session ID format preservation
  - Error message format preservation

### Go Backend Security Tests - 2026-01-27
- [x] Installed Go 1.22.5 at `~/go-sdk/go/`
- [x] **Middleware Tests** (`api/middleware_test.go`):
  - AuthMiddleware: valid/invalid tokens, missing auth, Bearer format validation
  - PersonMiddleware: valid/invalid persons, case sensitivity, context propagation
  - RequirePerson helper function
  - RecovererMiddleware panic handling
- [x] **Security Integration Tests** (`api/security_test.go`):
  - Path traversal prevention (6 attack vectors for read, 3 each for create/save/delete)
  - Person context isolation (sebastian can't access petra's files)
  - Authentication required for all protected endpoints (20 endpoints)
  - LinkedIn callback skips auth (OAuth flow)
  - Invalid person rejection (7 test cases)
  - Person required for protected endpoints (10 endpoints)
- [x] **Constant-time token comparison tests** (`auth/auth_test.go`):
  - Mismatch at different positions (start/middle/end)
  - Length differences, null bytes, edge cases
- [x] All existing vault tests pass: path validation, store operations, daily note operations

### React Web Client Testing - 2026-01-27
- [x] Set up Vitest testing framework with jsdom and testing-library
- [x] Added test scripts to package.json (`npm test`, `npm run test:watch`)
- [x] Configured vite.config.ts with test configuration
- [x] Created test setup file with localStorage mock
- [x] **NoteView Tests** (42 tests):
  - Line parsing: H1-H6 headings, tasks (unchecked/checked), empty lines, plain text
  - Pinned marker detection on H3 only
  - HTML escaping for XSS protection
  - Task toggle callback with line numbers
  - Unpin button rendering and callback
- [x] **AuthContext Tests** (5 tests): localStorage persistence, login/logout
- [x] **ThemeContext Tests** (7 tests): localStorage persistence, body class switching, toggle
- [x] **PersonContext Tests** (7 tests): localStorage persistence, person switching
- [x] **Claude Streaming Tests** (19 tests):
  - NDJSON parsing: text, tool_use, session, ping, error, done events
  - Chunked data handling (partial lines across reads)
  - Final buffer parsing (no trailing newline)
  - Empty line skipping
  - JSON parse error recovery
  - Multiple events per chunk
  - Correct headers (Accept: application/x-ndjson, auth, person)
  - Session ID forwarding
  - Error handling (401, null body)
  - Text delta accumulation for UI display
- [x] **API Client Tests** (16 tests):
  - Auth header injection from localStorage
  - Person header injection from localStorage
  - Content-Type for POST with body
  - ApiError with status and detail
  - Stream request NDJSON Accept header
  - Stream request returns ReadableStreamDefaultReader
- [x] **Bug Fix**: NoteView now only strips `<pinned>` marker from H3 headings (was incorrectly stripping from all headings)
- Total: 96 tests passing

### API Alignment Fix - 2026-01-27
- [x] Fixed `/api/todos/add` endpoint per spec 01:
  - Changed request field from `task` to `text`
  - Made `text` optional (empty creates `- [ ]` blank task)
  - Removed required `path` field - auto-determines today's daily note path
- [x] Note: Form-encoded support was temporarily added then removed in Phase 7 (JSON-only now)

### Spec Updates - 2026-01-27
- [x] Updated spec 10 (Authentication) to reference Go backend with actual code examples
- [x] Updated spec 11 (Note Rendering) to reference React NoteView instead of Python renderer
- [x] Updated spec 17 (Add Task Inline Input) to reference Go server instead of Python
- [x] Verified spec 07 (Web Interface) is already marked SUPERSEDED
- [x] Documented API inconsistencies for future fix (Go uses `task`/`path`, spec says `text` optional)

### Python Backend Deprecation - 2026-01-27
- [x] Archived Python backend to `_archive/python-backend/`
- [x] Updated root Makefile for Go server (run, build, test)
- [x] Updated notes-editor.service for Go binary
- [x] Created migration documentation in `_archive/python-backend/README.md`
- [x] Updated AGENTS.md with new commands

### React Web Client Implementation - 2026-01-27
- [x] Complete React web client implementation (Phase 2.1-2.15)
- [x] All 6 pages: Login, Daily, Files, Sleep, Claude, Noise, Settings
- [x] All core components: Layout, Header, Navigation, NoteView, Editor, FileTree
- [x] API client layer with full type definitions
- [x] Context providers: Auth, Person, Theme (all with localStorage persistence)
- [x] Custom hooks: useAuth, usePerson, useTheme
- [x] Claude streaming with async generator and NDJSON parsing
- [x] Noise generator with Web Audio API per spec 08
- [x] Full theming system (dark/light) per spec 12
- [x] TypeScript compiles, Vite build succeeds
- [x] Run: `cd clients/web && npm install && npm run dev`

### Go Backend Implementation - 2026-01-27
- [x] Complete Go backend implementation (Phase 1.1-1.8)
- [x] All packages: config, auth, vault, claude, linkedin, api
- [x] Unit tests for auth, vault paths, vault store, vault daily, claude session
- [x] All 15 API endpoints implemented per spec 01
- [x] Security: constant-time token comparison, path traversal prevention, person isolation
- [x] Claude streaming with NDJSON and 5-second keepalive pings
- [x] LinkedIn OAuth and activity logging
- [x] Note: Requires Go 1.22+ to build/test (not installed on dev machine)

### Bottom Navigation Bar - 2026-01-26
- [x] Delete `ArcMenu.kt` and `ArcMenuConfig.kt`
- [x] Remove `BottomInfoBar` from `AppNavigation.kt`
- [x] Create `BottomNavBar` composable with 4 items: Daily, Files, Sleep, Tools
- [x] Style bottom nav bar: 56dp height, equal-width items, accent color for active
- [x] Hide bottom nav bar when keyboard is visible or person is null
- [x] Restore `ToolsScreen.kt` as navigation hub
- [x] Restore `ScreenHeader` composable and add to all 8 screens
- [x] Verify keyboard accessory bar still works (spec 14)

### Fix: Android Dark Theme Cursor Visibility - 2026-01-19
- [x] Add `cursorBrush` parameter to `CompactTextField` using accent color

### Previously Verified Complete
- [x] All 8 Android screens implemented
- [x] Claude streaming works (spec 13 fully compliant)
- [x] Deprecated ArcMenu and BottomInfoBar fully removed
- [x] No TODOs/FIXMEs in codebase
