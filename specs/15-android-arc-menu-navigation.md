# Android Arc Menu Navigation Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-19

## Overview

This document specifies the replacement of the current static 4-button bottom navigation bar with an ergonomic arc menu positioned for right-hand thumb use in the bottom-right corner of the screen. The arc menu provides a two-level hierarchical navigation system that consolidates all navigation items including those currently accessed via the Tools screen.

**Related Specifications:**
- `03-android-app-architecture.md` - Overall app architecture and navigation
- `14-android-keyboard-accessory-bar.md` - Keyboard handling patterns to reuse

---

## Problem Statement

The current bottom navigation has several limitations:

1. **Limited space**: Only 4 items fit comfortably in a horizontal bar
2. **Tools screen indirection**: Secondary navigation items (Claude, Noise, Notifications, Settings, Kiosk) require navigating to a Tools hub screen first
3. **Thumb reach**: Bottom bar items at the left edge are difficult to reach with the right thumb on large phones
4. **Visual weight**: The full-width bottom bar takes up significant screen real estate

---

## Design Goals

1. **Ergonomic**: Optimized for right-hand single-thumb operation
2. **Efficient**: Direct access to all screens without intermediate hub
3. **Minimal**: Small footprint when collapsed; overlay pattern when expanded
4. **Familiar**: Arc/radial menu is a known mobile pattern (Facebook Messenger, Pinterest, etc.)

---

## User Interface Design

### Menu States

The menu has three states:

| State | Description |
|-------|-------------|
| `collapsed` | Single floating action button visible in bottom-right corner |
| `level1` | Primary arc expanded showing 4 items: Daily, Files, Claude, More |
| `level2` | Secondary arc showing 5 items: Sleep, Noise, Notifications, Settings, Kiosk |

### State Diagram

```
                    tap menu button
    ┌─────────────┐ ──────────────────► ┌─────────────┐
    │  collapsed  │                     │   level1    │
    └─────────────┘ ◄────────────────── └──────┬──────┘
                    tap outside              │
                    tap screen item          │ tap "More"
                                             ▼
                                       ┌─────────────┐
                                       │   level2    │
                                       └──────┬──────┘
                                              │ tap outside
                                              │ tap item
                                              ▼
                                       ┌─────────────┐
                                       │  collapsed  │
                                       └─────────────┘
```

### Visual Layout

#### Collapsed State
```
                              │
                              │
                              │ screen content
                              │
                              │
                              ├─────────────────────────┐
                              │                     ╭───╮│
                              │                     │ ☰ ││ ← menu button
                              │                     ╰───╯│
                              └─────────────────────────┘
```

#### Level 1 Expanded
```
                              │
                              │
                    ╭─────╮   │
                    │Daily│   │
                    ╰──┬──╯   │
               ╭─────╮ │      │
               │Files│─┼─────╮│
               ╰─────╯ │     ││
                 ╭─────┴╮  ╭─┴──╮
                 │Claude│  │More│ ← distinct styling
                 ╰──────╯  ╰────╯
                              │
                              └─────────────────────────┘
```
Items fan out from the bottom-right corner along an arc, positioned for natural thumb sweep from bottom-right toward top-left.

#### Level 2 Expanded
```
                              │
                    ╭─────╮   │
                    │Sleep│   │
                    ╰──┬──╯   │
               ╭─────╮ │      │
               │Noise│─┼─────╮│
               ╰─────╯ │     ││
            ╭───────╮╭─┴────╮││
            │Notifs ││ Sett ││ │
            ╰───────╯╰──────╯││
                 ╭─────╮  ╭──┴─╮
                 │Kiosk│  │Back│ ← returns to level1
                 ╰─────╯  ╰────╯
                              │
                              └─────────────────────────┘
```

### Visual Design Specifications

| Property | Value | Notes |
|----------|-------|-------|
| Arc center point | Bottom-right corner of screen | Origin for polar coordinates |
| Arc radius | 130dp | Adjustable; comfortable thumb reach |
| Arc start angle | 180° | Left direction |
| Arc end angle | 90° | Top direction |
| Menu button size | 56dp | Standard FAB size |
| Menu item touch target | 48dp minimum | Accessibility requirement |
| Menu item icon size | 24dp | Material Design standard |
| Label font size | 10sp | Small but legible |
| Shadow elevation | 6dp | Floating appearance |
| Item spacing | Equal angular distribution | Based on item count |

### Color and Styling

| Element | Light Theme | Dark Theme |
|---------|-------------|------------|
| Menu button background | `accent` | `accent` |
| Menu button icon | `buttonText` | `buttonText` |
| Item background | `panel` | `panel` |
| Item icon (inactive) | `muted` | `muted` |
| Item icon (active) | `accent` | `accent` |
| Item label | `text` | `text` |
| "More" item background | `accentDim` | `accentDim` |
| Scrim overlay | `#00000000` to `#33000000` | Optional semi-transparent backdrop |

