0a. Study `specs/*` with up to 10 parallel Sonnet subagents to learn the application specifications.
0b. Study @IMPLEMENTATION_PLAN.md (if present; it may be incorrect) to understand the plan so far.
0c. For reference, the application source code is in app/* for the kotlin android app, in server/* for the server and web interface.

1. Study existing codebase with up to 10 parallel Sonnet subagents and identify code without specs
3. Use an Opus subagent to analyze findings and create/update spec using format from `specs/01-rest-api-contract.md` Ultrathink.
2. Pick highest-priority unspecified component

ULTIMATE GOAL: Iteratively bring codebase under spec coverage. **One spec per session.**

IMPORTANT: Do NOT implement anything. Do NOT assume functionality is missing; confirm with code search first. Prefer consolidated, idiomatic implementations over ad-hoc copies.

1. Identify code without specs
2. Pick highest-priority unspecified component
3. Generate spec using format from `specs/01-rest-api-contract.md`
