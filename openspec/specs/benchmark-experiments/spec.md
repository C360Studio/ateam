# benchmark-experiments Specification

## Purpose
Defines controlled paired command-adapter experiments, honest offline evaluator self-tests, retained independently graded evidence, and descriptive comparison without unsupported agent-performance claims.

## Requirements

### Requirement: Honest evaluator self-tests
Self-test reports SHALL distinguish actual behavioral checks, continuity checks, and unsupported evaluators. Infrastructure errors SHALL NOT be reported as successful defect detection. Loading a report SHALL validate its schema and invariants, and rescoring SHALL execute evaluation rather than trusting stored success flags.

#### Scenario: Unsupported component evaluator
- **WHEN** a component has only a keyword matcher and no behavioral evaluator
- **THEN** its self-test is reported as skipped/unsupported and contributes no successful safety or mutation evidence

#### Scenario: Forged success summary
- **WHEN** an artifact claims success with no valid task evidence
- **THEN** loading or rescoring rejects it

### Requirement: Controlled bounded experiments
The CLI SHALL validate a versioned configuration before executing any adapter, support a dry run, run both configured instruction variants against the same tasks and repetition counts in a reproducible counterbalanced order, and bound every adapter/grading operation. Each attempt SHALL receive a fresh working directory and configuration home without implicit inherited credentials.

#### Scenario: Offline dry run
- **WHEN** a valid experiment is requested with dry-run enabled
- **THEN** the resolved configuration and schedule are returned without executing the adapter

#### Scenario: Timeout or malformed adapter response
- **WHEN** an adapter exceeds its time limit or returns an invalid response
- **THEN** the attempt remains in the report with a distinct non-success status and retained diagnostics

### Requirement: Retained independently graded artifacts
Runs SHALL retain the exact task request, raw adapter output, candidate source, resolved configuration, task/evaluator hashes, runtime provenance, separate execution/grading timing, and grade diagnostics in a unique directory. The grader SHALL use trusted task tests outside the adapter workspace. Missing token usage SHALL remain unknown.

#### Scenario: Unknown token usage
- **WHEN** an otherwise valid adapter response has no usage record
- **THEN** grading proceeds and usage remains null rather than zero

#### Scenario: Rescore retained candidate
- **WHEN** an intact run with a matching evaluator is rescored
- **THEN** candidate grading executes again without any adapter/model call and a new artifact is written without overwriting the original

### Requirement: Paired descriptive comparisons
Comparison SHALL reject incompatible artifacts and incomplete histories without an explicit stop reason. Complete runs and valid stopped schedule prefixes SHALL report every retained attempt, distinguish unpaired attempts, and disclose denominators for successful-pair timing and known usage. Reports SHALL NOT claim statistical significance, quality equivalence, or actual tool activation from configuration alone.

#### Scenario: Faster failed candidate
- **WHEN** the candidate fails a task that the baseline passes
- **THEN** the regression remains visible and the failed candidate is not counted as a successful-pair speed improvement

#### Scenario: Stopped after an unpaired attempt
- **WHEN** the native pilot stops after one arm due to unknown usage or a token ceiling
- **THEN** comparison preserves that attempt in arm summaries without manufacturing its paired comparison

### Requirement: Native agent evidence
The harness SHALL provide a Gemini CLI adapter with explicit API-key authentication, isolated attempts, bounded redacted output, CLI identity and inspectable execution settings. It SHALL normalize complete CLI-reported input, candidate and thought counts without double-counting cache. Incomplete accounting SHALL remain unknown, failures SHALL retain evidence, and unavailable backend-resolved settings SHALL remain unknown.

#### Scenario: Provider reports cached tokens
- **WHEN** Gemini reports prompt, cached prompt, candidate and thought counts
- **THEN** normalized input includes cached input once and normalized output includes candidate plus thought tokens once

#### Scenario: Failed or interrupted execution
- **WHEN** a native attempt cannot return a complete accounting record
- **THEN** its failure and retained output remain visible and no further native attempt starts with unknown usage

#### Scenario: Failed API requests lack complete accounting
- **WHEN** API request/error counters are absent or any request failed
- **THEN** normalized usage remains unknown even if a later request succeeds, and no further attempt starts

### Requirement: A/A calibration
The harness SHALL accept explicitly selected A/A experiments only with identical treatment configurations, execute independent arms in counterbalanced order, and report descriptive outcome/time/usage variation without claiming equivalence. Default A/B SHALL reject identical treatments.

#### Scenario: Identical arms
- **WHEN** identical instructions are selected in A/A mode
- **THEN** each arm starts a fresh session and the comparison identifies calibration evidence

### Requirement: Bounded native pilots
Native pilots SHALL expose attempt counts, time limits, authentication policy and usage stop limits in dry-run output without model calls. Execution SHALL stop starting native attempts after the usage ceiling or unknown usage and retain partial evidence. Documentation SHALL distinguish these controls from a hard monetary cap and identify live authorization as a separate boundary.

#### Scenario: Unknown usage
- **WHEN** a native attempt omits complete usage
- **THEN** no subsequent native attempt starts and retained evidence identifies the stop reason

### Requirement: Fixed-model harness treatments
The native pilot SHALL use gemini-3.8-flash with the same requested model and generation settings for both arms. A/B SHALL vary explicit harness instruction treatment; A/A SHALL hold that treatment identical. Observed usage from an unexpected model or missing per-model identity SHALL stop the pilot and SHALL NOT contribute a valid same-model quality result. Model routing and role-to-model defaults SHALL remain outside this benchmark change.

#### Scenario: Provider silently changes a model
- **WHEN** native per-model statistics include a model other than the configured model
- **THEN** the attempt retains its accounting evidence but is marked an execution error and further calls stop

### Requirement: Generated workspace trust
Native Gemini execution SHALL grant session-only trust to its runner-created workspace through the isolated child environment. It SHALL NOT modify the parent environment or global trust records. Managed policy refusal and configured tool restrictions SHALL remain enforced.

#### Scenario: Independent trusted attempts
- **WHEN** a native attempt starts in a fresh generated workspace
- **THEN** its child receives explicit session trust while other attempts and the parent retain separate state