### Animation Specifications

| Animation | Duration | Easing | Description |
|-----------|----------|--------|-------------|
| Expand (collapsed → level1) | 150ms | FastOutSlowIn | Items fan out sequentially from button position |
| Collapse (level1 → collapsed) | 150ms | FastOutSlowIn | Reverse of expand |
| Level transition (level1 ↔ level2) | 200ms | FastOutSlowIn | Crossfade items in place or rotate arc |
| Item press feedback | 100ms | Standard | Scale down to 0.95 then back |
| Active indicator | 200ms | LinearOutSlowIn | Smooth highlight transition |

---

## Menu Configuration

### Data Model

```kotlin
/**
 * Represents a single menu item in the arc menu.
 *
 * @param id Unique identifier for the item
 * @param icon Material icon to display
 * @param label Text label shown below icon
 * @param route Navigation route (null for submenu triggers like "More")
 * @param externalUrl URL to open in browser (for items like Kiosk)
 * @param children Nested items for submenu (only "More" uses this)
 */
data class ArcMenuItem(
    val id: String,
    val icon: ImageVector,
    val label: String,
    val route: String?,
    val externalUrl: String? = null,
    val children: List<ArcMenuItem>? = null
)
```

### Menu Structure

```kotlin
val arcMenuItems = listOf(
    ArcMenuItem(
        id = "daily",
        icon = Icons.Default.CalendarToday,
        label = "Daily",
        route = "daily"
    ),
    ArcMenuItem(
        id = "files",
        icon = Icons.Default.Folder,
        label = "Files",
        route = "files"
    ),
    ArcMenuItem(
        id = "claude",
        icon = Icons.Default.Chat,
        label = "Claude",
        route = "tool-claude"
    ),
    ArcMenuItem(
        id = "more",
        icon = Icons.Default.MoreHoriz,
        label = "More",
        route = null,
        children = listOf(
            ArcMenuItem(
                id = "sleep",
                icon = Icons.Default.NightsStay,
                label = "Sleep",
                route = "sleep"
            ),
            ArcMenuItem(
                id = "noise",
                icon = Icons.Default.VolumeUp,
                label = "Noise",
                route = "tool-noise"
            ),
            ArcMenuItem(
                id = "notifications",
                icon = Icons.Default.Notifications,
                label = "Notifications",
                route = "tool-notifications"
            ),
            ArcMenuItem(
                id = "settings",
                icon = Icons.Default.Settings,
                label = "Settings",
                route = "settings"
            ),
            ArcMenuItem(
                id = "kiosk",
                icon = Icons.Default.OpenInNew,
                label = "Kiosk",
                route = null,
                externalUrl = "https://thirdpartycheck.com/admin/kiosk"
            )
        )
    )
)
```

---

## Layout Geometry

### Polar Coordinate System

Items are positioned using polar coordinates with the arc center at the bottom-right corner of the screen:

```kotlin
/**
 * Calculate the position of an arc menu item.
 *
 * @param index Item index (0 = leftmost position)
 * @param itemCount Total items in current level
 * @param radius Distance from corner to item center
 * @param startAngle Arc start angle in degrees (180 = left)
 * @param sweepAngle Total arc sweep in degrees (90 = quarter circle)
 */
fun calculateItemPosition(
    index: Int,
    itemCount: Int,
    radius: Dp,
    startAngle: Float = 180f,
    sweepAngle: Float = 90f
): Offset {
    // Angular spacing between items
    val angleStep = sweepAngle / (itemCount - 1).coerceAtLeast(1)
    val angle = startAngle - (index * angleStep)
    val angleRadians = Math.toRadians(angle.toDouble())

    return Offset(
        x = (radius.toPx() * cos(angleRadians)).toFloat(),
        y = (radius.toPx() * sin(angleRadians)).toFloat()
    )
}
```

### Item Positioning Example

For 4 items (level 1) with 90° sweep from 180° to 90°:

| Item | Index | Angle | Position relative to corner |
|------|-------|-------|----------------------------|
| Daily | 0 | 180° | Left (x = -130dp, y = 0) |
| Files | 1 | 150° | Upper-left diagonal |
| Claude | 2 | 120° | Upper diagonal |
| More | 3 | 90° | Top (x = 0, y = -130dp) |

---

## Component Architecture

### Component Hierarchy

```
AppNavigation
└── ArcMenuHost
    ├── NavHost (screen content)
    ├── ScrimOverlay (when expanded)
    └── ArcMenu
        ├── ArcMenuButton (collapsed trigger)
        └── ArcMenuItems (when expanded)
            └── ArcMenuItem (repeated)
```

