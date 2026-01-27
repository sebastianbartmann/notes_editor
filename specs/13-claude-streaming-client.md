# Claude Chat Streaming Client Specification

> Status: Draft
> Version: 2.0
> Last Updated: 2026-01-27

## Overview

This document specifies the client-side implementations for consuming the Claude chat streaming API on Android and React Web platforms. It complements [04-claude-service.md](./04-claude-service.md) which covers the server-side Go implementation.

**Implementation:** Android (`clients/android/`), React (`clients/web/src/hooks/useClaudeStream.ts`)

The streaming client enables real-time display of Claude's responses as they are generated, providing visual feedback during tool execution and maintaining conversation context through session management.

**Key Responsibilities:**
- Establish HTTP connections with appropriate streaming configurations
- Parse NDJSON (Newline-Delimited JSON) events from the response stream
- Update UI incrementally as text deltas arrive
- Display status messages during tool execution
- Persist session IDs for conversation continuity
- Handle connection errors and stream interruptions gracefully

---

## Streaming Protocol

### NDJSON Format

The server sends responses as Newline-Delimited JSON (NDJSON), where each line is a complete JSON object. Clients must parse line-by-line rather than waiting for the full response.

**MIME Type:** `application/x-ndjson`

### Event Types

#### Text Content
Incremental text from Claude's response. Clients accumulate these deltas to build the complete message.

```json
{"type": "text", "delta": "partial response text"}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"text"` |
| `delta` | string | Partial text content to append |

#### Status Update
Informational message about current processing state, typically shown in a status bar.

```json
{"type": "status", "message": "Running tool: Read"}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"status"` |
| `message` | string | Human-readable status text |

#### Tool Invocation
Notification that Claude is executing a tool. Useful for showing detailed progress.

```json
{"type": "tool", "name": "Read", "input": {"file_path": "/path/to/file"}}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"tool"` |
| `name` | string | Tool name being executed |
| `input` | object | Tool input parameters |

**Common Tool Inputs:**
- `Read`: `{"file_path": "/path/to/file"}`
- `WebFetch`: `{"url": "https://example.com"}`
- `Bash`: `{"command": "ls -la"}`

#### Keep-Alive Ping
Sent every 5 seconds during periods of inactivity to prevent connection timeouts.

```json
{"type": "ping"}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"ping"` |

**Client Behavior:** Ignore silently; do not update UI.

#### Stream Complete
Signals successful completion of the response stream. Contains the session ID for future requests.

```json
{"type": "done", "session_id": "uuid-string"}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"done"` |
| `session_id` | string | UUID for session continuity |

#### Error Occurred
Indicates an error during processing. The stream may continue or terminate after this event.

```json
{"type": "error", "message": "Error description"}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"error"` |
| `message` | string | Error description |

---

## HTTP Configuration

### Request Configuration

**Endpoint:** `POST /api/claude/chat-stream`

