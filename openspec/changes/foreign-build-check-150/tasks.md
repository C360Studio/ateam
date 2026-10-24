## 1. Implementation

- [ ] 1.1 The build-versus-`HEAD` check in `internal/bootstrap` and the session-start hook runs only when the root `go.mod` declares `module github.com/C360Studio/ateam`; doctor reports `build_provenance: not applicable (binary is not built from this checkout)` elsewhere (#150).
- [ ] 1.2 Table test: a foreign `go.mod` (and no `go.mod`) gives no warning and the not-applicable field; the ateam module with a non-ancestor build gives the existing warning. `task mutate:check` with the module check dropped reports `detection`. The hook latency test does not regress (#150).