### Components to Create

#### ArcMenuButton

Floating action button that triggers menu expansion.

```kotlin
@Composable
fun ArcMenuButton(
    isExpanded: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
)
```

| Property | Description |
|----------|-------------|
| `isExpanded` | Controls icon animation (hamburger ↔ close) |
| `onClick` | Callback when button is tapped |

**Behavior:**
- Shows hamburger/menu icon when collapsed
- Shows close (X) icon when expanded
- Animates icon rotation during state change
- 56dp diameter with 6dp elevation shadow

#### ArcMenuItem

Individual menu item with icon and label.

```kotlin
@Composable
fun ArcMenuItem(
    item: ArcMenuItem,
    isActive: Boolean,
    position: Offset,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
)
```

| Property | Description |
|----------|-------------|
| `item` | Menu item data (icon, label, etc.) |
| `isActive` | Whether this is the current screen |
| `position` | Calculated position offset from corner |
| `onClick` | Callback when item is tapped |

**Behavior:**
- Displays icon centered above label
- Uses accent color when active
- Touch target minimum 48dp
- Press animation (scale feedback)
- "More" item has distinct background styling

#### ArcMenu

Container managing the arc menu state and positioning.

```kotlin
@Composable
fun ArcMenu(
    items: List<ArcMenuItem>,
    currentRoute: String?,
    menuState: ArcMenuState,
    onStateChange: (ArcMenuState) -> Unit,
    onNavigate: (String) -> Unit,
    onOpenExternal: (String) -> Unit,
    modifier: Modifier = Modifier
)
```

| Property | Description |
|----------|-------------|
| `items` | Root menu items configuration |
| `currentRoute` | Current navigation route for active indicator |
| `menuState` | Current menu state (collapsed/level1/level2) |
| `onStateChange` | Callback to update menu state |
| `onNavigate` | Callback for internal navigation |
| `onOpenExternal` | Callback for external URL opening |

### State Management

```kotlin
enum class ArcMenuState {
    COLLAPSED,
    LEVEL1,
    LEVEL2
}

// State hoisting in AppNavigation
var menuState by remember { mutableStateOf(ArcMenuState.COLLAPSED) }
```

### Components to Modify

#### AppNavigation.kt

| Change | Description |
|--------|-------------|
| Remove `BottomNavBar` composable | No longer needed |
| Remove bottom nav bar rendering | Including the Box wrapper |
| Add `ArcMenu` composable call | Positioned in bottom-right |
| Update navigation items | Reference new route structure |
| Add scrim overlay | When menu is expanded |

#### Remove ToolsScreen.kt

The Tools screen becomes obsolete as all its navigation targets are now directly accessible from the arc menu.

| Item | Old Path | New Path |
|------|----------|----------|
| Claude | Tools → Claude | Arc Menu → Claude |
| Noise | Tools → Noise | Arc Menu → More → Noise |
| Notifications | Tools → Notifications | Arc Menu → More → Notifications |
| Settings | Tools → Settings | Arc Menu → More → Settings |
| Kiosk | Tools → Kiosk | Arc Menu → More → Kiosk |

---

## Interaction Handling

### Tap Outside to Close

```kotlin
// In ArcMenu when expanded
if (menuState != ArcMenuState.COLLAPSED) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .pointerInput(Unit) {
                detectTapGestures {
                    onStateChange(ArcMenuState.COLLAPSED)
                }
            }
    ) {
        // Optional scrim
    }
}
```

### Navigation Flow

```kotlin
fun handleItemTap(item: ArcMenuItem) {
    when {
        // Submenu trigger
        item.children != null -> {
            onStateChange(ArcMenuState.LEVEL2)
        }
        // External link
        item.externalUrl != null -> {
            onOpenExternal(item.externalUrl)
            onStateChange(ArcMenuState.COLLAPSED)
        }
        // Internal navigation
        item.route != null -> {
            onNavigate(item.route)
            onStateChange(ArcMenuState.COLLAPSED)
        }
    }
}
```

### Back Navigation

When in Level 2, a "Back" item or back gesture should return to Level 1:

```kotlin
// Option A: Add Back item to level 2
val level2Items = (moreItem.children ?: emptyList()) + ArcMenuItem(
    id = "back",
    icon = Icons.Default.ArrowBack,
    label = "Back",
    route = null // Special handling
)

// Option B: Handle back press
BackHandler(enabled = menuState == ArcMenuState.LEVEL2) {
    menuState = ArcMenuState.LEVEL1
}
```

---

## Keyboard Handling

### Reuse Existing Pattern

