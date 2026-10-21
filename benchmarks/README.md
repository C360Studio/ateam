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

A real adapter must invoke its agent, apply `instructions`, capture provider usage, and return the resulting source. The native Gemini, Claude and Codex adapters described below are built-in integrations. The saved request proves which treatment was requested; it does not prove a skill or tool was actually used. Retain provider-native traces externally before making such claims.

## Versioned instruction bundles

Use file-backed bundles to compare real instruction edits while keeping the model, provider, reasoning, schedule and budgets shared. Replace the inline `baseline` and `candidate` strings with explicit manifest references:

```json
"bundles": {
  "baseline": "bundles/baseline.json",
  "candidate": "bundles/candidate.json"
}
```

Each manifest has this version-1 format:

```json
{
  "version": 1,
  "name": "boundary-review-v1",
  "files": ["instructions.md", "boundaries.md"]
}
```

Manifest paths resolve beneath the experiment configuration directory. Each manifest's directory is its bundle root; its ordered `files` resolve beneath that directory. Paths must be canonical relative paths using `/`, with no absolute paths, `.` or `..` segments, backslashes or symlinks. Only explicit regular UTF-8 text files are read. There is no globbing, directory discovery, network fetch, template expansion or execution. A manifest is at most 64 KiB and names 1–32 unique file paths; each arm's effective content is at most 1 MiB, including separators. Inline strings must be empty or omitted when `bundles` is selected.

The effective instructions are the exact file contents joined with two newline bytes (`\n\n`), in listed order. Existing whitespace and line endings are preserved. Names, paths and manifest metadata are retained as provenance but never prepended to instructions. `mode: aa` requires byte-identical effective content; omitted mode or `ab` requires different content. Two names for the same content are a valid A/A control, not an A/B change. Reordering files counts only when the resulting instruction bytes change.

Try the offline examples:

```sh
bin/ateam bench run benchmarks/examples/bundles-aa.json --dry-run
bin/ateam bench run benchmarks/examples/bundles-ab.json --dry-run
bin/ateam bench run benchmarks/examples/bundles-ab.json
```

The reference adapter returns fixed solutions regardless of instruction content. These commands demonstrate resolution, delivery and artifact handling, not prompt effectiveness. To use bundles with a native provider, make the same replacement in its existing experiment configuration; the workflow and shared adapter instruction field are unchanged.

Dry-run output retains the original references, raw manifests, ordered component contents and SHA-256 hashes, and effective content/hashes for both arms. Review the resolved `experiment.baseline` and `experiment.candidate` to see the actual treatment difference. New execution validates the saved snapshot and rereads its explicit inputs before starting attempts; changed or missing files require a fresh resolution. Once resolved for execution, all attempts use the retained bytes. Offline `compare` and `rescore` validate and use saved content without accessing the original bundle files, so they still work after those files are removed.

Resolution refuses bundles whose known snapshot and scheduled requests alone exceed the existing 64 MiB artifact limit, accounting for JSON escaping. Adapter responses can still exceed the remaining artifact space; choose smaller bundles or fewer repetitions for large experiments. Bundle content is copied into retained evidence and delivered to the selected adapter: include only text intended for that provider. Hashes detect corruption, not malicious forgery. Delivery of skill-like text is not evidence that a native skill or hook ran, or that the treatment improved performance.

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

The first native adapter uses Gemini CLI **0.64.0** for new runs, API-key authentication and **gemini-3.8-flash**. Both arms share one model and generation configuration. The experimental treatment is the explicit harness instruction text: identical for A/A, different for A/B. Routing, init roles and native skill/hook execution remain outside the instruction treatment. File-backed instruction bundles use the shared resolution described above. The existing generic command protocol remains available.

The examples explicitly select `gemini.cli_version: "0.64.0"`. Omitted or `"0.43.0"` selections retain historical settings and comparison output for offline replay but cannot start new inference. Recorded CLI version metadata must match the selected contract. To install a benchmark-local CLI without replacing a global installation:

```sh
npm install --prefix .ateam/tools/gemini-0.64.0 --ignore-scripts --no-audit --no-fund @google/gemini-cli@0.64.0
export PATH="$PWD/.ateam/tools/gemini-0.64.0/node_modules/.bin:$PATH"
```

