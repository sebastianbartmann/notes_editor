# Add Task Inline Input

> Status: Implemented
> Version: 1.0
> Last Updated: 2026-01-19

## Overview

Replace the current "add empty task then edit" workflow with an inline input flow where users type task text directly before adding. When clicking "Work task" or "Priv task", an input field appears allowing the user to type their task, then save or cancel.

## Current Behavior

**Android:**
- User clicks "Work task" or "Priv task" button
- System immediately calls `ApiClient.addTodo(category)`
- API adds blank `- [ ]` line to the note
- User must then manually edit the note to add task text

**Web:**
- HTMX form posts to `/api/todos/add` with only `category`
- API adds blank `- [ ]` line
- User must manually edit to add task text

## Target Behavior

**Both platforms:**
1. User clicks "Work task" or "Priv task"
2. UI transitions to input mode showing text field + Save + Cancel
3. User types task text (no markdown prefix needed)
4. On Save: system prepends `- [ ] ` and calls API with full task line
5. On Cancel: return to normal view without API call

---

## API Changes

### `POST /api/todos/add`

**Updated Request Body** (form-encoded):

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `category` | string | Yes | `"work"` or `"priv"` |
| `text` | string | No | Task text (blank if not provided) |

**Behavior Change:**
- If `text` is provided and non-empty: adds `- [ ] {text}` line
- If `text` is empty or not provided: adds `- [ ]` line (backwards compatible)

**Response (200):**
```json
{
  "success": true,
  "message": "Task added"
}
```

### Server Implementation

```python
@app.post("/api/todos/add")
async def add_todo(
    category: str = Form(...),
    text: str = Form("")  # Optional, defaults to empty
):
    if category not in ("work", "priv"):
        raise HTTPException(400, "Invalid category")

    task_line = f"- [ ] {text}" if text.strip() else "- [ ]"
    # ... insert task_line under appropriate category section
```

---

## Android Implementation

### UI States

```
┌─────────────────────────────────────┐
│  NORMAL STATE                       │
│  [Edit] [Work task] [Priv task]     │
└─────────────────────────────────────┘
           ↓ click task button
┌─────────────────────────────────────┐
│  INPUT STATE                        │
│  [________________________] [✓] [✕] │
│   TextField              Save Cancel│
└─────────────────────────────────────┘
```

### State Management

Add to DailyScreen.kt:

```kotlin
// New state variables
var taskInputMode by remember { mutableStateOf<String?>(null) } // null, "work", or "priv"
var taskInputText by remember { mutableStateOf("") }
```

### Component: TaskInputRow

```kotlin
@Composable
fun TaskInputRow(
    onSave: (String) -> Unit,
    onCancel: () -> Unit
) {
    var text by remember { mutableStateOf("") }

    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically
    ) {
        TextField(
            value = text,
            onValueChange = { text = it },
            modifier = Modifier.weight(1f),
            placeholder = { Text("Task description") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            keyboardActions = KeyboardActions(onDone = { onSave(text) })
        )
        IconButton(onClick = { onSave(text) }) {
            Icon(Icons.Default.Check, "Save")
        }
        IconButton(onClick = onCancel) {
            Icon(Icons.Default.Close, "Cancel")
        }
    }
}
```

### DailyScreen.kt Changes

Replace button click handlers:

```kotlin
// Current buttons section becomes conditional
if (taskInputMode != null) {
    TaskInputRow(
        onSave = { text ->
            scope.launch {
                try {
                    val response = ApiClient.addTodo(taskInputMode!!, text)
                    message = response.message
                    refresh(keepEditing = false)
                } catch (e: Exception) {
                    message = "Error: ${e.message}"
                }
            }
            taskInputMode = null
            taskInputText = ""
        },
        onCancel = {
            taskInputMode = null
            taskInputText = ""
        }
    )
} else {
    // Existing buttons
    CompactButton(text = "Edit") { isEditing = true }
    CompactTextButton(text = "Work task") { taskInputMode = "work" }
    CompactTextButton(text = "Priv task") { taskInputMode = "priv" }
}
```

### ApiClient.kt Changes

```kotlin
suspend fun addTodo(category: String, text: String = ""): ApiMessage =
    postForm("/api/todos/add", mapOf("category" to category, "text" to text))
```

---

## Web Implementation

### HTML Changes (editor.html)

Replace the HTMX forms with JS-driven elements:

