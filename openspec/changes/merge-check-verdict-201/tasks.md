## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#201). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `gateIsRecord` and `gateDesignated` ignore fenced lines via `unfencedLines` (#201).
- [ ] 2.2 `record` requires exactly one unfenced `Verdict: APPROVE` on the latest record, else a finding that fails the check, naming the verdict found (#201).
- [ ] 2.3 Tests on both forges: REQUEST CHANGES, APPROVE, supersession both ways, missing and duplicate verdict, fenced lines ignored; mutation checks with `task mutate:check` (#201).
- [ ] 2.4 Docs: review-pr skill § 5 and its `internal/bootstrap/templates.go` mirror, reviewer contract § Record (#201).
- [ ] 2.5 Harness spec requirement and scenarios (#201).
