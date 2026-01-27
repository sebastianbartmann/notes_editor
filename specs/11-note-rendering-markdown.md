# Note Rendering and Markdown Parsing Specification

> Status: Draft
> Version: 2.0
> Last Updated: 2026-01-27

## Overview

This document specifies the parallel implementations for rendering markdown notes with interactive elements on Android and React Web platforms. Both platforms share the same line-by-line parsing logic to ensure consistent note display across the application.

**Implementation:** Android (`clients/android/`), React (`clients/web/src/components/NoteView/`)

The rendering system parses markdown content into discrete line objects, classifies each line by type (heading, task, text, empty), and renders them with platform-appropriate styling and interactivity.

### Design Principles

1. **Consistency**: Both platforms recognize the same markdown patterns
2. **Line-Based**: Each line is parsed independently (no multi-line blocks)
3. **Interactive**: Task checkboxes are clickable and trigger API calls
4. **Read-Only Display**: Rendered view is for reading; editing uses raw textarea

---

## Line Types

Both platforms use an enumerated set of line types to classify parsed content.

### LineType Enum

| Value | Description |
|-------|-------------|
| `H1` | Level 1 heading (`# `) |
| `H2` | Level 2 heading (`## `) |
| `H3` | Level 3 heading (`### `) |
| `H4` | Level 4 heading (`#### `) |
| `TASK` | Checkbox task item (`- [ ]` or `- [x]`) |
| `TEXT` | Plain text line |
| `EMPTY` | Blank/whitespace-only line |

### Android Implementation

```kotlin
private enum class LineType {
    H1, H2, H3, H4, TASK, TEXT, EMPTY
}

private data class NoteLine(
    val lineNo: Int,      // 1-indexed line number
    val text: String,     // Display text (prefix stripped)
    val type: LineType,
    val done: Boolean = false  // For TASK type only
)
```

### React/TypeScript Implementation

The React implementation uses explicit line type classification similar to Android:

```typescript
enum LineType {
  H1, H2, H3, H4, TASK, TEXT, EMPTY
}

interface NoteLine {
  lineNo: number;    // 1-indexed line number
  text: string;      // Display text (prefix stripped)
  type: LineType;
  done?: boolean;    // For TASK type only
}
```

---

## Parsing Rules

### Regex Patterns

| Pattern | Platform | Purpose |
|---------|----------|---------|
| `^\s*-\s*\[([ xX])\]\s*(.*)$` | Android | Task detection |
| `^\s*-\s*\[([ xX])\]\s*(.*)$` | Web | Task detection |
| `^(#{1,6})\s+(.*)$` | Web | Heading detection (all levels) |
| `^(###\s+.*<pinned>.*)$` | Web | Pinned heading detection |

### Priority Order

Lines are matched in this order (first match wins):

1. **Task Line**: Matches `- [ ]` or `- [x]` pattern
2. **Heading H4**: Starts with `#### `
3. **Heading H3**: Starts with `### `
4. **Heading H2**: Starts with `## `
5. **Heading H1**: Starts with `# `
6. **Empty**: Blank or whitespace-only
7. **Text**: Default for all other lines

### Android Parsing Function

```kotlin
private fun parseNoteLines(content: String): List<NoteLine> {
    val lines = content.lines()
    val items = mutableListOf<NoteLine>()
    val taskRegex = Regex("^\\s*-\\s*\\[( |x|X)\\]\\s*(.*)$")

    for ((idx, raw) in lines.withIndex()) {
        val lineNo = idx + 1
        val trimmed = raw.trimEnd()

        if (trimmed.isBlank()) {
            items.add(NoteLine(lineNo, "", LineType.EMPTY))
            continue
        }

        val taskMatch = taskRegex.find(trimmed)
        if (taskMatch != null) {
            val marker = taskMatch.groupValues[1]
            val text = taskMatch.groupValues[2].ifBlank { trimmed }
            items.add(NoteLine(
                lineNo = lineNo,
                text = text,
                type = LineType.TASK,
                done = marker.equals("x", ignoreCase = true)
            ))
            continue
        }

        when {
            trimmed.startsWith("#### ") -> items.add(NoteLine(lineNo, trimmed.removePrefix("#### ").trim(), LineType.H4))
            trimmed.startsWith("### ") -> items.add(NoteLine(lineNo, trimmed.removePrefix("### ").trim(), LineType.H3))
            trimmed.startsWith("## ") -> items.add(NoteLine(lineNo, trimmed.removePrefix("## ").trim(), LineType.H2))
            trimmed.startsWith("# ") -> items.add(NoteLine(lineNo, trimmed.removePrefix("# ").trim(), LineType.H1))
            else -> items.add(NoteLine(lineNo, trimmed, LineType.TEXT))
        }
    }
    return items
}
```

