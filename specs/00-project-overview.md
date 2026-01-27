# Project Overview

> Status: Active
> Version: 2.0
> Last Updated: 2026-01-27

## Purpose

Notes Editor is a personal notes and daily journaling application with integrated AI assistant capabilities. It provides:

- **Daily notes** with structured todo management (work/private categories)
- **Sleep tracking** for children
- **Claude AI chat** with tool use (file operations, web search, LinkedIn integration)
- **File browser** for markdown vault management
- **White noise generator** for focus/sleep

The system is designed for a small user base (family members) with data stored in git-synced markdown files.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Clients                                  │
│  ┌─────────────────────┐          ┌─────────────────────┐       │
│  │   Android App       │          │   React Web App     │       │
│  │   (Kotlin/Compose)  │          │   (Vite/TypeScript) │       │
│  └──────────┬──────────┘          └──────────┬──────────┘       │
│             │                                 │                  │
└─────────────┼─────────────────────────────────┼──────────────────┘
              │         REST API (JSON)         │
              └─────────────────┬───────────────┘
                                │
┌───────────────────────────────┼───────────────────────────────────┐
│                               │                                    │
│  ┌────────────────────────────▼────────────────────────────────┐  │
│  │                     Go Backend Server                        │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐    │  │
│  │  │ HTTP API │ │  Vault   │ │  Claude  │ │   LinkedIn   │    │  │
│  │  │ handlers │ │  Store   │ │  Service │ │   Service    │    │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────┘    │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                    Markdown Vault (Git)                      │  │
│  │    ~/notes/{person}/daily/*.md, sleep_times.md, etc.        │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                    │
│                          Server Host                               │
└────────────────────────────────────────────────────────────────────┘
```

## Technology Stack

### Backend (Go)
- **Language:** Go 1.22+
- **HTTP Router:** Chi or standard library
- **Testing:** Go testing package with table-driven tests
- **External APIs:** Claude API, LinkedIn API
- **Storage:** Filesystem (markdown) + Git CLI

### Web Client (React)
- **Build:** Vite
- **Language:** TypeScript
- **Framework:** React 18+
- **Styling:** CSS modules or Tailwind
- **State:** React Context + hooks (no Redux needed)

### Android Client (Kotlin)
- **UI:** Jetpack Compose
- **Networking:** Ktor client
- **Architecture:** Single-activity, screen-based navigation

## Folder Structure

```
notes_editor/
├── server/                    # Go backend
│   ├── cmd/
│   │   └── server/
│   │       └── main.go        # Entry point
│   ├── internal/
│   │   ├── api/               # HTTP handlers
│   │   ├── vault/             # File storage + git
│   │   ├── claude/            # Claude AI service
│   │   ├── linkedin/          # LinkedIn integration
│   │   └── auth/              # Authentication
│   ├── go.mod
│   └── go.sum
│
├── clients/
│   ├── web/                   # React frontend
│   │   ├── src/
│   │   │   ├── components/
│   │   │   ├── pages/
│   │   │   ├── hooks/
│   │   │   ├── api/
│   │   │   └── App.tsx
│   │   ├── package.json
│   │   └── vite.config.ts
│   │
│   └── android/               # Android app
│       └── app/
│           └── src/main/
│               └── java/...
│
└── specs/                     # Specifications
```

## Design Principles

### 1. API-First
The Go server defines the contract. Both clients consume the same REST API. This enables:
- Independent client development
- Comprehensive server-side testing
- Clear separation of concerns

### 2. Server-Side Testability
All business logic lives in the Go server where it can be unit and integration tested. Clients are thin presentation layers.

### 3. Markdown as Storage
Notes are plain markdown files in a git repository. This provides:
- Human-readable data
- Version history via git
- Portability (no database lock-in)
- Conflict resolution via git merge

### 4. Multi-Person Support
Data is scoped by person (`sebastian/`, `petra/`). The `X-Notes-Person` header determines which vault subdirectory to use.

### 5. Offline-Tolerant
Git sync failures don't block operations. Changes are committed locally and pushed when connectivity allows.

## Authentication

Simple bearer token authentication:
- Token stored in environment variable
- Constant-time comparison to prevent timing attacks
- No user accounts or registration (family use only)

## Related Specifications

| Spec | Description |
|------|-------------|
| [01-rest-api-contract](./01-rest-api-contract.md) | Complete API documentation |
| [02-vault-storage-git-sync](./02-vault-storage-git-sync.md) | File storage and git operations |
| [03-android-app-architecture](./03-android-app-architecture.md) | Android client architecture |
| [04-claude-service](./04-claude-service.md) | Claude AI integration |
| [05-linkedin-service](./05-linkedin-service.md) | LinkedIn API integration |
| [19-go-server-architecture](./19-go-server-architecture.md) | Go server structure and testing |
| [20-react-web-client](./20-react-web-client.md) | React web client structure |
