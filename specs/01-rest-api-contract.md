# REST API Contract Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-18

## Overview

This document defines the REST API contract between the Android mobile application and the FastAPI server for the Notes Editor application. The API provides endpoints for managing daily notes, todos, sleep time tracking, file management, Claude AI chat, and application settings.

## Authentication

### Bearer Token Authentication
All API endpoints (except `/login` and `/api/linkedin/oauth/callback`) require authentication via Bearer token.

**Header Format:**
```
Authorization: Bearer <NOTES_TOKEN>
```

The token is validated using constant-time comparison (`secrets.compare_digest`).

### Person Context Header
Most endpoints require a person context to scope data access. This is provided via:

**Header:**
```
X-Notes-Person: <person_name>
```

Valid values: `sebastian`, `petra`

If the person header is missing or invalid, the API returns HTTP 400 with `"Person not selected"`.

### Common Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | `Bearer <token>` |
| `Accept` | Yes | `application/json` or `application/x-ndjson` for streams |
| `X-Notes-Person` | Yes* | User context (required for most endpoints) |

---

## Endpoints

### Daily Notes

#### `GET /api/daily`
**Purpose**: Fetch today's daily note for the authenticated person.

**Authentication**: Required
**Person Header**: Required

**Request**: No body required.

**Response (200)**:
```json
{
  "date": "2026-01-18",
  "path": "daily/2026-01-18.md",
  "content": "# daily 2026-01-18\n\n## todos\n..."
}
```

**Behavior**:
- Performs `git pull` before reading
- Creates the daily note file if it doesn't exist
- Carries forward incomplete todos and pinned notes from previous day

---

#### `POST /api/save`
**Purpose**: Overwrite the entire content of today's daily note.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | string | Yes | Full markdown content |

**Response (200)**:
```json
{
  "success": true,
  "message": "Note saved successfully"
}
```

**Error Response**:
```json
{
  "success": false,
  "message": "<error details>"
}
```

**Behavior**: Commits and pushes changes to git.

---

#### `POST /api/append`
**Purpose**: Append a timestamped entry to the "custom notes" section of today's note.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | string | Yes | Note content to append |
| `pinned` | string | No | `"on"` to mark entry as pinned |

**Response (200)**:
```json
{
  "success": true,
  "message": "Content appended successfully"
}
```

**Error (400)**:
```json
{
  "detail": "Content cannot be empty"
}
```

**Behavior**:
- Creates `## custom notes` section if it doesn't exist
- Adds entry with header `### HH:MM` or `### HH:MM <pinned>`
- Commits and pushes to git

---

#### `POST /api/clear-pinned`
**Purpose**: Remove all `<pinned>` markers from today's note.

**Authentication**: Required
**Person Header**: Required

**Request Body**: Empty

**Response (200)**:
```json
{
  "success": true,
  "message": "Pinned markers cleared"
}
```

---

### Todos

#### `POST /api/todos/add`
**Purpose**: Add a new todo item to a category in today's note.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `category` | string | Yes | `"work"` or `"priv"` |
| `text` | string | No | Task text (blank line if not provided) |

**Response (200)**:
```json
{
  "success": true,
  "message": "Task added"
}
```

**Error (400)**:
```json
{
  "detail": "Invalid category"
}
```

**Behavior**:
- Creates `## todos` section if needed
- Creates `### <category>` subsection if needed
- If `text` is provided: adds `- [ ] {text}` line
- If `text` is empty/missing: adds `- [ ]` line (backwards compatible)

---

#### `POST /api/todos/toggle`
**Purpose**: Toggle a todo item between checked and unchecked state.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Relative file path |
| `line` | integer | Yes | 1-indexed line number |

**Response (200)**:
```json
{
  "success": true,
  "message": "Task updated"
}
```

**Error (400)**:
```json
{
  "detail": "Invalid line number"
}
```

**Error (404)**:
```json
{
  "detail": "File not found"
}
```

**Behavior**:
- Toggles `- [ ]` to `- [x]` and vice versa
- Returns success with "Not a task line" if line doesn't match todo pattern

---

### Sleep Times

#### `GET /api/sleep-times`
**Purpose**: Get recent sleep time entries.

**Authentication**: Required
**Person Header**: Required

