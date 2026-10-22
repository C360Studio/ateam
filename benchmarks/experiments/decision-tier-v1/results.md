# Benchmark experiment report

decision&#x2D;tier&#x2D;v1&#x2D;ab — ab

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
Configuration SHA-256 | d4d6471a3edd67fa8f0deaf2771832b8c16334efa176cd78c7d0c09ca7c46f4c
Baseline instruction SHA-256 | 957138123c57e7952a4c986fe379c9028b7ad8c9a21ff841506fa3158df5ec6e
Candidate instruction SHA-256 | d1df3b100357d063de22eb5900fd1cc8cfac09f81c991a24faa5e0d6c84a679b
Runner SHA-256 | 171994bb2d725400771bca6c0f5fa5232244fc2a9c53a8498602d4407c1cf979
Evaluator SHA-256 | 2a93e1b6a9a3e02d156fd9719cb90e10ad69caedcea49c6bafdc7d979599d6d2
Go driver SHA-256 | e61851b2c0cde9b9ac4ae044fcce6ba9d55cb15f98ded3964793acaa8632242f
Go driver version | go version go1&#x2E;26&#x2E;4 darwin&#x2F;arm64
Native settings SHA-256 | bd54f1802eb9db701fecc7fdb1a03ddc21e14f70552bb6a402c3bbe6e98af17f
Native arguments SHA-256 | 1bbc5dc1bd6771bae6198660ccbba9f37f2925874420f5f6b0c7a73e6654665d
Command file SHA-256 (paths omitted) | 97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f&#x2C; c2e78b7df0e3b847316955301c13783f6f64db04d979dd69fdbfdcf018eec73c
Tasks | reuse&#x2D;email&#x2C; overbuild&#x2D;slug&#x2C; propagate&#x2D;limit
Budget policy | INCEPTION&#x5F;API&#x5F;KEY only in the Go runner&#x3B; Claude and LiteLLM 1&#x2E;99&#x2E;0 receive local tokens&#x3B; model mercury&#x2D;3&#x2C; low effort&#x2C; 8192 output tokens&#x2F;request&#x2C; max4 provider requests&#x2C; no retries&#x2F;fallback&#x3B; raw provider usage required for every request&#x3B; stop on missing&#x2F;incomplete usage&#x3B; context closes provider requests and process groups are killed on exit&#x2F;cancel&#x3B; max&#x5F;run&#x5F;seconds includes grading&#x3B; max&#x5F;total&#x5F;tokens checked between attempts&#x3B; no hard dollar cap&#x3B; CLI cost ignored

Repetitions: 2; seed: 0; per-attempt limit: 30 s; run limit: 720 s; token ceiling: 100000. Zero run/token limits mean no configured ceiling for command adapters.

## Outcomes and denominators

Scheduled: 12; retained: 12; unstarted: 0. Graded: 12 (10 passed, 2 failed); unevaluable: 0.

Stop reason: none. Replay: matched; changed attempt indexes: &#x5B;&#x5D; (zero-based).

Accounting basis: captured&#x2D;provider&#x2D;wire. Primary usage known for 12/12 retained attempts; unknown for 0. Known primary token subtotal: 15811; cost: unknown. Cached input and reasoning output are subsets, never added again. Partial provider observations without normalized usage are excluded.

Attempts retained&#x3A; 12&#x2F;12 scheduled&#x3B; 0 unpaired

Paired pilot&#x3A; 6 pairs

Baseline passed&#x3A; 6&#x2F;6&#x3B; candidate passed&#x3A; 4&#x2F;6

Quality changes&#x3A; 0 improvements&#x2C; 2 regressions&#x2C; 4 both passed&#x2C; 0 neither passed

Unevaluable quality pairs&#x3A; 0 &#x28;execution&#x2F;grading errors or timeouts&#x29;

baseline&#x2F;passed&#x3A; 6

candidate&#x2F;failed&#x3A; 2

candidate&#x2F;passed&#x3A; 4

