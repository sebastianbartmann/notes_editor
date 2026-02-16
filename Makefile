.PHONY: help server client test test-server test-client test-coverage install-client install-systemd status-systemd install-pi-gateway-systemd status-pi-gateway-systemd install-qmd-systemd status-qmd-systemd restart-services setup-android-build-toolchain build-android install-android deploy-android debug-android build build-server build-web lint clean build-pi-gateway run-pi-gateway android-test-setup android-emulator-start android-emulator-stop android-test android-test-report android-test-daily android-test-daily-scroll-focus android-test-files android-test-sleep android-test-claude android-test-settings android-test-nav

ANDROID_GRADLE_VERSION := 8.7
ANDROID_GRADLE_DIR := $(PWD)/app/gradle-$(ANDROID_GRADLE_VERSION)
ANDROID_GRADLE_BIN := $(ANDROID_GRADLE_DIR)/bin/gradle
ANDROID_GRADLE_ZIP := gradle-$(ANDROID_GRADLE_VERSION)-bin.zip
ANDROID_GRADLE_URL := https://services.gradle.org/distributions/$(ANDROID_GRADLE_ZIP)

ANDROID_SDK_ROOT := $(PWD)/app/android_sdk
ANDROID_CMDLINE_TOOLS_BIN := $(ANDROID_SDK_ROOT)/cmdline-tools/latest/bin
ANDROID_SDKMANAGER := $(ANDROID_CMDLINE_TOOLS_BIN)/sdkmanager
ANDROID_LOCAL_PROPERTIES := $(PWD)/app/android/local.properties

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
ANDROID_CMDLINE_TOOLS_URL := https://dl.google.com/android/repository/commandlinetools-mac-14742923_latest.zip
else ifeq ($(UNAME_S),Linux)
ANDROID_CMDLINE_TOOLS_URL := https://dl.google.com/android/repository/commandlinetools-linux-14742923_latest.zip
else
ANDROID_CMDLINE_TOOLS_URL :=
endif

.DEFAULT_GOAL := help

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Build:"
	@echo "    build         Build everything (server + web UI + pi gateway)"
	@echo "    build-server  Build Go server binary"
	@echo "    build-web     Build React web UI"
	@echo ""
	@echo "  Development:"
	@echo "    server        Run the Go server (dev: port 8080)"
	@echo "    client        Run the React dev server (port 5173)"
	@echo "    run-pi-gateway Build and run Pi gateway sidecar"
	@echo ""
	@echo "  Testing:"
	@echo "    test          Run all tests"
	@echo "    test-server   Run Go server tests"
	@echo "    test-client   Run React client tests"
	@echo "    test-coverage Run Go tests with coverage report"
	@echo "    lint          Run Go linter"
	@echo "    clean         Remove build artifacts"
	@echo ""
	@echo "  Setup:"
	@echo "    install-client  Install React client dependencies"
	@echo "    build-pi-gateway Install deps and build Pi gateway sidecar"
	@echo ""
	@echo "  Deployment:"
	@echo "    install-systemd Install/update systemd service"
	@echo "    status-systemd  Show systemd service status"
	@echo "    install-pi-gateway-systemd Install/update gateway sidecar service"
	@echo "    status-pi-gateway-systemd  Show gateway sidecar service status"
	@echo "    install-qmd-systemd Install/update qmd sidecar service"
	@echo "    status-qmd-systemd  Show qmd sidecar service status"
	@echo "    restart-services Restart qmd + gateway + server services"
	@echo ""
	@echo "  Android:"
	@echo "    setup-android-build-toolchain Install repo-local Android CLI build prerequisites"
	@echo "    build-android   Build the Android debug APK"
	@echo "    install-android Install the debug APK via adb (USB)"
	@echo "    deploy-android  Build and install the debug APK"
	@echo "    debug-android   Print adb error log output"
	@echo ""
	@echo "  Android Testing (Maestro):"
	@echo "    android-test-setup     One-time setup for Android testing"
	@echo "    android-emulator-start Start headless emulator"
	@echo "    android-emulator-stop  Stop emulator"
	@echo "    android-test           Run all Maestro UI tests"
	@echo "    android-test-report    Run tests and show summary"
	@echo "    android-test-daily     Run daily screen tests only"
	@echo "    android-test-daily-scroll-focus Run daily editor scroll/focus regression test"
	@echo "    android-test-files     Run files screen tests only"
	@echo "    android-test-sleep     Run sleep screen tests only"
	@echo "    android-test-claude    Run claude screen tests only"
	@echo "    android-test-settings  Run settings screen tests only"
	@echo "    android-test-nav       Run navigation tests only"

