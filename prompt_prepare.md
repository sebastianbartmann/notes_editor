0a. Study `specs/*` with up to 10 parallel Sonnet subagents to learn the application specifications.
0b. Study @IMPLEMENTATION_PLAN.md if present.

1. Use up to 10 Sonnet subagents to compare source code against specs. Look for: TODOs, placeholders, skipped/flaky tests, missing implementations, inconsistent patterns.
2. Use an Opus subagent to analyze findings and update @IMPLEMENTATION_PLAN.md as a prioritized bullet list. Ultrathink.
3. If spec gaps are discovered, create spec files first - implementation planning depends on complete specs.

IMPORTANT: Do NOT implement anything. Confirm assumptions with code search.