**Response (200)**:
```json
{
  "entries": [
    {
      "line_no": 15,
      "text": "2026-01-18 | Max | 19:30 | eingeschlafen"
    },
    {
      "line_no": 14,
      "text": "2026-01-17 | Max | 06:15 | aufgewacht"
    }
  ]
}
```

**Behavior**:
- Returns up to 20 most recent entries
- Entries are returned in reverse chronological order
- Creates sleep_times.md if it doesn't exist

---

#### `POST /api/sleep-times/append`
**Purpose**: Add a new sleep time entry.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `child` | string | Yes | Child's name |
| `entry` | string | Yes | Time entry (e.g., "19:30") |
| `asleep` | string | No | `"on"` to mark as fell asleep |
| `woke` | string | No | `"on"` to mark as woke up |

**Response (200)**:
```json
{
  "success": true,
  "message": "Entry added"
}
```

**Error (400)**:
```json
{
  "detail": "Entry cannot be empty"
}
```

**Behavior**:
- Creates entry in format: `YYYY-MM-DD | Name | entry | suffix`
- Suffix is `eingeschlafen` or `aufgewacht` based on flags

---

#### `POST /api/sleep-times/delete`
**Purpose**: Delete a sleep time entry by line number.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `line` | integer | Yes | 1-indexed line number |

**Response (200)**:
```json
{
  "success": true,
  "message": "Entry deleted"
}
```

**Error (400)**:
```json
{
  "detail": "Invalid line number"
}
```
or
```json
{
  "detail": "Line out of range"
}
```

---

### Files

#### `GET /api/files/list`
**Purpose**: List files and directories at a given path.

**Authentication**: Required
**Person Header**: Required

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | No | Relative path (default: `.`) |

**Response (200)**:
```json
{
  "entries": [
    {
      "name": "daily",
      "path": "daily",
      "is_dir": true
    },
    {
      "name": "notes.md",
      "path": "notes.md",
      "is_dir": false
    }
  ]
}
```

---

#### `GET /api/files/read`
**Purpose**: Read the content of a file.

**Authentication**: Required
**Person Header**: Required

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Relative file path (URL-encoded) |

**Response (200)**:
```json
{
  "path": "daily/2026-01-18.md",
  "content": "# daily 2026-01-18\n..."
}
```

**Error (400)**:
```json
{
  "detail": "Path is a directory"
}
```

**Error (404)**:
```json
{
  "detail": "File not found"
}
```

---

#### `POST /api/files/create`
**Purpose**: Create a new empty file.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Relative file path |

**Response (200)**:
```json
{
  "success": true,
  "message": "File created"
}
```

If file already exists:
```json
{
  "success": true,
  "message": "File already exists"
}
```

**Error (400)**:
```json
{
  "detail": "Path required"
}
```

---

#### `POST /api/files/save-json`
**Purpose**: Save content to a file (JSON response).

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Relative file path |
| `content` | string | Yes | File content |

**Response (200)**:
```json
{
  "success": true,
  "message": "File saved"
}
```

---

#### `POST /api/files/delete-json`
**Purpose**: Delete a file (JSON response).

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Relative file path |

**Response (200)**:
```json
{
  "success": true,
  "message": "File deleted"
}
```

**Error** (directory):
```json
{
  "success": false,
  "message": "Cannot delete a directory"
}
```

---

#### `GET /api/files/tree`
**Purpose**: Get HTML-rendered tree view of directory contents for HTMX integration.

**Authentication**: Required
**Person Header**: Required

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | No | Relative path (default: `.`) |

**Response (200)**: HTML fragment containing nested directory tree with HTMX attributes.

**Notes**: Used by web interface for lazy-loading directory contents. Returns HTML, not JSON.

---

#### `GET /api/files/open`
**Purpose**: Open a file in the web editor (returns full HTML page).

**Authentication**: Required
**Person Header**: Required

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Relative file path |

**Response (200)**: HTML page (`file_editor.html` template) with file content and rendered note view.

**Error (400)**:
```json
{
  "detail": "Path is a directory"
}
```

**Error (404)**:
```json
{
  "detail": "File not found"
}
```

---

#### `POST /api/files/save`
**Purpose**: Save file content (HTML response for HTMX).

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Relative file path |
| `content` | string | Yes | File content |

**Response (200)**: HTML page (`file_editor.html` template) with updated content and status message.

