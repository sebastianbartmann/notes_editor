# Sleep Tracking

## Purpose

Sleep Tracking is a shared family tool for logging children's sleep events. It provides a simple interface to record when children (Thomas and Fabian) fall asleep or wake up, with timestamps and optional notes. The log is shared across all users (not person-scoped) and auto-commits changes to git.

## Data Storage

### File Location

- **Path:** `~/notes/sleep_times.md` (at vault root, not inside a person folder)
- **Scope:** Shared across all users (Sebastian and Petra can both view and edit)

### File Format

The file is a simple markdown document with a header section followed by log entries:

```markdown
# Sleep Times

Log entries in the format:
- YYYY-MM-DD | Name | 19:30-06:10 | night

2025-01-15 | Fabian | 19:45 | eingeschlafen
2025-01-16 | Fabian | 06:30 | aufgewacht
2025-01-16 | Thomas | 20:00-06:45 | night
```

### Entry Format

Each entry follows the pattern:

```
YYYY-MM-DD | Name | time-or-range | status
```

| Field | Description |
|-------|-------------|
| Date | Auto-generated from current date (`YYYY-MM-DD`) |
| Name | Child name: `Thomas` or `Fabian` (auto-capitalized) |
| Time/Range | Free-form text, typically a time (`19:30`) or range (`19:30-06:10`) |
| Status | Optional: `eingeschlafen` (fell asleep) or `aufgewacht` (woke up) |

## Features

### Log Entry

- Select child (Thomas or Fabian) via radio buttons
- Enter time/range in free-form text field
- Optionally mark as "eingeschlafen" (fell asleep) or "aufgewacht" (woke up)
- Status checkboxes are mutually exclusive (selecting one clears the other)
- Date is auto-generated from server time

### View Recent Entries

- Displays the 20 most recent entries in reverse chronological order
- Each entry shows the full line text
- Entries can be deleted individually by line number

### Delete Entry

- Removes entry by its 1-based line number in the file
- Preserves file structure and trailing newlines
- Auto-commits deletion to git

### Auto-Initialization

If `sleep_times.md` does not exist, the system creates it with a header template on first access.

## API Endpoints

### GET /sleep-times

Returns the full HTML page for sleep tracking (web UI).

**Response:** HTML page with entry form and recent entries list.

### GET /api/sleep-times

Returns recent sleep entries as JSON.

**Response:**
```json
{
  "entries": [
    {"line_no": 15, "text": "2025-01-16 | Fabian | 06:30 | aufgewacht"},
    {"line_no": 14, "text": "2025-01-15 | Fabian | 19:45 | eingeschlafen"}
  ]
}
```

### POST /api/sleep-times/append

Adds a new sleep entry.

**Form Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `child` | Yes | Child name (Thomas or Fabian) |
| `entry` | Yes | Time or time range text |
| `asleep` | No | Set to `on` to append "eingeschlafen" status |
| `woke` | No | Set to `on` to append "aufgewacht" status |

**Response:**
```json
{
  "success": true,
  "message": "Entry added"
}
```

### POST /api/sleep-times/delete

Deletes an entry by line number.

**Form Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `line` | Yes | 1-based line number to delete |

**Response:**
```json
{
  "success": true,
  "message": "Entry deleted"
}
```

## Git Integration

All modifications trigger automatic git commit and push:
- File creation: "Create sleep times log"
- Entry append: "Append sleep times"
- Entry delete: "Delete sleep times entry"

A `git_pull()` is performed before reading the file to ensure the latest data.

## Platform Implementation

### Web (HTMX)

**Template:** `server/web_app/templates/sleep_times.html`

- Uses HTMX for form submissions (`hx-post`)
- Child selection via radio buttons (Fabian default)
- Status selection via mutually exclusive checkboxes (JS enforced)
- Success messages displayed briefly, then page auto-reloads
- Delete buttons inline with each entry

### Android (Jetpack Compose)

**Screen:** `SleepTimesScreen.kt`

- Uses `ApiClient` for all API calls
- Child selection via checkboxes with radio-button behavior
- Status checkboxes with mutual exclusion logic
- Entries displayed in a scrollable list
- Delete button for each entry
- Status messages shown at bottom of form
- Manual reload button in header

### Data Models (Android)

```kotlin
data class SleepTimesResponse(
    val entries: List<SleepEntry>
)

data class SleepEntry(
    val lineNo: Int,  // @SerialName("line_no")
    val text: String
)
```

## Key Files

| Component | Path |
|-----------|------|
| Server endpoints | `server/web_app/main.py` (lines 311-342, 737-822) |
| Vault store | `server/web_app/services/vault_store.py` |
| Web template | `server/web_app/templates/sleep_times.html` |
| Android screen | `app/android/.../SleepTimesScreen.kt` |
| Android API client | `app/android/.../ApiClient.kt` |
| Android models | `app/android/.../Models.kt` |
