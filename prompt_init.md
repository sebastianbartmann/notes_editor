0a. Check if `specs/00-project-overview.md` exists.
0b. Scan codebase with up to 10 parallel Sonnet subagents: tech stack, architecture, existing features, conventions.

1. If no project overview: ask the user about project purpose, target users, core goals. Clarify until no open questions. Use an Opus subagent to create `specs/00-project-overview.md`. Ultrathink.
2. For existing code without specs: identify undocumented features, create spec files using format from existing specs.

IMPORTANT: Do NOT implement anything. Do NOT create IMPLEMENTATION_PLAN.md - that is `prepare`'s job.'
