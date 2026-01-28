# Spec 21: Android Automated Testing with Maestro

> **Status:** Planned
> **Created:** 2026-01-28

## Overview

This spec defines the automated testing infrastructure for the Android app using Maestro, enabling visual UI testing with screenshot feedback. The setup is designed to be agent-accessible, allowing Claude Code agents to run tests and receive visual feedback on UI changes.

## Goals

1. **Visual feedback** - Screenshots capture actual rendered UI including layout, spacing, colors
2. **Agent-accessible** - Agents can run tests and read results without manual intervention
3. **Easy setup** - Single Makefile command to install all dependencies on new machines
4. **Reproducible** - Headless emulator with consistent configuration

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Agent runs: make android-test                          │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│  1. Start headless emulator (if not running)            │
│  2. Build and install debug APK                         │
│  3. Run Maestro flows                                   │
│  4. Collect screenshots to maestro/screenshots/         │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│  Agent reads screenshots for visual feedback            │
│  - Can detect layout issues, color problems, etc.       │
└─────────────────────────────────────────────────────────┘
```

## Dependencies

### Required Software

| Component | Version | Purpose |
|-----------|---------|---------|
| JDK | 17+ | Android build toolchain |
| Android SDK | Latest | Build tools, platform tools |
| Android Emulator | Latest | Headless device simulation |
| System Image | android-33;google_apis;x86_64 | Emulator OS |
| Maestro | Latest | UI test runner |

### Disk Space

- Android SDK: ~2GB
- System image: ~1GB
- Emulator: ~2GB
- Total: ~5GB

## Directory Structure

```
app/android/
├── maestro/
│   ├── flows/
│   │   ├── daily-screen.yaml
│   │   ├── files-screen.yaml
│   │   ├── sleep-screen.yaml
│   │   ├── claude-screen.yaml
│   │   ├── settings-screen.yaml
│   │   └── full-navigation.yaml
│   ├── screenshots/           # Output directory (gitignored)
│   └── README.md
├── app/
└── ...
```

## Makefile Targets

### Setup (run once on new machine)

```makefile
# Install all Android testing dependencies
android-test-setup:
	@echo "Installing Android testing dependencies..."
	# Install JDK if not present
	command -v java >/dev/null || sudo apt install -y openjdk-17-jdk
	# Download and install Android command-line tools
	./scripts/install-android-sdk.sh
	# Install Maestro
	curl -Ls "https://get.maestro.mobile.dev" | bash
	# Create AVD
	$(ANDROID_HOME)/cmdline-tools/latest/bin/avdmanager create avd \
		-n notes_editor_test \
		-k "system-images;android-33;google_apis;x86_64" \
		--force
	@echo "Setup complete. Run 'make android-test' to run tests."
```

### Emulator Management

```makefile
# Start headless emulator in background
android-emulator-start:
	@if ! adb devices | grep -q emulator; then \
		echo "Starting headless emulator..."; \
		$(ANDROID_HOME)/emulator/emulator -avd notes_editor_test \
			-no-window -no-audio -gpu swiftshader_indirect & \
		adb wait-for-device; \
		sleep 30; \
	else \
		echo "Emulator already running"; \
	fi

# Stop emulator
android-emulator-stop:
	adb -s emulator-5554 emu kill 2>/dev/null || true
```

### Test Execution

```makefile
# Run all Maestro tests (starts emulator if needed)
android-test: android-emulator-start
	cd app/android && ./gradlew installDebug
	maestro test app/android/maestro/flows/ \
		--output app/android/maestro/screenshots/
	@echo "Screenshots saved to app/android/maestro/screenshots/"

# Run specific flow
android-test-daily:
	maestro test app/android/maestro/flows/daily-screen.yaml \
		--output app/android/maestro/screenshots/

