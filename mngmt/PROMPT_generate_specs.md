0a. Study `specs/*` with up to 5 parallel Sonnet subagents to learn the application specifications.
0b. Study @IMPLEMENTATION_PLAN.md (if present; it may be incorrect) to understand the plan so far.
0c. For reference, the application source code is in app/* for the kotlin android app, in server/* for the server and web interface.

1. listen to the users request to change parts of this project
2. use up to 10 parallel Sonnet subagents to find all relevant code
3. discuss with the user all open questions by querying them until no open questions are left to be answered
4. Use an Opus subagent to analyze findings and create/update spec using format from `specs/01-rest-api-contract.md` Ultrathink. For minor tasks which do not require spec updates add todos into @IMPLEMENTATION_PLAN.md

ULTIMATE GOAL: define todos and create/update specs to later use to implement changes based on a conversation with the user

IMPORTANT: Do NOT implement anything. Do NOT assume functionality is missing; confirm with code search first. Prefer consolidated, idiomatic implementations over ad-hoc copies.
