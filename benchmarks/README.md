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
