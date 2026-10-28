## 1. Approval and measurement

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)`, removing this hold in the same edit, and ruling on design Open Questions 1 (D2) and 2 (D4) (#250).
  Ruled 2026-10-28, amended the same day: D2 = rename on draft so no required-name check run exists at a draft head; D4 = same bytes plus `edited` on the Gitea trigger lines; D5 merge-check split parked as its own issue (amended ruling comment on #250 / PR #252).
- [x] 1.2 Measure GitHub's stale-skip window on a throwaway draft PR whose `Required` skips on drafts: mark it ready and immediately query `gh api repos/{o}/{r}/commits/<sha>/check-runs` and `gh pr view --json mergeStateStatus` every 5s until the ready run's `Required` exists. Acceptance: the record shows whether a skipped `Required` was the latest context at the head while the ready run was in flight.
  Result (2026-10-28, throwaway PR #254, closed unmerged): the window is **minutes** for `Required`. The probe `ci.yml` had a guarded `Verify` (`sleep 120`) and `Required` with `needs: [verify]`. The draft run ([40770567153](https://github.com/C360Studio/ateam/actions/runs/40770567153)) left `Verify` and `Required` `skipped`. After `gh pr ready` at 21:16:38Z, the ready run ([40770596171](https://github.com/C360Studio/ateam/actions/runs/40770596171)) created `Verify` at 21:16:42Z (+4s) and `Gate files` at +5s. Its `Required` check run did not exist until 21:18:47Z (+129s), after `Verify` finished. For those 129s the skipped `Required` was the only `Required` context at the head, and `statusCheckRollup` reported it as `COMPLETED/SKIPPED`. `mergeStateStatus` stayed `BLOCKED` only because the probe touched a gate file, so `Gate files` failed. This measurement motivated D2's rename: a plain skip leaves a passing `Required` at the head for the length of `Verify`.
- [x] 1.3 Confirm the rename on a throwaway draft PR whose jobs carry the D2 `name:` expression: record that the draft head has no check run named `Required` or `Gate files`, that `gh pr view --json statusCheckRollup` and the checks UI show them as expected/waiting, and what they show in the first 10s after `gh pr ready` at the same SHA. Acceptance: the record cites both run URLs; if the expression is not evaluated in `name:`, stop and hand off.
  Result (2026-10-28, throwaway PR #256, closed unmerged; probe `ci.yml` with the D2 `name:`/`if:` pair on `Verify` (`sleep 60`) and on `Required` with `needs: [verify]`): the safety property holds, but the draft label is not what D2 expected. **A skipped job's `name:` is not evaluated.** The draft run ([40776449237](https://github.com/C360Studio/ateam/actions/runs/40776449237)) created skipped check runs named with the raw expression text, `github.event.pull_request.draft && 'Required (draft, skipped)' || 'Required'`, not `Required (draft, skipped)`. Either way, the draft head had no check run named `Required`. `statusCheckRollup` listed none, and `mergeStateStatus` was `BLOCKED`. After `gh pr ready` at 21:59:59Z, the ready run ([40776477758](https://github.com/C360Studio/ateam/actions/runs/40776477758)) evaluated the names. `Verify` existed at +7s, and `Required` existed at +74s once `Verify` had finished, at which point it ran and succeeded. From +1s to +68s, the rollup had no `Required` entry at all, so nothing stale could satisfy the ruleset. `Gate files` runs from the base branch (`pull_request_target`), so the probe could not exercise its rename. It failed throughout because the probe touched a gate file, which also kept the PR `BLOCKED`. The checks UI's "Expected — Waiting for status" text was not inspected; absence from the rollup is the evidence.

## 2. Implementation

- [x] 2.1 Add the draft guard and draft rename to `Verify`, `Merge check`, and `Required` (`.github/workflows/ci.yml:27`, `:65`, `:96-111`; Gitea twin), with `Required` as `if: always() && (...)`. Acceptance: static test plus `detection` mutations for a dropped guard, a dropped rename, `Required` without `always()`, `Required` accepting `skipped`, and a guard that skips `push` to `main`.
- [x] 2.2 Add the draft guard and rename to `Gate files` (`.github/workflows/gate-files.yml:20`; Gitea twin). Acceptance: `TestGateFilesWorkflowRunsFromBase` (`internal/bootstrap/gatefiles_test.go:134`) asserts both, and a mutation that drops either is detected.
- [x] 2.3 Add `edited` to the `types:` of `.gitea/workflows/ci.yml` and `.gitea/workflows/gate-files.yml` only, and give `TestGiteaWorkflowsTwinGitHub` (`gatefiles_test.go:172`) a documented exception for those two lines. Acceptance: the twin test passes and still fails on any other drift, including `edited` appearing in a GitHub file.
- [x] 2.4 Update the scenarios at `openspec/specs/harness/spec.md:445` and `:457` with the draft contract (guard, rename, `always()` form, Gitea `edited`), and update `TestCIWorkflowPinned`/`Sensitivity` (`internal/bootstrap/mergecheck_test.go:874-960`) to match. Acceptance: `task spec:check` and `task test:spec` are green.
- [x] 2.5 Add to `.agents/protocol.md` § 2 Verify: CI on a draft is deferred, not passed, and local gates are unchanged. Acceptance: `ateam doctor` parity is green.

## 3. Lifecycle evidence

- [ ] 3.1 On a throwaway PR (1.3's may be reused), record check states for draft opened, draft synchronize, non-draft opened, draft → ready at the same SHA, ready synchronize, ready → draft → ready, and auto-merge armed immediately after ready. Acceptance: the PR body table cites run URLs, and Gitea is recorded as executed or unavailable with the reason.

## Evidence

- 2.1-2.3: `TestCIWorkflowPinned`, `TestCIWorkflowPinnedSensitivity` (6 new cases), `TestGateFilesWorkflowRunsFromBase`, and `TestGiteaWorkflowsTwinGitHub` in `internal/bootstrap`. Twelve workflow mutants were each applied with `cp` and SHA-256 restore and run against those tests, and all twelve gave `detection` with the expected message:
  - `Verify` guard dropped
  - `Merge check` guard dropped
  - guard narrowed to `!draft`, which skips push to `main`
  - `Required` rename dropped
  - `Required` without `always()`
  - `Required` accepting `skipped`
  - `ready_for_review` trigger dropped
  - `Gate files` guard dropped
  - `Gate files` rename dropped
  - Gitea `ci.yml` `edited` dropped
  - GitHub `ci.yml` taking `edited`
  - Gitea `gate-files.yml` `edited` dropped
- Deviation from D3's text: `Gate files` runs on `pull_request_target`, so `github.event_name != 'pull_request'` would always be true there. Its guard is `${{ !github.event.pull_request.draft }}`.
- Addition: `cancel-in-progress` also excludes `edited`, on both forges. Without that, a Gitea body edit on a ready PR would cancel the same-head run in flight, which is the #87 failure. GitHub never sees `edited` in `ci.yml`, so the clause is inert there, and the twins stay byte-identical apart from the `types:` lines.
- Generated checks: not used. The contract is a fixed set of exact YAML expressions, and the table cases plus mutants cover each one.

