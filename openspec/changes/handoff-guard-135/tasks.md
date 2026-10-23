## 1. Implementation

- [ ] 1.1 Share one held-slot predicate between pre-compact and a guarded `SetHandoff` in `internal/state` (#135).
- [ ] 1.2 Refuse on the CLI (exit 2, `--force`), MCP (tool error, `force`), and HTTP (409, `?force=1`) paths; no PR comment on a refusal (#135).
- [ ] 1.3 Tests for refuse, force, same session, stale, and auto-save on each path, with mutation evidence (#135).
