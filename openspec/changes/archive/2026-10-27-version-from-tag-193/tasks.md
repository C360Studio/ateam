## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, and rules on the open question (#193).
  Evidence: the owner approved in Enjoy on 2026-10-27 and chose Go's VCS stamping. Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 `state.Version` becomes an unset `var`. `ReleaseVersion()` resolves the ldflags value first, then `Main.Version` when it is not empty or `(devel)`, then `dev`. `VersionString` uses it (`internal/state/state.go:63-70`, #193). Acceptance: table test over the four inputs, with a mutation detected.
- [x] 2.2 MCP `initialize` reports `serverInfo.version` from `ReleaseVersion()` instead of `"0.1.0"` (`internal/mcpserver/server.go:196`, #193). Acceptance: test that `initialize` matches `VersionString`'s version, with a mutation detected.
- [x] 2.5 `task build` stamps `state.Version` from `git describe --tags --always --dirty` (`Taskfile.yml:131`, #193). Acceptance: a `task build` in a linked worktree prints its own version.
- [x] 2.3 `docs/brownfield.md` § 1 and the `README.md` install and command-table lines explain the printed version for a source build, `go install @tag`, and `go run` (#193). Acceptance: no remaining claim that a release PR bumps the version.
- [x] 2.4 Add the harness scenario, and have both tests cite the requirement (#193). Acceptance: `task spec:check` and `task test:spec` are green.

## Evidence

- 2.1, 2.2: `TestReleaseVersionResolution` (6 cases) and `TestVersionStringUsesStampedVersion` in `internal/state/version_test.go`, and `TestMCPInitializeReportsReleaseVersion` in `internal/mcpserver/server_test.go`. Five mutants under `/tmp/mut193`, checked with `ateam mutate -expect-text`, all gave `detection`: the stamp ignored, `(devel)` accepted, an empty fallback instead of `dev`, `VersionString` hard-coded to `v0.3.2`, and MCP back to `0.1.0`. The SHA-256 of `state.go` (`74993de92aca1ebf…`) and of `server.go` was the same before and after.
- Generated checks: not used. The resolver is a three-step precedence over one string and one optional struct, and the table covers each step and both fallbacks.
- 2.5: in a standalone clone, plain `go build` printed `v0.3.3-0.…-31ae8d6d3535`, then `v0.3.3-test` once tagged, then `v0.3.3-test+dirty` with a dirty tree. In this linked worktree, plain `go build` printed main's `…fd9d205be754 (fd9d205)`. The `task build` stamp printed `v0.3.2-21-gea6e38d-dirty (ea6e38d)`.
- 2.3: README install and command-table lines and `docs/brownfield.md` § 1 updated.
- 2.4: scenario "CLI and MCP report the release version from the build" under "Two-tool MCP context budget". All three tests cite that requirement.
