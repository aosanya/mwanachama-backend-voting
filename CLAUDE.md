# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-voting

Runs an agency's polls: a question with choices, sent to chosen groups,
answered inside a window, counted, and published by a holder of the poll
power. Module path `github.com/aosanya/mwanachama-backend-voting`. Created
2026-10-22 (member app decisions 340, 341).

**A declared domain from the first commit.** The objects come from
`voting.blueprint.json` and the route table from `voting.operations.json`;
storage is `mwanachama-backend-shared/specstore` and the routes are built by
`mwanachama-backend-shared/dispatch.Table`. There are no row structs and no
`AutoMigrate`. `mwanachama-backend-catalog` is the reference;
`mwanachama-backend-files` is the nearest template for the file layout.

**Standalone library, no server of its own**, and **no auth model**: every
address is gated, and the mounting host decides who holds the poll power and
which groups a member belongs to.

## The rules that matter most

- **Polls are never secret, but the store is shaped for a secret ballot.**
  Who answered and what they chose are kept apart (see
  [documentation/2. design/polls.md](documentation/2.%20design/polls.md)).
  Never join the two into one object to save a query.
- **Members see only their own choice.** Counts and who chose what are for
  holders of the poll power until a result is published (decisions 344, 345).
- **This module names no domain and knows no people.** A member and a group
  are opaque ids the host passes in. Words such as "member", "role",
  "election" or "AGM" never appear in an identifier or a stored value;
  `domain_agnostic_test.go` enforces it.
- **The answering member comes from the caller**, never the body.
- **Each module stores and checks its own settings** (decisions 280, 281):
  poll defaults live here, in a definition layer and an override layer.

## Conventions

- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- `go test ./...` (sqlite) verifies a change.
- **Adding a field means editing `voting.blueprint.json` and the Go type in
  `models/` together.**
- An address is an entry in `voting.operations.json`.
- The six guard tests of the declared-domain standard, including
  `domain_agnostic_test.go`, stay green.
- No Go file over 300 lines; split by responsibility.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
