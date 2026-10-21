## ADDED Requirements

### Requirement: Mercury through the existing Claude runtime

Mercury experiments SHALL use an explicit optional Mercury configuration on the existing Claude runtime, a per-attempt pinned LiteLLM bridge, and provider usage captured before translation. Execution and replay SHALL reject missing or incomplete provider accounting rather than use translated Claude counters. Existing direct-provider configurations and historical artifacts SHALL remain unchanged.

#### Scenario: Resolve without inference

- **WHEN** a Mercury experiment is resolved with `--dry-run`
- **THEN** the runner records Claude/bridge versions and executable identities without starting inference
- **AND** unsupported model, reasoning, output format or bridge configuration is refused

#### Scenario: Execute a bounded schema task

- **WHEN** an explicitly authenticated Mercury attempt runs
- **THEN** Go starts an owned per-attempt bridge and capture proxy, keeps the provider key only in the Go runner, and executes the existing Claude structured single-file task flow
- **AND** existing process-group cleanup stops the native process and bridge on completion, error, timeout or cancellation

#### Scenario: Provider accounting is authoritative

- **WHEN** an attempt completes or is replayed
- **THEN** every captured model request has correlated complete provider usage with the expected controls and model
- **AND** absent, partial, invalid, retried or failed capture makes the attempt fail and stops later calls
- **AND** Go rejects bridge retries before provider dispatch and cancels provider HTTP requests with the attempt context
- **AND** native output truncation remains explicit in the retained envelope and offline replay
- **AND** cached and reasoning tokens remain subsets, and native CLI dollar estimates do not establish billing
