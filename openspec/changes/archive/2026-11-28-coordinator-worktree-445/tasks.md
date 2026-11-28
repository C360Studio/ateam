# Tasks: coordinator-worktree-445

## 1. Approval

- [x] 1.1 Owner approves the proposal (`Approved-by: Coby Leuschke (2026-11-28) sha256:3f609ca05310`), written by claude on the owner's instruction relayed by the coordinator.

## 2. Implementation

- [x] 2.1 GitHub handoff post finds the open PR first (premise: the GitHub branch of `PostPRHandoffComment` runs `gh pr comment <branch>` with no open-PR check, `internal/state/forge.go:1039-1078`, while Gitea returns empty when no open PR matches, `:1103`; callers `cmd/ateam/main.go:405` and `internal/mcpserver/server.go:347`, whose `skipped (no open PR for branch)` line at `:353` already handles the empty result). Accept: `TestPostPRHandoffCommentOnGitHubPostsOnlyToAnOpenPR` (no open PR, a merged-only branch, a list read that errors), `task mutate:check` detection with the base `forge.go` and with the empty-list return removed; existing GitHub fakes in `internal/state` (`handoff_post_test.go`, `commitstatus_test.go`, `token_test.go`), `cmd/ateam/commitstatus_test.go` and `internal/mcpserver` (`commitstatus_test.go`, `handoff_args_test.go`) answer the list call; the GitHub post moved to `ghPostPRHandoffComment` to stay under revive's function length.
- [x] 2.2 Prune keeps `<agent>/coordinator` unless `--force` (premise: `PruneVerdict` removes any owned clean branch labelled `merged (ancestor of main)` once `occupiedBy` is empty, `internal/state/prune.go:84-91`; pickup marks the same verdict `, prune candidate`, `internal/state/state.go:3354`). Accept: `TestPruneKeepsTheCoordinatorWorktree` (kept without `--force` as ancestor of main and via a merged PR, removed with it, `claude/coordinator-docs` still pruned, pickup line without the hint); `task mutate:check` detection with the base `prune.go`, a prefix match, and the rule disabled (pickup assertion).
- [x] 2.3 Harness spec delta applied at archive (tests cite the existing Requirements in `openspec/specs/harness/spec.md`); `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Review round 1

- [x] 3.1 Round-1 P2 (codex record at `a79efc2`): the open-PR lookup in `ghPostPRHandoffComment` ran on the caller's context, which the CLI leaves without a deadline (`cmd/ateam/main.go:404`). It now runs under `HandoffCommentReadTimeout` as a child of that context, and a stalled list returns a plain read error with nothing sent. Accept: `TestPostPRHandoffCommentBoundsAStalledOpenPRLookup`, shown able to fail with `task mutate:check` against the unbounded lookup.

## 4. Close

- [x] 4.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