The examples are pilot templates, not live authorization or performance results:

```sh
go build -o bin/ateam ./cmd/ateam
bin/ateam bench run benchmarks/examples/gemini-aa.json --dry-run
bin/ateam bench run benchmarks/examples/gemini-ab.json --dry-run
```

Each example requests 30 independent attempts (three tasks × five repetitions × two arms), counterbalanced ordering, 120 seconds per attempt, 3,600 seconds for the run, and a 100,000 reported-primary-token stop threshold. Review these illustrative limits before execution. The dry run probes `gemini --version` locally, hashes its executable, and records the requested model/reasoning, generated settings, exact arguments, schedule and limits without making a model call. Other CLI versions are refused until their configuration/telemetry contract is verified.

An optional `gemini` object selects the adapter; `command` contains only the CLI executable. New execution requires `cli_version: "0.64.0"` and `output_format: json-schema`, which requests native application/json output with exactly one string `source` property. The strict parser does not repair Markdown fences. Legacy configurations without this field retain their exact settings for offline replay but cannot start new inference. `auth: api-key-env` passes **only `GEMINI_API_KEY`** alongside a small runtime environment. Supply it through your environment/secret manager, never JSON, arguments, source control or an issue. OAuth login, ADC, other API keys and ambient user/project integrations are not inherited. Each attempt gets a new workspace, `HOME`, `GEMINI_CLI_HOME` and settings; dotenv sentinels prevent parent `.env` discovery. The adapter sets `GEMINI_CLI_TRUST_WORKSPACE=true` only in the child process for its newly generated workspace; this session-only trust does not alter global trust files or the parent environment. Managed system settings/policies refuse execution instead of being bypassed. The common scaffold and treatment are sent on stdin without arm/repetition labels.

The optional `gemini.max_output_tokens` field accepts an integer from 1 through 65,536 and applies identically to both arms and both native generation override scopes. Omission or JSON null preserves the historical 8,192 default and absent-field artifact representation. Explicit zero, negative, fractional or excessive values are rejected before native resolution. This is a shared per-call output allowance, not a total-run token or dollar cap; thinking can consume output allowance. Review the effective value in dry-run settings and budget policy. The [v2 instruction pilot](experiments/gemini-instructions-v2/README.md) selects 32,768 after an incomplete v1 response near the old limit.

Generated settings disable hooks, skills, extensions, subagents, ambient context and auxiliary model checks. Stable CLI settings explicitly disable `privacy.usageStatisticsEnabled` separately from telemetry. Context compression is configured to trigger at 100% of the model context window. Core generation settings pin the same model and thinking level, four model turns, and the shared selected output allowance (8,192 by default) per call. Only directory listing, file reading, grep and glob tools are enabled; no approval bypass is requested. Review the actual generated settings in dry-run output. These controlled defaults are shared by both arms; this version varies instruction text, not arbitrary configuration files. Reported tool activity is evidence of CLI observations, not proof a skill was effective.

Fresh directories and restricted tools are **not** an OS security sandbox. The protected Go grader executes generated source locally. Use a disposable external environment for untrusted agents/tasks.

Native JSON stdout and stderr are bounded to 1 MiB each. Inherited key bytes and nested JSON-escaped representations are redacted before persistence/parsing; this is not general secret discovery. Artifacts use private directories/files (0700/0600), remain local under ignored `benchmarks/runs`, and persist until deliberately removed. Review traces before publication. No key value is copied into resolved configuration.

JSON output retains per-model prompt, tool-prompt, candidate, thought and cache counters; stream-json omits thoughts in this CLI version. Normalized input is prompt + tool-prompt, output is candidates + thoughts; cache and reasoning counts are subsets, never added again. Native `input` is uncached prompt, not another primary count. Inconsistent totals, missing fields, truncation and interruptions leave usage unknown. API request/error counters are retained; missing counters or any failed API call make usage incomplete because failed calls have no reliable token accounting. CLI internals default some absent upstream counts to zero, so these remain **CLI-reported** observations, not proof of complete provider billing. Backend-resolved model/reasoning remain unknown; reported model keys must match the requested model or execution stops without grading that attempt. Cost remains unknown; tokens are not an invoice.

