# Polls

## Objects

| Object | Holds |
|---|---|
| `poll` | The question; the groups it goes to (opaque group ids); opens at, closes at; its settings (below); its state (draft, open, closed); its publication (published at, by whom, to which groups). |
| `choice` | One option on a poll: its label and its order. |
| `response` | One answer, held as two records (see the last section): its participation (poll, member id from the caller, the group it was given for under `per_group`, answered at) and its choices (poll, group, the votes placed on each choice). Created on the first answer and replaced when the poll allows a change. |

A member and a group are ids the host passes in. The module never stores a
name, a role or a membership list; it is told on each call which groups the
caller is in.

## Poll settings

| Setting | Values |
|---|---|
| How members choose | `one`, `up_to` with a number N, `quadratic` with a credit budget per member |
| Change allowed | yes: the response may be replaced until the poll closes; no: the first answer stands |
| Per group | `once`: one response per member; `per_group`: one response per member per chosen group they are in |
| Groups | the group ids the poll goes to |
| Window | opens at, closes at |

The agency's settings give the defaults a new poll starts with (decisions
280, 281): a definition layer handed over on publishing the agency, and an
owner's override layer kept beside it, logged, with Reset to definition. The
settings are declared through shared's module-settings engine (S64).

Save refuses a poll whose window is backwards, whose `up_to` exceeds its
choices, whose quadratic budget is missing or below 1, or that goes to no
group.

## Answering

An answer is taken only between opens at and closes at, and only from a
caller in one of the poll's groups (for `per_group`, the group named must be
one of them and one the caller is in).

- `one`: exactly one choice, one vote.
- `up_to`: between 1 and N distinct choices, one vote each.
- `quadratic`: any number of votes on any choices; the cost is the sum over
  choices of votes², and it must not exceed the budget. 3 votes on one choice
  cost 9 credits; 2 and 2 on two choices cost 8.

A second answer replaces the first when the poll allows a change and is
refused when it does not.

## Counting

For each choice, the sum of votes across responses; for `one` and `up_to`
that is the number of members who picked it, for `quadratic` the votes it
received. "How many answered" is the number of responses (per member, or per
member and group under `per_group`). Counts are computed from responses,
never stored beside them, so a changed answer cannot leave a stale total.

## Who sees what

- Holders of the poll power (checked by the host) read counts and the list
  of who chose what at any time (decision 344).
- A member reads their own response and nothing else until a result is
  published.
- Publishing records the groups a holder picks; members of those groups then
  read the counts per choice and how many answered (decision 345). A
  published result never carries who chose what.

## Room for a secret ballot later

Polls are never secret (decision 340), but a secret ballot is coming, so the
store keeps **who answered** apart from **what they chose** from the start:

- a participation record: poll, member id, group, answered at — enough to
  stop a second answer and to count turnout;
- a choice record: poll, group, the votes per choice — enough to count.

For a poll the two are linked by a response id so a holder can read who
chose what and a member can read and change their own. A secret ballot later
drops that link (or replaces it with a receipt only the member holds): the
participation record still stops double voting and the choice record still
counts, and no row ties the two. Nothing in counting or publishing reads the
link, so the later step changes what is written, not how anything is
counted.

## As built (VT1, 2026-10-22)

The scaffold declares four objects in `voting.blueprint.json`: `poll`,
`choice`, and the response as its two records, `participation` (who answered)
and `ballot` (what was chosen). The participation's id is the response id;
the ballot's `response_id` is the link a secret ballot will leave empty. The
ballot carries no `answered_by` and no `answered_at`, and the participation
carries no `votes`; `TestWhoAnsweredIsKeptApartFromWhatWasChosen` holds that.
Votes are a JSON object keyed by choice id, so `one`, `up_to` and `quadratic`
share one shape.

Who answered is `answered_by`, not `member_id`: the module names no member
(CLAUDE.md), and the host supplies the id from the caller. A unique index on
`(poll_id, answered_by, group_id)` is the database's half of "one answer per
person", with `group_id` empty under `once`.

The operations file declares four addresses, every one gated: make a poll,
read it, add a choice, list choices. A new poll is always a draft whatever
the body says, `created_by` is bound from the caller (no caller is 401), and
choices change only while the poll is a draft (409 otherwise). Save refuses
a poll that goes to no group, whose window is backwards, an `up_to` with no
N and a `quadratic` with a budget below 1. The `up_to` against the number of
choices check waits for the open step, because a draft gains its choices
after it is saved.

Answering, counting, opening, closing, publishing and the settings layers
are later rows. `domain_agnostic_test.go`'s word list does not carry `role`:
it is the declared-domain engine's own word for an object's slot
(`spec.ByRole`), not a domain's. It carries `member` and `election` and the
nouns of the two example domains (`club`, `school`).

## As built (VT2, 2026-10-22)

A poll is saved with its choices in one call: `choices` on the create body is
a list of labels, carried on `Poll` but not stored on the poll row
(`spec:"-"`); each becomes a `choice` row in the order given, numbered from
one, with its id, poll and time minted by the store whatever the body says.
Choices may still be added one at a time to a draft afterwards.

