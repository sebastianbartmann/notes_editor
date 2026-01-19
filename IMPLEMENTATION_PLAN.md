# Implementation Plan

> Last updated: 2026-01-19
> Status: Active

## Instructions
- Tasks marked `- [ ]` are incomplete
- Tasks marked `- [x]` are complete
- Work from top to bottom (highest priority first)
- Add new tasks as you discover them

## Current Sprint

_No active sprint_

## Backlog

_Future work goes here_

## Completed

### Add Task Inline Input (spec: 17-add-task-inline-input.md) - 2026-01-19

- [x] Server: Add optional `text` param to `POST /api/todos/add` endpoint in main.py
- [x] Server: Update `add_task_to_todos()` to insert `- [ ] {text}` line instead of blank
- [x] Android: Add `taskInputMode` and `taskInputText` state to DailyScreen.kt
- [x] Android: Create `TaskInputRow` composable (TextField + Save/Cancel buttons)
- [x] Android: Replace button handlers with conditional input mode UI
- [x] Android: Update `ApiClient.addTodo()` to accept optional `text` param
- [x] Web: Replace HTMX forms with JS-driven button elements in editor.html
- [x] Web: Add `showTaskInput()`, `cancelTaskInput()`, `saveTask()` JavaScript functions
- [x] Web: Add Enter/Escape key handlers for task input
- [x] Web: Add `.task-input-row` CSS styles

### Fix: Hide Append Section in Daily Notes Edit Mode - 2026-01-19
- [x] Android: Wrap append section in `if (!isEditing)` conditional in `DailyScreen.kt`
- [x] Web: Hide append form via JavaScript when `toggleEditMode()` is called, show on `cancelEdit()` in `editor.html`

### Fix: Android Noise Screen State Sync - 2026-01-19
- [x] Create `NoisePlaybackState` singleton object with shared `isPlaying` state
- [x] Update `NoiseService` to write `NoisePlaybackState.isPlaying` on play/pause/stop actions
- [x] Update `NoiseScreen` to observe `NoisePlaybackState.isPlaying` instead of local `remember` state

### Android Keyboard Accessory Bar Buttons Update - 2026-01-19
- [x] Replace `/` button with `(` and `)` buttons in KeyboardAccessoryBar.kt

### Android Bottom Info Bar (spec: 16-android-bottom-info-bar.md) - 2026-01-19

- [x] Create `BottomInfoBar` composable in AppNavigation.kt with route-to-title mapping
- [x] Integrate `BottomInfoBar` with existing `ArcMenu` as unified bottom section
- [x] Add keyboard visibility hiding (reuse existing `isKeyboardVisible` state)
- [x] Remove `ScreenTitle` from DailyScreen, add pull-to-refresh for Reload
- [x] Remove `ScreenTitle` from FilesScreen, add pull-to-refresh for Reload
- [x] Remove `ScreenTitle` from ClaudeScreen, move Clear button to input area
- [x] Remove `ScreenTitle` from SleepTimesScreen, add pull-to-refresh for Reload
- [x] Remove `ScreenTitle` from SettingsScreen
- [x] Remove `ScreenTitle` from NoiseScreen
- [x] Remove `ScreenTitle` from NotificationsScreen

### Android Arc Menu Navigation (spec: 15-android-arc-menu-navigation.md) - 2026-01-19

- [x] Create `ArcMenuItem` data class and menu configuration in `ArcMenuConfig.kt`
- [x] Implement `ArcMenuButton` composable (collapsed state FAB)
- [x] Implement `ArcMenuItem` composable (icon + label with active state)
- [x] Implement polar coordinate positioning logic (`calculateItemPosition`)
- [x] Implement `ArcMenu` composable with state management (collapsed/level1/level2)
- [x] Add expand/collapse animations (150ms fan-out along arc)
- [x] Add level transition animation (level1 ↔ level2)
- [x] Implement tap-outside-to-close scrim overlay
- [x] Integrate `ArcMenu` into `AppNavigation.kt`
- [x] Remove `BottomNavBar` from `AppNavigation.kt`
- [x] Delete `ToolsScreen.kt` (no longer needed)
- [x] Handle external URL opening (Kiosk item)
- [x] Add keyboard visibility handling (hide menu when keyboard visible)

### Android Keyboard Accessory Bar (spec: 14-android-keyboard-accessory-bar.md) - 2026-01-19

- [x] Enable edge-to-edge mode in MainActivity
- [x] Add `windowSoftInputMode="adjustResize"` to AndroidManifest
- [x] Add `imePadding()` to main Column in AppNavigation
- [x] Add `statusBars` inset padding to content area
- [x] Create KeyboardAccessoryBar composable with 7 buttons (↑↓←→/[])
- [x] Implement keyboard visibility detection via `WindowInsets.ime`
- [x] Implement text injection using KeyCharacterMap
- [x] Implement arrow key injection using KeyEvent dispatch
- [x] Hide bottom navigation when keyboard is visible
- [x] Style accessory bar to match app theme
