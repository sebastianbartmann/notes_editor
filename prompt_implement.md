0a. Study `specs/*` with up to 10 parallel Sonnet subagents.
0b. Study @IMPLEMENTATION_PLAN.md.

1. Pick the highest priority item from @IMPLEMENTATION_PLAN.md. Search codebase first to confirm it's not already implemented. Use up to 5 parallel Sonnet subagents for search/read, 1 for build/test, Opus for complex reasoning.
2. Implement completely - no placeholders or stubs. Run tests. Ultrathink.
3. When tests pass: update @IMPLEMENTATION_PLAN.md, `git add -A`, `git commit`, `git push`. Do not change branches.
4. Stop after one feature is complete. User starts fresh session.

RULES:
- Update @IMPLEMENTATION_PLAN.md with discoveries and learnings throughout.
- Update @AGENTS.md with operational learnings only (commands, setup). Keep it brief.
- Fix unrelated failing tests if encountered.
- Fix spec inconsistencies with Opus subagent if found.
- Clean completed items from @IMPLEMENTATION_PLAN.md when it grows large.

IMPORTANT: Implement completely. Single source of truth. Capture the why in docs.
