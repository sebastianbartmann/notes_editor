# Pi Gateway (Sidecar)

Local TypeScript sidecar for `runtime_mode=gateway_subscription`.

## Endpoints

- `GET /health`
- `POST /v1/chat-stream` (NDJSON stream)
- `POST /v1/runs/:run_id/tool-result`

## Run

```bash
cd pi-gateway
npm install
npm run build
PI_GATEWAY_PORT=4317 npm start
```

Default mode is `claude_cli` (subscription/token-based via Claude CLI) with tool-bridge enabled.

Before first use:

```bash
claude setup-token
```

Optional env vars:

- `PI_GATEWAY_MODE=claude_cli|mock` (default `claude_cli`)
- `PI_GATEWAY_CLAUDE_BIN=/path/to/claude`
- `PI_GATEWAY_CLAUDE_MODEL=claude-sonnet-4-5`
- `PI_GATEWAY_CLAUDE_DISABLE_TOOLS=true` (default true; gateway handles tool execution via Go)
- `PI_GATEWAY_DEFAULT_MAX_TOOL_CALLS=20`
- `PI_GATEWAY_TOOL_RESULT_TIMEOUT_MS=20000`

Server `.env` settings:

- `PI_GATEWAY_URL=http://127.0.0.1:4317` (defaulted by server if unset)
- `AGENT_ENABLE_PI_FALLBACK=true|false`
- `AGENT_MAX_RUN_DURATION=2m`
- `AGENT_MAX_TOOL_CALLS_PER_RUN=40`

## Tool-bridge behavior (`claude_cli` mode)

The gateway asks Claude CLI for a structured next step on each loop:

- `{"kind":"tool_call","tool":"...","args":{...}}`
- `{"kind":"final","text":"..."}`

On `tool_call`, gateway emits NDJSON `tool_call`, waits for Go to POST tool result, then continues with that tool output in context until final answer.

## Mock mode quick trigger

Send a message containing:

`[[tool:read_file {"path":"notes/test.md"}]]`

The gateway emits a `tool_call`, waits for Go to post `/v1/runs/:run_id/tool-result`, then continues the stream.
