# Offline native capture qualification

Addresses #58/#16; follows PR #69. This slice demonstrates a bounded provider-response capture in the actual pinned Claude Code → LiteLLM → local fixture stack. It preserves provider usage before normalization, correlates the observed requests, and refuses incomplete or failed captures. **It does not qualify a production runtime or authorize spending.** Native cancellation did not close the provider connection within the measured two-second window. #58 remains open.

## Measured path and limits

The first candidate failed before any provider request: the Inception route expected an OpenAI SDK client, and `AsyncHTTPHandler` lacked `.chat`. Supplying the installed `AsyncOpenAI` through LiteLLM's pre-deployment callback reached the fixture through HTTPX hooks. A tee-only stream captured the complete synthetic body but observed `closed`, not EOF: the SDK stops at `[DONE]` and closes its response. Neither a DONE marker nor a close proves iterator exhaustion.

The selected research profile wraps the public HTTPX stream, bounds it to 1 MiB/4096 chunks, writes and syncs each chunk before yielding, then calls public `response.aread()` in the response hook. This reads actual iterator EOF before returning the buffered response to the SDK, without patching private response fields or a bridge fork. Identity encoding is required. This changes buffering, latency, backpressure and cancellation behavior; it is an offline accounting instrument, not production streaming qualification. HTTPX documents response hooks as running before body consumption and `aread()` as the async body-read operation ([event hooks](https://www.python-httpx.org/advanced/event-hooks/)); LiteLLM documents custom callback injection ([hooks](https://docs.litellm.ai/docs/proxy/call_hooks)). The exact installed source pins in the evidence, rather than current documentation, govern these observations.

Setting router/bridge/SDK retries to zero was initially insufficient. One native call produced three HTTP 429 dispatches, each independently captured; the effective SDK retry setting was 2 and request headers counted 0, 1, 2. The pinned Inception parameter allowlist drops `max_retries`; the OpenAI handler then defaults it to 2 and overwrites the supplied SDK client's setting. The tested supported override preserves existing `allowed_openai_params`, adds `max_retries`, and passes zero. The final 429 control measured SDK0, retry header0 and one provider request, with no retry-control field leaked into provider JSON. This proves only this pinned path/control, not universal retry suppression.

Relevant pinned LiteLLM source at commit `fa647f742d7baefe8eb1181899d9c81b41559772`: `llms/inception/chat/transformation.py` supported parameters, inherited `llms/openai/chat/gpt_transformation.py` mapping, `utils.py` `_apply_openai_param_overrides`, and `llms/openai/openai.py` `completion`/`async_streaming`/`_get_openai_client`. The installed bytes of each cited source match the retained wheel. OpenAI SDK `_streaming.py` stops at DONE and closes; `_base_client.py` supplies retry behavior. Full file hashes are in the evidence bundle.

## Native evidence

Each row is a fresh native attempt, fresh bridge process/config/storage and owned local synthetic provider. The fixture independently observes requests, capture IDs, request hashes, status and response bytes. All 13 final cases use the same hook and driver hashes. Two deployment callbacks in one native call are not counted as two provider requests.

| Case | Provider requests | Native exit | Recorded end | Capture accepted / usage complete |
| --- | ---: | ---: | --- | --- |
| known | 1 | 0 | EOF | yes / yes: prompt20, completion5, cached3, reasoning2 |
| zero | 1 | 0 | EOF | yes / yes: explicit zeros |
| missing | 1 | 0 | EOF | yes / no: provider usage unknown |
| partial | 1 | 0 | EOF | yes / no: prompt20/cached3 known, completion unknown |
| HTTP 429 | 1 | 1 | EOF | no / no |
| disconnect | 1 | 1 | transport error | no / no; prefix retained |
| cancel | 1 | 143 | transport error | no / no; cancellation blocker below |
| intent failure | 0 | 1 | absent | no / no |
| body failure | 1 | 1 | capture error | no / no |
| completion file sync failure | 1 | 1 | valid-looking EOF JSON | no / no; failure marker vetoes acceptance |
| completion directory sync failure | 1 | 1 | valid-looking EOF JSON | no / no; failure marker vetoes acceptance |
| instrumentation disabled | 1 | 0 | absent | no / no; native success alone insufficient |
| oversized response | 1 | 1 | capture error | no / no; byte limit enforced |

Cancellation uses an explicit provider-request/durable-first-chunk handshake before terminating the CLI. After CLI exit, the fixture observed no provider socket close within two seconds (`connection_closed_after_cli_exit=false`), then deliberately truncated the response. The resulting transport error is not proof of propagated native cancellation. This is a concrete blocker for production qualification; the observation does not claim cancellation can never propagate.

Capture acceptance requires matching unique fixture/capture IDs and request hashes, the exact tested wire controls, effective SDK0, identical response bytes/hash/status and EOF, successful native completion, successful isolation controls, a completed supervisor, and no capture failure marker. The existing PR #69 reader is reused unchanged. Its diagnostic fields are not promoted to complete usage after invalid or failed exchange histories. Missing/partial provider fields remain unknown regardless of numeric CLI/bridge counters. Only known and zero contribute complete primary-usage subtotals (20/5 combined); this is a fixture arithmetic check, not billing.

## Isolation, custody and reproducibility

Pinned stack: Claude Code 2.1.292 (binary SHA256 `97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f`), Python3.14.5/macOS arm64, LiteLLM1.99.0, HTTPX0.28.1 and OpenAI2.54.0. The existing approved isolated 111-wheel installation was cloned and checked; no upgrade or new dependency resolution was performed. Wheel selection remains [the existing lock](offline/selected-wheels.lock). No paid endpoint was called.

Separate macOS OS policies allow the CLI only its fresh home and bridge port, and allow the bridge only isolated dependencies/storage and the fixture port. Fresh child environments contain only fake credentials; home secrets/keychain access and external egress are denied. Both policies passed IPv4/IPv6 external-egress refusal controls, and the native process could not overwrite the capture sentinel. The driver refuses existing managed Claude policy. The native call uses bare/safe/restricted mode, fresh settings, one turn, low effort, 1024 output tokens, no MCP/browser/skills/session persistence and Read/Glob/Grep availability; it requests no tools. This does not test tool execution or permission histories.

The bridge is trusted to write captures; native CLI writes to those files are denied. Intent is persisted before dispatch; chunk writes and completion records use flush/fsync. The failure injections exercise selected intent/body/completion-file/completion-directory failure points, **not power-loss durability or all filesystem failures**. A completion fsync error can leave valid-looking EOF JSON, so the fatal latch, native outcome and failure marker remain necessary. Best-effort failure-marker storage can itself fail. These finite artifacts do not authenticate origin or prove an arbitrary caller-supplied inventory exhaustive.

The driver signals owned process groups and waits for direct CLI/bridge leaders with bounded termination/escalation. All recorded direct leaders exited; a subsequent process inspection found no surviving capture driver/bridge command. The helper does not prove absence of every descendant if a leader has already exited; general process-tree lifecycle qualification remains outside these observations.

[Evidence](offline/capture/capture-observations.json) contains synthetic response text, minimal native/fixture/capture projections and raw-file hashes. Native prompts/full requests/private paths/credentials stay in ignored local storage; bodies over16KiB are represented by length/hash only. Independent technical review reconciled all13 final projections against raw runs. Three earlier trials retain the unsupported client, tee-close and measured retry failures. Their raw hashes are provenance breadcrumbs, not downloadable raw artifacts; reproducing them requires the documented earlier configurations. Historical PR #64/#69 artifacts and frozen Gemini evidence are unchanged.

Final hook SHA256: `3fd98944e892d07564bf9c754ac626a5da977c17e43c9233f6a7d91d64fdf691`.
Final driver SHA256: `18f0e2295947b7e9253a9055114295b6831a0a1d53c869e9c49c1906d08d1269`.
Commands and environment setup: [offline capture README](offline/capture/README.md).

## Checks and review

Decision: use finite deterministic generated grids and explicit native fixtures; no randomized fuzz exploration. The contract is intentionally limited to this fixture set, not a general network parser/runtime.

Rule: refuse missing/duplicate/foreign/mismatched inventory, incomplete transport, invalid controls, failed capture/native completion and fabricated complete usage; preserve exact provider bytes before SDK consumption.

Inputs: all13 actual native cases; four byte partitions; exact byte/chunk limits and one beyond each; eight native-exit/error/failure combinations; three orderings of all13 cases; repeat replay; selected write/sync faults. Each grid asserts its completed count or expected body/limit, so empty iteration cannot silently pass.

Run: `check_capture_hook.py` passes 10 pinned-dependency tests with HTTPX MockTransport and OS-denied networking. `check_capture_replay.py` passes six stdlib tests; `replay_capture.py` checks all13 observations. Two meaningful RED→GREEN corrections are retained locally: malformed `allowed_openai_params=False` must be rejected, and boolean bridge exit must not masquerade as integer zero. No random seed or fuzz exploration is claimed. Exact committed-head repository checks and independent review evidence are recorded in PR #71.

Mutation: copied-source checks produce 12 **detections**, each requiring baseline pass, named assertion failure, after-run pass and unchanged original SHA256. Hook mutations remove retry override, byte cap, pre-yield flush, incomplete termination, fatal latch and pre-SDK buffering. Replay mutations remove capture-failure veto, native-success veto, EOF requirement, control requirement, strict bridge-exit type and request-hash matching. No survivor, invalid or inconclusive result is counted as detection.

Not covered: paid calls, real provider billing, production streaming timing, long-run memory/disk durability, broad retry/fallback paths, full descendant cleanup, tool/permission/skill histories, Codex compatibility and cross-runtime comparisons. Native independent Codex technical review is not stronger-model or cross-service approval; owner-arranged cross-service review remains a merge prerequisite.

## Next boundary

Keep the runtime and spending holds. The next bounded slice should establish provider-request teardown after native cancellation and capture/supervisor failure, including descendant/process-group cleanup and incomplete-accounting refusal, without paid calls. Freeze a canary/A/A spending protocol only after the remaining native histories are qualified and independently reviewed; explicit new spending authorization is still required.
