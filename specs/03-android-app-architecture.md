# Android App Architecture Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-18

## Overview

This document specifies the architecture of the Notes Editor Android application. The app is built with Jetpack Compose and provides a mobile interface for managing daily notes, file browsing, sleep tracking, and AI-powered chat.

**Related Specifications:**
- `01-rest-api-contract.md` - REST API endpoints consumed by this app
- `02-vault-storage-git-sync.md` - Server-side storage layer

---

## Architecture Overview

The application follows a simplified MVVM-like architecture with singleton objects for shared state and services.

```
┌─────────────────────────────────────────────────────────────┐
│                        UI Layer                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐│
│  │   Screens   │ │ Components  │ │       Theme System      ││
│  │  (Compose)  │ │ (Reusable)  │ │ (Colors, Typography)    ││
│  └──────┬──────┘ └──────┬──────┘ └─────────────────────────┘│
└─────────┼───────────────┼───────────────────────────────────┘
          │               │
┌─────────┼───────────────┼───────────────────────────────────┐
│         │    State Layer│                                    │
│  ┌──────┴──────┐ ┌──────┴──────┐ ┌─────────────────────────┐│
│  │UserSettings │ │ClaudeSession│ │    Screen-local State   ││
│  │ (Singleton) │ │   Store     │ │  (remember/mutableState)││
│  └──────┬──────┘ └──────┬──────┘ └─────────────────────────┘│
└─────────┼───────────────┼───────────────────────────────────┘
          │               │
┌─────────┼───────────────┼───────────────────────────────────┐
│         │  Network Layer│                                    │
│         └───────┬───────┘                                    │
│          ┌──────┴──────┐                                     │
│          │  ApiClient  │                                     │
│          │ (Singleton) │                                     │
│          └──────┬──────┘                                     │
│                 │                                            │
│          ┌──────┴──────┐                                     │
│          │  AppConfig  │                                     │
│          │  (Static)   │                                     │
│          └─────────────┘                                     │
└─────────────────────────────────────────────────────────────┘
```

**Key Characteristics:**
- Single-activity architecture with Compose navigation
- Singleton objects for shared state (UserSettings, ClaudeSessionStore)
- Singleton network client (ApiClient) with multi-server failover
- Screen-local state managed via Compose `remember` and `mutableStateOf`
- Custom theme system using CompositionLocal providers

---

## Modules

### Entry Point

#### MainActivity

The single activity that hosts the entire application.

**File:** `MainActivity.kt`

**Responsibilities:**
- Initialize UserSettings from SharedPreferences
- Request runtime permissions (POST_NOTIFICATIONS on Android 13+)
- Set the Compose content with theme wrapper

**Lifecycle:**
```kotlin
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        UserSettings.init(this)  // Initialize settings singleton
        // Request POST_NOTIFICATIONS permission on Android 13+
        setContent {
            NotesEditorTheme {
                NotesEditorApp()
            }
        }
    }
}
```

---

### Navigation

#### AppNavigation

Defines the navigation graph and screen routing.

**File:** `AppNavigation.kt`

**Screen Definitions:**

| Route | Label | Description |
|-------|-------|-------------|
| `daily` | Daily | Today's note with todos |
| `files` | Files | File browser |
| `sleep` | Sleep | Sleep time tracking |
| `tools` | Tools | Navigation hub to tool screens |
| `settings` | Settings | Person selection, theme, env editor |
| `tool-claude` | Claude | AI chat interface |
| `tool-noise` | Noise | White noise player control |
| `tool-notifications` | Notifications | Test notifications |

**Route Type:**
```kotlin
sealed class Screen(val route: String, val label: String) {
    data object Daily : Screen("daily", "Daily")
    data object Files : Screen("files", "Files")
    data object Sleep : Screen("sleep", "Sleep")
    data object Tools : Screen("tools", "Tools")
    data object Settings : Screen("settings", "Settings")
    data object ToolClaude : Screen("tool-claude", "Claude")
    data object ToolNoise : Screen("tool-noise", "Noise")
    data object ToolNotifications : Screen("tool-notifications", "Notifications")
}
```