**Headers:**
| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer <token>` | Yes |
| `Accept` | `application/x-ndjson` | Yes |
| `X-Notes-Person` | `sebastian` or `petra` | Yes |
| `Content-Type` | `application/json` | Yes |

**Request Body (JSON):**
```json
{
  "message": "User's message",
  "session_id": "existing-session-id-or-null"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `message` | string | Yes | User's message |
| `session_id` | string | No | Existing session ID for continuity |

### Timeout Configuration

| Platform | Read Timeout | Connect Timeout | Write Timeout |
|----------|--------------|-----------------|---------------|
| Android | Infinite (0) | 30 seconds | 30 seconds |
| Web | N/A (browser) | N/A (browser) | N/A (browser) |

**Rationale:** Streaming responses can take minutes for complex queries. The server sends ping events every 5 seconds to keep connections alive.

### Response Configuration

**Content-Type:** `application/x-ndjson`
**Transfer-Encoding:** `chunked`

---

## Android Implementation

### Data Model

**File:** `Models.kt`

```kotlin
@Serializable
data class ClaudeStreamEvent(
    val type: String,
    val delta: String? = null,
    val name: String? = null,
    val input: JsonElement? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
    val message: String? = null
)
```

| Property | Type | Events | Description |
|----------|------|--------|-------------|
| `type` | String | All | Event type discriminator |
| `delta` | String? | text | Incremental text content |
| `name` | String? | tool | Tool name |
| `input` | JsonElement? | tool | Tool input as raw JSON |
| `sessionId` | String? | done | Session ID for continuity |
| `message` | String? | status, error | Status or error message |

### ApiClient Streaming

**File:** `ApiClient.kt`

#### Dedicated Streaming Client

A separate OkHttpClient instance is required for streaming to configure infinite read timeout:

```kotlin
private val streamClient = OkHttpClient.Builder()
    .readTimeout(0, TimeUnit.MILLISECONDS)  // Infinite timeout for streaming
    .build()
```

**Note:** The standard client with default timeouts would terminate long-running streams.

#### Stream Function Signature

```kotlin
fun claudeChatStream(message: String, sessionId: String?): Flow<ClaudeStreamEvent>
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `message` | String | User's message to send |
| `sessionId` | String? | Existing session ID or null for new session |

**Returns:** `Flow<ClaudeStreamEvent>` - Cold flow that emits events as they arrive.

#### Implementation Pattern

```kotlin
fun claudeChatStream(message: String, sessionId: String?): Flow<ClaudeStreamEvent> {
    val formBody = FormBody.Builder()
        .add("message", message)
        .apply { sessionId?.let { add("session_id", it) } }
        .build()

    return executeStream { baseUrl ->
        Request.Builder()
            .url("$baseUrl/api/claude/chat-stream")
            .post(formBody)
            .addHeader("Authorization", "Bearer $token")
            .addHeader("X-Notes-Person", person)
            .addHeader("Accept", "application/x-ndjson")
            .build()
    }
}
```

#### Stream Execution with Failover

```kotlin
private fun executeStream(buildRequest: (String) -> Request): Flow<ClaudeStreamEvent> = channelFlow {
    var lastError: Exception? = null

    for (baseUrl in baseUrls) {
        try {
            val request = buildRequest(baseUrl)
            val response = streamClient.newCall(request).execute()

            if (!response.isSuccessful) {
                lastError = IOException("HTTP ${response.code}")
                continue
            }

            response.body?.source()?.use { source ->
                while (!source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    if (line.isBlank()) continue

                    val event = json.decodeFromString<ClaudeStreamEvent>(line)
                    send(event)
                }
            }
            return@channelFlow  // Success - exit

        } catch (e: Exception) {
            lastError = e
            // Try next URL
        }
    }

    // All URLs failed
    send(ClaudeStreamEvent(type = "error", message = lastError?.message ?: "Connection failed"))
}
```

**Key Implementation Details:**

1. **channelFlow**: Used instead of `flow` because it supports sending from callbacks and suspending operations.

2. **Failover Logic**: Attempts each base URL in sequence. Only proceeds to next URL on connection/HTTP errors, not on stream parse errors.

3. **Line-by-Line Reading**: Uses `BufferedSource.readUtf8Line()` for proper NDJSON parsing. Empty lines are skipped.

4. **Resource Management**: The `use` block ensures the response body is closed even on exceptions.

5. **Error Propagation**: Connection failures emit an error event rather than throwing, allowing UI to display the error gracefully.

---

## Android UI Implementation

### ToolClaudeScreen

**File:** `ToolClaudeScreen.kt`

#### State Management

```kotlin
@Composable
fun ToolClaudeScreen(
    navController: NavController,
    apiClient: ApiClient
) {
    var inputText by remember { mutableStateOf("") }
    var isLoading by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf("") }
    val messages = ClaudeSessionStore.messages  // SnapshotStateList<ChatMessage>
    val scope = rememberCoroutineScope()

    // ... UI composition
}
```

| State Variable | Type | Purpose |
|----------------|------|---------|
| `inputText` | String | Current text in input field |
| `isLoading` | Boolean | Disables input during streaming |
| `statusMessage` | String | Status bar text (tool execution, errors) |
| `messages` | SnapshotStateList | Chat history (survives recomposition) |

#### Message Sending Flow

```kotlin
fun sendMessage() {
    val text = inputText.trim()
    if (text.isEmpty() || isLoading) return

    inputText = ""
    isLoading = true
    statusMessage = ""

    // Add user message
    messages.add(ChatMessage(role = "user", content = text))

    // Add placeholder for assistant response
    val assistantIndex = messages.size
    messages.add(ChatMessage(role = "assistant", content = ""))
    var assistantText = ""

    scope.launch {
        apiClient.claudeChatStream(text, ClaudeSessionStore.sessionId)
            .catch { e ->
                statusMessage = "Error: ${e.message}"
                isLoading = false
            }
            .collect { event ->
                handleStreamEvent(event, assistantIndex, assistantText) { newText ->
                    assistantText = newText
                }
            }
        isLoading = false
    }
}
```

#### Event Handling

```kotlin
private fun handleStreamEvent(
    event: ClaudeStreamEvent,
    assistantIndex: Int,
    currentText: String,
    updateText: (String) -> Unit
) {
    when (event.type) {
        "text" -> {
            val newText = currentText + event.delta.orEmpty()
            updateText(newText)
            messages[assistantIndex] = ChatMessage(role = "assistant", content = newText)
        }

        "status" -> {
            statusMessage = event.message ?: "Working..."
        }

        "tool" -> {
            val url = event.input?.jsonObject?.get("url")?.jsonPrimitive?.contentOrNull
            statusMessage = if (url != null) {
                "Tool: ${event.name} $url"
            } else {
                "Tool: ${event.name}"
            }
        }

        "ping" -> {
            // Keep-alive signal; no UI update required
        }

        "done" -> {
            ClaudeSessionStore.sessionId = event.sessionId
            statusMessage = ""
        }

        "error" -> {
            statusMessage = "Error: ${event.message}"
        }
    }
}
```

**Event Handling Behavior:**

| Event | UI Update | Side Effect |
|-------|-----------|-------------|
| `text` | Append to message, update list | None |
| `status` | Update status bar | None |
| `tool` | Update status bar with tool info | None |
| `ping` | None | None |
| `done` | Clear status bar | Save session ID |
| `error` | Show error in status bar | None |

---

## ClaudeSessionStore

**File:** `ClaudeSessionStore.kt`

A singleton object that persists chat state across screen navigations within the app session.

```kotlin
object ClaudeSessionStore {
    var sessionId: String? = null
    val messages = mutableStateListOf<ChatMessage>()

