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

A real adapter must invoke its agent, apply `instructions`, capture provider usage, and return the resulting source. Native Claude/Codex/Gemini adapters are outside this first slice. The saved request proves which treatment was requested; it does not prove a skill or tool was actually used. Retain provider-native traces externally before making such claims.

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

## Native Codex A/A calibration

`codex-aa.json` is a **pilot template**, not a live authorization or performance result:

```sh
go build -o bin/ateam ./cmd/ateam
bin/ateam bench run benchmarks/examples/codex-aa.json --dry-run
```

It requests 30 independent attempts (three tasks × five repetitions × two arms), identical instructions, counterbalanced ordering, 120 seconds per attempt, 3,600 seconds for the entire run, and a 100,000 reported-primary-token stop threshold. Replace `MODEL_TO_SELECT` with an explicitly selected model and review these illustrative limits before execution. Native execution refuses that placeholder. The dry run probes `codex --version` locally, hashes its executable, and reports requested model/reasoning, exact schedule, authentication policy and limits; it makes no model call. The installed CLI must support `exec --json --ephemeral --ignore-user-config --ignore-rules --output-schema`; implementation was checked against CLI 0.160.1. Unsupported flags/events fail visibly rather than silently changing the treatment.

For native execution, the optional `codex` object selects the built-in adapter and `command` contains only the CLI executable. Explicit `auth: api-key-env` inherits **only `CODEX_API_KEY`** alongside a small runtime environment. Supply the credential through your environment/secret manager, never in JSON, a command argument, source control or an issue. Existing ChatGPT login, `OPENAI_API_KEY`, user configuration, MCP configuration and project instructions are not imported. Each attempt receives a new workspace, `HOME`, `CODEX_HOME` and ephemeral session; the requested instruction text is sent on stdin. The same API credential is shared across attempts, but model tool shell environments do not inherit it. API-key-only authentication is an intentional first-adapter limit; subscription authentication needs a separately designed isolation policy.

The adapter requests Codex's read-only sandbox and never bypasses approvals. Fresh directories are **not** a security sandbox: Codex still has its documented tool access, and the protected Go grader executes generated source locally. Use a disposable external environment for untrusted agents/tasks. No arbitrary skill bundles are imported in this first adapter; vary explicit instruction text, and distinguish configured instructions from actual observed native item events. Native `item.completed` types show reported activity, not proof that a skill was effective or that a tool action succeeded.

Native JSONL stdout and stderr are bounded to 1 MiB each. Exact inherited key bytes and their JSON-escaped representation are redacted before persistence/parsing; this is not general secret discovery. Treat source, prompts and traces as sensitive. Artifacts use private directories/files (0700/0600), remain local under ignored `benchmarks/runs`, and are retained until you deliberately remove their run directory. Do not publish raw traces without reviewing them. No credential value is copied into resolved configuration.

Reported primary input/output totals are normalized only when both are available and valid. Cache-read and reasoning counts are subsets, never additional tokens. Optional native cache-write counts are retained separately without assuming their billing semantics. Missing/partial or contradictory usage remains unknown. Requested settings and the actual CLI version are recorded; resolved model is unknown unless native evidence supplies it (this adapter currently leaves it unknown), and resolved reasoning is not inferred from the request. No monetary estimate is produced: cost is unknown, and tokens are not an invoice.

`max_total_tokens` stops **starting another attempt** once reported primary totals reach the threshold. A single in-flight attempt can exceed it. Missing/invalid usage stops further calls; timeout/cancellation stops the current process group on Unix. `max_run_seconds` includes grading. These are execution controls, not a hard dollar cap: agree a dollar spending cap and its provider-side enforcement (or explicitly accept a bounded overshoot policy), model, attempt/time limits and trace retention **before a live pilot**. The local runner cannot guarantee a monetary cap. A stopped run exits nonzero, retains `run.json` with a stop reason and schedule prefix, and supports comparison/rescoring, including an unpaired last attempt. Usage on failed executions remains visible. A session ID reused across attempts is rejected and stops the pilot.

Before native execution, the CLI automatically runs and saves the offline evaluator self-test and refuses to proceed if it fails. Dry runs do not repeat self-tests.

After that separate live decision, set `CODEX_API_KEY` externally and execute the reviewed configuration with `bench run`. Use the emitted artifact path:

```sh
bin/ateam bench compare benchmarks/runs/experiment-.../run.json
bin/ateam bench compare benchmarks/runs/experiment-.../run.json --json
bin/ateam bench rescore benchmarks/runs/experiment-.../run.json
```

`mode: aa` requires identical instructions; omitted mode or `ab` requires different instructions. The command/model/reasoning/config are shared across both arms. Reports preserve all statuses and known/unknown usage denominators; they show descriptive mean/min/max/population-SD over the configured task mix, and compare only complete pairs. Different stochastic answers are expected. Neither a small mean difference nor nonsignificance establishes equivalence. Run A/A first to investigate accounting and harness discrepancies; separately authorize any subsequent off/full A/B experiment.

Use [the calibration report checklist](calibration-report.md) to link the immutable run, configuration/provenance, self-test evidence, comparison, rescore and discrepancies. Offline synthetic event fixtures establish parser/runner behavior, not provider billing accuracy. No live calibration has been performed for this implementation.