Save now refuses, besides VT1's checks, a choice with no label and an
`up_to` whose N is more than the choices it was saved with. A poll saved
with no choices is not held to N yet; that check waits for the open step,
when the choice list is final. The poll row and its choice rows are written
one after another rather than in one transaction: every check runs before the
first write, so only a database failure can leave a poll without its
choices.

## As built (VT3, 2026-10-22)

`POST /polls/{pollID}/answers` (`voting.answer.create`, `AnswerPoll`) takes
`votes`, keyed by choice id, and under `per_group` the `group_id` the answer
is for. The poll comes from the address and who answered from the caller
(`answered_by`, bound `from: caller`); a body naming either is overwritten,
and no caller is 401. It answers with the participation only: who answered,
for which group, when. What was chosen is written to the ballot and never
returned beside it.

**The caller's groups are the host's fact.** `UseGroups(fn)` gives the
manager a `Groups` function, `func(ctx, caller) ([]string, error)`, that the
host fills from its own membership; the module asks it on every answer and
stores nothing it returns. A manager given none puts the caller in no group,
so it takes no answer. The same shape as comm's `UseBelonging`.

The checks run in this order, all before the first write:

1. A caller: `ErrNoAnswerer`, 401.
2. The poll exists: `ErrPollNotFound`, 404.
3. Inside the window, `opens_at` included and `closes_at` excluded, compared
   as instants rather than as text (a stored `opens_at` need not carry the
   store's nanosecond layout): `ErrOutsideWindow`, 409.
4. The group. Under `once`, the caller is in at least one of the poll's
   groups and the stored `group_id` is empty. Under `per_group`, the answer
   names a group that is both one of the poll's and one of the caller's.
   Otherwise `ErrNotInPollGroups`, 404, the same answer as a missing poll.
5. The votes, against the poll's choices and mode: every key a choice on
   this poll, every count at least 1; `one` exactly one choice with one
   vote; `up_to` between 1 and N choices, one vote each; `quadratic` the sum
   of each count squared within the budget. Otherwise `ErrInvalidAnswer`,
   400.
6. A second answer, found by `(poll_id, answered_by, group_id)`: refused
   with `ErrAlreadyAnswered`, 409, unless the poll allows a change. A change
   keeps the response id, deletes the old participation and its ballot and
   writes both again, so there is never more than one ballot per response
   and a count cannot see the old choice. The unique index on the
   participation is the database's half of the same rule.

**Not checked yet:** the poll's `state`. Nothing opens a poll so far, so an
answer is taken from a draft inside its window; the state check, and holding
`up_to`'s N to the final choice list, land with the open step.

## As built (VT4, 2026-10-22)

`CountPoll(ctx, pollID)` answers a `Count`: the poll, each of its choices in
order with its label, position and `votes`, and `answered`. It is computed on
every call and stored nowhere.

- **Votes per choice** are the sum, over the poll's ballots, of the votes
  each ballot placed on that choice. Under `one` and `up_to` that is the
  number of answers that picked it; under `quadratic` it is the votes the
  choice received (3 on one choice and 2 and 2 on two give 5 and 2), never
  the credits spent. A choice nobody picked reads 0.
- **How many answered** is the number of participation rows on the poll: one
  per member under `once`, one per member and group under `per_group`.
- The two are read separately, the ballots for totals and the participations
  for turnout. Nothing in counting reads a ballot's `response_id` or joins a
  ballot to who answered, so a secret ballot later changes nothing here.
- A changed answer replaces its ballot (VT3), so a count never sees the old
  choice. A missing poll is `ErrPollNotFound`.

**No address is declared yet.** Decisions 344 and 345 give the count two
audiences with two different gates: a holder of the poll power at any time,
which is a host power check over an action, and a member of a published
group once a result is published, which is not a power but a fact about the
poll and the caller's groups. Whether that is one address whose action the
host grants to members too, with the module checking publication and groups,
or two addresses with two actions, is an owner question; the manager method
serves either.

## As built (VT7)

The poll defaults of decision 343 are declared in `voting.blueprint.json`'s
`settings` block through shared's S64: `mode` (default `one`), `max_choices`
and `budget` (default `0`, meaning none, so a poll that needs one must name
its own and is refused otherwise), `change_allowed` (default false) and
`per_group` (default `once`). `spec.CheckSettings` holds any layer's values to
those declarations.

The two layers are the host's. `UseSettings` gives the manager a
`SettingsLayers` function returning the definition layer and the owner's
overrides (each with who and when); `Settings` resolves them with
`spec.ResolveAll`, answering every effective value with the layer it came
from. Storing the overrides, logging each change and Reset to definition
(dropping the override) are the host's, the same split
`mwanachama-backend-files` takes. A manager given no layers resolves to the
module's defaults.

`CreatePoll` starts from the effective values: an empty `mode` or `per_group`
and a zero `max_choices` or `budget` are filled from them before the poll is
checked. `change_allowed` is a bool the request cannot leave unset, so a new
poll keeps what it was sent; filling it from the setting needs the wire to
tell "not sent" from false.
