# Design: `ateam-pin-334`

## Context

- `cmd/ateam/main.go:579` parses `--check` and ignores it. `:622` sets `passed` from `violations` alone, so any violation fails `ateam doctor` with or without `--check`. Warnings never fail it.
- `cmd/ateam/main.go:615` calls `bootstrap.HookBinaryStatus`. `internal/bootstrap/bootstrap.go:1133` resolves the Claude hook command's argv[0] (`claudeHookArgv0`, `:1163`, expanding `$CLAUDE_PROJECT_DIR` in `hookWords`, `:925`) and reads its `--version` (`state.ReadBinaryVersion`, `internal/state/state.go:154`). `:1147` returns without a finding outside the ateam checkout (`IsAteamCheckout`, `:1209`).
- `internal/state/state.go:106` `ReleaseVersion` gives the running binary's release: the stamped version, else the module version (`v0.3.9`), else `dev`. A `task build` after a tag prints `v0.3.9-8-gbd6941f`.
- `isAteamHook` (`bootstrap.go:973`) accepts any program ending in `/ateam`, so semsource's `scripts/ateam` wrapper counts. Doctor runs that wrapper's `--version`, which reports whichever binary the wrapper picked, and that can be a `PATH` fallback.
- semsource keeps its pin as `ATEAM_VERSION: v0.3.9` in `Taskfile.yml` and installs into the main checkout's `.tools/bin`. ateam reads no tracked config of its own: `.agents/review-policy.json` is a gate file, and `.ateam/` is gitignored.
- Spec: `openspec/specs/harness/spec.md:784` *Build provenance names the binary the hooks run* covers the ateam checkout only.

## Goals / Non-Goals

Goal: an adopter that declares a pin gets a failing doctor when the gates' binary or the hooks' binary is not that release. Non-goals are in `proposal.md`.

## Decisions

**D1 Where the pin lives.**
- **A (recommended): `.agents/ateam-version`, one line, for example `v0.3.9`.** Doctor reads it with `os.ReadFile`, with no subprocess and no new dependency. It works in every language, sits beside the other files ateam owns, and is not a gate file. A Taskfile can read the same file (`ATEAM_VERSION: {sh: cat .agents/ateam-version}`), so the pin has one source. Cost: one new file name that adopters must learn.
- B: read `ATEAM_VERSION` from `Taskfile.yml`. That would need YAML parsing (there is no YAML dependency in `go.mod`), it breaks on templated vars, and the Taskfile belongs to the adopter, not to ateam (`init` does not write one).
- C: go.mod `tool github.com/c360studio/ateam/cmd/ateam`. Go-only, and it pulls ateam's dependencies into the adopter's `go.sum`. Running hooks through `go tool` also costs more than the 10 ms hook budget. This option rules out non-Go adopters.
- Adopter seam: a repo that adds nothing sees no change. A repo that adds the file gets a doctor violation the first time a binary drifts.

**D2 What counts as a match.**
- **A (recommended): exact string equality after trimming whitespace.** `v0.3.9-8-gbd6941f` does not match `v0.3.9`, and that is the semsource case. `dev` and `unknown` never match.
- B: accept any `git describe` descendant of the tag. That would hide the exact drift this change exists to catch.
- A malformed file (empty, or not `v<major>.<minor>.<patch>` with an optional pre-release suffix) is a violation that names the file, so a typo cannot pass silently. The check is a small hand-written parse, because `golang.org/x/mod` is not a dependency.

**D3 How a mismatch is reported.**
- **A (recommended): a violation.** It fails `ateam doctor`, and therefore any CI step that runs doctor. The issue's complaint is that doctor passed.
- B: a warning, visible in JSON but never failing. Cheaper for adopters mid-upgrade, but it repeats today's silent pass.
- Adopter seam: bumping the release means editing the file and reinstalling in the same change. A CI runner that installs `go install …@$(cat .agents/ateam-version)` stays green by construction.

**D4 When the hook binary does not resolve** (`Version: "absent"`, `:1140`): report it with no violation. CI runners usually have no hooks or `PATH` install, and making that fail would block every adopter's CI. A hook binary whose `--version` cannot be read (`unreadable`) is a violation, because that binary does run.

**D5 Inside the ateam checkout,** the pin file is ignored. `checkBuildProvenance` (`:1240`) already compares against `HEAD`, which is stricter than a tag.

## Risks / Trade-offs

- Doctor shows only the Claude hook binary. A Codex or Gemini hook pointed at another binary can still drift. This is stated in the docs, not fixed (a non-goal).
- A wrapper that falls back to `PATH` (semsource `scripts/ateam`) now fails doctor on a machine without the pinned install. That is the intended signal, but it is new red for those developers.
- The README example `"$CLAUDE_PROJECT_DIR"/.tools/bin/ateam` resolves inside a linked worktree, where the install usually isn't. semsource's wrapper reads the main checkout instead (Open Question 2).

## Migration Plan

None for repos without the file. semsource: add `.agents/ateam-version`, point `ATEAM_VERSION` at it, and run doctor once.

## Open Questions

1. **Ruled 2026-11-08: `.agents/ateam-version`.** The file name and place. `.agents/ateam-version` is recommended. The alternative is a root `.ateam-version`, which is more visible but adds a dotfile at the top level.
2. **Ruled 2026-11-08: fold it into task 3.1.** Fold a README fix for linked worktrees into this change's docs task (recommended: a wrapper like semsource's, or an absolute path to the main checkout's `.tools/bin`), or file it separately.
3. **Ruled 2026-11-08: a violation.** A violation (D3 A, recommended) or a warning (D3 B).
