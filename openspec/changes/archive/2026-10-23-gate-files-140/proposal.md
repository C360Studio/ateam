# Change Proposal: `gate-files-140`

## Why

CI takes the review policy, `scripts/merge-check.sh`, the `ateam` binary it builds, and `.github/workflows/ci.yml` from the pull request's own tree, so a pull request blocked by a strength finding could loosen the gate that judges it (#140). Any check that lives in that tree can be edited by the same pull request.

## What Changes

- A `Gate files` workflow on `pull_request_target` (taken from the base branch, never from the pull request) runs `scripts/gate-files.sh` from the base branch. It reads the pull request's changed files through the API and fails when any is a gate file, until the owner adds the label `gate-change-approved`.
- Owner decision on #140 (2026-10-23): a deterministic base-branch check plus an owner-only label; no CODEOWNERS or separate agent identity for now. The label is an attestation under a shared login.
- The owner adds `Gate files` to the `main` ruleset's required checks after this merges.
