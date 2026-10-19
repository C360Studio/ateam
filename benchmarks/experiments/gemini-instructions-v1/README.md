# Gemini instruction pilot v1

## Question and treatment

Does a small, task-applicable subset of ATeam engineering guidance change correctness, failures, time or tokens relative to minimal task guidance on the existing three Go fixtures?

`baseline.json` supplies `baseline.md`. `candidate.json` supplies the same text followed by `ateam.md`, joined by the existing bundle resolver. The candidate is an explicitly adapted subset of the Ponytail decision ladder and engineering-quality guidance in `AGENTS.md` and `.agents/skills/ponytail/SKILL.md` at repository base `ce3a351` (full base identity is recorded in git). It selects minimal complete changes, reuse, standard library, readability, trust-boundary validation and requirement checks. It omits repository workflow, delegation, mutation execution and reply-summary rules because this adapter returns one source file through a fixed schema and has no shell.

This measures this instruction subset, not the complete ATeam harness versus a default CLI. Hooks, native skills, routing and reviews remain disabled. Arm labels and source filenames are provenance only. Treatment content and analysis are frozen before inference; no editing after observing A/B results.

## Fixed controls and sample

Gemini CLI 0.43.0; requested `gemini-3.8-flash`, low reasoning, API-key-only authentication. Existing isolated workspace/home, managed-policy refusal, read-only tools, native structured output and independent grader remain unchanged. Backend-resolved model/reasoning remain unknown where the CLI does not expose them.

- `canary.json`: safe-path, one repetition, two identical baseline attempts; 40,000 known-token threshold, 300 seconds overall.
- `aa.json`: safe-path, sql-user and gitea-webhook, two repetitions, 12 identical baseline attempts / six pairs; 180,000 known-token threshold, 900 seconds overall.
- `ab.json`: same tasks/repetitions, 12 attempts / six pairs; minimal baseline versus ATeam subset; 180,000 known-token threshold, 900 seconds overall.
- Every attempt: 120 seconds, existing four-turn / 8,192-output-token-per-call settings. Seed 42 counterbalances ordering; no random-sampling claim. Canary data is excluded from A/A and A/B samples, which are also reported separately.

Two repetitions per task are a deliberately small pilot, chosen before outcomes to fit the combined budget. They cannot establish equivalence, significance or general productivity benefits. No adaptive sample expansion, model fallback or automatic retries between runs. Internal CLI retries may still occur inside an attempt.

## Advancement and stop rules

Before sampling, pass full repository verification, doctor, offline self-test and independent protocol review. Record exact commit, binary/configuration hashes and dry-run snapshots. Do not run verification workloads concurrently with inference sampling.

Canary acceptance: both attempts finish, pass independent grading, have distinct native sessions, expected reported model, complete normalized primary usage and no API-error/incomplete-accounting evidence. Rescore offline and confirm identical grades/comparison with original bytes unchanged. Otherwise stop the live sequence.

A/A acceptance for proceeding: all 12 attempts and six pairs are present; no execution/grading infrastructure errors, timeouts, reused sessions, model mismatch or unknown usage; independent rescore reproduces outcomes and original bytes. Inspect every asymmetric quality outcome against source and grader diagnostics before advancing. Record explanations as observations, not causal proof. If anomalies cannot be distinguished from a measurement defect, stop. Real quality failures are retained and do not by themselves invalidate the instrument. No low-variance or equivalence threshold is inferred from six pairs.

A/B proceeds only after these checks and a review of remaining budget. Report every status and grade; pairs with errors remain visible, and paired time uses only pairs where both passed. Show denominators for time/token comparisons and task-level outcomes. Report token subsets without double counting. Do not select an automatic winner or attribute a bundle-wide effect to a particular sentence. If quality saturates, report that the tasks did not discriminate quality.

## Budget and credentials

The owner authorized a $2 target with token/time bounds, not a hard monetary cap. This sequence has at most 26 scheduled attempts and sums to 400,000 known primary-token stop thresholds. Before each later launch subtract actual known usage from that combined 400,000-token reserve, lowering its threshold if earlier overshoot consumed its allocation. No new call after unknown accounting or operational failure. The final in-flight attempt may overshoot; missing failed-call billing is not zero.

For planning only, [Google's published Standard Gemini 3.8 Flash prices](https://ai.google.dev/gemini-api/docs/pricing), checked 2026-10-19, are $0.75/M input and $3.75/M output including thinking, with $0.075/M cache input through December 31, 2026. Pricing all 400,000 planned tokens at the output rate is $1.50 before in-flight overshoot or unreported charges. These rates are not an invoice or verified service-tier attribution. Record the actual counts and rate basis separately from observed usage. Stop or narrow the remaining sample before starting if projected cost leaves insufficient reserve; do not silently enlarge limits.

Read only GEMINI_API_KEY from the owner's main-worktree .env into the launch process without copying or printing it, sourcing shell code, inheriting other credentials, or changing global trust. Pass it only to the existing native API adapter. Raw evidence, launcher and logs remain ignored/private; publish reviewed summaries and hashes only. A paid-tier report does not itself verify provider availability or billing.

## Reproduction and evidence

Build `bin/ateam` from the recorded commit. For each phase:

```sh
bin/ateam bench run benchmarks/experiments/gemini-instructions-v1/canary.json --dry-run
# After explicit authorization and setting GEMINI_API_KEY externally:
bin/ateam bench run benchmarks/experiments/gemini-instructions-v1/canary.json --output benchmarks/runs/gemini-instructions-v1-canary
bin/ateam bench compare <retained-run.json> --json
bin/ateam bench rescore <retained-run.json>
```

Repeat with aa/ab only after the gates above; any budget-adjusted local configuration and its digest must be retained alongside original predeclared configuration. Automatic evaluator self-tests run before each native launch. Preserve historical Gemini failures separately. Results will be linked here after execution or a concrete stop boundary.
