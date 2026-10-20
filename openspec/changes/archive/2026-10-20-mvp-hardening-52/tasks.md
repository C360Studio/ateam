# Tasks: `mvp-hardening-52`

## 1. Implementation & Verification

- [x] 1.1 Archive merged OpenSpec changes (`agy-hooks`, `context-window-limits`, `docs-latest-work`, `fable-blockers-39-42`) and report `done, not archived` in `spec-queue`, `pickup`, and `doctor` (#48).
- [x] 1.2 Report `build=unknown` and emit a `doctor` warning when build provenance is absent (#49).
- [x] 1.3 Preserve same-session 5-field handoff across `HEAD` advances in `pre-compact` and mark fallback auto-handoffs with `UNKNOWN —` (#50).
- [x] 1.4 Measure warm best-of-3 hook latency in `ateam doctor`, warning above `5000 us` (`5ms`) and failing `--check` above `25000 us` (`25ms`) (#51).
- [x] 1.5 Distinguish subject PRs from context PR mentions in `HandoffStaleReasons` and keep same-session current-HEAD handoffs fresh (#53).
- [x] 1.6 Port `scripts/merge-check.sh`, `.github/workflows/ci.yml`, `task merge:check`, rule/protocol updates, and contract tests (#46, #47).
- [x] 1.7 Visually mark merged/stale worktrees in Current Ops while preserving local dirty status and handling closed-unmerged, post-merge commits, and unsynced forge states (#23).
- [x] 1.8 Run `task mutate:check`, `ateam doctor --check`, and `task verify`.
