# Change Proposal: `commit-statuses-302`

Approved-by:

## Why

T3 Code, GitHub mobile, and Gitea all render a pull request's checks list, and T3 has no other slot for externally produced status (two source reads of T3 at `32a0531`, owner decision 2026-11-06: T3 is the human surface, ateam stays small). What ateam knows and no forge surface shows is the OpenSpec proposal's approval state, the handoff's freshness, and the review record at its sha. Publishing those three as informational commit statuses on the PR head makes every forge surface the console with no ateam UI (#302).

## What Changes

- A posting handoff (`ateam state handoff`, or the MCP `ateam_state` handoff) and `ateam state sync` for open PRs backing a local worktree, set three commit statuses on the PR head sha, each with a one-line description and a `target_url` to its evidence:
  - `ateam/proposal`: `success` approved by name and date, `pending` awaiting owner approval (`OPEN-Q`); a PR touching no OpenSpec change gets nothing or `error` (design Open Question 1).
  - `ateam/handoff`: `success` when the newest handoff's checkpoint equals the PR head, `pending` otherwise.
  - `ateam/review-record`: `success` on APPROVE, `failure` on REQUEST CHANGES, `pending` when the head moved past the reviewed sha, nothing when no record exists.
- `POST /repos/{owner}/{repo}/statuses/{sha}` with `WriteToken`: GitHub through `gh api`, Gitea over HTTP. Never a check run, never required.
- Idempotent: a status is re-posted only when its `state` or `description` differs from the latest one with that context on that sha. A failed write is one warning line and never fails handoff or sync.
- `ateam doctor` warns when status writes fail for lack of access, without echoing the token (design Open Question 3).
- `openspec/specs/harness/spec.md` gains one Requirement for the informational statuses.

## Track

`design`: a new forge write is a new spec Requirement. See `design.md`.

## Non-goals

- No `/runs` route, diff endpoint, polling, pairing, or QR; the desktop board is unchanged.
- No required check, no change to `task merge:check` or the `main` ruleset. The statuses describe; the gate decides.
- No status for CI, draft state, or mergeability: the forge already shows those.
- Never written from a hook. No T3-specific code (#303 verifies T3, #304 is the card). #273 and #181 unchanged.

## Definition of Done

- Tests with attached spec references for: the three contexts under approved, `OPEN-Q`, and no change; fresh and stale handoff; APPROVE, REQUEST CHANGES, stale, and absent record; idempotence; a write failure leaving handoff and sync exit codes unchanged; GitHub and Gitea fixtures.
- `task mutate:check` reports `detection` on the state chooser and the idempotence comparison.
- `ateam doctor` hook latency unchanged.
- Verified once by hand on this repository's draft PR: the three contexts visible in GitHub's checks list, recorded in `tasks.md`.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