**Navigation Behavior:**
- Start destination: `Settings` if no person selected, `Daily` otherwise
- Bottom navigation: Daily, Files, Sleep, Tools (hidden until person is set)
- Top-level navigation preserves back stack by popping to existing screens

---

### Configuration

#### AppConfig

Static configuration constants for the application.

**File:** `AppConfig.kt`

**Constants:**

| Constant | Type | Description |
|----------|------|-------------|
| `AUTH_TOKEN` | `String` | Bearer token for API authentication |
| `BASE_URLS` | `List<String>` | Ordered list of server URLs for failover |

**Example:**
```kotlin
object AppConfig {
    const val AUTH_TOKEN = "VJY9EoAf1xx1bO-LaduCmItwRitCFm9BPuQZ8jd0tcg"
    val BASE_URLS = listOf(
        "http://192.168.1.27:8000",   // Local network
        "http://100.87.83.30:8000"    // Tailscale
    )
}
```

**Usage Notes:**
- URLs are tried sequentially on connection failure
- Both local and VPN addresses enable connectivity in different network contexts

---

### State Management

#### UserSettings

Persistent user preferences stored in SharedPreferences.

**File:** `UserSettings.kt`

**Type:** Singleton object

**State Properties:**

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `person` | `String?` | `null` | Selected person context (`"sebastian"` or `"petra"`) |
| `theme` | `String` | `"dark"` | Theme preference (`"dark"` or `"light"`) |

**Public API:**

```kotlin
object UserSettings {
    var person by mutableStateOf<String?>(null)
    var theme by mutableStateOf("dark")

    fun init(context: Context)
    fun updatePerson(value: String)
    fun updateTheme(value: String)
}
```

| Method | Parameters | Description |
|--------|------------|-------------|
| `init` | `Context` | Load settings from SharedPreferences |
| `updatePerson` | `String` | Update person and persist to storage |
| `updateTheme` | `String` | Update theme and persist to storage |

**Implementation Notes:**
- Uses `mutableStateOf` for Compose reactivity
- SharedPreferences key: `notes_settings`
- Changes trigger immediate recomposition across the app

---

#### ClaudeSessionStore

In-memory state for the Claude AI chat session.

**File:** `ClaudeSessionStore.kt`

**Type:** Singleton object

**State Properties:**

| Property | Type | Description |
|----------|------|-------------|
| `sessionId` | `String?` | Current chat session ID from server |
| `messages` | `SnapshotStateList<ChatMessage>` | Chat history |

**Public API:**

```kotlin
object ClaudeSessionStore {
    var sessionId by mutableStateOf<String?>(null)
    val messages = mutableStateListOf<ChatMessage>()

    fun clear()
}
```

| Method | Description |
|--------|-------------|
| `clear` | Reset sessionId to null and clear all messages |

**Implementation Notes:**
- Session persists only in memory (lost on app restart)
- `mutableStateListOf` enables reactive list updates in Compose
- Server assigns `sessionId` on first message; subsequent messages include it for continuity

---

### Network Layer

#### ApiClient

HTTP client for all REST API communication.

**File:** `ApiClient.kt`

**Type:** Singleton object

**HTTP Clients:**

| Client | Read Timeout | Purpose |
|--------|--------------|---------|
| Standard | 30 seconds | Normal API requests |
| Streaming | Infinite | Claude chat streaming |

**Authentication:**

All requests include:
- `Authorization: Bearer <AUTH_TOKEN>`
- `X-Notes-Person: <UserSettings.person>` (when person is set)

**Multi-Server Failover:**

Requests try `BASE_URLS` in order until one succeeds:
1. First URL (local network)
2. Second URL (Tailscale VPN)

