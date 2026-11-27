# Tasks: stale-record-status-414

## 1. Approval

- [x] 1.1 Owner approves the proposal.

## 2. Implementation

- [x] 2.1 Later-commit rule in `chooseStatuses` (premise: the record branch compares SHAs only, `internal/state/commitstatus.go:172-175`; the gate's rule is `gateCommitAllowed`, `internal/review/gate.go:763-787`, with `gateIsDoc` at `gate.go:727` and `gateMergeMain` at `gate.go:758`). `publishTarget` (`commitstatus.go:548`) reads `<record>..<head>` with parents, messages and per-commit names (first-parent diff for merges) from the checkout and passes the verdict through when every commit passes the copied rule; GitHub only. Accept: the six table rows in the proposal's Definition of Done; mutation checks on the docs-only and merge-file rules.
- [x] 2.2 `ateam review notify --record` publishes `ateam/review-record` on the record sha (premise: `reviewNotify` parses the record at `cmd/ateam/reviewlaunch.go:432` and posts nothing; `PublishCommitStatuses` at `commitstatus.go:352`; `recordStatusOutcome` at `commitstatus.go:649`). Accept: fake-forge test for APPROVE, REQUEST CHANGES, disabled, `--decline`, a write the forge rejects (warning, exit 0, message still printed).
- [x] 2.3 Harness spec delta applied, `docs/adoption.md` table and paragraph; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.

## Evidence

- 2.1: `TestRecordStillCurrent` runs ten rows against a real git history (docs-only commits with a rename into `archive/`, a short record sha, a code commit, Markdown under `.agents/contracts/`, a clean `main` merge, a `main` merge touching a PR file, a merge of a non-`main` branch, a missing record, a record off head's history, no base ref, Gitea); `TestChooseStatusesD3Table` gains the two carried-verdict rows. `task mutate:check`: dropping the docs-only rule → detection; dropping the merge-file rule → detection.
- 2.2: `TestReviewNotifySetsRecordStatus` (fake `gh`): APPROVE → one `success` write, REQUEST CHANGES → one `failure` write, both on the full head sha from a short `Reviewed at`; `ATEAM_COMMIT_STATUSES=0` and `--decline` → no `gh api` call; a rejected write → exit 0, the tool line still printed, one warning. `task mutate:check`: dropping the publish call → detection; dropping the enabled check → detection.
- Examples suffice over generated checks: the rule is an all-commits predicate with one outcome per commit class, and each class has its own row; commit order does not change the answer.
- Round 2 (codex REQUEST CHANGES at 16620b6): `commitFiles` reads `git diff -z` and splits on NUL, so a non-ASCII or newline document name, which git C-quotes otherwise, stays a document. `TestRecordStillCurrent`'s docs-only commit now adds `docs/guide-é.md` and `docs/line\nbreak.md`; `task mutate:check` with the pre-fix `commitstatus.go` as the mutant → detection (`recordStillCurrent = false, want true`). Notify's test now fakes the status writer (`recordCommitStatus`), so it runs no fake `gh` and no clock: dropping the publish call or the enabled check → detection. `TestRecordCommitStatus` covers the writer itself (full sha from a short one, one context, missing commit) with the per-call timeout lifted; writing the short sha → detection.
