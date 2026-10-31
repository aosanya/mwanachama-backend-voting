# Requirements

## Problem

An agency's members need to be asked a question and counted: which date,
which project, how to spend a budget. Forms collect answers but are never
anonymous and have no count a member sees. The Vote card on the member app's
Home (decision 321) needs something to open.

## Decisions

Settled by the owner in the member app's
[requirements.md](https://github.com/aosanya/mwanachama-frontend-member/blob/master/documentation/1.%20requirements/requirements.md),
2026-10-22.

| # | Rule |
|---|---|
| 340 | Vote's first version is polls: a question with choices and a count, never secret, with no quorum or pass mark. Secret ballots, elections, quorum and AGM agendas are a later step. |
| 341 | Polls live in their own module, this public repository, built to the declared-domain standard, compiled into wakala-api and mounted per agency; switched off, it is not mounted. Forms' rule that answers are never anonymous stays untouched, and room is left for secret ballots later. |
| 342 | Running a poll is a permission ticked per role; owners and agents hold it when no role does, and a leader's permission reaches the groups inside theirs. The host checks this permission; this module never sees roles. |
| 343 | Each poll is configured on its own: how members choose (pick one, pick up to a number, or quadratic voting, where each member spends a budget of credits and n votes on one choice cost n² credits); whether a member may change their answer until it closes; whether a member in several of the chosen groups answers once or once per group; the groups it goes to; and when it opens and closes. The agency's settings give the defaults. |
| 344 | Before publishing, holders of the permission see the counts and who chose what; members never see one another's choices, only their own. |
| 345 | A poll's result shows only once a holder publishes it, to the groups they pick: the count for each choice (for quadratic voting, the votes each choice received) and how many answered. |
| 280 | Each module stores and checks its own settings. |
| 281 | The agency definition carries the values a template or designer sets and publishing hands them to each module; an owner's live override is kept in the module beside them, logged, with Reset to definition. |

## What the module must do

- Hold a poll: its question, its choices and its settings, refusing an
  inconsistent poll on save (a closing time before the opening, a "pick up
  to" larger than the number of choices, a quadratic poll with no budget).
- Take an answer only inside the window, from a member in one of the poll's
  groups, under the poll's rules for choosing, changing and answering per
  group.
- Count: votes per choice and how many answered.
- Show holders the counts and who chose what at any time; show a member
  only their own choice.
- Publish a result to the groups a holder picks; before that, no member sees
  any count.
- Keep the agency's poll defaults as settings with a definition layer and an
  override layer.

## Out of scope for now

Secret ballots, elections, quorum, a pass mark and AGM agendas (decision 340).

## Identity

The host (`mwanachama-wakala-api`) resolves who is calling, which groups they
are in and whether they hold the poll permission. This module receives opaque
member and group ids and never learns anything else about a person.
