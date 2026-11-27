# Design: pull-request-target-402

## Context

Opened in this session at `25187ea`:

- `internal/review/gate.go:286-318` — `kind()` switches on `GateOptions.Event`: `pull_request` (`:291`) is a pull-request run, `push` (`:293`) a push run, empty fails, and any other event (`:300-302`) exits 2 with "event %s has no rule here". This is the refusal #402 reports.
- `internal/review/gate.go:49` — the `Event` comment lists `pull_request or push`. `cmd/ateam/review.go:191` fills it from `GITHUB_EVENT_NAME`, read only when `GITHUB_ACTIONS` is true.
- `internal/review/gate.go:153-156` — the run knows its forge (`GitHub Actions` / `Gitea Actions`) before `kind()` runs.
- `internal/review/gate.go:329-361` — the awaiting probe skips `Merge check` unless a `reviewed-by:` line or a successful merge check stands at the head; `headPassed` is how it finds that pass (#359).
- `internal/review/gate_github.go:545-590` — GitHub `headPassed` reads check runs named `Merge check` at the **PR head sha** (`:549`).
- `internal/review/gate_gitea.go:438-461` with `gate.go:73` — Gitea `headPassed` matches the literal context `Merge check / Merge check (pull_request)`. Gitea puts the event in the context name, so a `pull_request_target` run's status would never match.
- `.github/workflows/merge-check.yml:6-10,44-45` — the workflow runs on `pull_request` and checks out `github.event.pull_request.base.sha`. Its comment gives the reason this repository stays there.
- `docs/adoption.md:135` — the `Gate files` row explains the #360 move to `pull_request` and that "review is the control". No row says when to use `pull_request_target`.
- `internal/bootstrap/templates.go:608` — the shipped reviewer note finds the rerun with `gh run list --branch <head-branch> --workflow merge-check.yml`.
- `internal/bootstrap/mergecheck_test.go:378-388` — the only event test, `push run`. No test covers an unknown event or `pull_request` under `GITHUB_ACTIONS=true`.

Unverified, from GitHub's event documentation as I remember it, not opened here: under `pull_request_target`, `GITHUB_SHA` is the base branch's last commit, and the run is listed under the base branch. If the check run for `Merge check` attaches to that sha rather than the PR head, two things break: the `main` ruleset may not count it on the PR, and `headPassed` (`gate_github.go:549`) never sees the pass. The reviewer's `gh run list --branch <head-branch>` would also find no run. I'm not sure which way GitHub behaves; D2 makes a live run decide.

## Goals / Non-Goals

**Goals:** `ateam review gate` and `--awaiting` accept `pull_request_target` on GitHub with the `pull_request` rule; the adoption guide says when to use each trigger.

**Non-goals:** this repository's triggers; `Gate files` on `pull_request_target`; Gitea support; an event override flag.

## Decisions

### D1 — Which forges accept the event

- **A (recommended): GitHub only.** One `case` arm in `kind()` on GitHub Actions; on Gitea Actions it exits 2 with "pull_request_target is not supported on Gitea: its status contexts carry the event, and the merge check reads `(pull_request)` contexts". Cost: ~10 lines plus tests. Forecloses nothing; Gitea can be added when an adopter asks.
- **B: both forges.** Derive Gitea's `headPassed` context and the default required contexts from the event. Cost: threading the event into `giteaSource` and the `ATEAM_REQUIRED_CHECKS` defaults, and a second set of Gitea fixtures. Blast radius covers every Gitea adopter's context names, for a request nobody has made.
- **Adopter seam:** a GitHub adopter changes `on: pull_request` to `on: pull_request_target` in its own merge-check workflow and keeps the base-sha checkout. Doing nothing leaves `pull_request` working as today.

### D2 — How the head-pass lookup is proven

- **A (recommended): live run before merge.** Run the built binary under `pull_request_target` in a private GitHub repository (the adopter's or a scratch one the owner approves) and record: the sha the `Merge check` check run attaches to, whether the `main` ruleset counts it on the PR, and `--awaiting` output after a pass. If the run attaches to the PR head, ship D1 A. If it does not, refuse the event on GitHub too, with that reason, and report back on #402 instead of shipping a weakened probe.
- **B: ship on fixtures alone.** Cheaper, but fixtures encode my assumption about where the check run lands. If it is wrong, the probe returns `awaiting=true` with a stale pass standing (the #359 hole), silently.
- **Adopter seam:** none until the live run settles it; the guide row is written from what it shows.

### D3 — Reviewer rerun note

- **A (recommended): docs only.** If the live run lists `pull_request_target` runs under the base branch, add one sentence to the adoption row: find the run by `--event pull_request_target` and `headSha`, not `--branch`. The shipped template (`templates.go:608`) is untouched.
- **B: change the shipped reviewer template.** It reaches every adopter's reviewer context for a trigger few use.

## Risks / Trade-offs

- `pull_request_target` runs with a write-capable token and secrets for fork PRs. The gate never checks out PR code; the guide row says the workflow must not either.
- Accepting a new event widens what counts as a pull-request run. An unknown event still exits 2, and a test pins it.

## Migration Plan

None for existing adopters: `pull_request` and `push` behave as before.

## Open Questions

1. **Owner ruling:** D1 — GitHub only (A) or both forges (B)? Recommendation A.
2. **Owner ruling:** D2 — which private repository runs the live check, and may I create a private scratch repository under the org for it? Recommendation: the adopter's repository if you can run it there; otherwise a scratch repository deleted after the run.
