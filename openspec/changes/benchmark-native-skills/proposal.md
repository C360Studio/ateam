# Isolated native skill treatments

## Why

Versioned instruction bundles (#26 / #27) preserve prompt inputs, but cannot show whether an agent discovered or activated a native skill. Roadmap #16 next needs a controlled skill experiment with honest execution evidence.

## What Changes

- Add an explicit, immutable single-skill treatment through the existing benchmark workflow, starting with Gemini after verifying its pinned CLI contract.
- Keep model, reasoning, ordinary instructions, skill identity/description and tool permissions fixed; vary only the declarative skill body, or use identical bodies for A/A.
- Retain requested, staged and observed native evidence separately, including unavailable evidence and unsuccessful attempts.
- Preserve existing inline/bundle experiments, independent grading and offline replay.

## Scope and stop boundary

Focused issue #28 addresses roadmap #16. This request stops at the issue, dedicated branch/worktree, OpenSpec design and draft PR. No implementation or paid runs are authorized by this claim. The deferred Gemini capacity retry remains deferred.

Implementation scope is the shared benchmark resolution/run/replay pipeline, Gemini execution/event handling, focused fixtures/examples and benchmark documentation. Native hooks, executable skills/resources, other provider skill adapters, reasoning/model comparisons, routing, herdr and UI changes remain separate work. The initial skill contract must be independently reviewed before implementation; unsupported consent/isolation/telemetry behavior is a concrete stop condition, not permission to weaken controls.
