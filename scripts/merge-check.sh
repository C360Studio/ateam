#!/usr/bin/env bash
# Merge check (openspec spec harness, "Merge gate and CI workflow enforcement").
# The check is `ateam review gate` (internal/review/gate.go) on GitHub and on
# Gitea; this wrapper keeps the entrypoint the workflows and `task merge:check`
# call. The binary is $ATEAM, else ateam on PATH.
#
# Usage: merge-check.sh <pr-number>
#   Exit 0 pass, 1 fail, 2 unavailable (a read failed or was not the shape asked for).
#   The kind of run comes from GITHUB_EVENT_NAME only when GITHUB_ACTIONS is true.
set -euo pipefail
exec "${ATEAM:-ateam}" review gate "$@"
