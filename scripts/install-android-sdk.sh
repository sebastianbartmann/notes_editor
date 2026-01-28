#!/bin/bash
# Install Android SDK for automated testing with Maestro
# See spec 21: Android Automated Testing with Maestro

set -e

ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$HOME/android-sdk}"
CMDLINE_TOOLS_URL="https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip"

echo "Installing Android SDK to $ANDROID_SDK_ROOT..."

# Create directory
mkdir -p "$ANDROID_SDK_ROOT/cmdline-tools"

# Download command-line tools if not present
if [ ! -d "$ANDROID_SDK_ROOT/cmdline-tools/latest" ]; then
    echo "Downloading command-line tools..."
    TMP_ZIP=$(mktemp)
    wget -q "$CMDLINE_TOOLS_URL" -O "$TMP_ZIP"
    unzip -q "$TMP_ZIP" -d /tmp/
    mv /tmp/cmdline-tools "$ANDROID_SDK_ROOT/cmdline-tools/latest"
    rm "$TMP_ZIP"
    echo "Command-line tools installed."
else
    echo "Command-line tools already installed."
fi

# Set up environment for this script
export ANDROID_HOME="$ANDROID_SDK_ROOT"
export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$PATH"

# Accept licenses
echo "Accepting SDK licenses..."
yes | sdkmanager --licenses >/dev/null 2>&1 || true

# Install required packages
echo "Installing SDK packages (this may take a few minutes)..."
sdkmanager --install \
    "platform-tools" \
    "emulator" \
    "platforms;android-33" \
    "build-tools;33.0.2" \
    "system-images;android-33;google_apis;x86_64"

echo ""
echo "Android SDK installed successfully!"
echo ""
echo "Add these lines to your shell profile (~/.bashrc or ~/.zshrc):"
echo ""
echo "  export ANDROID_HOME=$ANDROID_SDK_ROOT"
echo "  export ANDROID_SDK_ROOT=$ANDROID_SDK_ROOT"
echo "  export PATH=\$ANDROID_HOME/cmdline-tools/latest/bin:\$ANDROID_HOME/platform-tools:\$ANDROID_HOME/emulator:\$PATH"
echo ""
echo "Then run: source ~/.bashrc"
