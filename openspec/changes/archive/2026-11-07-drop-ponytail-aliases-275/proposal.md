# Change Proposal: `drop-ponytail-aliases-275`

Approved-by: Coby Leuschke (2026-11-07)

## Why

#270 (PR #274) collapsed Ponytail modes to `on`/`off` and kept `full`, `lite`, and `ultra` as aliases of `on` for one release, marked with a `// shortcut:` in `NormalizeMode`. That release has shipped; this change removes the aliases. It is the breaking change that makes v0.4.0 a minor release (#275).

## What Changes

- `NormalizeMode` (`internal/state/state.go`) accepts `on`, `""`, `off`, `stop`, `normal` only; `full`, `lite`, `ultra` return `(on, false)` like any unknown word, so `ateam state mode`, MCP, and `/api/mode` reject them. The `// shortcut:` is gone.
- `modeCommandRe` (`internal/hook/hook.go`) matches `/ponytail on|off` and `/mode on|off` only.
- A saved `.ateam/state.json` with a legacy level still loads as `on` through `Load`'s unknown-mode fallback.
- The harness spec's mode-switch scenario, `docs/hook-flow.md`, and the Ponytail skill drop the alias text.

## Track

`quick`: one scenario line in one capability, no new surface, no gate file.

## Non-goals

No change to `on`/`off` semantics, the JSON key, or the MCP and web action shapes.

## Definition of Done

- `TestNormalizeModeOnOffAliases` rejects the three words and still loads `{"mode":"ultra"}` as `on`; `TestDetectModeSwitchAndLadderPrompt` treats `/ponytail lite` like `/ponytail banana`; the web API test returns 400 for `ultra`. `task mutate:check` reports `detection` with `lite` re-added to `NormalizeMode`.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
