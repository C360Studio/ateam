# Change: Enforce commit-bound review records and re-review after a content commit (#75)

## Why

`scripts/merge-check.sh` previously checked only that a code PR's description had `implemented-by:` and `reviewed-by:` naming different agents without verifying that a PR comment review record existed for a commit in the PR's commit list or that any subsequent commits were limited to clean merges of `main` or documents-only edits.

## What Changes

- Require a PR comment review record (`reviewed-by: <agent>` or a `### Independent Cross-Agent Review` heading) naming a full or short commit SHA in the PR's commit list.
- Allow commits after the latest recorded review commit only when each is a merge of `main` whose diff touches none of the PR's changed files or a documents-only commit (same classification as #66), failing any other later commit with `review record at <sha> is older than content commit <sha>; request re-review`.
- Update `AGENTS.md`, `.agents/rules-index.md`, `internal/bootstrap/templates.go`, `openspec/specs/harness/spec.md`, and `TestMergeCheckReview`.
