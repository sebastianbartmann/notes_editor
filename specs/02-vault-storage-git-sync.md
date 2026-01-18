# Vault Storage and Git Sync Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-18

## Overview

This document specifies the storage and synchronization layer for the Notes Editor application. The system provides:

1. **Vault Storage** (`vault_store.py`): File operations within a sandboxed directory structure
2. **Git Sync** (`git_sync.py`): Version control and remote synchronization

All notes are stored as plain markdown files on the filesystem, organized by person and managed through Git for cross-device synchronization.

---

## Configuration

### Vault Root

The vault root directory is the base path for all file operations:

```python
VAULT_ROOT = Path.home() / "notes"  # e.g., /home/user/notes
```

### Git Directory

Git operations use the same directory as the vault:

```python
GIT_DIR = Path.home() / "notes"
```

### Multi-User Directory Structure

The vault supports multiple persons, each with their own subdirectory:

```
~/notes/
├── .git/
├── sebastian/
│   ├── daily/
│   │   └── 2026-01-18.md
│   └── sleep_times.md
└── petra/
    ├── daily/
    │   └── 2026-01-18.md
    └── sleep_times.md
```

---

## Vault Store Module

The `vault_store` module provides sandboxed file operations. All paths are relative to `VAULT_ROOT` and validated to prevent directory traversal attacks.

### Path Resolution

#### `resolve_path(relative_path: str) -> Path`

Resolves a vault-relative path to an absolute filesystem path with security validation.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `relative_path` | `str` | Path relative to vault root (e.g., `"sebastian/daily/2026-01-18.md"`) |

**Returns:** `Path` - Absolute filesystem path

**Raises:**
| Exception | Condition |
|-----------|-----------|
| `ValueError` | Path is empty |
| `ValueError` | Path is absolute (starts with `/`) |
| `ValueError` | Path escapes vault root via `..` traversal |

**Behavior:**
1. Rejects empty paths
2. Rejects absolute paths
3. Joins path with `VAULT_ROOT`
4. Resolves to canonical absolute path (eliminates `..` and symlinks)
5. Validates resolved path is within vault root

**Example:**
```python
# Valid paths
resolve_path("sebastian/daily/note.md")  # -> /home/user/notes/sebastian/daily/note.md
resolve_path("petra/notes.md")           # -> /home/user/notes/petra/notes.md

# Invalid paths (raises ValueError)
resolve_path("")                         # Empty path
resolve_path("/etc/passwd")              # Absolute path
resolve_path("../../../etc/passwd")      # Escapes vault root
resolve_path("sebastian/../../root")     # Escapes vault root
```

---

### File Operations

#### `read_entry(relative_path: str) -> str`

Reads the content of a file.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `relative_path` | `str` | Vault-relative path to the file |

**Returns:** `str` - File content as UTF-8 text

**Raises:**
| Exception | Condition |
|-----------|-----------|
| `ValueError` | Invalid path (see `resolve_path`) |
| `FileNotFoundError` | File does not exist |
| `IsADirectoryError` | Path is a directory |
| `UnicodeDecodeError` | File is not valid UTF-8 |

---

#### `write_entry(relative_path: str, content: str) -> None`

Writes content to a file, creating parent directories if needed.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `relative_path` | `str` | Vault-relative path to the file |
| `content` | `str` | Content to write |

**Returns:** `None`

**Raises:**
| Exception | Condition |
|-----------|-----------|
| `ValueError` | Invalid path (see `resolve_path`) |
| `IsADirectoryError` | Path is an existing directory |
| `PermissionError` | Insufficient filesystem permissions |

**Behavior:**
1. Resolves and validates path
2. Creates parent directories recursively (`mkdir -p` equivalent)
3. Overwrites file with new content (atomic write via `Path.write_text`)

---

#### `append_entry(relative_path: str, content: str) -> None`

Appends content to a file, creating the file and parent directories if needed.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `relative_path` | `str` | Vault-relative path to the file |
| `content` | `str` | Content to append |

