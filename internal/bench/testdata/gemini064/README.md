# Gemini CLI 0.64.0 protocol evidence

These are native outputs from the official npm CLI 0.64.0 running with synthetic `--fake-responses` inputs, not live model results or quality samples. Only session UUIDs were replaced with `fixture-session-064`. Credentials were synthetic; no user key or live provider request was used. The source files returned by the fake generator are protocol fixtures, not independently passing solutions.

The isolated package was installed with lifecycle scripts disabled under the ignored `.ateam/tools/gemini-0.64.0` directory. Npm package integrity: `sha512-FHrZ1NPf9QszyVMaa/BP3SsPy40ogBLnbZNWkv/PMi1Uz2zoDNMLGiBzBGmx4Bl7iLRq01K9PY/5kmloXFTpsA==`. Actual entry SHA-256: `ab113d84c8f1692fd6c95081366718897876d8664328c7bf0ccf9437d688145f`; imported CLI chunk: `7702da31cf099ca3e1d64fd195f15b7726b2c2829ec21e955558eb9dbfba47da`; core chunk: `d566e81b1df8d4d44a91bf7aeea96a58ec0e6c8934fa424315ef060507dcbb8c`.

## Native observations

- `success.json`: one request, 100 prompt + 10 candidate + 15 thought tokens, 80 cached input. Expected normalized input/output: 100/25. Native role counters duplicate the parent model counters and must not be added again.
- `tools.json`: two fake responses around one actual local read-only directory-listing tool call. Expected normalized input/output: 307/70, with cache130 and thoughts40 already included as subsets. Seven tool-prompt tokens are primary input.
- `missing-usage.json`: upstream fake response omitted usageMetadata. The CLI emitted every token counter as zero despite a nonempty completion and one API request. New-version parsing therefore keeps raw/source evidence but treats ambiguous zero snapshots as unknown. Legacy interpretation is unchanged.
- `turn-limit.stderr.txt`: four configured turns exhausted, exit53, empty stdout and duplicate bare/prefixed fatal errors on stderr, with no usage report. Failure charges are not reconstructed.

## Probe method and limits

The actual CLI selected FakeContentGenerator before creating a provider-backed generator. Fresh HOME/workspace/config/temp, empty dotenv sentinels, synthetic key, child-only trust and explicit privacy/telemetry settings were used. A probe-only Node preload blocked network transports while allowing bundled data-URL WASM; an observer captured generator requests without changing responses. These helpers are not production adapter options or an OS sandbox.

Captured requests confirmed the requested model, LOW thinking, 8192 output allowance, required-source JSON schema, four read-only tools and four-turn scheduling. This verifies interpreted CLI configuration, not backend enforcement. The probe disabled launcher relaunch/memory tuning for instrumentation; it did not prove every production startup path, live authentication, retry behavior or cross-platform isolation.

Local reproducibility evidence remains in `.enjoy-logs/gemini064-probe.py`, `gemini064-contract-report.md`, `gemini064-request-summary.json`, and per-probe command/input/stdout/stderr/network files. Fixture provenance: success-f2kpvbr4, tools-nonzero-0g_xfv3k, missing-usage-iyi__lr6, turnlimit-0duw9lj6 (all prefixed `gemini064-`). No background probe jobs remain.

Best-effort reports are CLI observations. Missing per-request usage may also be hidden inside positive aggregate counters, so a positive subtotal is not certified complete provider accounting or a bill. Streaming JSON omits required breakdowns; no telemetry export or fallback was enabled to work around this.
