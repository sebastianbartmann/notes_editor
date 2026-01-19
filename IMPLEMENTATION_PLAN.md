# Implementation Plan

> Last updated: 2026-01-19
> Status: Active

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
