# Proposal: pull-request-target-402
Approved-by:

## Track

`design`: touches a gate file (`internal/review/gate.go`), adds a harness Requirement, and changes what `ateam review gate` accepts in an adopting repository's CI.

## Decisions you are making by approving

- A private or internal GitHub repository can run the merge check from its base branch, so a pull request that edits the merge-check workflow is no longer judged by its own copy. The check applies the same rules it applies today; nothing else changes for a repository that keeps its current trigger.
- That trigger is accepted only once a live run in a private repository shows the check can see its own earlier pass on the pull request's latest commit. If it cannot, the waiting-for-review skip would leave a stale pass standing, so the trigger stays refused and the issue records why.
- On Gitea the new trigger stays refused, with a message saying why: the result names Gitea reports carry the trigger, and the check looks only for the current names.
- The adoption guide says which trigger to use: base-branch for private and internal repositories, the current one plus the gate-files check for public ones. It also says never to check out the pull request's code under the base-branch trigger.
- This repository is public and keeps its current triggers.

## Why

`ateam review gate` stops on `pull_request_target` with "event pull_request_target has no rule here" (#402). Under `pull_request` the workflow comes from the pull request, so a PR can rewrite the steps that produce `Merge check`. In a private repository GitHub's public-repository event policy (#360) does not apply, and `pull_request_target` closes that hole. The only workaround today is overriding `GITHUB_EVENT_NAME`, which the issue rules out.

## What Changes

1. `internal/review/gate.go`: `kind()` takes `pull_request_target` as a pull-request run on GitHub Actions and refuses it on Gitea Actions with a reason.
2. `docs/adoption.md`: a trigger row in the merge-gate table; the reviewer rerun note for runs listed under the base branch.
3. `openspec/specs/harness/spec.md`: one ADDED Requirement.

## Scope & Non-Goals

- **In-bounds:** the above, the event tests in `internal/bootstrap/mergecheck_test.go`, and the `GateOptions.Event` comment.
- **Non-goals:** switching this repository's workflows; `Gate files` on `pull_request_target`; Gitea support for the trigger; a `--event` flag; changing the shipped reviewer template beyond what the live run shows is wrong.

## Definition of Done

- Script-level tests: `pull_request_target` on GitHub Actions reads the PR and applies the review rule (pass and fail rows); on Gitea Actions it exits 2 naming the reason; an unknown event still exits 2. `task mutate:check` detects dropping the new case.
- A live `pull_request_target` run in a private repository, owner-provided or owner-approved, records where its `Merge check` run attaches and that `--awaiting` sees the pass at the head.
- The owner applies `gate-change-approved` at the head; `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green; the change is archived before ready.
