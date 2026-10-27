## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#183). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `ateam init` and this repository's `.gemini/settings.json` write `"context": {"fileName": "AGENTS.md"}` and no flat `contextFileName` (#183).
- [ ] 2.2 Doctor reads only `context.fileName`; a flat-only file fails with a violation naming the nested key. User-scope fallback and project precedence are unchanged (#183).
- [ ] 2.3 Tests cover nested pass, flat-only fail, nested `GEMINI.md` fail, and user-scope fallback on the nested key, with a mutation check on the flat-key rejection (#183).
- [ ] 2.4 Docs and the generated README table name the nested key; `docs/brownfield.md` gains a migration row; the harness spec gains one scenario cited by the tests (#183).
