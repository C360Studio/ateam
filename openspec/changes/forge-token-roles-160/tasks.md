## 1. Implementation

- [ ] 1.1 `state.ReadToken` / `state.WriteToken` resolve `ATEAM_TOKEN_READ` and `ATEAM_TOKEN_WRITE`, each falling back to the other; forge reads use the read token and writes the write token, before the existing Gitea and GitHub resolution (#160).
- [ ] 1.2 `ateam doctor` warns when reads and writes resolve to the same token, without printing a value (#160).
- [ ] 1.3 Tests and `task mutate:check` for swapped read/write precedence and the dropped single-token fallback (#160).
- [ ] 1.4 Docs name the two variables (#160).
