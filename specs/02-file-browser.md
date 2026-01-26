# File Browser

## Purpose

The File Browser provides a person-scoped hierarchical file tree for organizing documents, notes, and personal information within the vault. It allows users to navigate, create, read, edit, and delete files while maintaining strict isolation between different users' data spaces.

## Security Model

### Person Scoping

All file operations are scoped to the authenticated person's root directory (`~/notes/{person}/`). The person is determined from:
1. `X-Notes-Person` header (API clients)
2. `notes_person` cookie (web browser)

Requests without a valid person selection are redirected to `/settings` (web) or rejected with HTTP 400 (API).

### Path Validation

The `vault_store.resolve_path()` function enforces strict path security:

```python
def resolve_path(relative_path: str) -> Path:
    if not relative_path:
        raise ValueError("Path is required")
    if Path(relative_path).is_absolute():
        raise ValueError("Path must be vault-relative")

    full_path = (VAULT_ROOT / relative_path).resolve()
    if VAULT_ROOT.resolve() not in full_path.parents and full_path != VAULT_ROOT.resolve():
        raise ValueError("Path escapes vault root")

    return full_path
```

This prevents:
- **Absolute paths**: Only vault-relative paths are accepted
- **Directory traversal**: Paths like `../other_person/secret.md` are rejected by checking the resolved path stays within vault root
- **Path normalization attacks**: Uses `resolve()` to canonicalize paths before validation

### Hidden File Filtering

Files and directories starting with `.` are automatically filtered from directory listings, preventing exposure of system files like `.git/`.

## Features

### Directory Listing

Lists contents of a directory within the person's scope:
- Files sorted before directories
- Alphabetical sorting within each group (case-insensitive)
- Hidden files (`.`-prefixed) excluded
- Returns name, path (relative to person root), and `is_dir` flag

### Lazy-Loaded Tree Expansion

The web UI uses HTMX for on-demand directory loading:

```html
<button class="tree-toggle"
    data-target="children-{item_id}"
    hx-get="/api/files/tree?path={encoded_path}"
    hx-target="#children-{item_id}"
    hx-swap="innerHTML">+</button>
```

Clicking a directory toggle fetches its children via `/api/files/tree` and injects them into the DOM. This avoids loading the entire file tree upfront.

### CRUD Operations

| Operation | Description |
|-----------|-------------|
| **Create** | Creates empty file at specified path; parent directories created automatically |
| **Read** | Returns file content as plain text |
| **Update** | Overwrites file with new content |
| **Delete** | Removes file (directories cannot be deleted via API) |

### Markdown Rendering

Files are rendered with interactive elements:
- **Headings**: `#` through `######` rendered with appropriate styling
- **Task checkboxes**: `- [ ]` and `- [x]` rendered as interactive checkboxes that toggle via `/api/todos/toggle`
- **Pinned entries**: `### HH:MM <pinned>` headers shown with an "Unpin" button

### Git Auto-Commit

All write operations (create, save, delete) trigger automatic git commit and push:

```python
success, msg = git_commit_and_push("Update file")
```

The `git_sync` service:
1. Checks for uncommitted changes via `git status --porcelain`
2. Stages all changes with `git add .`
3. Commits with descriptive message
4. Pushes to remote (retries after pull if push fails)
5. Returns success even if push fails (changes saved locally)

## API Endpoints

### Web Pages

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/files` | GET | File browser page with tree view |
| `/file?path={path}` | GET | Single file view/edit page |

### JSON API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/files/list?path={path}` | GET | List directory contents as JSON |
| `/api/files/read?path={path}` | GET | Read file content as JSON |
| `/api/files/create` | POST | Create empty file (form: `path`) |
| `/api/files/save-json` | POST | Save file content (form: `path`, `content`) |
| `/api/files/delete-json` | POST | Delete file (form: `path`) |

### HTMX Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/files/tree?path={path}` | GET | Returns HTML fragment for directory children |
| `/api/files/open?path={path}` | GET | Returns file editor HTML fragment |
| `/api/files/save` | POST | Save file, returns editor HTML fragment |
| `/api/files/delete` | POST | Delete file, returns empty editor with `HX-Trigger: fileDeleted` |
| `/api/files/unpin` | POST | Remove `<pinned>` marker from entry (form: `path`, `line`) |

### Response Formats

**Directory listing (`/api/files/list`):**
```json
{
  "entries": [
    {"name": "notes.md", "path": "notes.md", "is_dir": false},
    {"name": "projects", "path": "projects", "is_dir": true}
  ]
}
```

**File read (`/api/files/read`):**
```json
{
  "path": "notes.md",
  "content": "# My Notes\n\nContent here..."
}
```

**Mutation responses:**
```json
{
  "success": true,
  "message": "File saved"
}
```

## Platform Differences

### Web (HTMX)

- **Tree rendering**: Server-side HTML generation via `render_tree()`
- **Lazy loading**: HTMX requests populate directory children on expand
- **Inline editing**: File content edited in a `<textarea>` within the page
- **Task toggle**: Checkboxes trigger HTMX POST to `/api/todos/toggle`
- **Git pull**: Performed on page load to sync latest changes

### Android (Jetpack Compose)

- **State management**: Uses Compose state (`mutableStateOf`) for tree expansion and file content
- **Tree rendering**: Client-side recursive `FileTree` composable
- **Lazy loading**: Coroutine-based API calls populate `entriesByPath` map on directory expand
- **Navigation**: `BackHandler` manages view/edit state transitions
- **API client**: OkHttp with `X-Notes-Person` header from `UserSettings.person`

Key differences:
| Aspect | Web | Android |
|--------|-----|---------|
| Tree HTML | Server-rendered | Client-rendered |
| Edit mode | Same page | Replaces tree view |
| Back button | Browser back | Custom `BackHandler` |
| Git sync | Server-side pull | Implicit via API |

## Data Model

### Vault Structure

```
~/notes/
  sebastian/           # Person root
    daily/
      2024-01-15.md
    projects/
      work/
        notes.md
    personal.md
  petra/               # Person root
    daily/
      2024-01-15.md
```

### Path Resolution Flow

1. Client sends path relative to person root (e.g., `projects/work/notes.md`)
2. Server prepends person prefix: `sebastian/projects/work/notes.md`
3. `resolve_path()` validates against vault root
4. File operations use the full path
5. Responses strip person prefix for client display

## Implementation Files

| File | Purpose |
|------|---------|
| `server/web_app/services/vault_store.py` | Core file operations with path validation |
| `server/web_app/services/git_sync.py` | Git commit and push automation |
| `server/web_app/renderers/file_tree.py` | HTML tree generation for web UI |
| `server/web_app/renderers/pinned.py` | Markdown rendering with interactive elements |
| `server/web_app/main.py` | FastAPI endpoint definitions |
| `app/android/.../FilesScreen.kt` | Android file browser UI |
| `app/android/.../ApiClient.kt` | Android API client |
