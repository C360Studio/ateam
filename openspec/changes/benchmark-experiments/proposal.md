# Reproducible benchmark experiments

Issue: https://github.com/C360Studio/ateam/issues/6

## Why
ATeam needs evidence about what prompt and skill changes do to agent outcomes. Existing reference-fixture checks cannot support agent performance claims, and some report fields overstate what was tested.

## What Changes
- Make fixture checks and saved reports honest about unsupported evaluators, failures, and provenance.
- Add a versioned JSON experiment contract, dry run, explicit command adapter, retained per-attempt artifacts, offline rescore, and paired comparison.
- Include an offline example and small Go task set; record unknown usage and insufficient evidence explicitly.
- Pin OpenSpec 1.14.3 as repository-local development tooling.
- **BREAKING**: `--rescore` no longer trusts self-test booleans; fixture artifacts are validated and rerun, while experiment rescoring uses retained candidate sources.

## Capabilities

### New Capabilities
- `benchmark-experiments`: controlled task execution, independent grading, artifact provenance, and paired reporting.

### Modified Capabilities
None. Existing harness requirements remain unchanged.

## Impact
`internal/bench`, benchmark CLI in `cmd/ateam`, `benchmarks/`, local npm tooling, and Taskfile spec validation. No UI or main README changes. No paid runs, merge, or generalized performance claim. Provider-native adapters and external task datasets are subsequent work.
