# Mercury instruction A/B v1 — frozen protocol

The owner subsequently authorized this one cohort; it completed 12/12 with matching offline regrading. See [the worked results](results.md), [machine-readable summary](results.json) and [evidence manifest](evidence.json). The pre-run protocol below is preserved for interpretation. Its one-time authorization is consumed; no repeat or additional inference is authorized.

Addresses #16; follows #58 / PR #77 and the canary/A/A preparation in PR #85. This request prepares a plan for review. Earlier canary and A/A permissions have been consumed and authorize no A/B calls. No new runtime, dependency, grader, task, native skill, hook or controller is proposed.

## Question and frozen treatment

Does adding the existing six-point ATeam engineering subset change fixture correctness, execution failures, time or token usage relative to minimal guidance in the same Mercury/Claude stack?

- **Baseline:** `baseline.json` supplies `baseline.md`: `Return a correct, minimal Go implementation. Follow the requested JSON schema.` These 78 bytes, deliberately without a trailing newline, exactly match the completed Mercury A/A baseline.
- **Candidate:** `candidate.json` supplies those same bytes, two newline separators and `ateam.md`. The six points cover minimum complete scope, reuse, standard library, readable code, trust-boundary validation and checking requirements. `ateam.md` is copied byte-for-byte from the frozen Gemini v3 treatment; no task-specific hints or solutions were added after Mercury calibration.
- Baseline instruction SHA-256: `957138123c57e7952a4c986fe379c9028b7ad8c9a21ff841506fa3158df5ec6e` (78 bytes).
- Candidate instruction SHA-256: `d1df3b100357d063de22eb5900fd1cc8cfac09f81c991a24faa5e0d6c84a679b` (921 bytes).

The prior [treatment provenance](../gemini-instructions-v3/README.md) adapts the Ponytail and engineering-quality guidance to a single source-file response. This experiment measures this entire added text, including its length. It cannot separate individual sentences or a general verbosity effect. Repository workflow, executable tests by the agent, delegation and exit-summary instructions are omitted. Native skills/hooks remain disabled; staged instruction snapshots do not prove model adherence. No full-harness or cross-runtime effect is claimed.

## Completed calibration supporting this proposal

Both phases used the same pinned runtime and minimal guidance. All provider exchanges returned HTTP 200 with complete raw usage; request controls were `mercury-3`, low effort, 8192 output tokens and retry count zero. Offline compare and independent regrading reproduced every grade and retained response. All eight native sessions were distinct. No failed sample was replaced.

| Separate phase | Passed / attempted | Provider requests | Input + output tokens |
| --- | ---: | ---: | ---: |
| Canary, safe-path | 2 / 2 | 3 | 2,590 + 5,658 = 8,248 |
| A/A, three tasks | 6 / 6 (3 / 3 each arm) | 6 | 4,280 + 9,682 = 13,962 |

These phases total 22,210 primary tokens; cached input and reasoning output are subsets. A/A execution averaged 6.03 seconds, ranging from 2.86 to 13.94 seconds, including runtime/bridge overhead and excluding grading. This is a small pipeline check. The 6/6 quality result suggests a ceiling effect; the proposed A/B may not distinguish correctness at all. Do not pool calibration samples with A/B or use Gemini results as Mercury controls.

Private evidence remains under `benchmarks/runs/` in the preparation worktree:

| Artifact | File SHA-256 |
| --- | --- |
| `mercury-canary/experiment-3633638727/run.json` | `15e166bed37f64473619fdd31d2eb146c202d47a2a189d2611f07927bf49520b` |
| `mercury-aa/experiment-2131329950/run.json` | `6f8a491c4b20a77bb919f753ff7a626b3e8ad3f3f126b1f4a8a00fc5cc258b54` |

The sibling `rescore-1748189762/run.json` and `rescore-585191893/run.json` respectively retain offline regrading. Summaries, authorization records and logs remain private in `.ateam/mercury-canary-preparation/` and `.enjoy-logs/`. No raw traces or account credentials are published.

## Sample, controls and budget

`ab.json` schedules **12 attempts / six matched pairs**: safe-path, sql-user and gitea-webhook, two repetitions each, seed 0. Each pair is the same task/repetition in both arms. The existing deterministic schedule reverses each task's leading arm in repetition two, giving three baseline-first and three candidate-first pairs. It is counterbalancing, not random assignment. The fixed small sample provides two observations per arm/task; no adaptive expansion or replacement samples.

| Control | Fixed value |
| --- | --- |
| Native CLI / bridge | Claude Code 2.1.292 / LiteLLM 1.99.0 |
| Provider / model / requested effort | Inception / `mercury-3` / low |
| Output / provider requests | 8192 requested output tokens per request; maximum four requests per attempt |
| Attempt / whole run time | 30 seconds per attempt; 720 seconds overall including grading |
| Accounted-token stop | 100,000 input + output tokens, checked between attempts |
| Permissions / response | Existing isolated home/workspace, read-only tools, structured single-file source; no permission bypass |
| Evaluator | Existing prompts, protected graders and pinned Go 1.26.4; no changes |
| Retries / fallback | None; no replacement attempt, model fallback or cap increase |

