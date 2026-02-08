# Notes Editor Agent Platform Migration Blueprint

## Status Snapshot (as of 2026-02-08)

Current implementation status:

- Completed: Agent API surface (`/api/agent/*`) with NDJSON v2 stream events and `/api/claude/*` compatibility bridge.
- Completed: Web and Android clients now use `/api/agent/chat-stream` and v2 event types.
- Completed: Per-person `agents.md` editing and file-backed actions (`agent/actions`) with confirmation metadata support.
- Completed: Dynamic `VALID_PERSONS` server config (replacing static person list behavior at runtime).
- Not completed: Pi gateway runtime (sidecar + Go adapter). Anthropic-backed Claude runtime still executes LLM calls.
- Not completed: LinkedIn hardening items, full runtime fallback policy, and final cutover/removal of legacy paths.

Important: the app now has an "agent platform layer", but Pi is not yet the core LLM driver.

## 1) Scope and Goal

This document defines the target architecture and implementation plan to replace the current hand-rolled Claude bot with a more capable agent platform that:

- Streams responses in Web and Android.
- Supports provider execution via API key mode and subscription-backed Pi agent mode.
- Keeps strict per-person vault isolation.
- Restores and hardens LinkedIn tool access.
- Gives each person an `agent/` workspace for agent-authored notes/artifacts.
- Makes system prompt (`agents.md`) editable in-app.
- Adds one-click prompt actions (file-backed buttons) for recurring automations.

Primary deliverable of the implementation project: a production-ready `Agent` subsystem across server, web, and Android, replacing the existing `/api/claude/*` UX and internals.

## 2) Current State Audit (from this repo, updated 2026-02-08)

### Backend

- New Agent service exists (`server/internal/agent/*`) with:
  - v2 stream events (`events.go`)
  - orchestration + run IDs + timeout/cancel hooks (`service.go`)
  - action file parsing/execution support (`actions.go`)
  - per-person agent config (`config.go`)
- New API handlers exist in `server/internal/api/agent.go`:
  - `POST /api/agent/chat`
  - `POST /api/agent/chat-stream`
  - `POST /api/agent/session/clear`
  - `GET /api/agent/session/history`
  - `POST /api/agent/stop`
  - `GET /api/agent/config`
  - `POST /api/agent/config`
  - `GET /api/agent/actions`
  - `POST /api/agent/actions/{id}/run`
- Legacy `/api/claude/*` endpoints are now compatibility wrappers to the Agent service.
- LLM execution is still Anthropic key mode via existing Claude runtime (`internal/claude/*`).
- LinkedIn OAuth callback and token persistence exist (`server/internal/api/linkedin.go`, `server/internal/linkedin/*`) but hardening items are still pending.
- Person validation now supports dynamic config via `VALID_PERSONS` in env (applied through `auth.SetValidPersons(...)`).

### Web

- `clients/web/src/pages/ClaudePage.tsx` now streams from `/api/agent/chat-stream` and handles v2 events.
- Quick action buttons are loaded from `/api/agent/actions` and can run action prompts.
- Settings page supports per-person agent runtime mode + prompt editing via `/api/agent/config`.
- `clients/web/src/api/agent.ts` and tests exist (`clients/web/src/api/agent.test.ts`).

### Android

- Chat screen now uses `/api/agent/chat-stream` and v2 events.
- Action chips are sourced from `/api/agent/actions` and can run actions (with explicit in-app confirmation flow for flagged actions).
- Settings screen now supports runtime mode + prompt editing via `/api/agent/config`.
- Navigation labels updated from "Claude" to "Agent" (route now `tool-agent`).

### Immediate regressions to account for

- Resolved: event-type contract drift across backend/web/android.
- Remaining: provider lock-in to Anthropic API key mode only (Pi mode not yet implemented).
- Partially resolved: first-class prompt/actions model exists; agent workspace policy hardening is still pending.

## 3) Target Architecture

## 3.1 High-level

Adopt a split architecture:

1. **Notes Server (Go) remains policy and data boundary**
- Auth, person scoping, vault access, LinkedIn credentials, and tool permissions stay in Go.
- UI clients never directly access external model providers.

2. **Agent Runtime Adapter (new module in Go)**
- New provider-agnostic interface (`internal/agent`) abstracts execution.
- Two runtime backends:
  - `AnthropicKeyRuntime` (compatibility + fallback).
  - `PiGatewayRuntime` (subscription mode via Pi agent stack).

3. **Pi Gateway (new sidecar service, TypeScript)**
- Wrap Pi stack (from `pi-mono` ecosystem) and expose local streaming endpoint.
- Handles subscription-authenticated sessions.
- Receives normalized prompts + tool results from Go server.

4. **Tool execution remains server-side in Go**
- Runtime requests tool calls; Go executes tools with person-scoped policies.
- Prevents privilege bypass from runtime/provider layer.

## 3.2 Why this split

- Keeps vault and LinkedIn security in one trusted layer.
- Avoids rewriting the full Go app in TS while still using Pi runtime internals.
- Enables incremental migration with compatibility mode.

## 4) Canonical Contracts

## 4.1 Unified stream event schema (v2)

Use one schema across backend + both clients:

```json
{"type":"start","session_id":"...","run_id":"..."}
{"type":"text","delta":"..."}
{"type":"tool_call","tool":"read_file","args":{...}}
{"type":"tool_result","tool":"read_file","ok":true,"summary":"..."}
{"type":"status","message":"..."}
{"type":"error","message":"..."}
{"type":"done","session_id":"...","run_id":"..."}
```

Rules:
- Remove ambiguous legacy `tool` event type.
- Keep backward adapter for old clients during migration window.
- Emit `start` early so clients can persist session state immediately.

## 4.2 Server API surface (new)

Add `/api/agent/*` and keep `/api/claude/*` as compatibility wrappers until cutover.

- `POST /api/agent/chat-stream`
  - body: `{ "session_id?": "...", "message": "...", "action_id?": "..." }`
  - response: NDJSON stream events v2.
- `POST /api/agent/chat`
  - non-stream fallback.
- `POST /api/agent/session/clear`
- `GET /api/agent/session/history?session_id=...`
- `GET /api/agent/config`
- `POST /api/agent/config`
  - stores/editable settings per person (prompt path, runtime mode, feature flags).
- `GET /api/agent/actions`
- `POST /api/agent/actions/{id}/run`

## 4.3 Tool call contract between runtime and server

Standardize tool envelope (runtime-agnostic):

```json
{
  "id": "call_123",
  "name": "write_file",
  "arguments": {"path":"agent/notes.md","content":"..."}
}
```

Go returns:

```json
{
  "id": "call_123",
  "ok": true,
  "content": "File written successfully",
  "error": ""
}
```

## 5) Data and File Layout

Per person (`<vault>/<person>/`):

- `agents.md` (editable system prompt for this person).
- `agent/actions/` (file-backed prompt actions; one file = one button).
- `agent/`
  - `memory.md` (long-lived agent notes).
  - `scratch/` (short-lived temp working files).
  - `runs/YYYY/MM/DD/<run-id>.md` (artifacts/reports).

Server-global:
- `.env` for secrets and runtime toggles.

## 5.1 Action file convention

Each file in `<vault>/<person>/agent/actions/` becomes one runnable button.

Conventions:

- Label: file stem (filename without extension).
- Allowed extensions: `.md`, `.prompt.md`.
- Action ID: slug derived from filename (for API route use).
- Prompt: file body is sent as the action prompt when clicked.
- Optional metadata: YAML front matter in the file.

Example file `Extract yesterday URLs.prompt.md`:

```md
---
requires_confirmation: true
---
Go through yesterday's notes and extract all URLs, then write excerpts into a new file in `agent/runs/{{date}}-url-digest.md`.
```

## 6) Security Model

Non-negotiable boundaries:

- Person identity comes only from authenticated request context (`X-Notes-Person` + middleware), never from runtime payload.
- All vault paths must resolve through `vault.ResolvePath` semantics.
- Tool allowlist enforced server-side per runtime request.
- Agent write scope defaults to:
  - read: full person vault,
  - write: full person vault, but destructive actions (`delete_file`, overwrite existing critical files) require explicit confirmation mode.
- LinkedIn actions guarded by explicit tool calls and logged to vault.
- Reject symlink escapes and hidden-path abuse in new tool paths (add explicit checks).

## 7) Provider Strategy

## 7.1 Modes

- `runtime_mode = anthropic_api_key`
- `runtime_mode = gateway_subscription`

Server chooses mode per request/person based on config.

Current status:
- Runtime mode is persisted per person (`/api/agent/config`) and exposed in Web/Android settings.
- Actual runtime backend switching is not implemented yet.
- Current execution path remains Anthropic-backed Claude runtime.

## 7.2 Pi subscription integration

Implement a local `Pi Gateway` service (Node/TypeScript) that:

- Manages Pi auth/session state.
- Exposes local HTTP streaming endpoint consumed by Go.
- Converts Go tool messages to Pi runtime format and back.

Go adapter (`internal/agent/runtime_pi_gateway.go`) responsibilities:
- Start request stream.
- Receive model events.
- Intercept tool calls and execute via Go `ToolExecutor`.
- Send tool results back to gateway.

## 7.3 Tool execution ownership and session lifecycle

Execution ownership:

- Runtime (Anthropic or Pi) decides tool selection (`tool name + args`).
- Go server is the only component allowed to execute side-effecting tools.
- Go enforces policy before execution (person scope, path safety, allowlist, confirmation requirements).
- Go returns normalized tool results; runtime consumes those results and continues reasoning.

Session layering:

- **Application session (source of truth)** lives in Go (`internal/agent/session.go`).
- **Runtime session (implementation detail)** lives in provider runtime (Anthropic message thread state or Pi session file/state).
- Go stores a mapping: `person + app_session_id -> runtime_session_id`.

Lifecycle per request:

1. Client sends `session_id` (or empty for new conversation) to `/api/agent/chat-stream`.
2. Go resolves person from auth context and loads/creates app session.
3. Go loads or creates mapped runtime session.
4. Runtime emits text + tool calls.
5. Go executes tools and returns results into the same runtime session.
6. On completion, Go persists app-level history and emits `done`.

Scoping rules:

- Session keys are namespaced by `person` to prevent cross-person leakage.
- Concurrent calls for same `person + session_id` must use a per-session lock.
- Runtime-side persistence (e.g., Pi session files) is treated as cache/state backend only; app truth is Go-managed.

## 7.4 Execution safeguards (minimum required)

1. Cancellation and timeout:
- Add client-driven cancellation (disconnect or explicit stop) that terminates active runtime stream and tool execution loop.
- Add hard `max_run_duration` timeout (server-configurable) and return a terminal `error` then `done`.

2. Concurrency policy:
- For the same `person + session_id`, allow only one active run.
- New requests on a busy session should return a deterministic conflict error (or queue later; start with reject).

3. Action metadata mini-spec:
- Support YAML front matter in action files with:
  - `requires_confirmation: boolean`
  - `max_steps: integer` (optional)
- Invalid front matter returns validation error and action does not run.

4. Provider fallback rule:
- If `runtime_mode=gateway_subscription` and gateway runtime is unavailable/auth-invalid:
  - if Anthropic key fallback is enabled, switch to `anthropic_api_key` for that run and emit `status`.
  - if fallback disabled, return explicit error with recovery hint.

5. Basic limits:
- Enforce `max_tool_calls_per_run`.
- Enforce `max_run_duration`.
- Enforce `max_prompt_bytes` for action file prompt payloads.

## 7.5 External references

- `clawdbot` org now points to `openclaw/openclaw`: https://github.com/openclaw/openclaw
- `pi-mono` monorepo: https://github.com/badlogic/pi-mono

Use these repos mainly for runtime/session/auth patterns, not for vault security logic (security stays in this codebase).

