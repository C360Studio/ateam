# Benchmark experiment report

decision&#x2D;tier&#x2D;v1&#x2D;aa — aa

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
Configuration SHA-256 | 63420e23fec401245ee76deb328db75a5616bf49127a82fbd17beaa3169829de
Baseline instruction SHA-256 | 957138123c57e7952a4c986fe379c9028b7ad8c9a21ff841506fa3158df5ec6e
Candidate instruction SHA-256 | 957138123c57e7952a4c986fe379c9028b7ad8c9a21ff841506fa3158df5ec6e
Runner SHA-256 | 171994bb2d725400771bca6c0f5fa5232244fc2a9c53a8498602d4407c1cf979
Evaluator SHA-256 | 2a93e1b6a9a3e02d156fd9719cb90e10ad69caedcea49c6bafdc7d979599d6d2
Go driver SHA-256 | e61851b2c0cde9b9ac4ae044fcce6ba9d55cb15f98ded3964793acaa8632242f
Go driver version | go version go1&#x2E;26&#x2E;4 darwin&#x2F;arm64
Native settings SHA-256 | bd54f1802eb9db701fecc7fdb1a03ddc21e14f70552bb6a402c3bbe6e98af17f
Native arguments SHA-256 | 1bbc5dc1bd6771bae6198660ccbba9f37f2925874420f5f6b0c7a73e6654665d
Command file SHA-256 (paths omitted) | 97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f&#x2C; c2e78b7df0e3b847316955301c13783f6f64db04d979dd69fdbfdcf018eec73c
Tasks | reuse&#x2D;email&#x2C; overbuild&#x2D;slug&#x2C; propagate&#x2D;limit
Budget policy | INCEPTION&#x5F;API&#x5F;KEY only in the Go runner&#x3B; Claude and LiteLLM 1&#x2E;99&#x2E;0 receive local tokens&#x3B; model mercury&#x2D;3&#x2C; low effort&#x2C; 8192 output tokens&#x2F;request&#x2C; max4 provider requests&#x2C; no retries&#x2F;fallback&#x3B; raw provider usage required for every request&#x3B; stop on missing&#x2F;incomplete usage&#x3B; context closes provider requests and process groups are killed on exit&#x2F;cancel&#x3B; max&#x5F;run&#x5F;seconds includes grading&#x3B; max&#x5F;total&#x5F;tokens checked between attempts&#x3B; no hard dollar cap&#x3B; CLI cost ignored

Repetitions: 1; seed: 0; per-attempt limit: 30 s; run limit: 360 s; token ceiling: 50000. Zero run/token limits mean no configured ceiling for command adapters.

## Outcomes and denominators

Scheduled: 6; retained: 6; unstarted: 0. Graded: 6 (6 passed, 0 failed); unevaluable: 0.

Stop reason: none. Replay: matched; changed attempt indexes: &#x5B;&#x5D; (zero-based).

Accounting basis: captured&#x2D;provider&#x2D;wire. Primary usage known for 6/6 retained attempts; unknown for 0. Known primary token subtotal: 7131; cost: unknown. Cached input and reasoning output are subsets, never added again. Partial provider observations without normalized usage are excluded.

A&#x2F;A calibration&#x3A; identical configured treatments&#x3B; stochastic outcomes may differ&#x2E;

Attempts retained&#x3A; 6&#x2F;6 scheduled&#x3B; 0 unpaired

Paired pilot&#x3A; 3 pairs

Baseline passed&#x3A; 3&#x2F;3&#x3B; candidate passed&#x3A; 3&#x2F;3

Quality discordance&#x3A; 0 candidate&#x2D;only passes&#x2C; 0 baseline&#x2D;only passes&#x2C; 3 both passed&#x2C; 0 neither passed

Unevaluable quality pairs&#x3A; 0 &#x28;execution&#x2F;grading errors or timeouts&#x29;

baseline&#x2F;passed&#x3A; 3

candidate&#x2F;passed&#x3A; 3

baseline usage completeness&#x3A; 3&#x2F;3 primary totals reported&#x3B; 0 unknown&#x3B; cache breakdown 3&#x2F;3&#x3B; reasoning breakdown 3&#x2F;3

baseline all&#x2D;attempt execution&#x3A; n&#x3D;3 mean&#x3D;3&#x2E;082 min&#x3D;2&#x2E;996 max&#x3D;3&#x2E;222 population&#x2D;SD&#x3D;0&#x2E;100 s

baseline passed execution&#x3A; n&#x3D;3 mean&#x3D;3&#x2E;082 min&#x3D;2&#x2E;996 max&#x3D;3&#x2E;222 population&#x2D;SD&#x3D;0&#x2E;100 s

baseline reported primary tokens&#x3A; n&#x3D;3 mean&#x3D;1080&#x2E;667 min&#x3D;942&#x2E;000 max&#x3D;1244&#x2E;000 population&#x2D;SD&#x3D;124&#x2E;519 tokens

candidate usage completeness&#x3A; 3&#x2F;3 primary totals reported&#x3B; 0 unknown&#x3B; cache breakdown 3&#x2F;3&#x3B; reasoning breakdown 3&#x2F;3

