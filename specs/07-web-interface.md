# Web Interface Specification (Legacy)

> Status: **SUPERSEDED**
> Superseded By: [20-react-web-client.md](./20-react-web-client.md)
> Version: 1.0
> Last Updated: 2026-01-18

## Deprecation Notice

**This specification describes the legacy Python/HTMX web interface which has been replaced by a React/Vite single-page application.** See [20-react-web-client.md](./20-react-web-client.md) for the current web client specification.

The legacy implementation is preserved here for historical reference only.

---

## Overview (Legacy)

The Notes Editor web interface was a server-rendered HTML application that provided a browser-based UI for managing daily notes, files, sleep tracking, and Claude AI chat. The interface used HTMX for dynamic interactions without full page reloads, Jinja2 templates for server-side rendering, and vanilla JavaScript for complex client-side behaviors.

---

## Architecture

### Technology Stack
| Component | Technology |
|-----------|------------|
| Web Framework | FastAPI |
| Templating | Jinja2 |
| Dynamic Interactions | HTMX |
| Styling | Custom CSS with theme support |
| Client Logic | Vanilla JavaScript |

### Request Flow
```
Browser Request
     │
     ▼
FastAPI Route Handler
     │
     ▼
Authentication Check
     │
     ▼
Business Logic (vault operations)
     │
     ▼
Jinja2 Template Rendering
     │
     ▼
HTML Response
     │
     ▼
HTMX Partial Updates (optional)
```

### Directory Structure
```
server/web_app/
├── main.py              # Route handlers
├── templates/           # Jinja2 templates
│   ├── editor.html
│   ├── files.html
│   ├── file_page.html
│   ├── file_editor.html
│   ├── llm.html
│   ├── noise.html
│   ├── settings.html
│   ├── sleep_times.html
│   └── tools.html
└── renderers/           # HTML rendering utilities
    ├── file_tree.py
    └── pinned.py
```

---

## Templates

### editor.html - Daily Notes Editor
**Route:** `GET /`

The main page for viewing and editing today's daily note.

**Features:**
- Displays rendered note with NoteView rendering
- Add todo buttons (work/priv categories)
- Edit mode toggle for raw markdown editing
- Append form with optional pin checkbox
- Clear pinned button for bulk unpin

**Key Elements:**
| Element | Purpose |
|---------|---------|
| `#note-view` | Rendered note display |
| `#edit-form` | Hidden raw markdown editor |
| `#append-form` | Form to add timestamped entries |
| `#message` | HTMX response message display |

### files.html - File Browser
**Route:** `GET /files`

Directory browser with lazy-loading tree structure.

**Features:**
- Create file form
- Lazy-loading directory tree via HTMX
- Click-to-expand directories
- File links open in editor

### sleep_times.html - Sleep Tracking
**Route:** `GET /sleep-times`

Tracks sleep and wake times for children.

**Features:**
- Child selection (Thomas/Fabian radio buttons)
- Status checkboxes (eingeschlafen/aufgewacht)
- Entry list with delete buttons
- Time input field

**Key Elements:**
| Element | Purpose |
|---------|---------|
| `#sleep-message` | HTMX response display |
| `.entry-list` | Recent sleep entries |

### settings.html - Settings Page
**Route:** `GET /settings`, `POST /settings`

Application configuration interface.

**Features:**
- Person selection (sebastian/petra)
- Theme selection (dark/light)
- `.env` file editor for environment variables

### tools.html - Tools Hub
**Route:** `GET /tools`

Navigation page for utility tools.

**Tools Grid:**
- Files - File browser
- Claude - AI chat interface
- Sleep Times - Sleep tracking
- Noise - White noise controls
- Settings - Configuration

### llm.html - Claude Chat Interface
**Route:** `GET /tools/llm`

Streaming chat interface for Claude AI.

**Features:**
- Real-time streaming responses via fetch API
- Session management (new/continue/clear)
- Status indicators for tool use
- Message history display

### noise.html - White Noise Controls
**Route:** `GET /tools/noise`

Audio playback controls for white noise.

### file_page.html / file_editor.html - File Editing
**Route:** `GET /file`

Individual file viewing and editing.

**Features:**
- Raw content display
- Save/delete operations
- Back navigation to file browser

---

## Renderers

### file_tree.py

Renders directory listings as interactive HTML trees with HTMX support.

**Function:**
```python
def render_tree(entries: list[dict]) -> str
```

**Input:** List of file entries with `name`, `path`, `is_dir` fields.

**Output HTML Structure:**

Directory entry:
```html
<div class="tree-item dir">
    <button class="tree-toggle" data-target="children-{id}"
            hx-get="/api/files/tree?path={encoded_path}"
            hx-target="#children-{id}" hx-swap="innerHTML">+</button>
    <span class="tree-name">{name}/</span>
</div>
<div id="children-{id}" class="tree-children"></div>
```

File entry:
```html
<a class="tree-item file" href="/file?path={encoded_path}">
    <span class="tree-name">{name}</span>
</a>
```

**Helper Function:**
```python
def _safe_id(path: str) -> str
```
Generates HTML-safe IDs by replacing non-alphanumeric characters.

