# Reproduce the isolated accounting probe

This is an opt-in research procedure for macOS arm64 with the already installed Python 3.14.5, uv 0.11.16, pip 26.1.1 and exact Claude 2.1.292 binary recorded in `stack.json`. Do not substitute a runtime/version silently. It performs no provider inference. The ordinary `replay.py` command needs only Python's standard library and none of this setup.

Run from the repository worktree. Use a fresh ignored root under the user's home. The examples use `.ateam/mercury-contract-reproduction`; preserve existing runs. Downloading wheels requires network access, but their code is not imported or built during this step.

```sh
mkdir -p .ateam/mercury-contract-reproduction .enjoy-logs
/usr/bin/env -i PATH=/opt/homebrew/bin:/usr/bin:/bin PIP_CONFIG_FILE=/dev/null \
  /opt/homebrew/bin/python3 -m pip --isolated --disable-pip-version-check --no-cache-dir \
  download --only-binary=:all: --require-hashes --index-url https://pypi.org/simple \
  -r openspec/changes/archive/2026-10-20-mercury-runtime-qualification/offline/selected-wheels.lock \
  --dest .ateam/mercury-contract-reproduction/wheelhouse
```

The lock includes all 111 selected wheels. Inspect their METADATA, entry points and startup files as data before installation; hashes must match the lock. Refuse source builds, unresolved dependencies or a platform mismatch. The recorded inspection found no wheel `.pth`/`sitecustomize.py`/`usercustomize.py`. This is not assurance that arbitrary package code is benign.

Create a local installation policy using the actual root and pinned executable path. It allows path metadata but denies home-file contents outside the fixture and binary, all network access, out-of-root writes and keychain service lookup:

```sh
python3 -I - <<'PY'
import json
from pathlib import Path
r = Path(".ateam/mercury-contract-reproduction").resolve()
cli = Path("/Users/coby/.local/bin/claude").resolve(strict=True)
home = str(Path.home())
policy = '(version 1)\n(allow default)\n(deny network*)\n(deny file-write*)\n'
policy += '(allow file-write* (subpath '+json.dumps(str(r))+') (literal "/dev/null"))\n'
policy += '(deny file-read* (subpath '+json.dumps(home)+'))\n'
policy += '(allow file-read-metadata (subpath '+json.dumps(home)+'))\n'
policy += '(allow file-read* (subpath '+json.dumps(str(r))+') (literal '+json.dumps(str(cli))+'))\n'
policy += '(deny mach-lookup (global-name "com.apple.securityd"))\n'
(r / "install.sb").write_text(policy)
PY
```

Copy the lock into the root, then create/install with the root as the working directory. No global environment or user configuration is changed:

```sh
cp openspec/changes/archive/2026-10-20-mercury-runtime-qualification/offline/selected-wheels.lock \
  .ateam/mercury-contract-reproduction/selected-wheels.lock
cd .ateam/mercury-contract-reproduction
/usr/bin/env -i PATH=/opt/homebrew/bin:/usr/bin:/bin \
  /usr/bin/sandbox-exec -f install.sb /opt/homebrew/bin/uv --no-config --offline \
  --cache-dir uv-cache venv --no-project --no-python-downloads \
  --python /opt/homebrew/bin/python3 venv
/usr/bin/env -i PATH=/opt/homebrew/bin:/usr/bin:/bin \
  /usr/bin/sandbox-exec -f install.sb /opt/homebrew/bin/uv --no-config --offline \
  --cache-dir uv-cache pip install --python venv/bin/python --no-index \
  --find-links wheelhouse --require-hashes --only-binary :all: -r selected-wheels.lock
```

Return to the repository worktree before the following command. Inspect applicable managed policies first; do not override them. The original run found no local managed Claude policy files. Inspect/verify the OS isolation checks and stop if they fail. The reproducer itself checks external IPv4/IPv6 denial and out-of-root read/write refusal before starting the bridge, and checks the Claude binary hash. Fresh CLI child environments and `PYTHON_DOTENV_DISABLED=1` exclude ambient credentials and automatic ancestor `.env` loading.

```sh
/usr/bin/env -i PATH=/opt/homebrew/bin:/usr/bin:/bin \
  /opt/homebrew/bin/python3 -I \
  openspec/changes/archive/2026-10-20-mercury-runtime-qualification/offline/reproduce.py \
  .ateam/mercury-contract-reproduction \
  /Users/coby/.local/bin/claude \
  .ateam/mercury-contract-reproduction/run-1
```

For another machine, supply its installed binary path and regenerate the policy; the binary hash must still match. The hard-coded interpreter and platform are part of this measured stack. This procedure has not been qualified on other platforms.

The driver prints owned PIDs/ports and writes logs from launch. Register them when using Enjoy; the bridge log is `.enjoy-logs/run-1-bridge.log`. It stops and waits for the bridge, closes the fixture/capture servers and preserves output. Do not treat a failed startup or timeout as a passed accounting test.

Compare `run-1/ledger.json` and each case's terminal output against `observations.json`. Request projections intentionally omit vendor prompt/device/session data; do not publish complete raw native prompts or credentials. Identical protocol/control/counter outcomes are expected; session IDs, ports, timing and full-trace hashes need not match. The fake token counts and any client dollar estimates are fixtures, not observed provider usage or billing.
