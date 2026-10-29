# Change Proposal: `merge-check-workflow-255`

Approved-by: Coby Leuschke (2026-10-29)

## Why

`Merge check` answers "may this PR merge now": a review record at the head, no open `class:flake`, an up-to-date branch. Its inputs change on PR-state events (the reviewer editing the description, a flake issue closing), but the job runs only inside `CI` under the `Required` aggregate on push-shaped `pull_request` events (`.github/workflows/ci.yml:10-11`, `:62-105`). It is therefore always evaluated before its inputs exist. Every code PR shows one red `Merge check` between ready and review, cleared only by `gh run rerun --failed` (`.agents/skills/review-pr/SKILL.md:37-39`, `docs/daily-use.md:99`). One open flake issue turns every PR's `Required` red next to a green `Verify`; that was three of the four failed runs in the 40-run sample taken for #250 (#255).

## What Changes

- New `.github/workflows/merge-check.yml` and its Gitea twin carry the `Merge check` job (same base-commit checkout, script, token env, and #250/#257 draft guard and rename). They trigger on the PR events that change its inputs, including `edited`, so the reviewer's `reviewed-by:` description edit re-runs it.
- `ci.yml` (both forges) keeps `Verify` and `Required` with `needs: [verify]`.
- The `main` ruleset requires `Required`, `Merge check`, and `Gate files` (owner edit; Gitea branch protection gains the second context).
- `ateam review gate`'s rule check learns the second context per design Decision 3.
- `/review-pr` step 5, `docs/daily-use.md`, the protocol, the harness spec, and the workflow pin tests drop the manual rerun and describe the new trigger.

## Track

`design`: it touches gate files (`.github/workflows/`, `.gitea/workflows/`, `internal/review/`), changes the harness spec's merge-gate Requirement, and changes adopter-visible `ateam review gate` behavior. See `design.md`. Gate-file label `gate-change-approved` is already on #255.

## Non-goals

- No change to what `ateam review gate` judges, the review-record format, or review-after-ready sequencing.
- No `issue_comment` / `issues` trigger: those run on the default branch and cannot satisfy a required context at the PR head (#192).
- No fix for the skipped-job raw-name display (#259); it applies to the new workflow unchanged.

## Definition of Done

- Both forges carry `merge-check.yml`; `ci.yml` no longer carries `merge-check`; the ruleset lists the three contexts.
- A cross-agent review on a real PR clears `Merge check` with no manual rerun (recorded on the PR).
- `TestCIWorkflowPinned`, `TestGateFilesWorkflowRunsFromBase`, `TestGiteaWorkflowsTwinGitHub`, and a new pin for `merge-check.yml` pass, with `task mutate:check` detection evidence for the trigger list and the draft guard.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green on a clean tree.
