# Interpretation: decision-opportunity tier v1

Human-authored. The generated `results.md`, `results.json` and `evidence.json` are facts; this file is the reading of them and the next decision.

## What the cohort shows

This is the first ATeam cohort in which the two arms differ in outcome. On the three prescribed tasks (`mercury-instructions-v1`) both arms passed 6/6; here the baseline passed 6/6 and the six-point candidate passed 4/6.

The two candidate failures are different in kind:

- `overbuild-slug`, repetition 1: the returned source was unparseable. The doc comment ran into the `func` keyword (`trimmed andfunc Slugify`), so both the behavior tests and the discipline gate failed on a parse finding. This is a generation or structured-output corruption, not a scope or reuse decision. It cannot be attributed to the instruction text on one observation.
- `propagate-limit`, repetition 2: the constant was raised to 24 and the error message still said 16. Behavior passed, the discipline gate failed. This is exactly the propagation miss the task was built to detect, and it occurred in the arm whose instructions say to finish every behavior the task requires.

The four both-passed pairs show no time difference (−0.033 s mean) and near-identical source size (1,386 versus 1,406 bytes). Across all six usage pairs the candidate used 1,841 more primary tokens, a 26% higher per-attempt mean, in the same direction as the +22.7% seen on the prescribed tasks.

## What it does not show

- The A/A, with identical instructions in both arms, differed by 647 tokens over three pairs and 0.297 s per pair. The A/B token delta is in the same direction and larger per pair, but six pairs cannot separate it from that spread.
- Two failures in six candidate attempts, one of them a parse corruption, is not evidence that the six-point subset harms quality. It is evidence that the tier can register a difference, which the prescribed tasks could not.
- `reuse-email` was passed by every attempt in every stage with the helper reused; at this difficulty it will not discriminate on Mercury.
- The discipline gate reads candidate syntax only. Values computed from a stale constant and dot imports are not detected.
- Mercury 3 at low effort is one model; nothing here transfers to another model or runtime.

## Next decision

1. Keep the tier. It is calibrated on this stack: the A/A passed cleanly, usage was complete on every attempt, and replay matched.
2. Before the `auto` arm of #95, run a larger baseline-versus-full cohort on the same three tasks: five repetitions (30 attempts, roughly 40,000 tokens, still under a dime at list price) with the same frozen controls. Predeclare that an unparseable response is reported as its own status row, separately from decision failures, so the two failure kinds above are not pooled.
3. Consider a harder reuse task for v2 of the tier (for example a helper whose name does not advertise its purpose), since `reuse-email` saturated.
4. No winner, significance, equivalence or ROI claim follows from this cohort.
