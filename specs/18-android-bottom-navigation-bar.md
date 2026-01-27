# Android Bottom Navigation Bar Specification

> Status: Implemented
> Version: 1.1
> Last Updated: 2026-01-27

## Overview

This document specifies the bottom navigation bar for the Notes Editor Android app. This replaces the previous Arc Menu navigation (spec 15) with a simpler, always-visible bottom bar.

**Location:** `clients/android/`

**Related Specifications:**
- `03-android-app-architecture.md` - Overall app architecture
- `14-android-keyboard-accessory-bar.md` - Keyboard handling (keyboard accessory bar remains)

---

## Design Goals

1. **Always visible**: Navigation targets visible without extra tap
2. **Familiar**: Standard Android bottom navigation pattern
3. **Simple**: Direct access to primary screens, Tools hub for secondary

---

## User Interface Design

### Layout

```
+----------------------------------+
| [Title]              [Action Btn] |  <- Per-screen top header
+----------------------------------+
|                                   |
|          Screen Content           |
|                                   |
+----------------------------------+
| [Daily] [Files] [Sleep] [Tools]   |  <- Bottom navigation bar
+----------------------------------+
```

When keyboard is visible:
```
+----------------------------------+
|          Screen Content           |
|     (resized via imePadding)      |
+----------------------------------+
|      Keyboard Accessory Bar       |
|  [arrow] [arrow] [(] [)] [[] []]  |
+----------------------------------+
|          Soft Keyboard            |
+----------------------------------+
```

### Navigation Items

| Item | Icon | Route | Description |
|------|------|-------|-------------|
| Daily | CalendarToday | `daily` | Today's note |
| Files | Folder | `files` | File browser |
| Sleep | NightsStay | `sleep` | Sleep tracking |
| Tools | Build | `tools` | Hub for secondary screens |

### Tools Screen Items

The Tools screen provides navigation to secondary destinations:

| Item | Icon | Route | Description |
|------|------|-------|-------------|
| Claude | Chat | `tool-claude` | AI chat |
| Noise | VolumeUp | `tool-noise` | White noise player |
| Notifications | Notifications | `tool-notifications` | Test notifications |
| Settings | Settings | `settings` | App settings |
| Kiosk | OpenInNew | (external URL) | Opens browser |

---

## Visual Design

### Bottom Bar Specifications

| Property | Value |
|----------|-------|
| Height | 56dp |
| Background | `AppTheme.colors.panel` |
| Item layout | Equal width distribution |
| Icon size | 24dp |
| Label font size | 12sp |
| Active color | `AppTheme.colors.accent` |
| Inactive color | `AppTheme.colors.muted` |

### Per-Screen Top Headers

| Property | Value |
|----------|-------|
| Height | 56dp |
| Background | `AppTheme.colors.background` |
| Title style | `AppTheme.typography.title` |
| Horizontal padding | 16dp |
| Action button (optional) | Right-aligned IconButton |

---

## Behavior

### Keyboard Visibility

- **Hide** bottom navigation bar when keyboard is visible
- **Show** keyboard accessory bar above the keyboard (spec 14)
- **Show** bottom navigation bar when keyboard is hidden

### Navigation

- Tapping an item navigates to that screen
- Active item is highlighted with accent color
- Back stack is managed with `popUpTo` to prevent deep stacking

### No Person Selected

When `UserSettings.person` is null:
- Hide bottom navigation bar
- Only Settings screen is accessible via direct navigation

---

## Component Architecture

### BottomNavBar Composable

```kotlin
@Composable
fun BottomNavBar(
    currentRoute: String,
    onNavigate: (String) -> Unit,
    modifier: Modifier = Modifier
)
```

| Property | Description |
|----------|-------------|
| `currentRoute` | Current navigation route for active indicator |
| `onNavigate` | Callback for navigation |

### Navigation Items Configuration

```kotlin
data class BottomNavItem(
    val route: String,
    val label: String,
    val icon: ImageVector
)

val bottomNavItems = listOf(
    BottomNavItem("daily", "Daily", Icons.Default.CalendarToday),
    BottomNavItem("files", "Files", Icons.Default.Folder),
    BottomNavItem("sleep", "Sleep", Icons.Default.NightsStay),
    BottomNavItem("tools", "Tools", Icons.Default.Build)
)
```

### ScreenTitle Composable (Restore)

```kotlin
@Composable
fun ScreenTitle(
    title: String,
    actionButton: @Composable (() -> Unit)? = null
)
```

---

## File Structure

Files to modify:

```
app/android/app/src/main/java/com/bartmann/noteseditor/
├── AppNavigation.kt      # Remove Arc Menu, add BottomNavBar
├── ArcMenu.kt            # Delete (no longer needed)
├── ArcMenuConfig.kt      # Delete (no longer needed)
├── BottomInfoBar.kt      # Delete (no longer needed)
├── UiComponents.kt       # Restore ScreenTitle component
├── ToolsScreen.kt        # Restore Tools hub screen
├── DailyScreen.kt        # Add ScreenTitle back
├── FilesScreen.kt        # Add ScreenTitle back
├── SleepTimesScreen.kt   # Add ScreenTitle back
├── ClaudeScreen.kt       # Add ScreenTitle back
├── NoiseScreen.kt        # Add ScreenTitle back
├── NotificationsScreen.kt # Add ScreenTitle back
└── SettingsScreen.kt     # Add ScreenTitle back
```

---

## Migration from Arc Menu

| Old (Arc Menu) | New (Bottom Nav) |
|----------------|------------------|
| Tap FAB to reveal menu | Items always visible |
| Level 1: Daily, Files, Claude, More | Bottom bar: Daily, Files, Sleep, Tools |
| Level 2: Sleep, Noise, Notifications, Settings, Kiosk | Tools screen hub |
| Bottom info bar with title + FAB | Per-screen top headers |

---

## Dependencies

- Android API 31+ (minSdk)
- Jetpack Compose Navigation
- Material Icons

---

## Testing Considerations

### Manual Testing Checklist

- [ ] Bottom nav bar visible on all screens (when person selected)
- [ ] Active item highlighted correctly
- [ ] Navigation works for all items
- [ ] Tools screen shows all secondary items
- [ ] Bottom nav hides when keyboard visible
- [ ] Keyboard accessory bar still works
- [ ] Per-screen titles display correctly
- [ ] Action buttons work (reload, etc.)
