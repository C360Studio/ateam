# Real-agent benchmark calibration

## Why
The merged paired benchmark foundation (#6, #7) grades fixed reference adapters. Issue #10 needs observed real-agent telemetry and an explicit A/A mode before prompt or skill performance claims.

## What Changes
- Add one built-in Codex CLI adapter with isolated sessions/config, explicit API-key environment authentication, bounded/redacted native event evidence, and honest usage completeness.
- Add explicit A/A mode with identical treatments and counterbalanced independent arms; preserve accidental-identical rejection in A/B.
- Add descriptive calibration reporting and a reviewable pilot configuration with explicit attempt/time/token spending policy.
- Verify with offline recorded-shape events and fake CLI integration tests, mutation checks, full gates, and independent review.

## Capabilities
### Modified Capabilities
- `benchmark-experiments`: native agent execution and A/A calibration.

## Impact
`internal/bench`, `cmd/ateam/bench.go`, benchmark examples/docs. No UI changes, new dependencies, additional providers, or performance claims. No paid execution until the user selects model and spending/attempt/time limits.
