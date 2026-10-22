# Benchmark experiment report

Synthetic prescribed&#x2D;task A&#x2F;B — ab

Descriptive observations only. No winner, statistical significance, equivalence, ROI, or skill-activation claim. Cost is unknown.

## Treatment and provenance

Field | Recorded value
--- | ---
Agent | fake
Agent version | 1
Requested model | none
Runtime | synthetic runtime
Native version | not recorded
Bridge version | not recorded
Configuration SHA-256 | 6dfa27d356bb631dc3047f4a4381da40d9e953f80cfd0c3b628b0b23d0502491
Baseline instruction SHA-256 | b595bcb7f0070c9a5eaa7ae9a85468f2e9b2cb4c005aeda4494c2768573df353
Candidate instruction SHA-256 | 5b995f0944f3d441fd11aa5d46531e170e1d3dd1387b0b1e378ed1d9f0de0981
Runner SHA-256 | 087bf5ca75c48138e467aae95a7822de0e76437105bbfdd2163148d047e445db
Evaluator SHA-256 | 651d43eb912048f704b4e838e7a630d2350ba10f6d3181cdfd4f87c8d6e6ab75
Go driver SHA-256 | 15ffb35e7eb03ea8de594435da97d22d884501ca88467c1feb3de7eb75146e75
Go driver version | synthetic Go
Native settings SHA-256 | not recorded
Native arguments SHA-256 | not recorded
Command file SHA-256 (paths omitted) | 1c94381c3359991b8855a03705afd3c7ccb70c23835e9f8570488658b07878a2
Tasks | safe&#x2D;path
Budget policy | not recorded

Repetitions: 2; seed: 42; per-attempt limit: 5 s; run limit: 0 s; token ceiling: 0. Zero run/token limits mean no configured ceiling for command adapters.

## Outcomes and denominators

Scheduled: 4; retained: 3; unstarted: 1. Graded: 2 (2 passed, 0 failed); unevaluable: 1.

Stop reason: cancelled. Replay: not&#x2D;provided; changed attempt indexes: &#x5B;&#x5D; (zero-based).

Accounting basis: adapter&#x2D;reported. Primary usage known for 0/3 retained attempts; unknown for 3. Known primary token subtotal: unknown; cost: unknown. Cached input and reasoning output are subsets, never added again. Partial provider observations without normalized usage are excluded.

Attempts retained&#x3A; 3&#x2F;4 scheduled&#x3B; 1 unpaired

Run stopped&#x3A; cancelled

Paired pilot&#x3A; 1 pairs

Baseline passed&#x3A; 1&#x2F;1&#x3B; candidate passed&#x3A; 1&#x2F;2

Quality changes&#x3A; 0 improvements&#x2C; 0 regressions&#x2C; 1 both passed&#x2C; 0 neither passed

Unevaluable quality pairs&#x3A; 0 &#x28;execution&#x2F;grading errors or timeouts&#x29;

baseline&#x2F;passed&#x3A; 1

candidate&#x2F;adapter&#x2D;error&#x3A; 1

candidate&#x2F;passed&#x3A; 1

baseline usage completeness&#x3A; 0&#x2F;1 primary totals reported&#x3B; 1 unknown&#x3B; cache breakdown 0&#x2F;1&#x3B; reasoning breakdown 0&#x2F;1

baseline all&#x2D;attempt execution&#x3A; n&#x3D;1 mean&#x3D;0&#x2E;000 min&#x3D;0&#x2E;000 max&#x3D;0&#x2E;000 population&#x2D;SD&#x3D;0&#x2E;000 s

baseline passed execution&#x3A; n&#x3D;1 mean&#x3D;0&#x2E;000 min&#x3D;0&#x2E;000 max&#x3D;0&#x2E;000 population&#x2D;SD&#x3D;0&#x2E;000 s

baseline reported primary tokens&#x3A; unknown &#x28;n&#x3D;0&#x29;

candidate usage completeness&#x3A; 0&#x2F;2 primary totals reported&#x3B; 2 unknown&#x3B; cache breakdown 0&#x2F;2&#x3B; reasoning breakdown 0&#x2F;2

candidate all&#x2D;attempt execution&#x3A; n&#x3D;2 mean&#x3D;0&#x2E;000 min&#x3D;0&#x2E;000 max&#x3D;0&#x2E;000 population&#x2D;SD&#x3D;0&#x2E;000 s

candidate passed execution&#x3A; n&#x3D;1 mean&#x3D;0&#x2E;000 min&#x3D;0&#x2E;000 max&#x3D;0&#x2E;000 population&#x2D;SD&#x3D;0&#x2E;000 s

candidate reported primary tokens&#x3A; unknown &#x28;n&#x3D;0&#x29;

Variation is descriptive across the configured task mix&#x3B; population SD describes these observations&#x2C; not uncertainty or equivalence&#x2E; Cache&#x2F;reasoning breakdowns are subsets and are not added to primary totals&#x2E;

Mean candidate&#x2D;minus&#x2D;baseline execution time&#x3A; 0&#x2E;000 s &#x28;1 both&#x2D;passed pairs&#x29;

Source bytes over 1 both&#x2D;passed pairs&#x3A; baseline 43&#x2C; candidate 43 &#x28;descriptive size&#x2C; not a quality verdict&#x29;

Token comparison&#x3A; unknown &#x28;1 missing&#x2D;usage pairs&#x29;

## Retained attempts

Index | Task | Repetition | Arm | Status | Quality passed | Execution ns | Grading ns | Input | Output | Cached subset | Reasoning subset | Source SHA-256
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
0 | safe&#x2D;path | 1 | baseline | passed | true | 100 | 7 | unknown | unknown | unknown | unknown | d1a0af70871c25c00625bb37162a5e78c330c3ccd8f49f6a43331c792a4b5899
1 | safe&#x2D;path | 1 | candidate | passed | true | 130 | 7 | unknown | unknown | unknown | unknown | d1a0af70871c25c00625bb37162a5e78c330c3ccd8f49f6a43331c792a4b5899
2 | safe&#x2D;path | 2 | candidate | adapter&#x2D;error | unknown | 1000 | 0 | unknown | unknown | unknown | unknown | not recorded

## Matched pairs

Baseline index | Candidate index | Quality eligible | Both-passed execution delta ns | Known-usage token delta
--- | --- | --- | --- | ---
0 | 1 | true | 30 | unknown

Deltas are candidate minus baseline. Execution deltas include only both-passed pairs; token deltas include pairs with both normalized usage observations, including failures. Unstarted attempts have no invented outcomes.

## Schedule

Index | Task | Repetition | Arm
--- | --- | --- | ---
0 | safe&#x2D;path | 1 | baseline
1 | safe&#x2D;path | 1 | candidate
2 | safe&#x2D;path | 2 | candidate
3 | safe&#x2D;path | 2 | baseline

## Evidence and interpretation

Original and replay artifacts are not included. Rederivation requires separately retained originals and the matching evaluator version; this report alone is not replayable. See evidence.json for byte digests and results.json for projected facts. A matched replay describes recorded outcome agreement, not proof of independent execution.

Raw instructions, generated source, traces, diagnostics, environment, paths and native session IDs are omitted. Included metadata still requires human review before publication. Keep hypotheses and the next decision in a separate human-authored interpretation.md.

Classifier decisions and classifier overhead are not recorded by this artifact schema and remain unmeasured. Execution time must not be presented as isolated classifier latency.