### pinned.py

Renders markdown notes as interactive HTML with checkboxes and unpin buttons.

**Function:**
```python
def render_with_pinned_buttons(content: str, file_path: str) -> str
```

**Regex Patterns:**
| Pattern | Purpose |
|---------|---------|
| `^(###\s+.*<pinned>.*)$` | Pinned heading detection (case-insensitive) |
| `^(#{1,6})\s+(.*)$` | Any markdown heading |
| `^\s*-\s*\[([ xX])\]\s*(.*)$` | Task line with checkbox |

**Output HTML by Line Type:**

Pinned heading:
```html
<div class="note-line note-heading pinned heading h3">
    <span class="line-text heading-text">{escaped_text}</span>
    <form class="pin-form" hx-post="/api/files/unpin" hx-target="#message" hx-swap="innerHTML">
        <input type="hidden" name="path" value="{file_path}">
        <input type="hidden" name="line" value="{line_no}">
        <button class="pin-action" type="submit">Unpin</button>
    </form>
</div>
```

Regular heading:
```html
<div class="note-line heading h{level}">
    <span class="heading-text">{hashes} {text}</span>
</div>
```

Task line:
```html
<div class="note-line task-line [done]">
    <form class="inline-form">
        <input type="hidden" name="line" value="{line_no}">
        <input type="hidden" name="path" value="{file_path}">
        <input type="checkbox" [checked] hx-post="/api/todos/toggle"
               hx-trigger="change" hx-target="#message" hx-swap="innerHTML"
               hx-include="closest form">
    </form>
    <span class="task-text">{text}</span>
</div>
```

Empty line:
```html
<div class="note-line empty">&nbsp;</div>
```

Text line:
```html
<div class="note-line">{escaped_text}</div>
```

---

## HTMX Patterns

### Form Submission with Target
Standard pattern for form actions that display status messages.

```html
<form hx-post="/api/endpoint" hx-target="#message" hx-swap="innerHTML">
    <input type="hidden" name="field" value="value">
    <button type="submit">Action</button>
</form>
```

### Lazy Loading (File Tree)
Loads child content on demand when directory is expanded.

```html
<button hx-get="/api/files/tree?path={encoded}"
        hx-target="#children-{id}"
        hx-swap="innerHTML">+</button>
<div id="children-{id}" class="tree-children"></div>
```

### Checkbox Change Trigger
Immediately submits form when checkbox state changes.

```html
<input type="checkbox"
       hx-post="/api/todos/toggle"
       hx-trigger="change"
       hx-target="#message"
       hx-swap="innerHTML"
       hx-include="closest form">
```

### After-Request Handling
JavaScript event listener for post-HTMX processing.

```javascript
document.body.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.xhr.responseURL.includes('/api/endpoint')) {
        // Show message, reload page, update UI
    }
});
```

---

## JavaScript Behaviors

### Edit Mode Toggle (editor.html)

Switches between rendered view and raw markdown editing.

```javascript
function toggleEditMode() {
    document.getElementById('note-view').style.display = 'none';
    document.getElementById('edit-form').style.display = 'block';
    sessionStorage.setItem('editMode', 'true');
}

function cancelEdit() {
    document.getElementById('note-view').style.display = 'block';
    document.getElementById('edit-form').style.display = 'none';
    sessionStorage.removeItem('editMode');
}
```

**Session Persistence:** Uses `sessionStorage` flag to re-open edit mode after page reload.

### File Tree Expansion (files.html)

Handles directory expand/collapse behavior.

**Behavior:**
- Click on tree toggle button expands directory
- Prevents re-fetching already loaded directories
- Toggles +/- indicator on expand/collapse

### Streaming Chat (llm.html)

Implements real-time streaming responses from Claude.

**Implementation:**
```javascript
// Uses fetch API with ReadableStream
const response = await fetch('/api/claude/chat-stream', {
    method: 'POST',
    body: formData
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
    const {done, value} = await reader.read();
    if (done) break;

    const chunk = decoder.decode(value);
    // Parse NDJSON events
    const lines = chunk.split('\n');
    for (const line of lines) {
        if (line.trim()) {
            const event = JSON.parse(line);
            // Handle event type: text, tool_use, session, ping, error
        }
    }
}
```

### Mutual Exclusion (sleep_times.html)

Ensures only one status checkbox can be selected.

```javascript
// eingeschlafen and aufgewacht checkboxes are mutually exclusive
asleepCheckbox.addEventListener('change', function() {
    if (this.checked) wokeCheckbox.checked = false;
});
wokeCheckbox.addEventListener('change', function() {
    if (this.checked) asleepCheckbox.checked = false;
});
```

---

## Navigation Structure

### Page Routes

| Route | Method | Template | Description |
|-------|--------|----------|-------------|
| `/` | GET | editor.html | Daily notes editor (main page) |
| `/login` | GET/POST | - | Authentication |
| `/settings` | GET/POST | settings.html | Settings page |
| `/files` | GET | files.html | File browser |
| `/file` | GET | file_page.html | File editor |
| `/tools` | GET | tools.html | Tools hub |
| `/tools/llm` | GET | llm.html | Claude chat |
| `/tools/noise` | GET | noise.html | White noise |
| `/sleep-times` | GET | sleep_times.html | Sleep tracking |

