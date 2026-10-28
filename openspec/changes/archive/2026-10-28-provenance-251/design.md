# Design: `provenance-251`

## Context

- `internal/state/state.go:25-52` `BuildRevision`: a stamped `BuildCommit`, else `vcs.revision`, else `unknown`. `go install module@tag` embeds `Main.Version` (read by `releaseVersion`, `:72-86`) and no `vcs.*` settings, so `BuildLabel` (`:55-61`) is `unknown` while `ateam version` prints `ateam v0.3.3 (unknown)`.
- `internal/bootstrap/bootstrap.go:960-966` `FormatStaleBinaryWarning` is shared by doctor and session-start. For `unknown`, it says `build with task build`; otherwise `run task build`. Neither names a path.
- `internal/hook/hook.go:346-359` `checkSessionBuildProvenance`: runs only in the ateam checkout (`bootstrap.IsAteamCheckout`, `bootstrap.go:984-999`, which line-scans `go.mod` to avoid a subprocess, #150). It compares the running binary's short revision with `HEAD` by string only, with no `git`. The running binary is the hook binary, so the hook already inspects the right binary. Only its label is wrong.
- `bootstrap.go:1006-1038` `BuildProvenanceStatus` / `checkBuildProvenance`: doctor's check of the running binary, using `git merge-base --is-ancestor` with a 2s timeout. Doctor is usually run as `./bin/ateam doctor` or `ateam doctor`, so "running" depends on how it was typed.
- `cmd/ateam/main.go:574-583`: `disk_build` comes from `bootstrap.DiskBuildStatus` → `state.ReadDiskBinaryBuild(root)` (`state.go:96`), which reads `./bin/ateam` only. No field names the `PATH` binary.
- Hook and MCP commands are the bare word `ateam`: `.claude/settings.json:9` and `.mcp.json`. `inspectHookConfigs` (`bootstrap.go:765+`) already parses every client's hook file for doctor parity.
- `internal/state/state.go:2400-2406` computes `DefaultAheadOfRemote` with `rev-list --count origin/main..main`. Behind-origin is never computed. `:2873` prints `ahead=` per worktree from `main...<head>`, so the `main` line is always `ahead=0`. `:2625` prints the `[ateam]` line from cached state, so the hook pays nothing for a count stored at sync.
- `README.md:75` documents both the symlink and `go install ...@<tag>`. This machine's `~/.local/bin/ateam` is a copy, not the symlink.

## Goals / Non-Goals

Goals: every provenance line names the binary it describes. A module install reads as its tag, not `unknown`. The advice can actually clear the warning. Pickup shows when local `main` is stale.

Non-goals: Windows `PATH`, `go build` binaries, hook latency with many worktrees, CI (#250).

## Decisions

### D1. Resolving a module tag to a commit for the behind-HEAD comparison

| Option | Cost | Blast radius | Forecloses |
| --- | --- | --- | --- |
| **A. Doctor resolves, hook labels only** (recommended) | Doctor: `git rev-parse <tag>^{commit}` (2s timeout) then the existing ancestry check. Hook: no subprocess. It prints `binary v0.3.3 (module install) is not built from HEAD 6e8d841; <D3 advice>` | `bootstrap.go:1013-1038`, `hook.go:346-359` | A precise "N commits behind" in the hook |
| B. Hook runs `git rev-parse` too | About 3–5ms on every session start, against the 10ms target that doctor already sees at 15ms | the same plus the hook budget | headroom for the next hook feature |
| C. Hook reads `.git/packed-refs` / `refs/tags/<tag>` | No subprocess. An annotated loose tag needs a zlib object read to peel, so it would be partial or need more code | new ref-parsing code in the hook | nothing, but it is the most code for one line |

Adopter seam: outside the ateam checkout nothing changes (#150). An adopter's `ateam version` prints `ateam v0.3.3 (module)`, not `(unknown)`, and their `go install` binary no longer reads as unprovenanced.

### D2. Which binary doctor reports as `hook_binary`

| Option | Cost | Blast radius | Forecloses |
| --- | --- | --- | --- |
| **A. `exec.LookPath(argv[0])` of the Claude hook command found by `inspectHookConfigs`, then `<path> version`** (recommended) | One lookup and one subprocess in doctor only | `bootstrap.go` near `DiskBuildStatus`; `main.go:582` new field | Per-client differences, which today all say bare `ateam` |
| B. `exec.LookPath("ateam")` with no config parsing | Smallest | the same | Reporting a client configured with an absolute path |
| C. Resolve each client (Claude, Codex, Gemini) separately | One field per client | larger output; one map field | nothing; most code |

The warning fires only in the ateam checkout, and only when `hook_binary.build` differs from `disk_build`, or when `hook_binary` is a module install. It names both paths. Doctor's running-MCP warning (`InspectRunningMCPProcesses`, `bootstrap.go:1399`) names the same path. Adopter seam: `hook_binary` is a new JSON key. Absent `PATH` `ateam` reports `{path: "", version: "absent"}` with no warning outside the checkout.

### D3. The advice text

| Option | Cost | Forecloses |
| --- | --- | --- |
| **A. Name the path and both fixes**: `task build && ln -sf "$PWD/bin/ateam" <path>` or `go install ./cmd/ateam` (recommended) | Longer line | — |
| B. Only `go install ./cmd/ateam` | Shortest | the README's symlink flow, which picks up rebuilds in place |

### D4. Behind-origin on pickup

Replace the `rev-list --count` at `state.go:2403` with `rev-list --left-right --count main...origin/main`. That is one git call, the same as today, and it fills a new `DefaultBehindRemote`. Print `behind=N` on the `main` worktree line when N > 0, and `| behind=N` on `[ateam]`. When `main` is dirty, `[ateam]` shows `dirty=true (uncommitted changes)`. No alternative is worth listing: the data is one flag away.

## Risks / Trade-offs

- D1 A leaves the hook unable to tell "a tag one commit back" from "a tag fifty back". Doctor resolves the tag and says whether it is behind `HEAD`, without a count (owner ruling D1 A: rev-parse, then the existing ancestry check).
- A pre-release tag (`v0.3.4-rc1`) is a tag, while a pseudo-version (`v0.3.3-0.2026...-abcdef123456`) embeds a 12-char revision. Treat the pseudo-version's suffix as the revision, which is more precise than the tag form.

## Migration Plan

None. The new JSON fields and the changed message text are additive. The `unknown` label stays for `go build` / `go run`.

## Open Questions

1. **Owner ruling:** D1 A (doctor-only resolution) or B (hook subprocess)?
2. **Owner ruling:** D2 A (Claude hook argv[0]) or B (bare `LookPath("ateam")`)?
3. Should the pseudo-version case (Risks, second bullet) be in this change or deferred? It is a few lines in `BuildRevision`. The recommendation is to include it.
