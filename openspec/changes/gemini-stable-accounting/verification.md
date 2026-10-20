# Verification record

implemented-by: codex (versioned stable Gemini adapter and explicit best-effort usage reports)

## Behavioral tests

- **Decision:** Finite examples cover the version enum, legacy/stable settings and provenance, successful/failed/missing usage shapes, and retained-history accounting states. No new JSON grammar or execution-history algorithm is introduced; existing parsers/history validation are reused. Hand-calculated totals and literal recorded fixtures are independent oracles. Generated exploration was not added for these finite combinations.
- **Rule:** `Versioned Gemini best-effort accounting` in the canonical benchmark specification.
- **Inputs:** Omitted/explicit0.43/explicit0.64/unsupported versions; nonexistent command and missing key to prove refusal ordering; native version mismatch; current/legacy privacy settings; role-duplicated, cache/thinking/tool counters; empty/fatal/all-zero native records; zero snapshots with malformed/empty final source or terminal error; known/unknown failed/unpaired retained histories; measured-zero summary representation and unchanged legacy text/JSON.
- **Run:** Stable selection failed before implementation against the old pin. Two early negative tests initially lacked required native limits; those fixtures were corrected before validating refusal behavior. Installed-CLI missing-metadata and zero-failure regressions showed meaningful RED before their guards, then GREEN. Logs: `.enjoy-logs/gemini-stable-version-red.log`, `gemini-stable-zero-red.log`, `gemini-stable-zero-failure-red.log`, `gemini-stable-focused-green-final.log`, `accounting-red.log`, `accounting-green-final.log`. The native tests are fixture execution, not paid model runs.
- **Mutation:** Four non-destructive overlays produced **detection** with expected assertions, baseline/after passes and unchanged source: remove the legacy-execution guard; ignore retained version mismatch; promote ambiguous zeros; omit known usage from failed/unpaired attempts. Logs: `.enjoy-logs/stable-mutation-{version,provenance,zero}.log` and `accounting-mutation.log`.
- **Not covered:** Live authentication/capacity, backend settings, actual invoices, complete retry/failure accounting, Windows native launch and the combined unmerged PR32 output-budget/PR43 fatal-diagnostic features. Historical supported-schema artifacts remain replayable; a new version requires a fresh live cohort before calibration claims.

## Actual CLI contract

Official stable release and npm metadata were verified on 2026-10-20. CLI0.64.0 was installed only in ignored `.ateam/tools/gemini-0.64.0` with lifecycle scripts disabled. Version and fake-response runs used synthetic credentials, fresh isolated homes, disabled telemetry/privacy exports and a probe-only network guard; no provider calls were made. Actual requests confirmed the requested model, schema, LOW reasoning, 8192 allowance and four read-only tools. Exactly four fake calls reached the turn-limit failure. Native outputs and independent numeric expectations are documented in [fixture provenance](../../../internal/bench/testdata/gemini064/README.md).

The JSON fatal path still omits stats. Missing upstream metadata also becomes zero; new-version parsing conservatively marks such snapshots unknown. Positive aggregate counters can conceal missing per-request data, so all reports remain best effort. No telemetry workaround or relaxed stop policy was introduced. A normal production-path dry run resolved the isolated stable executable and explicit accounting policy with no API key.

## Repository gate repairs

Strict OpenSpec initially failed on pre-existing main changes: the AGY MODIFIED block omitted canonical lifecycle scenarios, and forge/herdr/context-default requirement paragraphs exceeded500 characters. This change copies the unchanged canonical scenarios into the existing delta and moves unchanged details into scenarios, keeping corresponding deltas aligned. No runtime behavior, rule meaning or unrelated requirement is removed. Final clean-tree gates and exact-head independent review are recorded in PR56.
