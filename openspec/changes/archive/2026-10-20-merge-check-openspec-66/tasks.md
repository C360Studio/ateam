# Tasks: `merge-check-openspec-66`

## 1. Implementation & Verification

- [x] 1.1 Tighten `scripts/merge-check.sh` `jq` filter so files under `openspec/` are classified as documents only when ending in `.md`, `.yml`, `.yaml`, or `.json` (#66).
- [x] 1.2 Add `TestMergeCheckReview` cases in `internal/bootstrap/mergecheck_test.go` for `openspec/` `.md`/`.json`/`.yml` documents-only changes and `openspec/` `.py`/`.lock` code changes (#66).
- [x] 1.3 Update `AGENTS.md`, `.agents/rules-index.md`, `internal/bootstrap/templates.go`, and `openspec/specs/harness/spec.md` (`Merge gate and CI workflow enforcement`), prove mutation detection, and verify `scripts/merge-check.sh 64` and `task verify`.
