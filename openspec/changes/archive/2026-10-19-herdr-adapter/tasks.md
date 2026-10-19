# Tasks: Optional herdr adapter

implemented-by: claude (opus)

## 1. Design

- [x] 1.1 Reverify herdr env vars, event names, and manifest fields against herdr 0.9.5 (`herdr api schema --json`, CLI usage, binary strings). Runtime-only facts (plugin command cwd, action stdout, `herdr pane get` shape, manifest top-level keys) move to 3.2.
- [x] 1.2 Post the manifest shape and the notification hook-path decision as a comment on #24 (https://github.com/C360Studio/ateam/issues/24#issuecomment-6275048562).

## 2. Implementation

- [x] 2.1 HERDR-aware writer lease and pickup warning, with tests for set and unset `HERDR_*`.
- [x] 2.2 Herdr claim guidance in `.agents/protocol.md` and the `pickup` skill.
- [x] 2.3 `.agents/herdr/herdr-plugin.toml` manifest and `ateam state herdr` (`open-pane`, `on-event`); `ateam doctor --check` passes with and without it.
- [x] 2.4 Toast notifications on new spec-queue caveats, off the synchronous hook path, with a test that the gate never fires when `HERDR_ENV` is unset.
- [x] 2.5 herdr rows in `docs/hook-flow.md` and `docs/adoption.md`.

## 3. Verification

- [x] 3.1 Mutation evidence per `.agents/protocol.md`; `task verify` green.
- [x] 3.2 Manual herdr session check recorded in the PR: pickup via action, `worktree.created` fires, a caveat toast appears; confirm the runtime-only facts deferred from 1.1. Visual render in an attached TUI waived by owner on 2026-10-19; verified through a headless herdr 0.9.5 server log and `herdr plugin log` (PR #25).
