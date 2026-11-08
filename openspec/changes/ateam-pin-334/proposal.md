# Change Proposal: `ateam-pin-334`

Approved-by:

## Why

An adopting repository cannot declare which ateam release it pins, so `ateam doctor` has nothing to compare the binaries with (#334, split from #331). On semsource's machine the hooks resolved `~/.local/bin/ateam`, a dev build `v0.3.9-8-gbd6941f`, while the gates ran the pinned `v0.3.9 (module)`, and doctor passed. Outside the ateam checkout, `HookBinaryStatus` only reports (`internal/bootstrap/bootstrap.go:1147`).

## What Changes

- A repository may declare its pin in a tracked one-line file, `.agents/ateam-version` (for example `v0.3.9`). No file means no pin and no new doctor output.
- With a pin, `ateam doctor` reports `pin` and adds a violation when the running binary's release, or the release of the binary the Claude hook command resolves to (`hook_binary`), differs from it. A hook binary that does not resolve stays a report, not a violation.
- In the ateam checkout itself the pin is ignored; the existing build-provenance checks already cover it.
- README § *Install a release* and `docs/brownfield.md` §1 say how to declare the pin and point a Taskfile at the same file.
- `openspec/specs/harness/spec.md` gains one Requirement, `Doctor checks a declared ateam pin`.

## Track

`design`: adds exported surface an adopting repository reaches (the pin file and its format) and a doctor violation. See `design.md`.

## Non-goals

- No pin written by `ateam init`, no install or self-update command, and no release binaries.
- No check on the hook path: the session-start hook stays inside its 10 ms budget.
- No Codex or Gemini hook resolution in `hook_binary`. It keeps reading the Claude hook command (`claudeHookArgv0`, `:1163`).

## Definition of Done

- Doctor tests in a scratch adopter repository cover: no pin file (output unchanged); a pin matching both binaries (no violation); a pin differing from the running binary (violation); a pin differing from `hook_binary` (violation); a dev build `v0.3.9-8-g…` against pin `v0.3.9` (violation); an unresolved hook binary (no violation); a malformed pin file (violation naming the file); and the ateam checkout (pin ignored). `task mutate:check` reports `detection` for each comparison.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green. The change is archived before ready.
