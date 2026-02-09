# Pi Gateway (Sidecar)

Local TypeScript sidecar for `runtime_mode=gateway_subscription`.

## Endpoints

- `GET /health`
- `POST /v1/chat-stream` (NDJSON stream)

## Run

```bash
cd pi-gateway
npm install
npm run build
PI_GATEWAY_PORT=4317 npm start
```

Default mode is `pi_rpc` (Pi coding-agent in RPC mode) with server-delegated tools.

Before first use (to authenticate your subscription provider):

```bash
pi
# inside Pi: /login
```

Requirements:
- Node.js >= 20 (Pi uses JS features not available in Node 18)

Optional env vars:

- `PI_GATEWAY_MODE=pi_rpc|mock` (default `pi_rpc`)
- `PI_GATEWAY_PI_PROVIDER=anthropic` (default `anthropic`)
- `PI_GATEWAY_PI_MODEL=...` (optional; default model chosen by Pi)
- `PI_GATEWAY_PI_SESSION_DIR=...` (default `~/.pi/notes-editor-sessions`)
- `PI_GATEWAY_PI_TIMEOUT_MS=120000`
- `PI_GATEWAY_PI_EXTENSION_PATH=...` (default `pi-gateway/src/pi-notes-editor-extension.ts`)
- `NOTES_SERVER_URL=http://127.0.0.1:8080` (used by the Pi extension to call back into the Go server)
- `NOTES_TOKEN=...` (used by the Pi extension; typically comes from `server/.env`)

Server `.env` settings:

- `PI_GATEWAY_URL=http://127.0.0.1:4317` (defaulted by server if unset)
- `AGENT_ENABLE_PI_FALLBACK=true|false`
- `AGENT_MAX_RUN_DURATION=2m`
- `AGENT_MAX_TOOL_CALLS_PER_RUN=40`

## Tool Execution

Pi executes tools via a bundled extension (`src/pi-notes-editor-extension.ts`) that delegates each canonical tool call back to the Go server at `POST /api/agent/tools/execute` with the normal auth + person scoping.

## Mock mode quick trigger

Mock mode returns a single canned text response without invoking Pi.
