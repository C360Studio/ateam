# Change Proposal: `foreign-build-check-150`

## Why

The session-start hook and `ateam doctor` compare the binary's build commit with the current checkout's `HEAD`. In a repository that is not ateam, that `HEAD` is the adopter's commit, so every session prints `binary <build> is behind HEAD <head>; run task build` and doctor warns permanently, even though the binary was never built from that checkout (#150). In the ateam repo the check is useful and stays (#74, #121).

## What Changes

- The build-versus-`HEAD` check runs only when the checkout is the ateam source: the root `go.mod` declares `module github.com/C360Studio/ateam`, read with a line scan (no `go list`, no subprocess).
- Elsewhere, the hook skips the check silently, and doctor reports `build_provenance: not applicable (binary is not built from this checkout)` so its JSON stays explicit.

## Non-goals

- The comparison of the MCP server's build with the disk build (#116, #121), which compares the binary to itself.
- Cross-repo MCP process attribution (#151).