### Web Parsing Patterns

```python
PINNED_HEADING = re.compile(r"^(###\s+.*<pinned>.*)$", re.IGNORECASE)
HEADING_LINE = re.compile(r"^(#{1,6})\s+(.*)$")
TASK_LINE = re.compile(r"^\s*-\s*\[([ xX])\]\s*(.*)$")
```

---

## Android Rendering

### NoteView Composable

```kotlin
@Composable
fun NoteView(
    content: String,
    onToggleTask: (Int) -> Unit,  // Callback with 1-indexed line number
    modifier: Modifier = Modifier
)
```

### Styling by Line Type

| LineType | Typography | Color | Additional |
|----------|------------|-------|------------|
| `EMPTY` | `bodySmall` | `text` | Single space character |
| `H1` | `title` + `SemiBold` | `accent` | - |
| `H2` | `section` + `SemiBold` | `muted` | Text uppercased |
| `H3` | `section` + `SemiBold` | `accent` | - |
| `H4` | `bodySmall` + `SemiBold` | `text` | - |
| `TASK` | `body` | `text` or `muted` (if done) | Checkbox + clickable row |
| `TEXT` | `body` | `text` | - |

### Task Rendering

Tasks are rendered as a clickable `Row` containing:
- `AppCheckbox` reflecting the `done` state
- `AppText` with task text (muted color if completed)

Clicking anywhere on the row triggers `onToggleTask(lineNo)`.

---

## Web Rendering

### Render Function

```python
def render_with_pinned_buttons(content: str, file_path: str) -> str
```

**Parameters:**
- `content`: Raw markdown content
- `file_path`: Relative file path (used in forms for API calls)

**Returns:** HTML string with rendered note lines

### HTML Output Structure

Each line becomes a `<div class="note-line">` with additional classes based on type:

```html
<!-- Empty line -->
<div class="note-line empty">&nbsp;</div>

<!-- Plain text -->
<div class="note-line">Escaped text content</div>

<!-- Heading (h1-h6) -->
<div class="note-line heading h{level}">
  <span class="heading-text">{hashes} {text}</span>
</div>

<!-- Task (unchecked) -->
<div class="note-line task-line">
  <form class="inline-form">
    <input type="hidden" name="line" value="{line_number}">
    <input type="hidden" name="path" value="{file_path}">
    <input type="checkbox" hx-post="/api/todos/toggle" ...>
  </form>
  <span class="task-text">{text}</span>
</div>

<!-- Task (checked) -->
<div class="note-line task-line done">
  <form class="inline-form">
    <input type="hidden" name="line" value="{line_number}">
    <input type="hidden" name="path" value="{file_path}">
    <input type="checkbox" checked hx-post="/api/todos/toggle" ...>
  </form>
  <span class="task-text">{text}</span>
</div>
```

---

## Interactive Elements

### Task Toggle (Both Platforms)

**Android:**
- Clicking task row calls `onToggleTask(lineNo)`
- Parent screen (DailyScreen/FilesScreen) handles API call
- API endpoint: `POST /api/todos/toggle`

**Web:**
- Checkbox triggers HTMX POST on change
- Endpoint: `POST /api/todos/toggle`
- Form includes hidden `line` and `path` fields

### HTMX Attributes (Web)

```html
<input type="checkbox"
  hx-post="/api/todos/toggle"
  hx-trigger="change"
  hx-target="#message"
  hx-swap="innerHTML"
  hx-include="closest form">
```

---

## Pinned Entry Handling

The `<pinned>` marker is a web-specific feature for carrying forward important notes between daily entries.

### Detection Pattern

```python
PINNED_HEADING = re.compile(r"^(###\s+.*<pinned>.*)$", re.IGNORECASE)
```

Matches H3 headings containing `<pinned>` anywhere in the line (case-insensitive).

### Pinned Heading HTML