The arc menu should hide when the soft keyboard is visible, using the same detection mechanism from the keyboard accessory bar:

```kotlin
val density = LocalDensity.current
val imeBottom = WindowInsets.ime.getBottom(density)
val isKeyboardVisible = imeBottom > 0

// In the ArcMenu visibility check
if (!isKeyboardVisible && person != null) {
    ArcMenu(...)
}
```

This ensures:
1. Arc menu doesn't overlap with keyboard
2. Keyboard accessory bar remains visible and functional
3. Consistent behavior with existing implementation

---

## Accessibility

### Content Descriptions

```kotlin
ArcMenuItem(
    // ...
    contentDescription = "${item.label} navigation"
)

ArcMenuButton(
    // ...
    contentDescription = if (isExpanded) "Close menu" else "Open navigation menu"
)
```

### Touch Targets

All interactive elements must meet the 48dp minimum touch target requirement:

```kotlin
Box(
    modifier = Modifier
        .sizeIn(minWidth = 48.dp, minHeight = 48.dp)
        .clickable { onClick() }
) {
    // Icon and label content
}
```

### Screen Reader Support

- Menu items announced with label + "button" role
- Active item announced with "selected" state
- Menu state changes announced ("menu expanded", "menu collapsed")

---

## Edge Cases

### Small Screens

On screens where the arc would extend off-screen, reduce the radius:

```kotlin
val screenWidth = LocalConfiguration.current.screenWidthDp.dp
val screenHeight = LocalConfiguration.current.screenHeightDp.dp
val maxRadius = minOf(screenWidth, screenHeight) * 0.4f
val radius = minOf(130.dp, maxRadius)
```

### Landscape Orientation

In landscape, the arc might need adjustment:
- Consider horizontal arc (spreading items upward)
- Or maintain same pattern but verify touch targets don't overlap screen edges

### No Person Selected

When `UserSettings.person` is null:
- Hide the arc menu entirely (same as current bottom nav behavior)
- Only Settings screen is accessible via direct navigation

### Concurrent Animations

If user rapidly taps between states, cancel in-progress animations and start the new target state:

```kotlin
val animatable = remember { Animatable(0f) }

LaunchedEffect(menuState) {
    animatable.animateTo(
        targetValue = when (menuState) {
            ArcMenuState.COLLAPSED -> 0f
            ArcMenuState.LEVEL1 -> 1f
            ArcMenuState.LEVEL2 -> 2f
        },
        animationSpec = tween(150)
    )
}
```

---

## File Structure

New and modified files:

```
app/android/app/src/main/java/com/bartmann/noteseditor/
├── AppNavigation.kt          # Modified: Remove bottom nav, add arc menu
├── ArcMenu.kt                 # New: ArcMenu, ArcMenuButton, ArcMenuItem
├── ArcMenuConfig.kt           # New: Menu configuration data
└── ToolsScreen.kt             # Deleted: No longer needed
```

---

## Testing Considerations

### Unit Tests

| Test | Description |
|------|-------------|
| Position calculation | Verify items positioned correctly on arc |
| State transitions | All valid state transitions work |
| Invalid states | Graceful handling of edge cases |

### UI Tests

| Test | Description |
|------|-------------|
| Expand/collapse | Menu animates correctly |
| Navigation | Tapping item navigates to correct screen |
| Active indicator | Correct item highlighted based on current screen |
| External link | Kiosk opens browser |
| Keyboard hiding | Menu hides when keyboard visible |

### Manual Testing Checklist

- [ ] Test on various screen sizes (small phone, large phone, tablet)
- [ ] Test with different hand sizes / thumb reach
- [ ] Verify animations are smooth (no frame drops)
- [ ] Test with TalkBack enabled
- [ ] Test rapid tap sequences
- [ ] Test rotation during animation
- [ ] Verify keyboard accessory bar still works

---

## Limitations

| Limitation | Rationale |
|------------|-----------|
| Android only | Web interface keeps current navigation |
| Right-hand only | Initial version; left-hand mode is future enhancement |
| Two levels max | Keeps interaction simple; deeper nesting increases complexity |
| Fixed position | Bottom-right; no repositioning option initially |

---

## Future Enhancements

1. **Left-hand mode**: Mirror arc to bottom-left corner
2. **Haptic feedback**: Vibration on item selection
3. **Gesture shortcuts**: Swipe from corner to quick-navigate
4. **Customizable items**: User-configurable menu order
5. **Badges**: Notification count on menu items
6. **Long-press actions**: Context menu on items

---

## Dependencies

- Android API 31+ (minSdk)
- Compose Animation (for arc animations)
- Compose Foundation (for gestures, layout)
- Material Icons Extended (for additional icons)
