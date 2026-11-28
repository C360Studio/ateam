# Tasks: coordinator-worktree-445

## 1. Approval

- [ ] 1.1 Owner approves the proposal. Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 GitHub handoff post finds the open PR first (premise: the GitHub branch of `PostPRHandoffComment` runs `gh pr comment <branch>` with no open-PR check, `internal/state/forge.go:1039-1078`, while Gitea returns empty when no open PR matches, `:1103`; callers `cmd/ateam/main.go:405` and `internal/mcpserver/server.go:347`, whose `skipped (no open PR for branch)` line at `:353` already handles the empty result). Accept: new cases in `TestPostPRHandoffCommentSkipsARepeatOfTheNewestHandoff` or a sibling test for no open PR, a merged-only branch and a list read that errors; existing GitHub fakes in `handoff_post_test.go`, `commitstatus_test.go:764`, `token_test.go:93`, `state_test.go:384` answer the list call.
- [ ] 2.2 Prune keeps `<agent>/coordinator` unless `--force` (premise: `PruneVerdict` removes any owned clean branch labelled `merged (ancestor of main)` once `occupiedBy` is empty, `internal/state/prune.go:84-91`; pickup marks the same verdict `, prune candidate`, `internal/state/state.go:3354`). Accept: a prune test for keep, `--force` removal and the dry-run line, and a pickup test with no prune hint.
- [ ] 2.3 Harness spec delta applied; `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
