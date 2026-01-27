# Vault Storage and Git Sync Specification

> Status: Draft
> Version: 2.0
> Last Updated: 2026-01-27

## Overview

This document specifies the storage and synchronization layer for the Notes Editor application. The system provides:

1. **Vault Storage** (`internal/vault/store.go`): File operations within a sandboxed directory structure
2. **Git Sync** (`internal/vault/git.go`): Version control and remote synchronization

All notes are stored as plain markdown files on the filesystem, organized by person and managed through Git for cross-device synchronization.

---

## Configuration

### Vault Root

The vault root directory is the base path for all file operations:

```go
// From environment variable NOTES_ROOT
vaultRoot := os.Getenv("NOTES_ROOT")  // e.g., /home/user/notes
if vaultRoot == "" {
    vaultRoot = filepath.Join(os.Getenv("HOME"), "notes")
}
```

### Git Directory

Git operations use the same directory as the vault:

```go
gitDir := vaultRoot  // /home/user/notes
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

## Vault Store Package

The `vault` package provides sandboxed file operations. All paths are relative to `vaultRoot` and validated to prevent directory traversal attacks.

### Store Type

```go
type Store struct {
    rootPath string
}

func NewStore(rootPath string) *Store {
    return &Store{rootPath: rootPath}
}
```

### Path Resolution

#### `ResolvePath(relativePath string) (string, error)`

Resolves a vault-relative path to an absolute filesystem path with security validation.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `relativePath` | `string` | Path relative to vault root (e.g., `"sebastian/daily/2026-01-18.md"`) |

**Returns:** `string` - Absolute filesystem path, `error` - validation error if any

**Errors:**
| Error | Condition |
|-------|-----------|
| `ErrEmptyPath` | Path is empty |
| `ErrAbsolutePath` | Path is absolute (starts with `/`) |
| `ErrPathEscape` | Path escapes vault root via `..` traversal |

**Behavior:**
1. Rejects empty paths
2. Rejects absolute paths
3. Joins path with vault root
4. Cleans path (eliminates `..` and redundant separators)
5. Validates resolved path is within vault root

**Example:**
```go
// Valid paths
path, _ := store.ResolvePath("sebastian/daily/note.md")  // -> /home/user/notes/sebastian/daily/note.md
path, _ := store.ResolvePath("petra/notes.md")           // -> /home/user/notes/petra/notes.md

