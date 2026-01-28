# Sleep Tracking Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-18

## Overview

This document defines the sleep tracking feature in the Notes Editor application. Sleep tracking allows users to log sleep-related events for children, tracking when they fall asleep and wake up. The data is stored in a shared markdown file and supports both Android and web interfaces.

---

## Data Format

### File Location
```
{VAULT_ROOT}/sleep_times.md
```

**Note:** Unlike most other data in the application, sleep times are stored in a shared file at the vault root, not scoped to individual persons.

### File Structure
```markdown
# Sleep Times

Log entries in the format:
- YYYY-MM-DD | Name | 19:30-06:10 | night

2026-01-18 | Thomas | 19:30 | eingeschlafen
2026-01-18 | Fabian | 06:15 | aufgewacht
2026-01-17 | Thomas | 06:10 | aufgewacht
```

### File Creation
When the file doesn't exist, it is created with this header:
```markdown
# Sleep Times

Log entries in the format:
- YYYY-MM-DD | Name | 19:30-06:10 | night

```

### Entry Format
```
YYYY-MM-DD | ChildName | TimeEntry | Status
```

| Field | Description | Example |
|-------|-------------|---------|
| Date | ISO 8601 date, auto-generated from current date | `2026-01-18` |
| ChildName | Name of the child | `Thomas`, `Fabian` |
| TimeEntry | User-provided time string | `19:30`, `06:15`, `19:30-06:10` |
| Status | Sleep event type (German) | `eingeschlafen`, `aufgewacht` |

### Status Values
| Value | Meaning | English |
|-------|---------|---------|
| `eingeschlafen` | Child fell asleep | Fell asleep |
| `aufgewacht` | Child woke up | Woke up |

### Entry Parsing
When reading entries:
- Skip lines starting with `#` (headers)
- Skip lines starting with `- ` (documentation bullet points)
- All other non-empty lines are treated as entries

---

## Business Rules

### Child Selection
- Available children: `Thomas`, `Fabian`
- Selection is radio-button style (mutually exclusive)
- Default selection: `Fabian`

### Status Selection
- Asleep and Woke checkboxes are mutually exclusive
- Selecting one automatically unchecks the other
- At least one must be selected when submitting

### Entry Display
- Recent entries: last 20 entries shown
- Order: reverse chronological (newest first)
- Entries are identified by 1-indexed line numbers for deletion

### Date Handling
- Entry date is automatically set to the current date at submission time
- Date is formatted as `YYYY-MM-DD`

---

## UI Components

### Android (SleepTimesScreen.kt)

**Layout:**
- Child selection: Radio buttons for Thomas/Fabian
- Status: Two checkboxes (Asleep/Woke) with mutual exclusion
- Time entry: Text input field
- Submit button
- Recent entries list with delete buttons

**Behavior:**
- Selecting a child radio button deselects the other
- Selecting a status checkbox deselects the other
- Submit appends entry and refreshes list
- Delete button removes entry by line number

### Web (sleep_times.html)

**Layout:**
- Child selection: Radio input group
- Status: Checkbox inputs with JavaScript mutual exclusion
- Time entry: Text input
- Form submission via HTMX

**Behavior:**
- HTMX form posts to `/api/sleep-times/append`
- On success: auto-reloads entries via `hx-swap-oob`
- Delete buttons trigger `/api/sleep-times/delete`

---

## API Integration

See [01-rest-api-contract.md](./01-rest-api-contract.md) for full API documentation.

### Endpoints Summary

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/sleep-times` | List recent entries (last 20) |
| POST | `/api/sleep-times/append` | Add new sleep entry |
| POST | `/api/sleep-times/delete` | Delete entry by line number |

### Request/Response Examples

**GET /api/sleep-times**
```json
{
  "entries": [
    {
      "line": 15,
      "date": "2026-01-18",
      "child": "Thomas",
      "time": "19:30",
      "status": "eingeschlafen"
    },
    {
      "line": 14,
      "date": "2026-01-17",
      "child": "Thomas",
      "time": "06:10",
      "status": "aufgewacht"
    }
  ]
}
```

**POST /api/sleep-times/append**
```json
{
  "child": "Thomas",
  "time": "19:30",
  "status": "eingeschlafen"
}
```

**POST /api/sleep-times/delete**
```json
{
  "line": 15
}
```

---

## Git Integration

### Operations and Git Behavior

| Operation | Git Action |
|-----------|------------|
| File creation | Commit only |
| Append entry | Commit and push |
| Delete entry | Commit and push |

### Commit Messages
- File creation: Automatic commit with descriptive message
- Append/Delete: Commits changes with operation-specific message

---

## Known Limitations

1. **No Person Scoping**: Sleep times are stored in a shared file, not separated by person. All users see the same data.

2. **No Time Validation**: The time entry field accepts free-form text without validation (e.g., accepts "19:30", "around 7pm", etc.).

3. **Line Number Fragility**: Deletion uses line numbers which can become stale if multiple users modify the file concurrently.

4. **German-Only Status**: Status values are hardcoded in German (`eingeschlafen`, `aufgewacht`).

5. **Fixed Child List**: Children (Thomas, Fabian) are hardcoded; adding new children requires code changes.

6. **No Entry Editing**: Entries can only be added or deleted, not modified.

7. **No Date Override**: Users cannot manually set a date for entries (always uses current date).
