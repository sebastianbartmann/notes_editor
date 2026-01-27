.PHONY: help run build test install-systemd status-systemd build-android install-android deploy-android debug-android

.DEFAULT_GOAL := help

help:
	@echo "Available targets:"
	@echo "  run             Run the Go dev server (port 8080)"
	@echo "  build           Build the Go server binary"
	@echo "  test            Run Go server tests"
	@echo "  install-systemd Install/update systemd service"
	@echo "  status-systemd  Show systemd service status"
	@echo "  build-android   Build the Android debug APK"
	@echo "  install-android Install the debug APK via adb (USB)"
	@echo "  deploy-android  Build and install the debug APK"
	@echo "  debug-android   Print adb error log output"

run:
	cd server && go run ./cmd/server

build:
	cd server && go build -o bin/server ./cmd/server

test:
	cd server && go test -v ./...

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
