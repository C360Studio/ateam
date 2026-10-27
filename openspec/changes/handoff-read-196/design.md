# Design: `handoff-read-196`

Drafted by claude (opus, medium) in the design skill; ADR 0003 sets this role's default at frontier tier, high effort, so the owner may want a second read.

## Context

- `internal/state/state.go:2693-2733` `writePickupHandoffs`: prints the current slot's five fields, then one `formatOtherHandoffLine` per other slot in `st.Handoffs`, sorted by key.
- `state.go:2800-2806` `formatOtherHandoffLine`: `other handoffs: <key> (<age>[, PR #n])`. No goal.
- `state.go:2849-2898` `worktreeHandoffLines` (#136): reads each linked worktree's `.ateam/state.json` directly (`CacheFilePath`, `json.Unmarshal`), skips slots `st` already holds, prints `<cleanTaskText(goal)> (<key>, <age>[, stale])`, and judges staleness with `handoffStaleReasonsAt(h, branch, wt.Head)`.
- `internal/state/openspec.go:270` `cleanTaskText`: normalizes whitespace and clips at 104 bytes on a word boundary with `…`.
- `state.go:1181-1214` `ReconcilePickup`: resolves identity, prunes stale slots, and writes the cache through `Update`. Pickup is therefore not a read-only path.
- `state.go:1149` `PruneStaleHandoffs`: stale same-checkout slots are deleted on every pickup, so after a pickup only linked worktree caches can still hold a stale slot.
- `state.go:1435` `Load`: reads and parses the cache; on a missing file returns defaults plus the lease, with no write.
- `state.go:450` `FormatFiveFieldPrompt`, `state.go:584` `HandoffSlotKey`, `state.go:881` `splitHandoffSlotKey`.
- `internal/hook/hook.go:131` session-start calls `BuildPickupSummary`; the preview is inside the 10ms hook budget.
- `internal/mcpserver/server.go:52-75` `ateam_state` schema (7-value `action` enum); `server.go:257-292` dispatch, which always resolves `CurrentAgent` and, for pickup, calls `ReconcilePickup`. `TestMCPTokenBudget` measures 212 of 300 tokens.
- `cmd/ateam/main.go:149` `pickup` and `main.go:230-246` `runStatePickup`; `main.go:143` `handoff`.
- `openspec/specs/harness/spec.md:189-192` fixes the `other handoffs:` line format; `docs/mcp.md:97-103` documents `pickup` and `handoff`.

## Goals / Non-Goals

Goals: an attributed goal preview per other slot; an explicit read of one slot's five fields from CLI and MCP that mutates nothing. Non-goals: in `proposal.md`.

## Decisions

### D1. Preview shape

- **A (recommended): suffix on the existing line**, `other handoffs: <key> (<age>[, PR #n]): <cleanTaskText(goal)>`. Attribution comes first, so a goal is never read as the caller's own. Cost: one format change and one spec line; at most 104 extra bytes per slot in hook context. Forecloses nothing.
- **B: an indented `goal:` line under each slot.** Same bytes plus a newline; reads like the worktree section but splits one slot over two lines that scripts grep.
- **C: all five fields inline.** Answers the issue without a read path, but every session start then injects every other agent's constraints and checks; rejected for hook context cost.
- **Adopter seam:** anyone parsing `other handoffs:` lines with an anchored regex on `)$` breaks under A or B. Doing nothing keeps the #196 gap: the goal is reachable only through a state dump.

### D2. Read surface

- **A (recommended): a new subcommand and action.** CLI `ateam state read-handoff <agent>@<branch>`; MCP `action:"read_handoff"` with a new `slot` string. Cost: about 15 schema tokens (212 to ~227), one dispatch branch that skips identity resolution and `ReconcilePickup`. Blast radius: `cmd/ateam`, `internal/mcpserver`, `internal/state`. Forecloses nothing; the name says what it does.
- **B: a `slot` argument on `pickup`.** No new enum value, but pickup writes the cache, prunes, and resolves identity; a read-only mode inside a mutating action is one missed branch from violating the issue's ownership rule.
- **C: CLI only.** Fails the MCP half of the acceptance.
- **Adopter seam:** a client that pins the `action` enum sees a new value; nothing existing changes behaviour. With nothing done, agents keep dumping state.

### D3. Where the read looks, and purity

- **A (recommended): `Load(root)`, then each linked worktree's cache in `git worktree list` order**, exactly as `worktreeHandoffLines` reads them; the current checkout wins a duplicate key. Live heads come from `ReadFastGitHead` (no subprocess) so staleness uses `handoffStaleReasonsAt`. No `Update`, `Refresh`, lease call, or forge call.
- **B: `Refresh(ctx, root, false)` first.** Fresher git data, but `Refresh` writes the cache and runs git subprocesses; the regression test would have to tolerate a cache write, which the issue forbids.
- **Adopter seam:** a slot pruned by an earlier pickup reads as missing; the missing output says so.

### D4. Stale and missing

- **A (recommended): stale prints the fields under `STALE — <reasons>`, exit 0; missing exits 1 (MCP tool error) and lists existing slot keys.** The reader asked by name, so withholding fields helps no one; the banner keeps it from looking current.
- **B: stale is refused like missing.** Simpler to reason about, but a stale worktree slot is often the only record of why a branch exists.

## Risks / Trade-offs

- Hook context grows by up to ~110 bytes per other slot; a checkout with many slots pays it on every session start. Mitigation: the existing prune keeps only live slots.
- The read sees the cache as of the last write; it does not sync a forge, so a PR closed since then is judged by cached cards. The banner shows the reasons used.

## Migration Plan

None. Additive output and a new action; no state schema change.

## Open Questions

1. **Owner ruling:** D4: print stale fields under a banner (A), or refuse (B)?
2. **Owner ruling:** D2 naming: `read-handoff` / `read_handoff` with `slot`, or another name?
3. Not blocking: should the HTTP API (`internal/web`) get the same read? Proposed out of scope until asked.
