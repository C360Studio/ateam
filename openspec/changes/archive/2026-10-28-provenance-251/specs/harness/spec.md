## ADDED Requirements

### Requirement: Build provenance names the binary the hooks run

`ateam version`, `ateam doctor`, and the session-start hook SHALL label a binary installed with `go install <module>@<tag>` by its tag, never `unknown`. Doctor SHALL report the `PATH` binary the hooks run as `hook_binary` beside `disk_build`. In the ateam checkout, every stale-binary warning SHALL name the binary's path and the commands that replace it. Pickup SHALL report how many commits local `main` trails `origin/main`.

#### Scenario: Module install reads as its tag

- **WHEN** the binary carries main-module version `v0.3.3` and no stamped commit or `vcs.revision`
- **THEN** `ateam version` prints `ateam v0.3.3 (module)`, and the session-start hook in the ateam checkout names `v0.3.3` instead of saying the binary has no build provenance

#### Scenario: Hook binary differs from the checkout build

- **WHEN** doctor runs in the ateam checkout and the `ateam` on `PATH` reports a different build from `./bin/ateam`
- **THEN** `hook_binary` carries the `PATH` binary's path, version, and build, and a warning names both binaries and how to replace the `PATH` one

#### Scenario: Hook binary matches

- **WHEN** the `PATH` `ateam` and `./bin/ateam` report the same build
- **THEN** doctor emits no hook-binary warning

#### Scenario: Local main behind origin

- **WHEN** local `main` trails `origin/main` by 2 commits
- **THEN** pickup's `main` worktree line and the `[ateam]` summary show `behind=2`, and neither shows `behind` when the two are even
