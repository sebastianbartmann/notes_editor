# Implementation Plan

> Last updated: 2026-01-19
> Status: Planning

## Instructions
- Tasks marked `- [ ]` are incomplete
- Tasks marked `- [x]` are complete  
- Work from top to bottom (highest priority first)
- Add new tasks as you discover them

## Current Sprint

_No current sprint items_

## Backlog

_Future work goes here_

## Completed

_Completed tasks stay here for reference, will be cleaned up periodically_

### Android Keyboard Accessory Bar (spec: 14-android-keyboard-accessory-bar.md)

**Phase 1: Fix Keyboard Overlap**
- [x] Add `android:windowSoftInputMode="adjustResize"` to AndroidManifest.xml
- [x] Add `WindowCompat.setDecorFitsSystemWindows(window, false)` in MainActivity.onCreate()
- [x] Add `imePadding()` modifier to main content in AppNavigation.kt
- [x] Test content scrolls/resizes correctly when keyboard appears

**Phase 2: Implement Keyboard Accessory Bar**
- [x] Create `KeyboardAccessoryBar.kt` with basic UI (7 buttons: ↑↓←→/[])
- [x] Implement keyboard visibility detection using `WindowInsets.isImeVisible`
- [x] Add `AnimatedVisibility` for smooth show/hide transitions
- [x] Integrate accessory bar into AppNavigation.kt layout

**Phase 3: Input Injection**
- [x] Implement text character insertion for `/`, `[`, `]`
- [x] Implement arrow key simulation for cursor movement
- [x] Test input injection works with CompactTextField

**Phase 4: Polish**
- [x] Style the accessory bar to match app theme (AppTheme colors)
- [x] Add touch feedback/ripple effects to buttons
- [x] Test across all 6 screens with text inputs
- [x] Verify keyboard + bar don't overlap app content

6. specs/.gitkeep
# This folder contains specification documents
# Create one .md file per major feature or concern
