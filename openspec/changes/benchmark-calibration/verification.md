# Verification evidence

Issue #10; draft PR #13. Native fixture events are simulated, and no model call or paid calibration has been run. Installed CLI inspected: `codex-cli 0.160.1`. The 30-attempt template dry-run resolves that actual version and schedule without credentials or a model call; the model remains `MODEL_TO_SELECT`, which execution refuses.

## Independent plan/change review

Read-only reviewer identified partial-prefix comparison/rescore, failure preservation, native session identity, accounting ambiguity, output truncation, and nested credential redaction as acceptance boundaries. The latter two were confirmed by offline regressions and fixed: `io.Copy` cannot bypass the limited buffer; truncated native output invalidates accountable usage; nested JSON credential forms are redacted before parsing/persistence. Stop reasons must agree with the retained history. Final review and full clean-tree gate evidence are recorded in the PR after completion.

## Generated and boundary evidence

- Decision: native parser uses fuzz seeds/exploration plus exhaustive deterministic state histories; summaries and stop controls use deterministic boundary matrices.
- Rule: one thread/turn/terminal; missing and contradictory usage never becomes zero or complete; cache/reasoning subsets are not added; each arm starts independently; every outcome remains visible; only complete pairs compare; failure rescore never becomes successful execution.
- Inputs: six native fixture shapes, six fuzz seeds, all 24 four-event order permutations, duplicate keys/terminals, token count limits, source schemas; two arms × six outcomes × known/unknown usage, both orders, all incomplete prefixes of a four-attempt schedule; native token ceiling below/at/above, unknown usage, session reuse, timeout/cancel, output overflow before/after terminal, nested credential escaping, immutable partial rescore.
- Run: `GOCACHE=/private/tmp/ateam-bench-audit-cache go test ./internal/bench -run 'TestCodex|FuzzCodex|TestNative|TestExperimentMode|TestCalibration|TestDescribe' -count=1`; focused native and calibration race checks passed. Six seed replays and 57,538 exploratory fuzz executions in two seconds were recorded separately by the adapter worker. Full suite is a separate gate.
- Mutation: cache doublecount (`TestCodexRecordedEvents`), dropping failure usage (`TestCalibrationEveryOutcomeHasUsageDenominator`), removing token stop (`TestNativeBudgetStopsAndRescoresPartialHistory`), and reverting nested secret redaction (`TestCodexExecutionIsolationAndRedaction`) each produced **detection**, with baseline/expected assertion/after-run passed using `task mutate:check` overlays. An initial subtest selector was **inconclusive** because it selected no test; rerunning the full test produced detection. No repository files were restored/destructively reset.
- Not covered: live CLI/provider compatibility, provider billing reconciliation, monetary estimates, imported skill bundles, subscription auth, statistical equivalence, or generalized performance. Native descendant cleanup uses the existing process-group implementation and its independent PID-liveness test; adapter cancellation independently synchronizes process readiness.

## Live boundary

Implementation and pilot preparation are authorized. Live execution requires a selected model, explicit dollar cap/enforcement (or accepted in-flight overshoot policy), approved attempt/time limits, and credential/trace policy. Reported-token and time limits cannot guarantee a monetary cap. Use `benchmarks/calibration-report.md` after that decision; do not present simulated fixtures as a live calibration result.