**Notes**: Use `/api/files/save-json` for JSON response. This endpoint is for web HTMX forms.

---

#### `POST /api/files/delete`
**Purpose**: Delete a file (HTML response for HTMX).

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Relative file path |

**Response (200)**: HTML page with deletion confirmation. Sets `HX-Trigger: fileDeleted` header.

**Error** (directory): Returns HTML with error message "Cannot delete a directory".

**Notes**: Use `/api/files/delete-json` for JSON response. This endpoint is for web HTMX forms.

---

#### `POST /api/files/unpin`
**Purpose**: Remove the `<pinned>` marker from a specific line.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Relative file path |
| `line` | integer | Yes | 1-indexed line number |

**Response (200)**:
```json
{
  "success": true,
  "message": "Entry unpinned"
}
```

If line is not pinned:
```json
{
  "success": true,
  "message": "Entry already unpinned"
}
```

---

### Claude AI Chat

#### `POST /api/claude/chat`
**Purpose**: Send a message to Claude and receive a complete response.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `message` | string | Yes | User message |
| `session_id` | string | No | Existing session ID for conversation continuity |

**Response (200)** - Success:
```json
{
  "success": true,
  "session_id": "abc123",
  "response": "Claude's response text",
  "history": [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there!"}
  ]
}
```

**Response (200)** - Error:
```json
{
  "success": false,
  "message": "Error description",
  "response": "",
  "history": []
}
```

---

#### `POST /api/claude/chat-stream`
**Purpose**: Send a message to Claude and receive a streaming response.

**Authentication**: Required
**Person Header**: Required

**Request Headers**:
```
Accept: application/x-ndjson
```

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `message` | string | Yes | User message |
| `session_id` | string | No | Existing session ID |

**Response**: `StreamingResponse` with `application/x-ndjson` media type.

**Stream Event Types** (one JSON object per line):

Text delta:
```json
{"type": "text", "delta": "partial text"}
```

Tool use start:
```json
{"type": "tool_use", "name": "tool_name", "input": {...}}
```

Session info:
```json
{"type": "session", "session_id": "abc123"}
```

Keepalive (sent every 5 seconds):
```json
{"type": "ping"}
```

Error:
```json
{"type": "error", "message": "error description"}
```

---

#### `POST /api/claude/clear`
**Purpose**: Clear a chat session's history.

**Authentication**: Required
**Person Header**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session_id` | string | Yes | Session ID to clear |

**Response (200)**:
```json
{
  "success": true,
  "message": "Session cleared"
}
```

If session not found:
```json
{
  "success": false,
  "message": "Session not found"
}
```

---

#### `GET /api/claude/history`
**Purpose**: Retrieve the message history for a chat session.

**Authentication**: Required
**Person Header**: Required

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `session_id` | string | Yes | Session ID to retrieve history for |

**Response (200)** - Success:
```json
{
  "success": true,
  "history": [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there!"}
  ]
}
```

**Response (200)** - Session not found:
```json
{
  "success": false,
  "message": "Session not found",
  "history": []
}
```

---

### Settings

#### `GET /api/settings/env`
**Purpose**: Retrieve the current `.env` file content.

**Authentication**: Required
**Person Header**: Not required

**Response (200)**:
```json
{
  "success": true,
  "content": "KEY=value\nOTHER_KEY=other_value\n"
}
```

---

#### `POST /api/settings/env`
**Purpose**: Update the `.env` file content.

**Authentication**: Required
**Person Header**: Not required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `env_content` | string | Yes | New env file content |

**Response (200)**:
```json
{
  "success": true,
  "message": "Saved"
}
```

**Behavior**: Normalizes line endings and reloads environment variables.

---

### LinkedIn OAuth

#### `GET /api/linkedin/oauth/callback`
**Purpose**: Handle LinkedIn OAuth 2.0 authorization callback.

**Authentication**: Not required (OAuth flow)
**Person Header**: Not required

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `code` | string | Yes | Authorization code from LinkedIn |

**Response (200)**:
```json
{
  "success": true,
  "expires_in": 5184000
}
```

**Error (400)**:
```json
{
  "detail": "LinkedIn response missing access_token"
}
```

**Behavior**:
- Exchanges authorization code for access token via LinkedIn OAuth endpoint
- Persists access token to `.env` file
- Returns token expiration time (typically 60 days)

**Notes**: See [05-linkedin-service.md](./05-linkedin-service.md) for full OAuth flow documentation.

---

## Data Models

### ApiMessage
Generic success/error response.
```kotlin
data class ApiMessage(
    val success: Boolean = true,
    val message: String = ""
)
```

### DailyNote
Today's daily note content.
```kotlin
data class DailyNote(
    val date: String,      // "YYYY-MM-DD"
    val path: String,      // Relative path from person root
    val content: String    // Full markdown content
)
```

### SleepTimesResponse
List of sleep entries.
```kotlin
data class SleepTimesResponse(
    val entries: List<SleepEntry>
)

