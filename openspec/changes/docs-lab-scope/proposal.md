# Change: Clarify lab purpose and language support

implemented-by: codex

Issue: https://github.com/C360Studio/ateam/issues/4
Owner: Codex (`codex/docs-lab-scope`)

## Why

ATeam's public introduction underexplains its experimental purpose, strong workflow opinions, and Go-specific tooling. Readers need to distinguish general repository coordination from language support, and self-test results from evidence that the harness improves agent outcomes.

## What Changes

- Introduce ATeam as an opinionated, Go-first lab for agent harness experiments.
- Explain Ponytail's decision ladder and Go adaptation, guidance injection, and the limits of discipline modes.
- Add an adoption guide covering language boundaries, initialization, Taskfile ownership, and command dependencies.
- Align relevant agent documentation and bootstrap template prose with the same boundaries.
- Preserve concurrent README/UI work when integrating this branch.

## Impact

- Affected specs: harness documentation expectations.
- Expected implementation scope: README.md, docs/, relevant .agents/ documentation, internal/bootstrap/templates.go, and focused bootstrap tests if necessary.
- Read-only evidence scope: hook, state/shortcut, taskrunner, mutate, bench, CLI, and Taskfile implementations.
- No runtime behavior changes, UI changes, new language support, or benchmark redesign.

## Risks and Verification

Claims can drift between public docs and generated instructions. Verify both surfaces against source and exercise a disposable non-Go repository where practical. Record exact checked revisions and distinguish source inspection from runtime evidence. Use the repository preflight gates; do not interpret fixture self-tests as agent-performance measurements.

## Stop Boundary

Deliver the documentation and verification evidence through draft PR #5. Stop before merge or issue closure; independent approval remains the reviewer’s responsibility.