**Public API:**

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `fetchDaily` | - | `DailyNote` | Get today's daily note |
| `saveDaily` | `content: String` | `ApiMessage` | Save entire daily note |
| `appendDaily` | `content: String, pinned: Boolean` | `ApiMessage` | Append timestamped entry |
| `addTodo` | `category: String` | `ApiMessage` | Add empty todo item |
| `toggleTodo` | `path: String, line: Int` | `ApiMessage` | Toggle todo checkbox |
| `clearPinned` | - | `ApiMessage` | Remove all pinned markers |
| `unpinEntry` | `path: String, line: Int` | `ApiMessage` | Unpin specific entry |
| `fetchSleepTimes` | - | `SleepTimesResponse` | Get recent sleep entries |
| `appendSleepTimes` | `child: String, entry: String, asleep: Boolean, woke: Boolean` | `ApiMessage` | Add sleep entry |
| `deleteSleepEntry` | `line: Int` | `ApiMessage` | Delete sleep entry |
| `listFiles` | `path: String` | `FilesResponse` | List directory contents |
| `readFile` | `path: String` | `FileReadResponse` | Read file content |
| `createFile` | `path: String` | `ApiMessage` | Create empty file |
| `saveFile` | `path: String, content: String` | `ApiMessage` | Save file content |
| `deleteFile` | `path: String` | `ApiMessage` | Delete file |
| `claudeChat` | `message: String, sessionId: String?` | `ClaudeChatResponse` | Non-streaming chat |
| `claudeChatStream` | `message: String, sessionId: String?` | `Flow<ClaudeStreamEvent>` | Streaming chat |
| `claudeClear` | `sessionId: String` | `ApiMessage` | Clear chat session |
| `fetchEnv` | - | `EnvResponse` | Get .env file content |
| `saveEnv` | `content: String` | `ApiMessage` | Save .env file content |

**Exception Handling:**

```kotlin
class ApiHttpException(val code: Int, message: String) : Exception(message)
```

Thrown on HTTP error responses (non-2xx status codes).

---

### Data Models

#### Models

All data transfer objects used for API communication.

**File:** `Models.kt`

**Serialization:** Kotlinx Serialization (`@Serializable`)

**Models:**

```kotlin
@Serializable
data class DailyNote(
    val date: String,      // "YYYY-MM-DD"
    val path: String,      // Relative file path
    val content: String    // Markdown content
)

@Serializable
data class SleepTimesResponse(
    val entries: List<SleepEntry>
)

@Serializable
data class SleepEntry(
    @SerialName("line_no")
    val lineNo: Int,       // 1-indexed line number
    val text: String       // Full line text
)

@Serializable
data class FilesResponse(
    val entries: List<FileEntry>
)

@Serializable
data class FileEntry(
    val name: String,      // File/directory name
    val path: String,      // Relative path
    @SerialName("is_dir")
    val isDir: Boolean
)

@Serializable
data class FileReadResponse(
    val path: String,
    val content: String
)

@Serializable
data class ApiMessage(
    val success: Boolean = true,
    val message: String = ""
)

@Serializable
data class EnvResponse(
    val success: Boolean,
    val content: String = "",
    val message: String = ""
)

@Serializable
data class ChatMessage(
    val role: String,      // "user" or "assistant"
    val content: String
)

@Serializable
data class ClaudeChatResponse(
    val success: Boolean,
    val message: String = "",
    val response: String = "",
    @SerialName("session_id")
    val sessionId: String = "",
    val history: List<ChatMessage> = emptyList()
)

@Serializable
data class ClaudeStreamEvent(
    val type: String,              // "text", "tool_use", "session", "ping", "error"
    val delta: String? = null,
    val name: String? = null,
    val input: JsonElement? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
    val message: String? = null
)
```

---

### Theming

#### Theme System

Custom theme system providing consistent styling across the app.

**File:** `Theme.kt`

**Theme Components:**

| Component | Description |
|-----------|-------------|
| `AppColors` | Color palette for UI elements |
| `AppSpacing` | Standard spacing values |
| `AppTypography` | Text styles using JetBrains Mono font |

**AppColors Properties:**

