# Change Proposal: `hook-latency-153`

## Why

The real-repo session-start hook measures about 5.8ms internally against the 5ms target, and process start-up plus client plumbing add about 20ms to every invocation regardless. The budget is a regression tripwire for accidental `git` subprocess or forge calls on the hook path, not a user-facing latency promise (#153).

## What Changes

- The hook latency target moves from 5ms (`5000 us`) to 10ms (`10000 us`); `ateam doctor` warns above it. The `--check` ceiling stays at 25ms (`25000 us`).
- The rule that the hook path spawns no `git` and makes no network call is unchanged.
- README and `docs/adoption.md` stated a stale 50ms failure threshold; corrected to 25ms.

## Non-goals

- Caching worktree state reads in the pickup summary; a follow-up only if 10ms is approached again.