baseline usage completeness&#x3A; 6&#x2F;6 primary totals reported&#x3B; 0 unknown&#x3B; cache breakdown 6&#x2F;6&#x3B; reasoning breakdown 6&#x2F;6

baseline all&#x2D;attempt execution&#x3A; n&#x3D;6 mean&#x3D;3&#x2E;824 min&#x3D;2&#x2E;972 max&#x3D;5&#x2E;618 population&#x2D;SD&#x3D;1&#x2E;004 s

baseline passed execution&#x3A; n&#x3D;6 mean&#x3D;3&#x2E;824 min&#x3D;2&#x2E;972 max&#x3D;5&#x2E;618 population&#x2D;SD&#x3D;1&#x2E;004 s

baseline reported primary tokens&#x3A; n&#x3D;6 mean&#x3D;1164&#x2E;167 min&#x3D;931&#x2E;000 max&#x3D;1675&#x2E;000 population&#x2D;SD&#x3D;251&#x2E;696 tokens

candidate usage completeness&#x3A; 6&#x2F;6 primary totals reported&#x3B; 0 unknown&#x3B; cache breakdown 6&#x2F;6&#x3B; reasoning breakdown 6&#x2F;6

candidate all&#x2D;attempt execution&#x3A; n&#x3D;6 mean&#x3D;3&#x2E;832 min&#x3D;3&#x2E;062 max&#x3D;5&#x2E;840 population&#x2D;SD&#x3D;0&#x2E;942 s

candidate passed execution&#x3A; n&#x3D;4 mean&#x3D;3&#x2E;458 min&#x3D;3&#x2E;062 max&#x3D;3&#x2E;878 population&#x2D;SD&#x3D;0&#x2E;346 s

candidate reported primary tokens&#x3A; n&#x3D;6 mean&#x3D;1471&#x2E;000 min&#x3D;1069&#x2E;000 max&#x3D;2370&#x2E;000 population&#x2D;SD&#x3D;438&#x2E;088 tokens

Variation is descriptive across the configured task mix&#x3B; population SD describes these observations&#x2C; not uncertainty or equivalence&#x2E; Cache&#x2F;reasoning breakdowns are subsets and are not added to primary totals&#x2E;

Mean candidate&#x2D;minus&#x2D;baseline execution time&#x3A; &#x2D;0&#x2E;033 s &#x28;4 both&#x2D;passed pairs&#x29;

Source bytes over 4 both&#x2D;passed pairs&#x3A; baseline 1386&#x2C; candidate 1406 &#x28;descriptive size&#x2C; not a quality verdict&#x29;

Candidate&#x2D;minus&#x2D;baseline reported tokens&#x3A; 1841 &#x28;6 known&#x2D;usage pairs&#x3B; 0 missing&#x29;

## Retained attempts

