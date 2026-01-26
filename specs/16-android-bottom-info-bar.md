# Android Bottom Info Bar

**Status:** Implemented
**Version:** 1.0
**Created:** 2026-01-19

## Overview

Replace per-screen top headers with a unified bottom info bar that combines the screen title (left) with the Arc Menu FAB (right). This creates a consistent bottom navigation area across all Android screens.

## Current State

```
┌─────────────────────────────────────┐
│ [Title]              [Action Btn]   │  ← Top header (per-screen)
├─────────────────────────────────────┤
│                                     │
│          Content Area               │
│                                     │
│                                     │
│                                     │
│                             [FAB]   │  ← Arc Menu (bottom-right)
└─────────────────────────────────────┘
```

## Target State

```
┌─────────────────────────────────────┐
│                                     │
│          Content Area               │
│     (includes any action buttons)   │
│                                     │
│                                     │
├─────────────────────────────────────┤
│ [View-specific input if any]        │  ← e.g., Claude chat input
├─────────────────────────────────────┤
│ Title                       [FAB]   │  ← Unified bottom bar
└─────────────────────────────────────┘
```

## Component Specifications

### Bottom Info Bar

**Location:** AppNavigation.kt (alongside existing ArcMenu)

**Layout:**
```
┌─────────────────────────────────────┐
│ Title                       [FAB]   │
│ ↑                             ↑     │
│ 16dp padding              56dp btn  │
│ left-aligned           16dp padding │
└─────────────────────────────────────┘
```

**Dimensions:**
- Height: 56dp (matches FAB height + padding alignment)
- Horizontal padding: 16dp (both sides)
- Title vertical alignment: center with FAB

**Behavior:**
- Hides when keyboard is visible (same as current FAB behavior)
- Title and FAB hide/show together as a unit

### Route-to-Title Mapping

| Route           | Display Title   |
|-----------------|-----------------|
| `daily`         | Daily           |
| `files`         | Files           |
| `claude`        | Claude          |
| `sleep`         | Sleep           |
| `settings`      | Settings        |
| `noise`         | Noise           |
| `notifications` | Notifications   |

### Title Styling

- Typography: MaterialTheme.typography.titleLarge
- Color: MaterialTheme.colorScheme.onSurface
- Max lines: 1
- Overflow: ellipsis

## Implementation Details

### AppNavigation.kt Changes

```kotlin
@Composable
fun BottomInfoBar(
    currentRoute: String,
    isKeyboardVisible: Boolean,
    onArcMenuClick: () -> Unit,
    arcMenuExpanded: Boolean
) {
    if (isKeyboardVisible) return

    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp)
    ) {
        // Title on left
        Text(
            text = routeToTitle(currentRoute),
            style = MaterialTheme.typography.titleLarge,
            modifier = Modifier.align(Alignment.CenterStart)
        )

        // FAB on right (existing ArcMenu)
        ArcMenu(
            expanded = arcMenuExpanded,
            onToggle = onArcMenuClick,
            modifier = Modifier.align(Alignment.CenterEnd)
        )
    }
}

private fun routeToTitle(route: String): String = when (route) {
    "daily" -> "Daily"
    "files" -> "Files"
    "claude" -> "Claude"
    "sleep" -> "Sleep"
    "settings" -> "Settings"
    "noise" -> "Noise"
    "notifications" -> "Notifications"
    else -> ""
}
```

### ScreenLayout.kt Changes

Remove top header support. Each screen now only provides content.

```kotlin
@Composable
fun ScreenLayout(
    content: @Composable () -> Unit
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Box(modifier = Modifier.weight(1f)) {
            content()
        }
        // Bottom bar handled by AppNavigation
    }
}
```

## Migration: Action Buttons

Action buttons currently in top headers move into content areas:

| Screen           | Current Button | Migration Target                    |
|------------------|----------------|-------------------------------------|
| DailyScreen      | Reload         | Pull-to-refresh or floating button  |
| FilesScreen      | Reload         | Pull-to-refresh or floating button  |
| ClaudeScreen     | Clear          | Menu option or input area button    |
| SleepTimesScreen | Reload         | Pull-to-refresh or floating button  |

### Pull-to-Refresh Pattern

```kotlin
SwipeRefresh(
    state = rememberSwipeRefreshState(isRefreshing),
    onRefresh = { viewModel.reload() }
) {
    // Screen content
}
```

### Alternative: Content Header

For screens needing visible buttons:

```kotlin
Row(
    modifier = Modifier
        .fillMaxWidth()
        .padding(horizontal = 16.dp, vertical = 8.dp),
    horizontalArrangement = Arrangement.End
) {
    IconButton(onClick = { /* action */ }) {
        Icon(Icons.Default.Refresh, "Reload")
    }
}
```

## Affected Files

1. `AppNavigation.kt` - Add BottomInfoBar, integrate with existing keyboard visibility logic
2. `ScreenLayout.kt` - Remove top header, simplify to content-only wrapper
3. `ArcMenu.kt` - Minor refactor to work within BottomInfoBar Row
4. `DailyScreen.kt` - Remove ScreenTitle, add pull-to-refresh
5. `FilesScreen.kt` - Remove ScreenTitle, add pull-to-refresh
6. `ClaudeScreen.kt` - Remove ScreenTitle, move Clear to input area
7. `SleepTimesScreen.kt` - Remove ScreenTitle, add pull-to-refresh
8. `SettingsScreen.kt` - Remove ScreenTitle
9. `NoiseScreen.kt` - Remove ScreenTitle
10. `NotificationsScreen.kt` - Remove ScreenTitle

## Keyboard Visibility

Existing logic in AppNavigation.kt handles keyboard detection. The BottomInfoBar uses the same `isKeyboardVisible` state to hide both title and FAB together.

## Edge Cases

1. **Empty route:** Display empty string (bar still shows with FAB only)
2. **Long titles:** Single line with ellipsis truncation
3. **Landscape mode:** Same layout, bar stretches full width
4. **Screen transitions:** Title updates immediately on route change
