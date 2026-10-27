## ADDED Requirements

### Requirement: Default-branch handoff freshness follows the remote tip

A handoff in a default-branch slot SHALL be stale when its checkpoint differs from the default branch's remote-tracking tip already on disk, unless the checkpoint is the local HEAD and the local branch is ahead of that tip. A deliberate handoff written with no resolvable session id SHALL warn, and a slot refusal SHALL name the holder's writer-lease state when it is known.

#### Scenario: Checkpoint behind origin does not hold the slot

- **WHEN** `claude@main` holds a handoff at checkpoint `A`, local `main` is at `A`, and `origin/main` is at a later commit `B`
- **THEN** the handoff is stale with reason `origin/main moved from A -> B` and a new handoff from another session is written

#### Scenario: Unpushed local main keeps its handoff fresh

- **WHEN** the handoff's checkpoint is local `main`'s HEAD and local `main` is ahead of `origin/main`
- **THEN** the handoff is fresh and still refuses another session's write

#### Scenario: Handoff without a session id warns

- **WHEN** a deliberate handoff is written and no session id resolves
- **THEN** the writer is told the handoff will hold its slot against every session

#### Scenario: Refusal names the holder's writer lease

- **WHEN** a handoff is refused because another session's fresh handoff holds the slot and the writer lease belongs to that session
- **THEN** the refusal says whether that lease is live, with its pid, or exited; it says nothing about a lease it cannot attribute
