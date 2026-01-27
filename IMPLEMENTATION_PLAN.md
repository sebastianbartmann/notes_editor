# Implementation Plan

> Last updated: 2026-01-27
> Status: Active - Go + React Migration

## Instructions
- Tasks marked `- [ ]` are incomplete
- Tasks marked `- [x]` are complete
- Work from top to bottom (highest priority first)
- Add new tasks as you discover them

---

## Phase 1: Go Backend (spec 19)

### 1.1 Project Setup
- [ ] Create `server/` directory structure
  - [ ] Create `cmd/server/` for main entry point
  - [ ] Create `internal/api/` for HTTP handlers
  - [ ] Create `internal/vault/` for file operations
  - [ ] Create `internal/claude/` for AI service
  - [ ] Create `internal/linkedin/` for LinkedIn integration
  - [ ] Create `internal/auth/` for authentication
  - [ ] Create `internal/config/` for configuration
- [ ] Initialize Go module
  - [ ] Create `go.mod` with module name `notes-editor`, Go 1.22
  - [ ] Add dependencies: chi, cors, godotenv
- [ ] Create Makefile with targets: build, test, test-coverage, lint, run
- [ ] Create `.env.example` with all required environment variables

### 1.2 Configuration Package (`internal/config/`)
- [ ] Create `config.go`
  - [ ] Define `Config` struct (NotesToken, NotesRoot, AnthropicKey, LinkedInConfig)
  - [ ] Implement `Load()` function using godotenv
  - [ ] Validate required fields

### 1.3 Authentication Package (`internal/auth/`)
- [ ] Create `auth.go`
  - [ ] Implement `ValidateToken()` with constant-time comparison
  - [ ] Implement `PersonFromContext()` and `WithPerson()`
- [ ] Create `person.go`
  - [ ] Define person context key type
  - [ ] Define valid persons list (sebastian, petra)
  - [ ] Implement person validation logic
- [ ] Create `auth_test.go` with full test coverage

### 1.4 Vault Package (`internal/vault/`)

#### 1.4.1 Path Validation
- [ ] Create `paths.go`
  - [ ] Implement `ValidatePath()` - reject empty, absolute, traversal paths
  - [ ] Implement `ResolvePath()` - safely join root + person + path
  - [ ] Define custom errors: ErrEmptyPath, ErrAbsolutePath, ErrPathEscape

#### 1.4.2 Store Operations
- [ ] Create `store.go`
  - [ ] Define `Store` struct with rootPath
  - [ ] Implement `NewStore(rootPath)`
  - [ ] Implement `ReadFile(person, path)` - validate, read, return content
  - [ ] Implement `WriteFile(person, path, content)` - validate, create dirs, write
  - [ ] Implement `AppendFile(person, path, content)`
  - [ ] Implement `DeleteFile(person, path)` - idempotent delete
  - [ ] Implement `ListDir(person, path)` - filter hidden, sort entries
  - [ ] Define `FileEntry` struct (Name, Path, IsDir)
- [ ] Create `store_test.go`
  - [ ] Test path traversal prevention
  - [ ] Test all CRUD operations
  - [ ] Test hidden file filtering

#### 1.4.3 Git Sync
- [ ] Create `git.go`
  - [ ] Implement `GitPull()` with fallback to fetch+reset
  - [ ] Implement `GitCommitAndPush()` with retry on failure
  - [ ] Handle merge conflicts with remote-wins strategy
- [ ] Create `git_test.go`

#### 1.4.4 Daily Note Logic
- [ ] Create `daily.go`
  - [ ] Implement `GetOrCreateDaily(person, date)`
  - [ ] Implement `findPreviousNote()` - find most recent note
  - [ ] Implement `extractIncompleteTodos()` - parse `- [ ]` lines
  - [ ] Implement `extractPinnedNotes()` - find `<pinned>` entries
  - [ ] Implement `generateDailyNote()` - template with inheritance
  - [ ] Implement `addTask()` - add to work/priv category
  - [ ] Implement `toggleTask()` - toggle checkbox by line
  - [ ] Implement `clearAllPinned()` - remove all markers
  - [ ] Implement `unpinEntry()` - remove marker from specific line
  - [ ] Implement `appendEntry()` - add timestamped entry
- [ ] Create `daily_test.go`
  - [ ] Test todo inheritance
  - [ ] Test pinned note inheritance
  - [ ] Test task operations

### 1.5 Claude Package (`internal/claude/`)

