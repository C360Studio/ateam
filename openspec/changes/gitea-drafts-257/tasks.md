## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)`, removing this hold in the same edit, and ruling on design Open Questions 1 (D4) and 2 (D3) (#257).
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Add `edited`, the draft guard on `Verify` and `Merge check`, and `Required` under `always() && (…)` to `.gitea/workflows/ci.yml` (`:11`, `:27`, `:62`, `:100`). Acceptance: `ciWorkflowViolations(…, false)` (`internal/bootstrap/mergecheck_test.go:1069`) pins the Gitea form, and mutations that drop the guard, drop `edited`, or use plain `always()` on Gitea are each detected.
- [ ] 2.2 Add `edited` and the `label_updated`/`label_cleared` guard to `.gitea/workflows/gate-files.yml` (`:11`, `:23`). Acceptance: `TestGateFilesWorkflowRunsFromBase` asserts it, and a mutation to `unlabeled` is detected.
- [ ] 2.3 Update the `TestGiteaWorkflowsTwinGitHub` swaps (`internal/bootstrap/gatefiles_test.go:179`), the harness spec Gitea line (`openspec/specs/harness/spec.md:493`), the protocol draft paragraph (`.agents/protocol.md:41`), and the D4 minimum-version note with its probe recipe. Acceptance: `task verify` green, and `ateam doctor` parity green.

## 3. Lifecycle evidence

- [ ] 3.1 On the private test instance, with the candidate workflows on the default branch and protection requiring `CI / Required (pull_request)`, record:
  - WIP opened
  - a WIP push
  - leaving WIP at the same SHA, idle and with a WIP run in flight
  - a failed `Verify` on a ready PR, which must stay unmergeable
  - label revocation through `label_updated` on WIP
  - a merge attempt while WIP

  Acceptance: the record gives run ids and states without naming the host.