## 8) Backend Implementation Plan (Go)

## 8.1 New packages

- `server/internal/agent/`
  - `service.go` (high-level orchestration)
  - `runtime.go` (interface) - pending
  - `runtime_anthropic.go` - pending (currently bridged through `internal/claude`)
  - `runtime_pi_gateway.go` - pending
  - `events.go` (stream schema v2)
  - `actions.go`
  - `config.go`
  - `session.go` - pending
  - `tools.go` (migrated + hardened from `internal/claude/tools.go`) - pending

- `server/internal/api/agent.go`
  - new handlers for `/api/agent/*`.

## 8.2 Keep/bridge old endpoints

- Keep `server/internal/api/claude.go` as temporary compatibility layer:
  - internally call agent service.
  - map `tool_call` => `tool_use` for old clients only until both clients migrate.

## 8.3 LinkedIn reliability hardening

- On settings save, reload runtime config and reinitialize LinkedIn + Agent services.
- Add health endpoint returning LinkedIn config state (token present + last validation status).
- Add retry/backoff and clear error messages for LinkedIn API 401/403.

## 8.4 Person model upgrade

Current static person list is in code. Move to dynamic server config:
- `VALID_PERSONS=sebastian,petra,...` in env.
- Keep tests that enforce isolation regardless of person source.

## 9) Web App Plan

Files to add/update:

- Add `clients/web/src/pages/AgentPage.tsx` (new default experience).
- Add `clients/web/src/api/agent.ts` and `types` updates for stream v2.
- Add settings UI section for:
  - runtime mode,
  - system prompt editor (`agents.md`),
  - action folder info and action refresh behavior.
- Keep `ClaudePage` as alias or remove after migration.

Current status:
- `clients/web/src/api/agent.ts` is implemented.
- `clients/web/src/pages/ClaudePage.tsx` now behaves as Agent UX (still file-named "ClaudePage").
- Dedicated `AgentPage.tsx` rename is pending.

UX requirements:
- Token-by-token streaming with auto-scroll.
- Tool timeline panel (tool call/result events).
- Quick action buttons generated from `agent/actions/` files.
- Confirmation dialog when action metadata indicates risky/destructive behavior.

## 10) Android Plan

Files to add/update:

- Replace `ToolClaudeScreen` with `ToolAgentScreen` or evolve in place.
- Update `ApiClient.claudeChatStream` -> `agentChatStream` using new endpoint/events.
- Fix event handling mismatch (`tool` -> `tool_call` / `tool_result`).
- Add action chips/buttons sourced from `agent/actions/`.
- Add agent settings in `SettingsScreen` (runtime mode, prompt, action folder hint).

Current status:
- Evolved in place: `ToolClaudeScreen` now runs Agent flow with v2 events and action chips.
- `ApiClient.agentChatStream` and Agent config/actions methods are implemented.
- Runtime mode + prompt editor added to `SettingsScreen`.
- Class/file rename from Claude to Agent remains pending cleanup.

## 11) Testing Strategy

## 11.1 Backend

- Unit tests for runtime interface and adapters.
- Tool security tests:
  - traversal,
  - symlink escape,
  - person isolation,
  - destructive action confirmations.
- Stream contract tests for NDJSON v2 sequence correctness.
- LinkedIn integration tests with mocked HTTP client.

## 11.2 Web

- Stream parser tests for v2 event types.
- Agent page interaction tests:
  - streaming accumulation,
  - action-file run,
  - settings save/load,
  - confirmation behavior.

## 11.3 Android

- Event parser tests for v2 types.
- Compose UI tests for chat + action buttons.
- Maestro flow update for Agent screen and action trigger.

## 12) Migration and Rollout

Phase 1: Foundations
- Add `internal/agent` with Anthropic runtime only.
- Introduce `/api/agent/*` and v2 stream schema.
- Keep `/api/claude/*` compatibility mapping.

Status: Completed.

