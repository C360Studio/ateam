## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)`, removing this hold in the same edit, and ruling on design Open Questions 1 (D4) and 2 (D3) (#257).
  Evidence: the owner approved in Enjoy on 2026-10-29. D4: Gitea 28.x is the minimum. D3: accept a CI rerun on any Gitea PR edit. Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 Add `edited`, the draft guard on `Verify` and `Merge check`, and `Required` under `always() && (…)` to `.gitea/workflows/ci.yml` (`:11`, `:27`, `:62`, `:100`). Acceptance: `ciWorkflowViolations(…, false)` (`internal/bootstrap/mergecheck_test.go:1069`) pins the Gitea form, and mutations that drop the guard, drop `edited`, or use plain `always()` on Gitea are each detected.
- [x] 2.2 Add `edited` and the `label_updated`/`label_cleared` guard to `.gitea/workflows/gate-files.yml` (`:11`, `:23`). Acceptance: `TestGateFilesWorkflowRunsFromBase` asserts it, and a mutation to `unlabeled` is detected.
- [x] 2.3 Update the `TestGiteaWorkflowsTwinGitHub` swaps (`internal/bootstrap/gatefiles_test.go:179`), the harness spec Gitea line (`openspec/specs/harness/spec.md:493`), the protocol draft paragraph (`.agents/protocol.md:41`), and the D4 minimum-version note (`docs/adoption.md`, merge-gate table). Acceptance: `task verify` green, and `ateam doctor` parity green.

## 3. Lifecycle evidence

- [x] 3.1 On the private test instance, with the candidate workflows on the default branch and protection requiring `CI / Required (pull_request)`, record:
  - WIP opened
  - a WIP push
  - leaving WIP at the same SHA, idle and with a WIP run in flight
  - a failed `Verify` on a ready PR, which must stay unmergeable
  - label revocation through `label_updated` on WIP
  - a merge attempt while WIP
  Result (2026-10-29 UTC, private Gitea 28.2.0 test instance). The candidate tree (`486358e`) was committed onto the test repo's default branch, whose branch protection requires `CI / Required (pull_request)` with `block_on_outdated_branch`. Afterwards the tree was restored to its earlier content (identical tree, new commit). Each item names its throwaway PR, all now closed with their branches deleted:
  - **WIP opened / WIP push** (PR W, docs-only): `Verify`, `Merge check`, `Required`, and `Gate files` all posted `skipped` on both heads. A merge attempt while WIP returned `405 Work in progress PRs cannot be merged`.
  - **Leaving WIP at the same SHA** (PR W): all four contexts were `pending` within 2s. A merge attempt at +3s returned `Not all required status checks successful`. `Gate files` and `Merge check` succeeded, and `Required` and `Verify` succeeded at about +269s.
  - **Leaving WIP with a WIP run in flight:** on the real workflows a WIP run skips every job at once, so there is no long run to overlap. The probe workflow showed the protection: run 19542, held open 75s, was cancelled when WIP was removed, and no stale status replaced the ready run's statuses for 90s.
  - **Failed `Verify` on a ready PR** (PR F, a deliberately failing Go test, run 19563): `Verify` failed, `Merge check` failed (code PR without a review record), and `Required` **ran** under the compound `always()` and failed. A merge attempt was refused.
  - **Label revocation on WIP** (PR G, touching `scripts/gate-files.sh`):
    - `Gate files` failed while unlabelled, and passed with `gate-change-approved`.
    - Going WIP posted `skipped`.
    - Removing the label while WIP ran `Gate files` on `label_updated`, and it **failed**.
    - Leaving WIP ran it again, and it failed again.
    - A merge attempt was refused.
  - Not exercised: `label_cleared`, which removes every label at once. It is covered by the expression test and the `label_cleared` mutant.

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
- Codex review at `7c3e12a` (REQUEST CHANGES, P1): rerunning a completed WIP run after the PR is ready would post `skipped` under `CI / Required` or `Gate files`. Fixed at `734ed13` by renaming skipped WIP jobs, as on GitHub. Twenty-one workflow mutants gave `detection`, including Gitea rename dropped on `Required`, `Verify`, and `Gate files`, and the rename ignoring `label_updated`.
- **Rerun history, live** (same private 28.2.0 instance, candidate tree `734ed13` on the test default branch, then restored). PR R was opened as WIP, with a failing Go test and a `scripts/gate-files.sh` touch:
  - The WIP runs (CI 19576, Gate files 19577) posted only `CI / … (draft, skipped)` and `Gate files (draft, skipped)` contexts.
  - Leaving WIP posted `pending` within 3s. `Required` failed at +216s, and `Gate files` failed without the label.
  - `POST …/actions/runs/19576/rerun` and `…/19577/rerun` both returned 201. Their jobs skipped again under the draft names only. `CI / Required (pull_request)` and `Gate files` stayed `failure`, the combined status stayed `failure`, and a merge returned `Not all required status checks successful`.
  - Not reproduced: the unfixed overwrite itself, a baseline on `7c3e12a`. The finding is source-level, and the fix removes the required-name path.

