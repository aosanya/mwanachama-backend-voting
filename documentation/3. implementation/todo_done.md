# Task board — done

| ID | Task | Done | Notes |
|---|---|---|---|
| VT1 | 🧭 Scaffold the module in the declared-domain shape (decisions 340, 341) | 2026-10-22 | aa0303c (merged as d6470bf): blueprint with poll, choice and the response as two records (`participation`: who answered; `ballot`: what was chosen, linked only by `response_id`); `voting.wakala.json` plus club and school examples; four gated addresses over `dispatch.Table` (make a poll, always a draft, `created_by` from the caller; read a poll; add a choice to a draft; list choices); `Provision`, `cmd/ddl`, Makefile; the guard tests incl. `domain_agnostic_test.go` and `TestWhoAnsweredIsKeptApartFromWhatWasChosen`. Who answered is `answered_by` (CLAUDE.md bans "member"); "role" stays off the forbidden list because the engine's own API says it (polls.md "As built"). Verified: `go build`, `go vet`, `gofmt -l`, `go test ./...` ok. |