Index | Task | Repetition | Arm | Status | Quality passed | Execution ns | Grading ns | Input | Output | Cached subset | Reasoning subset | Source SHA-256
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
0 | reuse&#x2D;email | 1 | baseline | passed | true | 2971998166 | 368922083 | 764 | 194 | 508 | 133 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef
1 | reuse&#x2D;email | 1 | candidate | passed | true | 3878138875 | 336092000 | 905 | 623 | 502 | 589 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef
2 | overbuild&#x2D;slug | 1 | candidate | failed | false | 5840197875 | 87191541 | 830 | 1540 | 498 | 1474 | fd3485704bf62772eeeebb92b14ffc6657a05669813d13ef8fac5d96ddf0b3fc
3 | overbuild&#x2D;slug | 1 | baseline | passed | true | 5618424334 | 1567900625 | 702 | 371 | 514 | 323 | ff68dc7df5087f832097ee2caf68fbeecc1f50a8c777c1f72a6f7ba9575f0786
4 | propagate&#x2D;limit | 1 | baseline | passed | true | 3077957459 | 1857339292 | 836 | 95 | 520 | 68 | 57668fae7da98be042fd19090903cf54e6cad2ba79b21cdc33788535228c7d27
5 | propagate&#x2D;limit | 1 | candidate | passed | true | 3714080208 | 381630125 | 980 | 524 | 514 | 460 | 822c973ddbee14e79e39108ae7cfc1e36dc6c1a86849d9a142704edcb7072cd0
6 | reuse&#x2D;email | 2 | candidate | passed | true | 3062204209 | 319445833 | 896 | 173 | 497 | 145 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef
7 | reuse&#x2D;email | 2 | baseline | passed | true | 3150356208 | 397165250 | 772 | 319 | 513 | 282 | 875bc674e968a737db345c85ed480f50d96ab43005ecc7b818d62ab48b1d97ef
8 | overbuild&#x2D;slug | 2 | baseline | passed | true | 4763160458 | 391177250 | 696 | 979 | 509 | 917 | 25012cc9cdaf6de971e155e7c8d695ea50cb3bb684389a7956cf613f07620c65
9 | overbuild&#x2D;slug | 2 | candidate | passed | true | 3177393875 | 401516375 | 863 | 264 | 518 | 195 | 066b8fb58f2e5b23f96b4daa78896cf1edbc1898eabdd70e01b7deafc4634da7
10 | propagate&#x2D;limit | 2 | candidate | failed | false | 3322277500 | 308922541 | 993 | 235 | 520 | 204 | 00a1971d07b2a217914d1f9e47acab4f5f74c4237a09b7ec5d7e51331b228b11
11 | propagate&#x2D;limit | 2 | baseline | passed | true | 3363950792 | 323957792 | 822 | 435 | 511 | 381 | 57668fae7da98be042fd19090903cf54e6cad2ba79b21cdc33788535228c7d27

## Behavior and discipline

Decision-opportunity attempts pass only when both the behavior tests and the discipline gate pass.

Index | Task | Arm | Behavior | Discipline
--- | --- | --- | --- | ---
0 | reuse&#x2D;email | baseline | passed | passed
1 | reuse&#x2D;email | candidate | passed | passed
2 | overbuild&#x2D;slug | candidate | failed | failed
3 | overbuild&#x2D;slug | baseline | passed | passed
4 | propagate&#x2D;limit | baseline | passed | passed
5 | propagate&#x2D;limit | candidate | passed | passed
6 | reuse&#x2D;email | candidate | passed | passed
7 | reuse&#x2D;email | baseline | passed | passed
8 | overbuild&#x2D;slug | baseline | passed | passed
9 | overbuild&#x2D;slug | candidate | passed | passed
10 | propagate&#x2D;limit | candidate | passed | failed
11 | propagate&#x2D;limit | baseline | passed | passed

## Matched pairs

Baseline index | Candidate index | Quality eligible | Both-passed execution delta ns | Known-usage token delta
--- | --- | --- | --- | ---
0 | 1 | true | 906140709 | 570
3 | 2 | true | unknown | 1297
4 | 5 | true | 636122749 | 573
7 | 6 | true | -88151999 | -22
8 | 9 | true | -1585766583 | -548
11 | 10 | true | unknown | -29

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
6 | reuse&#x2D;email | 2 | candidate
7 | reuse&#x2D;email | 2 | baseline
8 | overbuild&#x2D;slug | 2 | baseline
9 | overbuild&#x2D;slug | 2 | candidate
10 | propagate&#x2D;limit | 2 | candidate
11 | propagate&#x2D;limit | 2 | baseline

## Evidence and interpretation

Original and replay artifacts are not included. Rederivation requires separately retained originals and the matching evaluator version; this report alone is not replayable. See evidence.json for byte digests and results.json for projected facts. A matched replay describes recorded outcome agreement, not proof of independent execution.

Raw instructions, generated source, traces, diagnostics, environment, paths and native session IDs are omitted. Included metadata still requires human review before publication. Keep hypotheses and the next decision in a separate human-authored interpretation.md.

Classifier decisions and classifier overhead are not recorded by this artifact schema and remain unmeasured. Execution time must not be presented as isolated classifier latency.