| Property | Description |
|----------|-------------|
| `background` | Main screen background |
| `panel` | Panel/card background |
| `panelBorder` | Panel border color |
| `text` | Primary text color |
| `muted` | Secondary/dimmed text |
| `accent` | Primary accent color |
| `accentDim` | Dimmed accent for backgrounds |
| `danger` | Error/delete actions |
| `input` | Input field background |
| `note` | Note content background |
| `button` | Button background |
| `buttonText` | Button text color |
| `checkboxFill` | Checked checkbox fill |

**AppSpacing Values:**

| Property | Value | Usage |
|----------|-------|-------|
| `xs` | 6dp | Tight spacing |
| `sm` | 10dp | Small gaps |
| `md` | 14dp | Medium spacing |
| `lg` | 18dp | Large gaps |
| `xl` | 24dp | Section spacing |

**AppTypography Styles:**

| Style | Description |
|-------|-------------|
| `body` | Standard body text |
| `bodySmall` | Smaller body text |
| `title` | Screen titles |
| `section` | Section headers |
| `label` | Form labels |

**Theme Access:**

```kotlin
// Via CompositionLocal
val colors = LocalAppColors.current
val spacing = LocalAppSpacing.current
val typography = LocalAppTypography.current
```

**Theme Switching:**

The `NotesEditorTheme` composable selects light or dark color scheme based on `UserSettings.theme`.

---

### UI Components

#### Reusable Components

Common UI components used across screens.

**File:** `UiComponents.kt`

**Components:**

| Component | Purpose |
|-----------|---------|
| `AppText` | Styled text with theme typography |
| `ScreenTitle` | Screen title header |
| `SectionTitle` | Section header within screens |
| `ScreenLayout` | Standard screen container (optional scroll) |
| `CompactDivider` | Thin horizontal divider |
| `CompactButton` | Primary button style |
| `CompactTextButton` | Text-only button |
| `CompactTextField` | Text input with focus styling |
| `AppCheckbox` | Themed checkbox |
| `Panel` | Card-like container with border |
| `NoteSurface` | Container for note content |
| `MessageBadge` | Inline status message |
| `StatusMessage` | Full-width status indicator |

---

#### NoteView

Markdown note renderer with interactive tasks.

**File:** `NoteView.kt`

**Features:**
- Parses markdown into line types: H1, H2, H3, H4, TASK, TEXT, EMPTY
- Renders headings with appropriate typography
- Interactive task checkboxes (calls `onToggleTask` callback)
- Supports task toggle by line number

**Line Type Detection:**

| Pattern | Type |
|---------|------|
| `# ` | H1 |
| `## ` | H2 |
| `### ` | H3 |
| `#### ` | H4 |
| `- [ ]` or `- [x]` | TASK |
| Empty line | EMPTY |
| Other | TEXT |

---

### Screens

#### DailyScreen

Today's daily note with todos and append functionality.

**Features:**
- Displays parsed markdown note with NoteView
- Add todo buttons (work/priv categories)
- Append note section (with optional pinned flag)
- Pull-to-refresh

---

#### FilesScreen

File browser with tree navigation.

**Features:**
- Directory listing with navigation
- File content viewing and editing
- Create and delete files
- Breadcrumb path display

---

#### SleepTimesScreen

Sleep time tracking for children.

**Features:**
- Recent sleep entries list
- Add new entry (child, time, asleep/woke flags)
- Delete entries

---

#### ToolsScreen

Navigation hub to tool sub-screens.

**Features:**
- Grid/list of available tools
- Navigation to Claude, Noise, Notifications screens

---

#### SettingsScreen

Application settings and configuration.

**Features:**
- Person selection (sebastian/petra)
- Theme toggle (dark/light)
- Environment file editor

---

#### ToolClaudeScreen

Claude AI chat interface with streaming.

**Features:**
- Message history display
- Text input for new messages
- Streaming response rendering
- Clear session button

---

#### ToolNoiseScreen

White noise player control.

**Features:**
- Play/stop controls
- Volume adjustment

---

#### ToolNotificationsScreen

Test notification functionality.

**Features:**
- Send test notifications
- Verify notification permissions

---

