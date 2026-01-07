# Notes Editor Android App (CLI)

This folder contains the native Android app. It talks to the existing FastAPI server over HTTP.

## Prerequisites

- JDK 17
- Android SDK command-line tools

## SDK setup (recommended layout)

```bash
export ANDROID_SDK_ROOT="/home/t14s/dev/notes_editor/app/android_sdk"
mkdir -p "$ANDROID_SDK_ROOT"

# Download the command line tools from Google and unzip into:
# $ANDROID_SDK_ROOT/cmdline-tools/latest
# Example path: $ANDROID_SDK_ROOT/cmdline-tools/latest/bin/sdkmanager

export PATH="$ANDROID_SDK_ROOT/cmdline-tools/latest/bin:$ANDROID_SDK_ROOT/platform-tools:$PATH"

sdkmanager "platform-tools" "platforms;android-35" "build-tools;35.0.0"
yes | sdkmanager --licenses
```

## Build

```bash
cd app/android

# If you already have a Gradle wrapper, use it:
# ./gradlew assembleDebug

# Otherwise install Gradle 8+ and run:
gradle assembleDebug
```

The debug APK will be at:
`app/android/app/build/outputs/apk/debug/app-debug.apk`

## App configuration

- Bearer token lives in `app/android/app/src/main/java/com/bartmann/noteseditor/AppConfig.kt`.
- Base URLs are defined in `AppConfig.kt` (LAN + Tailscale).
- Replace the placeholder noise file at `app/android/app/src/main/res/raw/noise.mp3`.