# Run tests and show summary
android-test-report:
	$(MAKE) android-test
	@echo "\n=== Test Screenshots ==="
	@ls -la app/android/maestro/screenshots/*.png 2>/dev/null || echo "No screenshots"
```

## Maestro Flow Examples

### Daily Screen Flow

```yaml
# maestro/flows/daily-screen.yaml
appId: com.bartmann.noteseditor
---
- launchApp
- takeScreenshot: "01-app-launch"

# Navigate to Daily (should be default)
- assertVisible: "Daily"
- takeScreenshot: "02-daily-screen"

# Test refresh
- tapOn:
    id: "refresh"
- waitForAnimationToEnd
- takeScreenshot: "03-after-refresh"

# Test add work task
- tapOn: "Work task"
- assertVisible: "Task description"
- takeScreenshot: "04-task-input-visible"
- inputText: "Test task from Maestro"
- takeScreenshot: "05-task-text-entered"
- tapOn:
    id: "save"
- waitForAnimationToEnd
- takeScreenshot: "06-task-saved"

# Test edit mode
- tapOn: "Edit"
- assertVisible: "Save"
- assertVisible: "Cancel"
- takeScreenshot: "07-edit-mode"
- tapOn: "Cancel"
- takeScreenshot: "08-edit-cancelled"
```

### Full Navigation Flow

```yaml
# maestro/flows/full-navigation.yaml
appId: com.bartmann.noteseditor
---
- launchApp
- takeScreenshot: "nav-01-launch"

# Verify bottom nav exists
- assertVisible: "Daily"
- assertVisible: "Files"
- assertVisible: "Sleep"
- assertVisible: "Tools"

# Navigate to Files
- tapOn: "Files"
- waitForAnimationToEnd
- takeScreenshot: "nav-02-files"

# Navigate to Sleep
- tapOn: "Sleep"
- waitForAnimationToEnd
- takeScreenshot: "nav-03-sleep"

# Navigate to Tools
- tapOn: "Tools"
- waitForAnimationToEnd
- takeScreenshot: "nav-04-tools"

# Navigate to Claude via Tools
- tapOn: "Claude"
- waitForAnimationToEnd
- takeScreenshot: "nav-05-claude"

# Back to Tools
- back
- takeScreenshot: "nav-06-back-to-tools"

# Navigate to Settings
- tapOn: "Settings"
- waitForAnimationToEnd
- takeScreenshot: "nav-07-settings"

# Back to Daily
- tapOn: "Daily"
- waitForAnimationToEnd
- takeScreenshot: "nav-08-back-to-daily"
```

### Sleep Screen Flow

```yaml
# maestro/flows/sleep-screen.yaml
appId: com.bartmann.noteseditor
---
- launchApp
- tapOn: "Sleep"
- waitForAnimationToEnd
- takeScreenshot: "sleep-01-initial"

# Select child
- tapOn: "Thomas"
- takeScreenshot: "sleep-02-thomas-selected"

# Select status
- tapOn: "Eingeschlafen"
- takeScreenshot: "sleep-03-status-selected"

# Enter time
- tapOn:
    hint: "Entry"
- inputText: "20:30"
- takeScreenshot: "sleep-04-time-entered"

# Submit (don't actually submit in test to avoid side effects)
# Just verify the form state
- assertVisible: "Append"
- takeScreenshot: "sleep-05-ready-to-submit"
```

## Setup Script

Create `scripts/install-android-sdk.sh`:

```bash
#!/bin/bash
set -e

ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$HOME/android-sdk}"
CMDLINE_TOOLS_URL="https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip"

echo "Installing Android SDK to $ANDROID_SDK_ROOT..."

# Create directory
mkdir -p "$ANDROID_SDK_ROOT/cmdline-tools"

# Download command-line tools
if [ ! -d "$ANDROID_SDK_ROOT/cmdline-tools/latest" ]; then
    echo "Downloading command-line tools..."
    wget -q "$CMDLINE_TOOLS_URL" -O /tmp/cmdline-tools.zip
    unzip -q /tmp/cmdline-tools.zip -d /tmp/
    mv /tmp/cmdline-tools "$ANDROID_SDK_ROOT/cmdline-tools/latest"
    rm /tmp/cmdline-tools.zip
fi

# Set up environment
export ANDROID_HOME="$ANDROID_SDK_ROOT"
export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$PATH"

# Accept licenses
yes | sdkmanager --licenses >/dev/null 2>&1 || true

# Install required packages
echo "Installing SDK packages..."
sdkmanager --install \
    "platform-tools" \
    "emulator" \
    "platforms;android-33" \
    "build-tools;33.0.2" \
    "system-images;android-33;google_apis;x86_64"

echo ""
echo "Android SDK installed successfully!"
echo ""
echo "Add to your shell profile:"
echo "  export ANDROID_HOME=$ANDROID_SDK_ROOT"
echo "  export PATH=\$ANDROID_HOME/cmdline-tools/latest/bin:\$ANDROID_HOME/platform-tools:\$ANDROID_HOME/emulator:\$PATH"
```

## Agent Usage

Agents can run tests and read screenshots for visual feedback:

```bash
# Run tests
make android-test

# Read screenshot for visual inspection
# (Agent uses Read tool on PNG files)
ls app/android/maestro/screenshots/
```

The agent can then use the Read tool on screenshot files to visually inspect the UI and identify issues with:
- Layout and spacing
- Color/theme application
- Text rendering and truncation
- Component visibility
- Navigation state

## Environment Variables

```bash
# Required in shell profile or .env
ANDROID_HOME=$HOME/android-sdk
ANDROID_SDK_ROOT=$HOME/android-sdk
PATH=$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$PATH

# Optional: For KVM acceleration (Linux)
# User must be in 'kvm' group: sudo usermod -aG kvm $USER
```

## CI Integration

For GitHub Actions or similar CI:

```yaml
# .github/workflows/android-test.yml
name: Android UI Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up JDK 17
        uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'

      - name: Setup Android SDK
        uses: android-actions/setup-android@v3

      - name: Install Maestro
        run: curl -Ls "https://get.maestro.mobile.dev" | bash

      - name: Start emulator
        uses: reactivecircus/android-emulator-runner@v2
        with:
          api-level: 33
          target: google_apis
          arch: x86_64
          script: |
            ./gradlew installDebug
            maestro test app/android/maestro/flows/

      - name: Upload screenshots
        uses: actions/upload-artifact@v4
        with:
          name: maestro-screenshots
          path: app/android/maestro/screenshots/
```

## Limitations

1. **x86_64 only** - ARM emulation is too slow; requires x86 host with KVM
2. **Linux/macOS** - Windows requires WSL2 for acceptable performance
3. **No GPU rendering** - Uses swiftshader (software rendering), slower but consistent
4. **Startup time** - Emulator cold boot takes 30-60 seconds

## Future Enhancements

- Screenshot comparison for regression detection
- Video recording of test flows
- Performance metrics collection
- Parallel test execution on multiple emulators
