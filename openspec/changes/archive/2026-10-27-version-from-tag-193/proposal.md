# Change Proposal: `version-from-tag-193`

Approved-by: Coby Leuschke (2026-10-27)

## Why

`internal/state/state.go:65` hardcodes `const Version = "v0.3.2"`. Each release needs a one-line version-bump PR before the tag (#191, #148). With `cross_provider` set, that PR needs a cross-agent review record, a CI run, and a rebase onto a moving main (#193). Adopters also see two wrong versions (#193 comment, SemTeams v0.3.2):

- `ateam mcp` answers `initialize` with `serverInfo.version` `"0.1.0 (<build>)"` (`internal/mcpserver/server.go:196`).
- `go install …/cmd/ateam@v0.3.2` prints whatever the constant says, not the tag it was installed from.

## What Changes

- `state.Version` becomes a `var` with no default. A new `state.ReleaseVersion()` returns the first value found, in this order:
  1. A value stamped with `-ldflags -X`.
  2. Otherwise `debug.ReadBuildInfo().Main.Version`, when it is not empty or `(devel)`.
  3. Otherwise `dev`.
- `VersionString` uses `ReleaseVersion()` and keeps the `ateam <version> (<build>)` shape.
- MCP `initialize` reports `serverInfo.version` as `<ReleaseVersion()> (<build>)`.
- `task build` stamps `state.Version` from `git describe --tags --always --dirty` beside the commit it already stamps. A build of the `v0.3.3` tag prints `v0.3.3`, and a build of main between tags prints `v0.3.2-19-gfd9d205`. `go install …@v0.3.3` has no stamp and prints the tag Go embeds. `go run` and builds without VCS data print `dev`.
- `docs/brownfield.md` § 1 and the README install text explain the printed version for source builds.
- After this change, a release is a tag and `gh release create`, with no PR.
- The harness spec gains one scenario for the version that the CLI and MCP report.

## Track

`quick`. The change modifies one requirement, "Two-tool MCP context budget", and adds none. It touches `internal/state`, `internal/mcpserver`, `Taskfile.yml`, and docs. It touches no gate file, boot path, durability or ack behavior, and no surface that adopters import (both packages are `internal/`).

## Owner rulings (Enjoy, 2026-10-27)

1. Use Go's VCS stamping as the fallback instead of `-ldflags` only.
2. Go treats only a `.git` directory as a VCS root. A build in a linked worktree (`.git` file) therefore stamps the parent checkout's version, or none. The owner chose option A: `task build` also stamps `git describe`, the same way it already stamps the commit for the same reason.

## Non-goals

- Prebuilt release binaries or a release workflow.
- Changing how build provenance works: `BuildCommit`, `BuildLabel`, and the doctor and session-start staleness checks stay keyed on the commit.
- The `ateam v0.1.0` stub string in `internal/mcpserver/server_test.go:185`. It is a fake disk binary that the build parser reads, not a version claim.

## Definition of Done

- `task verify` is green on a clean tree, and `ateam doctor` parity is green.
- A table test of the `ReleaseVersion` resolution covers four inputs: a stamped value, a module tag, `(devel)` or empty, and no build info. A test shows that MCP `initialize` reports the same version as `VersionString`. `task mutate:check` shows that both tests can fail.
- A `task build` binary in a linked worktree prints its own `git describe` version, not the parent checkout's or `dev`.
