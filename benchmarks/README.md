# Benchmark experiments

ATeam separates **offline evaluator self-tests** from **paired command-adapter experiments**. The built-in tasks and reference adapter are a pilot for validating the measurement pipeline. Their results are not evidence that ATeam improves real agents.

## Quick start

Build the CLI (`go build -o bin/ateam ./cmd/ateam`), then from the repository root:

```sh
bin/ateam bench --selftest --markdown
bin/ateam bench run benchmarks/examples/offline.json --dry-run
bin/ateam bench run benchmarks/examples/offline.json
bin/ateam bench compare benchmarks/runs/experiment-<id>/run.json
bin/ateam bench rescore benchmarks/runs/experiment-<id>/run.json
```

`run` prints the unique `run.json` path. A completed experiment returns success even if candidates fail; inspect the comparison and status counts for task outcomes. Runner/infrastructure interruption returns nonzero and prints the partial evidence directory. `compare --json` emits machine-readable results. Flags follow the configuration/artifact filename.

The example uses Node only for a tiny **offline reference adapter**. It returns fixed known solutions for path resolution, SQL lookup, and webhook verification. It makes no model/API calls and reports token usage as unknown. Use it to check installation and artifacts, not to compare prompts. The Go CLI itself adds no runtime dependencies; experiments explicitly require their chosen adapter and the Go toolchain for grading.

## Configuration and adapter contract

Copy `examples/offline.json`. Version 1 accepts one command argv, agent/model/version labels, baseline and candidate instruction strings, 1–3 supported tasks, 1–100 repetitions, a nonnegative order seed, and a 1–3600 second adapter deadline. Both arms share the same adapter and task definitions. The first arm alternates across tasks and repetitions; the seed chooses starting parity. This is reproducible counterbalancing, not random sampling.

Commands run without a shell. Executable names resolve through PATH; relative script/file arguments resolve relative to the configuration file. Existing regular file arguments are made absolute and hashed with the executable. Referenced modules, remote models, and files loaded internally by a wrapper are not transitively hashed: pin them in the adapter deployment and version label. All labels are declarations, not provider verification.

On stdin the adapter receives one JSON object:

```json
{
  "version": 1,
  "trial": { "task": "safe-path", "repetition": 1, "arm": "baseline" },
  "prompt": "The complete task package, function signature, and behavior...",
  "instructions": "The selected instruction variant..."
}
```

On stdout it must return exactly one JSON object:

```json
{
  "version": 1,
  "source": "package safepath\n...",
  "usage": {
    "input_tokens": 1200,
    "output_tokens": 240,
    "cached_input_tokens": 800,
    "reasoning_output_tokens": 100
  }
}
```

Usage may be `null` or omitted. If supplied, both primary counts are required. Input includes cached tokens and output includes reasoning tokens; subset counts are never added again. Normalize provider semantics in the adapter. Counts are adapter-reported, not independently audited. The runner does not estimate dollars or count output characters as tokens. Stdout/stderr each have a 1 MiB retained-output limit; exceeding it makes the adapter attempt an error.

A real adapter must invoke its agent, apply `instructions`, capture provider usage, and return the resulting source. The native Gemini adapter described below is the first built-in integration; Claude and Codex can use the generic protocol until dedicated adapters are added. The saved request proves which treatment was requested; it does not prove a skill or tool was actually used. Retain provider-native traces externally before making such claims.

## Isolation, grading, and evidence

Each attempt gets a temporary working directory and configuration home, with a minimal environment. Parent credentials, agent settings and project instructions are not automatically inherited. PATH still identifies installed executables. Credentials/configuration needed by a real adapter must be handled deliberately in its implementation.

**This is not an OS sandbox.** Adapters and generated Go code run with the user's OS permissions. Use trusted commands/candidates for local pilots and an external container/VM for untrusted workloads. Unix execution cancels the entire process group and bounds pipe waits; other platforms kill only the direct child. Background work that deliberately escapes a process group is outside this isolation guarantee.

