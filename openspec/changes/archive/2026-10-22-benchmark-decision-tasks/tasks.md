# Decision-opportunity tier — #111

## Claim
- [x] Read #111, the bench package, report exporter and OpenSpec layout; worktree `claude/bench-decision-tasks` from main `55d525e`.
- [x] Open draft [PR #114](https://github.com/C360Studio/ateam/pull/114) with `Closes #111` and `implemented-by: claude (opus)`.

## Implementation
- [x] Add `ExistingSource`, the three decision tasks with good/bad/mutant/test fixtures, prompts and `existing.go` grading module.
- [x] Add the `go/ast` discipline gate with separate behavior/discipline grade fields and retained-run validation.
- [x] Extend self-test results, parsing and Markdown for the tier.
- [x] Add both-passed source bytes to `compare`; project behavior/discipline in report export; regenerate synthetic report fixtures with the reason stated.
- [x] Add Mercury dry-run examples, offline reference-adapter bundles variant and README section.
- [x] Tests with spec attribution and independent oracles; `task mutate:check` on the discipline gate reports `detection`.

## Verify and close
- [x] `task verify` on a clean tree and `ateam doctor --check`; record results in the PR.
- [x] Promote the delta into `openspec/specs/benchmark-experiments/spec.md` and archive this change before the PR leaves draft.

## Completion evidence

- Task limit kept at 1..3; the tier runs as its own experiment (no validation change).
- `task mutate:check` on `internal/bench/decision.go` reported `detection` for the literal-once, forbidden-call and exported-name rules (`TestDisciplineGateFindings`) and for retained-grade consistency (`TestDecisionExperimentEvidence`).
- Offline reference-adapter run of `benchmarks/examples/decision-bundles-ab.json` completed 6/6 attempts, compared, rescored (replay `matched`) and exported with per-attempt behavior and discipline.
- Synthetic report fixtures rebound to the new evaluator digest; reason recorded in `internal/bench/testdata/report/README.md`.
- Final-head `task verify`, `ateam doctor --check` and command output are recorded in PR #114. Cross-agent review is arranged by the owner; the PR stays draft until then.
- Current requirements are promoted to `openspec/specs/benchmark-experiments/spec.md`; this directory retains the historical proposal, design, delta and checklist.
