# Tasks

## Preparation
- [x] Split issue #30 and isolate the pilot from native skill work.
- [x] Freeze explicit bundles, configurations, advancement gates and spending policy.
- [x] Independently review protocol and complete clean-tree verification before sampling.

## Execution
- [x] Attempt the owner-authorized $5 v2 canary with an explicit shared output allowance; retain both attempts, confirm one unevaluable pair by immutable offline replay and stop on a native failure with missing usage.
- [x] Attempt the bounded availability pair and validate retained evidence: first native response was incomplete JSON; the runner stopped with no grade or pair, and offline replay matched.
- [ ] Run A/A only after canary acceptance; inspect discrepancies and retain immutable replay evidence.
  Hold: v2 retained one passing response and one CLI turn-limit failure with unknown usage; canary acceptance failed. Investigate the completion/accounting boundary before another frozen cohort.
- [ ] Run A/B only after A/A acceptance and budget review; preserve all outcomes.
  Hold: no accepted A/A calibration exists for this path.

## Delivery
- [x] Publish a sanitized report or exact blocked boundary, with hashes, limitations and retained evidence references.
- [x] Independently review final results and run relevant final verification before PR delivery; evidence is recorded in PR #32.
