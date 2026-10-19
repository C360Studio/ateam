# A/A calibration evidence checklist

No live result is recorded here. Complete this checklist beside a run's retained artifacts after a separately authorized pilot; preserve the original run directory.

- Authorization: chosen provider/model/reasoning, dollar cap and enforcement or accepted overshoot policy, attempt count, per-attempt/overall time limits, reported-token threshold, credential source (never value), trace retention decision.
- Configuration: original reviewed JSON, dry-run output, config digest, actual CLI version and executable digest, requested versus resolved settings (unknown where unavailable), runner and Go grader identities.
- Checks: exact commit and `task verify`, `ateam doctor`, `ateam bench --selftest` evidence. Synthetic parser fixtures are offline evidence, not a billing reconciliation.
- Results: relative links to `run.json`, `resolved.json`, `attempt-*.json`, `bench compare --json` and human-readable output. Record scheduled/retained/unpaired attempt counts, stop reason and every terminal status.
- Accounting: known, partial, invalid and missing usage; reported input/output/cache/reasoning breakdowns; compare any independently available provider usage for the same isolated pilot. Record discrepancies and unknowns, never replace them with zero. Cost remains unknown unless independently sourced with an explicit price basis.
- Isolation: independent observed session IDs, identical A/A instruction/config fingerprints and counterbalanced schedule; distinguish requested config from observed item activity.
- Replay: relative link to immutable rescore artifact; confirm original bytes unchanged, no agent calls and execution-failure statuses preserved.
- Interpretation: descriptive variation across the configured task mix; failure/timeout bias, unresolved telemetry issues, no equivalence or generalized performance claim. A/A discrepancies must be understood before separately authorizing an A/B pilot.