Stable reports add an `accounting` record with `basis: cli-reported-best-effort`, a nullable known normalized token subtotal, reported/unknown retained-attempt counts and `cost_status: unknown`. This includes available normalized usage from failed and unpaired attempts; unstarted scheduled attempts are not counted as unknown observations. Partial raw native counters stay retained but do not enter that subtotal. Existing quality/time/failure results and paired token denominators remain visible. Unknown is never zero and a subtotal is not the whole bill.

The actual 0.64.0 offline contract probe showed that omitted upstream usage metadata becomes all-zero native counters. For this version, an all-zero normalized snapshot is conservatively marked partial/unknown, while retaining raw stats, source and any error. Historical parsing remains unchanged. A positive aggregate can still hide missing per-request metadata; these are CLI-reported observations, not certified provider totals. JSON fatal exits still omit usage. No streaming/telemetry workaround, automatic retry or relaxed stop policy is enabled. See [the stable CLI contract evidence](../internal/bench/testdata/gemini064/README.md).

`max_total_tokens` stops **starting another attempt** once known primary totals reach the threshold. An in-flight attempt can exceed it. Unknown usage, missing/changed model identity or reused session IDs stop further calls. Timeout/cancellation terminates the process group on Unix; `max_run_seconds` includes grading. These controls cannot guarantee a dollar cap. Agree a monetary cap with provider-side enforcement or an accepted overshoot policy before a live pilot. A stopped run exits nonzero and retains its schedule prefix and stop reason, including an unpaired final attempt. New operational failures use `attempt-timeout`, `cancelled` or `adapter-error` before consulting missing identity/usage; explicit identity refusals keep their model-specific reason. Older recorded stop reasons remain replayable. Reused sessions are refused before candidate grading.

Before native execution, the CLI automatically runs and saves the offline evaluator self-test and refuses to proceed if it fails. After separately approving live limits, set `GEMINI_API_KEY` externally and run the reviewed configuration. Then use the emitted artifact path:

```sh
bin/ateam bench compare benchmarks/runs/experiment-.../run.json
bin/ateam bench compare benchmarks/runs/experiment-.../run.json --json
bin/ateam bench rescore benchmarks/runs/experiment-.../run.json
```

`mode: aa` requires identical instructions; omitted mode or `ab` requires different instructions. Reports preserve terminal statuses and known/unknown usage denominators, and show descriptive mean/min/max/population-SD over the task mix. Only complete pairs contribute paired comparisons. Different stochastic answers are expected; a small difference or nonsignificance does not establish equivalence. Calibrate A/A first, investigate discrepancies, then run the same-model harness A/B.

