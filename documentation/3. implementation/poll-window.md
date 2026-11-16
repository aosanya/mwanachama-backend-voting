# Poll window

A poll's `opens_at` and `closes_at` are read as RFC 3339 instants, never as text (hub issue MUN682).

## What a save does

`CreatePoll` runs `windowHold` before the other settings checks:

1. `opens_at` is parsed with `time.Parse(time.RFC3339Nano, ...)`. A value that does not parse is refused with `ErrInvalidPoll`, 400, `invalid poll: opens_at is not an RFC 3339 timestamp`.
2. `closes_at` the same way, `invalid poll: closes_at is not an RFC 3339 timestamp`.
3. The two instants are compared. A window that closes at or before it opens is refused, 400, `invalid poll: closes_at falls after opens_at`. `2026-11-08T00:00:00Z` to `2026-11-08T12:00:00+14:00` is refused (the close is 22:00 UTC the day before); `2026-11-08T05:00:00+14:00` to `2026-11-08T00:00:00Z` is a real ten-hour window and is accepted.
4. Each end, once converted to UTC, must have a year from 0000 to 9999, because the readers (`insideWindow`, `hasOpened`) parse with `time.RFC3339Nano` and a year outside that range formats as text that does not parse back. `9999-12-31T23:00:00-02:00` is 10000-01-01T01:00:00Z in UTC and `0000-01-01T00:00:00+02:00` is year -1; both are refused, 400, `invalid poll: closes_at falls outside the years 0000 to 9999 in UTC` (or `opens_at`). `0001-01-01T00:00:00+02:00` is `0000-12-31T22:00:00Z`, which parses, and is accepted, as is `9999-12-31T23:59:59Z`.
5. Both are stored as the same instants in UTC, `time.RFC3339Nano` layout (`2026-11-08T10:00:00+02:00` is stored as `2026-11-08T08:00:00Z`), and the created poll answers with that text.

What time.Parse refuses is refused here: `2026-02-30`, hour `24`, second `60`, a date alone, a time with no zone, lower case `t` or `z`, and a value with a space before or after it. An empty value is refused earlier by the blueprint's required check.

Shared holds no window helper and forms' `ValidateWindow` belongs to another module, so the voting package parses with the standard library as `answer.go` and `audience.go` already do.

## Rows already stored

The save is the only change. No stored poll is rewritten, deleted or flagged.

- A row whose window parses, in any offset, keeps its text and reads by instant as before.
- A row whose window does not parse (written before this change) keeps its text. Its readers already treat an unreadable window as closed: a resident gets `ErrPollNotFound` from `OpenToReader` and an answer gets `ErrOutsideWindow`. The owner and the poll runner still read it.

`TestStoredPollsKeepTheWindowTheyWereWrittenWithAndReadAsBefore` pins both, and passes with and without the fix.

Whether to repair or report those rows is the owner's decision and is not made here.

## Hub copy

The hub copy `projects/hub-steward/code/wakala-api/modules/voting` needs the same change in `poll.go` until ISS-105 replaces it with an import of this module. The change is not made there from this repository.

## Tests

`poll_window_test.go` (manager) and `routes/poll_window_test.go` (the real mux, 400 and 201).
