# Tasks

## Contract and implementation
- [x] Review the installed Codex API auth, configuration, native identity and usage contract independently (app-server CLI-resolved identity; no unsupported output/turn caps).
- [x] Implement isolated API-only execution and strict native event parsing.
- [x] Integrate configuration, validation, provenance, stop rules and replay without changing existing provider artifacts.
- [x] Add fixed-model A/A and A/B examples and workflow documentation.

## Verification and delivery
- [x] Add independent parser/accounting/process/history oracles and detect critical mutations.
- [ ] Complete read-only change review, parity/self-tests and clean-tree full verification.
- [ ] Publish implementation and evidence through draft PR #13.

## Live pilot
- [ ] Run and report a separate Luna A/A pilot with immutable offline replay.
  Hold: User reports OPENAI_API_KEY is ready in main's ignored .env; live execution depends on verified implementation and a bounded canary before the full pilot.
