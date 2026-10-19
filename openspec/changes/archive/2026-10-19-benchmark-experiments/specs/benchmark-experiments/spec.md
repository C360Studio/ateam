## ADDED Requirements

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
Comparison SHALL reject incompatible or incomplete artifacts, report all attempted pairs and quality outcomes, and separately disclose denominators for successful-pair timing and known usage. Reports SHALL NOT claim statistical significance, quality equivalence, or actual tool activation from configuration alone.

#### Scenario: Faster failed candidate
- **WHEN** the candidate fails a task that the baseline passes
- **THEN** the regression remains visible and the failed candidate is not counted as a successful-pair speed improvement
