## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1 and 2 (#229). Evidence: the owner approved in Enjoy on 2026-10-28, ruling OQ1 A (remove host-managed worktrees too; git is the source of truth) and OQ2 deferred (`--all` out of scope); Claude wrote the header at their request.

## 2. Classifier

- [ ] 2.1 Add a pure `PruneVerdict(st, wt, agent) (remove bool, reason string)` in `internal/state` beside `worktreePRStatusAndAge` (`internal/state/state.go:1985`). It reads the label, `Dirty` (`state.go:495`), the owner from `inferAgentFromBranch` (`state.go:2490`), or for a detached checkout from the `<agent>-review-pr-<n>` path, and the closed card for PR n. Acceptance: a table test covers each label, a dirty worktree, an unknown dirty state, another agent's prefix, an unprefixed branch, the current checkout, a review checkout whose PR is merged, open, or not in the cache, and a Gitea card. `task mutate:check` reports `detection` with the dirty check dropped and with the ownership check dropped (#229).

## 3. Command

- [ ] 3.1 Add `ateam state prune [--dry-run] [--agent]` to the dispatch (`cmd/ateam/main.go:134-160`). It refreshes like `runStatePickup` (`main.go:241`) and resolves the agent like `main.go:247`. For each `remove` verdict it runs `git worktree remove <path>` (without `--force`) and `git branch -D <branch>`. It runs `git push origin --delete <branch>` only when `git ls-remote origin refs/heads/<branch>` equals the local head. It then refreshes again so the cache drops the removed worktrees. Acceptance: the temporary-repository table test in the proposal's Definition of Done passes through the CLI with a bare `origin`. A `--dry-run` run leaves the `git worktree list` and `git ls-remote` output unchanged. `task mutate:check` reports `detection` with the remote-head comparison dropped (#229).

## 4. Pickup hint

- [ ] 4.1 `writePickupWorktrees` (`state.go:2870`) appends `, prune candidate` inside the parentheses when `PruneVerdict` returns `remove` for `st.CurrentAgent`. Acceptance: a pickup render test matches the full line. `TestHookLatencyBudget` passes (#229).

## 5. Docs and verify

- [ ] 5.1 `.agents/protocol.md:44` Close text, `AGENTS.md` § Close, `stateUsage` (`cmd/ateam/main.go:166`), and `README.md` name `ateam state prune`. The spec delta joins `openspec/specs/harness/spec.md` on archive. Acceptance: `ateam doctor` parity and the `AGENTS.md` size check pass (#229).
- [ ] 5.2 `task verify` passes on a clean tree and `ateam doctor` parity passes (#229).
