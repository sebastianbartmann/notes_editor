# Notes Editor - Project Overview

## Project Purpose

Notes Editor is a personal "second brain" application designed for capturing daily notes, managing tasks, and organizing personal knowledge. It provides both a web interface (FastAPI + HTMX) and a native Android client, with all data stored in a git-synced local vault.

## Target Users

- **Primary:** The owner (Sebastian) - full daily use for task management and knowledge organization
- **Secondary:** Wife (Petra) - uses specific features to a lesser degree
- Multi-person support allows each user to have their own isolated data space

## Core Goals

1. **Task Management with Carryover** - Daily notes with categorized tasks (work/private) that automatically carry forward incomplete items
2. **File Organization / Personal Knowledge Base** - Person-scoped file tree for organizing documents, notes, and personal information
3. **Deeper LLM/Personal Assistant Integration** - Evolving toward smarter assistance through Claude integration and tool capabilities

## Tech Stack Overview

| Layer | Technology |
|-------|------------|
| Backend | FastAPI + Jinja2 + HTMX, Python 3.12+, uvicorn |
| Storage | File-based vault (`~/notes/`), git-synced, no database |
| Web Frontend | Vanilla JS + HTMX 1.9.10, custom CSS (dark/light themes) |
| Android | Jetpack Compose, OkHttp, Kotlin 1.9.24, MediaSession |

## Key Features

- **Daily Notes** - Auto-created markdown files with tasks, pinned entries, and incomplete task carryover
- **File Browser** - Person-scoped tree navigation with create/read/edit/delete and markdown rendering
- **Sleep Tracking** - Shared log for children's sleep events
- **Noise Playback** - Rain sounds via Web Audio API (web) and foreground service with media controls (Android)
- **Claude Chat** - Agent SDK integration with file tools (Read/Write/Edit/Glob/Grep) and WebSearch
- **LinkedIn Tools** - OAuth authentication, post creation, comment reading/posting via MCP server

## Architecture Principles

- **File-Based Storage** - All data stored as files in `~/notes/` vault, no database required
- **Git Sync** - All writes auto-commit and push to git for version history and backup
- **Multi-Person Support** - Person selection via `X-Notes-Person` header/cookie, data isolated to `{person}/` subfolders
- **Stateless Server** - No server-side session state; preferences stored client-side (cookies/SharedPreferences)
- **Platform Parity** - Android mirrors web features with native controls where appropriate

## Future Direction

The primary evolution path is deeper LLM and personal assistant integration. This includes:
- Enhanced Claude tool capabilities for personal data interaction
- Smarter task suggestions and automation
- More sophisticated knowledge base queries and organization assistance
