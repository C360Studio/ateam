# Verification record

implemented-by: codex (optional shared Gemini output allowance, legacy compatibility and pilot v2)

## Focused regression and mutation evidence

- Decision: finite examples cover one optional bounded integer. No new grammar, interacting dimensions or order-dependent history is introduced; no fuzz exploration was run.
- Rule: `Explicit Gemini output allowance` in the canonical benchmark specification.
- Inputs: omitted/null, explicit default 8192, boundaries 1/65536, actual pilot 32768, invalid 0/-1/65537/fraction/string; fresh/stale native settings; legacy settings fixture.
- Run: `GOCACHE=/private/tmp/ateam-bench-audit-cache go test ./internal/bench -run '^TestGemini(OutputAllowance|LegacySettingsReplay|SettingsPinExecutionPolicy|NativeSourceSchema)' -count=1`. RED before implementation rejected the new field or retained 8192; GREEN after implementation. Exact logs: `.enjoy-logs/output-budget-red.log`, `output-budget-green.log`.
- Mutation: forcing explicit output allowances back to 8192 produced **detection** in `TestGeminiOutputAllowance`; removing range validation produced **detection** in `TestGeminiOutputAllowanceRefusal`. Both used non-destructive Go overlays, expected assertions, baseline and after-run passes. Logs: `.enjoy-logs/output-budget-mutation-ignore.log`, `output-budget-mutation-validation.log`.
- Not covered: backend enforcement, finish reasons and resolved reasoning identity are unavailable from these offline tests. Full clean-tree evidence and independent exact-head review are recorded in PR #32 before native sampling.

Assertions check both override scopes and the actual workspace settings written by execution. Invalid integer cases also bypass the public parser and call resolution with a nonexistent executable, proving refusal before command lookup/key use. Fresh retained settings validate; stale allowances refuse execution before output-directory creation and fail replay validation even with a recomputed artifact digest. Omitted/null settings/serialization/digests retain historical defaults; explicit 8192 records explicit provenance while requesting the same settings. Dry-run budget-policy text tracks the effective allowance.

The v2 live configuration/protocol preserves v1 treatment bytes, uses a fresh cohort and includes prior 14623 known tokens in the combined $5 planning target. New model calls require the fresh canary gate; no performance claim follows from this feature alone.

## Live outcome and delivery gates

Clean-tree full verification and doctor passed at `8aa9c6d91e5472e80c508933956435e44311795d`, including offline benchmark self-tests, before building the live binary. Log: `.enjoy-logs/gemini-v2-preflight.log`. The next commit `daaa4fea7e51b96099c9db3c3f43035664d4dc33` only removed a specification EOF blank line; strict OpenSpec, diff checks and independent exact-head review passed before sampling. Historical v1 replay still reproduced its pre-feature comparison with original bytes unchanged.

The [v2 report](../../../benchmarks/experiments/gemini-instructions-v2/results.md) records one passing source and one native turn-limit failure with missing usage. Offline rescore preserved the original and identical comparison. This blocks A/A and A/B under the frozen protocol. Final report review and exact delivery-head verification are recorded in PR #32; the live evidence remains bound to the earlier binary, not a later reporting commit.

## Stable CLI reconciliation

PR #32 now includes main’s explicit 0.64.0 execution contract and PR #43’s optional fatal evidence. Focused allowance/stable/accounting/fatal tests pass (`.enjoy-logs/pr32-integration-focused.log`). Real v1 and v2 artifacts were rescored with the combined implementation: original SHA-256 bytes and comparison output remain unchanged; replay paths/hashes are retained in `.enjoy-logs/pr32-reconciled-historical-replay.json`. V3 dry-run settings show 32768 in both generation overrides, stable privacy opt-out and explicit best-effort accounting. The new protocol changes shared task guidance and CLI version together, keeping the candidate-only instruction treatment fixed. No live inference occurred before protocol review and clean-tree verification.
