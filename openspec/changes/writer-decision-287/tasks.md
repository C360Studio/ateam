# Tasks: writer-decision-287

## 1. Approval

- [ ] 1.1 Owner approves the proposal and design, including a ruling on D3. Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Shared decision (premise `internal/state/state.go:3120-3145`, `internal/state/writerstatus.go:46-78`): split the place check out of `classifyWriterLease` and add one exported function returning the competing lease (or nil) from a `*State`, using the same session, client-PID and herdr-pane exclusions as today. Accept: table test over local live, local exited, elsewhere within and past the TTL, no host, same session, same herdr pane.
- [ ] 2.2 Summary (premise `state.go:3133-3145`): print the warning from 2.1; print the place in place of `(exited)` for a lease from elsewhere. Accept: literal output assertions in `state_test.go`.
- [ ] 2.3 Hook (premise `internal/hook/hook.go:152-156`, and `:104-108` if D3 is B): read 2.1 instead of searching `ctxText`. Accept: hook tests from the Definition of Done, including a multi-line decision card and a handoff field carrying the warning text.
- [ ] 2.4 Spec: ADDED Requirement in `openspec/specs/harness/spec.md`. Accept: `task spec:check` and `task test:spec`.
- [ ] 2.5 `task mutate:check` for both reversions; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
