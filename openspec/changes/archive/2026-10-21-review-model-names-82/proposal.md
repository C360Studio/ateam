# Change Proposal: `review-model-names-82`

## Why

merge-check required `implemented-by:` and `reviewed-by:` to name different agents, but it couldn't see models. That left "a reviewer is never weaker than the implementer" unenforceable, and reviewer models went unnamed on half of recent code PRs (#82, ADR 0001 step 2).

## What Changes

- `scripts/merge-check.sh` requires both PR-body lines to name a model in parentheses after the agent, such as `implemented-by: claude (opus)` and `reviewed-by: codex (gpt-6)`. Several implementer models can be comma-separated. A missing or blank model is a review finding with the line to write. On a draft it is printed and fails nothing, the same as the other findings.
- Review record comments are unchanged.
- The `merge:check+review` rows in `AGENTS.md`, `.agents/rules-index.md`, and both bootstrap templates, plus the harness spec merge-gate scenario, name the requirement.
- Deferred to the second half of #82: the strength check against `.agents/review-policy.json` (`ateam review compare`). It waits on the model ranking (#81).
