# Implementation Plan

> Last updated: 2026-01-26
> Status: Active

## Instructions
- Tasks marked `- [ ]` are incomplete
- Tasks marked `- [x]` are complete
- Work from top to bottom (highest priority first)
- Add new tasks as you discover them

## Current Sprint

### Replace Arc Menu with Bottom Navigation Bar - 2026-01-26

**Goal:** Replace the expandable arc menu with a static 4-button bottom navigation bar, restore per-screen top headers, and keep the keyboard accessory bar.

#### Phase 1: Remove Arc Menu Components
- [x] Delete `ArcMenu.kt` (arc menu composable)
- [x] Delete `ArcMenuConfig.kt` (menu configuration)
- [x] Remove `BottomInfoBar` from `AppNavigation.kt`
- [x] Remove arc menu state management from `AppNavigation.kt`

#### Phase 2: Create Bottom Navigation Bar
- [x] Create `BottomNavBar` composable in `AppNavigation.kt` with 4 items: Daily, Files, Sleep, Tools
- [x] Style bottom nav bar: 56dp height, equal-width items, accent color for active
- [x] Wire navigation callbacks to NavController
- [x] Hide bottom nav bar when keyboard is visible (reuse existing IME detection)
- [x] Hide bottom nav bar when person is null

#### Phase 3: Restore Tools Screen
- [x] Restore/create `ToolsScreen.kt` as navigation hub
- [x] Add items: Claude, Noise, Notifications, Settings, Kiosk (external link)
- [x] Add Tools route to navigation graph

#### Phase 4: Restore Per-Screen Top Headers
- [x] Restore `ScreenHeader` composable in `UiComponents.kt` (title + optional action button)
- [x] Add `ScreenHeader` to `DailyScreen.kt` with reload action
- [x] Add `ScreenHeader` to `FilesScreen.kt` with reload action
- [x] Add `ScreenHeader` to `SleepTimesScreen.kt` with reload action
- [x] Add `ScreenHeader` to `ClaudeScreen.kt` with clear action
- [x] Add `ScreenHeader` to `NoiseScreen.kt`
- [x] Add `ScreenHeader` to `NotificationsScreen.kt`
- [x] Add `ScreenHeader` to `SettingsScreen.kt`
- [x] Add `ScreenHeader` to `ToolsScreen.kt`

#### Phase 5: Cleanup & Testing
- [x] Verify keyboard accessory bar still works (spec 14)
- [x] Remove any unused imports/code
- [ ] Test navigation on all screens (manual testing required)
- [ ] Test keyboard visibility behavior (bottom nav hides, accessory shows) (manual testing required)
- [ ] Test with person=null (only settings accessible) (manual testing required)

**Implementation Notes:**
- Removed `padding: PaddingValues` parameter from screen functions since bottom bar no longer overlays content
- `ScreenLayout` now has simpler signature without padding parameter
- Bottom nav bar renders in Column layout after content, not as overlay
- Tools screen uses list layout (rows) rather than grid for navigation items

## Backlog

_Future work goes here_

## Completed

### Fix: Android Dark Theme Cursor Visibility - 2026-01-19
- [x] Add `cursorBrush` parameter to `CompactTextField` in `UiComponents.kt` using accent color (`AppTheme.colors.accent`)