```html
<div class="note-line note-heading pinned heading h3">
  <span class="line-text heading-text">{escaped_line}</span>
  <form class="pin-form"
    hx-post="/api/files/unpin"
    hx-target="#message"
    hx-swap="innerHTML">
    <input type="hidden" name="path" value="{file_path}">
    <input type="hidden" name="line" value="{line_number}">
    <button class="pin-action" type="submit">Unpin</button>
  </form>
</div>
```

### Unpin Button

- Visible only on pinned H3 headings
- Calls `POST /api/files/unpin` with path and line number
- Removes `<pinned>` marker from the line

### Android Pinned Handling

Android does not render the Unpin button. The `<pinned>` marker appears as part of the heading text. Users can clear all pinned markers via the "Clear" button which calls `POST /api/clear-pinned`.

---

## CSS Classes

### Container Classes

| Class | Element | Purpose |
|-------|---------|---------|
| `note-view` | Container div | Wrapper for entire rendered note |

### Line Classes

| Class | Element | Purpose |
|-------|---------|---------|
| `note-line` | `<div>` | Base class for all lines |
| `empty` | `<div>` | Empty/blank line |
| `heading` | `<div>` | Any heading line |
| `h1` - `h6` | `<div>` | Heading level modifier |
| `task-line` | `<div>` | Task/checkbox line |
| `done` | `<div>` | Completed task modifier |
| `note-heading` | `<div>` | Heading with special styling |
| `pinned` | `<div>` | Pinned heading modifier |

### Text Classes

| Class | Element | Purpose |
|-------|---------|---------|
| `heading-text` | `<span>` | Heading text content |
| `line-text` | `<span>` | General line text |
| `task-text` | `<span>` | Task description text |

### Form Classes

| Class | Element | Purpose |
|-------|---------|---------|
| `inline-form` | `<form>` | Task toggle form |
| `pin-form` | `<form>` | Unpin button form |
| `pin-action` | `<button>` | Unpin button styling |

### CSS Variable Dependencies

```css
--text: #e6e6e6;      /* Default text color */
--muted: #9aa0a6;     /* Secondary/completed text */
--accent: #d9832b;    /* Heading highlights */
--panel-border: #2a2d33;
--note: #101317;      /* Note background */
```

---

## Integration Notes

### DailyScreen (Android)

```kotlin
NoteView(
    content = content,
    onToggleTask = { lineNo ->
        if (path.isNotBlank()) {
            scope.launch {
                val response = ApiClient.toggleTodo(path, lineNo)
                message = response.message
                refresh()
            }
        }
    },
    modifier = Modifier.fillMaxWidth().weight(1f)
)
```

- Displays today's daily note content
- Task toggles call API and refresh content
- Edit mode switches to raw textarea

### FilesScreen (Android)

```kotlin
NoteView(
    content = fileContent,
    onToggleTask = {},  // No-op for file browser
    modifier = Modifier.fillMaxWidth().weight(1f)
)
```

- Displays arbitrary file content
- Task toggle is disabled (empty callback)
- Read-only preview before editing

### Web Templates

The web uses `render_with_pinned_buttons()` in templates:

```python
# In route handler
rendered_content = render_with_pinned_buttons(content, file_path)
return templates.TemplateResponse("daily.html", {
    "rendered_content": rendered_content,
    ...
})
```

---

## Platform Differences

| Feature | Android | Web |
|---------|---------|-----|
| Heading levels | H1-H4 only | H1-H6 |
| Pinned detection | Not parsed specially | Regex with Unpin button |
| Task toggle | Row click | Checkbox change event |
| H2 uppercase | Applied in render | CSS `text-transform` |
| Heading prefix | Stripped from display | Preserved in HTML |
| Empty line | Space character | `&nbsp;` entity |
| XSS protection | N/A (native) | HTML escaping via `escape()` |

### Heading Level Support

Android explicitly handles H1-H4 via `startsWith()` checks. Lines starting with `##### ` or `###### ` fall through to TEXT type.

Web uses a single regex `^(#{1,6})\s+(.*)$` supporting all six levels, though only H1-H4 have distinct CSS styling defined.

### Text Transformation

Android applies `.uppercase()` to H2 text in the Composable. Web achieves the same via CSS `text-transform: uppercase` on `.heading.h2 .heading-text`.

### Security

Web implementation uses `html.escape()` for all user content to prevent XSS. Android renders via native Compose text components which handle escaping inherently.