The grader runs its own tests against the returned source in a separate temporary module; it does not accept agent-written tests. Go module networking/toolchain switching and inherited Go configuration are disabled. Candidate execution is bounded by a 15-second Go test timeout and 30-second process deadline. Status distinguishes passed, failed, adapter-error, timeout, cancelled, and grading-error. Recognized compiler/package errors fail the candidate; unrecognized startup failures remain inconclusive.

The run directory retains resolved configuration, an artifact for every completed attempt, and a complete `run.json`. Each attempt includes the exact request, source, raw adapter output, grader diagnostics, source hash, and separate agent/grading durations. Run metadata records command/script hashes, grader-source/test digest, actual Go driver hash/version, ATeam binary hash, build runtime/platform, and canonical original-configuration digest. Runtime is ATeam build provenance; the separate grader identity describes actual grading.

Artifacts use exclusive writes and unique directories. The complete artifact has a 64 MiB loading/writing limit; reduce repetitions/output if exceeded. Hashes detect accidental corruption, not malicious forgery. Partial histories are retained but cannot be compared as complete experiments. Keep artifacts private when prompts or traces contain sensitive project data.

`rescore` validates evidence, requires matching grader sources/tests and Go driver, then reruns grading from saved source without calling the adapter. It creates a new artifact beside the original, preserves original execution timing, and records new grading timing. It never overwrites the original. Interrupted rescoring does not produce a completed result. Timing refers to the original agent attempt, not the rescore.

## Interpreting comparisons

Quality counts include observed passes in each arm and paired pass/fail improvements/regressions. Pairs containing execution or grading errors/timeouts are separately unevaluable for quality. Full status counts remain visible. A candidate that failed quickly cannot improve the successful-pair execution-time average.

Time is candidate-minus-baseline mean duration among pairs where both passed, with the denominator displayed. Token deltas include pairs with usage on both sides (including known usage from failed candidates); missing pairs are explicitly counted. Failed adapter calls may have consumed unreported tokens, so known-usage totals are not a full bill. Neither metric is an unconditional productivity estimate.

There are no confidence intervals, significance tests, equivalence claims, or automatic winners in v1. Use pilot variability to choose a larger experiment, predeclare quality tolerances and analysis, and validate changes on held-out representative tasks before publishing performance claims. Public small fixtures are vulnerable to memorization and measure only their tested behavior; webhook behavioral tests do not prove constant-time execution.

## Self-tests and development tooling

`bench --selftest` evaluates four Go fixture families and two local continuity checks. The Svelte task is explicitly unsupported/skipped until a real compiler/behavioral grader exists. Continuity checks do not claim mutation detection. Schema-token counts are estimates and hook timing is one local observation, not an end-to-end agent latency distribution.

`--save` writes a unique `selftest-*.json`; the previous overwrite-only `latest.json` convention is removed. `--rescore <selftest.json>` validates the schema/invariants and reruns the installed reference suite. Legacy unversioned self-test reports are rejected.

Repository OpenSpec tooling is pinned locally:

```sh
npm ci
npm run spec:check
node_modules/.bin/openspec show benchmark-experiments --type spec
```

`task spec:check` prefers this local installation. Existing shared agent skill adapters are unchanged.

## Gemini API calibration and harness A/B

The first native adapter uses Gemini CLI **0.43.0**, API-key authentication and **gemini-3.8-flash**. Both arms share one model and generation configuration. The experimental treatment is the explicit harness instruction text: identical for A/A, different for A/B. Routing, init roles, other native providers and arbitrary skill bundles are outside this first adapter. The existing generic command protocol remains available.

The examples are pilot templates, not live authorization or performance results:

```sh
go build -o bin/ateam ./cmd/ateam
bin/ateam bench run benchmarks/examples/gemini-aa.json --dry-run
bin/ateam bench run benchmarks/examples/gemini-ab.json --dry-run
```

Each example requests 30 independent attempts (three tasks × five repetitions × two arms), counterbalanced ordering, 120 seconds per attempt, 3,600 seconds for the run, and a 100,000 reported-primary-token stop threshold. Review these illustrative limits before execution. The dry run probes `gemini --version` locally, hashes its executable, and records the requested model/reasoning, generated settings, exact arguments, schedule and limits without making a model call. Other CLI versions are refused until their configuration/telemetry contract is verified.

