# Gemini instruction pilot v3

## Question and treatment

Does a small, task-applicable subset of ATeam engineering guidance change correctness, failures, time or tokens relative to minimal task guidance on the existing three Go fixtures?

`baseline.json` supplies `baseline.md`. `candidate.json` supplies the same text followed by `ateam.md`, joined by the existing bundle resolver. The six-point candidate-only ATeam text is byte-identical to v2. Both arms receive the same new self-contained-task sentence in the baseline. This v3 cohort follows the failed v1/v2 canaries; no samples are pooled. The candidate is an explicitly adapted subset of the Ponytail decision ladder and engineering-quality guidance in `AGENTS.md` and `.agents/skills/ponytail/SKILL.md` at repository base `ce3a351` (full base identity is recorded in git). It selects minimal complete changes, reuse, standard library, readability, trust-boundary validation and requirement checks. It omits repository workflow, delegation, mutation execution and reply-summary rules because this adapter returns one source file through a fixed schema and has no shell.

This measures this instruction subset, not the complete ATeam harness versus a default CLI. Hooks, native skills, routing and reviews remain disabled. Arm labels and source filenames are provenance only. Treatment content and analysis are frozen before inference; no editing after observing A/B results.

## Fixed controls and sample

Gemini CLI 0.64.0; requested `gemini-3.8-flash`, low reasoning, API-key-only authentication. Existing isolated workspace/home, managed-policy refusal, read-only tools, native structured output and independent grader remain unchanged. Backend-resolved model/reasoning remain unknown where the CLI does not expose them.

- `canary.json`: safe-path, one repetition, two identical baseline attempts; 100,000 known-token threshold, 600 seconds overall.
- `aa.json`: safe-path, sql-user and gitea-webhook, two repetitions, 12 identical baseline attempts / six pairs; 400,000 known-token threshold, 1,800 seconds overall.
- `ab.json`: same tasks/repetitions, 12 attempts / six pairs; minimal baseline versus ATeam subset; 400,000 known-token threshold, 1,800 seconds overall.
- Every attempt: 180 seconds, four turns and explicit shared `gemini.max_output_tokens: 32768` per call. This allowance accommodates thinking plus final output; it does not change the requested low reasoning setting. Seed 42 counterbalances ordering; no random-sampling claim. Canary data is excluded from A/A and A/B samples, which are also reported separately.

Two repetitions per task are a deliberately small pilot, chosen before outcomes to fit the combined budget. They cannot establish equivalence, significance or general productivity benefits. No adaptive sample expansion, model fallback or automatic retries between runs. Internal CLI retries may still occur inside an attempt.

## Advancement and stop rules

Before sampling, pass full repository verification, doctor, offline self-test and independent protocol review. Record exact commit, binary/configuration hashes and dry-run snapshots. Do not run verification workloads concurrently with inference sampling.

Canary acceptance: both attempts finish, pass independent grading, have distinct native sessions, expected reported model, CLI-reported normalized primary usage and no observed API-error or accounting gaps. This means the adapter’s required usage schema is present and nonzero; it does not prove complete provider accounting. A positive native aggregate can still hide a request whose upstream metadata was absent. Rescore offline and confirm identical grades/comparison with original bytes unchanged. Otherwise stop the live sequence.

A/A acceptance for proceeding: all 12 attempts and six pairs are present; no execution/grading infrastructure errors, timeouts, reused sessions, model mismatch or unknown usage; independent rescore reproduces outcomes and original bytes. Inspect every asymmetric quality outcome against source and grader diagnostics before advancing. Also check session uniqueness across phases. Record explanations as observations, not causal proof. If anomalies cannot be distinguished from a measurement defect, stop. Real quality failures are retained and do not by themselves invalidate the instrument. No low-variance or equivalence threshold is inferred from six pairs.

A/B proceeds only after these checks and a review of remaining budget. Report every status and grade; pairs with errors remain visible, and paired time uses only pairs where both passed. Show denominators for time/token comparisons and task-level outcomes. Report token subsets without double counting. Do not select an automatic winner or attribute a bundle-wide effect to a particular sentence. If quality saturates, report that the tasks did not discriminate quality.

## Amendment from v2

