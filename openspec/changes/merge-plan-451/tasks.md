# Tasks: merge-plan-451

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-28) sha256:9da6d30c9485`).

## 2. Implementation

- [x] 2.1 Coordinate step 2 launch prompt: "arm auto-merge only when my message tells you to" (premise: the prompt names no arming rule, `.agents/skills/coordinate/SKILL.md:18`, `internal/bootstrap/templates.go:413`). Accept: the sentence in both copies; rebase onto #448's prompt if it lands first.
- [x] 2.2 Coordinate step 3: the order (first `ready`, gate-file last), the shared-file check with `git diff --name-only main...<branch>`, review only after every earlier sharing unit merged, and one release at a time (premise: APPROVE leads straight to close, `SKILL.md:22`, `templates.go:417`). Accept: the sentences in both copies.
- [x] 2.3 Coordinate step 4: name a change to `ateam state next` or the `Order:` line since the last handoff (premise: `SKILL.md:26`, `templates.go:421`). Step 5: `Merge order:` in Addresses (premise: `SKILL.md:34`, `templates.go:429`). Accept: both clauses in both copies.
- [x] 2.4 Developer contract: arm only on the coordinator's message, catch up by a merge of `main`, hand off `ready` if that merge needs a new record (premise: no arming rule, `.agents/contracts/developer.md:7`, `internal/bootstrap/review.go:23`; clean-merge rule `internal/review/gate.go:801`). Accept: byte parity in `TestCanonicalReviewContracts`.
- [x] 2.5 `.agents/protocol.md:45` (§ 2 step 5): one sentence sending a coordinated run to the coordinator's message before arming, as its own sentence so the Close wording `wording_test.go:36` pins stays whole; `protocolCloseStep` (`templates.go:726`) carries it. Accept: `TestInitRepoAndParitySensitivity` and `ateam doctor --check`.
- [x] 2.6 Harness spec ADDED *Coordinator merge plan*; shipped-text test in `internal/bootstrap/coordinator_test.go` citing it, each template literal shown `detection` with `task mutate:check` (`templates.go` "Release one approved PR at a time" and `review.go` "arm auto-merge only when its message tells you to": both `detection`).
- [ ] 2.7 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
