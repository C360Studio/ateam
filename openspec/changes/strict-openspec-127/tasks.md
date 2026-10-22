## 1. Implementation

- [x] 1.1 Move detail from the three over-long requirement statements into scenarios, keeping headers and normative content (#127).
- [x] 1.2 Install the root packages in the CI job that runs `task verify` (#127).
- [x] 1.3 Print a skip notice from `spec:check` when `node_modules/.bin/openspec` is absent, with a test; `task mutate:check -- -pkg ./internal/state -test TestOpenSpecQueueAndCheck -file internal/state/openspec.go`: notice never set and notice never cleared, both `detection` at sha256 `eaf7dc67` (#127).
