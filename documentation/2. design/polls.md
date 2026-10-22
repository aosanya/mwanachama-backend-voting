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
