# Change: Pass selected reviewer run in host launch prompts, default low-band candidate selection to low, and promote Gemini Carbon to frontier (#379, #389)

Approved-by: coby (2026-11-16)

## Why

1. `#379`: `.agents/hosts/enjoy.toml`, `.agents/hosts/t3code.toml`, and `.agents/hosts/herdr.toml` did not include `{model}` or `{effort}` in the reviewer prompt text, and `bandValue` in `internal/review/compare.go` picked the first entry of a provider's low band (`"none"` for Codex, `"minimal"` for Gemini) rather than `"low"`.
2. `#389`: Gemini Carbon (`gemini:gemini-4-carbon`) is the new frontier Gemini tier, while `gemini:gemini-4-argon` moves to `workhorse` in `.agents/review-policy.json` and `docs/adr/0002-model-tiers-and-effort-bands.md`.

## What Changes

1. Update `.agents/hosts/enjoy.toml`, `.agents/hosts/t3code.toml`, and `.agents/hosts/herdr.toml` so `[launch] tool` names `provider {agent}, model {model}, reasoning effort {effort}` in the reviewer prompt and instructs the reviewer to use them in `reviewed-by:`.
2. Update `bandValue` in `internal/review/compare.go` so `band == 0` (`low` band) selects `"low"` when present in the provider's low band while keeping `"none"` and `"minimal"` accepted by `ateam review compare`.
3. Place `gemini:gemini-4-carbon` in `frontier` and move `gemini:gemini-4-argon` to `workhorse` in `.agents/review-policy.json` and `docs/adr/0002-model-tiers-and-effort-bands.md`.

## Impact

- Affected specs: `openspec/specs/harness/spec.md`, `openspec/specs/model-review/spec.md`
- Affected code: `.agents/hosts/*.toml`, `.agents/review-policy.json`, `docs/adr/0002-model-tiers-and-effort-bands.md`, `internal/review/compare.go`, `internal/review/compare_test.go`, `internal/review/launch_test.go`, `cmd/ateam/reviewlaunch_test.go`