#### 1.5.1 Session Management
- [ ] Create `session.go`
  - [ ] Define `ChatMessage` struct (Role, Content)
  - [ ] Define `Session` struct (ID, Person, Messages, Mutex)
  - [ ] Define `SessionStore` with map and RWMutex
  - [ ] Implement `GetOrCreate()`, `Clear()`, `GetHistory()`
- [ ] Create `session_test.go`

#### 1.5.2 Tool Definitions
- [ ] Create `tools.go`
  - [ ] Define `read_file` tool schema
  - [ ] Define `write_file` tool schema
  - [ ] Define `list_directory` tool schema
  - [ ] Define `search_files` tool schema
  - [ ] Define `web_search` tool schema
  - [ ] Define `web_fetch` tool schema
  - [ ] Define `linkedin_post` tool schema
  - [ ] Define `linkedin_read_comments` tool schema
  - [ ] Define `linkedin_post_comment` tool schema
  - [ ] Define `linkedin_reply_comment` tool schema
  - [ ] Implement `executeTool()` dispatcher

#### 1.5.3 Chat Service
- [ ] Create `service.go`
  - [ ] Define `Service` struct (apiKey, store, linkedin, sessions)
  - [ ] Implement `NewService()`
  - [ ] Implement `Chat()` - non-streaming with tool loop
  - [ ] Define system prompt with security warnings
- [ ] Create `stream.go`
  - [ ] Define `StreamEvent` struct (Type, Delta, Name, Input, SessionID, Message)
  - [ ] Implement `ChatStream()` - returns event channel
  - [ ] Handle text deltas, tool use, ping keepalives
  - [ ] Implement 5-second keepalive ping
- [ ] Create `service_test.go` with mocked API

### 1.6 LinkedIn Package (`internal/linkedin/`)

#### 1.6.1 OAuth
- [ ] Create `oauth.go`
  - [ ] Define `TokenResponse` struct
  - [ ] Implement `ExchangeCodeForToken()`
  - [ ] Implement `PersistAccessToken()` - update .env file

#### 1.6.2 API Client
- [ ] Create `client.go`
  - [ ] Define `Service` struct (config, vaultRoot, client)
  - [ ] Implement `GetPersonURN()`
  - [ ] Implement `CreatePost(text, person)`
  - [ ] Implement `ReadComments(postURN)`
  - [ ] Implement `CreateComment(postURN, text, parentURN, person)`
- [ ] Create `client_test.go`

#### 1.6.3 Activity Logging
- [ ] Create `logging.go`
  - [ ] Implement `LogPost()` - CSV append
  - [ ] Implement `LogComment()` - CSV append
  - [ ] CSV format: timestamp, action, post_urn, comment_urn, text, response

### 1.7 API Package (`internal/api/`)

#### 1.7.1 Middleware
- [ ] Create `middleware.go`
  - [ ] Implement `AuthMiddleware()` - validate Bearer token
  - [ ] Implement `PersonMiddleware()` - extract X-Notes-Person header
  - [ ] Implement `LoggingMiddleware()` - log request/response
  - [ ] Implement `RecovererMiddleware()` - panic recovery

#### 1.7.2 Router
- [ ] Create `router.go`
  - [ ] Implement `NewRouter()` with chi
  - [ ] Configure CORS middleware
  - [ ] Mount all API routes

#### 1.7.3 Handlers
- [ ] Create `daily.go`
  - [ ] `GET /api/daily` - get/create today's note
  - [ ] `POST /api/save` - save note content
  - [ ] `POST /api/append` - append timestamped entry
  - [ ] `POST /api/clear-pinned` - clear all pinned markers
- [ ] Create `todos.go`
  - [ ] `POST /api/todos/add` - add task to category
  - [ ] `POST /api/todos/toggle` - toggle checkbox by line
- [ ] Create `sleep.go`
  - [ ] `GET /api/sleep-times` - get recent entries
  - [ ] `POST /api/sleep-times/append` - add entry
  - [ ] `POST /api/sleep-times/delete` - delete by line
- [ ] Create `files.go`
  - [ ] `GET /api/files/list` - list directory
  - [ ] `GET /api/files/read` - read file content
  - [ ] `POST /api/files/create` - create empty file
  - [ ] `POST /api/files/save` - save file content
  - [ ] `POST /api/files/delete` - delete file
  - [ ] `POST /api/files/unpin` - unpin specific entry
- [ ] Create `claude.go`
  - [ ] `POST /api/claude/chat` - non-streaming chat
  - [ ] `POST /api/claude/chat-stream` - NDJSON streaming
  - [ ] `POST /api/claude/clear` - clear session
  - [ ] `GET /api/claude/history` - get session history
