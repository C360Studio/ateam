# Change Proposal: `handoff-freshness-210`

Approved-by:

## Why

A session's `ateam state handoff` from `main` can be refused by another session's handoff that is already out of date (#210, observed 2026-10-27 after PR #187). Two causes:

1. `handoffStaleReasonsAt` judges a checkpoint against `headSHAForBranch`, which reads the local checkout. Agents rarely pull the shared `main` checkout, so a `main`-slot handoff stays "fresh" after `origin/main` moves, and `HoldsSlotAgainst` refuses the next write. A session that has just landed a PR ends on an unpulled `main` almost by construction.
2. A handoff written with no session id holds the slot against everyone, and the refusal can only say "an unknown session", so the refused session cannot tell whether the holder is still live. `effectiveSessionID` returns `""` silently.

## What Changes

- `SnapshotGit` records the remote-tracking tip of the default branch (`refs/remotes/origin/main`, else `origin/master`) already on disk, and how many commits local `main` is ahead of it. No fetch, no network call.
- A default-branch slot handoff is stale with reason `origin/main moved from <checkpoint> -> <tip>` when its checkpoint differs from that tip, unless the checkpoint is the local HEAD and local `main` is ahead of the remote (unpushed local commits keep their own handoff fresh). No remote ref on disk: today's local-HEAD rule only.
- A deliberate handoff (CLI, MCP, web) written with no resolvable session id warns on stderr (CLI) or in the returned result (MCP, web): `handoff written without a session id; it will hold slot <key> against every session`.
- The `HandoffSlotHeldError` message gains the writer lease's state for the holder when it is known: `holder's writer lease: live (pid N)` or `exited`, matched by session id; omitted when unknown.

## Non-goals

- Whether an id-less block should keep holding the slot (#210 keeps today's behaviour, only makes it visible).
- `--force` semantics, slot keys (`agent@branch`, #135), and `PR 187`-style goal mentions.
- Fetching on the hook path or reading the forge's default-branch name; `main`/`master` stays the convention used elsewhere in `internal/state`.

## Definition of Done

- Tests in `internal/state`: a `main`-slot handoff behind `origin/main` is stale and does not refuse a new write; one at the remote tip, or at a local HEAD ahead of the remote, still refuses; no remote ref keeps today's rule; an id-less deliberate handoff warns; the refusal names lease state when known. Each shown able to fail by `task mutate:check`.
- `ateam doctor` hook latency still within budget; `task verify` green on a clean tree.
