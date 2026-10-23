## 1. Implementation

- [x] 1.1 `Policy.Compare`: placeholder efforts are missing; an unranked model or effort passes only when both sides are unranked, else a finding naming the side with tier-entry suggestions (#125).
- [x] 1.2 `TestCompare` cases for each side unranked, both unranked, unmapped effort at equal and unequal tiers, placeholders, codex `none`, and the suggestion; `TestMergeCheckReview` fixtures moved to the new ruling. `task mutate:check -- -pkg ./internal/review -test TestCompare -file internal/review/compare.go`, all `detection` at sha256 `1e9794c4`: one-side-unranked tier passes, one-side-unranked band passes, placeholders allowed, wrong side named, suggestion dropped (#125).
- [x] 1.3 ADR 0002 § 2 and § 3 amended, model-review spec scenario, reviewer contract text (#125).