- [ ] Create `settings.go`
  - [ ] `GET /api/settings/env` - read .env file
  - [ ] `POST /api/settings/env` - update .env file
- [ ] Create `linkedin.go`
  - [ ] `GET /api/linkedin/oauth/callback` - OAuth callback

#### 1.7.4 Error Handling
- [ ] Create `errors.go`
  - [ ] Implement `writeError()` helper with proper HTTP status codes
  - [ ] Implement `writeSuccess()` helper
  - [ ] Implement `writeStreamError()` for NDJSON
  - [ ] Define error response format: `{"detail": "message"}` for 400/404
  - [ ] Implement 401 Unauthorized for auth failures
  - [ ] Implement 400 Bad Request for validation errors
  - [ ] Implement 404 Not Found for missing resources

### 1.8 Main Entry Point
- [ ] Create `cmd/server/main.go`
  - [ ] Load configuration
  - [ ] Initialize all services
  - [ ] Create router
  - [ ] Start HTTP server on port 8080
  - [ ] Handle graceful shutdown

### 1.9 Go Backend Testing
- [ ] Integration tests for full request/response cycles
- [ ] Test authentication flow
- [ ] Test person context isolation
- [ ] Test concurrent request handling

---

## Phase 2: React Web Client (spec 20)

### 2.1 Project Setup
- [ ] Create `clients/web/` directory
- [ ] Initialize npm project with `package.json`
  - [ ] Add dependencies: react, react-dom, react-router-dom
  - [ ] Add dev dependencies: vite, @vitejs/plugin-react, typescript, @types/react
  - [ ] Add scripts: dev, build, preview
- [ ] Create `tsconfig.json` with React settings
- [ ] Create `vite.config.ts` with API proxy to localhost:8080
- [ ] Create `index.html` root HTML file

### 2.2 Core Application
- [ ] Create `src/main.tsx` entry point
- [ ] Create `src/App.tsx` with routing and provider hierarchy
- [ ] Create `src/index.css` with CSS variables and global styles

### 2.3 API Client Layer (`src/api/`)
- [ ] Create `types.ts` - all API response/request types
- [ ] Create `client.ts` - base HTTP client with auth headers
- [ ] Create `daily.ts` - fetchDaily, saveDaily, appendDaily, toggleTask
- [ ] Create `files.ts` - listFiles, getFile, saveFile, createFile, deleteFile
- [ ] Create `todos.ts` - addTodo, toggleTodo
- [ ] Create `sleep.ts` - fetchSleepTimes, appendSleepTime, deleteSleepTime
- [ ] Create `claude.ts` - streamChat, clearSession

### 2.4 Context Providers (`src/context/`)
- [ ] Create `AuthContext.tsx`
  - [ ] AuthState interface, login/logout functions
  - [ ] Sync token to localStorage
- [ ] Create `PersonContext.tsx`
  - [ ] PersonState interface, setPerson function
  - [ ] Sync person to localStorage
- [ ] Create `ThemeContext.tsx`
  - [ ] Theme type (dark/light), setTheme function
  - [ ] Update data-theme attribute on document

### 2.5 Custom Hooks (`src/hooks/`)
- [ ] Create `useAuth.ts` - return auth state and methods
- [ ] Create `usePerson.ts` - return person state and methods
- [ ] Create `useTheme.ts` - return theme state and methods
- [ ] Create `useApi.ts` - loading/error state wrapper
- [ ] Create `useClaudeStream.ts`
  - [ ] Manage chat messages state
  - [ ] Implement sendMessage with NDJSON streaming
  - [ ] Parse stream events (text, tool, ping, done, error)
  - [ ] Implement clearSession

### 2.6 Layout Components (`src/components/Layout/`)
- [ ] Create `Layout.tsx` - main wrapper with header and nav
- [ ] Create `Header.tsx` - page title, theme toggle, person selector
- [ ] Create `Navigation.tsx` - links to: Daily, Files, Sleep, Claude, Noise, Settings

### 2.7 NoteView Components (`src/components/NoteView/`)
- [ ] Create `NoteView.tsx` - main renderer
  - [ ] Accept props: content, path, onTaskToggle, onUnpin
  - [ ] Parse lines with LineType enum (H1-H6, TASK, TEXT, EMPTY)
  - [ ] Task regex: `^\s*-\s*\[([ xX])\]\s*(.*)$`
  - [ ] Heading regex: `^(#{1,6})\s+(.*)$`
  - [ ] Pinned detection: `<pinned>` in H3
  - [ ] Render empty lines with `&nbsp;` entity
  - [ ] Apply HTML escaping for XSS protection
