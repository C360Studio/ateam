# Design: `task-lease-284`

## Context

Opened in this session at `main` 11705c1 (v0.3.5). Decisions 1-5 of the [#284 ruling](https://github.com/C360Studio/ateam/issues/284#issuecomment-6603390041) are taken as given; this file records how they land and the one place the code disagrees with them.

- `internal/state/state.go:1580` `Load` fills `st.Writer` from `ReadWriterLease` (`:1369`) on both paths (`:1586`, `:1614`). Hook, CLI, MCP, and web all reach `BuildPickupSummary` (`:2754`) with a `State` and no root, so the file has to be read before the summary is built.
- `state.go:608-612`: `json:"-"` fields already carry per-run values on `State`, and stay out of `state.json` and `ateam state status`.
- `internal/hook/hook.go:142-145`: session-start builds the summary, then writes the writer lease unless `strings.Contains(ctxText, "WARNING: another writer may be active")`. Any text in the summary can suppress that write.
- `state.go:671-699` `isCallerActiveWriterLease`: liveness first (`:672-680`), then caller identity (`:681-698`).
- `state.go:495` `handoffPRRefRe` is the only parser of `Addresses`, so a new token passes through it. `:2867-2869` picks the current slot as `st.Handoffs[st.CurrentHandoffKey()]`.
- `cmd/ateam/main.go:129-166` `runState` dispatches subcommands; `stateUsage` is `:168-186`. No `writer` subcommand exists.
- `internal/web/server.go:280` puts the summary in `PickupText`, so the panel that already shows pickup text shows these lines too. No UI code changes.
- `.gitignore:2` already covers `.ateam/`. Precedent: `docs/adoption.md:92-100` (the hook path only reads env), `openspec/specs/harness/spec.md:335`.

## Goals / Non-Goals

Goals: the agent sees the task lease; a stale `aip-lease:` handoff is named; the daemon can read writer liveness without parsing text. Non-goals are in `proposal.md`.

## Decisions

**D1. Seam for the daemon.** It writes this file atomically (temp file plus rename) and refreshes it itself:

```json
{"id":"<opaque>","issue":284,"holder":"<name>","expires_at":"2026-11-03T12:00:00Z","source":"<name>"}
```

All five fields are required; unknown fields are ignored. If it writes nothing, ateam behaves as it does today, with no line and no warning. `expired` is display only: ateam never deletes the file.

**D2. What counts as malformed (trust boundary).** The file sits in a checkout an agent can edit, and its values enter agent context.

- A) Parse only. Cost: a `holder` holding a newline forges pickup lines, and a FIFO at the path hangs every hook.
- B) **Recommended:** `os.Stat` must report a regular file, read at most 64 KiB, and require each string non-empty, at most 256 bytes, with no control characters; `issue` > 0; `expires_at` RFC 3339. Anything else is the ruling's one warning line. Reasons are fixed strings that never echo file content. Cost: about 40 lines. Forecloses long free-text fields.

**D3. Where it is read.**

- A) **Recommended:** `Load` fills `st.TaskLease` and `st.TaskLeaseError` (`json:"-"`) beside `st.Writer`, and `BuildPickupSummary` prints them right after the writer block. Every pickup surface gets it from one place; the cost on the hook path is one `stat` when the file is absent.
- B) Append at the hook and CLI call sites only. Cost: MCP pickup diverges from CLI pickup. Rejected.

**D4. The writer-lease check stays independent of the file (ruling decision 3).** With D3 A, a `holder` of `WARNING: another writer may be active` would stop session-start from writing the writer lease.

- A) **Recommended:** match `"\nWARNING: another writer may be active"` at `hook.go:143`. The summary's first line is always `[ateam]`, and D2 B bars newlines from file values, so only the real warning matches. One line, plus a test holding that exact `holder`.
- B) Have `BuildPickupSummary` return the decision separately. Cost: changes a function with five callers.

**D5. Addresses token.** Regex `(?i)\baip-lease:([A-Za-z0-9_-]+)` over the current slot's `Addresses`; the first token is compared with `id` exactly. Line: `handoff lease <id> is not the current task lease <id2>`. Nothing on a match, with no token, or with no valid file. Legacy `LastHandoff` states are not read: they predate the token.

**D6. `ateam state writer --json`.** Prints the stored `WriterLease` fields plus `live` and `exited` (`PID > 0` and not alive), or `null`, exit 0. Without `--json` it exits 2 with usage: no text form was asked for. An unparsable `writer.json` exits 1 on stderr, since `null` would tell the daemon the checkout is free.

**D7. `live` (needs a ruling: Open Question 1).** The ruling says `live` uses `isCallerActiveWriterLease` with no caller session, "so a foreign live lease reads `true`". It does not: a probe at 11705c1 returned `false` for a foreign lease with a live PID, both with no session in the environment (`:694-696` compares the lease PID with the caller's) and with `ATEAM_SESSION_ID` set (`:686-688`).

- A) **Recommended:** extract `:672-680` as `writerLeaseLive(lease)` (agent present, PID alive when set, within the 10-minute TTL when `last_seen` is set), call it from `isCallerActiveWriterLease` with no behaviour change, and use it for `live`. This gives the ruling's stated outcome.
- B) Call the function as written. Cost: the daemon reads `false` for the very lease it needs to detect and assigns a task to an occupied checkout.

## Risks / Trade-offs

- PID liveness is per PID namespace: a daemon outside the sandbox that wrote the lease sees `exited: true` and `live: false` for a running writer. The daemon must run `ateam` in the same namespace as the sessions it asks about.
- PID reuse can keep `live` true until the TTL lapses. Same limit as the warning line today.
- `Stat` then read leaves a small window to swap in a FIFO; the hook would hang only if someone replaced the file in that window. Marked with a `// shortcut:`.
- The first-token rule misreads a handoff that names two leases; no adopter has asked for that.

## Migration Plan

None. Absent file and no `writer` call mean today's behaviour byte for byte.

## Open Questions

1. Resolved 2026-11-03: the owner approved D7 A, `live` as a liveness-only rule. Approval also accepts the D2 and D4 additions.
