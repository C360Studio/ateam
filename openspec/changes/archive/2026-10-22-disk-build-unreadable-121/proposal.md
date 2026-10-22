# Change Proposal: `disk-build-unreadable-121`

## Why

When reading `./bin/ateam --version` failed (missing binary, timeout, non-zero exit, no `(build)` in the output), `PopulateMCPProvenance` recorded the running MCP build as the disk build. The pickup header then read "disk equals running", which is exactly the comparison the stale-binary warning exists to make (#121).

## What Changes

- The pickup header shows `build=absent` when there is no `./bin/ateam` and `build=unreadable` when its version cannot be read. Staleness is then judged against checkout `HEAD`, as before.
