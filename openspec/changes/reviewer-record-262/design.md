# Design: `reviewer-record-262`

## Context

Opened in this session against `main` `d725f3c`.

- `.github/workflows/merge-check.yml:9-11` triggers on `pull_request` `[opened, synchronize, reopened, ready_for_review, edited]`. Lines `:16-18` queue one run per PR. Lines `:27-29` hold the `merge-check` job with the #250 guard: `name: ${{ github.event.pull_request.draft && 'Merge check (draft, skipped)' || 'Merge check' }}`, `if: ${{ !github.event.pull_request.draft }}`. Lines `:41-61` check out the base, build `ateam`, and run `scripts/merge-check.sh` with `ATEAM_REQUIRED_CHECKS`.
- `.gitea/workflows/merge-check.yml:9-64` is the same, except for the WIP comment and Gitea contexts. Gitea sets `draft` from `WIP:` (#257).
- `internal/review/gate.go:362-372`: the gate counts a description line only if it starts in column 1 with `reviewed-by:`, after dropping `\r`. It reports a missing one at `:431` (`no line starts reviewed-by:`).
- `.agents/contracts/reviewer.md:11-22` and template `internal/bootstrap/review.go:41-55` (`reviewerRecordPRComment`): § Record covers only the comment. The template reaches adopters through `review.go:143` / `templates.go:579`, and `CheckParity` flags any copy that differs from it (`review_test.go:106` `TestReviewSetupDrift`). The `review_record: aip` variant is `reviewrecord.go:59-63`.
- `.agents/skills/review-pr/SKILL.md:37-39` (template `internal/bootstrap/templates.go:512`): § 5 assumes the check "has failed". After `REQUEST CHANGES` it says to "leave it red", and the rerun it describes covers only an approval whose line already existed.
- `.agents/protocol.md:89` expects "one red `Merge check` between marking the PR ready and the record". `docs/daily-use.md:102` says the same.
- `openspec/specs/harness/spec.md:772-795`: Requirement "Merge check runs on pull-request state events" and its pin scenario. `:496` notes that GitHub does not evaluate a skipped job's `name:`, so a skipped run shows the raw expression text, which is still not a required name (#259).
- `internal/bootstrap/mergecheck_test.go:1015-1064` (`mergeCheckWorkflowViolations`): pins the exact name at `:1039` and the `if:` at `:1042`. Sensitivity rows are at `:1082-1110`. `draftName` is at `:881`.
- GitHub and Gitea reruns replay the saved event payload (`spec.md:499`), so a rerun sees the description as it was when that event fired.

## Goals / Non-Goals

Goals: no failed `Merge check` while the record is awaited. A contract-only reviewer leaves the gate correct after either verdict. Non-goals: see `proposal.md`.

## Decisions

### D1. How the workflow tells "awaiting" from "judgeable"

- **A (recommended). An expression on the event payload.** Use `if: ${{ !github.event.pull_request.draft && contains(github.event.pull_request.body, 'reviewed-by:') }}`. The name gains a middle arm, `... || !contains(github.event.pull_request.body, 'reviewed-by:') && 'Merge check (awaiting review record, skipped)' || 'Merge check'`. Cost: two lines per forge, and no new job. `contains` is looser than the gate: it matches anywhere, ignores case, and also matches inside fences. Looser only means the full gate runs and judges, which is today's behaviour. It can never skip a description the gate would accept, because every line starting with `reviewed-by:` also contains it. A null body counts as `''`, so the job is skipped. The body never reaches a shell.
- **B. A probe job** (the issue's wording). It reads the body through the API, `grep '^reviewed-by:'`, and `merge-check` runs on `needs: probe`. It matches exactly, and a rerun sees the current body. Cost: a second job and check row per run, a token read, a second job to pin, and `needs` semantics in both twins. A failed probe skips `merge-check` (fail-closed).
- **C. Docs only.** Keep the red and reword the promise. This contradicts the owner ruling.

Adopter seam: none. Adopters do not copy ateam's workflows (`docs/adoption.md:77`), and repos that run `ateam review gate` in their own CI keep today's red. With A, a rerun of an "awaiting" run stays skipped, because it replays the old payload. The description edit is what starts the full check, which `review-pr` § 5 already orders: record first, then edit.

### D2. Where the reviewer learns the description line and the refresh

- **A (recommended). In § Record** of the base reviewer contract template and this repo's copy. A short paragraph says three things. After `Verdict: APPROVE`, the gate needs `reviewed-by: <agent> (<model>, <effort>)` in the PR description. After either verdict, refresh `Merge check` and confirm its result at the reviewed head. If you lack permission to edit or rerun, report that instead of claiming the check changed. See `review-pr` § 5. Cost: drift on upgrade. Every adopter's `.agents/contracts/reviewer.md` fails `ateam doctor` parity until it is refreshed, as with any contract change.
- **B. Only in `review-pr` § 5** (status quo). This keeps the semteams#387 failure.

### D3. Refresh after a rejection that follows an approval at the same head

- **A (recommended). Rerun the whole latest `merge-check.yml` run for the head, and watch until it fails.** Use `gh run rerun "$run"` with no `--failed`, because `--failed` skips a run that passed. Await a run still in progress first. Bind it with `--branch <head>` and confirm `headSha` equals the reviewed SHA. In a repo that runs the gate inside `ci.yml`, rerun that job with `gh run rerun --job <job-id>`. On Gitea, use the Actions page.
- **B. The reviewer deletes its `reviewed-by:` line** so that `edited` skips the job. This is unsafe: the skipped run has a different name, so the earlier green `Merge check` at the same head still counts.
- **C. Push an empty commit.** This changes the reviewed head and voids the record. Rejected.

An initial rejection leaves no line, so the check stays skipped and the forge keeps blocking. The § 5 wording changes from "leave it red" to "leave it absent".

## Risks / Trade-offs

- Before the approving edit, the skipped check shows on GitHub as "Expected — waiting for status", and auto-merge waits. This is the draft behaviour from #250.
- A fix push after an approval still runs the full gate and fails until a new record exists. That red is a real finding (stale record), not noise.
- Gitea's `contains` and `pull_request.body` in the Actions payload have not been proven live yet (Migration step 3).

## Migration Plan

1. Land the workflows, contract, skill, protocol, docs, spec delta, and tests. The owner applies `gate-change-approved`.
2. On this PR: mark it ready and record the skipped run's ID. Post an approving record, then make the description edit, and record the passing run. If authorized (Q3), post a `REQUEST CHANGES` record at the same head, rerun, record the failure, and then post a fresh approval.
3. On the Gitea test instance, run a throwaway-PR probe for the awaiting skip and the edit run, then restore the test repo (as #255 did).

## Open Questions

Resolved by the owner on 2026-10-30: Q1 D1 A (payload expression), Q2 `Merge check (awaiting review record, skipped)`, Q3 yes, prove D3 live on #277.