A stderr-only `FatalTurnLimitedError` with actual CLI exit 53 identifies the configured local four-turn limit. New artifacts retain optional `native.gemini_failure` details when the strict error envelope is unambiguous; malformed/conflicting envelopes, nonempty stdout, other exits, timeouts and truncated streams do not gain this classification. The reported error session stays separate from normalized session/model identity. Raw streams remain redacted, usage stays unknown, no grade is invented and subsequent attempts stop. Replay re-derives new evidence; historical artifacts without the optional field keep their original interpretation. This is diagnostic evidence, not recovered billing. Complete fatal-exit usage is tracked in [issue #38](https://github.com/C360Studio/ateam/issues/38); streaming output and asynchronous local telemetry are not equivalent complete-usage replacements.

Use [the calibration report checklist](calibration-report.md) to link immutable run/configuration, self-test, comparison and replay evidence. Synthetic fixtures establish parser/runner behavior, not provider billing accuracy. Earlier startup attempts did not complete calibration; see [the first pilot startup report](calibration-pilot-2026-10-19.md). The later [instruction-pilot canary](experiments/gemini-instructions-v1/results.md) returned one response without an API error, but incomplete JSON near the requested output limit stopped it before A/A or A/B. The [v2 canary](experiments/gemini-instructions-v2/results.md), with a larger explicit output allowance, retained one passing response and one turn-limit failure with unknown usage. The fresh stable-CLI [v3 pilot](experiments/gemini-instructions-v3/results.md) passed its canary and completed A/A; A/B stopped after six of twelve attempts on another four-turn-limit failure with unknown usage. No instruction-treatment winner is established.

## Claude Haiku API calibration

The second native adapter uses **Claude Code 2.1.292**, **claude-haiku-5-5**, and API-key authentication. This is a separate same-model calibration; Gemini and Claude results are not mixed into a harness A/B comparison. Codex Luna uses the separate native adapter described below.

```sh
bin/ateam bench run benchmarks/examples/claude-aa.json --dry-run
bin/ateam bench run benchmarks/examples/claude-ab.json --dry-run
```

The `claude` configuration object explicitly selects `auth: api-key-env`, shared `reasoning: low`, and `output_format: json-schema`. Select exactly one native provider per experiment. The examples retain the 30-attempt, 100,000-token, 120-second/attempt and one-hour/run limits. Dry runs need no API key and invoke only the installed CLI version probe, not a model.

Supply `ANTHROPIC_API_KEY` through the environment/secret manager, never the experiment JSON. Bare mode prevents subscription/OAuth/keychain fallback and ambient instruction, hook, skill, plugin and MCP discovery. Each attempt has a fresh HOME and CLAUDE_CONFIG_DIR; no session persistence or resume. Restricted mode exposes Read/Glob/Grep with explicit read-only permission rules, plus the native StructuredOutput return tool. Generated settings and arguments are recorded. The child requests four agentic turns, 8192 output tokens per request and zero configured API retries and one structured-output attempt (MAX_STRUCTURED_OUTPUT_RETRIES=1); the CLI documents the output setting as applying to most requests, so it is not a hard total budget. Credentials and escaped representations are redacted as for Gemini. CLI stdout/stderr are bounded; these controls are not an OS sandbox.

Local managed settings/drop-ins/MCP and macOS managed preferences refuse execution. Windows Claude execution is refused until registry-policy inspection is supported. Remote managed settings remain enforced by Claude Code. **Before a live pilot, record an isolated `claude doctor` remote-policy result with the same API credential, and review any applied policy; local checks and bare mode alone do not prove absence of remote configuration.** Do not disable policy fetching to manufacture isolation. [Claude bare mode](https://code.claude.com/docs/en/headless) and [server-managed settings](https://code.claude.com/docs/en/server-managed-settings) document these boundaries.

Claude accounting differs from Gemini: the whole-call `modelUsage` map includes auxiliary requests, whereas top-level `usage` covers the main agent loop. Normalize each model's **inputTokens + cacheCreationInputTokens + cacheReadInputTokens** as input and **outputTokens** as output. Never add the overlapping top-level totals. Cache reads and reported thinking are subsets; absent thinking stays unknown. Missing primary model counters leave normalized usage unknown; execution-crash/API-error outcomes cannot claim complete accounting. The CLI receives a native JSON schema requiring a source string, and the adapter reads only `structured_output`. It never falls back to prose or strips Markdown fences. Old text-output artifacts remain replayable with their original failure outcomes, but their configurations cannot start new executions. A success requires clean terminal status and nonempty structured source JSON, and every reported model must match the configured model. Raw counters and the optional `total_cost_usd` are retained; the latter is a **client-side estimate**, not provider billing or a spending guarantee. [Official accounting semantics](https://code.claude.com/docs/en/agent-sdk/cost-tracking).

The common stop rules, self-test gate, comparison and offline immutable replay apply unchanged. API credential availability, policy preflight and a successful live A/A remain required before declaring Haiku calibrated. The $2 target is an operator-approved pilot policy, not a new automatic spending authorization embedded in the example.

The [Haiku A/A pilot](haiku-calibration-pilot-2026-10-19.md) completed 30 passing attempts and 15 evaluable pairs. It records descriptive variation, immutable replay and billing limits; it is not a harness-performance improvement claim.

### Codex Luna through the API

Use `benchmarks/examples/codex-aa.json` for identical treatments or `codex-ab.json` for different instruction strings. Both arms share `gpt-6-luna` and explicit low effort. The adapter requires Codex CLI 0.160.1 and `OPENAI_API_KEY`; supply the key through the process environment, never the experiment JSON. Run the same commands as the other providers:

```sh
ateam bench run benchmarks/examples/codex-aa.json --dry-run
ateam bench run benchmarks/examples/codex-aa.json
ateam bench compare <run.json>
ateam bench rescore <run.json>
```

The adapter uses native app-server **stdio** because `codex exec --json` omits model/effort identity. It validates the native thread/start response before requesting inference and records **CLI-resolved** model, effort and provider. Provider-reported backend identity remains unknown; this is a different assurance source from Claude modelUsage. Any observed reroute is refused. Structured source comes from native outputSchema, with no prose/fence repair.

A fixed `benchmark_openai` provider points to `https://api.openai.com/v1`, reads only the explicit API key and disables both HTTP and stream retries. Fresh HOME/CODEX_HOME, an ephemeral thread and controlled configuration prevent subscription/session inheritance. Host-managed policy is retained or refused, not bypassed. Shell and external capabilities are disabled, native read-only/no-network sandbox settings are checked before inference, and unexpected tool items stop execution. The complete native tool-definition inventory is not independently verified; arbitrary skills/hooks are outside this initial calibration.

The examples schedule 30 attempts with a 100,000 reported-token threshold, 120 seconds per attempt and one hour overall. Codex has no verified per-request output or model-turn cap in this adapter. Token limits stop between attempts, and elapsed-time cancellation does not prove a provider request was unbilled; the $2 pilot policy is a target, not a hard guarantee. A bounded canary must validate installed native behavior before the full live pilot.

Native thread usage contains cumulative `total` and per-request `last` snapshots. The latest complete cumulative total is normalized once; cached input and reasoning output are subsets and are not added again. Missing or invalid accounting stays unknown, and native failures remain visible during replay. Resolved settings, native events, usage and observed item types do not prove that a skill or tool was used. See the [official app-server contract](https://learn.chatgpt.com/docs/app-server) and [Luna model documentation](https://developers.openai.com/api/docs/models/gpt-6-luna).

The [Luna A/A pilot](luna-calibration-pilot-2026-10-19.md) completed 30 attempts with 26 passes, four retained quality failures and complete usage. Both identical arms passed 13/15; this is calibration evidence, not a harness improvement. Its explicitly revised 275,536-token threshold accommodates the observed native context size; the example's illustrative 100,000-token threshold may stop before 30 attempts.

## Mercury through the existing Claude runtime

`examples/mercury-canary.json` and `examples/mercury-aa.json` select `claude.mercury.bridge` (an explicitly installed LiteLLM1.99.0 executable), fixed `mercury-3` and low effort. Claude remains pinned to2.1.292 and uses the same structured single-file task interface. Set the bridge path to the isolated installed executable; this adapter does not install dependencies. Dry-run probes executable versions and hashes, but starts no bridge or inference:

```sh
bin/ateam bench run benchmarks/examples/mercury-canary.json --dry-run
```

Execution requires `INCEPTION_API_KEY`, retained only in the Go runner. Claude and LiteLLM receive a fresh local token/base URL; direct Claude continues to require `ANTHROPIC_API_KEY`. Go owns both process groups, kills/waits for the bridge, and closes/waits for its capture proxy on every exit. Attempt cancellation directly cancels the outbound provider HTTP request. The existing OS-isolation limits above still apply.

An in-process Go loopback proxy records bounded provider responses before translation. LiteLLM remains an external dependency; no custom Python callback is loaded. Config explicitly forwards `max_retries: 0`, and the proxy rejects SDK retries and further requests after a failed exchange before dispatch. Go validates the journal and derives primary/cache/reasoning usage; missing or incomplete usage fails the attempt and stops the run. Native CLI counters and dollar estimates are not authoritative. `run.json` retains an envelope of native output and provider journal so compare/rescore validate the same accounting without a live bridge. The proxy buffers responses; do not interpret its timing as unmodified streaming performance. Wrapper-loaded dependencies are not transitively hashed; retain the pinned isolated environment alongside executable provenance.

General multi-turn tool/permission qualification is deferred; the actual `StructuredOutput` tool path required by these tasks is tested offline. Paid runs require separate approval of the [canary/A/A proposal](mercury-canary-plan.md).

The authorized Mercury canary and A/A subsequently passed 2/2 and 6/6. The [instruction A/B v1 protocol](experiments/mercury-instructions-v1/README.md) records calibration evidence and proposes a separately authorized 12-attempt comparison of minimal guidance with a fixed ATeam subset; planning does not authorize execution.
