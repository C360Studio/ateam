## 1. Implementation

- [ ] 1.1 Guard the pre-compact branch in `internal/hook/hook.go` so a non-stale foreign handoff is not overwritten, and name the holder in the hook output (#124).
- [ ] 1.2 Table test in `internal/hook`: a non-stale handoff from session A survives pre-compact from session B, and the output names A; nil, stale, and same-session cases keep today's behavior (#124).
- [ ] 1.3 Record mutation detection on the foreign guard with `task mutate:check` (#124).
