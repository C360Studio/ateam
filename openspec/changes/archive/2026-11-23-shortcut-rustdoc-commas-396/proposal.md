# Proposal: shortcut-rustdoc-commas-396
Approved-by: Coby Leuschke (2026-11-23) sha256:a7c0fa07553e

## Track

`quick`: one function family in `internal/state/shortcut.go`, one MODIFIED Requirement (*Inline shortcut inventory*) gaining a scenario; no new Requirement, gate file, or exported surface.

## Why

The first Rust adopter (#382) hit two silent misreads in `ateam state debt`: a shortcut inside rustdoc (`/// shortcut:`, `//! shortcut:`, `/// // shortcut:`) is skipped, and a comma inside the limit (`raw frames (0x30..=0x3F, centre 0x800) …`) truncates it at the first comma.

## What Changes

1. `shortcutPattern` also accepts `///` and `//!` leaders, optionally followed by a nested `//`.
2. The limit/trigger split uses the first comma outside `()` and `[]`; when the brackets before every comma are unbalanced (no top-level comma exists), it falls back to the first comma, as today (owner ruling on #396, 2026-11-23).
3. `docs/adoption.md` shortcut-syntax row states the accepted leaders and the comma rule.

## Scope & Non-Goals

- **In-bounds:** `internal/state/shortcut.go`, its tests in `internal/state/state_test.go`, `docs/adoption.md`, `openspec/specs/harness/spec.md`.
- **Non-goals:** `{}` or quote nesting, block comments, HTML comments, new file types.

## Definition of Done

- Each doc-comment form and the bracketed-comma example round-trip with the expected limit and trigger; unbalanced input keeps today's split. A fuzz target checks the split against a reference model; `task mutate:check` detects removal of each new rule.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived before ready.
