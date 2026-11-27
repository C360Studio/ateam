# Tasks: stale-record-status-414

## 1. Approval

- [ ] 1.1 Owner approves the proposal.
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Later-commit rule in `chooseStatuses` (premise: the record branch compares SHAs only, `internal/state/commitstatus.go:172-175`; the gate's rule is `gateCommitAllowed`, `internal/review/gate.go:763-787`, with `gateIsDoc` at `gate.go:727` and `gateMergeMain` at `gate.go:758`). `publishTarget` (`commitstatus.go:548`) reads `<record>..<head>` with parents, messages and per-commit names (first-parent diff for merges) from the checkout and passes the verdict through when every commit passes the copied rule; GitHub only. Accept: the six table rows in the proposal's Definition of Done; mutation checks on the docs-only and merge-file rules.
- [ ] 2.2 `ateam review notify --record` publishes `ateam/review-record` on the record sha (premise: `reviewNotify` parses the record at `cmd/ateam/reviewlaunch.go:432` and posts nothing; `PublishCommitStatuses` at `commitstatus.go:352`; `recordStatusOutcome` at `commitstatus.go:649`). Accept: fake-forge test for APPROVE, REQUEST CHANGES, disabled, `--decline`, a write the forge rejects (warning, exit 0, message still printed).
- [ ] 2.3 Harness spec delta applied, `docs/adoption.md` table and paragraph; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
