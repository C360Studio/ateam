## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, and rules on the open question (#193).
  Evidence: the owner approved in Enjoy on 2026-10-27 and chose Go's VCS stamping. Claude wrote the header at their request.

## 2. Implementation

- [ ] 2.1 `state.Version` becomes an unset `var`. `ReleaseVersion()` resolves the ldflags value first, then `Main.Version` when it is not empty or `(devel)`, then `dev`. `VersionString` uses it (`internal/state/state.go:63-70`, #193). Acceptance: table test over the four inputs, with a mutation detected.
- [ ] 2.2 MCP `initialize` reports `serverInfo.version` from `ReleaseVersion()` instead of `"0.1.0"` (`internal/mcpserver/server.go:196`, #193). Acceptance: test that `initialize` matches `VersionString`'s version, with a mutation detected.
- [ ] 2.3 `docs/brownfield.md` § 1 and the `README.md` install and command-table lines explain the printed version for a source build, `go install @tag`, and `go run` (#193). Acceptance: no remaining claim that a release PR bumps the version.
- [ ] 2.4 Add the harness scenario, and have both tests cite the requirement (#193). Acceptance: `task spec:check` and `task test:spec` are green.