An optional `gemini` object selects the adapter; `command` contains only the CLI executable. `auth: api-key-env` passes **only `GEMINI_API_KEY`** alongside a small runtime environment. Supply it through your environment/secret manager, never JSON, arguments, source control or an issue. OAuth login, ADC, other API keys and ambient user/project integrations are not inherited. Each attempt gets a new workspace, `HOME`, `GEMINI_CLI_HOME` and settings; dotenv sentinels prevent parent `.env` discovery. Managed system settings/policies refuse execution instead of being bypassed. The common scaffold and treatment are sent on stdin without arm/repetition labels.

Generated settings disable hooks, skills, extensions, subagents, ambient context and auxiliary model checks. Context compression is configured to trigger at 100% of the model context window. Core generation settings pin the same model and thinking level, four model turns, and 8,192 maximum output tokens per call. Only directory listing, file reading, grep and glob tools are enabled; no approval bypass is requested. Review the actual generated settings in dry-run output. These controlled defaults are shared by both arms; this version varies instruction text, not arbitrary configuration files. Reported tool activity is evidence of CLI observations, not proof a skill was effective.

Fresh directories and restricted tools are **not** an OS security sandbox. The protected Go grader executes generated source locally. Use a disposable external environment for untrusted agents/tasks.

Native JSON stdout and stderr are bounded to 1 MiB each. Inherited key bytes and nested JSON-escaped representations are redacted before persistence/parsing; this is not general secret discovery. Artifacts use private directories/files (0700/0600), remain local under ignored `benchmarks/runs`, and persist until deliberately removed. Review traces before publication. No key value is copied into resolved configuration.

JSON output retains per-model prompt, tool-prompt, candidate, thought and cache counters; stream-json omits thoughts in this CLI version. Normalized input is prompt + tool-prompt, output is candidates + thoughts; cache and reasoning counts are subsets, never added again. Native `input` is uncached prompt, not another primary count. Inconsistent totals, missing fields, truncation and interruptions leave usage unknown. API request/error counters are retained; missing counters or any failed API call make usage incomplete because failed calls have no reliable token accounting. CLI internals default some absent upstream counts to zero, so these remain **CLI-reported** observations, not proof of complete provider billing. Backend-resolved model/reasoning remain unknown; reported model keys must match the requested model or execution stops without grading that attempt. Cost remains unknown; tokens are not an invoice.

`max_total_tokens` stops **starting another attempt** once known primary totals reach the threshold. An in-flight attempt can exceed it. Unknown usage, missing/changed model identity or reused session IDs stop further calls. Timeout/cancellation terminates the process group on Unix; `max_run_seconds` includes grading. These controls cannot guarantee a dollar cap. Agree a monetary cap with provider-side enforcement or an accepted overshoot policy before a live pilot. A stopped run exits nonzero and retains its schedule prefix and stop reason, including an unpaired final attempt.

Before native execution, the CLI automatically runs and saves the offline evaluator self-test and refuses to proceed if it fails. After separately approving live limits, set `GEMINI_API_KEY` externally and run the reviewed configuration. Then use the emitted artifact path:

```sh
bin/ateam bench compare benchmarks/runs/experiment-.../run.json
bin/ateam bench compare benchmarks/runs/experiment-.../run.json --json
bin/ateam bench rescore benchmarks/runs/experiment-.../run.json
```

`mode: aa` requires identical instructions; omitted mode or `ab` requires different instructions. Reports preserve terminal statuses and known/unknown usage denominators, and show descriptive mean/min/max/population-SD over the task mix. Only complete pairs contribute paired comparisons. Different stochastic answers are expected; a small difference or nonsignificance does not establish equivalence. Calibrate A/A first, investigate discrepancies, then run the same-model harness A/B.

Use [the calibration report checklist](calibration-report.md) to link immutable run/configuration, self-test, comparison and replay evidence. Synthetic fixtures establish parser/runner behavior, not provider billing accuracy. No live calibration has been performed.
