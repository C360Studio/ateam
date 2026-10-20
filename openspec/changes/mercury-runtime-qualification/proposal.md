# Remaining Mercury runtime qualification

## Why

Issue #58 remains open. The completed offline audit, usage-provenance and bounded capture slices are [archived](../archive/2026-10-20-mercury-runtime-qualification/README.md) with PR #71. They establish a research capture path and concrete blockers, not a qualified production runtime.

## What Changes

Carry forward the three unfinished qualification gates and their existing spec delta. The next offline cancellation/failure teardown and descendant-cleanup slice is claimed on `codex/mercury-cancellation-qualification`; see [its scope and acceptance checks](cancellation-plan.md). This documentation claim does not start implementation or authorize spending.

## Impact

Addresses #58 and #16; related #28/#59. Preserve existing native runtimes and frozen Gemini evidence. No production code, runtime configuration or dependency change. The pending Mercury boundary requirement remains here until the broader qualification change is complete; archiving the research evidence does not silently promote it into a completed production specification.
