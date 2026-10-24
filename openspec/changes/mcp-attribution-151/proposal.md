# Change Proposal: `mcp-attribution-151`

## Why

`ateam doctor` and the session-start hook warn about every running `ateam mcp` process on the machine as if it served the current repository. With two repositories open, each prints a false `restart the client` warning about the other's servers, with `build unknown` because the other repository's registry is not readable (#151). `state.LoadLiveMCPProcesses` falls back to a process scan that does not know which repository a process serves; the scan stays because it catches a server that predates its registry (#63, #67).

The hook's once-per-session `MCP server PID n (build x) is older than HEAD` line (`bootstrap.CheckClientMCPProcesses`) also compares the server's build against the current checkout's `HEAD`, so in a repository that is not ateam it is false for every server (#151 comment 6388138323).

## What Changes

- A live `ateam mcp` process belongs to this repository when this root's registry lists it, or when its working directory is this root or a linked worktree of it (`lsof -p` on macOS, `/proc/<pid>/cwd` on Linux). Doctor lists other processes in `running_mcp_processes` with `repo: other` and never warns about them; the hook never prints a line for them. A process whose working directory cannot be read keeps today's behaviour.
- The hook's `older than HEAD` comparison runs only where `bootstrap.IsAteamCheckout(root)` holds, the same gate #150 put on the build-provenance check.

## Non-goals

- Windows working-directory lookup; doctor does not support Windows today.
- The MCP-versus-disk build comparison (#116, #121).