## Navigation Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    App Start                                 │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ Person set?   │
              └───────┬───────┘
                      │
         ┌────────────┴────────────┐
         │ No                      │ Yes
         ▼                         ▼
┌─────────────────┐       ┌─────────────────┐
│    Settings     │       │      Daily      │
│    Screen       │       │     Screen      │
└─────────────────┘       └─────────────────┘
                                  │
                          ┌───────┴───────┐
                          │ Bottom Nav    │
                          └───────────────┘
                                  │
         ┌────────────┬───────────┼───────────┬────────────┐
         ▼            ▼           ▼           ▼            │
    ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │
    │  Daily  │ │  Files  │ │  Sleep  │ │  Tools  │        │
    └─────────┘ └─────────┘ └─────────┘ └────┬────┘        │
                                             │             │
                          ┌──────────────────┼─────────┐   │
                          ▼                  ▼         ▼   │
                    ┌──────────┐      ┌─────────┐ ┌──────┐ │
                    │  Claude  │      │  Noise  │ │Notif.│ │
                    └──────────┘      └─────────┘ └──────┘ │
                                                           │
                                                           ▼
                                                    ┌───────────┐
                                                    │ Settings  │
                                                    │ (via nav) │
                                                    └───────────┘
```

**Navigation Rules:**
1. Bottom nav items (Daily, Files, Sleep, Tools) preserve their back stacks
2. Tool sub-screens push onto the Tools back stack
3. Settings is accessible but not shown in bottom nav when person is selected
4. When person is null, only Settings screen is accessible

---

## State Management

### State Scopes

| Scope | Mechanism | Lifetime | Example |
|-------|-----------|----------|---------|
| App-wide persistent | `UserSettings` + SharedPreferences | Survives app restart | Person, theme |
| App-wide in-memory | Singleton objects | Process lifetime | ClaudeSessionStore |
| Screen-local | `remember { mutableStateOf() }` | Screen composition | Loading states, form inputs |
| Request-scoped | Local variables | Single request | API response data |

### Reactive Updates

State changes propagate automatically through Compose:

```
UserSettings.person changes
        │
        ▼
mutableStateOf triggers recomposition
        │
        ▼
NotesEditorApp recomposes
        │
        ├── Navigation start destination updates
        ├── Bottom nav visibility updates
        └── ApiClient headers update (next request)
```

---

## Error Handling

### Network Errors

| Scenario | Handling |
|----------|----------|
| Server unreachable | Try next URL in BASE_URLS |
| All servers failed | Show error message to user |
| HTTP error (4xx/5xx) | Throw `ApiHttpException`, display message |
| Timeout | Retry with next server |

### API Error Responses

```kotlin
// Check success field in response
val response = ApiClient.saveDaily(content)
if (!response.success) {
    showError(response.message)
}
```

### Screen Error States

Screens typically maintain error state:

```kotlin
var error by remember { mutableStateOf<String?>(null) }

// In LaunchedEffect or event handler
try {
    // API call
    error = null
} catch (e: Exception) {
    error = e.message
}

// In UI
error?.let { StatusMessage(it, isError = true) }
```

---

## Dependencies

### Core Dependencies

| Dependency | Purpose |
|------------|---------|
| Jetpack Compose | UI framework |
| Compose Navigation | Screen navigation |
| OkHttp | HTTP client |
| Kotlinx Serialization | JSON parsing |
| Kotlin Coroutines | Async operations |

### Compose Dependencies

| Dependency | Purpose |
|------------|---------|
| `compose.ui` | Core UI primitives |
| `compose.material3` | Material Design components |
| `compose.foundation` | Layout and gestures |
| `navigation-compose` | Navigation controller |

### Custom Font

JetBrains Mono is bundled for monospace text rendering throughout the app.

---

## Future Considerations

1. **Offline Support:** Cache daily note for offline viewing
2. **ViewModel Migration:** Move screen state to ViewModels for better lifecycle handling
3. **Dependency Injection:** Consider Hilt/Koin for testability
4. **Error Retry:** Automatic retry with exponential backoff
5. **Push Notifications:** Server-sent notifications for note updates
