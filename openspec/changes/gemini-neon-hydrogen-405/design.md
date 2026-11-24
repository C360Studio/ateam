# Design: gemini-neon-hydrogen-405

## Context

- `.agents/review-policy.json:4-6`: Gemini tier entries are `gemini-4-carbon` (frontier), `gemini-4-argon` (workhorse), and `gemini-3.8-flash` and `gemini-3.1-pro` (light).
- `.agents/review-policy.json:11`: `"gemini": {"low": ["minimal", "low"], "mid": ["medium"], "high": ["high"]}`.
- `docs/adr/0002-model-tiers-and-effort-bands.md:24`: the Gemini effort row says the current models accept `low`/`medium`/`high` only.
- `docs/adr/0002-model-tiers-and-effort-bands.md:36-38, 46, 58-65`: tier table, band table and the JSON copies.
- `docs/adr/0002-model-tiers-and-effort-bands.md:86`: model text matches tier names ignoring case and spaces. A suffix such as `-xhigh` is not stripped.
- `internal/review/compare_test.go:18`: `TestCommittedPolicyTiers` holds a literal copy of the committed tiers.
- `agy models` output (2026-11-24): `gemini-4-{neon,hydrogen}-{low,medium,high,xhigh}`, `gemini-4-{carbon,argon}-{low,medium,high}`, `gemini-3.8-flash-{low,medium,high}`, plus `gemini-pro-orchestrator{,-neon,-argon}`. There is no `gemini-3.1-pro`.

## Goals / Non-Goals

Goal: Neon and Hydrogen runs are ranked, and `xhigh` maps to a band. Non-goals are listed in the proposal.

## Decisions

**D1: tier placement (ruled).** Option A puts Neon in frontier and Hydrogen in workhorse. Option B puts Neon in workhorse and Hydrogen in light. The owner chose B. As a result, only Carbon can review frontier work among Gemini models.
Adopter seam: an adopting repository with its own policy is not affected. A repository that copied ateam's policy must add the new IDs itself.

**D2: agy effort suffix (ruled).** Option A strips `-{low,medium,high,xhigh}` from model text in `internal/review`. That is a code change in a gate directory and could misread a future model whose name ends in one of those words. Option B documents the split form. The owner chose B. If an agent writes `gemini-4-neon-xhigh` anyway, compare reports `unranked` and suggests `gemini-4-neon`, which is a loud failure.

**D3: drop `gemini-3.1-pro` (ruled).** Keeping it costs nothing, but agy no longer offers it, so the entry would be stale. Once it is dropped, a historic PR line citing 3.1 Pro would read as unranked. No open PR cites it.

## Risks / Trade-offs

The tiers rest on Google's positioning and the owner's judgment, not on a benchmark. Moving a model later takes one policy edit.

## Migration Plan

None beyond the edit. Merged PRs are not rechecked.

## Open Questions

None. Every decision was ruled on #405 on 2026-11-24.
