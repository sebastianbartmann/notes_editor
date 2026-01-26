# Claude Chat

## Purpose

Claude Chat provides an AI assistant interface within Notes Editor, allowing users to interact with Claude for help with their notes, files, and general queries. The feature integrates the `claude_agent_sdk` to provide tool-augmented conversations where Claude can read, write, and search files within the user's vault, perform web searches, and interact with LinkedIn.

## Architecture Overview

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Web Client    │      │   Android App   │      │  claude_agent_  │
│   (llm.html)    │      │ (ToolClaude     │      │      sdk        │
│                 │      │     Screen)     │      │                 │
└────────┬────────┘      └────────┬────────┘      └────────┬────────┘
         │                        │                        │
         │  POST /api/claude/     │                        │
         │  chat-stream           │                        │
         └────────────┬───────────┘                        │
                      ▼                                    │
              ┌───────────────┐                            │
              │  FastAPI      │                            │
              │  Endpoints    │                            │
              └───────┬───────┘                            │
                      │                                    │
                      ▼                                    │
              ┌───────────────┐     async for msg in       │
              │ claude_       │─────────────────────────────┤
              │ service.py    │     query(prompt, options) │
              └───────┬───────┘                            │
                      │                                    │
                      ▼                                    │
              ┌───────────────┐
              │  In-Memory    │
              │  Sessions     │
              │  (_sessions)  │
              └───────────────┘
