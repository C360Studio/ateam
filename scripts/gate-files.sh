#!/usr/bin/env bash
# Gate files check (#140). Runs from .github/workflows/gate-files.yml (or
# .gitea/workflows/gate-files.yml) on pull_request_target, so the forge takes
# this script and that workflow from the base branch: a pull request cannot
# edit the check that judges it.
#
# A pull request whose changed files (or a rename's previous path) include a
# gate file fails until it carries the owner-only label below. With a shared
# forge login the label is an attestation, not proof of who applied it.
#
# Usage: gate-files.sh <pr-number>
# A read that fails exits 2 naming the read; it never counts as "no gate files".
# Needs gh and jq on GitHub. With ATEAM_FORGE=gitea it needs curl and jq and
# reads Gitea's API v1 at GITHUB_SERVER_URL for GITHUB_REPOSITORY (Gitea
# Actions sets both) with GITEA_TOKEN (#143).
set -uo pipefail

readonly approval=gate-change-approved
readonly gate='^(\.agents/review-policy\.json|scripts/merge-check\.sh|scripts/gate-files\.sh|\.github/workflows/|\.gitea/workflows/|internal/review/)'

pr=${1:-}
case "$pr" in
  '' | *[!0-9]*) echo "gate-files: usage: gate-files.sh <pr-number>" >&2; exit 2 ;;
esac

if [ "${ATEAM_FORGE:-}" = gitea ]; then
  api=${GITHUB_SERVER_URL:-}
  api="${api%/}/api/v1/repos/${GITHUB_REPOSITORY:-}"
  # The token goes to curl on stdin (-H @-), never in its argv, which other
  # users of a shared runner can read; printf is a builtin, so neither is it.
  get() { printf 'Authorization: token %s\n' "${GITEA_TOKEN:-}" | curl -fsS --max-time 30 -H @- -H 'Accept: application/json' "$api/$1"; }
  read_pull() { get "pulls/$pr"; }
  # Every page until an empty one; 200 pages of 50 is past GitHub's 3,000-file
  # ceiling, and reaching it is a failed read, never a short list.
  read_files() {
    local page=1 body
    while [ "$page" -le 200 ]; do
      body=$(get "pulls/$pr/files?page=$page&limit=50") || return 1
      [ "$(printf '%s' "$body" | jq 'length' 2>/dev/null)" = 0 ] && return 0
      printf '%s\n' "$body"
      page=$((page + 1))
    done
    return 1
  }
  read_labels() { printf '%s' "$pull" | jq -r '(.labels // [])[].name'; }
else
  read_pull() { gh api "repos/{owner}/{repo}/pulls/$pr"; }
  read_files() { gh api --paginate "repos/{owner}/{repo}/pulls/$pr/files?per_page=100"; }
  read_labels() { gh pr view "$pr" --json labels --jq '.labels[].name'; }
fi

if ! pull=$(read_pull); then
  echo "gate-files: unavailable: could not read pull request #$pr" >&2
  exit 2
fi
if ! pages=$(read_files); then
  echo "gate-files: unavailable: could not read the changed files of pull request #$pr" >&2
  exit 2
fi
# Every page must be a list of file objects. The listing stops at 3,000 files,
# so a list shorter than the PR's changed_files is incomplete: a gate file past
# the cut would read as "no gate file". Both cases exit 2, never 0.
if ! list=$(printf '%s' "$pages" | jq -c -s 'if all(.[]; type == "array" and all(.[]; type == "object" and (.filename | type) == "string" and (.previous_filename == null or (.previous_filename | type) == "string"))) then add // [] else error("not pages of files") end'); then
  echo "gate-files: unavailable: the changed files of pull request #$pr are not a list of files" >&2
  exit 2
fi
if ! printf '%s' "$pull" | jq -e --argjson n "$(printf '%s' "$list" | jq length)" '(.changed_files | type) == "number" and .changed_files == $n' >/dev/null; then
  echo "gate-files: unavailable: the file list of pull request #$pr is incomplete or its changed_files is missing" >&2
  exit 2
fi
files=$(printf '%s' "$list" | jq -r '.[] | .filename, (.previous_filename // empty)')
touched=$(printf '%s\n' "$files" | grep -E "$gate" | sort -u)
if [ -z "$touched" ]; then
  echo "gate-files: ok: pull request #$pr touches no gate file"
  exit 0
fi
if ! labels=$(read_labels); then
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
