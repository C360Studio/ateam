## 1. Implementation

- [x] 1.1 Guard the pre-compact branch in `internal/hook/hook.go` so a non-stale foreign handoff is not overwritten, and name the holder in the hook output (#124).
- [x] 1.2 Table test in `internal/hook`: a non-stale handoff from session A survives pre-compact from session B, and the output names A; nil, stale, and same-session cases keep today's behavior (#124).
- [x] 1.3 Record mutation detection with `task mutate:check -- -pkg ./internal/hook -test TestPreCompactSavesAutoHandoff -file internal/hook/hook.go`: guard removed (`if false && foreignSession`) and auto-save exemption removed, both `detection` at sha256 `37217fd6` (#124).
