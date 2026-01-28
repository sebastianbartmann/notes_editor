.PHONY: help server client test test-server test-client install-client install-systemd status-systemd build-android install-android deploy-android debug-android build build-server build-web

.DEFAULT_GOAL := help

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Build:"
	@echo "    build         Build everything (server + web UI)"
	@echo "    build-server  Build Go server binary"
	@echo "    build-web     Build React web UI"
	@echo ""
	@echo "  Development:"
	@echo "    server        Run the Go server (port 8080)"
	@echo "    client        Run the React dev server (port 5173)"
	@echo ""
	@echo "  Testing:"
	@echo "    test          Run all tests"
	@echo "    test-server   Run Go server tests"
	@echo "    test-client   Run React client tests"
	@echo ""
	@echo "  Setup:"
	@echo "    install-client  Install React client dependencies"
	@echo ""
	@echo "  Deployment:"
	@echo "    install-systemd Install/update systemd service"
	@echo "    status-systemd  Show systemd service status"
	@echo ""
	@echo "  Android:"
	@echo "    build-android   Build the Android debug APK"
	@echo "    install-android Install the debug APK via adb (USB)"
	@echo "    deploy-android  Build and install the debug APK"
	@echo "    debug-android   Print adb error log output"

# Build
build: build-web build-server

build-server:
	cd server && go build -o bin/server ./cmd/server

build-web:
	cd clients/web && npm install && npm run build
	rm -rf server/static
	cp -r clients/web/dist server/static

# Development
server:
	./server/bin/server

client:
	cd clients/web && npm run dev

# Testing
test: test-server test-client

test-server:
	cd server && go test -v ./...

test-client:
	cd clients/web && npm test

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
