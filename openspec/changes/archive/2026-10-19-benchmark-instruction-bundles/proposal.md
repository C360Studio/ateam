# Versioned benchmark instruction bundles

## Why

Inline instruction comparison works for small pilots. Harness experiments need file-backed, versioned prompt treatments whose exact content survives local edits and remains reviewable in dry runs and retained evidence.

## What Changes

- Resolve explicit ordered instruction files into immutable per-arm snapshots.
- Show component provenance and effective treatment differences in the existing dry-run workflow.
- Preserve inline experiments, fixed model/reasoning controls, independent grading and offline replay.

## Scope and stop boundary

Addresses roadmap #16 through focused issue #26. The initial design claim is expanded by the owner to implementation, independent review and verified PR delivery. Deliver text instruction bundles only; this change authorizes no paid benchmark calls. Native skill/hook execution, reasoning/model comparisons, role routing, UI and provider-interface cleanup remain separate slices. Gemini retries remain deferred.

Implementation is complete only when offline examples, input-boundary and history tests, mutation evidence, independent review and full repository verification pass. Requested/loaded instruction content must not be described as observed skill execution.
