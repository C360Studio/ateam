# Change Proposal: `gemini-context-183`

Approved-by:

## Why

`ateam init` writes the top-level `"contextFileName": "AGENTS.md"` into `.gemini/settings.json`, and `ateam doctor` accepts it (#183). Released Gemini CLI does not read that key. In the 0.43.0 Homebrew bundle, the CLI sets its context file only from `settings.context?.fileName` (`gemini-YQATFAPB.js`) and `settings.merged.context.fileName` (`interactiveCli-MHCNTGQB.js`). The bundle has no settings migration from the flat key. So every adopter on current Gemini CLI loads `GEMINI.md`, not `AGENTS.md`, while doctor reports parity green.

## What Changes

- `ateam init` writes `"context": {"fileName": "AGENTS.md"}` and stops writing the flat key. This repository's `.gemini/settings.json` makes the same change.
- Doctor reads only the nested `context.fileName`. A file that sets only the flat key fails with a violation that names the nested key to set. The user-scope fallback (`ATEAM_HOOK_SCOPE=user`) and the project-wins precedence stay as they are, keyed on the nested key.
- Docs that name the flat key change to the nested key: `.agents/README.md`, the generated README table in `internal/bootstrap/templates.go`, `README.md`, `docs/adoption.md`, `docs/brownfield.md` (with a migration row for the old key), and `docs/mcp.md`.
- The harness spec gains one scenario for the Gemini context key under an existing requirement.

## Track

`quick`. The change modifies one existing requirement and adds none. It touches only `internal/bootstrap`, docs, and one config file. It adds no exported Go surface and touches no boot path, durability behavior, or gate file.

## Non-goals

- Accepting a list value for `context.fileName` (the existing `// shortcut:` stays).
- Rewriting an adopter's existing `.gemini/settings.json`. `init` keeps preserving existing client files; doctor's violation and the brownfield row tell the adopter what to change.
- Checking that a live Gemini session loads `AGENTS.md`. Doctor checks files, not client behavior.

## Definition of Done

- `task verify` is green on a clean tree, and `ateam doctor` parity is green.
- Tests in `internal/bootstrap` cover four cases: nested `AGENTS.md` passes, flat-only `AGENTS.md` fails and names `context.fileName`, nested `GEMINI.md` fails, and the user-scope fallback reads the nested key. `task mutate:check` shows that the flat-key rejection can fail.
- A freshly initialized temp repository passes `ateam doctor --check` and contains no `contextFileName`.
