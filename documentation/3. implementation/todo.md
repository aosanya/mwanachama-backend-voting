# Task board — pending

| ID | Task | Status | Depends on |
|---|---|---|---|
| VT4 | 🧭 **Counting** (member app decisions 343, 345): votes per choice and how many answered, computed from responses; for quadratic, the votes each choice received summed. 20:55 run 331845: `CountPoll` — votes per choice summed from ballots (quadratic: votes received), how many answered from participations, never joined, never stored (24b0f8e merged as c08d112; polls.md As built VT4; go test green). Left: the address and its gate — waits on the owner: one address the host also grants to members (module checks publication and groups) or two addresses with two actions (decisions 344, 345). | 📋 | VT3 |
| VT5 | 🧭 **Holders read counts and who chose what** (member app decision 344): before and after publishing, behind the host's poll-power gate; members never reach it. | ⏸️ | VT4 |
| VT6 | 🧭 **Publish a result to picked groups** (member app decisions 344, 345): a holder publishes to the groups they pick; a member in those groups reads the counts per choice and how many answered, never who chose what; any member reads their own choice. | ⏸️ | VT4 |
