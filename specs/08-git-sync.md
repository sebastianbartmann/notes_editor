# Git Sync

## Purpose

Git Sync provides automatic version control for the Notes Editor vault, ensuring all changes are tracked and synchronized across devices. It handles git operations transparently during normal app usage, with pull operations before reads and commit/push operations after writes.

## Vault Repository

**Location:** `~/notes/` (the vault root directory)

The vault is a git repository containing all user data:
- Daily notes per person (`{person}/daily/*.md`)
- Personal files per person (`{person}/**/*`)
- Shared files (`sleep_times.md`)

## Core Operations

### Pull (`git_pull`)

Fetches and merges remote changes before read operations.

**Trigger points:**
- Page loads (daily notes, file browser, sleep times)
- API read operations
- Claude tool file operations

**Process:**
1. Check if vault is a git repository (skip if not)
2. Abort any stuck rebase or merge operations
3. Execute `git pull --no-rebase -X theirs`
4. If pull fails, fallback to hard reset

**Stuck operation cleanup:**
```python
# Abort stuck rebase
if (GIT_DIR / ".git" / "rebase-merge").exists():
    subprocess.run(["git", "rebase", "--abort"], cwd=GIT_DIR)
if (GIT_DIR / ".git" / "rebase-apply").exists():
    subprocess.run(["git", "rebase", "--abort"], cwd=GIT_DIR)

# Abort stuck merge
if (GIT_DIR / ".git" / "MERGE_HEAD").exists():
    subprocess.run(["git", "merge", "--abort"], cwd=GIT_DIR)
```

### Commit and Push (`git_commit_and_push`)

Stages, commits, and pushes changes after write operations.

**Trigger points:**
- Saving daily notes or files
- Appending notes or tasks
- Toggling tasks
- Creating or deleting files
- Sleep tracking entries

**Process:**
1. Check if vault is a git repository (skip if not)
2. Check for uncommitted changes via `git status --porcelain`
3. Skip if no changes detected
4. Stage all changes with `git add .`
5. Commit with provided message
6. Push to remote
7. If push fails, pull and retry push once

## Conflict Resolution

Git Sync uses an aggressive "theirs wins" strategy to prevent conflicts from blocking normal operation.

### Pull Strategy

```bash
git pull --no-rebase -X theirs
```

- `--no-rebase`: Uses merge instead of rebase (simpler conflict handling)
- `-X theirs`: On conflicts, automatically accept remote version

### Fallback: Hard Reset

If pull fails even with the "theirs" strategy:

```python
# Fetch latest remote state
subprocess.run(["git", "fetch", "origin"], cwd=GIT_DIR)

# Get current branch name
branch = subprocess.run(
    ["git", "rev-parse", "--abbrev-ref", "HEAD"],
    cwd=GIT_DIR, capture_output=True, text=True
).stdout.strip()

# Reset to remote branch
subprocess.run(["git", "reset", "--hard", f"origin/{branch}"], cwd=GIT_DIR)
```

This ensures the local repository always recovers to a working state, potentially discarding local-only changes in favor of the remote version.

## Return Value Pattern

Both functions return `tuple[bool, str]` for success status and message:

```python
def git_pull() -> tuple[bool, str]: ...
def git_commit_and_push(message: str = "Update notes") -> tuple[bool, str]: ...
```

### Success Cases

| Scenario | Return |
|----------|--------|
| No git repository | `(True, "No git repository")` |
| Pull successful | `(True, "Pulled latest changes")` |
| Recovery via reset | `(True, "Recovered by resetting to remote")` |
| No changes to commit | `(True, "No changes to commit")` |
| Commit and push successful | `(True, "Changes committed and pushed")` |
| Push retry successful | `(True, "Changes committed and pushed (after sync)")` |
| Commit ok, push failed | `(True, "Changes committed locally (push failed, will retry on next sync)")` |

### Failure Cases

| Scenario | Return |
|----------|--------|
| Pull failed, reset failed | `(False, "Git pull failed")` |
| Exception during pull | `(False, "Error: {exception}")` |

**Note:** `git_commit_and_push` returns success (`True`) even when push fails, as the data is safely committed locally.

## Commit Message Conventions

Operations use descriptive commit messages indicating the action performed:

| Operation | Commit Message |
|-----------|----------------|
| Create daily note | `"Create daily note YYYY-MM-DD"` |
| Create sleep log | `"Create sleep times log"` |
| Save note | `"Update note"` |
| Append note | `"Append note at HH:MM"` |
| Append pinned note | `"Append pinned note at HH:MM"` |
| Add work task | `"Add work task"` |
| Add priv task | `"Add priv task"` |
| Toggle task | `"Toggle todo"` |
| Clear pinned | `"Clear pinned markers"` |
| Unpin entry | `"Unpin entry"` |
| Create file | `"Create file"` |
| Update file | `"Update file"` |
| Delete file | `"Delete file"` |
| Append sleep times | `"Append sleep times"` |
| Delete sleep entry | `"Delete sleep times entry"` |
| Claude WebFetch | `"Log WebFetch request"` |

## Error Handling

### Graceful Degradation

The sync mechanism is designed to never block normal operation:

1. **No repository:** Operations succeed with informational message
2. **Pull failures:** Automatic recovery via hard reset
3. **Push failures:** Changes saved locally, retry on next sync
4. **Exceptions:** Caught and returned as messages, never raised

### Exception Handling

```python
try:
    # git operations
except subprocess.CalledProcessError as exc:
    error_msg = exc.stderr if exc.stderr else str(exc)
    return True, f"Changes saved locally: {error_msg}"
except Exception as exc:
    return True, f"Changes saved locally: {str(exc)}"
```

## Usage Examples

### Before Read Operations

```python
from .services.git_sync import git_pull

@app.get("/")
async def index():
    git_pull()  # Sync before reading
    # ... load and display daily note
```

### After Write Operations

```python
from .services.git_sync import git_commit_and_push

@app.post("/api/save")
async def save_note(content: str):
    # ... write file
    success, msg = git_commit_and_push("Update note")
    return {"success": success, "message": msg}
```

### With Custom Commit Messages

```python
# Dynamic message with context
git_commit_and_push(f"Create daily note {date_str}")
git_commit_and_push(f"Append {'pinned ' if is_pinned else ''}note at {time_str}")
git_commit_and_push(f"Add {category} task")
```

## Key Files

| File | Purpose |
|------|---------|
| `server/web_app/services/git_sync.py` | Core sync functions |
| `server/web_app/main.py` | Usage in API endpoints |
| `server/web_app/services/claude_service.py` | Usage in Claude tool operations |
