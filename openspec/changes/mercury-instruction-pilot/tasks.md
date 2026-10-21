# Mercury instruction A/B pilot

## Preparation
- [x] Reconcile completed canary/A/A evidence and define the bounded instruction-only question.
- [x] Freeze exact instruction bundles, configuration, schedule, budget, stop rules and interpretation limits.
- [x] Validate exact bundle bytes, fixed controls, calibration evidence and the counterbalanced schedule with a credential-free dry-run.
- [x] Obtain independent read-only protocol review; bind final-head approval and required full repository verification evidence in the delivery PR before owner review.

## Execution boundary
- [x] Obtain separate owner authorization before executing this one A/B cohort; the owner explicitly authorized the frozen twelve-attempt run on October 21, 2026. That permission is consumed and grants no repeat.
- [x] Retain every outcome and independently replay/grade the cohort: 12/12 passed, both arms 6/6, complete provider usage and unchanged original evidence. Save the [worked results](../../../benchmarks/experiments/mercury-instructions-v1/results.md), machine-readable summary and private-evidence manifest; no retry or further inference. Archive this completed change at delivery-PR close.
