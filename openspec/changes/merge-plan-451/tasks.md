# Tasks: merge-plan-451

## 1. Approval

- [ ] 1.1 Owner approves the proposal and design.
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Coordinate step 2 launch prompt: "arm auto-merge only when my message tells you to" (premise: the prompt names no arming rule, `.agents/skills/coordinate/SKILL.md:18`, `internal/bootstrap/templates.go:413`). Accept: the sentence in both copies, rebased onto #448's prompt.
- [ ] 2.2 Coordinate step 3: the order (first `ready`, gate-file last), the shared-file check with `git diff --name-only main...<branch>`, review only after every earlier sharing unit merged, and one release at a time (premise: APPROVE leads straight to close, `SKILL.md:22`, `templates.go:417`). Accept: the sentences in both copies.
- [ ] 2.3 Coordinate step 4: name a change to `ateam state next` or the `Order:` line since the last handoff (premise: `SKILL.md:26`, `templates.go:421`). Step 5: `Merge order:` in Addresses (premise: `SKILL.md:34`, `templates.go:429`). Accept: both clauses in both copies.
- [ ] 2.4 Developer contract: arm only on the coordinator's message, catch up by a merge of `main`, hand off `ready` if that merge needs a new record (premise: no arming rule, `.agents/contracts/developer.md:7`, `internal/bootstrap/review.go:23`; clean-merge rule `internal/review/gate.go:801`). Accept: byte parity in `TestCanonicalReviewContracts`.
- [ ] 2.5 `.agents/protocol.md:45` (§ 2 step 5): one sentence sending a coordinated run to the coordinator's message before arming. Accept: protocol markers still pass `ateam doctor --check`.
- [ ] 2.6 Harness spec ADDED *Coordinator merge plan*; shipped-text test in `internal/bootstrap/coordinator_test.go` citing it, each template literal shown `detection` with `task mutate:check`.
- [ ] 2.7 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