### Navigation Flow
```
         ┌─────────────────┐
         │   Login Page    │
         └────────┬────────┘
                  │ auth
                  ▼
         ┌─────────────────┐
         │  Daily Editor   │ ◄─────────────────┐
         │       (/)       │                   │
         └────────┬────────┘                   │
                  │                            │
         ┌────────┴────────┐                   │
         ▼                 ▼                   │
┌─────────────────┐ ┌─────────────┐           │
│   Tools Hub     │ │   Settings  │           │
│   (/tools)      │ │ (/settings) │           │
└────────┬────────┘ └─────────────┘           │
         │                                     │
    ┌────┼────┬────────┬──────────┐           │
    ▼    ▼    ▼        ▼          ▼           │
 Files  LLM  Sleep   Noise      Back ─────────┘
```

---

## Theme System

### CSS Class Application
Theme is applied via CSS class on the `<body>` element.

```html
<body class="theme-dark">  <!-- or theme-light -->
```

### Available Themes
| Theme | Class | Description |
|-------|-------|-------------|
| Dark | `theme-dark` | Default theme, dark background |
| Light | `theme-light` | Light background variant |

### Theme Storage
- Stored in HTTP cookie: `theme=dark` or `theme=light`
- Read by Jinja template context as `theme_class`
- Changeable via Settings page

### Theme Application (Jinja)
```jinja2
<body class="{{ theme_class }}">
```

---

## Authentication

### Methods
1. **Bearer Token**: `Authorization: Bearer <token>` header
2. **Cookie-Based**: Session cookie for browser requests

### Person Context
- Stored in cookie: `person=sebastian` or `person=petra`
- Passed to templates via Jinja context
- Required by most route handlers via `require_auth()` dependency

### Auth Flow
```
Request
   │
   ▼
┌──────────────────┐
│ Check Auth Token │
│ or Session Cookie│
└────────┬─────────┘
         │
    ┌────┴────┐
    │ Valid?  │
    └────┬────┘
    Yes  │  No
    │    │
    ▼    ▼
Continue  Redirect
 Route    to /login
```

---

## Common UI Patterns

### CSS Classes

| Class | Purpose |
|-------|---------|
| `.panel` | Card container with border/shadow |
| `.topbar` | Navigation bar with brand and context info |
| `.button` | Base button styling |
| `.button.ghost` | Transparent background button |
| `.button.ghost.danger` | Red ghost button for destructive actions |
| `.note-line` | Single line in rendered note |
| `.task-line` | Task item with checkbox |
| `.task-line.done` | Completed task (strikethrough) |
| `.tree-item` | File tree entry |
| `.tree-item.dir` | Directory entry |
| `.tree-item.file` | File entry |

### Status Messages
```html
<mark id="message"></mark>        <!-- General message target -->
<mark id="sleep-message"></mark>  <!-- Sleep times specific -->
```

HTMX responses populate these elements with success/error messages.

### Form Patterns
```html
<!-- Standard action form -->
<form hx-post="/api/action" hx-target="#message" hx-swap="innerHTML">
    <input type="hidden" name="path" value="{{ path }}">
    <button type="submit">Action</button>
</form>

<!-- Inline checkbox form -->
<form class="inline-form">
    <input type="hidden" name="context" value="...">
    <input type="checkbox" hx-post="/api/toggle"
           hx-trigger="change" hx-include="closest form">
</form>
```

---

## Integration Notes

### API Endpoints Used
The web interface calls these API endpoints via HTMX:

| Endpoint | Used By |
|----------|---------|
| `POST /api/save` | editor.html - save full note |
| `POST /api/append` | editor.html - add timestamped entry |
| `POST /api/todos/add` | editor.html - add empty todo |
| `POST /api/todos/toggle` | editor.html - checkbox toggle |
| `POST /api/clear-pinned` | editor.html - clear all pinned |
| `POST /api/files/unpin` | editor.html - unpin single entry |
| `GET /api/files/tree` | files.html - lazy load directory |
| `POST /api/files/create` | files.html - create new file |
| `POST /api/files/save-json` | file_page.html - save file |
| `POST /api/files/delete-json` | file_page.html - delete file |
| `POST /api/sleep-times/append` | sleep_times.html - add entry |
| `POST /api/sleep-times/delete` | sleep_times.html - remove entry |
| `POST /api/claude/chat-stream` | llm.html - streaming chat |
| `POST /api/claude/clear` | llm.html - clear session |

### Git Operations
All write operations trigger git commit/push through the API layer. The web interface does not interact with git directly.

### Vault Integration
File paths in the web interface are relative to the person's vault directory. The server handles path resolution and access control.

### Error Handling
- HTMX responses display errors in `#message` elements
- JavaScript alerts for critical errors
- Form validation via HTML5 attributes
- Server-side validation returns appropriate error messages
