# Design: `doctor-verify-target-415`

## Context

Opened in this session on `claude/415-doctor-verify-target` at `main` 25187ea. The failure evidence is in the #415 body (semsource at 70c42d1).

- `internal/bootstrap/bootstrap.go:446`: the only `task verify` check in doctor is a marker in the preflight skill's text (`requiredSkillMarkers`). Nothing reads a Taskfile.
- `internal/bootstrap/bootstrap.go:229-239`: `CheckParity` runs file and text parity only. Its callers include about 40 tests that init a temp repository with no Taskfile and expect zero violations (for example `internal/bootstrap/parity_test.go:42`, `settings_test.go:25`, `review_test.go:27`).
- `cmd/ateam/main.go:650-735`: `runDoctor` builds `violations` from `CheckParity` (`:656`), the MCP budget, hook latency and the pin, and `passed` is `len(violations) == 0` (`:709`). Warnings never change the verdict.
- `internal/taskrunner/taskrunner.go:22-39`: `ListTargets` requires the root's own `Taskfile.yml` (`:28-30`, so an ancestor's Taskfile is never read), asks `task --taskfile Taskfile.yml --list-all --json` with a 10 s timeout (`:42-50`), and falls back to the indentation scan of top-level `tasks:` (`:70-71`, already marked `// shortcut:`). It reads `task` and `desc` from the JSON (`:52-57`), not `aliases`. `ateam task`, the `ateam_task` MCP tool and the board use it (`cmd/ateam/main.go:579`, `internal/mcpserver/server.go:441`, `internal/web/server.go:283`), and `taskrunner.Run` accepts only a listed name, so `ateam task verify` already rejects an alias-only `verify`.
- `.agents/skills/preflight/SKILL.md:20,23` runs `task verify` or `ateam_task {"target": "verify"}`; `.agents/skills/review-pr/SKILL.md:22` tells the reviewer to "run `task verify` in the checkout" on every code PR.
- `docs/daily-use.md:3` and `docs/adoption.md:112-113` tell an adopter to add a `verify` task and then pass `doctor --check`; `docs/brownfield.md:144` says "keep your own `task verify` or CI entry point". `docs/adoption.md:57` shows an adopter CI step running `ateam doctor --check`.
- `cmd/ateam/pin_test.go:38-52` and `codextrust_test.go:47` compare doctor's violations relative to a baseline or by content, so a new violation on their temp repositories leaves them green.

## Goals / Non-Goals

Goal: a repository where `task verify` cannot run fails `ateam doctor --check` with a line naming the gap. Non-goals are in `proposal.md`.

## Decisions

**D1. Severity.**
- A) **Recommended:** a violation, both when `Taskfile.yml` is missing and when it has no `verify`. It enforces what the docs already require. Blast radius: every adopter whose CI runs `doctor --check` without the target goes red on upgrade; the message names the fix. It forecloses a repository that runs reviews with some other command, which the shipped skills do not support anyway.
- B) A warning. No adopter turns red, but warnings live in doctor's JSON and nobody reads them on CI; semsource would have stayed green and its reviews would still have broken.
- C) A violation when `Taskfile.yml` exists without `verify`, a warning when there is no Taskfile at all. It spares an adopter using a non-Task CI entry point (the brownfield wording), but that adopter's reviewers still run `task verify` and fail.
- Seam: an adopter who does nothing after upgrading and has the target sees no change. One without it sees one violation line in `doctor --check` until they add `verify` to `Taskfile.yml`.

**D2. How the target is found.**
- A) **Recommended:** `taskrunner.ListTargets`, the lookup `ateam task` uses. No new parser; Task resolves `includes:` (a flattened include lists as `verify`); without Task it falls back to the file scan. Cost: one `task` subprocess per doctor run, under its existing 10 s timeout, only when `Taskfile.yml` exists.
- B) The file scan only, with no subprocess. Faster and deterministic, but a `verify` from an include reads as missing: a false violation.
- Seam: a doctor verdict on a machine without Task can differ from one with it only for a `verify` that comes from an include, and then it errs toward the violation.

**D3. Aliases.**
- A) **Recommended:** a `verify` that exists only as an alias does not count. That matches `ateam task verify` and `ateam_task`, which reject it today (`taskrunner.Run`), so the preflight MCP path is broken in such a repository too.
- B) Read `aliases` from Task's JSON and count them. Lets `aliases: [verify]` pass doctor while `ateam task verify` still refuses it, unless `taskrunner` is widened as well, which is a second change.

**Placement (not an owner decision).** The check lives in `internal/bootstrap` and is called from `runDoctor`, not from `CheckParity`: parity is file and text only, spawns nothing, and its init-based tests would all need a Taskfile fixture.

## Risks / Trade-offs

- `ListTargets` names only `Taskfile.yml`; a repository using `Taskfile.yaml` or `taskfile.yml` gets the missing violation although `task verify` works. `ateam task` has the same limit today; widen both together if one is reported.
- A Taskfile that fails to parse under Task falls back to the scan, so a broken include can hide behind a top-level `verify`. Doctor checks the name, not that the task runs.
- On CI, doctor runs inside `task verify` here, so `task --list-all` runs nested; it reads the Taskfile and runs nothing.

## Migration Plan

Release note: "`ateam doctor --check` now fails when `Taskfile.yml` has no `verify` task; add one that runs your checks." No data migration.

## Open Questions

1. **Owner ruling:** violation in both cases (D1 A, recommended), warning (D1 B), or split (D1 C)?
2. **Owner ruling:** should an alias-only `verify` count (D3 B)? Recommended: no (D3 A); revisit with `taskrunner` if an adopter asks.
