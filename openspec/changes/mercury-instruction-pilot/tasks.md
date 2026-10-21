# Mercury instruction A/B pilot

## Preparation
- [x] Reconcile completed canary/A/A evidence and define the bounded instruction-only question.
- [x] Freeze exact instruction bundles, configuration, schedule, budget, stop rules and interpretation limits.
- [x] Validate exact bundle bytes, fixed controls, calibration evidence and the counterbalanced schedule with a credential-free dry-run.
- [x] Obtain independent read-only protocol review; bind final-head approval and required full repository verification evidence in the delivery PR before owner review.

## Execution boundary
- [ ] Obtain separate owner authorization before executing this one A/B cohort.
  Hold: the current request prepares a plan for review only; consumed canary/A/A permissions authorize no A/B calls.
- [ ] After authorization, retain every outcome and independently replay/grade the cohort; report all failures and limitations without automatic retries or further inference.