Phase 2: Client migration
- Web + Android switch to `/api/agent/*`.
- Fix event mismatch regression.
- Add editable prompt + action-buttons UI.

Status: Mostly completed.
Remaining in phase 2:
- Final naming cleanup (`ClaudePage`/`ToolClaudeScreen` -> `Agent*`).
- Optional UX polish (tool timeline panel and richer action metadata UX).

Phase 3: Pi gateway mode
- Ship sidecar service and Go adapter.
- Add runtime mode selector.
- Validate subscription auth flow and reconnection behavior.

Status: Not started for runtime execution path.
Note: selector UI exists, but does not yet switch backend runtime.

Phase 4: Hardening and cutover
- Remove deprecated stream events and old endpoint internals.
- Expand tests and add operational docs.

Status: Not started.

## 13) Work Split for Parallel Agents

Recommended parallel workstreams:

1. **Agent-Core (Go backend)**
- Build `internal/agent`, `/api/agent/*`, stream v2, compatibility bridge.

2. **Tooling-Security (Go backend)**
- Migrate/harden tools, add `agent/` workspace policy, destructive-action confirmation rules.

3. **Web-Agent UX**
- New Agent page, action buttons from files, prompt editor, settings integration.

4. **Android-Agent UX**
- Stream v2 integration, new screen/components, settings integration.

5. **Pi-Gateway Integration (TS + Go adapter)**
- Sidecar service, runtime adapter, auth/session handling, tool-bridge protocol.

6. **QA/Automation**
- Cross-layer tests, regression checks, Maestro updates, rollout checklist.

## 14) Risks and Mitigations

- **Runtime complexity (Go + TS sidecar)**
  - Mitigation: strict local protocol contract and adapter tests.

- **Subscription auth fragility**
  - Mitigation: explicit health states in UI, re-auth flow, clear fallback to Anthropic key mode.

- **Accidental vault damage from powerful tools**
  - Mitigation: confirmation gates, dry-run mode for high-risk actions, write-audit logs in `agent/runs`.

- **LinkedIn API churn**
  - Mitigation: versioned client wrapper, centralized error mapping, test fixtures.

## 15) Definition of Done

- Web and Android both support stable streaming agent chat via `/api/agent/chat-stream`.
- Per-person prompt (`agents.md`) editable in both clients.
- Action-file buttons available and runnable with safe confirmation.
- Agent writes artifacts under per-person `agent/` directory.
- LinkedIn tools functional again and covered by tests.
- Pi subscription mode functional (plus Anthropic fallback).
- `/api/claude/*` either fully bridged or removed after successful cutover.

Current status by DoD item:

- Done: Web and Android support stable streaming agent chat via `/api/agent/chat-stream`.
- Done: Per-person prompt (`agents.md`) editable in both clients.
- Done: Action-file buttons available and runnable with confirmation flow.
- Pending: Agent write-artifact policy under `agent/runs/...` not fully formalized/automated yet.
- Pending: LinkedIn hardening and test expansion.
- Pending: Pi subscription mode and fallback execution.
- Partial: `/api/claude/*` is bridged; removal/cutover not done.

## 17) Next Session Starting Point

Priority order:

1. Implement Pi gateway sidecar (TypeScript) and Go runtime adapter (`runtime_pi_gateway.go`).
2. Introduce provider runtime interface (`runtime.go`) and explicit anthropic adapter (`runtime_anthropic.go`).
3. Wire runtime mode switching from per-person config with fallback/status behavior.
4. Add per-session concurrency lock policy and deterministic conflict handling.
5. Complete hardening tasks: LinkedIn health/reload and final legacy endpoint cutover plan.

## 16) Decisions Resolved

Resolved:

1. `agents.md` is per person: `<vault>/<person>/agents.md`.
2. Default agent write scope is full person vault (not `agent/`-only).
3. `agent/` is unrestricted workspace for agent-owned artifacts and notes.
4. Pi gateway deployment mode is bundled with the server process manager.
5. Action buttons are per-person files under `<vault>/<person>/agent/actions/` (filename drives label).
