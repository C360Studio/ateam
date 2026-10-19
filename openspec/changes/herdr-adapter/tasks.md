# Tasks: Optional herdr adapter

implemented-by: claude (opus)

## 1. Design

- [ ] 1.1 Reverify herdr env vars, event names, and manifest fields against an installed herdr binary (`herdr --help`, `herdr api schema --json`). Hold: herdr is not installed on the dev machine.
- [ ] 1.2 Post the manifest shape and the notification hook-path decision as a comment on #24.

## 2. Implementation

- [ ] 2.1 HERDR-aware writer lease and pickup warning, with tests for set and unset `HERDR_*`.
- [ ] 2.2 Herdr claim guidance in `.agents/protocol.md` and the `pickup` skill.
- [ ] 2.3 `herdr-plugin.toml` manifest; `ateam doctor --check` passes with and without it.
- [ ] 2.4 Toast notifications on new spec-queue caveats, off the synchronous hook path, with a test that the gate never fires when `HERDR_ENV` is unset.
- [ ] 2.5 herdr rows in `docs/hook-flow.md` and `docs/adoption.md`.

## 3. Verification

- [ ] 3.1 Mutation evidence per `.agents/protocol.md`; `task verify` green.
- [ ] 3.2 Manual herdr session check recorded in the PR: pickup via action, `worktree.created` fires, a caveat toast appears.