**Returns:** `None`

**Raises:**
| Exception | Condition |
|-----------|-----------|
| `ValueError` | Invalid path (see `resolve_path`) |
| `IsADirectoryError` | Path is an existing directory |
| `PermissionError` | Insufficient filesystem permissions |

**Behavior:**
1. Resolves and validates path
2. Creates parent directories recursively
3. Opens file in append mode
4. Writes content at end of file

**Note:** Does not add newline automatically; caller must include newlines in content.

---

#### `delete_entry(relative_path: str) -> None`

Deletes a file.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `relative_path` | `str` | Vault-relative path to the file |

**Returns:** `None`

**Raises:**
| Exception | Condition |
|-----------|-----------|
| `ValueError` | Invalid path (see `resolve_path`) |
| `IsADirectoryError` | Path is a directory (directories cannot be deleted) |

**Behavior:**
1. Resolves and validates path
2. If file does not exist, returns silently (idempotent)
3. If path is a directory, raises `IsADirectoryError`
4. Deletes the file

---

### Directory Operations

#### `list_dir(relative_path: str) -> list[dict]`

Lists contents of a directory.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `relative_path` | `str` | Vault-relative path to the directory |

**Returns:** `list[dict]` - List of entry dictionaries

**Entry Dictionary:**
```python
{
    "name": str,     # Entry name (e.g., "note.md")
    "path": str,     # Vault-relative path (e.g., "sebastian/daily/note.md")
    "is_dir": bool   # True if directory, False if file
}
```

**Raises:**
| Exception | Condition |
|-----------|-----------|
| `ValueError` | Invalid path (see `resolve_path`) |
| `FileNotFoundError` | Directory does not exist |
| `NotADirectoryError` | Path is a file, not a directory |

**Behavior:**
1. Resolves and validates path
2. Iterates directory contents
3. Excludes hidden files (names starting with `.`)
4. Sorts entries: files first, then directories, alphabetically (case-insensitive)

**Example Response:**
```python
[
    {"name": "notes.md", "path": "sebastian/notes.md", "is_dir": False},
    {"name": "todo.md", "path": "sebastian/todo.md", "is_dir": False},
    {"name": "daily", "path": "sebastian/daily", "is_dir": True},
]
```

---

## Git Sync Module

The `git_sync` module provides Git operations for synchronizing the vault with a remote repository.

### Pull Operation

#### `git_pull() -> tuple[bool, str]`

Pulls latest changes from the remote repository.

**Parameters:** None

**Returns:** `tuple[bool, str]`
- `bool`: Success status
- `str`: Status message or error description

**Behavior:**

1. **Repository Check:**
   - Verifies `.git` directory exists in `GIT_DIR`
   - Returns `(False, "Not a git repository")` if missing

2. **Abort Existing Operations:**
   - Aborts any in-progress rebase: `git rebase --abort`
   - Aborts any in-progress merge: `git merge --abort`
   - Failures are ignored (operations may not be in progress)

3. **Pull with Merge Strategy:**
   - Executes: `git pull --no-rebase -X theirs`
   - `--no-rebase`: Uses merge instead of rebase
   - `-X theirs`: On conflicts, accepts remote version

4. **Fallback on Failure:**
   - If pull fails, attempts fetch and hard reset:
     ```
     git fetch origin
     git reset --hard origin/<current_branch>
     ```
   - This discards local changes in favor of remote

**Return Values:**
| Condition | Return |
|-----------|--------|
| Success | `(True, "Pull successful")` |
| No remote changes | `(True, "Already up to date")` |
| Fallback succeeded | `(True, "Reset to remote successful")` |
| Not a git repo | `(False, "Not a git repository")` |
| Network error | `(False, "<error message>")` |

---

### Commit and Push Operation

#### `git_commit_and_push(message: str = "Update notes") -> tuple[bool, str]`

Stages all changes, commits, and pushes to remote.

**Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `message` | `str` | `"Update notes"` | Commit message |

