# Proposal: design-track-paths-401

## Decisions you are making by approving

- A repository can name the files where a small diff has a big effect, and a pull request that touches one fails `task verify` unless it carries a design.
- Only the owner can shorten that list, because removing an entry would quietly switch the check off.
- On a local checkout or in CI without a comparison branch, the check says it did not run instead of passing silently.

## Track

`design`: adds a spec Requirement (`harness`), adds an exported setting reachable from adopting repositories, and the list itself weakens a merge gate when edited (see D1).

## Why

The protocol's design-track triggers (`.agents/protocol.md:31`) are prose, reviewed by agents and humans. An adopter (#401) has a few files (arming, command authority, actuator output) where small diffs carried wrong defaults through as ordinary slices. Their glob rule under `.agents/rules/` is also prose: nothing checks that a change touching those paths carries a `design.md`. No setting for this exists today (`internal/state/settings.go:31-38`).

## What Changes

1. A per-repository list of `design_paths` globs (where it lives is D1).
2. `ateam state spec-check` lists the paths the branch changes against a base (D2). When one matches a glob and the diff adds or edits no `openspec/changes/**/design.md` (in-flight or archived), it fails, naming the path and the glob.
3. Docs: `docs/adoption.md` § Team settings or the policy docs (per D1), and `.agents/protocol.md:31` points to the setting.

## Non-goals

- Checking that the design is good or approved beyond what spec-check already enforces for in-flight changes.
- Putting ateam's own gate files into the list. They already have their own gate.
- Moving the check into `ateam review gate` (owner ruling, 2026-11-24).

## Definition of Done

Fixture tests show four cases: a matching path with no design.md fails and names the path and glob; the same diff with a design.md passes; a diff with no matches passes; no base gives a visible "not checked" line. A mutation check on the match detects. `task verify` passes.
