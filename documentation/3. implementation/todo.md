# Task board — pending

| ID | Task | Status | Depends on |
|---|---|---|---|
| VT3 | 🧭 **Answering under the poll's settings** (member app decision 343): only inside the window and from a caller in one of the poll's groups, the member from the caller; one choice, up to N distinct choices, or quadratic votes whose squared cost fits the budget; a second answer replaces the first or is refused as the poll says; once per member or once per member per group. | 📋 | VT2 |
| VT4 | 🧭 **Counting** (member app decisions 343, 345): votes per choice and how many answered, computed from responses; for quadratic, the votes each choice received summed. | ⏸️ | VT3 |
| VT5 | 🧭 **Holders read counts and who chose what** (member app decision 344): before and after publishing, behind the host's poll-power gate; members never reach it. | ⏸️ | VT4 |
| VT6 | 🧭 **Publish a result to picked groups** (member app decisions 344, 345): a holder publishes to the groups they pick; a member in those groups reads the counts per choice and how many answered, never who chose what; any member reads their own choice. | ⏸️ | VT4 |
| VT7 | 🧭 **Voting settings** (member app decision 343; decisions 280, 281): the poll defaults (how members choose, budget, change allowed, once or per group) declared through shared S64, with a definition layer and an owner override layer, logged, with Reset to definition; a new poll starts from the effective values. | 📋 | VT2 |
