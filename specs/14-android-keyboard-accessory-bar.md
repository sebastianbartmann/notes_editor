# Android Keyboard Accessory Bar Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-19

## Overview

This document specifies the implementation of a keyboard accessory bar for the Notes Editor Android application. The primary goal is to fix keyboard overlap issues where the soft keyboard covers input fields, making text invisible while typing. Additionally, a toolbar with special keys will be added above the keyboard for improved text input efficiency.

## Problem Statement

Currently, the Android app has keyboard overlap issues:
- No `windowSoftInputMode` set in AndroidManifest
- No IME insets handling (only `navigationBars` insets are handled)
- When the soft keyboard appears, it overlaps with text input fields
- This is particularly problematic in the Claude chat screen

## Requirements

### Functional Requirements

1. **Keyboard Overlap Fix**
   - App content must resize when soft keyboard appears
   - Text input fields must remain visible and accessible while typing
   - Applies to all screens with text input

2. **Keyboard Accessory Bar**
   - Display a toolbar with 7 keys when the soft keyboard is open
   - Keys:
     - Arrow keys: `Up`, `Down`, `Left`, `Right` (4 keys)
     - Special characters: `/`, `[`, `]` (3 keys)
   - Bar must be visible only when soft keyboard is open
   - Bar positioned above the keyboard, below app content

3. **Affected Screens**
   - Claude chat
   - Daily notes
   - Files editor
   - Sleep times
   - Settings
   - UiComponents (CompactTextField used across screens)

### Non-Functional Requirements

- Smooth animations when keyboard appears/disappears
- Minimal visual footprint for the accessory bar
- No performance impact on text input responsiveness

---

## Technical Design

### Architecture Overview

```
+----------------------------------+
|           App Content            |
|        (NavHost + Screens)       |
+----------------------------------+
|      Keyboard Accessory Bar      |  <- Only visible when IME is open
|   [Up][Dn][Lt][Rt] [/] [[] []]   |
+----------------------------------+
|         Bottom Nav Bar           |
+----------------------------------+
|          Soft Keyboard           |  <- System IME
+----------------------------------+
```

### Key Components

#### 1. AndroidManifest Configuration

Add `windowSoftInputMode` to the main activity:

```xml
<activity
    android:name=".MainActivity"
    android:windowSoftInputMode="adjustResize"
    ...>
```

#### 2. Edge-to-Edge Display Setup

In `MainActivity.kt`, enable edge-to-edge mode:

```kotlin
override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)
    WindowCompat.setDecorFitsSystemWindows(window, false)
    // ... rest of setup
}
```

#### 3. IME Insets Handling

Apply `imePadding()` modifier to the main content container in `AppNavigation.kt`:

```kotlin
Column(
    modifier = Modifier
        .fillMaxSize()
        .imePadding()  // Add this
) {
    NavHost(...)
    BottomNavBar(...)
}
```

#### 4. KeyboardAccessoryBar Composable

New composable that:
- Detects keyboard visibility via `WindowInsets.isImeVisible`
- Renders a row of buttons for the 7 keys
- Injects characters/cursor movements into the focused TextField

```kotlin
@Composable
fun KeyboardAccessoryBar(
    modifier: Modifier = Modifier
) {
    val isKeyboardVisible = WindowInsets.isImeVisible

    AnimatedVisibility(visible = isKeyboardVisible) {
        Row(
            modifier = modifier
                .fillMaxWidth()
                .background(MaterialTheme.colorScheme.surfaceVariant)
                .padding(horizontal = 8.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            // Arrow keys
            AccessoryKey("Up") { sendKeyEvent(KeyEvent.KEYCODE_DPAD_UP) }
            AccessoryKey("Dn") { sendKeyEvent(KeyEvent.KEYCODE_DPAD_DOWN) }
            AccessoryKey("Lt") { sendKeyEvent(KeyEvent.KEYCODE_DPAD_LEFT) }
            AccessoryKey("Rt") { sendKeyEvent(KeyEvent.KEYCODE_DPAD_RIGHT) }

            Spacer(modifier = Modifier.width(16.dp))

            // Special characters
            AccessoryKey("/") { insertText("/") }
            AccessoryKey("[") { insertText("[") }
            AccessoryKey("]") { insertText("]") }
        }
    }
}
```

#### 5. Text Injection Mechanism

Two approaches for injecting input:

**Option A: InputConnection (Recommended)**
Use `BaseInputConnection` to send text/key events to the currently focused view:

```kotlin
private fun getCurrentInputConnection(context: Context): InputConnection? {
    val imm = context.getSystemService(Context.INPUT_METHOD_SERVICE) as InputMethodManager
    val focusedView = (context as? Activity)?.currentFocus
    return focusedView?.onCreateInputConnection(EditorInfo())
}

private fun insertText(text: String) {
    getCurrentInputConnection(context)?.commitText(text, 1)
}

private fun sendKeyEvent(keyCode: Int) {
    getCurrentInputConnection(context)?.apply {
        sendKeyEvent(KeyEvent(KeyEvent.ACTION_DOWN, keyCode))
        sendKeyEvent(KeyEvent(KeyEvent.ACTION_UP, keyCode))
    }
}
```

**Option B: Compose State Hoisting**
Pass `TextFieldValue` state up and inject characters directly:

```kotlin
@Composable
fun KeyboardAccessoryBar(
    textFieldValue: TextFieldValue,
    onValueChange: (TextFieldValue) -> Unit
) {
    // Insert character at cursor position
    fun insertChar(char: String) {
        val selection = textFieldValue.selection
        val newText = textFieldValue.text.replaceRange(
            selection.start, selection.end, char
        )
        onValueChange(TextFieldValue(
            text = newText,
            selection = TextRange(selection.start + char.length)
        ))
    }
}
```

### Layout Structure

Updated `AppNavigation.kt` layout:

```kotlin
@Composable
fun AppNavigation(...) {
    Scaffold(
        modifier = Modifier.imePadding(),
        bottomBar = {
            Column {
                KeyboardAccessoryBar()
                BottomNavBar(navController)
            }
        }
    ) { paddingValues ->
        NavHost(
            modifier = Modifier.padding(paddingValues),
            ...
        )
    }
}
```

---

## Files to Modify

| File | Changes |
|------|---------|
| `app/android/app/src/main/AndroidManifest.xml` | Add `android:windowSoftInputMode="adjustResize"` |
| `app/android/app/src/main/java/com/bartmann/noteseditor/MainActivity.kt` | Enable edge-to-edge with `WindowCompat.setDecorFitsSystemWindows(window, false)` |
| `app/android/app/src/main/java/com/bartmann/noteseditor/AppNavigation.kt` | Add `imePadding()`, integrate `KeyboardAccessoryBar` |
| `app/android/app/src/main/java/com/bartmann/noteseditor/ui/KeyboardAccessoryBar.kt` | New file: `KeyboardAccessoryBar` composable |

---

## Implementation Steps

### Phase 1: Fix Keyboard Overlap

1. Add `android:windowSoftInputMode="adjustResize"` to AndroidManifest.xml
2. Add `WindowCompat.setDecorFitsSystemWindows(window, false)` in MainActivity.onCreate()
3. Add `imePadding()` modifier to main content in AppNavigation.kt
4. Test that content scrolls/resizes correctly when keyboard appears

### Phase 2: Implement Keyboard Accessory Bar

5. Create `KeyboardAccessoryBar.kt` with basic UI (7 buttons in a row)
6. Implement keyboard visibility detection using `WindowInsets.isImeVisible`
7. Add `AnimatedVisibility` for smooth show/hide transitions
8. Integrate accessory bar into AppNavigation.kt layout

### Phase 3: Input Injection

9. Implement text character insertion for `/`, `[`, `]`
10. Implement arrow key simulation for cursor movement
11. Test input injection works with CompactTextField and other text fields

### Phase 4: Polish

12. Style the accessory bar to match app theme
13. Add touch feedback/ripple effects to buttons
14. Test across all 6 screens with text inputs
15. Verify keyboard + bar don't overlap app content

---

## Testing Considerations

### Manual Testing

1. **Keyboard Overlap**
   - Open each screen with text input
   - Tap input field to bring up keyboard
   - Verify content scrolls and input remains visible
   - Type long text to ensure cursor stays visible

2. **Accessory Bar Visibility**
   - Verify bar appears when keyboard opens
   - Verify bar disappears when keyboard closes
   - Test on different screen sizes

3. **Key Functionality**
   - Test `/`, `[`, `]` insert correct characters
   - Test arrow keys move cursor correctly
   - Test in single-line and multi-line text fields

4. **Screen-Specific Testing**
   - Claude chat: Test during message composition
   - Daily notes: Test in note editor
   - Files: Test in file content editor
   - Sleep times: Test in time input
   - Settings: Test in env content editor

### Edge Cases

- Rotating device with keyboard open
- Switching between apps while keyboard is visible
- Hardware keyboard connected (accessory bar should still work)
- Multi-window mode on tablets

---

## Dependencies

- Android API 30+ (for modern WindowInsets API)
- AndroidX Core library (for WindowCompat)
- Compose Foundation (for imePadding, WindowInsets)

---

## Notes

- The accessory bar implementation should be purely additive and not modify existing TextField behavior
- Arrow keys should work in both single-line and multi-line text fields
- Consider adding haptic feedback on key press for better UX
- Future enhancement: Make the key set configurable in settings
