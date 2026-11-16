# Tasks: review-launch-and-carbon-379-389

## 1. Tasks

- [x] 1.1 Update `.agents/hosts/enjoy.toml`, `.agents/hosts/t3code.toml`, and `.agents/hosts/herdr.toml` to pass `{agent}`, `{model}`, and `{effort}` in the reviewer prompt.
- [x] 1.2 Prefer `"low"` in `bandValue` when selecting a `low`-band candidate in `internal/review/compare.go`.
- [x] 1.3 Place `gemini:gemini-4-carbon` in `frontier` and move `gemini:gemini-4-argon` to `workhorse` in `.agents/review-policy.json` and `docs/adr/0002-model-tiers-and-effort-bands.md`.
- [x] 1.4 Add unit tests and mutation verification for low-band candidate selection and shipped host manifests.
