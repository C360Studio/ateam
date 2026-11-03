# Design: `task-lease-284`

## Context

Opened in this session at `main` 11705c1 (v0.3.5). Decisions 1-4 of the [#284 ruling](https://github.com/C360Studio/ateam/issues/284#issuecomment-6603390041) are taken as given. Decision 5 is replaced: the owner approved a first draft on 2026-11-03, the adopter then asked for four writer states, host awareness, and version detection, and the owner directed this revision the same day.

- `internal/state/state.go:1580` `Load` fills `st.Writer` from `ReadWriterLease` (`:1369`) on both paths (`:1586`, `:1614`). Hook, CLI, MCP, and web all reach `BuildPickupSummary` (`:2754`) with a `State` and no root, so the file has to be read before the summary is built.
- `state.go:608-612`: `json:"-"` fields already carry per-run values on `State`, and stay out of `state.json` and `ateam state status`.
- `internal/hook/hook.go:142-145`: session-start builds the summary, then writes the writer lease unless `strings.Contains(ctxText, "WARNING: another writer may be active")`. Any text in the summary can suppress that write.
- `state.go:589-598` `WriterLease` holds agent, session, PID, `last_seen`, and herdr IDs: nothing says where the PID lives.
- `state.go:671-699` `isCallerActiveWriterLease` answers "is this lease mine?"; for a foreign lease with a live PID it returns `false` (probed at 11705c1). It is not the question the daemon asks.
- Three sites write the lease: `hook.go:145` session-start, only with no different-writer warning; `:159` before-prompt and `:277` turn-end, unconditionally. `last_seen` therefore does not advance during a turn.
- `cmd/ateam/main.go:69-71` `version` ignores its arguments. Today `ateam version --json` prints text and exits 0, and `ateam state writer` exits 1 like any failure.
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

**D6. `ateam state writer --json` shape (replaces ruling decision 5).** Always one object: `state`, `reason` (fixed text, empty for `none` and `live`), then the stored fields when a lease was read: `agent`, `session_id`, `pid`, `last_seen`, `herdr_pane_id`, `herdr_workspace_id`, `host`, `pid_ns`. Exit 0 for all four states, including an unreadable lease, so a non-zero exit always means the command itself failed. Exit 2 for bad usage; no text form.

- A) **Recommended:** the above. Cost: none to anyone, since the command is unreleased.
- B) Encode state in exit codes. Forecloses adding a state later, and collides with the old binary's exit 1.

**D7. How `state` is decided.** First match wins:

| Lease | `state` | `reason` |
| --- | --- | --- |
| no file | `none` | |
| unreadable or unparsable | `unknown` | `lease unreadable` |
| no agent or no `last_seen` | `unknown` | `lease incomplete` |
| no `host` (older ATeam) | `unknown` if within TTL, else `stale` | `no host recorded` / `past ttl` |
| other host or other `pid_ns` | `unknown` if within TTL, else `stale` | `other host` / `other pid namespace` / `past ttl` |
| no PID | `unknown` | `no pid recorded` |
| PID dead | `stale` | `pid exited` |
| PID alive, past TTL | `unknown` | `pid alive past ttl` (Open Question 1) |
| PID alive, within TTL | `live` | |

`isCallerActiveWriterLease` stays untouched, so the draft's `writerLeaseLive` extraction is dropped. A daemon treats `unknown` as occupied.

**D8. Host identity in the lease.**

- A) **Recommended:** `WriteWriterLease` records `host` (`os.Hostname`) and `pid_ns` (`os.Readlink("/proc/self/ns/pid")`, empty where that fails, so no build tags), both `omitempty`. Identity matches only when `host` is non-empty and both fields are equal. Cost: two cheap calls on three hook paths, measured against the 10ms target; a longer lease file. The hostname enters a gitignored file only.
- B) No identity; every PID result reads `unknown`. Cost: `live` could never be reported, which removes the daemon's reason to ask.

**D9. Detecting the command.**

- A) **Recommended:** `ateam version --json` prints `{"version","commit","capabilities":["state-writer-json","task-lease-file"]}`. Output that does not parse as JSON means an ATeam older than this, because old `version` ignores the flag and prints text. Plain `version` is unchanged.
- B) Parse the version string and compare. Cost: every client carries a version table, and dev builds have no comparable version.

**D10. Authority (docs only).** The check is advisory, and ATeam is not the final authority. The session-start guard is not atomic with the daemon's check, and before-prompt and turn-end overwrite a foreign lease anyway, so a second session takes the lease at its first prompt. Nothing blocks edits, and a "yield" happens only if an agent acts on the warning. `docs/adoption.md` says so and tells the daemon to give each task its own worktree, which is what isolates writers. Enforcing the lease would contradict ruling decision 3 and stays out.

## Risks / Trade-offs

- `last_seen` does not advance during a turn, so a writer on another host in a turn longer than 10 minutes reads `stale`. Only a heartbeat could fix that, which the ruling excludes; the D10 worktree rule is the protection.
- Off Linux there is no `pid_ns`, so two containers sharing a hostname compare as the same place and can read `live` or `stale` when the truth is `unknown`.
- `stale` across hosts assumes clocks roughly agree. A hostname change on the same machine reads `unknown` until the next lease write.
- PID reuse can read as `live` until the TTL lapses, then `unknown`, which errs toward occupied.
- `Stat` then read leaves a small window to swap in a FIFO; the hook would hang only if someone replaced the file in that window. Marked with a `// shortcut:`.
- The first-token rule misreads a handoff that names two leases; no adopter has asked for that.

## Migration Plan

Absent file means today's pickup output. Leases written by an older ATeam read `unknown` or `stale` until the next prompt rewrites them.

## Open Questions

1. Resolved 2026-11-03: the owner approved `unknown`. D7's "PID alive, past TTL" reads `unknown`. The earlier summary to the owner said past TTL is `stale` on any host. On the same host that is wrong for a long turn, because `last_seen` is frozen while the writer runs, and `stale` invites the daemon to assign into an occupied checkout. Cost of `unknown`: an idle session left open blocks assignment until it exits.
