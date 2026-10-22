# Benchmark experiment report

decision&#x2D;tier&#x2D;v1&#x2D;canary — aa

Descriptive observations only. No winner, statistical significance, equivalence, ROI, or skill-activation claim. Cost is unknown.

## Treatment and provenance

Field | Recorded value
--- | ---
Agent | claude
Agent version | not recorded
Requested model | mercury&#x2D;3
Runtime | go1&#x2E;26&#x2E;4 darwin&#x2F;arm64
Native version | 2&#x2E;1&#x2E;292 &#x28;Claude Code&#x29;
Bridge version | LiteLLM&#x3A; Current Version &#x3D; 1&#x2E;99&#x2E;0
Configuration SHA-256 | 799a3735217038fc804dc1dca0affa5448c8f25d76bd05d9e88c284d8d0b6ebc
Baseline instruction SHA-256 | 957138123c57e7952a4c986fe379c9028b7ad8c9a21ff841506fa3158df5ec6e
Candidate instruction SHA-256 | 957138123c57e7952a4c986fe379c9028b7ad8c9a21ff841506fa3158df5ec6e
Runner SHA-256 | 171994bb2d725400771bca6c0f5fa5232244fc2a9c53a8498602d4407c1cf979
Evaluator SHA-256 | 0311ba302fe09f4875f932a4dfcb3ff8c2d1bb0ae07f0b7554d5d28ca1485c32
Go driver SHA-256 | e61851b2c0cde9b9ac4ae044fcce6ba9d55cb15f98ded3964793acaa8632242f
Go driver version | go version go1&#x2E;26&#x2E;4 darwin&#x2F;arm64
Native settings SHA-256 | bd54f1802eb9db701fecc7fdb1a03ddc21e14f70552bb6a402c3bbe6e98af17f
Native arguments SHA-256 | 1bbc5dc1bd6771bae6198660ccbba9f37f2925874420f5f6b0c7a73e6654665d
Command file SHA-256 (paths omitted) | 97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f&#x2C; c2e78b7df0e3b847316955301c13783f6f64db04d979dd69fdbfdcf018eec73c
Tasks | reuse&#x2D;email
Budget policy | INCEPTION&#x5F;API&#x5F;KEY only in the Go runner&#x3B; Claude and LiteLLM 1&#x2E;99&#x2E;0 receive local tokens&#x3B; model mercury&#x2D;3&#x2C; low effort&#x2C; 8192 output tokens&#x2F;request&#x2C; max4 provider requests&#x2C; no retries&#x2F;fallback&#x3B; raw provider usage required for every request&#x3B; stop on missing&#x2F;incomplete usage&#x3B; context closes provider requests and process groups are killed on exit&#x2F;cancel&#x3B; max&#x5F;run&#x5F;seconds includes grading&#x3B; max&#x5F;total&#x5F;tokens checked between attempts&#x3B; no hard dollar cap&#x3B; CLI cost ignored

Repetitions: 1; seed: 0; per-attempt limit: 30 s; run limit: 180 s; token ceiling: 50000. Zero run/token limits mean no configured ceiling for command adapters.

## Outcomes and denominators

Scheduled: 2; retained: 2; unstarted: 0. Graded: 2 (2 passed, 0 failed); unevaluable: 0.

Stop reason: none. Replay: matched; changed attempt indexes: &#x5B;&#x5D; (zero-based).

Accounting basis: captured&#x2D;provider&#x2D;wire. Primary usage known for 2/2 retained attempts; unknown for 0. Known primary token subtotal: 1977; cost: unknown. Cached input and reasoning output are subsets, never added again. Partial provider observations without normalized usage are excluded.

A&#x2F;A calibration&#x3A; identical configured treatments&#x3B; stochastic outcomes may differ&#x2E;

Attempts retained&#x3A; 2&#x2F;2 scheduled&#x3B; 0 unpaired

Paired pilot&#x3A; 1 pairs

Baseline passed&#x3A; 1&#x2F;1&#x3B; candidate passed&#x3A; 1&#x2F;1

Quality discordance&#x3A; 0 candidate&#x2D;only passes&#x2C; 0 baseline&#x2D;only passes&#x2C; 1 both passed&#x2C; 0 neither passed

Unevaluable quality pairs&#x3A; 0 &#x28;execution&#x2F;grading errors or timeouts&#x29;

baseline&#x2F;passed&#x3A; 1

candidate&#x2F;passed&#x3A; 1

baseline usage completeness&#x3A; 1&#x2F;1 primary totals reported&#x3B; 0 unknown&#x3B; cache breakdown 1&#x2F;1&#x3B; reasoning breakdown 1&#x2F;1