# Build
build: build-pi-gateway build-web build-server

build-server:
	cd server && go build -o bin/server ./cmd/server

build-web:
	cd clients/web && npm install && npm run build
	rm -rf server/static
	cp -r clients/web/dist server/static

# Development
server: build-server
	SERVER_ADDR=:8080 ./server/bin/server

client:
	cd clients/web && npm run dev

# Testing
test: test-server test-client

test-server:
	cd server && go test -v ./...

test-client:
	cd clients/web && npx tsc --noEmit && npm test

test-coverage:
	cd server && go test -v -coverprofile=coverage.out ./...
	cd server && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: server/coverage.html"

lint:
	cd server && golangci-lint run ./...

clean:
	rm -rf server/bin server/static server/coverage.out server/coverage.html

build-pi-gateway:
	cd pi-gateway && npm install && npm run build

run-pi-gateway: build-pi-gateway
	cd pi-gateway && npm start

# Setup
install-client:
	cd clients/web && npm install

install-systemd:
	sudo cp notes-editor.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable notes-editor
	sudo systemctl restart notes-editor

status-systemd:
	sudo systemctl status notes-editor

install-pi-gateway-systemd:
	sudo cp notes-pi-gateway.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable notes-pi-gateway
	sudo systemctl restart notes-pi-gateway

status-pi-gateway-systemd:
	sudo systemctl status notes-pi-gateway

install-qmd-systemd:
	sudo cp notes-qmd.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable notes-qmd
	sudo systemctl restart notes-qmd

status-qmd-systemd:
	sudo systemctl status notes-qmd

restart-services:
	sudo systemctl daemon-reload
	sudo systemctl restart notes-qmd
	sudo systemctl restart notes-pi-gateway
	sudo systemctl restart notes-editor

setup-android-build-toolchain:
	@[ -n "$(ANDROID_CMDLINE_TOOLS_URL)" ] || (echo "Unsupported OS for automated Android cmdline-tools setup."; exit 1)
	@java -version >/dev/null 2>&1 || (echo "Java runtime not found. Install JDK 17 first (example on macOS: brew install --cask temurin@17)."; exit 1)
	@java -version 2>&1 | head -n 1 | grep '"17\.' >/dev/null || (echo "JDK 17 is required. Current Java:"; java -version; exit 1)
	@mkdir -p "$(ANDROID_SDK_ROOT)/cmdline-tools" "$(PWD)/.tmp/android-toolchain"
	@if [ ! -x "$(ANDROID_GRADLE_BIN)" ]; then \
		echo "Installing Gradle $(ANDROID_GRADLE_VERSION) into app/"; \
		curl -fsSL "$(ANDROID_GRADLE_URL)" -o "$(PWD)/.tmp/android-toolchain/$(ANDROID_GRADLE_ZIP)"; \
		unzip -q "$(PWD)/.tmp/android-toolchain/$(ANDROID_GRADLE_ZIP)" -d "$(PWD)/app"; \
	fi
	@if [ ! -x "$(ANDROID_SDKMANAGER)" ]; then \
		echo "Installing Android SDK command-line tools into app/android_sdk/"; \
		curl -fsSL "$(ANDROID_CMDLINE_TOOLS_URL)" -o "$(PWD)/.tmp/android-toolchain/cmdline-tools.zip"; \
		unzip -q "$(PWD)/.tmp/android-toolchain/cmdline-tools.zip" -d "$(ANDROID_SDK_ROOT)/cmdline-tools"; \
		rm -rf "$(ANDROID_SDK_ROOT)/cmdline-tools/latest"; \
		mv "$(ANDROID_SDK_ROOT)/cmdline-tools/cmdline-tools" "$(ANDROID_SDK_ROOT)/cmdline-tools/latest"; \
	fi
	@"$(ANDROID_SDKMANAGER)" --sdk_root="$(ANDROID_SDK_ROOT)" "platform-tools" "platforms;android-35" "build-tools;35.0.0"
	@yes | "$(ANDROID_SDKMANAGER)" --sdk_root="$(ANDROID_SDK_ROOT)" --licenses >/dev/null
	@printf "sdk.dir=%s\n" "$(ANDROID_SDK_ROOT)" > "$(ANDROID_LOCAL_PROPERTIES)"
	@echo "Android build toolchain is ready."

