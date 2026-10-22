## 1. Implementation

- [x] 1.1 Add `review.Policy.Compare` and `ateam review compare` with tests for every case ADR 0002 § Implementation item 2 lists (#82).
- [x] 1.2 Call compare from `scripts/merge-check.sh`, require effort once a policy exists, and gate the different-agent check on `cross_provider` (#82).
- [x] 1.3 Build the binary in the CI merge-check job and in `task merge:check` (#82).
- [x] 1.4 Record mutation detection on the tier comparison and on the band comparison: `task mutate:check -- -pkg ./internal/review -test TestCompare -file internal/review/compare.go` with `rt < dt-1`, `rb < db-1` and effort-slot (`false && p.isModel`) mutants, all `detection` at sha256 `53b59c4e` (#82).
- [x] 1.5 Update the instruction text listed in ADR 0002 § 5 (#82).
- [x] 1.6 Fix the Claude go-reviewer findings: a model in the effort slot is a finding, an empty group or a status 1 without a finding fails closed (bash 3.2), a colon in the effort is rejected, and the script header names the pairs form (#82).
