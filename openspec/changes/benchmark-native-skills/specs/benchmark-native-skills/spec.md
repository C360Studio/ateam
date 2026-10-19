## ADDED Requirements

### Requirement: Explicit isolated native skill treatment
A native skill experiment SHALL resolve bounded explicit skill files into immutable per-arm snapshots. The initial supported contract SHALL hold skill identity, description, ordinary instructions, model, reasoning and execution controls fixed while varying only the declarative body. Unsupported adapters or capabilities SHALL fail before inference.

#### Scenario: Reviewable skill comparison
- **WHEN** a dry run resolves two valid skills with different bodies
- **THEN** it identifies the body difference, exact hashes and fixed controls without model calls

#### Scenario: Identical native skills calibrate the path
- **WHEN** an A/A experiment selects identical skill bytes and fixed controls
- **THEN** both arms use the same native skill path and their source labels do not change the prompt

#### Scenario: Unsafe or unsupported skill selection
- **WHEN** a request selects an unsupported adapter, escaping path, symlink, oversized input or executable configuration
- **THEN** resolution refuses it without reading credentials or invoking inference

#### Scenario: Fresh arm isolation
- **WHEN** a supported native skill attempt starts
- **THEN** only that arm's snapshotted skill is staged in a fresh workspace and home with the reviewed tool permissions
- **AND** ambient skills, hooks and unrelated capabilities are not enabled

### Requirement: Native activation evidence has distinct stages
Evidence SHALL distinguish requested and staged skill bytes from native discovery, activation requests and activation results. Observed activation SHALL require identified native evidence. Missing or incomplete evidence SHALL remain unknown. Skill activation SHALL NOT imply instruction compliance or improved performance.

#### Scenario: Request without a successful result
- **WHEN** a retained trace contains a skill activation request but no identifiable successful result
- **THEN** reporting does not assert successful activation

#### Scenario: Configuration is the only evidence
- **WHEN** a skill was staged but discovery or activation telemetry is absent
- **THEN** staging is reported and the unobserved native stages remain unknown

#### Scenario: Activation failure or unrelated skill
- **WHEN** native events show a failed activation or activation of another skill
- **THEN** those events do not count as successful activation of the selected treatment

### Requirement: Skill evidence preserves replay and unsuccessful attempts
The benchmark SHALL retain failed, unactivated, timed-out and partial attempts with available usage and independent grades. Comparison and rescore SHALL use saved snapshots and traces without source files or model calls. New inference SHALL refuse stale or tampered resolution. Existing instruction-only artifacts SHALL remain replayable.

#### Scenario: Treatment sources disappear after the run
- **WHEN** saved skill evidence is rescored after the original files are removed
- **THEN** results derive from retained evidence without modifying the original artifact or invoking a model

#### Scenario: Skill sources change before execution
- **WHEN** selected skill bytes change after resolution
- **THEN** inference is refused rather than silently using a different treatment

#### Scenario: Some attempts never activate the skill
- **WHEN** only part of a run has successful activation evidence
- **THEN** comparison retains every attempt and reports activation evidence alongside quality and usage without filtering to activated attempts

### Requirement: Native skill calibration claims require evidence
A new native skill path SHALL remain uncalibrated until evidence from a bounded, explicitly authorized live A/A run meets predeclared acceptance criteria. A calibration attempt SHALL be distinct from an accepted calibration outcome. Offline fixtures and dry runs SHALL NOT establish calibration or performance improvement.

#### Scenario: Offline implementation checks pass
- **WHEN** skill isolation and trace fixtures pass without real inference
- **THEN** the path remains uncalibrated and the result does not claim skill performance improvement

#### Scenario: A calibration attempt lacks sufficient evidence
- **WHEN** a live A/A attempt encounters provider failures or unknown activation and does not meet predeclared acceptance criteria
- **THEN** its evidence is retained and the path remains uncalibrated
