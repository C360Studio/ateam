## 1. Approval

- [x] 1.1 The owner approves `proposal.md` and `design.md` by filling in `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit. The approval also rules on design OQ1: whether Gitea moves too, and who updates its branch protection context (#360).
  - Evidence: the owner approved in Enjoy on 2026-11-09. OQ1 is A: Gitea moves too, and the owner updates the Gitea branch protection context. Claude wrote the header at their request.

## 2. Trigger

- [x] 2.1 Switch `.github/workflows/gate-files.yml` from `pull_request_target` to `pull_request` and rewrite its header comment (D1). Do the same for `.gitea/workflows/gate-files.yml` (D2; OQ1 ruled A).
  - Premise: `.github/workflows/gate-files.yml:3-10`, `.gitea/workflows/gate-files.yml:3-11`. Types, permissions (`:16-18`), the draft/WIP guards and the job name stay byte-identical.
  - Acceptance: `TestGateFilesWorkflowTrigger` passes. `task mutate:check` reports `detection` when `pull_request:` is restored to `pull_request_target:` in either twin.
  - Evidence: at `dbe1177`, both twins trigger on `pull_request` and the GitHub and Gitea headers are rewritten. The Gitea header stays the GitHub one with two swapped lines, which the twin swaps in `TestGiteaWorkflowsTwinGitHub` pin (`gatefiles_test.go:247-250`). The mutation check used `cp` plus SHA-256 rather than `task mutate:check`, because Go's `-overlay` does not cover a YAML file the test reads with `os.ReadFile`. Restoring `pull_request_target:` in `.github/workflows/gate-files.yml`, then separately in `.gitea/workflows/gate-files.yml`, gave `detection` both times (`lacks "\non:\n  pull_request:\n"`), and both files' SHA-256 matched after restore.
- [x] 2.2 Rename `TestGateFilesWorkflowRunsFromBase` to `TestGateFilesWorkflowTrigger` and flip its trigger assertions, keeping every other ban and pin (D4).
  - Premise: `internal/bootstrap/gatefiles_test.go:173-213`, with the trigger check at `:183`, the ban at `:200`, and the comment pin at `:251`.
  - Evidence: `TestGateFilesWorkflowTrigger` requires `\non:\n  pull_request:\n` and bans `pull_request_target:`, `pull_request.head`, `head_ref`, `ref:` and `write`. It passes with `-race`, alongside `TestGateFilesCheck` and `TestGateFilesCheckGitea`.

## 3. Rule text and docs

- [x] 3.1 Rewrite the base-branch claim and add the residual-risk and fail-closed text (D3). The places: `AGENTS.md:90` and `.agents/rules-index.md:41` (same row, so the counts stay equal), `.agents/protocol.md:93`, `scripts/gate-files.sh:2-5`, and `docs/adoption.md`. The `docs/adoption.md` text covers the residual risk, what an adopter with a copied workflow does before 2026-12-02, and the Gitea context rename.
  - Acceptance: `ateam doctor --check` passes (AGENTS.md size and row parity). `grep -rn pull_request_target` outside `openspec/changes/archive/` matches only the new risk text and the `merge-check.yml` header.
  - Evidence: the `AGENTS.md` and `.agents/rules-index.md` rows, `.agents/protocol.md:93`, the `scripts/gate-files.sh` header and a new `Gate files` row in the `docs/adoption.md` merge-gate table are rewritten. `AGENTS.md` is 15,027 bytes. `ateam doctor --check` exits 0. `git grep pull_request_target` outside the archive matches only the new risk text, both `merge-check.yml` headers, the test's ban and pins, and this change's own files.
- [x] 3.2 Finalize the MODIFIED delta in `specs/harness/spec.md` (drafted with the proposal) for `Requirement: Merge gate and CI workflow enforcement`. It changes only the trigger wording in Scenarios *Gate-file changes wait for the owner* (`openspec/specs/harness/spec.md:478`) and *Draft pull requests defer CI…* (`:510-511`), and adds an AND clause for the self-judging risk and the required-name property.
  - Acceptance: `task spec:check` and `task test:spec` pass.
  - Evidence: the delta rewrites the two Scenarios and adds the self-judging and required-name AND clause. `task verify` exits 0, including `spec:check` and `test:spec`.

## 4. Live proof and close

- [ ] 4.1 On this PR, record the `pull_request` `Gate files` run URLs: one failing without the label, one passing with the label and a `gate-change-approved: <head sha>` comment the owner said yes to. Record that the owner changed the Gitea branch protection context (OQ1 A).
- [ ] 4.2 `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green on a clean tree, and a not-weaker review record is posted at the final head.
  - Evidence: at `dbe1177`, `task verify`, `ateam doctor --check` and `ateam bench --selftest` exit 0 on a clean tree. The review record is still pending.
- [ ] 4.3 Archive this change before marking the PR ready, and land it before 2026-12-02.
