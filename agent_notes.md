## 2026-02-11

- `notes-editor.service` currently has `Requires=notes-pi-gateway.service`; if gateway crashes, notes-editor restarts and in-memory sync state (`last_pull_at`) resets.
- On prod, pi-gateway needed Node 20+; system unit was using `/usr/bin/node` (v18), causing restart loops.
- Manual git workflow endpoints were added (`/api/git/status`, `/api/git/commit`, `/api/git/push`, `/api/git/pull`, `/api/git/commit-push`) to support explicit sync control from app UI.
- Manual pull endpoint uses ff-only semantics and returns actionable failure text on divergence/conflicts.
