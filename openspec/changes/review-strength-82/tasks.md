## 1. Implementation

- [ ] 1.1 Add `review.Policy.Compare` and `ateam review compare` with tests for every case ADR 0002 § Implementation item 2 lists (#82).
- [ ] 1.2 Call compare from `scripts/merge-check.sh`, require effort once a policy exists, and gate the different-agent check on `cross_provider` (#82).
- [ ] 1.3 Build the binary in the CI merge-check job and in `task merge:check` (#82).
- [ ] 1.4 Record mutation detection on the tier comparison and on the band comparison (#82).
- [ ] 1.5 Update the instruction text listed in ADR 0002 § 5 (#82).