V2 retained one passing response followed by a stderr-only four-turn-limit failure with unknown usage. This fresh cohort uses the independently checked stable CLI 0.64.0 and best-effort accounting from PR #56, fatal diagnostics from PR #43, and the shared 32768 output allowance from PR #32. Both arms add: “This task is self-contained; no repository files are required, and the complete Go source belongs in the response’s source field.” This addresses observed searches for nonexistent repository files without increasing turns or changing tool permissions. It is a proposed setup improvement, not proof of the prior failure’s cause.

CLI version and shared guidance both change relative to v2, so cross-cohort improvement cannot be attributed to either individually. Within v3 only the unchanged six-point ATeam subset differs in A/B. The baseline is minimal task guidance, not untouched vanilla CLI. Preserve the failed cohorts and stop on another failed gate; no outcome-driven prompt repair or budget increase.

The optional output allowance accepts integers 1..65536; omission/null stays 8192 for historical settings. Both native override scopes must agree. Requested settings are not proof of resolved backend settings. The pinned CLI is installed locally under `.ateam/tools/gemini-0.64.0`; no global upgrade or trust change is needed.

## Budget and credentials

The owner authorized a **$5 combined target**, including v1/v2’s 35,413 known primary tokens and illustrative $0.05295675 plus unknown failed-attempt charges, not a hard monetary cap. V3 schedules at most 26 new attempts. Predeclared phase thresholds sum to 900,000 known primary tokens (100k+400k+400k). Maintain a cumulative 1,000,000 known-primary-token reserve including v1/v2. Before each launch, the effective phase threshold must be no greater than the smaller of its predeclared threshold and 1,000,000 minus all actual known prior usage. If a reduction is needed, retain the adjusted configuration with unchanged treatment/control fields. Do not start if no reserve remains. Do not enlarge thresholds or the sample after outcomes.

Within a phase the existing runner stops further attempts after unknown accounting or native execution/identity/session failure. Grading infrastructure errors do not currently stop the runner immediately: preserve them, inspect them at phase completion and prohibit the next phase. The final in-flight attempt may overshoot, and failed-call billing can be unknown. A monetary target is not a provider-side cap.

For planning only, [Google's published Standard Gemini 3.8 Flash prices](https://ai.google.dev/gemini-api/docs/pricing), checked 2026-10-20, are $0.75/M input and $3.75/M output including thinking, with $0.075/M cache input through December 31, 2026. Pricing the entire 1,000,000-token reserve at the output rate gives $3.75, leaving $1.25 target headroom for overshoot or unreported charges. The predeclared thresholds plus known v1/v2 usage total 935,413, leaving 64,587 within that token reserve. Neither calculation is an invoice or verified service-tier attribution. Inspect actual usage/cost projections at each phase boundary; stop if the remaining target no longer supports the next phase and reserve. No automatic native failure retry or model fallback.

Read only GEMINI_API_KEY from the owner's main-worktree .env into the launch process without copying or printing it, sourcing shell code, inheriting other credentials, or changing global trust. Pass it only to the existing native API adapter. Raw evidence, launcher and logs remain ignored/private; publish reviewed summaries and hashes only. A paid-tier report does not itself verify provider availability or billing.

The 900,000-token phase thresholds priced entirely at the output rate plus known historical priced usage give an illustrative $3.42795675 before overshoot and unknown charges. One in-flight attempt can exceed the 64,587-token arithmetic reserve: four requested 32768 output allowances alone exceed it, before input/retries. Review each completed phase before advancing; these calculations are conservative planning aids, not proven remaining balance or a hard cap.

## Reproduction and evidence

Build `bin/ateam` from the recorded commit. For each phase:

```sh
bin/ateam bench run benchmarks/experiments/gemini-instructions-v3/canary.json --dry-run
# After explicit authorization and setting GEMINI_API_KEY externally:
bin/ateam bench run benchmarks/experiments/gemini-instructions-v3/canary.json --output benchmarks/runs/gemini-instructions-v3-canary
bin/ateam bench compare <retained-run.json> --json
bin/ateam bench rescore <retained-run.json>
```

Repeat with aa/ab only after the gates above; any budget-adjusted local configuration and its digest must be retained alongside original predeclared configuration. Automatic evaluator self-tests run before each native launch. Preserve historical Gemini failures separately. V3 results will be recorded separately from [v1](../gemini-instructions-v1/results.md) and [v2](../gemini-instructions-v2/results.md).
