# Change: Deterministic test-to-spec attribution

Issue: https://github.com/C360Studio/ateam/issues/18
Owner: Codex (`codex/test-spec-attribution`)
implemented-by: codex
Status: Implementation and verification in progress.

## Why

ATeam validates spec shape and Hold placement but does not require tests to cite the requirements they prove. Semengine's existing prose citations demonstrate the convention, not a verified universal enforcement mechanism.

## What Changes

- Define machine-readable test comments citing one or more spec requirements.
- Add an offline deterministic checker, verification gate, and meaningful attribution for existing Go and Vitest tests.
- Align repository guidance, generated instructions, and adoption documentation with supported languages and review limits.

## Impact

Expected scope: CLI/state or a focused checker package, Taskfile.yml, Go and UI test comments, OpenSpec requirements, AGENTS.md, .agents/rules-index.md, relevant protocol/skills, internal/bootstrap/templates.go, and adoption documentation. Existing tests, fixtures, CLI callers, parity tests, and verification wiring must be inspected before implementation.

## Risks and Verification

Comment attachment, nested tests, fixture source strings, duplicate requirement titles, and paths escaping the repository can cause false acceptance or rejection. Use failing-first fixtures, an independent expected diagnostic oracle, parser generated checks where justified, and mutation evidence. A valid citation does not establish that the assertion proves the requirement; independent review still does that.

## Stop Boundary

The owner authorized driving PR #19 to completion. Implement the contract, verify it, obtain independent review, and land the PR when acceptance and repository merge gates pass.