**Returns:** `tuple[bool, str]`
- `bool`: Success status
- `str`: Status message or error description

**Behavior:**

1. **Check for Changes:**
   - Executes: `git status --porcelain`
   - Returns early if no changes: `(True, "No changes to commit")`

2. **Stage Changes:**
   - Executes: `git add .`
   - Stages all new, modified, and deleted files

3. **Commit:**
   - Executes: `git commit -m "<message>"`
   - Uses provided or default message

4. **Push:**
   - Executes: `git push`
   - On failure, pulls and retries push once

5. **Graceful Degradation:**
   - If push ultimately fails, changes remain committed locally
   - Returns `(False, "<error>")` but data is preserved

**Return Values:**
| Condition | Return |
|-----------|--------|
| Success | `(True, "Changes pushed successfully")` |
| No changes | `(True, "No changes to commit")` |
| Commit succeeded, push failed | `(False, "Push failed: <error>")` |
| Commit failed | `(False, "Commit failed: <error>")` |

---

## Security Considerations

### Path Traversal Prevention

The `resolve_path` function implements multiple layers of protection:

1. **Empty Path Rejection:** Prevents operations on vault root itself
2. **Absolute Path Rejection:** Blocks paths starting with `/`
3. **Canonical Resolution:** Uses `Path.resolve()` to eliminate `..` sequences
4. **Containment Validation:** Verifies resolved path is within vault root

**Attack Vectors Mitigated:**
- `../../../etc/passwd` - Blocked by containment check
- `/etc/passwd` - Blocked by absolute path check
- `foo/../../etc/passwd` - Blocked by containment check after resolution

### Hidden File Filtering

The `list_dir` function excludes files starting with `.`:
- Protects `.git` directory from exposure
- Hides system files (`.DS_Store`, etc.)
- Prevents accidental exposure of `.env` or similar

### Git Conflict Resolution

The sync strategy prioritizes data preservation:
- Uses `-X theirs` to accept remote on conflicts (assumes remote is authoritative)
- Fallback to hard reset ensures sync completes but may lose local changes
- Local commits are preserved even if push fails

---

## Integration Notes

### API Layer Integration

The REST API (`main.py`) uses these modules as follows:

```python
# Path prefixing with person context
def get_person_path(person: str, relative_path: str) -> str:
    return f"{person}/{relative_path}"

# Reading a file
@app.get("/api/files/read")
async def read_file(path: str, person: str):
    full_path = get_person_path(person, path)
    content = vault_store.read_entry(full_path)
    return {"path": path, "content": content}

# Writing with git sync
@app.post("/api/save")
async def save_note(content: str, person: str):
    path = get_person_path(person, get_today_path())
    vault_store.write_entry(path, content)
    success, msg = git_sync.git_commit_and_push("Update daily note")
    return {"success": success, "message": msg}
```

### Typical Operation Sequence

**Reading data:**
```
1. git_pull()           # Sync from remote
2. read_entry(path)     # Read file content
3. Return to client
```

**Writing data:**
```
1. write_entry(path, content)       # Write to filesystem
2. git_commit_and_push(message)     # Sync to remote
3. Return success/failure to client
```

### Error Handling Pattern

```python
try:
    content = vault_store.read_entry(path)
except FileNotFoundError:
    raise HTTPException(404, "File not found")
except IsADirectoryError:
    raise HTTPException(400, "Path is a directory")
except ValueError as e:
    raise HTTPException(400, str(e))
```

### Concurrent Access

The current implementation does not handle concurrent access:
- Multiple simultaneous writes may cause race conditions
- Git operations are not locked
- For single-user or low-concurrency scenarios only

---

## Limitations

1. **No Directory Deletion:** `delete_entry` only handles files
2. **No File Renaming:** Must delete and recreate
3. **UTF-8 Only:** Binary files not supported
4. **No Locking:** Concurrent access may cause issues
5. **Remote-Wins Conflict Resolution:** Local changes may be lost on conflicts
