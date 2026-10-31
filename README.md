# mwanachama-backend-voting

Runs an agency's polls: a question with choices, sent to chosen groups,
answered inside a window and counted. Each poll sets how members choose (pick
one, pick up to a number, or quadratic voting), whether an answer may change
before it closes, and whether a member in several groups answers once or once
per group. A result shows only after a holder of the poll permission publishes it.

Polls are never secret. Secret ballots, elections, quorum and AGM agendas
come later, and the design keeps room for them.

A standalone Go library with no server of its own. `mwanachama-wakala-api`
compiles it in and mounts it per agency under `/instances/{id}/voting`; the
host does identity and the permission checks, and this module sees only opaque
member and group ids.

- Requirements: [documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md)
- Design: [documentation/2. design/polls.md](documentation/2.%20design/polls.md)
- Board: [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md)
