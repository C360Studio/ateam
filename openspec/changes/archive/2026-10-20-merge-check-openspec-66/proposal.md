# Change Proposal: `merge-check-openspec-66`

## Why

`scripts/merge-check.sh:144` inherited `semengine/scripts/merge-check.sh` (`AGENTS.md:163`), classifying any file whose name starts with `openspec/` as a document regardless of extension. In `ateam`, OpenSpec changes under `openspec/changes/` now carry executable reproduction scripts and lockfiles (for example, PR #64 adds `check_sentinel.py`, `replay.py`, `reproduce.py`, and `selected-wheels.lock` under `openspec/changes/mercury-runtime-qualification/offline/`), causing `merge-check.sh` to report the pull request as `documents only` and skip cross-agent review (`reviewed-by:`).

## What Changes

1. **Extension-based document classification under `openspec/` (`scripts/merge-check.sh`):**
   - Tighten the `jq` filter at `scripts/merge-check.sh:144` so a changed file (or `previous_filename` on rename) under `openspec/` is classified as a document only when its path ends in `.md`, `.yml`, `.yaml`, or `.json`.
   - Any other path under `openspec/` (such as `.py`, `.sh`, `.go`, `.lock`, or executables) is classified as code and requires `implemented-by:` and cross-agent `reviewed-by:` on non-draft pull requests.
2. **Tests, rules index, and OpenSpec contract (`internal/bootstrap/mergecheck_test.go`, `AGENTS.md`, `.agents/rules-index.md`, `openspec/specs/harness/spec.md`):**
   - Add `TestMergeCheckReview` subtests covering `.md` + `.json` + `.yml` under `openspec/changes/...` (`documents only`) and `.py` + `.lock` under `openspec/changes/...` (`is a code pull request`).
   - Update the `merge:check+review` rule rows in `AGENTS.md`, `.agents/rules-index.md`, and `internal/bootstrap/templates.go`, and update the `Merge gate and CI workflow enforcement` scenarios in `openspec/specs/harness/spec.md`.
