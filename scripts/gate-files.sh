#!/usr/bin/env bash
# Gate files check (#140). Runs from .github/workflows/gate-files.yml on
# pull_request_target, so GitHub takes this script and that workflow from the
# base branch: a pull request cannot edit the check that judges it.
#
# A pull request whose changed files (or a rename's previous path) include a
# gate file fails until it carries the owner-only label below. With a shared
# forge login the label is an attestation, not proof of who applied it.
#
# Usage: gate-files.sh <pr-number>
# A read that fails exits 2 naming the read; it never counts as "no gate files".
# Needs gh.
set -uo pipefail

readonly approval=gate-change-approved
readonly gate='^(\.agents/review-policy\.json|scripts/merge-check\.sh|scripts/gate-files\.sh|\.github/workflows/|internal/review/)'

pr=${1:-}
case "$pr" in
  '' | *[!0-9]*) echo "gate-files: usage: gate-files.sh <pr-number>" >&2; exit 2 ;;
esac

if ! files=$(gh api --paginate "repos/{owner}/{repo}/pulls/$pr/files?per_page=100" --jq '.[] | .filename, (.previous_filename // empty)'); then
  echo "gate-files: unavailable: could not read the changed files of pull request #$pr" >&2
  exit 2
fi
touched=$(printf '%s\n' "$files" | grep -E "$gate" | sort -u)
if [ -z "$touched" ]; then
  echo "gate-files: ok: pull request #$pr touches no gate file"
  exit 0
fi
if ! labels=$(gh pr view "$pr" --json labels --jq '.labels[].name'); then
  echo "gate-files: unavailable: could not read the labels of pull request #$pr" >&2
  exit 2
fi
echo "gate-files: pull request #$pr touches gate files:"
printf '%s\n' "$touched" | sed 's/^/  /'
if printf '%s\n' "$labels" | grep -qx "$approval"; then
  echo "gate-files: ok: labelled $approval by the owner"
  exit 0
fi
echo "gate-files: FAIL: a gate-file change waits for the owner to add the label $approval" >&2
exit 1
