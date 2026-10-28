## 1. Approval and measurement

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)`, removing this hold in the same edit, and ruling on design Open Questions 1 (D2) and 2 (D4) (#250).
  Ruled 2026-10-28: D2 = A, D4 resolved by `edited`, D5 merge-check split (comment on #250 / PR #252).
- [ ] 1.2 Measure GitHub's stale-skip window on a throwaway draft PR whose `Required` skips on drafts: mark it ready and immediately query `gh api repos/{o}/{r}/commits/<sha>/check-runs` and `gh pr view --json mergeStateStatus` every 5s until the ready run's `Required` exists. Acceptance: the record shows whether a skipped `Required` was the latest context at the head while the ready run was in flight. Acceptance per D2's decision rule: seconds means ship the aggregate as is; minutes means `Verify` becomes the required context directly.

## 2. Implementation

- [ ] 2.1 Add the draft guard to `Verify` and `Required` (`.github/workflows/ci.yml:27`, `:96-111`; Gitea twin), with `Required` needing `verify` only and skipping on drafts (D2 = A). Acceptance: static test plus `detection` mutations for a dropped guard, for `Required` accepting `skipped`, and for the guard skipping `push` to `main`.
- [ ] 2.2 Add the draft guard and `edited` trigger to `Gate files` (`.github/workflows/gate-files.yml:20`; Gitea twin). Acceptance: `TestGateFilesWorkflowRunsFromBase` (`internal/bootstrap/gatefiles_test.go:134`) asserts it, and a mutation that drops it is detected.
- [ ] 2.3 Move `Merge check` into `merge-check.yml` on both forges (D5: triggers `opened, synchronize, reopened, ready_for_review, edited, labeled, unlabeled`, draft guard, base checkout, Gitea token env), and make the `ci.yml` Gitea `edited` choice with its twin-test exception (`gatefiles_test.go:172`). Acceptance: `TestGiteaWorkflowsTwinGitHub` passes and still fails on any other drift.
- [ ] 2.4 Update the scenarios at `openspec/specs/harness/spec.md:445` and `:457` with the scheduling contract, and update `TestCIWorkflowPinned`/`Sensitivity` (`internal/bootstrap/mergecheck_test.go:874-960`) to match. Acceptance: `task spec:check` and `task test:spec` are green.
- [ ] 2.5 Add to `.agents/protocol.md` § 2 Verify: CI on a draft is deferred, not passed, and local gates are unchanged; drop the § 4 sentence that expects a single Merge check miss between ready and the record, and the review-pr § 5 rerun step (#192). Acceptance: `ateam doctor` parity is green.
- [ ] 2.6 Owner edits the `main` ruleset to require `Required`, `Merge check`, and `Gate files`, after 2.3 lands a first `Merge check` check run. Acceptance: `gh api repos/{o}/{r}/rulesets/<id>` lists the three contexts.

## 3. Lifecycle evidence

- [ ] 3.1 On a throwaway PR, record check states for draft opened, draft synchronize, non-draft opened, draft → ready at the same SHA, ready synchronize, ready → draft → ready, and auto-merge armed at ready. Acceptance: the PR body table cites run URLs, and Gitea is recorded as executed or unavailable with the reason.
