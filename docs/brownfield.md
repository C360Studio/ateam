# Bringing ateam into an existing repository

This is the path for a repository that already has its own `AGENTS.md`, `CLAUDE.md`, `.claude/settings.json`, a
`Taskfile.yml`, and CI. `ateam init` never overwrites those. `ateam doctor --check` then lists, one line at a time,
what you merge by hand; when it exits 0 you are done. Boundaries (Go-first, no language detection, parity is not
integration) are in the [adoption guide](adoption.md); read them first.

Every output below was recorded on the v0.3.2 candidate (`eeda39d`, the commit that landed this guide; the adopter
checks from #158, #159 and #152 are in it) against a scratch repository: a hand-written `AGENTS.md` and `CLAUDE.md`, a
`.claude/settings.json` with one permission and one `SessionStart` hook, and a one-task `Taskfile.yml`.

## 1. Build the binary

Releases carry no binary assets yet, so build from source as in the
[README](../README.md#prerequisites-and-installation) (released adopters: `git checkout v0.3.2`; reviewing before that tag exists: build `main` at or after `eeda39d`. Do not
build `v0.3.1`: it predates the adopter checks this guide describes), put `bin/ateam` on the PATH of your agents, and check it:

```bash
ateam version   # ateam v0.3.2 (<commit>)
```

The release part comes from the build, not from the source. `task build` stamps `git describe`, so a tag checkout prints the tag and a later commit prints `<tag>-<n>-g<commit>`. A plain `go build` prints the version Go embeds from git, which can be the parent checkout's inside a linked worktree, or `dev` without git data. Use `task build`. A smoke pass qualifies a release candidate only when `ateam doctor` shows `hook_binary`, `disk_build`, and `head_commit` at the candidate commit, because the hooks run the `ateam` on `PATH`, not `./bin/ateam`.

## 2. Run init without `--force`

```bash
cd your-repo
ateam init
```

Init adds only what is missing: the `.agents/` tree (protocol, rules index, six skills, contracts), the
`.claude/skills` and `.gemini/commands` adapters, `openspec/`, and any of `.mcp.json`, `.codex/` and `.gemini/settings.json` you do not
have. It appends four lines to `.gitignore` (`.ateam/`, `.worktrees/`, `.enjoy-logs/`, `benchmarks/runs/`). Your
`AGENTS.md`, `CLAUDE.md`, `.claude/settings.json` and `Taskfile.yml` stay byte-for-byte unchanged, and `backed_up` in
the JSON output is empty. Init does not create a Taskfile.

`--force` is not the migration path. It replaces `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` and
`.agents/rules-index.md` with ateam's text, copying yours first to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>`. Client
configuration is never replaced, so `--force` resolves none of the violations below.

### Existing specialist contracts

Your repository may already keep role contracts under `.agents/contracts/` with its own names (`go-developer.md`,
`acme-reviewer.md`). Init adds the base `developer.md`, `reviewer.md` and `architect.md` beside them and touches none of
yours. A subagent that reads only its specialist contract never sees the base rules, so link the two:

- **Keep the base contracts as init wrote them.** With routes in `.agents/review-policy.json`, `ateam review setup`
  drift-checks `developer.md` and `reviewer.md`: each must still contain every paragraph ateam generates. Add your own
  paragraphs if you must; do not reword the generated ones.
- **Open each specialist contract with one pointer sentence.** For example: "Read `.agents/contracts/reviewer.md`
  first; this contract adds this repository's rules on top." Use `developer.md` or `architect.md` for those roles.
- **Keep the record grammar in one place.** `reviewed-by: <agent> (<model>, <effort>)`, `Reviewed at <sha>` and
  `Verdict:` live in [reviewer contract § Record](../.agents/contracts/reviewer.md#record). Delete any copy in a
  specialist contract; a stale copy is how a record ends up with an effort the gate cannot compare.
- **If your contract already has a base name**, such as your own `architect.md`, init leaves it in place and the
  `/design` skill reads it as the architect contract. Until #217 decides whether `architect.md` is drift-checked, copy
  the paragraphs of ateam's [architect contract](../.agents/contracts/architect.md) into yours unchanged and keep your
  rules after them.

## 3. Resolve the doctor violations

```bash
ateam doctor --check   # JSON on stdout; exit 1 while "violations" is non-empty
```

On the scratch repository it reported nine violations. The only warning was that forge reads and writes share one
token (see [forge tokens](adoption.md#forge-tokens)); it does not fail the check.

| Doctor says | What to do by hand |
| --- | --- |
| `CLAUDE.md: missing @AGENTS.md import` | Make `CLAUDE.md` an import: a title and `@AGENTS.md`, at most 10 lines, no `##` headings or tables. Move its prose into `AGENTS.md`. |
| `rule section "N. ..." is in .agents/rules-index.md but missing from AGENTS.md` (or a title or row-count disagreement) | Both files need the same numbered `## N.` or `### N.` sections, with the same titles and table rows. Write your own rules and replace the shipped `.agents/rules-index.md` with their mirror. Keep `AGENTS.md` under 24,000 bytes. |
| `.agents/protocol.md: missing "<phrase>"` | You edited the protocol. Keep these phrases: `Deliverable`, `non-goals`, `Definition of Done`, `Stop boundary`, `ateam state handoff`, `openspec/changes/archive/`. Headings are yours. |
| `<file>: missing hook command for event <event>` and `missing --agent <client>` | Merge ateam's hook commands into the file next to your own (below). Doctor looks for the exact text `ateam hook --event <event> --agent <client>`. |
| `.gemini/settings.json: missing context.fileName AGENTS.md` | Add `"context": {"fileName": "AGENTS.md"}` at the top level. If the file has a `context` object, add `"fileName": "AGENTS.md"` inside it. |
| `... (Gemini CLI ignores the top-level contextFileName)` | Replace the top-level `"contextFileName": "AGENTS.md"` with `"context": {"fileName": "AGENTS.md"}`. Gemini CLI reads only the nested key, so the old line had it loading `GEMINI.md`. |
| `<file>: expected enabled ateam stdio registration ...` | Add the `ateam` server to the file's existing servers, from the [MCP guide](mcp.md#migrating-existing-configurations). |

The scratch repository's `AGENTS.md` kept its three sentences of product rules and gained one numbered table of two
rows; `.agents/rules-index.md` repeated that heading and those two rows. Product rules belong in your `AGENTS.md`;
the workflow rules ateam ships live in `.agents/protocol.md` and the skills, so nothing forces them into `AGENTS.md`.

Hooks for Claude Code are the `.claude/settings.json` route that works today: map `SessionStart`, `UserPromptSubmit`,
`PreCompact` and `Stop` to `session-start`, `before-prompt`, `pre-compact` and `turn-end`. The merged file passed:

```json
{
  "permissions": { "allow": ["Bash(make test)"] },
  "autoCompactWindow": 500000,
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "echo acme-session" }] },
      { "hooks": [{ "type": "command", "command": "ateam hook --event session-start --agent claude" }] }
    ],
    "UserPromptSubmit": [
      { "hooks": [{ "type": "command", "command": "ateam hook --event before-prompt --agent claude" }] }
    ],
    "PreCompact": [
      { "hooks": [{ "type": "command", "command": "ateam hook --event pre-compact --agent claude" }] }
    ],
    "Stop": [
      { "hooks": [{ "type": "command", "command": "ateam hook --event turn-end --agent claude" }] }
    ]
  }
}
```

Codex uses the same shape in `.codex/hooks.json` with `--agent codex`; Gemini uses `SessionStart`, `BeforeAgent`,
`PreCompress` and `AfterAgent` in `.gemini/settings.json` with `--agent gemini`. If your project pins its settings
file, register the hooks in `.claude/settings.local.json` or `~/.claude/settings.json` instead and run doctor with
`ATEAM_HOOK_SCOPE=user`; see [adoption boundaries](adoption.md) for which files each client reads.

Rerun `ateam doctor --check` until `"passed": true`. Then start a session: the `session-start` hook injects the pickup
summary, and `ateam state pickup` prints it on demand.

## 4. What stays yours afterwards

- **Skills.** Doctor checks the six canonical skills (`pickup`, `handoff`, `preflight`, `ponytail`, `review-pr`, `design`)
  strictly: numbered steps, workflow markers, and an adapter that links to them. Any other directory under
  `.agents/skills/` is an adopter skill, listed as `adopter_skills` and checked only for links that resolve (for
  example, the opt-in [plain technical prose recipe](adoption.md#plain-technical-prose-ste-style-anti-buzzword-recipe)). To use
  your own `preflight`, put it at `.agents/skills/<yours>/SKILL.md` and point the Claude and Gemini adapters at it;
  doctor reports `skill_aliases` such as `alias: .claude/skills/preflight -> .agents/skills/acme-preflight`. An aliased
  target inherits the canonical skill's obligations: it is not an ordinary adopter skill, and doctor requires `## 1.`
  and the canonical markers in it (`file:line` and `adopter seam` for design; `Goal:` and `ateam state handoff` for
  handoff; `Verify Write Ownership` and `ateam state pickup` for pickup; `6-Step Decision Ladder` and `// shortcut:`
  for ponytail; `task verify` and `Record Evidence` for preflight; `reviewed-by:` and `Reviewed at` for review-pr, or
  `review-implementation.md` and `--reviewer-record` under `review_record: "aip"`). Add the markers to your file or
  consolidate onto the generic one. Keep the generic `.agents/skills/preflight/SKILL.md` either way: Codex and
  Antigravity read it directly, so doctor still requires it with its markers; deleting it fails with
  `missing canonical skill`.
- **OpenSpec tracks.** If your repository already has `openspec/config.yaml`, `ateam init` leaves it untouched
  (otherwise writing `schema: spec-driven`). Non-trivial changes choose `quick` (`proposal.md` + `tasks.md`, no
  `design.md`) or `design` (`proposal.md` + `design.md` via `/design` + `tasks.md`), gated on `Approved-by:` and Task 1's
  `Hold: OPEN-Q proposal awaits owner approval`; copy `rules: design:` from ateam's `openspec/config.yaml` only if you
  also invoke `openspec instructions design` alongside `/design` (#218). `ateam state spec-check` uses
  `node_modules/.bin/openspec` or `openspec` on `PATH` for strict validation and prints a skip notice when neither is
  installed.
- **Verification.** Do not copy ateam's `Taskfile.yml`. Keep your own `task verify` or CI entry point and add the
  `ateam` binary to it: `ateam doctor --check` for parity, and `ateam review gate <pr>` for the merge gate (see
  [the merge gate](adoption.md#the-merge-gate-on-github-and-gitea)).
- **Review.** Reviewer strength comes from `.agents/review-policy.json`; set `"cross_provider": true` to require a
  different agent family (the adopter default is off). The record your own tooling should read is
  [reviewer contract § Record](../.agents/contracts/reviewer.md#record), or set `"review_record": "aip"` and
  compare with `ateam review compare --reviewer-record`; see [model review](model-review.md#review-record-mode-review_record).
  Mapping historical records is yours.

## 5. Gate new tests first

`ateam test-spec` fails on every uncited `Test`/`Fuzz`/`Example` and Vitest test, so a repository with years of tests
fails it on day one. Start with the diff scope in your pull-request CI:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0   # --base needs the merge base; a depth-1 clone exits 2
- run: ateam test-spec --base origin/${{ github.base_ref }}
```

With `--base <ref>`, a missing citation is reported only for a test whose declaration, body, or attached comment
lines changed since the merge base of `<ref>` and `HEAD`, counting uncommitted edits and untracked files, so a local
run shows what the PR will. Editing a legacy test therefore obliges citing it; untouched ones never fail. Malformed or
dangling citations and unparseable test files fail anywhere in the tree, so renaming a requirement still catches
every test that cites it, and an older house citation form fails until it is rewritten. An unknown `<ref>`, a root
outside a git work tree, or a clone without the merge base exits 2 rather than checking everything or nothing.

Once `ateam test-spec` with no flag reports zero, add it to `task verify` and keep the `--base` run or drop it.