data class SleepEntry(
    @SerialName("line_no")
    val lineNo: Int,       // 1-indexed line number
    val text: String       // Full line text
)
```

### FilesResponse
Directory listing.
```kotlin
data class FilesResponse(
    val entries: List<FileEntry>
)

data class FileEntry(
    val name: String,      // File/directory name
    val path: String,      // Relative path
    @SerialName("is_dir")
    val isDir: Boolean
)
```

### FileReadResponse
File content.
```kotlin
data class FileReadResponse(
    val path: String,      // Relative path
    val content: String    // File content
)
```

### EnvResponse
Environment settings.
```kotlin
data class EnvResponse(
    val success: Boolean,
    val content: String = "",
    val message: String = ""
)
```

### ClaudeChatResponse
Non-streaming chat response.
```kotlin
data class ClaudeChatResponse(
    val success: Boolean,
    val message: String = "",
    val response: String = "",
    @SerialName("session_id")
    val sessionId: String = "",
    val history: List<ChatMessage> = emptyList()
)

data class ChatMessage(
    val role: String,      // "user" or "assistant"
    val content: String
)
```

### ClaudeStreamEvent
Streaming chat event.
```kotlin
data class ClaudeStreamEvent(
    val type: String,              // "text", "tool_use", "session", "ping", "error"
    val delta: String? = null,     // Text content (for type="text")
    val name: String? = null,      // Tool name (for type="tool_use")
    val input: JsonElement? = null,// Tool input (for type="tool_use")
    @SerialName("session_id")
    val sessionId: String? = null, // Session ID (for type="session")
    val message: String? = null    // Error message (for type="error")
)
```

---

## Error Handling

### HTTP Status Codes

| Code | Meaning | When Used |
|------|---------|-----------|
| 200 | OK | Successful request |
| 400 | Bad Request | Invalid input, missing required fields, invalid person |
| 401 | Unauthorized | Missing or invalid auth token |
| 404 | Not Found | File/resource not found |

### Error Response Formats

**FastAPI HTTPException** (validation errors):
```json
{
  "detail": "Error message"
}
```

**Application-level errors** (operation failures):
```json
{
  "success": false,
  "message": "Error description"
}
```

### Common Error Scenarios

1. **Missing Authentication**: HTTP 401 with `{"detail": "Unauthorized"}`
2. **Missing Person Header**: HTTP 400 with `{"detail": "Person not selected"}`
3. **File Not Found**: HTTP 404 with `{"detail": "File not found"}`
4. **Invalid Input**: HTTP 400 with `{"detail": "<specific error>"}`
5. **Git Operation Failed**: HTTP 200 with `{"success": false, "message": "<git error>"}`

---

## Notes

### Content Type
- All POST requests use `application/x-www-form-urlencoded` encoding
- All responses use `application/json` except for streaming endpoints

### Path Handling
- Paths are relative to the person's root directory in the vault
- The server automatically prefixes paths with the person's directory
- URL encoding is required for paths in query parameters

### Git Integration
- Most write operations trigger `git commit` and `git push`
- Read operations trigger `git pull` to ensure fresh data
- Git failures result in `success: false` but don't cause HTTP errors

### Multi-Server Fallback
The Android client supports multiple base URLs and will try them in order if one fails (for network resilience).

### Boolean Form Fields
Boolean flags in form data use the convention:
- Present with value `"on"` = true
- Absent or empty = false

### Streaming Timeout
The streaming endpoint uses an infinite read timeout on the client side. The server sends `{"type": "ping"}` every 5 seconds to keep the connection alive.