build-android:
	GRADLE_USER_HOME="$(PWD)/.gradle" \
	$(PWD)/app/gradle-8.7/bin/gradle --no-daemon -Dorg.gradle.daemon=false -Dorg.gradle.jvmargs= -p $(PWD)/app/android :app:assembleDebug

install-android:
	ADB_SERVER_SOCKET=tcp:localhost:5038 \
	$(PWD)/app/android_sdk/platform-tools/adb install -r \
	$(PWD)/app/android/app/build/outputs/apk/debug/app-debug.apk

deploy-android: build-android install-android

debug-android:
	ADB_SERVER_SOCKET=tcp:localhost:5038 \
	$(PWD)/app/android_sdk/platform-tools/adb logcat -d *:E

# Android Testing with Maestro
# Uses ANDROID_HOME if set, otherwise defaults to $HOME/android-sdk
ANDROID_HOME ?= $(HOME)/android-sdk
MAESTRO_FLOWS := $(PWD)/app/android/maestro/flows
MAESTRO_SCREENSHOTS := $(PWD)/app/android/maestro/screenshots
MAESTRO_BIN := $(shell command -v maestro 2>/dev/null)
ifeq ($(MAESTRO_BIN),)
MAESTRO_BIN := $(HOME)/.maestro/bin/maestro
endif

android-test-setup:
	@echo "Installing Android testing dependencies..."
	@command -v java >/dev/null || (echo "Installing OpenJDK 17..." && sudo apt install -y openjdk-17-jdk)
	@./scripts/install-android-sdk.sh
	@echo "Installing Maestro..."
	@curl -Ls "https://get.maestro.mobile.dev" | bash
	@if command -v maestro >/dev/null 2>&1; then \
		echo "Maestro installed at: $$(command -v maestro)"; \
	elif [ -x "$(HOME)/.maestro/bin/maestro" ]; then \
		echo "Maestro installed at: $(HOME)/.maestro/bin/maestro"; \
	else \
		echo "Maestro install did not produce an executable."; \
		echo "Ensure maestro is on PATH or available at ~/.maestro/bin/maestro."; \
		exit 1; \
	fi
	@echo "Creating AVD..."
	@$(ANDROID_HOME)/cmdline-tools/latest/bin/avdmanager create avd \
		-n notes_editor_test \
		-k "system-images;android-33;google_apis;x86_64" \
		--force
	@echo ""
	@echo "Setup complete! Run 'make android-test' to run tests."

android-emulator-start:
	@if ! $(ANDROID_HOME)/platform-tools/adb devices | grep -q emulator; then \
		echo "Starting headless emulator..."; \
		$(ANDROID_HOME)/emulator/emulator -avd notes_editor_test \
			-no-window -no-audio -gpu swiftshader_indirect & \
		$(ANDROID_HOME)/platform-tools/adb wait-for-device; \
		echo "Waiting for emulator to boot..."; \
		sleep 30; \
		echo "Emulator ready."; \
	else \
		echo "Emulator already running."; \
	fi

