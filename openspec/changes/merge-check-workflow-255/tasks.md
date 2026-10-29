## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1-3 (#255). Hold: OPEN-Q proposal awaits owner approval

## 2. Workflows

- [ ] 2.1 After PR #258 lands, rebase on `main`. Move the `merge-check` job from `.github/workflows/ci.yml:62-94` into `.github/workflows/merge-check.yml` with the D2 event, D4 triggers and concurrency, and the #250 draft guard and rename. `required` (`ci.yml:103-106`) needs only `verify`. The Gitea twin follows #258's form. Acceptance: a new `TestMergeCheckWorkflowPinned` and its sensitivity table, the updated `TestCIWorkflowPinned`, `TestGateFilesWorkflowRunsFromBase` (base checkout read from the new file), and `TestGiteaWorkflowsTwinGitHub` (new file in the swap table) pass. `task mutate:check` reports `detection` with `edited` dropped from the trigger list and with the draft guard dropped (#255).

## 3. Rule check

- [ ] 3.1 Implement the D3 ruling in `internal/review/gate_github.go:159-176` and `gate_gitea.go:166-187` (`gate.go:59`). Acceptance: `internal/review/gate_test.go` rows cover a ruleset with every required context, one missing, and the default with no env on both forges. `task mutate:check` reports `detection` with the per-context loop cut to the first entry (#255).

## 4. Docs and spec

- [ ] 4.1 `/review-pr` step 5 (`.agents/skills/review-pr/SKILL.md:37-39` and its template `internal/bootstrap/templates.go:514`) follows D5. `docs/daily-use.md:99`, `.agents/protocol.md`, and the harness spec (the ADDED Requirement here, plus a MODIFIED "Merge gate and CI workflow enforcement" for `openspec/specs/harness/spec.md:434-489` once D2-D4 are ruled) describe the new workflow and the three contexts. Acceptance: `ateam doctor` parity and `task spec:check` pass (#255).

## 5. Live proof and verify

- [ ] 5.1 The owner adds `Merge check` to the `main` ruleset. A cross-agent review on this PR clears `Merge check` through the description edit with no manual rerun; the PR records the run IDs. A Gitea test-instance probe records the new context, and the test repo is restored (#255).
- [ ] 5.2 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#255).