- [ ] Create `TaskLine.tsx` - interactive checkbox
  - [ ] Render checkbox with checked state
  - [ ] Call onTaskToggle with line number
  - [ ] Style completed tasks (strikethrough, muted)
- [ ] Create `UnpinButton.tsx` - per-entry unpin action
  - [ ] Render button in pinned H3 headings
  - [ ] Call onUnpin with path and line number
  - [ ] Style: border accent-dim, accent text, hover background
- [ ] Create `NoteView.module.css`
  - [ ] Style headings (H1: 16px bold, H2: 14px uppercase muted, H3: 13px accent, H4: 12px)
  - [ ] Style task checkboxes with accent-color
  - [ ] Style pinned headings: background #151a1f, border-radius 4px
  - [ ] Style unpin button: border-radius 999px, font-size 11px

### 2.8 Editor Component (`src/components/Editor/`)
- [ ] Create `Editor.tsx` - markdown editor
  - [ ] Textarea with content
  - [ ] Save/Cancel buttons
  - [ ] Track local edits
- [ ] Create `Editor.module.css`

### 2.9 FileTree Component (`src/components/FileTree/`)
- [ ] Create `FileTree.tsx` - directory browser
  - [ ] Lazy loading for subdirectories
  - [ ] Expand/collapse state
  - [ ] File selection callback
- [ ] Create `FileTree.module.css`

### 2.10 Chat Components (`src/components/Chat/`)
- [ ] Create `ChatWindow.tsx` - main chat interface
  - [ ] Message list, input field, send button
  - [ ] Auto-scroll to latest message
- [ ] Create `ChatMessage.tsx` - message bubble
  - [ ] User/assistant styling
  - [ ] Markdown rendering
- [ ] Create `StreamingText.tsx` - incremental text display

### 2.11 Other Components

#### SleepForm (`src/components/SleepForm/`)
- [ ] Create `SleepForm.tsx` - sleep entry form
  - [ ] Child selection: Thomas/Fabian radio buttons (default: Fabian)
  - [ ] Status checkboxes: Eingeschlafen/Aufgewacht (mutual exclusion)
  - [ ] Time entry input field with placeholder
  - [ ] Submit button
  - [ ] Reset form after successful submit
- [ ] Create `SleepHistory.tsx` - recent entries list
  - [ ] Display last 20 entries in reverse chronological order
  - [ ] Delete button per entry (uses line number)
  - [ ] Format: YYYY-MM-DD | Name | Time | Status

#### NoisePlayer (`src/components/NoisePlayer/`)
- [ ] Create `NoisePlayer.tsx` - Web Audio procedural noise (spec 08)
  - [ ] Create AudioContext on user interaction
  - [ ] Generate 2-second white noise buffer
  - [ ] Create bass layer: 900Hz lowpass + 50Hz highpass, gain 0.3, +4dB boost
  - [ ] Create high layer: 6000Hz lowpass + 1200Hz highpass, gain 0.08
  - [ ] Implement LFO modulation: 0.07 Hz sine wave, 0.025 gain depth
  - [ ] Implement drift timer: random gain adjustment every 2.4 seconds
  - [ ] Base gain: 0.24
  - [ ] Play/stop toggle button
  - [ ] Display playing state

### 2.12 Page Components (`src/pages/`)
- [ ] Create `DailyPage.tsx`
  - [ ] Fetch daily note on mount
  - [ ] NoteView for display, Editor for edit mode
  - [ ] Task toggle, append form
- [ ] Create `FilesPage.tsx`
  - [ ] FileTree for navigation
  - [ ] NoteView/Editor for selected file
  - [ ] Create/delete file actions
- [ ] Create `SleepPage.tsx`
  - [ ] SleepForm component for new entries
  - [ ] SleepHistory component showing recent 20 entries
  - [ ] Handle delete entry action
  - [ ] Refresh list after add/delete
- [ ] Create `ClaudePage.tsx`
  - [ ] ChatWindow with streaming
  - [ ] Clear session button
- [ ] Create `NoisePage.tsx`
  - [ ] NoisePlayer component
- [ ] Create `SettingsPage.tsx`
  - [ ] Person selector
  - [ ] Theme toggle
  - [ ] Logout button

### 2.13 Routing
- [ ] Configure BrowserRouter in App.tsx
- [ ] Define routes: /, /daily, /files, /files/:path, /sleep, /claude, /noise, /settings
- [ ] Wrap routes in Layout component
- [ ] Wrap app in provider hierarchy (Auth > Person > Theme)

