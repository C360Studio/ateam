# Design: `gitea-drafts-257`

Drafted by claude (opus, medium) in the design skill. This touches gate files (`.gitea/workflows/`), so the owner may want a second read before ruling.

## Context

Lines are on `main` at `e2c8d08`.

- `.gitea/workflows/ci.yml`:
  - `:11` triggers `pull_request` on `opened, synchronize, reopened, ready_for_review`.
  - `:17-19` sets `concurrency` with `cancel-in-progress: ${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}`.
  - `:27`, `:62`, and `:98` give plain names to `Verify`, `Merge check`, and `Required`.
  - `:100` puts `Required` under exactly `if: always()`.
- `.gitea/workflows/gate-files.yml:11` triggers on `…labeled, unlabeled`, and `:23` names the job `Gate files` with no guard.
- `internal/bootstrap/gatefiles_test.go:179` `TestGiteaWorkflowsTwinGitHub` maps each GitHub draft line back to the Gitea form. `internal/bootstrap/mergecheck_test.go:884` `draftViolations` and `:1069` `ciWorkflowViolations(…, deferDrafts)` require plain names, no guard, and exactly `if: always()` when `deferDrafts` is false.
- `openspec/specs/harness/spec.md:493` says "the Gitea twins do not defer drafts". `.agents/protocol.md:41` says Gitea CI runs on WIP PRs as before.
- v1.22.6 source, read in PR #252 review: `jobparser` does not evaluate `name:`, the scheduler treats only an exact `always()` as an override, `toCommitStatus` maps skipped to success, label events are `label_updated`, and `edited` does not cancel older runs.
- Live probe, 2026-10-29, on the private test instance reporting `28.2.0`, with a throwaway `probe257.yml` on PR 5 (closed, branch and label deleted):
  - A WIP title gives `pull_request.draft=true`.
  - `name:` is evaluated (`Dump (draft)`).
  - `Agg` (`needs: [fail]`, `if: always() && (…)`) was `skipped` on WIP and ran to `success` after `Fail` failed once the PR left WIP.
  - Skipped jobs post the state `skipped`.
  - Leaving WIP posted `pending` for every job at once.
  - With the `ci.yml` concurrency rule, leaving WIP 15s into a 75s WIP job **cancelled** that run (19542), and `Hold`/`Agg` kept the ready run's `success` for 90s.
  - Adding and removing a label both gave `action=label_updated` (runs 19544, 19546).
- Gitea `main` source: `services/pull/commit_status.go` `MergeRequiredContextsCommitStatus` combines the latest status per required context, and `modules/commitstatus` `Combine` counts `skipped` as success. A skipped `Required` is therefore a pass. That is safe only because Gitea refuses to merge a WIP PR, and the ready run replaces the status at once.
- `28.2.0` is a current upstream Gitea release: the second maintenance release of the 28 series, dated 2026-10-18 ([release notes](https://blog.gitea.com/release-of-28.2.0/)). The series follows 1.27.x after a renumbering. The notes do not say in which release compound `always()`, `edited` cancellation, or `name:` evaluation arrived; they arrived somewhere after 1.22.

## Goals / Non-Goals

Goal: no `Verify` runner work on Gitea WIP pushes, with no path for a WIP-era or stale result to satisfy `CI / Required (pull_request)` or `Gate files` on a ready PR. Non-goals are in `proposal.md`.

## Decisions

### D1. Guard form on Gitea

- **A (recommended): the #250 guard, plus `edited`.**
  - `Verify` and `Merge check` take `if: github.event_name != 'pull_request' || !github.event.pull_request.draft`, and `Required` takes `always() && (…)`.
  - `edited` is added to the `ci.yml` and `gate-files.yml` types, so leaving WIP starts the ready run.
  - Cost: about five lines per file.
  - It relies on compound `always()`, which v1.22.6 does not honour. There, a failed `Verify` on a ready PR would leave `Required` skipped, which is a pass. This forecloses older Gitea (see D4).
- **B: keep exactly `if: always()` on `Required` and decide WIP inside its step.**
  - This works on every version, but on WIP the step must either pass, which reports a draft as verified (#250 forbids that), or fail, which turns WIP red (the noise #250 removed).
- **Adopter seam:** a Gitea adopter copying these workflows needs a Gitea build that passes the D4 probe. If they do nothing, older Gitea fails open on a ready PR with a failed prerequisite.

### D2. Names on Gitea

- **A (recommended): keep plain names.** No stale window needs closing. Leaving WIP posts `pending` at once and cancels the WIP run, and Gitea refuses to merge WIP. The rename would also break the required context on any build that does not evaluate `name:`.
- **B: copy the GitHub rename.** The twins would match byte for byte, but that buys nothing here and costs compatibility.

### D3. `edited` and cancellation

- Leaving WIP is an `edited` event. The existing `cancel-in-progress` expression is true for `edited`, so the WIP run is cancelled. That cancellation is the observed protection against a stale WIP overwrite.
- Cost: a description edit on a ready Gitea PR, such as the reviewer's `reviewed-by:` line, also reruns CI and cancels the in-flight run. The cancelled run posts a non-pass, which fails closed, until the new run finishes. That is noise, not a safety gap. A title-only filter is not possible on Gitea 28.2.0. Probes on 2026-10-29 (runs 19550-19555) showed that `github.event.changes` is `null` for both a body edit and a title edit, so the event does not say which field changed. The WIP state itself is read from the title, through `pull_request.draft`; only the trigger is broad.

### D4. Minimum Gitea version (open question 1)

- **A (recommended): require Gitea 28.x, the verified series, and document it.** The probe steps ship as the acceptance recipe in `docs/` for anyone on another build, and older Gitea should keep the pre-#257 twins. Pinning the exact 1.2x release where each behaviour arrived is not attempted: there is no older instance to probe.
- **B: keep Gitea unchanged** until the release line of `28.2.0` is identified.

### D5. `Gate files` label revocation

Use `!github.event.pull_request.draft || github.event.action == 'label_updated' || github.event.action == 'label_cleared'`. Any label change on a WIP PR re-runs the gate, so revoking `gate-change-approved` posts a failure under `Gate files` while the PR is still WIP. `unlabeled` is not needed on Gitea.

## Risks / Trade-offs

- The probe ran on a single instance of unknown lineage. D4 A makes the version a documented precondition, not something the workflows detect.
- The twin test grows: the Gitea `types:`, `if:`, and `edited` lines become documented exceptions to the GitHub file.

## Open Questions

1. **D4 ruled 2026-10-29 by the owner (Enjoy): Gitea 28.x is the minimum.**
2. **D3 ruled 2026-10-29 by the owner (Enjoy): accept a CI rerun on every Gitea PR edit.** A title-only filter is impossible on Gitea 28.2.0, because `changes` is `null`.
