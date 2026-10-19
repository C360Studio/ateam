# Gemini first, fixed-model harness experiments

## Why
The user selected Gemini Flash with API billing as the first live benchmark. The experimental variable is harness A versus harness B on the same model. Model ranking, routing and role assignment do not belong in this first benchmark. PR #13 is still unmerged; replace its first native provider instead of adding a multi-provider routing layer.

## What Changes
- Replace the unmerged Codex-specific native adapter with Gemini CLI authenticated only by GEMINI_API_KEY, pinned to gemini-3.8-flash and a verified CLI version.
- Preserve shared A/A validation, paired scheduling, grading, failure evidence, budget stops, isolation, immutable artifacts and offline rescore.
- Use native JSON output because installed Gemini CLI 0.43.0 stream-json omits reasoning counts. Retain exact native output and native per-model statistics, and stop on unexpected or unverified model usage.
- Make generated settings/arguments inspectable in dry-run provenance and keep both arms' model/tool execution settings identical. The initial variable is explicit harness instruction text; broader skill/hook treatments require future scoped support.

## Capabilities
### Modified Capabilities
- `benchmark-experiments`: Gemini native accounting and fixed-model harness treatment boundary.

## Impact
`internal/bench` native adapter/parser/tests/fixtures, shared runner/config, benchmark docs/examples, CLI self-test selection. No UI, init role defaults, model router, new dependency, other model adapter or live paid calls. The generic command-adapter workflow remains available.

implemented-by: https://github.com/C360Studio/ateam/pull/13
