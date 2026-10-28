# Design: `draft-ci-250`

Drafted by claude (opus) in the design skill; amended 2026-10-28 by the owner (Fable judge session) after task 1.2 and a Gitea source read. Owner rulings are marked **(ruled)**.

## Context

- `.github/workflows/ci.yml:10-11` triggers `pull_request` on `opened, synchronize, reopened, ready_for_review`. `ci.yml:17-19` cancels superseded PR runs except `ready_for_review`, the #87 guard.
- `ci.yml:27-62` `Verify` (15 min) and `ci.yml:65-92` `Merge check` (base checkout, #140) have no `if:`. `ci.yml:96-111` `Required` needs both, runs `if: always()`, and fails on any result other than `success`, including `skipped`.
- `.gitea/workflows/ci.yml` differs from the GitHub file only in the merge-check token env. `TestGiteaWorkflowsTwinGitHub` (`internal/bootstrap/gatefiles_test.go:172`) holds the two files together, with one text exception already for `gate-files.yml`.
- `.github/workflows/gate-files.yml:8-10` runs on `pull_request_target` for `opened, synchronize, reopened, ready_for_review, labeled, unlabeled`, from the base checkout, one job, no `if:`.
- The `main-required-checks` ruleset (live, read 2026-10-28) requires `Required` and `Gate files`, `strict_required_status_checks_policy: true`.
- `TestCIWorkflowPinned` and `TestCIWorkflowPinnedSensitivity` (`mergecheck_test.go:874-960`) pin the trigger types and `cancelInProgressExpr` (`mergecheck_test.go:1074`). `TestGateFilesWorkflowRunsFromBase` (`gatefiles_test.go:134`) pins the gate-files trigger.
- Spec: "Merge gate and CI workflow enforcement" (`openspec/specs/harness/spec.md:436`), scenarios at `:445` and `:457`.
- GitHub docs (contexts, "Context availability"): `jobs.<job_id>.name` may use the `github` context. GitHub docs (troubleshooting required status checks): a job skipped by `if:` reports success to a required check; a required context with no check run shows "Waiting for status to be reported" and blocks.
- Task 1.2 (throwaway PR #254): with a plain `if:` skip, the draft's skipped `Required` was the only `Required` at the head for 129 s after `gh pr ready`, because the ready run's `Required` check run appears only when its `needs` finish.
- Gitea (`go-gitea/gitea` `main`, read 2026-10-28): `modules/structs/pull.go` has `Draft bool json:"draft"` since v1.22.0, set from `IsWorkInProgress` (case-insensitive title prefix in `WORK_IN_PROGRESS_PREFIXES`, default `WIP:,[WIP]`). The Actions payload is `ToPayloadPullRequest` → same struct, and `GenerateGiteaContext` exposes it as `github.event`. `services/actions/run.go` parses with `WithGitContext`, and `jobparser.jobName` interpolates `name:` with it, so the status context `<workflow> / <job> (<event>)` carries the evaluated name. Merge gating (`services/pull/commit_status.go`, `modules/commitstatus`) takes the latest status per required context; all-skipped combines to pending (blocked), skipped beside a success combines to success. Title edits emit `edited`; there is no `ready_for_review`. **Correction (Codex review on PR #252, checked against v1.22.6):** v1.22.6 pins `gitea.com/gitea/act v0.259.1`, which does not interpolate `job.Name`. A ready run's status context would therefore be the raw expression, not `CI / Required (pull_request)`. In v1.22.6, `services/actions/commit_status.go:140-151` maps a skipped job to `success`, and `notifier_helper.go:354-364` calls `CreateCommitStatus` for every job as soon as `InsertRun` returns. A run created on leaving WIP therefore replaces a skipped `success` with `pending` at once.

## Goals / Non-Goals

Goals: zero runner work and zero red on draft events on both forges; full gates on every ready revision; no path where a ready PR merges on a draft-era result. Non-goals are in `proposal.md`.

## Decisions

### D1. Where the draft guard lives

Job-level `if:` on `Verify`, `Merge check`, `Required`, and `Gate files`: `github.event_name != 'pull_request' || !github.event.pull_request.draft` (`pull_request_target` for `Gate files`). `Required` writes it as `always() && (...)`, since the existing `always()` would otherwise override the skip. Triggers and concurrency stay as they are, so #87 is untouched. Dropping `opened`/`synchronize` from the trigger was rejected: a push to a ready PR would then run nothing.

### D2. What a draft leaves at its head **(ruled: nothing under a required name)**

A skipped job still creates a check run under the job's name, and GitHub counts it as passing. So each guarded job also takes `name: ${{ github.event.pull_request.draft && '<job> (draft, skipped)' || '<job>' }}`. A draft head then has skipped check runs under non-required names and no check run named `Required` or `Gate files`; the required contexts read "Waiting for status" and block. On `ready_for_review` at the same SHA the ready run is the first to create them. The 129 s window of task 1.2 does not exist in this shape, the `Required` aggregate stays as it is, and the ruleset is unchanged. Rejected: B (fail fast, red on every draft push: a red that means "not yet" teaches people to ignore red); C (a fast first job: its check run does not carry the required name); dropping the aggregate so `Verify` is required directly (shrinks the window to seconds, does not close it).

### D3. `Gate files` on drafts

Same guard and rename, except that `unlabeled` still runs on a draft: `if: ${{ !github.event.pull_request.draft || github.event.action == 'unlabeled' }}`, and the name keeps `Gate files` for that action. Otherwise, take a PR that passed `Gate files` at head H, convert it to draft, and remove `gate-change-approved`. The removal would be skipped under another name, and the earlier pass at H would remain until the ready run's `Gate files` appears (Codex review, PR #252). A draft that touches gate files therefore turns red when the label is removed, which is a real revocation. Labels still re-run it on a ready PR, and the base-branch checkout (#140) is unchanged.

### D4. Gitea **(ruled: same bytes, plus `edited` on the Gitea trigger lines)**

The guard and rename work unchanged on Gitea 1.22+ (Context). Below 1.22 `draft` is absent, the expression is falsy, and CI keeps running on WIP PRs as today: inert, not a regression. Because leaving WIP is a title edit, `.gitea/workflows/ci.yml` and `.gitea/workflows/gate-files.yml` add `edited` to their `types:` so the ready transition starts the run; GitHub does not take `edited` (it would rerun `Verify` on every body edit, including the `reviewed-by:` line). `TestGiteaWorkflowsTwinGitHub` gains one documented exception for the `types:` line of each file. **Amended after review:** the Gitea twins keep plain job names, because Gitea 1.22 does not evaluate `name:` (Context correction). Gitea does not need the rename either: a new run posts pending statuses for every job when it is created, so no skipped `success` survives the WIP → ready edit. The twin test lists the three `ci.yml` names, the `Gate files` name, and their comments as Gitea exceptions, and `ciWorkflowViolations` checks the plain names on Gitea.

### D5. Merge-check split **(ruled: parked, not in this change)**

`Merge check` measures PR state (review record, open `class:flake`, up-to-date), which changes on PR-state events and never on a push; evaluating it under `Required` on push events is why #192's post-ready red and the open-flake red exist. The fix is its own workflow and required context triggered on `edited`/`labeled`/`unlabeled` too, with a ruleset edit. It is a different problem from #250 and is filed as #255.

## Risks / Trade-offs

- A mis-written guard such as `github.event.pull_request.draft == false` on `push` (where `pull_request` is absent) would skip `main` verification. The guard tests `event_name` first; a static test plus a mutation pins it, and another pins the `always() &&` form on `Required`.
- The rename relies on `jobs.<job_id>.name` evaluating the `github` context (GitHub docs). Task 1.3 confirms it live before the branch lands, and records the "Waiting for status" state after ready.
- `converted_to_draft` is not a trigger, so an in-flight ready run finishes and posts; harmless, the next ready event reruns.
- Gitea `edited` also fires on body edits, so a ready Gitea PR reruns CI on a description edit. Accepted: Gitea is the secondary forge and the alternative is a title-only guard on an unverified `changes.title` field.

## Open Questions

None. D2 and D4 were ruled on 2026-10-28; task 1.3 is a precondition, not an open question.
