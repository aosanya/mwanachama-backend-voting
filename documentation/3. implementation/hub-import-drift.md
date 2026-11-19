# Hub import drift

Before the hub imports this module and deletes its in-tree copy
(`projects/hub-steward/code/wakala-api/modules/voting`), the two were compared
file by file and behaviour by behaviour, on 2026-11-19, with
[`scripts/hub-copy-diff.sh`](hub-copy-diff.md) (hub import paths rewritten,
comments dropped, layout normalised). The hub tree was read, never changed. This
page lists every difference and what was done with it. After the merge below,
this module is the union of both sides.

## Command output before the merge

| Output line | Meaning |
|---|---|
| `Only in hub: open_asker_test.go` | A hub test this module lacks |
| `diff models/models.go`, `diff poll_list.go`, `diff voting.operations.json` | Hub code and wording this module lacks |
| `diff poll.go`, `diff voting.blueprint.json` | This module's own window rule, which the hub lacks |
| `Only in backend: ...` | Tests and `spec/` that only this module carries (listed below) |

## Only in the hub

| What | Where in the hub | Disposition |
|---|---|---|
| The asker of an open question reads it whatever their groups: `ListOpenPolls` also returns an open poll whose maker is the caller, even when the caller is in none of the poll's groups, and no longer returns early for a caller in no group (hub commit db19613a) | `poll_list.go` | Merged into `poll_list.go` by hand; the early return for no groups is removed because it hid the asker's own polls |
| Two flags on every open-list item: `mine` (the caller asked the poll) and `in_audience` (one of the poll's groups is the caller's own), so a host offers Send only to the audience | `models/models.go` (`OpenPoll`) | Merged |
| The `list_open_polls` description names `mine` and `in_audience` and says an asker sees their own open polls | `voting.operations.json` | Merged; the text is the hub's |
| `open_asker_test.go`: the asker outside the audience, the asker inside it, a closed question not listed for its asker | `open_asker_test.go` | Moved unchanged; green |

The hub's changes before this one (hub commits ff2381c7 to 5063fbe7) were carried
over earlier and are listed in [hub copy diff](hub-copy-diff.md); the older
`ca448f7a` copy of `ListOpenPolls` is the one this module made first
(`f957eed`).

## Only in this module

| What | Disposition |
|---|---|
| The poll window rule in `poll.go` (`windowHold`, `windowEnd`): `opens_at` and `closes_at` parse as RFC 3339, compare by instant, are stored as UTC, must fall in the years 0000 to 9999, and a window that does not close after it opens is refused with 400 naming the field; the two blueprint descriptions state the rule | Kept. The hub's `poll.go` compares the two strings as text, so the hub gains the rule at import; see [poll window](poll-window.md) |
| `scripts/`, `cmd/ddl`, `spec/` (the example agency specs), `documentation/`, `CLAUDE.md`, `Makefile`, `LICENSE` | Kept; they are the repository around the module, not module code |
| Tests the hub never held: `answer_test.go`, `control_bytes_test.go`, `count_test.go`, `domain_agnostic_test.go`, `guards_test.go`, `poll_list_test.go`, `poll_window_test.go`, `reader_view_test.go`, `responses_test.go`, `result_cover_test.go`, `settings_test.go`, `testdb_test.go`, `testrun_test.go`, `validate_cover_test.go`, and in `routes/`: `answer_test.go`, `open_polls_test.go`, `poll_window_test.go`, `result_test.go`, `routes_test.go`, `table_cover_test.go` | Kept |

## Differing beyond the import path

Every file the hub and this module share is identical after the import rewrite
except the four above: `poll.go` and `voting.blueprint.json` (this module's
window rule) and `poll_list.go`, `models/models.go` and
`voting.operations.json` (the hub's asker change). No two behaviours were found
that conflict for the same operation without one side simply being newer; the one
overlap is under "Changes to existing tests" below, and the open questions are
under "Owner questions".

## Changes to existing tests

- `routes/open_polls_test.go`, `TestTheOpenPollsRouteAnswersAnEmptyArrayForACallerInNoGroup`:
  it made the poll and read the list as the same caller, who is in no group, and
  expected `[]`. With the asker change that caller now sees their own poll. The
  test now makes the poll as one caller and reads as another who is in no group
  and asked nothing, which pins the same rule (an empty array, never `null`).
  A new test, `TestTheOpenPollsRouteListsAPollTheCallerAskedWhateverTheirGroups`,
  pins the asker's own list with `mine` true and `in_audience` false. A helper,
  `newAnsweringServerOver` in `routes/answer_test.go`, lets two callers share one
  manager.
- No pinned count changed: the operations table still holds the same addresses
  and actions, and no guard test changed.

## Hub wiring that is not part of the module (not moved)

These stay in the hub and change only their import path when the hub imports this
module.

| Hub file | What it does |
|---|---|
| `internal/api/http/resident_voting.go` | The actions a resident may call (`residentVotingActions`, `residentRoleVotingActions`) and `residentPolls`, which gates `ShowPoll` and `ListChoices` through `OpenToReader` |
| `internal/api/http/voting_routes.go`, `operation_routes.go` | Mounts the routes under the instance prefix with the polls mount |
| `internal/api/http/membership.go`, `internal/accounts/router.go` | Reach a `PollManager` for an instance |
| `internal/agencies/registry.go`, `caches.go`, `test_copy.go` | Build the manager (`SpecForMount`, `Provision`, `NewPollManager`, `UseGroups`, `UseGroupExists`), cache it and give a test run its copy |
| `modules/agency/models/permissions_catalog.json` | The `poll.run` permission and the actions it grants |
| `internal/api/http/misuse_removedmember_sweep_test.go`, `misuse_residentsweep_test.go` | Hub tests of the wiring above |
| `code/app/src/features/polls/` | The member app screens that read `mine` and `in_audience` |

## Owner questions

1. Two tests in `result_cover_test.go` fail on this module's last commit, before
   and after this merge: `TestPublishingAgainReplacesThePublisherAndGroups`
   expects a second publish to replace the publisher and groups, while
   `PublishResult` (hub commit 186c5e0f, which both sides share) refuses a second
   publish with `ErrAlreadyPublished`; and
   `TestAPublishedCountIsReadOnlyByAMemberOfAPublishedGroup` expects a count of
   one answer to be readable, while the minimum count of five withholds it. The
   code is the same in the hub. Which is right: the tests, or the code? Neither
   was changed.
2. The hub gains the poll window rule at import: a poll whose `opens_at` or
   `closes_at` is not an RFC 3339 timestamp, or does not close after it opens, is
   refused with 400, and stored times are normalised to UTC. Is that acceptable
   for polls the hub already holds, whose stored values were never checked?
