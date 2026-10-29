## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)`, removing this hold in the same edit, and ruling on design Open Questions 1 (D4) and 2 (D3) (#257).
  Evidence: the owner approved in Enjoy on 2026-10-29. D4: Gitea 28.x is the minimum. D3: accept a CI rerun on any Gitea PR edit. Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 Add `edited`, the draft guard on `Verify` and `Merge check`, and `Required` under `always() && (…)` to `.gitea/workflows/ci.yml` (`:11`, `:27`, `:62`, `:100`). Acceptance: `ciWorkflowViolations(…, false)` (`internal/bootstrap/mergecheck_test.go:1069`) pins the Gitea form, and mutations that drop the guard, drop `edited`, or use plain `always()` on Gitea are each detected.
- [x] 2.2 Add `edited` and the `label_updated`/`label_cleared` guard to `.gitea/workflows/gate-files.yml` (`:11`, `:23`). Acceptance: `TestGateFilesWorkflowRunsFromBase` asserts it, and a mutation to `unlabeled` is detected.
- [x] 2.3 Update the `TestGiteaWorkflowsTwinGitHub` swaps (`internal/bootstrap/gatefiles_test.go:179`), the harness spec Gitea line (`openspec/specs/harness/spec.md:493`), the protocol draft paragraph (`.agents/protocol.md:41`), and the D4 minimum-version note (`docs/adoption.md`, merge-gate table). Acceptance: `task verify` green, and `ateam doctor` parity green.

## 3. Lifecycle evidence

- [ ] 3.1 On the private test instance, with the candidate workflows on the default branch and protection requiring `CI / Required (pull_request)`, record:
  - WIP opened
  - a WIP push
  - leaving WIP at the same SHA, idle and with a WIP run in flight
  - a failed `Verify` on a ready PR, which must stay unmergeable
  - label revocation through `label_updated` on WIP
  - a merge attempt while WIP

  Acceptance: the record gives run ids and states without naming the host.

## Evidence

- 2.1-2.3: `TestCIWorkflowPinned`, `TestCIWorkflowPinnedSensitivity`, `TestGateFilesWorkflowRunsFromBase`, and `TestGiteaWorkflowsTwinGitHub` in `internal/bootstrap`. `ciWorkflowViolations(…, renamed)` now requires the draft guard and the compound `always()` on both forges, and plain names on Gitea. Nineteen workflow mutants were each applied with `cp` and SHA-256 restore, and all gave `detection`: the eleven GitHub mutants from #250, plus eight Gitea ones.
  - `Verify` guard dropped
  - `Required` back to plain `always()`
  - guard narrowed to `!draft`, which skips push
  - `ci.yml` `edited` dropped
  - `ci.yml` taking the GitHub rename
  - `Gate files` matching `unlabeled` instead of `label_updated`
  - `label_cleared` dropped
  - `gate-files.yml` `edited` dropped
- Deviation: D4 A called for shipping the probe steps as a recipe in `docs/`. After the owner ruled 28.x as the minimum, the docs only state the requirement and its reason (`docs/adoption.md`), because a recipe would be unrequested surface. `.github/workflows/ci.yml` changed only in a comment that pointed at the old Gitea form.
- Generated checks: not used. The contract is a fixed set of exact YAML expressions, and the table cases plus mutants cover each one.