### 2.14 Theming (`src/index.css`)

#### Dark Theme (Default)
- [ ] `--bg: #0f1012`
- [ ] `--panel: #15171a`
- [ ] `--panel-border: #2a2d33`
- [ ] `--text: #e6e6e6`
- [ ] `--muted: #9aa0a6`
- [ ] `--accent: #d9832b`
- [ ] `--accent-dim: #7a4a1d`
- [ ] `--danger: #d66b6b`
- [ ] `--input: #0f1114`
- [ ] `--note: #101317`

#### Light Theme (body.theme-light)
- [ ] `--bg: #e9f7f7`
- [ ] `--panel: #f6fbff`
- [ ] `--panel-border: #c7e3e6`
- [ ] `--text: #1a2a2f`
- [ ] `--muted: #4f6f78`
- [ ] `--accent: #3aa7a3`
- [ ] `--accent-dim: #c9f1ef`
- [ ] `--input: #f2fafb`
- [ ] `--note: #f9fdff`

#### Spacing and Typography
- [ ] `--space-1: 6px`, `--space-2: 10px`, `--space-3: 14px`, `--space-4: 18px`
- [ ] `--radius: 6px`
- [ ] `--font: "IBM Plex Mono", monospace`
- [ ] Base font-size: 14px, line-height: 1.5

### 2.15 Build Configuration
- [ ] Vite dev server with API proxy
- [ ] Production build to `dist/`
- [ ] Source maps for debugging

---

## Phase 3: Migration and Cleanup

### 3.1 Deprecate Python Backend
- [ ] Document migration path for any custom integrations
- [ ] Archive Python code to `_archive/python-backend/`
- [ ] Update deployment scripts for Go backend
- [ ] Update CI/CD pipelines

### 3.2 Update Specs
- [ ] Mark spec 07 (web-interface) as fully deprecated
- [ ] Update spec 10 to reference Go auth implementation
- [ ] Verify all specs match Go+React implementation

---

## Phase 4: Testing

### 4.1 Go Backend Tests - Security Critical
- [ ] Test path traversal prevention (../../../etc/passwd attacks)
- [ ] Test person context isolation (sebastian can't access petra's files)
- [ ] Test constant-time token comparison
- [ ] Test authentication middleware rejects invalid tokens

### 4.2 Go Backend Tests - Core Logic
- [ ] Test daily note creation with inherited todos
- [ ] Test daily note creation with inherited pinned notes
- [ ] Test todo toggle (checked/unchecked)
- [ ] Test task addition to categories
- [ ] Test pinned marker operations
- [ ] Test git pull with conflict resolution
- [ ] Test git commit and push with retry

### 4.3 Go Backend Tests - Services
- [ ] Test Claude session management (concurrent access)
- [ ] Test Claude tool execution
- [ ] Test NDJSON streaming format
- [ ] Test 5-second keepalive ping in streaming
- [ ] Test LinkedIn OAuth token exchange
- [ ] Test LinkedIn CSV activity logging

### 4.4 Go Backend Tests - API
- [ ] Integration tests for all endpoints
- [ ] Test error response formats (400, 401, 404)
- [ ] Test CORS headers
- [ ] Test request validation

### 4.5 React Web Client Tests
- [ ] Test NoteView line parsing (H1-H6, tasks, text, empty)
- [ ] Test task toggle state management
- [ ] Test streaming text incremental display
- [ ] Test theme switching (dark/light)
- [ ] Test person context switching
- [ ] Test localStorage persistence (token, person, theme)
- [ ] Test useClaudeStream NDJSON parsing
- [ ] Test useClaudeStream error handling

### 4.6 Android App Tests (existing gap)
- [ ] Unit tests for ApiClient failover logic
- [ ] Unit tests for NoteView markdown parsing
- [ ] UI tests for daily screen task toggle
- [ ] UI tests for navigation flows

---

## Phase 5: Documentation

### 5.1 Go Backend Documentation
- [ ] Create `server/README.md` with setup instructions
- [ ] Document all environment variables in `.env.example`
- [ ] Document API endpoints (link to spec 01)

### 5.2 React Web Client Documentation
- [ ] Create `clients/web/README.md` with setup instructions
- [ ] Document development workflow (npm scripts)
- [ ] Document build and deployment

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
- [ ] Test navigation on all screens
- [ ] Test keyboard visibility behavior (bottom nav hides, accessory shows)
- [ ] Test with person=null (only settings accessible)

---

## Completed

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
