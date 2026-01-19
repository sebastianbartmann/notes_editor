# Android Keyboard Accessory Bar Specification

> Status: Implemented
> Version: 1.1
> Last Updated: 2026-01-19

## Overview

This document specifies the implementation of a keyboard accessory bar for the Notes Editor Android application. The primary goal is to fix keyboard overlap issues where the soft keyboard covers input fields, making text invisible while typing. Additionally, a toolbar with special keys is added above the keyboard for improved text input efficiency.

## Problem Statement

The Android app had keyboard overlap issues:
- No `windowSoftInputMode` set in AndroidManifest
- No IME insets handling (only `navigationBars` insets were handled)
- When the soft keyboard appeared, it overlapped with text input fields
- This was particularly problematic in the Claude chat screen

## Requirements

### Functional Requirements

1. **Keyboard Overlap Fix**
   - App content must resize when soft keyboard appears
   - Text input fields must remain visible and accessible while typing
   - Applies to all screens with text input

2. **Keyboard Accessory Bar**
   - Display a toolbar with 7 keys when the soft keyboard is open
   - Keys:
     - Arrow keys: `↑`, `↓`, `←`, `→` (4 keys)
     - Special characters: `/`, `[`, `]` (3 keys)
   - Bar must be visible only when soft keyboard is open
   - Bar positioned above the keyboard, replacing bottom navigation

3. **Bottom Navigation Behavior**
   - Hide bottom info bar (title + arc menu) when keyboard is visible
   - Show bottom info bar when keyboard is hidden

---

## Technical Design

### Architecture Overview

When keyboard is hidden:
```
+----------------------------------+
|           App Content            |
|        (NavHost + Screens)       |
+----------------------------------+
|        Bottom Info Bar           |
+----------------------------------+
```

When keyboard is visible:
```
+----------------------------------+
|           App Content            |
|     (resized via imePadding)     |
+----------------------------------+
|      Keyboard Accessory Bar      |
|    [↑] [↓] [←] [→] [/] [[] []]   |
+----------------------------------+
|          Soft Keyboard           |
+----------------------------------+
```

### Key Implementation Details

#### 1. Edge-to-Edge Mode (Required)

Edge-to-edge mode must be enabled for `WindowInsets.ime` to be reported to the app. Without this, the system consumes IME insets and the app cannot detect or respond to keyboard visibility.

```kotlin
// MainActivity.kt
WindowCompat.setDecorFitsSystemWindows(window, false)
```

#### 2. AndroidManifest Configuration

```xml
<activity
    android:name=".MainActivity"
    android:windowSoftInputMode="adjustResize"
    ...>
```

Note: `adjustResize` alone does NOT resize the window when edge-to-edge is enabled. The app must use `imePadding()` to handle the resize.

#### 3. IME Padding on Main Layout

Apply `imePadding()` to the root Column so the entire layout moves above the keyboard:

```kotlin
Column(
    modifier = Modifier
        .fillMaxSize()
        .imePadding()  // Pushes content above keyboard
) {
    // Content, accessory bar, bottom nav
}
```

#### 4. Keyboard Visibility Detection

Use `WindowInsets.ime.getBottom(density)` to detect keyboard visibility:

```kotlin
val density = LocalDensity.current
val imeBottom = WindowInsets.ime.getBottom(density)
val isKeyboardVisible = imeBottom > 0
```

This only works when edge-to-edge mode is enabled.

#### 5. Input Injection via KeyCharacterMap

Text characters are injected using `KeyCharacterMap` which converts characters to key events:

```kotlin
private fun commitText(view: View, text: String) {
    val charMap = KeyCharacterMap.load(KeyCharacterMap.VIRTUAL_KEYBOARD)
    val events = charMap.getEvents(text.toCharArray())
    events?.forEach { event ->
        view.dispatchKeyEvent(event)
    }
}
```

Arrow keys use direct `KeyEvent` dispatch:

```kotlin
private fun sendKeyEvent(view: View, keyCode: Int) {
    val eventTime = SystemClock.uptimeMillis()
    view.dispatchKeyEvent(KeyEvent(eventTime, eventTime, KeyEvent.ACTION_DOWN, keyCode, 0))
    view.dispatchKeyEvent(KeyEvent(eventTime, eventTime, KeyEvent.ACTION_UP, keyCode, 0))
}
```

---

## Files Modified

| File | Changes |
|------|---------|
| `AndroidManifest.xml` | Added `android:windowSoftInputMode="adjustResize"` |
| `MainActivity.kt` | Added `WindowCompat.setDecorFitsSystemWindows(window, false)` |
| `AppNavigation.kt` | Added `imePadding()` to Column, hide bottom nav when keyboard visible |
| `KeyboardAccessoryBar.kt` | New file with accessory bar composable |

---

## Key Learnings

1. **Edge-to-edge is required** for `WindowInsets.ime` to work. Without it, IME insets are consumed by the system.

2. **Don't mix adjustResize with imePadding incorrectly** - with edge-to-edge enabled, `adjustResize` doesn't automatically resize the window. You must use `imePadding()` to handle the resize manually.

3. **imePadding placement matters** - apply it to the root container so all content moves up together.

4. **KeyCharacterMap is reliable** for injecting text characters across different keyboard layouts.

---

## Dependencies

- Android API 31+ (minSdk)
- AndroidX Core library (for WindowCompat)
- Compose Foundation (for imePadding, WindowInsets)

---

## Related Specifications

- `16-android-bottom-info-bar.md`
