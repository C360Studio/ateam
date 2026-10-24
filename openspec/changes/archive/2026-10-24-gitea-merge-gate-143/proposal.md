# Change Proposal: `gitea-merge-gate-143`

## Why

The merge check (`scripts/merge-check.sh`) and the gate-files check read GitHub only, through `gh`. A repository on Gitea had no up-to-date rule, no review check, no known-flake check, and no stop on gate-file changes (#143).

## What Changes

- The merge check becomes `ateam review gate` in `internal/review/` (`gate.go`); `scripts/merge-check.sh` is a wrapper that runs it, so `task merge:check` and both CI workflows keep their entrypoint. One unexported forge interface carries the reads; the judgement and its output lines are shared.
- GitHub reads keep the exact `gh` argv the script used, so the existing fake-`gh` suite is the parity oracle (`// shortcut:` until that suite moves to `httptest`).
- Gitea reads use the guarded forge client (`internal/review/forge.go`) against API v1. The up-to-date rule is branch protection whose status checks are on and list exactly `CI / Required (pull_request)` (owner ruling 1, no glob) with `block_on_outdated_branch`. The CI job uses only the job token Gitea Actions provides (owner ruling: adopters create no bot accounts or extra secrets). That token reads the branch (`protected`, `enable_status_check`, `status_check_contexts`) but not the protection rule: on a 403 or 404 for the rule the branch fields decide and the run prints `rule strict=unverified`; with a token that can read the rule (an admin's local run) `block_on_outdated_branch` is checked. Any other failed read (401, 5xx, wrong shape) exits 2. Lists are paginated and checked against `X-Total-Count`; the PR commit list is normalized oldest first and must end at head. Closing references come from each open `class:flake` issue's timeline (owner ruling 3). Every commit after the review record needs a new review until a live run shows Gitea's commit file lists are complete (owner ruling 5).
- Forge selection: `ATEAM_FORGE`; under Actions, `GITHUB_SERVER_URL` (Gitea Actions sets it) with `GITHUB_REPOSITORY`; otherwise `origin`; with none, GitHub through `gh`. The selection line names the forge and repository, never the host (owner ruling 6).
- Gate files on Gitea (owner ruling 4): `.gitea/workflows/` joins the gate paths; `.gitea/workflows/ci.yml` and `.gitea/workflows/gate-files.yml` are the GitHub workflows with only the credentials changed; `scripts/gate-files.sh` reads Gitea with `curl` when `ATEAM_FORGE=gitea`.
- The Close rule names Gitea's auto-merge call (`POST /repos/{owner}/{repo}/pulls/{n}/merge` with `{"do":"squash","merge_when_checks_succeed":true}`) beside `gh pr merge --squash --auto` in every copy.

## Non-goals

- The approval note of #142; no-bypass checks on either forge (owner ruling 2); GitHub Enterprise Server; moving GitHub reads off `gh`.
- Verifying `block_on_outdated_branch` from CI on Gitea: the job token cannot read it, and no extra secret is asked for; the Gitea runner setup is the owner's.