The only within-run treatment difference is the candidate's appended instruction text. Across calibration and A/B, experiment name/mode, bundle provenance, repetitions and run-level budgets also change as declared here; do not interpret cross-phase timing differences causally. Native arguments/settings, task prompts, grader identity and per-attempt controls must match A/A. Shared provider caching is not flushed; preserve cache counts and order effects. Record requested versus reported model/controls; do not invent backend reasoning-resolution evidence.

Proposed new A/B spending target: **$1**, subject to separate explicit authorization even if promotional credits apply. At the regular list rates checked October 21, 2026 ($0.20/M input, $0.02/M cached input, $0.70/M output), 100,000 primary tokens priced entirely as output is $0.07. [Inception pricing](https://docs.inceptionlabs.ai/get-started/models) is the source; recheck rates/account availability before authorization if they have changed. The owner reports free-tier credit, but remaining credit and invoice finality have not been independently verified. Prior 22,210 tokens remain a separate historical subtotal, not an automatic budget transfer.

The token threshold can overshoot on the final in-flight attempt. Missing usage can leave unknown charges. Timeouts close local requests/process groups; remote computation cancellation is unproved. Neither time/token controls nor a $1 target enforce a hard monetary cap. No credit purchase or account-setting change is included.

## Freeze, review and execution boundary

1. Review this protocol, both instruction manifests and exact content, the dry-run schedule/settings, and calibration evidence. Complete repository verification and doctor; serialize verification and any later inference workload.
2. The prepared local config is `.ateam/mercury-ab-preparation/ab.json`. It differs from the portable tracked config only in the two explicit executable paths; the four bundle files beside it are exact byte copies of the tracked files. The resolver requires files beneath the config directory. Recheck all copies/hashes before authorization and launch; retain every snapshot.
3. Reuse the frozen runner from [canary readiness](../../mercury-canary-readiness.md): source `54e9fa1`, runtime sources equal merged `e88be05`, SHA-256 `ef61c27ccb28fbfd3b478bf6ae9fbf1009cf8a461af0f20d5c74c0005e1b4de5`. The prepared local config's canonical digest is `3ab3d3b04c1eac6f9d19f8333e871314fd2e85d055039b7bef07dcbcf6c7fd22`. Full executable/manifest/component/settings/grader/evaluator identities are in the credential-free `dry-run.json` beside it. Preserve the existing isolated bridge environment; hashes do not authenticate all transitive dependencies.
4. Obtain explicit approval for this one 12-attempt protocol. Load only the existing Inception key into the Go runner through the established private environment path. Children receive local tokens. No global configuration change, installation or new credential exposure.
5. After approval, the runner performs its automatic offline self-test before inference. Execute once, retaining every attempt, native envelope and provider exchange. Compare and rescore offline; check source/response/usage equivalence and original artifact hashes. Native sessions must be distinct within the run and disjoint from both calibration phases; the latter is a post-run review, not an automatic cross-run guard.

```sh
# Credential-free preparation only:
env -i PATH="$PATH" .ateam/mercury-canary-preparation/ateam bench run .ateam/mercury-ab-preparation/ab.json --dry-run

# Proposed command; DO NOT EXECUTE without separate A/B authorization:
.ateam/mercury-canary-preparation/ateam bench run .ateam/mercury-ab-preparation/ab.json --output benchmarks/runs/mercury-instructions-v1-ab

# Offline analysis of the retained run:
.ateam/mercury-canary-preparation/ateam bench compare <run.json> --json
.ateam/mercury-canary-preparation/ateam bench rescore <run.json>
```

## Stop rules and predeclared analysis

The runner stops subsequent attempts on execution/identity/session failures, missing or incomplete usage, or exhausted time/token controls. Do not restart a partial run. A genuine graded quality failure remains in the sample and does not stop the runner. Grading infrastructure errors currently do not stop later attempts automatically: retain them, flag the run as unsuitable for quality inference, and require investigation before any future run. No new supervisor or stronger automatic guarantee is implied.

Primary descriptive outcome: per-arm independent grader passes and quality failures, by task and repetition, with scheduled (six/arm), attempted, graded and unevaluable denominators shown separately. Unstarted attempts remain unstarted, never fabricated quality failures. Show all six planned pairs and any missing mate. Report both-pass, baseline-only, candidate-only and neither-pass counts; inspect every discordant source against grader diagnostics without altering the cohort, prompts or tests.

Secondary outcomes: all-attempt statuses/time/usage; paired candidate-minus-baseline execution-time deltas only for both-passed pairs; token deltas only for pairs with complete primary usage, with quality status visible. Report each task/pair, total input/output tokens, cached/reasoning subsets, provider request counts, range and mean, and exact contributing denominators. Report execution and grading separately; do not turn buffered end-to-end time into streaming-token throughput. Cost is a dated price estimate unless billing evidence establishes an actual charge.

No significance, equivalence, automatic winner or general productivity claim is justified by six pairs. If both arms pass everything, report saturated correctness and exploratory cost/time differences only. Treatment wording, task selection, sample size and analysis stay frozen after launch. Any future harder suite or treatment revision needs a new predeclared cohort and authorization. Finishing this plan or a later A/B run does not authorize more inference.