// Invalid paths (returns error)
_, err := store.ResolvePath("")                          // ErrEmptyPath
_, err := store.ResolvePath("/etc/passwd")               // ErrAbsolutePath
_, err := store.ResolvePath("../../../etc/passwd")       // ErrPathEscape
_, err := store.ResolvePath("sebastian/../../root")      // ErrPathEscape
```

---

### File Operations

#### `ReadFile(person, relativePath string) (string, error)`

Reads the content of a file.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `string` | Person context (e.g., "sebastian") |
| `relativePath` | `string` | Path relative to person's directory |

**Returns:** `string` - File content as UTF-8 text, `error` - error if any

**Errors:**
| Error | Condition |
|-------|-----------|
| `ErrInvalidPath` | Invalid path (see `ResolvePath`) |
| `os.ErrNotExist` | File does not exist |
| `ErrIsDirectory` | Path is a directory |

---

#### `WriteFile(person, relativePath string, content string) error`

Writes content to a file, creating parent directories if needed.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `string` | Person context |
| `relativePath` | `string` | Path relative to person's directory |
| `content` | `string` | Content to write |

**Returns:** `error` - error if any

**Errors:**
| Error | Condition |
|-------|-----------|
| `ErrInvalidPath` | Invalid path (see `ResolvePath`) |
| `ErrIsDirectory` | Path is an existing directory |
| `os.ErrPermission` | Insufficient filesystem permissions |

**Behavior:**
1. Resolves and validates path
2. Creates parent directories recursively (`os.MkdirAll`)
3. Writes file atomically (`os.WriteFile`)

---

#### `AppendFile(person, relativePath string, content string) error`

Appends content to a file, creating the file and parent directories if needed.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `string` | Person context |
| `relativePath` | `string` | Path relative to person's directory |
| `content` | `string` | Content to append |

**Returns:** `error` - error if any

**Behavior:**
1. Resolves and validates path
2. Creates parent directories recursively
3. Opens file in append mode
4. Writes content at end of file

**Note:** Does not add newline automatically; caller must include newlines in content.

---

#### `DeleteFile(person, relativePath string) error`

Deletes a file.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `string` | Person context |
| `relativePath` | `string` | Path relative to person's directory |

**Returns:** `error` - error if any

**Errors:**
| Error | Condition |
|-------|-----------|
| `ErrInvalidPath` | Invalid path (see `ResolvePath`) |
| `ErrIsDirectory` | Path is a directory (directories cannot be deleted) |

**Behavior:**
1. Resolves and validates path
2. If file does not exist, returns nil (idempotent)
3. If path is a directory, returns `ErrIsDirectory`
4. Deletes the file

---

### Directory Operations

#### `ListDir(person, relativePath string) ([]FileEntry, error)`

Lists contents of a directory.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `string` | Person context |
| `relativePath` | `string` | Path relative to person's directory |

**Returns:** `[]FileEntry` - List of entries, `error` - error if any

**FileEntry Type:**
```go
type FileEntry struct {
    Name  string `json:"name"`    // Entry name (e.g., "note.md")
    Path  string `json:"path"`    // Vault-relative path (e.g., "daily/note.md")
    IsDir bool   `json:"is_dir"`  // True if directory, False if file
}
```

**Errors:**
| Error | Condition |
|-------|-----------|
| `ErrInvalidPath` | Invalid path (see `ResolvePath`) |
| `os.ErrNotExist` | Directory does not exist |
| `ErrNotDirectory` | Path is a file, not a directory |

**Behavior:**
1. Resolves and validates path
2. Reads directory contents (`os.ReadDir`)
3. Excludes hidden files (names starting with `.`)
4. Sorts entries: files first, then directories, alphabetically (case-insensitive)

**Example Response:**
```go
[]FileEntry{
    {Name: "notes.md", Path: "notes.md", IsDir: false},
    {Name: "todo.md", Path: "todo.md", IsDir: false},
    {Name: "daily", Path: "daily", IsDir: true},
}
```

---

## Git Sync Package

The `git` functions in the `vault` package provide Git operations for synchronizing the vault with a remote repository.

### Pull Operation

#### `GitPull() (bool, string)`

Pulls latest changes from the remote repository.

**Parameters:** None

**Returns:** `bool` - Success status, `string` - Status message or error description

**Behavior:**

1. **Repository Check:**
   - Verifies `.git` directory exists in vault root
   - Returns `(false, "Not a git repository")` if missing

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
| Success | `(true, "Pull successful")` |
| No remote changes | `(true, "Already up to date")` |
| Fallback succeeded | `(true, "Reset to remote successful")` |
| Not a git repo | `(false, "Not a git repository")` |
| Network error | `(false, "<error message>")` |

---

### Commit and Push Operation

#### `GitCommitAndPush(message string) (bool, string)`

Stages all changes, commits, and pushes to remote.

**Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `message` | `string` | - | Commit message (use "Update notes" as default) |

**Returns:** `bool` - Success status, `string` - Status message or error description

**Behavior:**

1. **Check for Changes:**
   - Executes: `git status --porcelain`
   - Returns early if no changes: `(true, "No changes to commit")`

2. **Stage Changes:**
   - Executes: `git add .`
   - Stages all new, modified, and deleted files

3. **Commit:**
   - Executes: `git commit -m "<message>"`
   - Uses provided message

4. **Push:**
   - Executes: `git push`
   - On failure, pulls and retries push once

5. **Graceful Degradation:**
   - If push ultimately fails, changes remain committed locally
   - Returns `(false, "<error>")` but data is preserved

**Return Values:**
| Condition | Return |
|-----------|--------|
| Success | `(true, "Changes pushed successfully")` |
| No changes | `(true, "No changes to commit")` |
| Commit succeeded, push failed | `(false, "Push failed: <error>")` |
| Commit failed | `(false, "Commit failed: <error>")` |

### Git Command Execution

```go
func runGitCommand(args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = vaultRoot
    output, err := cmd.CombinedOutput()
    return string(output), err
}
```

---

## Security Considerations

### Path Traversal Prevention

The `ResolvePath` function implements multiple layers of protection:

1. **Empty Path Rejection:** Prevents operations on vault root itself
2. **Absolute Path Rejection:** Blocks paths starting with `/`
3. **Path Cleaning:** Uses `filepath.Clean()` to normalize paths
4. **Containment Validation:** Verifies resolved path is within vault root using `strings.HasPrefix`

**Attack Vectors Mitigated:**
- `../../../etc/passwd` - Blocked by containment check
- `/etc/passwd` - Blocked by absolute path check
- `foo/../../etc/passwd` - Blocked by containment check after resolution

### Hidden File Filtering

The `ListDir` function excludes files starting with `.`:
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

The REST API handlers use the vault package:

```go
// Path prefixing with person context
func (h *Handler) GetFileContent(w http.ResponseWriter, r *http.Request) {
    person := auth.PersonFromContext(r.Context())
    path := r.URL.Query().Get("path")

    content, err := h.store.ReadFile(person, path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            writeError(w, 404, "File not found")
            return
        }
        writeError(w, 400, err.Error())
        return
    }

    json.NewEncoder(w).Encode(FileReadResponse{
        Path:    path,
        Content: content,
    })
}