    fun clear() {
        sessionId = null
        messages.clear()
    }
}
```

| Property | Type | Purpose |
|----------|------|---------|
| `sessionId` | String? | Current session ID for API continuity |
| `messages` | SnapshotStateList | Chat history for UI display |

**Lifecycle:**
- Survives screen rotations and navigation
- Cleared on explicit user action (clear chat button)
- Lost on app process termination

**Usage in ToolClaudeScreen:**
```kotlin
// Read session ID for API calls
apiClient.claudeChatStream(text, ClaudeSessionStore.sessionId)

// Update session ID from done event
"done" -> ClaudeSessionStore.sessionId = event.sessionId

// Display messages
LazyColumn {
    items(ClaudeSessionStore.messages) { message ->
        ChatBubble(message)
    }
}
```

---

## Web Implementation

### Fetch API with ReadableStream

**File:** `llm.html`

#### Stream Initialization

```javascript
let sessionId = null;

async function sendMessage(message) {
    const formData = new URLSearchParams();
    formData.append('message', message);
    if (sessionId) {
        formData.append('session_id', sessionId);
    }

    const res = await fetch('/api/claude/chat-stream', {
        method: 'POST',
        headers: {
            'Authorization': `Bearer ${token}`,
            'X-Notes-Person': person,
            'Accept': 'application/x-ndjson',
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: formData.toString()
    });

    if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
    }

    await processStream(res);
}
```

#### Stream Processing with Buffer Management

```javascript
async function processStream(res) {
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let assistantText = '';
    const contentEl = createMessageElement('assistant');

    while (true) {
        const { value, done } = await reader.read();
        if (done) break;

        // Decode chunk and append to buffer
        buffer += decoder.decode(value, { stream: true });

        // Split on newlines, keeping incomplete line in buffer
        const lines = buffer.split('\n');
        buffer = lines.pop();  // Last element may be incomplete

        for (const line of lines) {
            if (!line.trim()) continue;

            const event = JSON.parse(line.trim());
            assistantText = handleEvent(event, assistantText, contentEl);
        }
    }

    // Process any remaining buffer content
    if (buffer.trim()) {
        const event = JSON.parse(buffer.trim());
        handleEvent(event, assistantText, contentEl);
    }
}
```

**Buffer Management Explanation:**

1. **Chunked Delivery:** The browser may receive partial JSON lines across multiple chunks.

2. **Split Strategy:** Split buffer on `\n`, keeping the last segment (potentially incomplete) for the next iteration.

3. **Final Flush:** After the stream ends, process any remaining content in the buffer.

#### Event Handling

```javascript
function handleEvent(event, assistantText, contentEl) {
    if (event.type === 'text') {
        assistantText += event.delta || '';
        updateMessage(contentEl, assistantText);
    } else if (event.type === 'status') {
        setStatus(event.message || 'Working...');
    } else if (event.type === 'tool') {
        const toolDetail = event.input?.url ? ` ${event.input.url}` : '';
        setStatus(`Tool: ${event.name}${toolDetail}`);
    } else if (event.type === 'ping') {
        // Keep-alive signal; no UI update required
    } else if (event.type === 'done') {
        sessionId = event.session_id;
        setStatus('');
    } else if (event.type === 'error') {
        assistantText += `\nError: ${event.message}`;
        updateMessage(contentEl, assistantText);
    }

    return assistantText;
}
```

#### Helper Functions

```javascript
function createMessageElement(role) {
    const el = document.createElement('div');
    el.className = `message ${role}`;
    document.getElementById('chat-container').appendChild(el);
    return el;
}

