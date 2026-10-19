## ADDED Requirements

### Requirement: Immutable instruction bundle resolution
A benchmark MAY select explicit versioned instruction bundles. Resolution SHALL retain ordered source content, component hashes and effective instruction content before adapter invocation. Input reads SHALL be bounded and confined to declared local bundle inputs without implicit discovery or execution.

#### Scenario: Explicit ordered files resolve offline
- **WHEN** a dry run selects an instruction bundle
- **THEN** the result contains the ordered immutable treatment snapshot and execution limits without a model call

#### Scenario: Invalid input is refused before inference
- **WHEN** an input is missing, escapes its allowed root, uses an unsupported symlink or exceeds configured content bounds
- **THEN** resolution fails before invoking an adapter

### Requirement: Effective treatment comparison
A/A SHALL require identical effective instruction content and fixed execution controls. A/B SHALL require an effective treatment difference. Display labels, paths or version metadata alone SHALL NOT count as a treatment difference.

#### Scenario: Metadata differs but instructions do not
- **WHEN** two differently labeled bundles resolve to identical instruction content
- **THEN** metadata differences remain visible but do not qualify as a harness A/B treatment

### Requirement: Bundle replay independence
Offline comparison and rescore SHALL use retained bundle content without rereading original files or invoking a model. New inference SHALL refuse stale or tampered resolution. Existing inline experiments and artifacts SHALL remain compatible.

#### Scenario: Source bundle files are removed
- **WHEN** a retained experiment is rescored after its original bundle files are removed
- **THEN** rescore uses saved evidence without model calls or changes to original artifact bytes

#### Scenario: Inputs changed after resolution
- **WHEN** a new execution receives resolution whose input content has changed
- **THEN** it refuses inference rather than silently executing a different treatment

### Requirement: Instruction evidence is not execution evidence
Bundle provenance SHALL distinguish requested and delivered instruction content from observed native execution. Delivering skill-like text SHALL NOT establish that a native skill or hook executed.

#### Scenario: Skill text is included as instructions
- **WHEN** a bundle includes text describing a skill
- **THEN** evidence records instruction delivery without asserting native skill execution
