# Change: Explain hook flow and opinions

Issue: https://github.com/C360Studio/ateam/issues/8
implemented-by: codex

## Why

The platform event mapping and adoption guide do not explain the complete hook lifecycle, state side effects, or the distinction between automatic mechanics and agent-followed policy.

## What Changes

Add a linked lifecycle guide with a flow diagram, event table, rationale/tradeoffs, mode and writer behavior, explicit handoff distinctions, and failure/troubleshooting guidance. Trace claims to code and existing tests. No runtime behavior or policy changes.

## Impact

Scope: README.md, .agents/README.md, docs/adoption.md, docs/hook-flow.md, and this OpenSpec change. Source review covers hook, state, CLI, and generated wiring. Acceptance: source-accurate event/state/output documentation, valid local links, parity and full preflight gates. Stop at a draft PR; no merge without instruction.
