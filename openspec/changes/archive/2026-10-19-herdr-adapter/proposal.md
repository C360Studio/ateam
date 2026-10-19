# Change: Optional herdr adapter

Issue: https://github.com/C360Studio/ateam/issues/24
implemented-by: claude (opus)

## Why

When ateam runs inside a herdr pane, three ateam pieces overlap with herdr primitives: the writer lease records a short-lived hook PID that proves nothing about a live agent, worktree claims could also open a grouped herdr workspace, and Hold/blocked transitions have no out-of-band signal. herdr already exposes pane identity, worktree creation, plugin actions/events, and notifications.

## What Changes

An optional adapter, every behavior gated on `HERDR_ENV=1` with today's behavior as the fallback:

- Writer lease records `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID`; the pickup warning names the pane.
- Protocol and `pickup` skill document `herdr worktree create` for claims; git worktree parsing stays the source of truth.
- A `herdr-plugin.toml` manifest exposing pickup/handoff/spec-queue actions, a `worktree.created` event, and a queue popup pane.
- Notifications when holds rise or the queue reports `HALT`/`RED`/`BLOCKED`, kept off the synchronous hook path.

## Impact

Scope: `internal/state`, `internal/hook` (env reads only), `.agents/protocol.md`, the `pickup` skill, `docs/hook-flow.md`, `docs/adoption.md`, the new manifest. Non-goals: hard herdr dependency, rewrite, the web-board ADR, the #16 multi-agent driver, and herdr agent state as a benchmark oracle. Stop boundary: design comment on #24 before implementation.
