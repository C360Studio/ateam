# Compatibility audit and plan

## Decision and scope

The first candidate remains **Claude Code → local LiteLLM Anthropic Messages translation → Inception Mercury 3 Chat Completions**. It is not yet qualified. Static inspection identifies concrete blockers in the documented launcher and translation contract; the stock launcher is unsuitable as an ATeam fixed-control execution contract. No bridge was installed or run and no model request was made during this investigation.

This is a planning claim, not permission to add a fourth runtime or a replacement agent loop. Existing adapters and the frozen Gemini v1/v2/v3 evidence remain untouched. Codex is blocked for this slice by ATeam's fixed OpenAI/Responses contract; that does not prove no future translation path could work. Two qualified paths are still required for #59.

## Reconciled baseline and ownership

On October 20, 2026, local main was clean at `e37563b688182fb975e1013d28231fccec66a747`; remote main and merged PR #60 were `c660e71eb0b4f1d99d3049e7ba1d48dd28119ab3`. Pickup reported a possible Claude writer on main and an unrelated older handoff. No existing worktree was taken over. This new `codex/mercury-offline-qualification` worktree was fast-forwarded to remote main before edits.

PR #43 remains open at `64770752c23814b26dffa7d0824fbe110af407f1`; PR #32 remains open at `76d3d87d88b11d723d08a2e80b49e32f27ff2607`. Their current bodies and October 20 review comments now record Claude `fable-5.1` approval, unlike the stale roadmap/handoff assertion that cross-agent review was missing. API check status during pickup showed Merge check SUCCESS and Verify IN_PROGRESS; historical body text reports earlier Verify passes. Do not equate historical passes with completion of an in-progress run. Neither PR is merged by this task. Reconcile their final landing before implementation depends on their changes.

PR #29 claims #28 on `codex/benchmark-native-skills`; do not duplicate that work. No `.agents/review-policy.json` exists at this baseline, so no configured stronger-model ranking can be asserted. Independent same-session-model review of this plan is not cross-provider review or approval of a future code PR.

Publication refresh: PR #32's Verify, Merge check and Required checks are now SUCCESS at the same head. PR #43 advanced to `92a0c1422c8303fbb0a4b9d423e749b3a67be1ca`; Merge check is SUCCESS and Verify is IN_PROGRESS. Its prior review cited above belongs to the older head and is not asserted as exact-head review of this update. Both remain open; remote main remains `c660e71`. These observations are snapshots, not promises about subsequent landing.

## Audited execution stack

| Layer | Evidence / proposed pin | What remains unknown |
| --- | --- | --- |
| ATeam | Main `c660e71`; existing native Claude contract in `internal/bench/native.go` | New integration schema is not implemented |
| Native CLI | Installed `claude --version`: `2.1.292 (Claude Code)`, matching ATeam's current pin | Actual Mercury request shape, custom-model effort and safe/bare gateway behavior |
| Interpreter | Installed Python `3.14.5`; source package declares `>=3.10,<3.15` | Dependency resolution and supported installed environment |
| Bridge | Documented `litellm[proxy]==1.99.0`; upstream tag resolves to `fa647f742d7baefe8eb1181899d9c81b41559772` | Wheel/source equivalence, hashes and complete transitive lock; no installed bridge detected on PATH |
| Launcher | Inception's `mercury-cc` documentation inspected October 20, 2026 | Mutable documentation is not an immutable executable contract |
| Provider/model | `https://api.inceptionlabs.ai/v1/chat/completions`, `mercury-3`; proposed alias recorded separately | Provider model revision is not exposed/pinned by the alias |
| Incoming API | Anthropic-compatible `/v1/messages`; example header `anthropic-version: 2023-06-01` | Full CLI beta-header and streaming contract |
| Outgoing API | Inception Chat Completions documentation retrieved October 20, 2026 | No immutable API revision advertised; freeze contract snapshot/hash before execution |

The installed Claude executable's observed SHA-256 is `97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f`; recheck it before a later run. This identifies the inspected local binary, not an independently authenticated vendor distribution.

