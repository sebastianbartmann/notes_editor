# Daily Note Format Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-18

## Overview

This document defines the structure and format of daily notes in the Notes Editor application. Daily notes are markdown files that serve as the primary organizational unit for daily tasks and timestamped entries. The format supports task management with categories, timestamped notes, and a pinned notes system for content that carries forward across days.

---

## File Structure

### File Location
```
{VAULT_ROOT}/{person}/daily/YYYY-MM-DD.md
```

**Examples:**
- `~/notes/sebastian/daily/2026-01-18.md`
- `~/notes/petra/daily/2026-01-15.md`

### File Naming
- Format: `YYYY-MM-DD.md`
- Uses ISO 8601 date format
- Files are sorted lexicographically for chronological ordering

### Document Structure
```markdown
# daily YYYY-MM-DD

## todos

### work
- [ ] task 1
- [x] completed task

### priv
- [ ] personal task

## custom notes

### HH:MM
Note content here

### HH:MM <pinned>
This note carries forward to next day
```

---

## Section Definitions

### Header
**Pattern:** `^# daily \d{4}-\d{2}-\d{2}$`

The document header identifies the file as a daily note and specifies its date. Must be the first non-empty line.

**Example:**
```markdown
# daily 2026-01-18
```

### Todos Section
**Pattern:** `^##\s+todos\s*$` (case-insensitive)

Contains categorized task lists. Supports two built-in categories but allows additional custom categories.

**Built-in Categories:**
| Category | Header | Purpose |
|----------|--------|---------|
| work | `### work` | Work-related tasks |
| priv | `### priv` | Personal/private tasks |

### Custom Notes Section
**Pattern:** `^##\s+custom notes\s*$` (case-insensitive)

Contains timestamped entries. Each entry has a time-based header and may optionally be marked as pinned.

---

## Task Format

### Task Line Pattern
```regex
^\s*-\s*\[([ xX])\]\s*(.*)$
```

| Component | Description |
|-----------|-------------|
| `^\s*` | Optional leading whitespace |
| `-\s*` | Dash followed by optional space |
| `\[([ xX])\]` | Checkbox: space = unchecked, x/X = checked |
| `\s*(.*)$` | Optional space and task description |

### Task States
| Syntax | State |
|--------|-------|
| `- [ ]` | Incomplete/unchecked |
| `- [x]` | Complete/checked |
| `- [X]` | Complete/checked (alternate) |

### Examples
```markdown
### work
- [ ] Review pull request
- [x] Deploy to staging
- [ ] Update documentation

### priv
- [ ] Call dentist
- [x] Buy groceries
```

### Adding Tasks
New tasks are added as empty checkboxes:
```markdown
- [ ]
```

The user fills in the description after creation.

---

## Pinned Notes

### Pinned Header Pattern
```regex
^(###\s+.*<pinned>.*)$
```
(case-insensitive for `<pinned>`)

### Time Header Patterns
| Type | Pattern | Example |
|------|---------|---------|
| Regular | `### HH:MM` | `### 14:30` |
| Pinned | `### HH:MM <pinned>` | `### 14:30 <pinned>` |

### Purpose
Pinned notes automatically carry forward to the next day's note when a new daily note is created. This is useful for:
- Ongoing reminders
- Multi-day tasks
- Notes that need continued attention

### Clear Pinned Operation
Removes `<pinned>` markers from all entries in the current note.

**Pattern Replacement:**
```regex
^(### \d{2}:\d{2})\s*<pinned>
```
Replaced with: `$1`

### Unpin Single Entry
Removes `<pinned>` from a specific line by replacing the substring.

---

## Inheritance Logic

When creating a new daily note, content is inherited from the most recent previous note.

### Finding Previous Note
1. List all files in the `daily/` directory
2. Sort filenames in descending order (newest first)
3. Skip the current date
4. Select the first (most recent) file

### Inherited Content

#### Incomplete Todos
**Extraction Process:**
1. Find `## todos` section
2. Extract all lines until next `## ` section or end of file
3. Filter out lines matching `^\s*-\s*\[x\]` (completed tasks)
4. Include remaining lines (unchecked tasks + category headers)

**Example:**
Previous note:
```markdown
## todos

### work
- [x] Deploy
- [ ] Review PR

### priv
- [ ] Call dentist
```

Inherited content:
```markdown
## todos

### work
- [ ] Review PR

### priv
- [ ] Call dentist
```

#### Pinned Notes
**Extraction Process:**
1. Find `## custom notes` section
2. Scan for `### ` entries containing `<pinned>` (case-insensitive)
3. Collect entire entry (header + body) until next `### ` or section end
4. Join all pinned entries with double newlines

**Example:**
Previous note:
```markdown
## custom notes

### 10:30
Regular note

### 14:00 <pinned>
Important reminder that
spans multiple lines

### 16:00 <pinned>
Another pinned item
```

Inherited content:
```markdown
### 14:00 <pinned>
Important reminder that
spans multiple lines

### 16:00 <pinned>
Another pinned item
```

---

## Parsing Rules

### Section Extraction
**Pattern:** `^##\s+{section_name}\s*$` (case-insensitive)

**Algorithm:**
1. Search for line matching section pattern
2. Start capturing from the line after the section header
3. Stop capturing when reaching:
   - Next `## ` section header
   - End of file
4. Return stripped content

### Heading Detection
**Pattern:** `^(#{1,6})\s+(.*)$`

Matches all markdown heading levels (h1-h6).

### Line Number References
All line operations use 1-indexed line numbers for consistency with user-facing editors.

---

## Integration Notes

### Git Operations
- **Create:** New daily note triggers `git commit -m "Create daily note YYYY-MM-DD"`
- **Save:** Full save triggers `git commit` and `git push`
- **Append:** Adding entry triggers `git commit` and `git push`

### API Endpoints
| Operation | Endpoint |
|-----------|----------|
| Get/Create daily | `GET /api/daily` |
| Save entire note | `POST /api/save` |
| Append timestamped entry | `POST /api/append` |
| Add task | `POST /api/todos/add` |
| Toggle task | `POST /api/todos/toggle` |
| Clear all pinned | `POST /api/clear-pinned` |
| Unpin single entry | `POST /api/files/unpin` |

### Section Creation
If a required section doesn't exist during an operation:
- `## todos` is created when adding a task
- `## custom notes` is created when appending an entry
- Category headers (`### work`, `### priv`) are created as needed

---

## Examples

### Complete Daily Note
```markdown
# daily 2026-01-18

## todos

### work
- [ ] Review PR #123
- [x] Deploy to staging
- [ ] Write documentation

### priv
- [ ] Schedule dentist appointment
- [x] Pick up dry cleaning

## custom notes

### 09:15
Morning standup notes:
- Discussed sprint goals
- Blocker on API integration

### 11:30 <pinned>
IMPORTANT: Server maintenance scheduled for Sunday 02:00

### 14:45
Meeting with design team about new UI

### 16:00 <pinned>
Remember to follow up with client about contract
```

### Minimal Daily Note
```markdown
# daily 2026-01-18

## todos

### work

### priv

## custom notes
```

### Newly Created Note (with inheritance)
```markdown
# daily 2026-01-19

## todos

### work
- [ ] Review PR #123
- [ ] Write documentation

### priv
- [ ] Schedule dentist appointment

## custom notes

### 11:30 <pinned>
IMPORTANT: Server maintenance scheduled for Sunday 02:00

### 16:00 <pinned>
Remember to follow up with client about contract
```
