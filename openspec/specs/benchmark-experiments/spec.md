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

### Requirement: Explicit Gemini output allowance
Gemini experiments MAY set shared `gemini.max_output_tokens` to an integer from 1 through 65536. Omission or null SHALL preserve the historical 8192 allowance and absent-field serialization. Both generation override scopes SHALL request the same effective allowance for both arms. Dry runs and retained settings SHALL expose that allowance; invalid values SHALL fail before native CLI inspection or inference.

#### Scenario: Explicit allowance is delivered
- **WHEN** a Gemini experiment selects 32768 output tokens
- **THEN** both core and selected-model generation overrides request 32768 while model, reasoning and other controls remain unchanged

#### Scenario: Invalid allowance is rejected
- **WHEN** an experiment specifies zero, a negative value, more than 65536 or a non-integer allowance
- **THEN** configuration resolution refuses it before inspecting the native executable or reading credentials

#### Scenario: Historical evidence remains replayable
- **WHEN** an older experiment omits the output allowance
- **THEN** regenerated native settings retain 8192 and serialization omits the new field, preserving historical artifact validation and offline replay

#### Scenario: Retained allowance disagrees with configuration
- **WHEN** saved native generation settings or original configuration have been altered to a different output allowance
- **THEN** execution and retained-evidence validation refuse the inconsistent provenance rather than silently accepting the new allowance

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

### Requirement: Claude API calibration adapter
The benchmark runner SHALL support Claude Code API-key authentication with a fixed Haiku model, shared explicit effort across arms, isolated sessions, bounded execution, inspectable configuration and no subscription fallback. It SHALL preserve native per-model usage and terminal evidence, normalize cache tokens exactly once and leave missing accounting unknown. It SHALL reject unexpected models and preserve failed attempts during offline replay.

#### Scenario: Same-model Haiku A/A
- **WHEN** identical treatments select the Claude native adapter
- **THEN** both arms use the same configured model and effort in independent sessions
- **AND** only explicit API credentials reach the isolated child

#### Scenario: Unusable accounting
- **WHEN** Claude fails or returns incomplete, inconsistent or unexpected-model accounting
- **THEN** the runner retains the terminal evidence and applicable stop reason
- **AND** it makes no unsupported quality or billing claim

#### Scenario: Policy and authentication preflight
- **WHEN** Claude runs on a managed installation or without the required API credential
- **THEN** managed policy is never bypassed and missing credentials never fall back to a subscription
- **AND** the live calibration requires recorded policy preflight

#### Scenario: Structured replies and legacy replay
- **WHEN** a new Claude experiment executes
- **THEN** its explicit json-schema contract requires a nonempty source in native structured_output without prose fallback
- **AND** legacy text-output artifacts remain replayable with unchanged failure outcomes but cannot launch new model calls

### Requirement: Codex API calibration adapter
The benchmark runner SHALL support API-only Codex execution with fixed Luna model configuration, shared explicit effort, isolated sessions, structured source replies and bounded execution. It SHALL preserve native terminal/accounting evidence, leave unknown values unknown, reject unsupported identity claims and retain unsuccessful attempts during offline replay.

#### Scenario: Same-model Luna A/A
- **WHEN** both arms select identical treatments through the Codex native adapter
- **THEN** they use identical requested model/settings in independent sessions
- **AND** configured selection is not misrepresented as provider-reported model identity

#### Scenario: Accounting and terminal integrity
- **WHEN** native output is malformed, incomplete, contradictory or lacks required accounting or identity evidence
- **THEN** the runner retains the evidence and applicable refusal/stop outcome
- **AND** cached input and reasoning output are never added twice to primary totals

#### Scenario: API-only isolation
- **WHEN** the adapter executes
- **THEN** only the explicitly selected API credential is available to the authorized provider transport
- **AND** subscription fallback, ambient configuration and session reuse are excluded without bypassing host policy

### Requirement: Gemini native structured source
New Gemini benchmark execution SHALL request native JSON schema output containing exactly one string source property. Legacy prose-output configurations SHALL remain replayable with unchanged saved settings but SHALL NOT start new inference. Strict parsing SHALL NOT repair fenced or prose output.

#### Scenario: Native schema is requested
- **WHEN** a new Gemini experiment starts
- **THEN** native generation configuration requests application/json and a schema requiring only source

#### Scenario: Legacy replay remains immutable
- **WHEN** a previously recorded Gemini experiment without schema mode is rescored
- **THEN** its original configuration and native evidence remain unchanged and no model is called

### Requirement: Operational stops precede missing telemetry
Native timeout, cancellation and adapter failures SHALL retain an outcome-derived stop reason rather than being described only as unavailable model identity. Existing artifacts with prior stop reasons SHALL remain replayable when supported by their original evidence.

#### Scenario: Timeout without native output
- **WHEN** a native attempt times out before reporting model identity
- **THEN** the new run records an attempt timeout and starts no further attempt

### Requirement: Reused sessions are rejected before grading
A repeated native session identifier SHALL prevent candidate grading for that attempt.

#### Scenario: Duplicate session contains executable source
- **WHEN** a second native attempt reports an already seen session identifier
- **THEN** its generated source is not compiled or executed and the run stops with session reuse evidence