function updateMessage(el, text) {
    el.innerHTML = marked.parse(text);  // Render markdown
    scrollToBottom();
}

function setStatus(message) {
    document.getElementById('status-bar').textContent = message;
}

function scrollToBottom() {
    const container = document.getElementById('chat-container');
    container.scrollTop = container.scrollHeight;
}
```

---

## Session Management

### Client-Side Session Persistence

Both platforms persist the session ID client-side for conversation continuity.

| Platform | Storage Mechanism | Persistence Scope |
|----------|-------------------|-------------------|
| Android | `ClaudeSessionStore` singleton | App process lifetime |
| Web | JavaScript variable | Page session |

### Session Lifecycle

```
1. Initial Request (no session_id)
   Client: POST /api/claude/chat-stream {message: "Hello"}
   Server: ... {"type": "done", "session_id": "abc-123"}
   Client: Store session_id = "abc-123"

2. Subsequent Request (with session_id)
   Client: POST /api/claude/chat-stream {message: "Follow up", session_id: "abc-123"}
   Server: ... {"type": "done", "session_id": "abc-123"}

3. Clear Session
   Client: POST /api/claude/clear {session_id: "abc-123"}
   Client: session_id = null, messages = []
```

### Session ID Format

- Type: UUID v4 string
- Example: `"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
- Generated by server on first request

---

## Error Handling

### Android Error Handling

#### Network Errors

```kotlin
.catch { e ->
    when (e) {
        is UnknownHostException -> statusMessage = "No internet connection"
        is SocketTimeoutException -> statusMessage = "Connection timed out"
        is IOException -> statusMessage = "Network error: ${e.message}"
        else -> statusMessage = "Error: ${e.message}"
    }
    isLoading = false
}
```

#### Stream Parse Errors

```kotlin
try {
    val event = json.decodeFromString<ClaudeStreamEvent>(line)
    send(event)
} catch (e: SerializationException) {
    Log.e("ApiClient", "Failed to parse event: $line", e)
    // Continue processing; don't fail entire stream
}
```

#### Failover Behavior

1. Try first base URL
2. On connection failure, try next URL
3. After all URLs exhausted, emit error event
4. UI displays error in status bar

### Web Error Handling

#### Fetch Errors

