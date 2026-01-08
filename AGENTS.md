# Notes Editor Overview

Notes Editor is a personal/family “second brain” app with a FastAPI + HTMX web UI and a native Android client. Data lives in a local vault with per-person subfolders (e.g., `sebastian/`, `petra/`), and each client selects its person root and theme locally while the server remains user‑agnostic.

Key features: daily notes with tasks and pinned entries, a file tree editor scoped to the selected person, and shared tools such as sleep tracking, noise playback, and a Claude tool. The Android app mirrors the web layout and adds native conveniences (media‑style noise controls, persistent settings).