This is a source-audit stack, not a completed dry-run manifest. The [Inception guide](https://docs.inceptionlabs.ai/integrations/claude-code) documents the bridge and states that Anthropic does not support non-Claude routing. Record that support limitation without inventing endorsement or a terms-violation finding. The [Claude gateway guide](https://code.claude.com/docs/en/llm-gateway) is a protocol reference, not certification of Mercury.

## Concrete findings from pinned source

All LiteLLM source links below are bound to `fa647f742d7baefe8eb1181899d9c81b41559772`, resolved with `git ls-remote`. Files were downloaded as data into ignored `.ateam/mercury-audit/`; no Python module was imported. These are static findings and test hypotheses, not observed end-to-end behavior.

1. **Unsupported controls can disappear.** The documented launcher enables `--drop_params`. In pinned [`utils.py`](https://github.com/BerriAI/litellm/blob/fa647f742d7baefe8eb1181899d9c81b41559772/litellm/utils.py), `get_optional_params` removes unsupported nondefault fields when dropping is enabled, otherwise raises. The [Inception allowlist](https://github.com/BerriAI/litellm/blob/fa647f742d7baefe8eb1181899d9c81b41559772/litellm/llms/inception/chat/transformation.py) includes reasoning and output controls but excludes `parallel_tool_calls`. Removing the flag alone is insufficient: loss can happen in earlier translation.
2. **Effort is conditional, not a label equivalence.** In the [Messages adapter](https://github.com/BerriAI/litellm/blob/fa647f742d7baefe8eb1181899d9c81b41559772/litellm/llms/anthropic/experimental_pass_through/adapters/transformation.py), `_translate_thinking_to_openai` returns without mapping if `thinking` is absent. `output_config` is removed from untranslated fields, so `output_config.effort` alone does not survive this path. Adaptive thinking can use that effort; disabled thinking maps to `none`, not Mercury's `instant`; enabled budgets are bucketed. Do not claim `claude --effort low` or thinking-disabled is an effective Mercury control without captured requests. Reject unsupported values rather than silently mapping them.
3. **Output-budget naming disagrees with the guide.** The Messages adapter copies `max_tokens` through; Inception inherits [`OpenAILikeChatConfig.map_openai_params`](https://github.com/BerriAI/litellm/blob/fa647f742d7baefe8eb1181899d9c81b41559772/litellm/llms/openai_like/chat/transformation.py), whose default rewrites `max_completion_tokens` to `max_tokens`. The integration guide says direct Inception requests use `max_completion_tokens`. The API reference still includes deprecated `max_tokens`, so this is a contract discrepancy, not proof the server rejects it. Prove the outgoing field and endpoint semantics before reporting an enforced cap.
4. **Tool and schema mapping need boundary tests.** The Messages adapter translates tool names/choices and either `output_format` or `output_config.format` into `response_format` (legacy field takes precedence). Its tool-choice mapper does not preserve `disable_parallel_tool_use`. Named-tool truncation and reverse mapping need collision tests. Mercury supporting parallel calls does not establish preservation of the CLI's requested tool policy. Streaming assembly, tool-result correlation and Claude's structured-output terminal event are not verified.
5. **Translated usage cannot establish completeness.** The adapter uses `or 0` for missing primary counts, clamps input minus cache subsets at zero, and can emit zero input/output for a stream chunk without usage. It maps nested cache details but does not preserve reasoning-token details in the Anthropic usage object. ATeam must retain provider-side presence/shape evidence before conversion and mark missing/inconsistent counts unknown or invalid. Do not sum Claude aggregate usage and provider totals as separate billing buckets. CLI cost estimates are not Inception billing.
6. **Launcher isolation is weaker than an experiment contract.** Static launcher inspection shows inherited environment, shared home/config, a shared port, readiness based on a model-name match, persistent logs and a PID-file stop. This does not verify ownership of a pre-existing listener or PID. Unsetting the Anthropic API key does not sanitize other credentials, endpoint overrides, proxy variables or subscription state. Loopback binding and the dummy token do not by themselves prove authenticated bridge isolation. The plan must use an owned process and fresh allowlisted environment, with credentials only in the intended process.
7. **A top-level version is not a dependency lock.** Pinned [`pyproject.toml`](https://github.com/BerriAI/litellm/blob/fa647f742d7baefe8eb1181899d9c81b41559772/pyproject.toml) includes ranges and proxy extras. Before a later installation, inspect distributions, build hooks and transitive hashes, then review a reproducible environment. Do not execute the documentation's installation commands during this claim.

Audit file SHA-256 values for reproduction:

| File, relative to upstream root | SHA-256 |
| --- | --- |
| `litellm/llms/inception/chat/transformation.py` | `235c996797b9022c31ccf31199079991e73b346b5fe10a59484e14ab85ab1d0f` |
| `litellm/llms/anthropic/experimental_pass_through/adapters/transformation.py` | `e3384d13103fb73d4e10dc18bb1d1891b0d1e2de95e45b3feaccd9539f7ddcb9` |
| `litellm/utils.py` | `6265c8128c49d03574b951f0313c522b9f16b5563c51c1b75edddc05d4829d4b` |
| `pyproject.toml` | `efcf6b73f8b11116fa4a9e53e0aad26c5580ff904ec5c2c1bd6cb11193a471d3` |

## Repository integration boundary

Read scope: `internal/bench/{experiment.go,native.go,claude.go,claude_evidence.go,claude_events.go,codex.go}` plus archived Claude qualification design and Taskfile gates. Follow-up callers/tests/fixtures include native resolution, `executeClaude`, `parseNativeOutput`, experiment execution/replay, Claude parser/process/control-flow tests, native usage fixtures, CLI dispatch, benchmark examples and benchmark spec. No changes to these files are proposed in this planning PR.

`experiment.go` requires Claude Haiku or Codex Luna for their respective native configurations. `claude.go` requires an Anthropic key and an environment that excludes alternate endpoints; its telemetry contract assumes Claude. `codex.go` fixes the OpenAI base URL and Responses wire API. Changing an environment variable or substituting an executable cannot qualify Mercury through either current adapter.

The smallest later integration should extend the existing native-Claude execution boundary with explicit provider/bridge identity and a separately versioned evidence contract, reusing scheduling, graders, budgets and replay. Whether this is a new config variant or a separate dispatch case is deferred until offline wire evidence resolves the blockers. Do not disguise Mercury as the current Claude model or use generic command mode to bypass native validation. Preserve historical artifacts byte-for-byte.

## Offline qualification sequence for a later authorized slice

1. Freeze CLI/build, audited distribution hashes and dependency lock, Python, bridge/config, API documentation snapshots, model alias and requested controls. Review any managed policy without overriding it. Bind fresh workspace/config, secret-redacted provenance and all attempt IDs before execution.
2. After installation authorization, run the actual pinned CLI and bridge against a loopback capture stub with fake credentials and OS-enforced external-network denial. Never route the stub to Inception. Assert no contact with subscription services, telemetry, update endpoints or inherited proxies; preserve existing managed policy and normal user settings. Refuse if the environment cannot enforce the boundary.
3. Capture both sides of every translated request, with correlation IDs and counts. Fix one effective effort and output allowance only after proving their mapping. Disable dropping, retries, fallback, background/helper inference, auto-compaction and remote tools where supported; where a required behavior cannot be disabled or measured, return a blocker. Do not weaken permissions to make the probe succeed.
4. Use bounded fixture histories for text, structured output, parallel tool calls, named tool selection, tool errors, missing usage, cached/reasoning subsets, truncation, duplicate/conflicting IDs, malformed SSE, HTTP 401/402/429/5xx, midstream disconnect, timeout and cancellation. Verify no hidden request after failure and termination of owned CLI/bridge children. Exercise denied writes/shell/network and disallowed reads through actual runtime permissions, not a mock assertion about flags.
5. Retain redacted raw fixture bytes and independent expected values. Compare provider, bridge and CLI accounting once per request/history; unknown remains unknown even when translated counters are zero. Keep error and partial histories in offline replay; identity aliases never substitute for provider-reported identity. Test tampering, source removal, stale hashes, replay idempotence and backward compatibility.
6. Use generated checks for interacting counters, ID/tool mappings and streaming order, compared with a test-owned reference model. Deterministic seeds must activate boundary assertions. Prove regressions fail by nondestructive mutations: drop effort, change output cap, permit an inherited credential, retry once, replace missing usage with zero, alias an unexpected model and ignore a cancellation. Record detection/survivor/invalid/inconclusive. These are future acceptance checks, not tests claimed as run here.
7. Reuse immutable instruction snapshots. Distinguish requested/staged bytes from observed CLI loading. Current bare Claude mode disables discovered customization; choose and review an explicit instruction delivery contract without calling it full bootstrap activation. Native skill treatment stays owned by #28; report hooks/skills as requested, staged, advertised or observed executing, with unknowns retained.

If these pass, obtain independent change review and repository verification before presenting an installation/runtime qualification as complete. Missing provider-visible usage cannot be repaired solely from Claude's normalized terminal event; that is a valid blocking outcome.

## Live advancement boundary

No numeric live budget is inherited or authorized here. After offline qualification, propose a separately frozen canary then A/A protocol with an explicit new spending target, attempt/token/time/output limits, request/retry accounting and overshoot disclosure. Require identical policy/model/effort/task/permissions/budget across A/A arms; retain all failures and unknown charges, stop on unknown usage, and forbid automatic retry/fallback/cap increases. Advance only after explicit authorization and independent gate review. A/B instructions and #59 cross-runtime comparisons require their own subsequent declarations; no performance claim follows from this plan.
