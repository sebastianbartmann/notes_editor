0a. Study `specs/*` with up to 5 parallel Sonnet subagents to learn the application specifications.

1. Use up to 10 parallel Sonnet subagents to find all relevant code for the user's request.
2. Ask clarifying questions until no open questions remain.
3. Use an Opus subagent to create/update spec using format from existing specs. Ultrathink. For minor tasks which do not require spec updates, add todos to @IMPLEMENTATION_PLAN.md.

IMPORTANT: Do NOT implement anything. Do NOT assume functionality is missing; confirm with code search first.