```html
<!-- Normal state -->
<div id="task-buttons" class="actions view-actions">
    <button id="edit-btn" class="button" onclick="toggleEditMode()">Edit</button>
    <button class="button ghost" onclick="showTaskInput('work')">Work task</button>
    <button class="button ghost" onclick="showTaskInput('priv')">Priv task</button>
</div>

<!-- Input state (hidden by default) -->
<div id="task-input" class="task-input-row" style="display: none;">
    <input type="text" id="task-text" placeholder="Task description" class="task-input">
    <button class="button" onclick="saveTask()">Save</button>
    <button class="button ghost" onclick="cancelTaskInput()">Cancel</button>
</div>
```

### JavaScript

```javascript
let currentTaskCategory = null;

function showTaskInput(category) {
    currentTaskCategory = category;
    document.getElementById('task-buttons').style.display = 'none';
    document.getElementById('task-input').style.display = 'flex';
    document.getElementById('task-text').value = '';
    document.getElementById('task-text').focus();
}

function cancelTaskInput() {
    currentTaskCategory = null;
    document.getElementById('task-input').style.display = 'none';
    document.getElementById('task-buttons').style.display = 'flex';
}

function saveTask() {
    const text = document.getElementById('task-text').value;

    fetch('/api/todos/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
            category: currentTaskCategory,
            text: text
        })
    })
    .then(response => response.json())
    .then(data => {
        document.getElementById('message').textContent = data.message;
        if (data.success) {
            location.reload();
        }
    })
    .catch(err => {
        document.getElementById('message').textContent = 'Error: ' + err.message;
    });

    cancelTaskInput();
}

// Handle Enter key in input
document.addEventListener('DOMContentLoaded', function() {
    const taskInput = document.getElementById('task-text');
    if (taskInput) {
        taskInput.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                saveTask();
            } else if (e.key === 'Escape') {
                cancelTaskInput();
            }
        });
    }
});
```

### CSS

```css
.task-input-row {
    display: flex;
    gap: 8px;
    align-items: center;
}

.task-input {
    flex: 1;
    padding: 8px 12px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 14px;
}
```

---

## UX Flow Diagram

```
                    ┌──────────────────┐
                    │   Daily View     │
                    │                  │
                    │ [Edit] [Work] [Priv]
                    └────────┬─────────┘
                             │
              ┌──────────────┴──────────────┐
              │ click "Work task"           │ click "Priv task"
              ▼                             ▼
    ┌─────────────────────┐      ┌─────────────────────┐
    │ INPUT MODE (work)   │      │ INPUT MODE (priv)   │
    │                     │      │                     │
    │ [___________] [✓][✕]│      │ [___________] [✓][✕]│
    └──────────┬──────────┘      └──────────┬──────────┘
               │                            │
    ┌──────────┴──────────┐      ┌──────────┴──────────┐
    │                     │      │                     │
    ▼ Save                ▼ Cancel
┌─────────┐          ┌─────────┐
│ API call│          │ Discard │
│ POST    │          │ input   │
│ /todos  │          └────┬────┘
│ /add    │               │
└────┬────┘               │
     │                    │
     ▼                    ▼
┌────────────────────────────────┐
│        Return to Normal View    │
│        (refresh on success)     │
└────────────────────────────────┘
```

---

## Files to Modify

| File | Changes |
|------|---------|
| `server/routes/api.py` | Add optional `text` param to `/api/todos/add` |
| `app/android/.../DailyScreen.kt` | Add input mode state, TaskInputRow composable |
| `app/android/.../ApiClient.kt` | Add `text` param to `addTodo()` |
| `server/web_app/templates/editor.html` | Replace HTMX forms with JS-driven input flow |
| `server/web_app/static/css/style.css` | Add `.task-input-row` and `.task-input` styles |

---

## Design Decisions

1. **Inline input vs modal**: Inline keeps context visible and feels lighter. Modal would obscure content unnecessarily.

2. **Text prepending on client vs server**: Server prepends `- [ ] ` prefix. Client sends plain text. This keeps markdown logic centralized and client code simpler.

3. **Backwards compatibility**: Empty `text` param produces same behavior as before (blank task line), so existing integrations continue to work.

4. **Enter/Escape key handling**: Web supports keyboard shortcuts for power users. Android uses IME action for the same.

5. **Auto-focus**: Input field receives focus immediately on show for quick typing.

---

## Related Specifications

- [01-rest-api-contract.md](./01-rest-api-contract.md) - API contract (update needed)
- [03-android-app-architecture.md](./03-android-app-architecture.md) - Android architecture
- [07-web-interface.md](./07-web-interface.md) - Web interface patterns
