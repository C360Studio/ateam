# Change Proposal: `mvp-hardening-52`

## Why

Fable dogfooding round 2 (#52) identified six concrete gaps in `ateam`'s merge gate, CI workflow, OpenSpec queue/pickup reporting, build provenance, pre-compact handoff preservation, and `doctor` hook latency checks (#46–#51), plus two remaining `v0.1 MVP hardening` items (#53 context PR mentions in handoff staleness and #23 merged/stale worktree badges in Current Ops).

## What Changes

1. **Cross-agent review & merge gate (#46) + CI workflow (#47):**
   - Port `scripts/merge-check.sh`, `.github/workflows/ci.yml` (`Verify`, `Merge check`, `Required` on `ubuntu-24.04`), `task merge:check` in `Taskfile.yml`, and contract tests (`TestMergeCheckKnownFlake`, `TestMergeCheckUpToDateRule`, `TestMergeCheckReview`, `TestCIWorkflowPinned`, `TestCIWorkflowPinnedSensitivity`) from `semengine`.
   - Update `AGENTS.md`, `.agents/rules-index.md`, `.agents/protocol.md`, and `internal/bootstrap/templates.go` with the `merge:check+review` rule and Close-gate OpenSpec archiving step.
2. **Finished OpenSpec changes in `spec-queue`, `pickup`, and `doctor` (#48):**
   - Archive the four merged OpenSpec changes (`agy-hooks`, `context-window-limits`, `docs-latest-work`, `fable-blockers-39-42`).
   - Distinguish completed unarchived changes (`N/N`, `N > 0`, `0` caveats) from active in-flight changes in `FormatOpenSpecQueue`, `BuildPickupSummary`, and `ateam doctor` warnings.
3. **Empty build provenance on plain `go build` (#49):**
   - Return `unknown` from `BuildRevision()` when neither `BuildCommit` nor `vcs.revision` is present, and emit a `doctor` warning (`binary has no build provenance; build with task build`).
4. **`pre-compact` auto-handoff intent preservation (#50):**
   - Preserve the 5 semantic handoff fields when `LastHandoff` belongs to the same active `session_id` on the same branch even if `HEAD` advanced during the session; prefix fallback auto-handoffs with `UNKNOWN — Auto-saved before context compaction`.
5. **`ateam doctor` `<5ms` hook latency budget (#51):**
   - Measure warm best-of-3 hook latency in `runDoctor`, warn above `5000 us` (`5ms`), and fail `--check` above `25000 us` (`25ms`).
6. **Context PR mentions in handoff staleness (#53):**
   - Treat a fresh same-session handoff at current `HEAD`/`branch` as non-stale, and only check subject PRs (at the start of `goal` or matching `branch` when no open PR exists for `branch`) for closed/merged staleness.
7. **Merged worktree badge in Current Ops (#23):**
   - Track PR `merged` boolean and `head_sha` plus per-worktree `dirty`, `ahead_count`, and `merged_into_main` so Current Ops marks merged/stale worktrees with `✓ Merged / potentially stale` and evidence while keeping local dirty status visible and avoiding false positives on open PRs, closed-unmerged PRs, post-merge commits, or unsynced forges.