// Writing with git sync
func (h *Handler) SaveNote(w http.ResponseWriter, r *http.Request) {
    person := auth.PersonFromContext(r.Context())
    // ... parse request body ...

    if err := h.store.WriteFile(person, path, content); err != nil {
        writeError(w, 400, err.Error())
        return
    }

    success, msg := h.store.GitCommitAndPush("Update daily note")
    json.NewEncoder(w).Encode(ApiMessage{
        Success: success,
        Message: msg,
    })
}
```

### Typical Operation Sequence

**Reading data:**
```
1. GitPull()              // Sync from remote
2. ReadFile(person, path) // Read file content
3. Return JSON to client
```

**Writing data:**
```
1. WriteFile(person, path, content)  // Write to filesystem
2. GitCommitAndPush(message)         // Sync to remote
3. Return success/failure to client
```

### Concurrent Access

The current implementation does not handle concurrent access:
- Multiple simultaneous writes may cause race conditions
- Git operations are not locked
- For single-user or low-concurrency scenarios only

---

## Testing

### Unit Tests

```go
func TestStore_ReadFile(t *testing.T) {
    tests := []struct {
        name    string
        person  string
        path    string
        setup   func(root string)
        want    string
        wantErr error
    }{
        {
            name:   "reads existing file",
            person: "sebastian",
            path:   "daily/2026-01-27.md",
            setup: func(root string) {
                os.MkdirAll(filepath.Join(root, "sebastian", "daily"), 0755)
                os.WriteFile(
                    filepath.Join(root, "sebastian", "daily", "2026-01-27.md"),
                    []byte("# daily 2026-01-27\n"),
                    0644,
                )
            },
            want: "# daily 2026-01-27\n",
        },
        {
            name:    "rejects path traversal",
            person:  "sebastian",
            path:    "../../../etc/passwd",
            wantErr: ErrPathEscape,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            root := t.TempDir()
            if tt.setup != nil {
                tt.setup(root)
            }
            store := NewStore(root)

            got, err := store.ReadFile(tt.person, tt.path)
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Errorf("got error %v, want %v", err, tt.wantErr)
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

---

## Limitations

1. **No Directory Deletion:** `DeleteFile` only handles files
2. **No File Renaming:** Must delete and recreate
3. **UTF-8 Only:** Binary files not supported
4. **No Locking:** Concurrent access may cause issues
5. **Remote-Wins Conflict Resolution:** Local changes may be lost on conflicts
