# Change Proposal: `provenance-251`

Approved-by: Coby (2026-10-28)

## Why

Hooks (`.claude/settings.json:9`) and MCP (`.mcp.json`) run the bare `ateam` on `PATH`, but doctor's `disk_build` and behind-HEAD check read only `./bin/ateam` (#251). A `go install github.com/c360studio/ateam/cmd/ateam@v0.3.3` (`README.md:75`) carries a module version and no `vcs.revision`, so `BuildRevision` (`internal/state/state.go:25-52`) returns `unknown`. The session-start hook then prints `binary has no build provenance; build with task build` (`internal/bootstrap/bootstrap.go:963`), and `task build` cannot clear it: it writes a binary the hooks never run. Separately, pickup's default-branch line shows `ahead=0` and never how far local `main` trails `origin/main`. In the 2026-10-28 dogfood pass it trailed by 9 commits.

## What Changes

1. **Module installs report their tag.** When there is no stamped commit and no `vcs.revision` but the main module version is a release tag, the build label is `<tag> (module)`, not `unknown`. Inside the ateam checkout the stale-binary warning names the tag and says how far it trails `HEAD` (`design.md` D1). Outside the checkout nothing prints, as today (#150).
2. **Doctor reports the binary the hooks run.** A new `hook_binary: {path, version, build}` field sits beside `disk_build`. When it differs from `./bin/ateam` in the ateam checkout, doctor warns and names both binaries (`design.md` D2).
3. **Advice names the binary to rebuild.** Stale-binary warnings print the resolved path and the two fixes: `task build && ln -sf "$PWD/bin/ateam" <path>`, or `go install ./cmd/ateam` (`design.md` D3).
4. **Pickup shows behind-origin.** The default-branch worktree line and the `[ateam]` summary show `behind=N` when local `main` trails `origin/main`. The `[ateam]` line says `uncommitted changes` when `main` is dirty.
5. Docs: `docs/brownfield.md` § 1 says a smoke pass qualifies a release candidate only when doctor shows `hook_binary`, `disk_build`, and `HEAD` at the candidate commit.

## Track

`design`. It changes the session-start hook output (a boot path) and adds a doctor JSON field that adopters read. See `design.md`.

## Non-goals

- `go build` binaries (`dev (unknown)`): #193 already points to `task build`.
- Windows `PATH` resolution, and per-client hook commands other than bare `ateam` (`design.md` D2).
- Hook latency with many worktrees (seen in the same pass; `ateam state prune`, #229, addresses it).
- No collision with #250 / PR #252: that change touches CI workflows and the merge-gate requirement. This one touches neither.

## Definition of Done

- `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
- Tests with a `detection` mutation each: a fake build info with a tagged module version and no vcs settings yields the tag label; doctor with a fake `PATH` `ateam` whose version differs from `./bin/ateam` warns, and equal versions do not; pickup on a temp repo whose `main` is 2 behind its origin prints `behind=2`.
- The session-start hook stays within its `<10ms` budget (`ateam doctor` `hook_latency_us`). Under D1 A the hook gains no subprocess.
