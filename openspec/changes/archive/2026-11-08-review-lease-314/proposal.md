# Change Proposal: `review-lease-314`

Approved-by: Coby Leuschke (2026-11-07)

## Why

A read-only reviewer takes the writer lease of the checkout it starts in (#314, #315). In the #303 live run, a T3 Code review thread started in `main` and left `Writer lease: agent=codex` with the different-writer warning, though nobody was writing there. The hooks write the lease at session start, before each prompt, and at turn end (`internal/hook/hook.go:145`, `:159`, `:277`) for every session.

## What Changes

- **#315:** with `ATEAM_ROLE=review` in its environment, a session's hooks write no writer lease on session-start, before-prompt, or turn-end. Nothing else about the lease changes.
- **#314:** `.agents/hosts/t3code.toml` `[launch]` has the requester create the detached `.worktrees/{agent}-review-pr-{pr}` checkout at `{sha}` first, then pass `workspaceStrategy {type: existing_worktree}` for it to `t3_thread_launch`. The reviewer thread, its hooks, and the skill then share one checkout, and the reviewer's lease stays there.
- `.agents/hosts/herdr.toml` `[launch]` adds `--env ATEAM_ROLE=review` to `herdr pane split`, so a herdr reviewer does not take the lease of the implementer's worktree.
- `review-pr` skill: a reviewer that is already in the detached review checkout uses it instead of creating one.
- `docs/adoption.md` says where this stops: Enjoy's `start_conversation` and T3's shared `codex app-server` cannot carry a per-session variable, so Enjoy reviewers still take the lease of the project checkout, and T3's own `codex exec` naming runs still take the thread worktree's lease. A terminal reviewer is started as `ATEAM_ROLE=review codex exec …`.
- `openspec/specs/harness/spec.md` gains one Requirement, `Read-only review session writes no writer lease`.

## Track

`design`: adds a spec Requirement, touches the session-start boot path, and edits gate files (`.agents/hosts/`). See `design.md`.

## Non-goals

- No lease taken on first mutation (#315 option A, not chosen).
- No Enjoy or T3 naming-run fix; these are documented as limits (owner scoping on #315, 2026-11-07).
- No change to handoff ownership, `ateam state prune`, or #321's occupancy check.

## Definition of Done

- A hook test shows that with `ATEAM_ROLE=review`, session-start, before-prompt, and turn-end leave an existing lease byte-for-byte unchanged, and that without it they still write the lease. `task mutate:check` reports `detection` with each of the three guards removed.
- The T3 launch is checked in one live T3 Codex review: the thread's cwd is the detached checkout, and `main`'s lease is unchanged.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green. The owner applies `gate-change-approved`. The change is archived before ready.
