# Maestro UI Tests

Automated UI tests for the Notes Editor Android app using [Maestro](https://maestro.mobile.dev/).

## Setup

Run the one-time setup to install Android SDK and Maestro:

```bash
make android-test-setup
```

This installs:
- Android SDK with emulator and system image
- Maestro CLI for UI testing

## Running Tests

### Run All Tests
```bash
make android-test
```

This will:
1. Start the headless emulator (if not running)
2. Build and install the debug APK
3. Run all test flows
4. Save screenshots to `screenshots/`

### Run Individual Tests
```bash
make android-test-daily      # Daily screen tests
make android-test-files      # Files screen tests
make android-test-sleep      # Sleep screen tests
make android-test-claude     # Claude screen tests
make android-test-settings   # Settings screen tests
make android-test-nav        # Full navigation tests
```

### Emulator Management
```bash
make android-emulator-start  # Start headless emulator
make android-emulator-stop   # Stop emulator
```

## Test Flows

| Flow | Description | Screenshots |
|------|-------------|-------------|
| `daily-screen.yaml` | Tests daily note display, refresh, task add, edit mode | 8 |
| `full-navigation.yaml` | Tests bottom nav and screen transitions | 8 |
| `sleep-screen.yaml` | Tests sleep tracking form | 5 |
| `files-screen.yaml` | Tests file browser | 4 |
| `claude-screen.yaml` | Tests Claude chat interface | 4 |
| `settings-screen.yaml` | Tests settings, theme, person selection | 5 |

## Screenshots

Screenshots are saved to `screenshots/` directory after each test run. This directory is gitignored.

Screenshot naming:
- `01-app-launch.png` - Initial app state
- `nav-02-files.png` - After navigating to files
- `sleep-03-status-selected.png` - After selecting sleep status

## Writing New Tests

Maestro uses YAML-based flow definitions. Basic syntax:

```yaml
appId: com.bartmann.noteseditor
---
- launchApp
- takeScreenshot: "01-initial"
- tapOn: "Button Text"
- assertVisible: "Expected Text"
- inputText: "Some input"
- waitForAnimationToEnd
```

See [Maestro documentation](https://maestro.mobile.dev/getting-started/writing-your-first-flow) for complete syntax.

## Agent Usage

Agents can run tests and read screenshots for visual feedback:

```bash
# Run tests
make android-test

# List screenshots
ls app/android/maestro/screenshots/

# Agent uses Read tool on PNG files to visually inspect UI
```

## Troubleshooting

### Emulator won't start
- Ensure KVM is enabled: `sudo usermod -aG kvm $USER`
- Check virtualization in BIOS
- Try: `make android-emulator-stop && make android-emulator-start`

### Tests fail to find elements
- Check element text matches exactly (case-sensitive)
- Add `waitForAnimationToEnd` after navigation
- Verify APK is up to date: `make build-android`

### No screenshots generated
- Check emulator is running: `adb devices`
- Ensure app is installed: `adb shell pm list packages | grep noteseditor`
