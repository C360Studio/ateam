# Tasks: doctor-relative-path-warning-378

## 1. Approval

- [x] 1.1 Record owner approval for #378 (`Approved-by: Coby Leuschke (2026-11-16) sha256:968417bb2154`).

## 2. Implementation & Verification

- [x] 2.1 Add `CheckRelativeCommandPaths(root)` in `internal/bootstrap/mcp.go` and wire it into `runDoctor` in `cmd/ateam/main.go`.
- [x] 2.2 Add unit tests in `internal/bootstrap/mcp_test.go` covering relative hook commands, relative MCP commands, `cwd` exemption, bare/absolute/`$VAR`/`~/` exemptions, and deduplication per file+command.
- [x] 2.3 Update `openspec/specs/harness/spec.md` and `README.md`, run `task mutate:check`, `task verify`, `ateam doctor --check`, and `ateam bench --selftest`, then archive this change.
