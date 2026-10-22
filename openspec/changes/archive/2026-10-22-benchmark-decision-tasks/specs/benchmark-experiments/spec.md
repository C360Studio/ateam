## ADDED Requirements

### Requirement: Decision-opportunity tasks

The benchmark SHALL provide a `decision-opportunity` tier whose tasks a naive solution can pass behaviorally while missing a reuse, minimal-surface or change-propagation decision. A task MAY carry a pre-existing package file that the prompt shows verbatim, states must not be returned, and the grader compiles beside the single returned file. Existing tasks, their prompts and their serialized evidence SHALL remain unchanged.

#### Scenario: Existing package file

- **WHEN** a decision-opportunity task with a pre-existing file is requested
- **THEN** the prompt contains the file verbatim with the instruction not to return it, and grading compiles it beside the returned file

### Requirement: Separate discipline grading

A stdlib syntax-tree discipline gate SHALL grade the decision-opportunity tier. Attempt grades and self-test results SHALL record behavior and discipline outcomes as separate fields, and a tier attempt SHALL pass only when both pass.

#### Scenario: Self-test proves both gates discriminate

- **WHEN** the offline self-test evaluates a decision-opportunity task
- **THEN** the good fixture passes behavior and discipline, the bad fixture passes behavior and is caught by discipline only, and the mutant is caught by the behavior tests

#### Scenario: Correct but undisciplined candidate

- **WHEN** a retained candidate passes the behavior tests but reimplements an existing helper, exports unrequested declarations or leaves a dependent site at a replaced value
- **THEN** the attempt is failed with behavior recorded as passed and discipline recorded as failed

#### Scenario: Contradictory discipline evidence

- **WHEN** a retained decision attempt omits either outcome or claims a pass that contradicts them, or an existing-task attempt carries discipline outcomes
- **THEN** loading, comparison, rescoring and export reject the artifact

### Requirement: Descriptive source size comparison

Comparison SHALL report summed retained source bytes per arm over pairs where both arms passed, omit the measure when no such pair exists, and SHALL NOT derive a winner from it.

#### Scenario: Bloat signal on passing pairs

- **WHEN** both arms pass a pair with sources of different sizes
- **THEN** comparison reports each arm's source bytes over the both-passed pairs and excludes failed or unevaluable pairs
