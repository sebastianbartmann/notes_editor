# Daily Notes Feature

## Purpose

Daily Notes is the primary feature of Notes Editor, providing a day-by-day journal for capturing tasks, quick notes, and ideas. Each day has its own markdown file, auto-created on first access with intelligent carryover of incomplete work from the previous day.

## File Storage

**Location:** `~/notes/{person}/daily/YYYY-MM-DD.md`

**Example:** `~/notes/sebastian/daily/2025-01-15.md`

Files are stored per-person in the vault and synced via git on every change.

## File Format

Daily notes follow a structured markdown format:

```markdown
# daily YYYY-MM-DD

## todos

### work

- [ ] Incomplete task
- [x] Completed task

### priv

- [ ] Personal task

## custom notes

### HH:MM

Free-form note content here.

### HH:MM <pinned>

This note will carry forward to tomorrow.
```

### Sections

| Section | Purpose |
|---------|---------|
| `# daily YYYY-MM-DD` | Header with date |
| `## todos` | Task lists organized by category |
| `### work` / `### priv` | Task subcategories under todos |
| `## custom notes` | Timestamped free-form entries |
| `### HH:MM` | Individual note entry (24-hour format) |
| `### HH:MM <pinned>` | Pinned entry that carries forward |

### Task Format

- `- [ ]` - Incomplete task
- `- [x]` - Completed task (case-insensitive `x` or `X`)

## Auto-Creation and Carryover

When accessing today's daily note, if the file does not exist:

1. **Create file** with header `# daily YYYY-MM-DD`
2. **Find most recent previous note** (excluding today)
3. **Extract incomplete todos** - All `- [ ]` items from `## todos` section
4. **Extract pinned notes** - All `### HH:MM <pinned>` entries from `## custom notes`
5. **Write initial content** with carried-over items
6. **Git commit** the new file

This ensures continuity: unfinished tasks and important pinned notes automatically appear in the new day's file.

## Features

### Task Management

- **Add task** - Creates empty `- [ ]` under specified category (work/priv)
- **Toggle task** - Switches between `- [ ]` and `- [x]` by line number
- **Categories** - Tasks organized under `### work` and `### priv` subsections

### Custom Notes

- **Append note** - Adds timestamped `### HH:MM` entry with content
- **Pinned option** - Checkbox to add `<pinned>` marker to entry
- **Clear pinned** - Removes all `<pinned>` markers from current note
- **Unpin single** - Removes `<pinned>` marker from specific entry by line

### Edit Mode

- **View mode** - Rendered markdown with interactive checkboxes
- **Edit mode** - Raw textarea for full content editing
- **Save** - Writes full content and commits to git

## API Endpoints

### GET /

**Response:** HTML page (web UI)

Renders the daily notes editor page for the selected person. Pulls latest git changes, ensures today's file exists with carryover, and displays the note.

### GET /api/daily

**Response:**
```json
{
  "date": "2025-01-15",
  "path": "daily/2025-01-15.md",
  "content": "# daily 2025-01-15\n\n..."
}
```

Returns today's daily note content. Creates file with carryover if needed.

### POST /api/save

**Form data:**
- `content` (required) - Full note content

**Response:**
```json
{
  "success": true,
  "message": "Note saved successfully"
}
```

Overwrites the entire daily note content.

### POST /api/append

**Form data:**
- `content` (required) - Note text to append
- `pinned` (optional) - "on" to add `<pinned>` marker

**Response:**
```json
{
  "success": true,
  "message": "Content appended successfully"
}
```

Appends a new timestamped entry under `## custom notes`. Creates the section if it does not exist.

### POST /api/todos/add

**Form data:**
- `category` (required) - "work" or "priv"

**Response:**
```json
{
  "success": true,
  "message": "Task added"
}
```

Adds empty `- [ ]` task under the specified category. Creates `## todos` section and category subsection if needed.

### POST /api/todos/toggle

**Form data:**
- `path` (required) - Relative file path
- `line` (required) - Line number (1-indexed)

**Response:**
```json
{
  "success": true,
  "message": "Task updated"
}
```

Toggles task at specified line between `- [ ]` and `- [x]`.

### POST /api/clear-pinned

**Response:**
```json
{
  "success": true,
  "message": "Pinned markers cleared"
}
```

Removes all `<pinned>` markers from `### HH:MM <pinned>` headers in today's note.

### POST /api/files/unpin

**Form data:**
- `path` (required) - Relative file path
- `line` (required) - Line number of pinned header

**Response:**
```json
{
  "success": true,
  "message": "Entry unpinned"
}
```

Removes `<pinned>` marker from a specific entry header.

## Platform Differences

### Web (FastAPI + HTMX)

**View mode:**
- Rendered HTML with styled headings and task lines
- Interactive checkboxes trigger `/api/todos/toggle` via HTMX
- Pinned entries show "Unpin" button
- "Work task" / "Priv task" buttons add empty tasks and open edit mode

**Edit mode:**
- Full textarea with raw markdown
- Save/Cancel buttons
- Auto-opens after adding a task

**Append section:**
- Textarea for new entry content
- Pin checkbox
- "Add" button appends timestamped entry
- "Clear" button removes all pinned markers

### Android (Jetpack Compose)

**View mode:**
- `NoteView` composable renders content
- Tap checkbox triggers toggle API call
- "Reload" button refreshes content

**Edit mode:**
- `CompactTextField` for editing raw content
- "Save" / "Cancel" buttons
- Back button exits edit mode

**Task buttons:**
- "Work task" / "Priv task" add task and enter edit mode

**Append section:**
- Text field for new entry
- Pin checkbox (clickable row with label)
- "Add" button appends entry
- "Clear" button (danger styled) clears pinned markers

**State management:**
- Uses Compose `remember` and `mutableStateOf`
- `LaunchedEffect` loads on screen open
- `BackHandler` intercepts back button in edit mode

## Git Integration

All operations commit and push to git:

| Operation | Commit Message |
|-----------|----------------|
| Create daily note | "Create daily note YYYY-MM-DD" |
| Save note | "Update note" |
| Append entry | "Append note at HH:MM" or "Append pinned note at HH:MM" |
| Add task | "Add work task" or "Add priv task" |
| Toggle task | "Toggle todo" |
| Clear pinned | "Clear pinned markers" |
| Unpin entry | "Unpin entry" |

## Key Implementation Files

| File | Purpose |
|------|---------|
| `server/web_app/main.py` | API endpoints, carryover logic, file operations |
| `server/web_app/templates/editor.html` | Web UI template with HTMX interactions |
| `server/web_app/renderers/pinned.py` | HTML rendering with checkbox/unpin controls |
| `app/android/.../DailyScreen.kt` | Android Compose UI |
| `app/android/.../ApiClient.kt` | Android API client |
