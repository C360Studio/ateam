# Design: `gate-files-pull-request-360`

## Context

Inventory taken at `3812615`. Every line below was opened in this session.

- `.github/workflows/gate-files.yml:3-10` triggers on `pull_request_target` (types `opened, synchronize, reopened, ready_for_review, labeled, unlabeled`) and checks out the base branch with `persist-credentials: false`. Its permissions are `contents: read` and `pull-requests: read` (`:16-18`). The job at `:26-27` is named `Gate files` and is skipped and renamed on a draft except on `unlabeled` (#250). It runs `scripts/gate-files.sh "$PR_NUMBER"` with `GH_TOKEN: ${{ github.token }}`.
- `.gitea/workflows/gate-files.yml:3-11` is the same, plus `edited`, with WIP guards on `label_updated`/`label_cleared` (#257) and `ATEAM_FORGE=gitea`.
- `scripts/gate-files.sh:2-5` says in its header that it runs from the base branch. It reads files, labels and comments through the API only. The logic does not depend on the event.
- `.github/workflows/merge-check.yml:6-10` already runs on `pull_request` and says so in its header. It checks out `base.sha` to build. It is not touched here.
- The live `main` ruleset (`main-required-checks`, read with `gh api .../rules/branches/main`) requires the contexts `Required`, `Gate files` and `Merge check`. GitHub matches on the job's check name, not the event, so the name does not change.
- On Gitea a status context is `<workflow> / <job> (<event>)` (see `.gitea/workflows/merge-check.yml:103`: `CI / Required (pull_request)`). Moving the trigger renames the Gate files context from `… (pull_request_target)` to `… (pull_request)`.
- Rule text: `AGENTS.md:90`, `.agents/rules-index.md:41`, and `.agents/protocol.md:93` ("That check runs from the base branch on `pull_request_target`…"). In the spec, `openspec/specs/harness/spec.md:478` (Scenario *Gate-file changes wait for the owner*) and `:510-511` (Scenario *Draft pull requests defer CI…*) name the trigger. `README.md:283` and `docs/adoption.md:98` mention `Gate files` but not the trigger.
- `internal/bootstrap/templates.go` carries no gate-files text, and `ateam init` ships no gate-files workflow (grep at `3812615`). So the issue's "adopter templates" premise does not hold. Adopters copy the workflow by hand, and `docs/adoption.md` is where they learn about it.
- `internal/bootstrap/gatefiles_test.go:173-213` (`TestGateFilesWorkflowRunsFromBase`) requires `pull_request_target:` (`:183`) and bans `"\n  pull_request:\n"` (`:200`). Line `:251` pins the draft comment text.

## Goals / Non-Goals

Goal: `Gate files` keeps running in every public repository after 2026-12-02 with no admin event policy, and the rule text states the protection it gives. Non-goals: see `proposal.md`.

## Decisions

### D1. Trigger (owner-ruled 2026-11-09)

- **A (ruled): `pull_request`.** It is one line per workflow and needs no repository policy. The cost: a PR that edits `gate-files.sh` or `gate-files.yml` is judged by its edited copy, so a neutered copy passes its own check without the label. Review is then the only control on those PRs.
- **B: keep `pull_request_target` and set an allowing event policy.** This keeps the base-branch property, but it needs admin on every public adopter and stays one setting away from failing. Not chosen.
- **Adopter seam:** the GitHub check name stays `Gate files`, so no ruleset edit is needed. An adopter who copied the old workflow and does nothing gets `startup_failure` on that required check from 2026-12-02. Every PR then blocks (fail-closed), and `docs/adoption.md` says how to fix it.

### D2. Gitea parity (owner question OQ1)

- **A (recommended): move Gitea too.** Both twins keep the same shape, so `gatefiles_test.go` checks them with one rule. Cost: the Gitea required context is renamed, and branch protection's `status_check_contexts` must change from `Gate files / Gate files (pull_request_target)` to `… (pull_request)` in the same window. Until it does, every PR fails closed on a missing context.
- **B: keep `pull_request_target` on Gitea.** This keeps the base-branch property there and avoids a context change. But it forks the twins and the test, and no Gitea default policy is known either way (the issue's open note).

### D3. Fail-closed statement

The protocol and `docs/adoption.md` state the residual risk once, along with the property that limits it. The ruleset requires a check *named* `Gate files`, so a PR that deletes the workflow or renames the job reports no such check and cannot merge. A PR that keeps a job under that name can pass itself, either with a weakened script or by skipping the job, because GitHub counts a skipped job as passing. Codex's review of `a9fafff` pointed this out, and an earlier draft overstated it. The review contract already treats `scripts/` and workflows as code. No new mechanism is added (#360 non-goals).

### D4. Test

`TestGateFilesWorkflowRunsFromBase` becomes `TestGateFilesWorkflowTrigger`. It requires `"\n  pull_request:\n"`, bans `pull_request_target`, and keeps the bans on `pull_request.head`, `head_ref`, `ref:` and `write`. It also keeps the draft/WIP guard pins and the `merge-check.yml` base-checkout check. The header comment pinned at `:251` is updated with the workflow.

## Risks / Trade-offs

- **Self-judging** (D1, D3): accepted by the owner and documented.
- **Transition on this PR:** until it merges, the base's `pull_request_target` copy still runs on this PR alongside the new `pull_request` copy. That gives two `Gate files` check runs at one head, with the same script logic. A brownout during that window can fail the base copy; rerun it after the brownout. The deadline gives time for that.
- **Fork PRs:** under `pull_request` a fork gets a read-only token, which is all the script uses. A first-time contributor's run waits for approval, so the check is absent and the PR blocks (fail-closed).

## Migration Plan

1. Land this PR before 2026-12-02. On GitHub, nothing else changes.
2. The owner updates the Gitea branch protection context when the PR merges (OQ1 A), and the agent records it in `tasks.md`.

## Open Questions

- **OQ1 (owner, ruled 2026-11-09):** Gitea parity, A or B (D2)? Ruling: A, Gitea moves too. The owner updates the Gitea branch protection context to `Gate files / Gate files (pull_request)` when this merges.