baseline all&#x2D;attempt execution&#x3A; n&#x3D;1 mean&#x3D;3&#x2E;736 min&#x3D;3&#x2E;736 max&#x3D;3&#x2E;736 population&#x2D;SD&#x3D;0&#x2E;000 s

baseline passed execution&#x3A; n&#x3D;1 mean&#x3D;3&#x2E;736 min&#x3D;3&#x2E;736 max&#x3D;3&#x2E;736 population&#x2D;SD&#x3D;0&#x2E;000 s

baseline reported primary tokens&#x3A; n&#x3D;1 mean&#x3D;934&#x2E;000 min&#x3D;934&#x2E;000 max&#x3D;934&#x2E;000 population&#x2D;SD&#x3D;0&#x2E;000 tokens

candidate usage completeness&#x3A; 1&#x2F;1 primary totals reported&#x3B; 0 unknown&#x3B; cache breakdown 1&#x2F;1&#x3B; reasoning breakdown 1&#x2F;1

candidate all&#x2D;attempt execution&#x3A; n&#x3D;1 mean&#x3D;3&#x2E;252 min&#x3D;3&#x2E;252 max&#x3D;3&#x2E;252 population&#x2D;SD&#x3D;0&#x2E;000 s

candidate passed execution&#x3A; n&#x3D;1 mean&#x3D;3&#x2E;252 min&#x3D;3&#x2E;252 max&#x3D;3&#x2E;252 population&#x2D;SD&#x3D;0&#x2E;000 s

candidate reported primary tokens&#x3A; n&#x3D;1 mean&#x3D;1043&#x2E;000 min&#x3D;1043&#x2E;000 max&#x3D;1043&#x2E;000 population&#x2D;SD&#x3D;0&#x2E;000 tokens

Variation is descriptive across the configured task mix&#x3B; population SD describes these observations&#x2C; not uncertainty or equivalence&#x2E; Cache&#x2F;reasoning breakdowns are subsets and are not added to primary totals&#x2E;

Mean candidate&#x2D;minus&#x2D;baseline execution time&#x3A; &#x2D;0&#x2E;484 s &#x28;1 both&#x2D;passed pairs&#x29;

Source bytes over 1 both&#x2D;passed pairs&#x3A; baseline 228&#x2C; candidate 228 &#x28;descriptive size&#x2C; not a quality verdict&#x29;

Candidate&#x2D;minus&#x2D;baseline reported tokens&#x3A; 109 &#x28;1 known&#x2D;usage pairs&#x3B; 0 missing&#x29;

## Retained attempts

Index | Task | Repetition | Arm | Status | Quality passed | Execution ns | Grading ns | Input | Output | Cached subset | Reasoning subset | Source SHA-256
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
0 | reuse&#x2D;email | 1 | baseline | passed | true | 3735733167 | 336332250 | 764 | 170 | 9 | 132 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef
1 | reuse&#x2D;email | 1 | candidate | passed | true | 3251692666 | 439028083 | 776 | 267 | 9 | 225 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef

## Behavior and discipline

Decision-opportunity attempts pass only when both the behavior tests and the discipline gate pass.

Index | Task | Arm | Behavior | Discipline
--- | --- | --- | --- | ---
0 | reuse&#x2D;email | baseline | passed | passed
1 | reuse&#x2D;email | candidate | passed | passed

## Matched pairs

Baseline index | Candidate index | Quality eligible | Both-passed execution delta ns | Known-usage token delta
--- | --- | --- | --- | ---
0 | 1 | true | -484040501 | 109

Deltas are candidate minus baseline. Execution deltas include only both-passed pairs; token deltas include pairs with both normalized usage observations, including failures. Unstarted attempts have no invented outcomes.

## Schedule

Index | Task | Repetition | Arm
--- | --- | --- | ---
0 | reuse&#x2D;email | 1 | baseline
1 | reuse&#x2D;email | 1 | candidate

## Evidence and interpretation

Original and replay artifacts are not included. Rederivation requires separately retained originals and the matching evaluator version; this report alone is not replayable. See evidence.json for byte digests and results.json for projected facts. A matched replay describes recorded outcome agreement, not proof of independent execution.

Raw instructions, generated source, traces, diagnostics, environment, paths and native session IDs are omitted. Included metadata still requires human review before publication. Keep hypotheses and the next decision in a separate human-authored interpretation.md.

Classifier decisions and classifier overhead are not recorded by this artifact schema and remain unmeasured. Execution time must not be presented as isolated classifier latency.
