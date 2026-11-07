# Design: `review-lease-314`

## Context

Opened in this session on `claude/review-lease-314` at `main` ba69ba1. The owner's rulings: #314 option 1 (requester-created detached checkout, `existing_worktree`), and #315 option B scoped on 2026-11-07 (the issue comment of that date). The scoping: hooks honor `ATEAM_ROLE=review`, herdr and terminal launches set it, T3 is covered by #314, and Enjoy and T3 naming runs are documented as limits.

- `internal/hook/hook.go:145`: session-start writes the lease when `sessionID != ""` and the rendered summary has no different-writer warning (the #287 coupling). `:159` before-prompt and `:277` turn-end write it unconditionally. These are the only `WriteWriterLease` callers (`internal/state/state.go:1455`). A reviewer that skipped only session-start would take the lease on its first prompt.
- `hook.go:89-91` `HandleWithAgent` passes `state.ResolveEnvSessionID()` into `handle` (`:100`). `Probe` (`:96-98`) already runs "as no session" so `doctor` never takes a lease (#204). This is the precedent for one parameter that `handle` honors.
- `.agents/hosts/t3code.toml:15`: the launch text names no `workspaceStrategy`, so T3 defaults to `root` (#314 evidence). `docs/adoption.md:166`: T3 runs Codex as one shared `codex app-server` with a per-thread cwd, so no per-thread env var reaches Codex's hooks. `:168`: T3's own `codex exec` naming runs fire the hooks in the thread's cwd.
- `.agents/hosts/herdr.toml:12`: `herdr pane split --pane {return_to}` opens the reviewer's pane in the implementer's cwd, so a herdr reviewer takes the implementer's worktree lease today. `herdr pane split` takes `--env KEY=VALUE` and `--cwd PATH` (`herdr pane split --help`, herdr installed at `/opt/homebrew/bin/herdr`, read in this session).
- `.agents/hosts/enjoy.toml:12`: `start_conversation` takes provider, model, reasoning, title, and text only, with no env and no cwd (tool schema read in this session). The reviewer starts in the project checkout.
- `.agents/skills/review-pr/SKILL.md:22`: the reviewer runs `git worktree add --detach .worktrees/<agent>-review-pr-<n> <head-sha>` itself. If the requester already created that checkout, this fails. `:35` removes it after posting.
- `internal/state/prune.go:107-112`: prune reads each worktree's lease for occupancy. #321 says the review-checkout branch skips that check; a reviewer's lease inside its own review checkout is the evidence that check would need.

## Goals / Non-Goals

Goals: no read-only reviewer replaces a writer's lease on hosts whose launch can say so, and the docs name the hosts where it cannot. Non-goals are in `proposal.md`.

## Decisions

**D1. Where the role is read.**
- A) **Recommended:** `HandleWithAgent` reads `os.Getenv("ATEAM_ROLE") == "review"` and passes a `readOnly bool` into `handle`. `handle` guards the three `WriteWriterLease` calls with `!readOnly`. `Probe` passes `false` and is unchanged. Cost: about 6 lines plus tests. Blast radius: `internal/hook` only; the CLI, the MCP server, and handoff keep their lease logic.
- B) Read it inside `state.WriteWriterLease`. One guard, but `ateam state handoff` and the MCP path would also skip it, and a state-package function would read process env. Rejected.
- C) Clear `sessionID` for a reviewer, as `Probe` does. This also drops the session id from the stale-MCP marker and handoff staleness, which is more than the ruling asks. Rejected.
- Seam: an adopter sets nothing, and nothing changes. A value other than `review` (including `Review`) is ignored, so there is no partial match.

**D2. T3 launch (#314).** The tool text becomes: run `git worktree add --detach .worktrees/{agent}-review-pr-{pr} {sha}`, then call `t3_thread_launch` with `workspaceStrategy {type: existing_worktree, …}` naming that checkout's absolute path, plus the current title, message, and modelSelection. It sets no `ATEAM_ROLE`, because the lease belongs in the review checkout (prune occupancy, #321).
- Alternative: #314 option 2 (`type: worktree, baseRef`), not chosen. It produces a `t3/<hex8>` branch that prune treats as a host claim.
- The field name for the path is unverified (Open Question 1).

**D3. herdr launch.** Add `--env ATEAM_ROLE=review` to the `herdr pane split` call. `--cwd` the review checkout is the other option. It would also need the requester to create the checkout, which makes herdr's launch match T3's. Not chosen: the ruling names the env var for herdr, and the reviewer's skill already creates the checkout.

**D4. review-pr skill.** Line 22 gains: "If your working directory is already that checkout at the head (a host launch created it), use it." This is a wording change only. `review-pr` is not a gate file today (#325 would make it one).

**D5. Terminal and limits.** `docs/adoption.md` gains three things: one line under the host table saying a terminal reviewer is started as `ATEAM_ROLE=review codex exec …`, the Enjoy limit, and the T3 naming-run limit with today's workaround (`/handoff` early).

## Risks / Trade-offs

- A herdr reviewer under D3 leaves no lease anywhere, including in its own review checkout. Prune therefore has no occupancy evidence for it, which matches today's #321 gap and does not widen it.
- `ATEAM_ROLE=review` exported in a writer's shell silences that writer's lease. Pickup's different-writer warning then cannot protect it. The variable is documented as set by review launches only.
- D2 depends on T3 accepting a detached HEAD as an `existing_worktree`. If it refuses, the fallback is option 2 or `ATEAM_ROLE` for Claude threads only, and that needs a new ruling.

## Migration Plan

None. The hooks change takes effect for new sessions once the binary is rebuilt (`build=` provenance). Manifests are read per request.

## Open Questions

1. **(verify, no ruling)** What is T3's `existing_worktree` path field, and does T3 accept a detached checkout? Checked in one live T3 Codex review before the gate label. If T3 refuses, this returns to the owner.
2. **(owner)** Should D1 A also guard the session-start stale-MCP marker for a reviewer? Recommended: no. The marker is per session and harmless.
