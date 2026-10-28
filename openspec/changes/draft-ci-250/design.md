# Design: `draft-ci-250`

Drafted by claude (opus) in the design skill. This touches gate files and the CI merge gate, so it is a critical-stage design: the owner may want a second read (Fable judge or Codex) before ruling on D2.

## Context

- `.github/workflows/ci.yml:10-11` triggers `pull_request` on `opened, synchronize, reopened, ready_for_review`. `ci.yml:17-19` cancels superseded PR runs except `ready_for_review`, the #87 guard.
- `ci.yml:27-62` `Verify` (15 min) and `ci.yml:65-92` `Merge check` (base checkout, #140) have no `if:`. `ci.yml:96-111` `Required` needs both, runs `if: always()`, and fails on any result other than `success`, including `skipped`.
- `.gitea/workflows/ci.yml` differs from the GitHub file only in the merge-check token env. `TestGiteaWorkflowsTwinGitHub` (`internal/bootstrap/gatefiles_test.go:172`) holds the two files together.
- `.github/workflows/gate-files.yml:8-10` runs on `pull_request_target` for `opened, synchronize, reopened, ready_for_review, labeled, unlabeled`, from the base checkout, and has one job with no `if:`. The Gitea twin is the same apart from env.
- The `main-required-checks` ruleset (live, read 2026-10-28) requires `Required` and `Gate files`, with `strict_required_status_checks_policy: true`.
- `scripts/merge-check.sh` already exempts drafts from the review finding (`mergecheck_test.go:591`), but the known-flake finding still fails them (#250 evidence, run 40730301502).
- `TestCIWorkflowPinned` and `TestCIWorkflowPinnedSensitivity` (`mergecheck_test.go:874-960`) pin the trigger types and `cancelInProgressExpr` (`mergecheck_test.go:1074`). `TestGateFilesWorkflowRunsFromBase` (`gatefiles_test.go:134`) pins the gate-files trigger.
- Spec: "Merge gate and CI workflow enforcement" (`openspec/specs/harness/spec.md:436`), with scenarios at `:445` (Pinned CI workflow and Required aggregate) and `:457` (gate files).
- GitHub documents that a skipped job reports success to a required status check.

## Goals / Non-Goals

Goals: no `Verify`, `Merge check`, or `Gate files` runner work on draft events. Full gates on every ready revision. No path where a ready PR merges on a draft-era or stale result. Non-goals are listed in `proposal.md`.

## Owner rulings (Coby Leuschke, 2026-10-28)

D1 stands. **D2 = A** (grey drafts), with task 1.2 measured before the workflow lands: if a skipped `Required` is superseded within seconds of `ready_for_review`, ship as is; if the ready run's `Required` check run does not exist until its `needs` finish, drop the aggregate and make `Verify` the required context directly. D2 B is not taken: a red that means "not yet" teaches people to ignore red. D3 stands, plus `edited` on `Gate files`. **D4 is resolved by `edited`** (see D5). **D5 (new): `Merge check` moves to its own workflow and required context.** Full text: PR #252 / issue #250 comment of 2026-10-28.

## Decisions

### D1. Where the draft guard lives

- **A (recommended): a job-level `if:` on `Verify` and `Merge check`:** `github.event_name != 'pull_request' || !github.event.pull_request.draft`. The same guard goes on the `Gate files` job. Cost: three one-line conditions per forge. Triggers and concurrency stay as they are, so #87 is untouched. This rules out per-step skipping, which would still spend runner start-up time.
- **B: drop `opened`/`synchronize` from the trigger and rely on `ready_for_review`.** This is wrong: a push to a ready PR would then run nothing.
- **Adopter seam:** none. Workflows are not templated by `ateam init`. An adopter copying them gets draft-deferral and must keep `Required` (D2) consistent.

### D2. What `Required` reports on a draft (owner ruling)

The ready run starts at the same SHA as the last draft run. If the draft run left `Required` as `skipped`, GitHub may count it as satisfied until the ready run's `Required` check run exists. `Required` needs two jobs, so it may not exist until about 6 minutes in. Auto-merge armed right after ready could fire inside that window.

- **A: skip `Required` on drafts** (`if: always() && <not draft>`). Drafts show grey and no runner work runs. This is safe only if GitHub replaces or pends the required context as soon as the ready run is queued. That is unproven. Task 1.2 measures it on a throwaway PR before any choice is made. If the window exists, A is unsafe.
- **B (recommended): `Required` runs on drafts and fails fast** with `draft: CI deferred until ready; this is not a verification result`. Cost: one ~10s job per draft push and a red ✗ that says why. The red check is honest and cannot be read as a pass, and no stale success ever exists at the head. It does not meet the issue's wish for a draft that is not red.
- **C: skip on drafts, plus a fast first job on `ready_for_review`** that `Required` also needs. This only shrinks the window: its check run does not carry the `Required` name, so it does not close it.
- **Adopter seam:** under B, a draft's red `Required` names the reason in its log line. Under A, an adopter who arms auto-merge as they mark the PR ready inherits whatever window D2's measurement finds.

### D3. `Gate files` on drafts

The same window applies: `Gate files` is required. Its ready run is a single job queued immediately, so its window is seconds rather than minutes. Recommended: skip on draft (D1 A) and include it in the task 1.2 measurement. If the measurement shows a window, it follows D2's ruling, failing fast with `draft: gate-file check deferred`. Labels still re-run it on a ready PR, and the base-branch checkout (#140) is unchanged.

### D4. Gitea

Gitea Actions has no `ready_for_review` event, and its draft state is the `WIP:`/`[WIP]` title prefix. Whether `github.event.pull_request.draft` is populated in Gitea's payload is unverified. Recommended: use the same guard and also add the `edited` type on Gitea only, so a title change from WIP re-triggers the run. Gitea parity evidence is recorded as executed or unavailable (proposal DoD). This breaks the byte-for-byte twin rule. `TestGiteaWorkflowsTwinGitHub` gains a documented exception for the trigger line, and the alternative is accepting that a WIP → ready transition on Gitea needs a push.

### D5. `Merge check` is its own workflow and required context (owner ruling)

`Merge check` measures PR state (review record, open `class:flake`, up-to-date), not code, so its inputs change on PR-state events, never on a push. Bundled under `Required` on push events it is always evaluated before its inputs exist (#192, the flake red). `merge-check.yml` (both forges, twins apart from the token env) triggers on `pull_request: [opened, synchronize, reopened, ready_for_review, edited, labeled, unlabeled]`, skips drafts, and keeps the base-branch checkout (#140). `ci.yml` keeps `Verify` and `Required` with `needs: [verify]`. The ruleset requires `Required`, `Merge check`, `Gate files`. The reviewer's `reviewed-by:` body edit is an `edited` event at the PR head, so the check clears itself and `Verify` does not rerun; review-pr § 5's rerun step and protocol § 4's "one red Merge check" sentence go. A ready PR with no record yet shows one honest `Merge check` red; it stays. On Gitea, leaving `WIP:` is an `edited` event, so `merge-check.yml` and `gate-files.yml` need no twin exception; `ci.yml` takes either a documented exception on the trigger line or `edited` on GitHub guarded by `github.event.changes.title`. Gitea's `pull_request.draft` and `changes.title` are unverified: if absent, the Gitea guard tests the `WIP:` title prefix.

## Risks / Trade-offs

- A mis-written guard such as `github.event.pull_request.draft == false` on `push` (where `pull_request` is absent) would skip `main` verification. The guard tests `event_name` first, and a static test plus a mutation pins it.
- The ready → draft → ready sequence: `converted_to_draft` is not a trigger, so an in-flight ready run finishes and posts. That is harmless, because the next ready event reruns.

## Open Questions

None. D2 and D4 were ruled on 2026-10-28 (above); the D2 measurement (task 1.2) is a precondition, not an open question.
