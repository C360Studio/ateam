# Design: `rereview-same-session-372`

## Context

Opened on 2026-11-11 at `59a6473`:

- `cmd/ateam/reviewlaunch.go:146` `reviewRequest` reads the PR body and head, computes `Eligible`, picks a run with `chooseReviewer` (`:248`), and prints the manifest's `[launch]` command or tool. Nothing in it reads an earlier record or a reviewer session.
- `cmd/ateam/reviewlaunch.go:55` `detectHost` returns the host and the caller's own session: `T3CODE_THREAD_ID` on T3 Code, `HERDR_PANE_ID` on herdr, and nothing on Enjoy, whose agent passes `--return-to`.
- `cmd/ateam/reviewlaunch.go:273` `notifyText(pr, sha)` is the whole wake message (design D6 of #295: no record text crosses sessions). `reviewNotify` (`:280`) sends it through `[notify]`, prints the tool on a tool-only host, and returns 0 silently on `cli` (`:307`, #358).
- `.agents/hosts/enjoy.toml:15` `[notify] tool` is `queue_message for conversation {return_to}`. `herdr.toml` `[notify] command = ["herdr","agent","prompt","{return_to}"]`. `t3code.toml` `[notify] command = ["t3","thread","send","{return_to}"]`. All three are already "send this fixed text to session X".
- `internal/state/state.go:588` the PR card caches `Review *review.RecordComment` (sha, verdict, agent, model, effort) from `ateam state sync`. Pickup prints it at `:3211-3239`.
- `.agents/skills/review-pr/SKILL.md:22` reviews the complete base-to-head diff in a detached checkout. `:35` removes that checkout once the record is posted and runs `notify`.
- `docs/daily-use.md:84` and `openspec/specs/model-review/spec.md:236` (*Review request spec*) and `:273` (*Review notify text*) describe a fresh review on every round.

## Goals / Non-Goals

Goals: after `REQUEST CHANGES`, the next `ateam review request` reaches the same reviewer session when one is known and still eligible, with a fresh launch as the fallback. Non-goals: a record of launched reviewers (#327), a gate or record-format change, and the `cli` notify gap (#358).

## Decisions

### D1. How the implementer learns the reviewer's session — **owner ruling**

- **A. The notify message carries it (recommended).** `notifyText` gains `; to re-review after fixes run ateam review request --reviewer-session <id> <n>`. The reviewer's id comes from `detectHost` in its own environment, or `--from` on Enjoy. It is validated as one line from `[A-Za-z0-9._:-]`, so the message stays fixed text. Cost: about 15 lines. Blast radius: `cmd/ateam` only, with no gate file. Forecloses nothing. The id lives in the implementer's context, and a new implementer session (after a handoff) does not know it, so it falls back to a fresh reviewer. That fallback is safe.
- **B. The record comment carries a `review-session:` line, cached by sync.** This survives implementer handoffs, and request can default without any flag. Cost: parser and cache changes in `internal/review` (a gate file) and `internal/state`. It also puts host session ids into a public forge comment.
- **C. Request records the launched reviewer in `.ateam/state.json`.** This is #327's scope. Tool-text hosts return the new id to the agent, not to ateam, so it would need a second command.

Adopter seam: under A, an adopter that never upgrades keeps today's behaviour, and a reviewer on an older binary sends today's message, which leads to a fresh launch.

### D2. When request re-routes

- **A. Request decides from the cache plus the flag (recommended).** With `--reviewer-session`, it re-routes only when the cached newest record for the PR is `REQUEST CHANGES` at a sha other than the head, and `chooseReviewer` accepts the record's run (agent, model, effort) against the current implementers. A fix commit can add a stronger implementer, so the reviewer may no longer pass. Any other case prints today's fresh launch with `rereview` set to the reason it fell back. `--fresh` skips the check.
- **B. A separate `ateam review rereview` subcommand.** This duplicates the policy, PR read, and eligibility plumbing, and the agent has to choose between two commands, which works against "by default".

### D3. Transport

- **A. Reuse `[notify]` with the reviewer as target (recommended).** The three manifests already send fixed text to a session id, so no gate file changes. The message is `PR #<n> moved to <head> after your REQUEST CHANGES at <sha>; check out <head> detached, review the full base-to-head diff and <sha>..<head>, rerun task verify, post a new record, then ateam review notify --record <file> --return-to <implementer> <n>`. When a notify command fails, as with a dead herdr pane or a closed T3 thread, request prints the fresh launch spec with `rereview: unreachable` and exits 0. On Enjoy (tool text), the printed tool adds: `if that conversation is gone, run ateam review request --fresh <n>`.
- **B. A new `[rereview]` table per manifest.** It is more flexible, but it edits three gate files for text that `[notify]` already expresses.

### D4. A reviewer whose context is too large — **owner ruling**

- **A. The reviewer declines (recommended).** The re-review message adds: `if your context is past your handoff threshold, run ateam review notify --decline <n>`. That sends the fixed text `reviewer declined the re-review of PR #<n>; run ateam review request --fresh <n>`. Cost: one flag. The reviewer is the only party that can see its own context size.
- **B. Cap the rounds.** Re-route at most twice, then go fresh. This is deterministic, but request sees only the newest cached record, so it would need to fetch every comment to count rounds.
- **C. No guard.** This conflicts with the hand-off-before-auto-compact rule.

## Risks / Trade-offs

- Anchoring: a returning reviewer may check only its own findings. The message and the skill paragraph both name the full base-to-head diff and `task verify`. A reviewer session cannot be forced to comply, as is true for a fresh one too.
- A wrong but reachable id (a mistyped session) wakes an unrelated session with fixed text. That is the same exposure `--return-to` has today.
- A stale cache means request has no record and launches fresh. The notify message already asks for `ateam state sync` first.

## Migration Plan

This is additive. Without `--reviewer-session`, the output is unchanged except for a new `rereview` field, which is omitted when empty. Update the `review-pr` skill (`:35`), `docs/daily-use.md:84`, and the two Requirements.

## Open Questions

1. **Owner:** D1. Is A acceptable, given it loses the reviewer id across implementer handoffs? Or should B ship the default as surviving handoffs, at the cost of a gate-file PR?
2. **Owner:** D4. Choose A (decline) or B (cap).
