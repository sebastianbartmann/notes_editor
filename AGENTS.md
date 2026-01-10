# Notes Editor Overview

Notes Editor is a personal/family “second brain” app with a FastAPI + HTMX web UI and a native Android client. Data lives in a local vault with per-person subfolders (e.g., `sebastian/`, `petra/`), and each client selects its person root and theme locally while the server remains user-agnostic.

Key features: daily notes with tasks and pinned entries, a file tree editor scoped to the selected person, and shared tools such as sleep tracking, noise playback, and a Claude tool. The Android app mirrors the web layout and adds native conveniences (media-style noise controls, persistent settings).

# Repository Guidelines

## Project Structure & Module Organization

- `server/web_app/` holds the FastAPI app (`main.py`), Jinja templates in `templates/`, static assets in `static/`, and supporting code in `renderers/` and `services/`.
- `app/android/` contains the Android client; build tooling lives in `app/gradle-8.7/` and `app/android_sdk/`.
- Root files like `Makefile`, `notes-editor.service`, and `pyproject.toml` define local workflows and service configuration.

## Build, Test, and Development Commands

- `uv sync` installs Python dependencies (recommended over pip).
- `make run` starts the dev server with auto-reload at `0.0.0.0:8000`.
- `make install` installs/refreshes the systemd unit and restarts the service.
- `make status` checks the systemd service status.
- `make android-build` builds the Android debug APK.
- `make deploy-android` builds and installs the debug APK via adb.

## Coding Style & Naming Conventions

- Python: 4-space indentation, `snake_case` for functions/vars, `PascalCase` for classes.
- Kotlin/Android: follow standard Android conventions; keep resource names lowercase with underscores (e.g., `noise_player.xml`).
- Keep modules small and prefer explicit imports over wildcard imports.

## Testing Guidelines

- There is no automated test suite yet. When adding tests, use `pytest`, place them under a new `tests/` directory, and name files `test_*.py`.
- For manual verification, run `make run` and validate the web UI plus Android flows you touched.

## Commit & Pull Request Guidelines

- Commit messages in history are short, imperative, and sentence case (e.g., "Add Claude streaming updates").
- PRs should include a clear summary, steps to verify, and screenshots for UI changes (web or Android).
- Link any related issues and call out config changes (e.g., `notes-editor.service` or env vars).

## Security & Configuration Notes

- The app expects `NOTES_TOKEN` to be set for auth; do not commit secrets.
- The Android client uses a bearer token in `app/android/app/src/main/java/com/bartmann/noteseditor/AppConfig.kt`; update it locally and keep it out of commits when possible.