android-emulator-stop:
	@$(ANDROID_HOME)/platform-tools/adb -s emulator-5554 emu kill 2>/dev/null || echo "No emulator running."

android-check-maestro:
	@if [ ! -x "$(MAESTRO_BIN)" ]; then \
		echo "Maestro executable not found."; \
		echo "Expected either 'maestro' on PATH or $(HOME)/.maestro/bin/maestro."; \
		echo "Run 'make android-test-setup' to install prerequisites."; \
		exit 1; \
	fi

android-test: android-emulator-start android-check-maestro
	@echo "Building and installing debug APK..."
	@GRADLE_USER_HOME="$(PWD)/.gradle" \
	$(PWD)/app/gradle-8.7/bin/gradle --no-daemon -Dorg.gradle.daemon=false -Dorg.gradle.jvmargs= -p $(PWD)/app/android :app:assembleDebug
	@$(ANDROID_HOME)/platform-tools/adb install -r $(PWD)/app/android/app/build/outputs/apk/debug/app-debug.apk
	@echo "Running Maestro tests..."
	@mkdir -p $(MAESTRO_SCREENSHOTS)
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS) --output $(MAESTRO_SCREENSHOTS)
	@echo ""
	@echo "Screenshots saved to: app/android/maestro/screenshots/"

android-test-report: android-test
	@echo ""
	@echo "=== Test Screenshots ==="
	@ls -la $(MAESTRO_SCREENSHOTS)/*.png 2>/dev/null || echo "No screenshots generated."

android-test-daily: android-emulator-start android-check-maestro
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS)/daily-screen.yaml --output $(MAESTRO_SCREENSHOTS)
	@echo "Screenshots saved to: app/android/maestro/screenshots/"

android-test-daily-scroll-focus: android-emulator-start android-check-maestro
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS)/daily-editor-scroll-focus.yaml --output $(MAESTRO_SCREENSHOTS)
	@echo "Screenshots saved to: app/android/maestro/screenshots/"

android-test-files: android-emulator-start android-check-maestro
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS)/files-screen.yaml --output $(MAESTRO_SCREENSHOTS)
	@echo "Screenshots saved to: app/android/maestro/screenshots/"

android-test-sleep: android-emulator-start android-check-maestro
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS)/sleep-screen.yaml --output $(MAESTRO_SCREENSHOTS)
	@echo "Screenshots saved to: app/android/maestro/screenshots/"

android-test-claude: android-emulator-start android-check-maestro
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS)/claude-screen.yaml --output $(MAESTRO_SCREENSHOTS)
	@echo "Screenshots saved to: app/android/maestro/screenshots/"

android-test-settings: android-emulator-start android-check-maestro
	@echo "Building and installing debug APK..."
	@GRADLE_USER_HOME="$(PWD)/.gradle" \
	$(PWD)/app/gradle-8.7/bin/gradle --no-daemon -Dorg.gradle.daemon=false -Dorg.gradle.jvmargs= -p $(PWD)/app/android :app:assembleDebug
	@$(ANDROID_HOME)/platform-tools/adb install -r $(PWD)/app/android/app/build/outputs/apk/debug/app-debug.apk
	@echo "Running Maestro settings flow..."
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS)/settings-screen.yaml --output $(MAESTRO_SCREENSHOTS)
	@echo "Screenshots saved to: app/android/maestro/screenshots/"

android-test-nav: android-emulator-start android-check-maestro
	@$(MAESTRO_BIN) test $(MAESTRO_FLOWS)/full-navigation.yaml --output $(MAESTRO_SCREENSHOTS)
	@echo "Screenshots saved to: app/android/maestro/screenshots/"
