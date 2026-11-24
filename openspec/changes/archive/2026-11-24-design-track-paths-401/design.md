# Design: design-track-paths-401

## Context

- `internal/state/settings.go:31-38`: `.agents/ateam.json` keys. `DisallowUnknownFields` at `:71` means any new key is a schema change.
- `docs/adoption.md:178`: ateam.json "is not a gate file: a key that can weaken a merge gate belongs in `.agents/review-policy.json` or gate-protected workflow env, never here". AGENTS.md § 3 carries the same rule.
- `internal/review/policy.go:12-24`: the policy struct. `internal/review/` and `.agents/review-policy.json` are gate files (`scripts/gate-files.sh`).
- `cmd/ateam/main.go:141,209`: `state spec-check` takes no arguments today. `internal/testspec/diff.go:40-61` already resolves `--base <ref>` to a merge base, with clear errors for a missing ref or shallow history.
- `.github/workflows/ci.yml:32,59`: `actions/checkout` with default depth 1, then `task verify`. There is no merge base in CI, and the workflow is a gate file.
- `.agents/protocol.md:31`: the track triggers. Open PR #407 (#398) also edits `internal/state/openspec.go`, so the new code goes in a new file to avoid conflicts.

## Goals / Non-Goals

Goal: a touched trigger path without a design fails a gate an agent cannot skip. Non-goals are listed in the proposal.

## Decisions

**D1: where the list lives (owner question).**
- A: `design_paths` in `.agents/ateam.json`, as the issue proposes. It is cheap and not a gate file, but deleting a glob in the same PR switches the check off for that PR. That breaks the rule at `docs/adoption.md:178`.
- B: `design_paths` in `.agents/review-policy.json`. Every edit to the list needs the owner's gate approval, and the protection is inherited for free. Cost: parsing lives in `internal/review/policy.go`, a gate file, so this PR needs one gate approval. Adding a path also takes an owner approval, which seems right for a list of high-consequence files.
- Recommend B. Adopter seam: an adopter without the key sees no change. With B, the adopter's existing owner gate covers edits.

**D2: what the diff is compared against (owner question).**
- A: `spec-check --base <ref>`, as in test-spec. Without `--base`, the check tries the merge base with the default branch (`default_branch` setting, else `origin/main`/`origin/master`). If that doesn't resolve, it prints `design paths: not checked (no base; pass --base)` and passes. No network access.
- B: the same as A, but CI also sets `fetch-depth: 0` and passes `--base origin/$GITHUB_BASE_REF`. That is a workflow edit (gate file) and makes the check binding in CI.
- C: spec-check fetches the base itself in CI. It's hidden network access from a validation command. Rejected.
- Recommend A in this PR plus B as a separate step for ateam's own CI. Adopters wire their own CI. Adopter seam: an adopter who never passes `--base` in CI gets the "not checked" line in the log, never a silent pass.

**D3: glob syntax.**
- A: stdlib `path.Match` against the slash path, plus a trailing `/**` meaning "everything under". No dependency.
- B: full doublestar matching. That needs a dependency or a custom matcher.
- Choose A. An invalid pattern fails at load time, naming the entry.

**D4: what counts as "carries a design".** The diff adds or modifies a file matching `openspec/changes/*/design.md` or `openspec/changes/archive/*/design.md`. It does not check that the design mentions the path. That would be a prose judgment left to review. The check stops the zero-design case only.

## Risks / Trade-offs

- A PR can attach any unrelated design.md edit to pass. Review catches that; the check stops the silent path.
- Under D2 A, ateam's own CI stays "not checked" until B lands. The log says so.

## Migration Plan

None. The key is optional and absence keeps today's behaviour.

## Open Questions

- **Q1 (owner): D1, A or B?** B is recommended.
- **Q2 (owner): D2. Should B (the CI wiring) ship in this PR or as a follow-up?** Recommended: same PR, so the check is binding, with one gate approval covering D1 B and the workflow.
