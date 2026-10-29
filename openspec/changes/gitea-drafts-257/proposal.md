# Change Proposal: `gitea-drafts-257`

Approved-by:

## Why

#250 (PR #252, `e2c8d08`) stopped CI runner work on GitHub draft PRs. Gitea was left unchanged because Codex's review, working from the v1.22.6 source, found four behaviours that made the GitHub workflow unsafe there (#257). Gitea still runs the full `Verify` on every WIP push.

A live probe on 2026-10-29, on a private Gitea test instance running `28.2.0` (the current upstream 28 series, which follows 1.27.x), found that three of the four do not hold on that build:

| v1.22.6 finding | Seen on the test instance |
|---|---|
| `name:` expressions are not evaluated | Evaluated: the job posted `Probe / Dump (draft) (pull_request)` |
| A compound `always() && (...)` does not override a failed dependency | Overrides: `Agg` ran and succeeded after `Fail` failed |
| A skipped job posts `success` | Posts `skipped`, which branch protection still combines as a pass |
| Label changes arrive as `label_updated`, not `unlabeled` | **Holds**: both adding and removing a label gave `action=label_updated` |
| `edited` does not cancel an older WIP run | Cancelled: with the existing `concurrency` rule, leaving WIP cancelled the in-flight WIP run, and no stale status replaced the ready run's statuses |

Leaving WIP is an `edited` event, and the ready run posted `pending` for every job at once.

## What Changes

These apply to `.gitea/workflows/` only. GitHub is unchanged.

- `ci.yml`:
  - `edited` is added to the `pull_request` types.
  - `Verify` and `Merge check` take the #250 draft guard, `if: github.event_name != 'pull_request' || !github.event.pull_request.draft`.
  - `Required` keeps `needs: [verify, merge-check]` under `if: always() && (<guard>)`.
  - Job names stay plain (design D2).
- `gate-files.yml`:
  - `edited` is added to the trigger.
  - The job runs under `if: ${{ !github.event.pull_request.draft || github.event.action == 'label_updated' || github.event.action == 'label_cleared' }}`, so a label removal on a WIP PR still re-runs it.
- The `concurrency` rule stays, so `edited` cancels an in-flight run. This is what stops a stale WIP result (design D3).
- `TestGiteaWorkflowsTwinGitHub` and `ciWorkflowViolations(…, false)` move from "Gitea does not defer" to the Gitea deferral contract, with mutation evidence.
- The harness spec's Gitea twin line and the protocol's draft paragraph are updated.
- The minimum Gitea version is documented (open question 1).

## Track

`design`. It touches gate files (`.gitea/workflows/`) and the CI merge-gate requirement.

## Non-goals

- No GitHub workflow change; no change to `scripts/merge-check.sh`, `scripts/gate-files.sh`, or branch-protection contexts.
- No version detection inside workflows, and no support for Gitea builds that fail the probe.

## Definition of Done

- `task verify` is green on a clean tree, and `ateam doctor` parity is green.
- The workflow tests pin the Gitea contract, and every guard, the `label_updated` exception, `edited`, and the exact `always()` form each has a `detection` mutation.
- Live lifecycle evidence on the test instance, with the candidate workflows on its default branch and branch protection requiring `CI / Required (pull_request)`:
  - WIP opened
  - a WIP push
  - leaving WIP at the same SHA, including while a WIP run is in flight
  - a failed `Verify` on a ready PR, which must stay blocked
  - label revocation on a WIP PR through `label_updated`
  - an attempt to merge while WIP
- The Gitea host is never named in this repository.
- The owner adds `gate-change-approved` before landing.