```

## Session Management

Sessions are stored in-memory on the server using a dictionary keyed by session ID. Each session contains:

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | `str` | UUID identifying the chat session |
| `person` | `str` | The person this session belongs to |
| `messages` | `list[ChatMessage]` | Conversation history (role + content) |
| `agent_session_id` | `str | None` | SDK session ID for resumption |

Session behavior:
- New sessions created automatically if no valid `session_id` provided
- Sessions are person-scoped; a session ID from a different person creates a new session
- Clearing a session removes it from memory and resets the SDK session
- **Session resumption**: The SDK session ID is captured and used for subsequent messages, allowing Claude to maintain context across tool uses

## Allowed Tools

The agent is configured with the following tools:

| Tool | Purpose |
|------|---------|
| `Read` | Read file contents from the vault |
| `Write` | Write/create files in the vault |
| `Edit` | Edit existing files with find/replace |
| `Glob` | Find files by pattern matching |
| `Grep` | Search file contents with regex |
| `WebSearch` | Search the web for information |
| `WebFetch` | Fetch and read web page content |
| `linkedin_post` | Create LinkedIn posts |
| `linkedin_read_comments` | Read comments on LinkedIn posts |
| `linkedin_post_comment` | Post comments on LinkedIn |
| `linkedin_reply_comment` | Reply to LinkedIn comments |

LinkedIn tools are also available via MCP server with `mcp__linkedin__` prefix.

## Permission Mode

The agent runs with `permission_mode="acceptEdits"`, which auto-accepts all file edit operations. This allows Claude to read, write, and modify files within the person's vault without requiring explicit approval for each operation.

## Working Directory

The agent's working directory (`cwd`) is set to the person's vault folder: `{VAULT_ROOT}/{person}/`. All relative file paths in tool operations are resolved against this directory.

## System Prompt

Claude receives a system prompt that:
1. Identifies it as a Notes Editor assistant
2. Specifies the user's notes directory path
3. Instructs concise, helpful responses
4. Contains security guidance about untrusted web search results

## WebFetch Logging

All WebFetch requests are logged to `{VAULT_ROOT}/claude/webfetch_logs/requests.md` with:
- Timestamp
- Person who made the request
- URL fetched

Logs are committed and pushed to git after each entry.

## API Endpoints

### GET /tools/llm
Returns the Claude chat web interface HTML page.

### POST /api/claude/chat
Non-streaming chat endpoint (used less frequently).

**Request:**
- Form data: `message` (required), `session_id` (optional)
- Header: `X-Notes-Person` (required)

**Response:**
```json
{
  "success": true,
  "session_id": "uuid",
  "response": "Claude's response text",
  "history": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ]
}
```

### POST /api/claude/chat-stream
Primary streaming chat endpoint.

**Request:**
- Form data: `message` (required), `session_id` (optional)
- Header: `X-Notes-Person` (required)

**Response:** NDJSON stream (see Streaming Protocol below)

### GET /api/claude/history
Retrieve conversation history for a session.

**Request:**
- Query param: `session_id` (required)
- Header: `X-Notes-Person` (required)

**Response:**
```json
{
  "success": true,
  "history": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ]
}
```

### POST /api/claude/clear
Clear a chat session.

**Request:**
- Form data: `session_id` (required)
- Header: `X-Notes-Person` (required)

**Response:**
```json
{
  "success": true,
  "message": "Session cleared"
}
```

## Streaming Protocol

The `/api/claude/chat-stream` endpoint returns `application/x-ndjson` (newline-delimited JSON). Each line is a complete JSON object with a `type` field indicating the event type.

### Event Types

| Type | Fields | Description |
|------|--------|-------------|
| `text` | `delta` | Incremental text content from Claude |
| `status` | `message` | Status update (e.g., "Tool finished: Read") |
| `tool` | `name`, `input` | Tool invocation notification |
| `ping` | - | Keep-alive event (sent every 5 seconds during silence) |
| `done` | `session_id` | Stream complete, includes session ID for future requests |
| `error` | `message` | Error occurred during processing |

### Example Stream

```
{"type":"status","message":"Running tool: Glob"}
{"type":"tool","name":"Glob","input":{"pattern":"**/*.md"}}
{"type":"status","message":"Tool finished: Glob"}
{"type":"text","delta":"I found "}
{"type":"text","delta":"several markdown files..."}
{"type":"ping"}
{"type":"done","session_id":"abc-123-def"}
```

## Web Client Implementation

The web client (`llm.html`) implements:
- Message input with Enter key submit (Shift+Enter for newline)
- Streaming response display with incremental updates
- Status bar showing tool activity
- Clear button to reset session
- Auto-scroll to latest message
- Session ID persistence in JavaScript variable

## Android Client Implementation

### ClaudeSessionStore
Singleton object holding:
- `sessionId`: Current session ID (nullable)
- `messages`: Observable list of `ChatMessage` objects
- `clear()`: Resets both fields

### ToolClaudeScreen
Composable screen that:
- Uses Kotlin Flow to consume NDJSON stream
- Updates message list in real-time as text events arrive
- Shows status messages during tool execution
- Persists session ID from `done` events
- Calls `/api/claude/clear` on Clear button

### ApiClient
- `claudeChatStream()`: Returns `Flow<ClaudeStreamEvent>` for streaming
- `claudeClear()`: Posts to clear endpoint
- Uses `OkHttpClient` with no read timeout for streaming

### ClaudeStreamEvent Data Model
```kotlin
data class ClaudeStreamEvent(
    val type: String,
    val delta: String? = null,      // for text events
    val name: String? = null,        // for tool events
    val input: JsonElement? = null,  // for tool events
    val sessionId: String? = null,   // for done events
    val message: String? = null      // for status/error events
)
```

## Platform Differences

| Aspect | Web | Android |
|--------|-----|---------|
| Session storage | JavaScript variable | ClaudeSessionStore singleton |
| Stream consumption | ReadableStream + TextDecoder | OkHttp + Kotlin Flow |
| Message display | DOM manipulation | Jetpack Compose LazyColumn |
| Text selection | Browser native | SelectionContainer composable |

## Key Files

| Path | Description |
|------|-------------|
| `server/web_app/services/claude_service.py` | Session management, SDK integration, streaming logic |
| `server/web_app/main.py` (lines 971-1059) | API endpoint definitions |
| `server/web_app/templates/llm.html` | Web chat interface |
| `app/android/.../ToolClaudeScreen.kt` | Android chat screen |
| `app/android/.../ClaudeSessionStore.kt` | Android session state holder |
| `app/android/.../ApiClient.kt` | Android HTTP client with streaming support |
| `app/android/.../Models.kt` | Kotlin data classes for API responses |
