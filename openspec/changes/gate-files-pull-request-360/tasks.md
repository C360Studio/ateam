## 1. Approval

- [ ] 1.1 The owner approves `proposal.md` and `design.md` by filling in `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit. The approval also rules on design OQ1: whether Gitea moves too, and who updates its branch protection context (#360).
  - Hold: OPEN-Q proposal awaits owner approval

## 2. Trigger

- [ ] 2.1 Switch `.github/workflows/gate-files.yml` from `pull_request_target` to `pull_request` and rewrite its header comment (D1). Do the same for `.gitea/workflows/gate-files.yml` (D2; OQ1 ruled A).
  - Premise: `.github/workflows/gate-files.yml:3-10`, `.gitea/workflows/gate-files.yml:3-11`. Types, permissions (`:16-18`), the draft/WIP guards and the job name stay byte-identical.
  - Acceptance: `TestGateFilesWorkflowTrigger` passes. `task mutate:check` reports `detection` when `pull_request:` is restored to `pull_request_target:` in either twin.
- [ ] 2.2 Rename `TestGateFilesWorkflowRunsFromBase` to `TestGateFilesWorkflowTrigger` and flip its trigger assertions, keeping every other ban and pin (D4).
  - Premise: `internal/bootstrap/gatefiles_test.go:173-213`, with the trigger check at `:183`, the ban at `:200`, and the comment pin at `:251`.

## 3. Rule text and docs

- [ ] 3.1 Rewrite the base-branch claim and add the residual-risk and fail-closed text (D3). The places: `AGENTS.md:90` and `.agents/rules-index.md:41` (same row, so the counts stay equal), `.agents/protocol.md:93`, `scripts/gate-files.sh:2-5`, and `docs/adoption.md`. The `docs/adoption.md` text covers the residual risk, what an adopter with a copied workflow does before 2026-12-02, and the Gitea context rename.
  - Acceptance: `ateam doctor --check` passes (AGENTS.md size and row parity). `grep -rn pull_request_target` outside `openspec/changes/archive/` matches only the new risk text and the `merge-check.yml` header.
- [ ] 3.2 Finalize the MODIFIED delta in `specs/harness/spec.md` (drafted with the proposal) for `Requirement: Merge gate and CI workflow enforcement`. It changes only the trigger wording in Scenarios *Gate-file changes wait for the owner* (`openspec/specs/harness/spec.md:478`) and *Draft pull requests defer CI…* (`:510-511`), and adds an AND clause for the self-judging risk and the required-name property.
  - Acceptance: `task spec:check` and `task test:spec` pass.

## 4. Live proof and close

- [ ] 4.1 On this PR, record the `pull_request` `Gate files` run URLs: one failing without the label, one passing with the label and a `gate-change-approved: <head sha>` comment the owner said yes to. Record that the owner changed the Gitea branch protection context (OQ1 A).
- [ ] 4.2 `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green on a clean tree, and a not-weaker review record is posted at the final head.
- [ ] 4.3 Archive this change before marking the PR ready, and land it before 2026-12-02.
