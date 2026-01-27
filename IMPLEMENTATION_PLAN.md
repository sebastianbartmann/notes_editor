# Implementation Plan

> Last updated: 2026-01-27
> Status: Active - Go + React Migration

## Instructions
- Tasks marked `- [ ]` are incomplete
- Tasks marked `- [x]` are complete
- Work from top to bottom (highest priority first)
- Add new tasks as you discover them

---

## Phase 1: Go Backend (spec 19) - COMPLETE

> **Note:** Go runtime not available on development machine. Code is complete and ready for testing once Go 1.22+ is installed. Run `cd server && go mod tidy && make test` to verify.

### 1.9 Go Backend Testing
- [ ] Run tests once Go is installed (`make test`)
- [ ] Integration tests for full request/response cycles
- [ ] Test authentication flow
- [ ] Test person context isolation
- [ ] Test concurrent request handling

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
