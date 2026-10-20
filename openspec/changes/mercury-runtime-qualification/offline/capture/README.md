# Reproduce the offline capture evidence

These are opt-in research scripts, not production adapters. Read [the report](../../capture-report.md) and [scope](../../capture-plan.md) first. Run Python normally, **never with `-O`**: assertion-based evidence checks are intentional. Go/Vitest repository CI does not execute these Python checks.

## No-dependency replay

From the repository root, with Python3:

```sh
python3 -I openspec/changes/mercury-runtime-qualification/offline/capture/replay_capture.py
python3 -I openspec/changes/mercury-runtime-qualification/offline/capture/check_capture_replay.py
python3 -I openspec/changes/mercury-runtime-qualification/offline/capture/check_mutations.py replay
```

These read the retained synthetic evidence, import only stdlib plus the existing local usage reader, and make no network/model calls. They do not rerun the native stack or authenticate the supplied artifacts. The mutation runner writes temporary copies and never changes originals.

## Pinned-dependency hook tests

Use the already approved isolated macOS arm64 Python3.14.5 environment with the exact existing [wheel lock](../selected-wheels.lock), LiteLLM1.99.0, HTTPX0.28.1 and OpenAI2.54.0. This slice reused that installation; installing/upgrading elsewhere needs a new scoped decision. Verify all wheel hashes before any import, and inspect installed source hashes against `source_pins` in `capture-observations.json`. Do not use a global Python dependency installation.

Create an ignored isolated directory `R` under the current worktree, with `R/venv`, copies of `capture_hook.py`, `check_capture_hook.py`, `check_mutations.py`, and an empty `R/unit-work`. Use an OS policy allowing reads only of that directory within the user's home, writes only under `unit-work` and `/dev/null`, denying network access and keychain access. The macOS policy used here was:

```scheme
(version 1)
(allow default)
(deny network*)
(deny file-write*)
(allow file-write* (subpath "ABSOLUTE_R/unit-work") (literal "/dev/null"))
(deny file-read* (subpath "ABSOLUTE_USER_HOME"))
(allow file-read-metadata (subpath "ABSOLUTE_USER_HOME"))
(allow file-read* (subpath "ABSOLUTE_R"))
(deny mach-lookup (global-name "com.apple.securityd"))
```

Replace placeholders with actual absolute paths, save as `R/unit.sb`, and run with cwd `R` (an outside cwd triggers a denied home read during dependency import):

```sh
/usr/bin/env -i PATH=/opt/homebrew/bin:/usr/bin:/bin \
  PYTHON_DOTENV_DISABLED=1 LITELLM_LOCAL_MODEL_COST_MAP=True DO_NOT_TRACK=1 \
  CAPTURE_ROOT="$PWD/unit-work" CAPTURE_ATTEMPT=unit \
  CAPTURE_ENDPOINT=http://127.0.0.1:1/v1/chat/completions \
  /usr/bin/sandbox-exec -f unit.sb "$PWD/venv/bin/python" -I "$PWD/check_capture_hook.py"
```

Repeat that command replacing `check_capture_hook.py` with `check_mutations.py` and appending `hook`. All HTTP uses MockTransport. The hook suite tests ten behaviors; the copied-source runner expects six detections. Retain stdout/stderr and exit codes. Initial setup failures due to relative executable resolution/outside cwd were corrected before the successful runs; those failures are not test passes.

## Native fixture reproduction

This launches local servers and the actual native binary; first inspect current process/port ownership and the installed binary/source. It requires macOS `sandbox-exec`, `/opt/homebrew/bin/python3`, the pinned Claude binary and an existing verified `R/venv`. Copy `capture_hook.py` and `capture_reproduce.py` beside each other under `R`. Keep `R` within the user's home and the current worktree; run the driver from the worktree root with a **fresh**, absent output directory inside `R` for each case. No managed Claude policy may be installed. Do not loosen the driver's OS policies to get a run to pass.

```sh
/usr/bin/env -i PATH=/opt/homebrew/bin:/usr/bin:/bin \
  /opt/homebrew/bin/python3 -I ABSOLUTE_R/capture_reproduce.py \
  ABSOLUTE_R ABSOLUTE_PINNED_CLAUDE ABSOLUTE_R/fresh-known known
```

Cases: `known`, `zero`, `missing`, `partial`, `http-error`, `disconnect`, `cancel`, `intent-fail`, `body-fail`, `end-file-fail`, `end-dir-fail`, `disabled`, `oversize`. Every invocation prints actual driver/bridge PIDs and ports promptly. Capture its stdout/stderr from launch, register any background job/server, keep process ownership and inspect shutdown. Bridge logs use `.enjoy-logs/<output-name>-bridge.log` and refuse overwrite. The driver enforces fake credentials, fresh native home/settings, external-egress controls, evidence ownership and explicit cancellation synchronization. Never run against a real provider URL.

Raw `fixture.json` contains full native requests and must remain ignored. For review, project only the request hash/ID/path, model/stream/max_tokens/reasoning_effort/stream_options controls, unexpected retry-field names, synthetic response body/status, native usage/result status, summary/isolation results, capture records and raw-file hashes. Omit private paths/prompts/credentials and replace bodies above16KiB with length/hash. Reconcile the projection independently against every raw case before using the replay checker. Do not overwrite the published observations with a new cohort or present a modified hook/driver as this recorded run.