### Requirement: Immutable instruction bundle resolution
A benchmark MAY select explicit versioned instruction bundles. Resolution SHALL retain ordered source content, component hashes and effective instruction content before adapter invocation. Input reads SHALL be bounded and confined to declared local bundle inputs without implicit discovery or execution.

#### Scenario: Explicit ordered files resolve offline
- **WHEN** a dry run selects an instruction bundle
- **THEN** the result contains the ordered immutable treatment snapshot and execution limits without a model call

#### Scenario: Explicit input bounds and composition
- **WHEN** an experiment selects version-1 bundle manifests
- **THEN** A version-1 manifest SHALL name 1–32 unique ordered file paths naming regular UTF-8 files beneath its directory, SHALL be at most 64 KiB, and SHALL compose at most 1 MiB per arm including two-newline separators. Composition SHALL preserve file bytes and exclude manifest metadata. Bundle selection SHALL reject nonempty inline instruction strings.

#### Scenario: Invalid input is refused before inference
- **WHEN** an input is missing, escapes its allowed root, uses an unsupported symlink or exceeds configured content bounds
- **THEN** resolution fails before invoking an adapter

#### Scenario: Known evidence exceeds the artifact limit
- **WHEN** the resolved bundle snapshot and scheduled requests alone exceed the 64 MiB artifact limit after JSON encoding
- **THEN** resolution refuses the experiment before inference even when each arm fits its individual content bound

### Requirement: Effective treatment comparison
A/A SHALL require identical effective instruction content and fixed execution controls. A/B SHALL require an effective treatment difference. Display labels, paths or version metadata alone SHALL NOT count as a treatment difference.

#### Scenario: Metadata differs but instructions do not
- **WHEN** two differently labeled bundles resolve to identical instruction content
- **THEN** metadata differences remain visible but do not qualify as a harness A/B treatment

### Requirement: Bundle replay independence
Offline comparison and rescore SHALL use retained bundle content without rereading original files or invoking a model. New inference SHALL refuse stale or tampered resolution. Existing inline experiments and artifacts SHALL remain compatible. Retained evidence validation SHALL reconstruct the effective instructions, verify manifest/component/effective hashes and ordering, and verify that execution controls match the original configuration.

#### Scenario: Source bundle files are removed
- **WHEN** a retained experiment is rescored after its original bundle files are removed
- **THEN** rescore uses saved evidence without model calls or changes to original artifact bytes

#### Scenario: Inputs changed after resolution
- **WHEN** a new execution receives resolution whose input content has changed
- **THEN** it refuses inference rather than silently executing a different treatment
- **AND** Metadata-only input edits SHALL also invalidate a resolution for new execution.

### Requirement: Instruction evidence is not execution evidence
Bundle provenance SHALL distinguish requested and delivered instruction content from observed native execution. Delivering skill-like text SHALL NOT establish that a native skill or hook executed.

#### Scenario: Skill text is included as instructions
- **WHEN** a bundle includes text describing a skill
- **THEN** evidence records instruction delivery without asserting native skill execution

### Requirement: Gemini fatal stderr evidence
The runner SHALL retain recognized Gemini turn-limit error evidence without treating absent native usage as zero or executing later attempts. Historical artifacts without this evidence SHALL remain replayable unchanged.

#### Scenario: Stderr-only turn limit
- **WHEN** Gemini exits with code53, empty stdout and an unambiguous bounded native FatalTurnLimitedError envelope on stderr
- **THEN** the runner retains reported error/session details and a local four-turn-limit diagnostic
- **AND** usage stays unknown, the attempt has no grade and no subsequent attempt executes

#### Scenario: Untrusted framing and compatibility
- **WHEN** stderr contains malformed, conflicting or duplicate-key candidate envelopes, streams are truncated, or the process times out or exits with another code
- **THEN** the runner does not synthesize recognized turn-limit evidence
- **AND** replay of newly normalized evidence re-derives it from raw stderr while historical artifacts preserve their original outcomes

### Requirement: Versioned Gemini best-effort accounting
New Gemini execution SHALL explicitly select the verified stable CLI 0.64.0. Historical 0.43.0 configurations and artifacts SHALL remain replayable without changing their settings or comparison interpretation. Reports SHALL distinguish known normalized usage from unknown usage and SHALL NOT claim reconciled cost.

#### Scenario: Explicit stable execution
- **WHEN** a new Gemini experiment is resolved or executed
- **THEN** legacy or unsupported selection is refused before executable probes or credentials
- **AND** the actual CLI and retained version metadata must match the selected stable contract

#### Scenario: Partial retained history
- **WHEN** a stable CLI run contains reported and unknown usage, including failed or unpaired attempts
- **THEN** its report includes the known normalized token subtotal and reported/unknown retained-attempt counts
- **AND** no reported usage yields an unknown subtotal, a measured zero stays zero, cost stays unknown and conservative stop rules remain in force

#### Scenario: Historical replay
- **WHEN** an artifact using omitted or explicit 0.43.0 selection is rescored
- **THEN** the historical settings and comparison remain unchanged
- **AND** mismatched retained native-version provenance is refused

#### Scenario: Zero-defaulted native usage
- **WHEN** the stable CLI supplies an all-zero normalized snapshot that cannot distinguish omitted upstream metadata from measured usage
- **THEN** the runner retains raw observations, source and errors but marks normalized usage partial and unknown
- **AND** historical parsing keeps its original interpretation

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
