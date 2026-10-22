# Decision-opportunity tier — #111

## Claim
- [x] Read #111, the bench package, report exporter and OpenSpec layout; worktree `claude/bench-decision-tasks` from main `55d525e`.
- [ ] Open draft PR with `Closes #111` and `implemented-by: claude (opus)`.

## Implementation
- [ ] Add `ExistingSource`, the three decision tasks with good/bad/mutant/test fixtures, prompts and `existing.go` grading module.
- [ ] Add the `go/ast` discipline gate with separate behavior/discipline grade fields and retained-run validation.
- [ ] Extend self-test results, parsing and Markdown for the tier.
- [ ] Add both-passed source bytes to `compare`; project behavior/discipline in report export; regenerate synthetic report fixtures with the reason stated.
- [ ] Add Mercury dry-run examples, offline reference-adapter bundles variant and README section.
- [ ] Tests with spec attribution and independent oracles; `task mutate:check` on the discipline gate reports `detection`.

## Verify and close
- [ ] `task verify` on a clean tree and `ateam doctor --check`; record results in the PR.
- [ ] Promote the delta into `openspec/specs/benchmark-experiments/spec.md` and archive this change before the PR leaves draft.