```javascript
try {
    const res = await fetch(url, options);
    if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`);
    }
    await processStream(res);
} catch (e) {
    setStatus(`Error: ${e.message}`);
    appendErrorMessage(e.message);
}
```

#### Stream Parse Errors

```javascript
try {
    const event = JSON.parse(line.trim());
    handleEvent(event, assistantText, contentEl);
} catch (e) {
    console.error('Failed to parse event:', line, e);
    // Continue processing remaining lines
}
```

### Error Recovery Strategies

| Scenario | Android | Web |
|----------|---------|-----|
| Connection refused | Failover to next URL | Show error, allow retry |
| Mid-stream disconnect | Emit error event | Catch in read loop |
| Invalid JSON line | Log and skip | Log and skip |
| HTTP 401 | Show auth error | Redirect to login |
| HTTP 500 | Failover or show error | Show error |

---

## Platform Comparison

### Feature Parity

| Feature | Android | Web |
|---------|---------|-----|
| NDJSON parsing | Yes | Yes |
| Text streaming | Yes | Yes |
| Status display | Yes | Yes |
| Tool status | Yes | Yes |
| Session persistence | In-memory | In-memory |
| Markdown rendering | Compose + custom | marked.js |
| Auto-scroll | LazyColumn | scrollTop |

### Implementation Differences

| Aspect | Android | Web |
|--------|---------|-----|
| HTTP Client | OkHttpClient | Fetch API |
| Stream API | Flow/channelFlow | ReadableStream |
| Timeout Config | OkHttpClient.Builder | Browser default |
| JSON Parsing | kotlinx.serialization | JSON.parse |
| Reactive Updates | Compose State | DOM manipulation |
| Failover | Multi-URL support | Single endpoint |

### Code Structure Comparison

**Stream Reading:**

| Android (Kotlin) | Web (JavaScript) |
|------------------|------------------|
| `source.readUtf8Line()` | `reader.read()` + buffer split |
| Line-based iteration | Chunk-based with manual buffering |
| `channelFlow { send() }` | `while (true) { ... }` |

**State Updates:**

| Android (Kotlin) | Web (JavaScript) |
|------------------|------------------|
| `mutableStateOf()` | Direct variable |
| `SnapshotStateList` | DOM elements |
| Automatic recomposition | Manual DOM updates |

---

## Sequence Diagrams

### Successful Request Flow

```
User          Android/Web         Server           Claude
  |                |                  |                |
  |--[Send msg]--->|                  |                |
  |                |--[POST stream]-->|                |
  |                |                  |--[Query]------>|
  |                |                  |<-[Tokens]------|
  |                |<-[text delta]----|                |
  |<-[Update UI]---|                  |                |
  |                |<-[text delta]----|                |
  |<-[Update UI]---|                  |                |
  |                |<-[ping]----------|                |
  |                |                  |--[Tool call]-->|
  |                |<-[tool event]----|                |
  |<-[Show status]-|                  |                |
  |                |<-[text delta]----|                |
  |<-[Update UI]---|                  |                |
  |                |<-[done]----------|                |
  |<-[Clear status]|                  |                |
  |                |--[Store session]-|                |
```

### Error Recovery Flow (Android)

```
Android              Server1             Server2
   |                    |                    |
   |--[POST stream]--->|                    |
   |<--[Connection X]--|                    |
   |                                        |
   |--[POST stream (failover)]------------->|
   |<--[200 OK]----------------------------|
   |<--[text delta]------------------------|
   |<--[done]------------------------------|
```

---

## Implementation Checklist

### Android

- [ ] Create `ClaudeStreamEvent` data class in Models.kt
- [ ] Configure streaming OkHttpClient with infinite read timeout
- [ ] Implement `claudeChatStream()` returning Flow
- [ ] Implement `executeStream()` with NDJSON parsing
- [ ] Add multi-URL failover support
- [ ] Create `ClaudeSessionStore` singleton
- [ ] Build `ToolClaudeScreen` composable
- [ ] Implement event handling for all 6 event types
- [ ] Add error handling with user-friendly messages
- [ ] Test with slow networks and interruptions

### Web

- [ ] Set up Fetch with proper headers
- [ ] Implement ReadableStream processing
- [ ] Add buffer management for chunked lines
- [ ] Handle all 6 event types
- [ ] Implement session ID persistence
- [ ] Add markdown rendering (marked.js)
- [ ] Add auto-scroll behavior
- [ ] Add error handling and display
- [ ] Test with various response sizes

---

## Related Specifications

- [01-rest-api-contract.md](./01-rest-api-contract.md) - API contract including `/api/claude/chat-stream`
- [04-claude-service.md](./04-claude-service.md) - Server-side streaming implementation