candidate all&#x2D;attempt execution&#x3A; n&#x3D;3 mean&#x3D;3&#x2E;379 min&#x3D;2&#x2E;731 max&#x3D;4&#x2E;558 population&#x2D;SD&#x3D;0&#x2E;835 s

candidate passed execution&#x3A; n&#x3D;3 mean&#x3D;3&#x2E;379 min&#x3D;2&#x2E;731 max&#x3D;4&#x2E;558 population&#x2D;SD&#x3D;0&#x2E;835 s

candidate reported primary tokens&#x3A; n&#x3D;3 mean&#x3D;1296&#x2E;333 min&#x3D;918&#x2E;000 max&#x3D;2029&#x2E;000 population&#x2D;SD&#x3D;518&#x2E;166 tokens

Variation is descriptive across the configured task mix&#x3B; population SD describes these observations&#x2C; not uncertainty or equivalence&#x2E; Cache&#x2F;reasoning breakdowns are subsets and are not added to primary totals&#x2E;

Mean candidate&#x2D;minus&#x2D;baseline execution time&#x3A; 0&#x2E;297 s &#x28;3 both&#x2D;passed pairs&#x29;

Source bytes over 3 both&#x2D;passed pairs&#x3A; baseline 1294&#x2C; candidate 1235 &#x28;descriptive size&#x2C; not a quality verdict&#x29;

Candidate&#x2D;minus&#x2D;baseline reported tokens&#x3A; 647 &#x28;3 known&#x2D;usage pairs&#x3B; 0 missing&#x29;

## Retained attempts

Index | Task | Repetition | Arm | Status | Quality passed | Execution ns | Grading ns | Input | Output | Cached subset | Reasoning subset | Source SHA-256
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
0 | reuse&#x2D;email | 1 | baseline | passed | true | 2995844541 | 323379417 | 769 | 173 | 511 | 144 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef
1 | reuse&#x2D;email | 1 | candidate | passed | true | 2731322417 | 357596333 | 778 | 164 | 518 | 140 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef
2 | overbuild&#x2D;slug | 1 | candidate | passed | true | 4558061125 | 372744000 | 683 | 1346 | 499 | 1281 | c5dc43134b4a3ab41d12c3eef7f23ab800a951e90231266d71cf9e5a810c73ce
3 | overbuild&#x2D;slug | 1 | baseline | passed | true | 3028469917 | 361140708 | 704 | 352 | 515 | 284 | 1e5c502aa339047ac8bd67af6810cafeebf283514bcf57afd9e5332682ee6ac0
4 | propagate&#x2D;limit | 1 | baseline | passed | true | 3221614167 | 359923792 | 826 | 418 | 513 | 391 | 57668fae7da98be042fd19090903cf54e6cad2ba79b21cdc33788535228c7d27
5 | propagate&#x2D;limit | 1 | candidate | passed | true | 2847887167 | 359118417 | 824 | 94 | 512 | 66 | 57668fae7da98be042fd19090903cf54e6cad2ba79b21cdc33788535228c7d27

## Behavior and discipline

Decision-opportunity attempts pass only when both the behavior tests and the discipline gate pass.

Index | Task | Arm | Behavior | Discipline
--- | --- | --- | --- | ---
0 | reuse&#x2D;email | baseline | passed | passed
1 | reuse&#x2D;email | candidate | passed | passed
2 | overbuild&#x2D;slug | candidate | passed | passed
3 | overbuild&#x2D;slug | baseline | passed | passed
4 | propagate&#x2D;limit | baseline | passed | passed
5 | propagate&#x2D;limit | candidate | passed | passed

## Matched pairs

Baseline index | Candidate index | Quality eligible | Both-passed execution delta ns | Known-usage token delta
--- | --- | --- | --- | ---
0 | 1 | true | -264522124 | 0
3 | 2 | true | 1529591208 | 973
4 | 5 | true | -373727000 | -326

Deltas are candidate minus baseline. Execution deltas include only both-passed pairs; token deltas include pairs with both normalized usage observations, including failures. Unstarted attempts have no invented outcomes.

## Schedule

Index | Task | Repetition | Arm
--- | --- | --- | ---
0 | reuse&#x2D;email | 1 | baseline
1 | reuse&#x2D;email | 1 | candidate
2 | overbuild&#x2D;slug | 1 | candidate
3 | overbuild&#x2D;slug | 1 | baseline
4 | propagate&#x2D;limit | 1 | baseline
5 | propagate&#x2D;limit | 1 | candidate

## Evidence and interpretation

Original and replay artifacts are not included. Rederivation requires separately retained originals and the matching evaluator version; this report alone is not replayable. See evidence.json for byte digests and results.json for projected facts. A matched replay describes recorded outcome agreement, not proof of independent execution.

Raw instructions, generated source, traces, diagnostics, environment, paths and native session IDs are omitted. Included metadata still requires human review before publication. Keep hypotheses and the next decision in a separate human-authored interpretation.md.

Classifier decisions and classifier overhead are not recorded by this artifact schema and remain unmeasured. Execution time must not be presented as isolated classifier latency.
