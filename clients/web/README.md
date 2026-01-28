# Notes Editor Web Client

React web client for the Notes Editor application.

## Prerequisites

- Node.js 18+
- npm

## Setup

Install dependencies:

```bash
npm install
```

## Development

**Start the development server:**

```bash
npm run dev
```

The dev server starts on http://localhost:5173 with hot reload. API requests are proxied to the Go backend at http://localhost:8080.

**Run the Go backend** (in a separate terminal):

```bash
cd ../../server && go run ./cmd/server
```

Or from the repository root:

```bash
make server
```

## Scripts

| Script | Description |
|--------|-------------|
| `npm run dev` | Start Vite dev server with API proxy |
| `npm run build` | TypeScript check and production build |
| `npm run preview` | Preview production build locally |
| `npm test` | Run tests once |
| `npm run test:watch` | Run tests in watch mode |

## Testing

Run all tests:

```bash
npm test
```

Run tests in watch mode during development:

```bash
npm run test:watch
```

Test files are co-located with source files using the `*.test.ts(x)` naming convention.

**Test coverage:**
- `NoteView.test.tsx` - Line parsing, task toggle, pinned markers
- `AuthContext.test.tsx` - Token persistence, login/logout
- `ThemeContext.test.tsx` - Theme switching, body class
- `PersonContext.test.tsx` - Person selection persistence
- `claude.test.ts` - NDJSON streaming, event parsing
- `client.test.ts` - API client, headers, errors

## Production Build

Build for production:

```bash
npm run build
```

Output is written to `dist/`. The root Makefile copies this to `server/static/` for deployment:

```bash
# From repository root
make build-web
```

## Architecture

```
src/
├── api/           # API client modules
│   ├── client.ts  # Base HTTP client with auth
│   ├── types.ts   # TypeScript type definitions
│   ├── daily.ts   # Daily note endpoints
│   ├── files.ts   # File management endpoints
│   ├── todos.ts   # Todo endpoints
│   ├── sleep.ts   # Sleep tracking endpoints
│   └── claude.ts  # Claude AI streaming
├── components/    # Reusable UI components
│   ├── Layout/    # App layout, header, navigation
│   ├── NoteView/  # Markdown note renderer
│   ├── Editor/    # Text editor
│   └── FileTree/  # Directory browser
├── context/       # React context providers
│   ├── AuthContext.tsx
│   ├── PersonContext.tsx
│   └── ThemeContext.tsx
├── hooks/         # Custom React hooks
│   ├── useAuth.ts
│   ├── usePerson.ts
│   └── useTheme.ts
├── pages/         # Page components
│   ├── LoginPage.tsx
│   ├── DailyPage.tsx
│   ├── FilesPage.tsx
│   ├── SleepPage.tsx
│   ├── ClaudePage.tsx
│   ├── NoisePage.tsx
│   └── SettingsPage.tsx
├── App.tsx        # Root component with routing
├── main.tsx       # Entry point
└── index.css      # Global styles and theme
```

See [specs/20-react-web-client.md](../../specs/20-react-web-client.md) for detailed architecture documentation.

## Theming

The app supports dark (default) and light themes. Theme variables are defined in `src/index.css`:

- Dark theme: Applied by default
- Light theme: Applied when `body` has class `theme-light`

See [specs/12-theming-styling-system.md](../../specs/12-theming-styling-system.md) for color specifications.
