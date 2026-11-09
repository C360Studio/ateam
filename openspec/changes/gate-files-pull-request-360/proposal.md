# Change Proposal: `gate-files-pull-request-360`

Approved-by: Coby Leuschke (2026-11-09)

## Why

GitHub's new default Actions event policy blocks `pull_request_target` in public repositories that set no explicit policy. It is in evaluate mode now and enforced from **December 2, 2026** ([Securely using pull_request_target](https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target#default-policy-for-pull_request_target)). The 2026-11-09 brownout already caused two `startup_failure` runs of the required `Gate files` check on #356 at `632e825`. To keep `pull_request_target`, every public adopter would need an admin event policy. The owner ruled on 2026-11-09 to move to `pull_request` and document the risk this adds (#360).

## What Changes

- `.github/workflows/gate-files.yml` and its Gitea twin trigger on `pull_request` instead of `pull_request_target`. The event types, the read-only permissions, the draft and WIP skip and rename, the concurrency group, and the job name `Gate files` all stay the same. The script now runs from the pull request's checkout and still reads PR state only through the API.
- The text that says the gate "runs from the base branch on `pull_request_target`" is rewritten. It appears in `AGENTS.md:90`, `.agents/rules-index.md:41`, `.agents/protocol.md:93`, `scripts/gate-files.sh:2-5`, both workflow headers, and two Scenarios of `Requirement: Merge gate and CI workflow enforcement` (`openspec/specs/harness/spec.md:478`, `:510-511`).
- The protocol and `docs/adoption.md` record two facts. First, the residual risk: a PR that edits the gate's own files is judged by its edited copy, so review of those PRs is the control. Second, the fail-closed property: the `main` ruleset still requires a check named `Gate files`, so a PR that deletes or renames the job stays blocked.
- `TestGateFilesWorkflowRunsFromBase` (`internal/bootstrap/gatefiles_test.go:173-213`) is renamed and its assertions are flipped. It now requires `pull_request:` and bans `pull_request_target`. It keeps the bans on checking out the head ref and on write permissions.

## Track

`design`: this touches gate files and changes a stated repo-wide rule and a spec Requirement's scenarios. See `design.md`. The PR needs the owner's `gate-change-approved` label and a comment naming its head.

## Non-goals

- A bot identity with CODEOWNERS-enforced approval (option B in #360; not chosen).
- A hard-coded guard in the script for the gate's own files (offered; not chosen).
- An Actions event policy that allows `pull_request_target`.
- Changing `Merge check`, which already runs on `pull_request` and builds from the base commit.

## Definition of Done

- On this PR's head, a `pull_request` run of `Gate files` fails without the label and passes with the label plus a `gate-change-approved: <head sha>` comment. The run URLs are recorded in `tasks.md`.
- On the flipped workflow test, `task mutate:check` reports `detection` when the trigger is restored to `pull_request_target`.
- `grep -rn pull_request_target` outside `openspec/changes/archive/` matches only the new risk text and the explanation in the `Merge check` header.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and a not-weaker review record is posted at the final head. The change is archived before the PR is marked ready, and the PR lands before 2026-12-02.
