# Change Proposal: `review-return-to-306`

Approved-by: Coby Leuschke (2026-11-07)

## Why

On #305 the Codex reviewer approved but could not notify the implementing Enjoy conversation: Enjoy puts no conversation id in the environment, the implementer ran `ateam review request` without `--return-to`, and the command printed an empty `return_to` and a launch line ending `--return-to  305` with no warning (#306).

## What Changes

- `ateam review request` exits 2 with one line naming `--return-to` when the host manifest's `launch` tool or command names `{return_to}` and none is known from the flag or the host's environment. `--dry-run` prints the same line as a warning and exits 0.
- `docs/adoption.md` states the refusal under the host table.
- The `Review request spec` Requirement in `openspec/specs/model-review/spec.md` gains one scenario.

## Track

`quick`: one scenario on an existing Requirement in one capability, no new exported surface, no gate file.

## Non-goals

- No change to `.agents/hosts/enjoy.toml` (a gate file) or to `ateam review notify`.
- No discovery of an Enjoy conversation id.

## Definition of Done

- `TestReviewRequest` covers command and tool manifests without `--return-to` (exit 2, nothing printed, nothing run), `--dry-run` (exit 0), a herdr pane id from the environment, and a manifest without the placeholder; `task mutate:check` reports `detection` with the check disabled and with the dry-run exception removed.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
