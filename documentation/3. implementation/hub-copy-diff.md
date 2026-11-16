# Hub copy diff

`mwanachama-backend-voting` is the custodian of the voting module. The hub (`projects/hub-steward/code/wakala-api/modules/voting`) holds a copy made on 2026-11-04 and changed it afterwards. Every change made in that copy exists here, so the two differ by the import path only (hub issue VT5, the voting part of ISS-105).

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-voting]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/voting`, relative to this repository's parent folder. The exit code is 1 when the output has any line starting with `diff `, `Binary files ` or `Only in hub`, and 0 otherwise, so `Only in backend` lines alone (this repository's own tests and `spec`) leave it at 0. It is 1 today for the reason given under "Changes made here first": the hub's `poll.go` lacks the MUN682 change. The script and `scripts/hubdiff` are the ones of `mwanachama-backend-forms`, pointed at `modules/voting`; the script here carries the fix of hub issue S70.

## What it does

1. Copies the `*.go` and `*.json` files of both trees into two temporary folders, leaving out `cmd/`, `documentation/`, `scripts/` and `.git/`.
2. Rewrites the hub's imports: `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/` and `wakalaserver/modules/voting` becomes this module's path.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub` and prints its output.
5. Exits 1 if that output has a `diff ` line (a file differs, JSON files included), a `Binary files ` line or an `Only in hub` line (the hub has something this repository lacks); otherwise exits 0.

## Result of 2026-11-16

As of the VT5 commit, no file of the hub differed from its file here after the rewrite, and no file existed only in the hub, so the only output was `Only in backend` lines and the script exits 0 on that state. The `*.json` files (`voting.blueprint.json`, `voting.operations.json`, `voting.wakala.json`) were already copied and compared by the same `diff -r`; they are identical. Those `Only in backend` lines are:

| Only here | Reason |
|---|---|
| `answer_test.go`, `count_test.go`, `guards_test.go`, `responses_test.go`, `settings_test.go`, `testrun_test.go`, `testdb_test.go`, `domain_agnostic_test.go`, `routes/answer_test.go`, `routes/result_test.go`, `routes/routes_test.go` | Tests this repository carried before the hub copy was made or beside it. The hub never held them. Two of them were adapted to hub behaviour: `routes/result_test.go` now expects a published count of one answer to be withheld (the minimum count) and a publication to a group the poll did not go to to be refused. |
| `control_bytes_test.go` | The red test written for hub 6ea02a12 (control bytes in a poll question or choice are refused); the hub commit had none. |
| `reader_view_test.go` | The red test written for hub 440adeef and 1fa9e664 (`OpenToReader`, `ShowPoll`, the respondent view); the hub commits had none. |
| `spec/` | The example agency specs of this repository (`club.voting.json`, `school.voting.json`). The hub never carried them. |

## The hub commits carried over

One commit here per group of hub commits, each naming the hub hashes: ff2381c7, 7578461e, 92c070dd, fc913374 (minimum count); 186c5e0f (groups of the agency); 105f528e, 3b1729a7 (poll tests and their helpers); 7d1530a9, eaa9e35e, ed495945 (one ballot set under races); 6ea02a12 (control bytes); 440adeef, 1fa9e664 (reader audience and respondent view); fab457e0 (double ballots repaired at `Provision`); 43d51edd, 725c928a, 5063fbe7 (route tests).

The poll tests of 105f528e come before the answer-race tests because those tests use their helpers. The hub's later tests and code are taken as they stand at the last commit that touched each file.

`go.mod` lists `gorm.io/driver/postgres` as a direct require from the first commit, because the hub's disclosure test opens a Postgres connection when one is configured.

## Changes made here first

Until the hub imports this module (ISS-105), each change made here first leaves `poll.go` and the new test files different from the hub copy, and the same change is needed there.

- MUN682: until the hub's `poll.go` gets the same change, the script prints a `diff poll.go hub/poll.go` block (the `time` import, the `windowHold` call and function, the RFC 3339 comparison in the validation) and exits 1. Against a scratch copy of the hub whose `poll.go` is this repository's, it exits 0. `windowHold` in `poll.go` and `poll_window_test.go`, `routes/poll_window_test.go`; see [poll window](poll-window.md).
