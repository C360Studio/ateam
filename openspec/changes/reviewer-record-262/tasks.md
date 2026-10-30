## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1-3 (#262). Hold: OPEN-Q proposal awaits owner approval

## 2. Workflows

- [ ] 2.1 Add the D1 guard and the awaiting name to the `merge-check` job in `.github/workflows/merge-check.yml:27-29` and `.gitea/workflows/merge-check.yml:28-30`, and update their header comments. Acceptance: `mergeCheckWorkflowViolations` (`internal/bootstrap/mergecheck_test.go:1039-1043`) pins the new `if:` and `name:` on both forges. `TestMergeCheckWorkflowPinnedSensitivity` (`:1082-1110`) gains the rows "awaiting guard dropped" and "name kept while awaiting", and each is reported. `TestGiteaWorkflowsTwinGitHub` passes (#262).

## 3. Contract, skill, protocol, docs

- [ ] 3.1 Add the D2 paragraph to § Record in `internal/bootstrap/review.go:41-55` and in `.agents/contracts/reviewer.md:11-22`. Acceptance: `ateam doctor --check` parity passes, and `TestReviewSetupDrift` (`internal/bootstrap/review_test.go:106`) still reports a drifted copy (#262).
- [ ] 3.2 Rewrite `review-pr` § 5 (`.agents/skills/review-pr/SKILL.md:37-39`, template `internal/bootstrap/templates.go:512`) to cover the four cases: initial rejection (absent), initial approval with a new line (the edit runs it), approval with the line already present (rerun the whole run), and approval then rejection at the same head (D3 A: whole-run rerun bound to the head, observed failing, permissions reported). Update `.agents/protocol.md:89` and `docs/daily-use.md:102` to describe the skipped state instead of "one red". Acceptance: `ateam doctor` parity passes, and no "one red `Merge check`" remains (`grep -rn 'one red' .agents docs internal/bootstrap`) (#262).
- [ ] 3.3 Spec delta `specs/harness/spec.md` modifies "Merge check runs on pull-request state events" (`openspec/specs/harness/spec.md:772-795`). Acceptance: `task spec:check` and `task test:spec` pass (#262).

## 4. Live proof and verify

- [ ] 4.1 (After ready, recorded on this PR.) Record the awaiting-skip run, then the approving-edit run passing. If Open Question 3 is approved, record the rejection-after-approval rerun failing, then a fresh approval. Run the Gitea test-instance probe for the skip and the edit run, then restore the test repo (#262).
- [ ] 4.2 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#262).
