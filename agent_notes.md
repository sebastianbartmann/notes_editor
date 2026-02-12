## 2026-02-11

- `notes-editor.service` currently has `Requires=notes-pi-gateway.service`; if gateway crashes, notes-editor restarts and in-memory sync state (`last_pull_at`) resets.
- On prod, pi-gateway needed Node 20+; system unit was using `/usr/bin/node` (v18), causing restart loops.
- Manual git workflow endpoints were added (`/api/git/status`, `/api/git/commit`, `/api/git/push`, `/api/git/pull`, `/api/git/commit-push`) to support explicit sync control from app UI.
- Manual pull endpoint uses ff-only semantics and returns actionable failure text on divergence/conflicts.
- Agent session state (web + Android) currently tracks only one active session ID per person/client; cross-device continuation exists only if session ID is manually reused.
- Agent API supports `GET /api/agent/session/history?session_id=...` and clear-by-ID, but has no person-scoped session listing/metadata endpoint yet (needed for session picker UI).
- Potential risk: `agent.Service.selectSessionRuntime()` prefers Anthropic runtime for history/clear if available, regardless of per-person runtime mode, which can misread session history when gateway runtime is active.
- In `runtime_mode=gateway_subscription`, Pi sidecar uses Anthropic OAuth ("Claude subscription") through Pi, and tool names are restricted to Claude Code canonical names only. The extension explicitly warns that custom tool names can cause auth rejection ("only authorized for Claude Code"), so it exposes only `ls`, `read`, `write`, `grep`, `glob`, `websearch`, `webfetch` and maps them to server canonical tools (`list_directory`, `read_file`, `write_file`, `search_files`, `glob_files`, `web_search`, `web_fetch`).
- Sidecar enforces extension-owned tools by launching Pi with `--no-tools` plus `--extension ...` (`pi-gateway/src/server.ts`), so broader backend tools (for example LinkedIn tools supported by `ToolExecutor`) are not directly available in subscription mode unless explicitly bridged via those allowed names.
- `search_files` is now qmd-backed in Go via MCP HTTP (`deep_search` on `http://prod.local:8181/mcp`), with a cached MCP session header. It is intentionally a hard dependency (no fallback): qmd unavailable/error means tool execution fails loudly.
- Added async qmd indexing manager (`qmd update` + `qmd embed`) with debounce and loud logging/error retention. It is wired to sync success hooks (background pull/push) and manual git flows (`RecordManualPull/Push`, plus reset-clean trigger).
- qmd MCP HTTP mode (`qmd mcp --http`) currently fails for repeated requests with `Stateless transport cannot be reused across requests` (500 from qmd Bun server). For reliability, `search_files` now uses direct CLI execution (`qmd query --json --line-numbers -c <person> -n 50`) from Go instead of MCP.
- qmd collection bootstrap detail: `qmd collection add <path> --name <person>` defaults to recursive `**/*.md` and works; manually forcing masks like `"*.md,*.txt"` caused empty collections in our vault layout. Index manager should ensure per-person collections exist before running `qmd update/embed`.
- For fast grep behavior, `search_files` now shells out to `qmd search --json --line-numbers` (not `qmd query`), because `query` can trigger heavyweight local model boot/download on first run.

## 2026-02-12

- Agent/Claude streaming now normalizes assistant output by dropping only leading blank lines (not all leading spaces) at the server runtime layer. Implemented via shared `internal/textnorm.LeadingBlankLineTrimmer`, wired into both Anthropic (`internal/claude/processStream`) and Pi gateway (`internal/agent/runtime_pi_gateway`) so web and Android clients receive consistent deltas and session history does not start with stray empty lines.
- Android header now mirrors web by showing both sync and index badges (`AppSync.status` + `AppSync.indexStatus`), both refreshed through `AppSync.refreshStatus()` polling.
- Removed agent action debug/server diagnostics text from Android Agent screen; keep only actionable user-facing status/error messaging.
- Added subscription-mode Bash bridge: Pi extension now exposes Claude Code-compatible `bash` and maps it to server canonical tool `run_bash` via `/api/agent/tools/execute`. Server executes `bash -lc` in person vault root only, with timeout (default 10s, max 60s), capped stdout/stderr buffers (64KiB each), and JSON-wrapped result payload (`<bash_result_json>...`).
