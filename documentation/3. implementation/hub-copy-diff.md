# Hub copy diff

`mwanachama-backend-voting` is the custodian of the voting module. The hub (`projects/hub-steward/code/wakala-api/modules/voting`) holds a copy made on 2026-11-04 and changed it afterwards. Every change made in that copy exists here, so the two differ by the import path only (hub issue VT5, the voting part of ISS-105).

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-voting]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/voting`, relative to this repository's parent folder. The exit code is 1 when the trees differ in any way, files only here included, so it is 1 today; read the output, not the code. The script and `scripts/hubdiff` are the ones of `mwanachama-backend-forms`, pointed at `modules/voting`.

## What it does

1. Copies the `*.go` and `*.json` files of both trees into two temporary folders, leaving out `cmd/`, `documentation/`, `scripts/` and `.git/`.
2. Rewrites the hub's imports: `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/` and `wakalaserver/modules/voting` becomes this module's path.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub`.

## Result of 2026-11-16

No file of the hub differs from its file here after the rewrite, and no file exists only in the hub. What remains is `Only in backend` lines:

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

- MUN682: `windowHold` in `poll.go` and `poll_window_test.go`, `routes/poll_window_test.go`; see [poll window](poll-window.md).
