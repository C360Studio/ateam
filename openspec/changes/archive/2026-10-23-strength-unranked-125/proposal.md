# Change Proposal: `strength-unranked-125`

## Why

PR 114 passed the strength gate with `implemented-by: claude (opus, unknown)` and `reviewed-by: codex (GPT-6, unknown)`: `unknown` was a present effort that mapped to nothing, so it took the `unranked` pass instead of the missing-effort finding, and a frontier implementer was gated by a reviewer whose tier and band were both unknown (#125).

## What Changes

- A placeholder effort (`unknown`, `n/a`, `na`, `?`) is a missing effort: a finding.
- Owner ruling on #125 (2026-10-22), amending ADR 0002 § 2: an unranked model or effort passes only when both sides of the compared pair are unranked. Against a ranked run it is a finding naming the unranked side and, for a model, the tier entries its text begins.
