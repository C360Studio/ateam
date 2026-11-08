# Change Proposal: `ateam-version-341`

Approved-by: Coby Leuschke (2026-11-08)

## Why

#333 made `.agents/ateam.json` the one committed home for team settings, and AGENTS.md §3 now requires it. #334 shipped the adopter release pin as a separate file, `.agents/ateam-version` (`internal/bootstrap/pin.go:13`), so that rule is broken on day one (#341). No adopter uses the file yet, which makes this the cheap moment to move it.

## What Changes

- Add an optional `ateam_version` key to `.agents/ateam.json`. `state.LoadSettings` validates it at load with the same release-tag check `PinStatus` uses today (`pin.go`, `isReleaseTag`). The check moves into `internal/state`, which bootstrap already imports.
- `PinStatus` reads the key first and falls back to `.agents/ateam-version`. When both exist and disagree, doctor warns and names the key as the winner. The doctor comparison itself is unchanged.
- README § *Install a release* (`README.md:98`) shows reading the pin without ateam installed: `jq -r .ateam_version .agents/ateam.json`. The file is documented as deprecated, to be dropped in a later release.
- `docs/adoption.md` § Team settings, the AGENTS.md §3 / rules-index key list, and both harness Requirements name the key.

## Track

`quick`, under the approved #333 design (owner, 2026-11-08). It adds one key to the D1 file and reuses the D1 parser. It needs no new Decision.

## Non-goals

- Removing `.agents/ateam-version` (a later release).
- Changing what doctor compares, or the release-tag grammar.
- Pinning other tools.

## Definition of Done

- With neither the key nor the file, doctor output is unchanged.
- The key alone, the file alone, and both agreeing produce today's pin result.
- When both exist and disagree, the key wins and doctor warns.
- A malformed key makes `LoadSettings` fail, and doctor `--check` fails.
- Tests use independent expected values. `ateam mutate` reports `detection` for the read order and the key validation.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green, and the change is archived before ready.
