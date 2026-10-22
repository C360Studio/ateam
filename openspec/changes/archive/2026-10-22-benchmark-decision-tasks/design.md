# Design: decision-opportunity tasks and discipline grading

## Task shape

`TaskSpec` gains `ExistingSource` (the pre-existing package file, `json:"-"`) and an unexported discipline rule. The tier's fixtures live in `decision.go`, which joins `bench.go` and the process files in the embedded evaluator-source digest so a gate change invalidates stale rescoring.

| Task | Existing file | Request | Discipline rule | Bad (passes tests) | Mutant (fails tests) |
| --- | --- | --- | --- | --- | --- |
| `reuse-email` | `normalizeEmail` helper | `SameAccount(a, b string) bool` | calls `normalizeEmail`; no `strings.TrimSpace`, `strings.ToLower` or `strings.EqualFold` | re-derives normalization with `strings` | normalizes one side only |
| `overbuild-slug` | none | `Slugify(title string) string` | `Slugify` is the only exported top-level name | exported `Options` struct and `SlugifyWith` wrapper | leaves a trailing hyphen |
| `propagate-limit` | `Validate` using `MaxLen` and `LengthError` | return `limits.go` with `MaxLen` raised from 16 to 24 | no INT or STRING literal still carries the replaced whole number `16` | raises the constant, leaves the message at 16 | constant off by one (23) |

The reuse gate names the helper's own calls rather than deriving them from its body: explicit, and the fixture is fixed. The propagation gate is literal-level: INT literals compare by value through `go/constant` (so `0x10` is 16) and STRING literals match the value only at digit boundaries (so `116` and `240` do not count). It cannot see dot imports or computed values, which is acceptable for a fixed fixture and documented as a gate limit, not a general analysis. Review of c8e0398 replaced an earlier "24 in exactly one literal" rule: that failed the minimal complete edit (both the constant and the message changed to 24), so it graded an unrequested refactor rather than propagation. The behavior tests no longer assert the exact message; a hand-edited or derived message both pass.

## Prompt and grading module

`taskPrompt` appends the existing file inside a fenced block after one sentence: it already exists in the package, is compiled with your file, and must not be returned. `fixtureModule` writes it as `existing.go`. A candidate that repeats the file fails to compile (redeclaration), which is a behavior failure. The package-mismatch diagnostic check accepts `go`'s message when `existing.go` sorts first.

## Recorded evidence

`CandidateEvaluation` adds `behavior`, `discipline_gate` (`passed`/`failed`) and `discipline_diagnostics`, all `omitempty`; they stay comparable strings so existing equality checks on the grade still compile. For the tier, `passed = behavior && discipline` and the verdict follows it. The discipline gate runs even when behavior fails so a reviewer sees both signals. Retained-run validation rejects decision attempts missing either field or contradicting `passed`, and non-decision attempts that carry them. Report export projects the two fields per attempt (`omitempty`) and replay association flags changed behavior or discipline.

Self-test `TaskResult` adds `good_passed_discipline`, `bad_passed_behavior`, `bad_caught_by_discipline` (`omitempty`) with evaluation `discipline`. The good fixture must pass both gates, the bad fixture must pass the behavior tests through the real candidate grader and fail the discipline gate, and the mutant must be detected by `mutate.Run`. `bad_caught_by_gate` means "caught as intended", that is, by discipline only. Discipline tasks count toward evaluated/passed totals, not the existing coding-only safety and mutation rates, whose meaning is unchanged.

## Comparison

`Comparison.both_passed_source_bytes` maps arm to summed retained source bytes over both-passed pairs; omitted when there are none. `FormatComparison` prints one descriptive line. No winner or threshold.

## Compatibility

New fields are `omitempty`, so existing grades and comparisons without both-passed pairs serialize unchanged. The evaluator digest covers grader sources by design, so editing `bench.go` and adding `decision.go` changes `evaluator_sha256` for every task set: runs recorded by an earlier binary report `evaluator mismatch` and must be rescored or exported with their matching binary, as before. The synthetic report fixtures under `internal/bench/testdata/report` are test-owned; their inputs carry only an updated evaluator digest and run digest, with outcomes and other hashes unchanged, and the generated outputs are re-exported. Saved self-test reports from the 7-task suite no longer validate against the 10-task suite, matching existing totals validation.

## Task limit

Kept at 1..3. The tier runs as its own three-task experiment; raising the limit would touch validation, tests and artifact-size reasoning for no current need.
