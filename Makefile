.PHONY: help run install status android-build android-install deploy-android debug-android

.DEFAULT_GOAL := help

help:
	@echo "Available targets:"
	@echo "  run      Run the dev server"
	@echo "  install  Install/update systemd service"
	@echo "  status   Show systemd service status"
	@echo "  android-build  Build the Android debug APK"
	@echo "  android-install  Install the debug APK via adb (USB)"
	@echo "  deploy-android  Build and install the debug APK"
	@echo "  debug-android  Print adb error log output"

run:
	NOTES_TOKEN="VJY9EoAf1xx1bO-LaduCmItwRitCFm9BPuQZ8jd0tcg" uv run uvicorn server.web_app.main:app --reload --host 0.0.0.0 --port 8000

install:
	sudo cp notes-editor.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable notes-editor
	sudo systemctl restart notes-editor

status:
	sudo systemctl status notes-editor

android-build:
	GRADLE_USER_HOME="$(PWD)/.gradle" \
	$(PWD)/app/gradle-8.7/bin/gradle --no-daemon -p $(PWD)/app/android :app:assembleDebug

android-install:
	ADB_SERVER_SOCKET=tcp:localhost:5038 \
	$(PWD)/app/android_sdk/platform-tools/adb install -r \
	$(PWD)/app/android/app/build/outputs/apk/debug/app-debug.apk

deploy-android: android-build android-install

debug-android:
	ADB_SERVER_SOCKET=tcp:localhost:5038 \
	$(PWD)/app/android_sdk/platform-tools/adb logcat -d *:E
