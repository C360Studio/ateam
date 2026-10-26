package bootstrap

const claudeSettingsJSON = `{
  "autoCompactWindow": 500000,
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event session-start --agent claude"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event before-prompt --agent claude"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event pre-compact --agent claude"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event turn-end --agent claude"
          }
        ]
      }
    ]
  }
}
`

const codexHooksJSON = `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event session-start --agent codex"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event before-prompt --agent codex"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event pre-compact --agent codex"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event turn-end --agent codex"
          }
        ]
      }
    ]
  }
}
`

const geminiSettingsJSON = `{
  "contextFileName": "AGENTS.md",
  "model": {
    "chatCompression": {
      "contextPercentageThreshold": 0.5
    }
  },
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event session-start --agent gemini"
          }
        ]
      }
    ],
    "BeforeAgent": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event before-prompt --agent gemini"
          }
        ]
      }
    ],
    "PreCompress": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event pre-compact --agent gemini"
          }
        ]
      }
    ],
    "AfterAgent": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event turn-end --agent gemini"
          }
        ]
      }
    ]
  },
  "mcpServers": {
    "ateam": {
      "command": "ateam",
      "args": ["mcp"]
    }
  }
}
`

const codexMCPConfig = `model_context_window = 500000
model_auto_compact_token_limit = 450000

[mcp_servers.ateam]
command = "ateam"
args = ["mcp"]
`

const claudeMCPJSON = `{
  "mcpServers": {
    "ateam": {
      "type": "stdio",
      "command": "ateam",
      "args": ["mcp"]
    }
  }
}
`

const defaultAgentsMD = `# Repository Agent Instructions

ATeam is an opinionated experimentation lab primarily grounded in Go development. Initialization uses the same templates for all project languages; it does not detect language support or create a Taskfile. Review and adapt these instructions before adoption. Go mutation checks require non-test .go source; verification targets belong to the adopting project. Shortcut scanning covers only .go, .ts, .js, .svelte, .sh, .yml, and .yaml files (excluding Go tests and .test.ts files). Other languages are not validated as complete workflows. Ponytail modes change injected guidance, not model settings or guaranteed compliance. Benchmark self-tests validate reference fixtures, not improved agent performance.

Guidance for coding agents. Codex and Gemini load this file as ` + "`AGENTS.md`" + `; ` + "`CLAUDE.md`" + ` imports it (` + "`@AGENTS.md`" + `), keeping a single copy strictly under 24,000 bytes.

## Ponytail decision ladder

You are a lazy senior developer. The best code is the code never written. Solve the whole problem with the least new code, and end every reply with 1–2 lines stating **what you skipped or did not check** and **any residual risk**.

1. **Does it need to exist?** Skip unrequested features, options, and abstractions.
2. **Already in this codebase?** Reuse existing helpers, components, and conventions.
3. **Standard library or platform feature?** Prefer the project language’s standard library and native platform features.
4. **Installed dependency?** Use what is already in ` + "`go.mod`" + ` / ` + "`package.json`" + `; never add a dependency for a few lines.
5. **Readable at a glance?** Prefer flat, obvious code over clever one-liners.
6. **Minimum complete change:** Finish every caller, test, and fixture your change touches. Mark any intentional deferral with ` + "`// shortcut: <the limit>, <when to upgrade>`" + `.

## Rules and what enforces them

### 1. Ponytail decision ladder and code discipline

| Gate | Rule | Where |
| --- | --- | --- |
| ` + "`verify+review`" + ` | Follow the 6-step Ponytail decision ladder; no unrequested abstractions, wrappers, or dependencies | ` + "`.agents/skills/ponytail/SKILL.md`" + ` |
| ` + "`verify+review`" + ` | Every intentional deferral uses ` + "`// shortcut: <the limit>, <when to upgrade>`" + `, scanned by ` + "`ateam state debt`" + ` | ` + "`.agents/skills/ponytail/SKILL.md`" + ` |
| ` + "`verify+review`" + ` | Non-trivial logic ships with a test using an independent oracle and proven able to fail via non-destructive mutation check (` + "`task mutate:check`" + ` or ` + "`cp`" + ` + SHA-256; never ` + "`git checkout`" + `/` + "`restore`" + `/` + "`stash`" + `/` + "`reset`" + `), reporting ` + "`detection`" + `, ` + "`survivor`" + `, ` + "`invalid`" + `, or ` + "`inconclusive`" + ` | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify+review`" + ` | Changes with interacting inputs, stated laws (round-trip, idempotence), or order-dependent histories record whether generated checks (` + "`testing.F`" + ` / PBT) are used or why examples suffice, prove boundary reach and assertion activation, and check histories against a reference model | ` + "`.agents/protocol.md`" + ` |
| ` + "`" + `verify+review` + "`" + ` | Every Go Test/Fuzz/executable Example and Vitest test carries attached spec references (` + "`" + `// spec:` + "`" + ` or ` + "`" + `//aip:verifies` + "`" + `) resolved by ` + "`" + `task test:spec` + "`" + `; syntactic attribution does not prove semantic coverage | ` + "`" + `ateam test-spec` + "`" + `; ` + "`" + `.agents/protocol.md` + "`" + ` |
| ` + "`verify`" + ` | Go packages pass ` + "`gofmt`" + `, ` + "`go vet`" + `, and ` + "`revive`" + ` with zero warnings | ` + "`Taskfile.yml`" + ` |
| ` + "`review`" + ` | Agent replies end with 1–2 lines stating what was skipped/not checked and any residual risk | ` + "`.agents/skills/ponytail/SKILL.md`" + ` |

### 2. Hooks, MCP budget, and multi-agent parity

| Gate | Rule | Where |
| --- | --- | --- |
| ` + "`verify`" + ` | ` + "`AGENTS.md`" + ` stays under 24,000 bytes, ` + "`CLAUDE.md`" + ` only imports ` + "`@AGENTS.md`" + `, and rule row counts match ` + "`.agents/rules-index.md`" + ` | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | Canonical skills in ` + "`.agents/skills/*/SKILL.md`" + ` have matching thin adapters in ` + "`.claude/skills/`" + ` and ` + "`.gemini/commands/`" + ` | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | Hook configs in ` + "`.claude/settings.json`" + `, ` + "`.codex/hooks.json`" + `, ` + "`.gemini/settings.json`" + `, and ` + "`.agents/hooks.json`" + ` map native events to ` + "`ateam hook`" + `; with ` + "`ATEAM_HOOK_SCOPE=user`" + `, ` + "`ateam doctor`" + ` also accepts one complete local or user-scope file (` + "`.claude/settings.local.json`" + `, ` + "`~/.claude/settings.json`" + `, ` + "`~/.codex/hooks.json`" + `, ` + "`~/.gemini/settings.json`" + `) and names the file under ` + "`hook_sources`" + ` | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | ` + "`ateam mcp`" + ` exposes only the 2 multiplexed tools (` + "`ateam_state`" + `, ` + "`ateam_task`" + `) with ` + "`<300`" + ` total schema tokens | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | ` + "`ateam hook`" + ` executes normalized lifecycle events in ` + "`<10ms`" + ` target budget (` + "`doctor`" + ` warns ` + "`>10ms`" + `, fails ` + "`--check`" + ` ` + "`>25ms`" + `) with zero external runtime dependencies | ` + "`.agents/README.md`" + ` |

### 3. State, OpenSpec, Forge (GitHub/Gitea), and Definition of Done

| Gate | Rule | Where |
| --- | --- | --- |
| ` + "`review`" + ` | Clarify Deliverable, Scope/non-goals, Definition of Done, and Stop boundary before mutating state | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify+review`" + ` | Shared state syncs with GitHub (` + "`gh`" + `/API) or Gitea (` + "`tea`" + `/API v1) and git worktrees, cached in gitignored ` + "`.ateam/state.json`" + ` | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify`" + ` | OpenSpec changes/specs validate via ` + "`spec:check`" + `, and ` + "`Hold:`" + ` / caveats (` + "`HALT`" + `, ` + "`RED`" + `, ` + "`BLOCKED`" + `, ` + "`WONTDO`" + `, ` + "`OPEN-Q`" + `) are parsed by ` + "`spec:queue`" + ` | ` + "`openspec/`" + ` |
| ` + "`review`" + ` | Non-trivial work's OpenSpec ` + "`proposal.md`" + ` stays one page (≤ 60 lines; ` + "`spec:check`" + ` warns past it) and carries ` + "`Approved-by:`" + `, empty until the owner fills ` + "`<name> (<YYYY-MM-DD>)`" + `; the change's first task holds ` + "`Hold: OPEN-Q proposal awaits owner approval`" + ` until the owner removes it in the same edit, and an agent whose Stop boundary is proposal approval stops there and hands off | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify`" + ` | ` + "`ateam bench --selftest`" + ` verifies good/bad reference fixtures and mutation detection before any live benchmark run | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify`" + ` | ` + "`task verify`" + ` runs formatting, OpenSpec check, test-to-spec attribution, lint, unit/parity tests, benchmark self-test, and UI checks on a clean tree | ` + "`Taskfile.yml`" + ` |
| verify+review | Review requires an independent reviewer not weaker than every implementer by model tier, then effort band, optional provider separation and current trusted-base evidence; the dormant configured path adds scoped owner ceiling exceptions | ateam review; .agents/contracts/reviewer.md |
` + agentsMergeCheckRow + `
` + closeRuleRow + `

`

const defaultRulesIndexMD = `# Rules Enforcement Index

## 1. Ponytail decision ladder and code discipline

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| ` + "`verify+review`" + ` | Follow the 6-step Ponytail decision ladder; no unrequested abstractions, wrappers, or dependencies | ` + "`ateam hook`" + ` |
| ` + "`verify+review`" + ` | Every intentional deferral uses ` + "`// shortcut: <the limit>, <when to upgrade>`" + `, scanned by ` + "`ateam state debt`" + ` | ` + "`ateam state debt`" + ` |
| ` + "`verify+review`" + ` | Non-trivial logic ships with a test using an independent oracle and proven able to fail via non-destructive mutation check (` + "`task mutate:check`" + ` or ` + "`cp`" + ` + SHA-256; never ` + "`git checkout`" + `/` + "`restore`" + `/` + "`stash`" + `/` + "`reset`" + `), reporting ` + "`detection`" + `, ` + "`survivor`" + `, ` + "`invalid`" + `, or ` + "`inconclusive`" + ` | ` + "`ateam mutate`" + `; ` + "`ateam bench --selftest`" + ` |
| ` + "`verify+review`" + ` | Changes with interacting inputs, stated laws (round-trip, idempotence), or order-dependent histories record whether generated checks (` + "`testing.F`" + ` / PBT) are used or why examples suffice, prove boundary reach and assertion activation, and check histories against a reference model | ` + "`task test:unit`" + `; review |
| ` + "`" + `verify+review` + "`" + ` | Every Go Test/Fuzz/executable Example and Vitest test carries attached spec references (` + "`" + `// spec:` + "`" + ` or ` + "`" + `//aip:verifies` + "`" + `) resolved by ` + "`" + `task test:spec` + "`" + `; syntactic attribution does not prove semantic coverage | ` + "`" + `ateam test-spec` + "`" + `; ` + "`" + `.agents/protocol.md` + "`" + ` |
| ` + "`verify`" + ` | Go packages pass ` + "`gofmt`" + `, ` + "`go vet`" + `, and ` + "`revive`" + ` with zero warnings | ` + "`Taskfile.yml`" + ` |
| ` + "`review`" + ` | Agent replies end with 1–2 lines stating what was skipped/not checked and any residual risk | review |

## 2. Hooks, MCP budget, and multi-agent parity

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| ` + "`verify`" + ` | ` + "`AGENTS.md`" + ` stays under 24,000 bytes, ` + "`CLAUDE.md`" + ` only imports ` + "`@AGENTS.md`" + `, and rule row counts match ` + "`.agents/rules-index.md`" + ` | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | Canonical skills in ` + "`.agents/skills/*/SKILL.md`" + ` have matching thin adapters in ` + "`.claude/skills/`" + ` and ` + "`.gemini/commands/`" + ` | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | Hook configs in ` + "`.claude/settings.json`" + `, ` + "`.codex/hooks.json`" + `, ` + "`.gemini/settings.json`" + `, and ` + "`.agents/hooks.json`" + ` map native events to ` + "`ateam hook`" + `; with ` + "`ATEAM_HOOK_SCOPE=user`" + `, ` + "`ateam doctor`" + ` also accepts one complete local or user-scope file (` + "`.claude/settings.local.json`" + `, ` + "`~/.claude/settings.json`" + `, ` + "`~/.codex/hooks.json`" + `, ` + "`~/.gemini/settings.json`" + `) and names the file under ` + "`hook_sources`" + ` | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | ` + "`ateam mcp`" + ` exposes only the 2 multiplexed tools (` + "`ateam_state`" + `, ` + "`ateam_task`" + `) with ` + "`<300`" + ` total schema tokens | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | ` + "`ateam hook`" + ` executes normalized lifecycle events in ` + "`<10ms`" + ` target budget (` + "`doctor`" + ` warns ` + "`>10ms`" + `, fails ` + "`--check`" + ` ` + "`>25ms`" + `) with zero external runtime dependencies | ` + "`ateam doctor`" + ` |

## 3. State, OpenSpec, Forge (GitHub/Gitea), and Definition of Done

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| ` + "`review`" + ` | Clarify Deliverable, Scope/non-goals, Definition of Done, and Stop boundary before mutating state | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify+review`" + ` | Shared state syncs with GitHub (` + "`gh`" + `/API) or Gitea (` + "`tea`" + `/API v1) and git worktrees, cached in gitignored ` + "`.ateam/state.json`" + ` | ` + "`ateam state pickup`" + ` |
| ` + "`verify`" + ` | OpenSpec changes/specs validate via ` + "`spec:check`" + `, and ` + "`Hold:`" + ` / caveats (` + "`HALT`" + `, ` + "`RED`" + `, ` + "`BLOCKED`" + `, ` + "`WONTDO`" + `, ` + "`OPEN-Q`" + `) are parsed by ` + "`spec:queue`" + ` | ` + "`ateam state spec-check`" + ` |
| ` + "`review`" + ` | Non-trivial work's OpenSpec ` + "`proposal.md`" + ` stays one page (≤ 60 lines; ` + "`spec:check`" + ` warns past it) and carries ` + "`Approved-by:`" + `, empty until the owner fills ` + "`<name> (<YYYY-MM-DD>)`" + `; the change's first task holds ` + "`Hold: OPEN-Q proposal awaits owner approval`" + ` until the owner removes it in the same edit, and an agent whose Stop boundary is proposal approval stops there and hands off | ` + "`ateam state spec-check`" + `; review |
| ` + "`verify`" + ` | ` + "`ateam bench --selftest`" + ` verifies good/bad reference fixtures and mutation detection before any live benchmark run | ` + "`ateam bench --selftest`" + ` |
| ` + "`verify`" + ` | ` + "`task verify`" + ` runs formatting, OpenSpec check, test-to-spec attribution, lint, unit/parity tests, benchmark self-test, and UI checks on a clean tree | ` + "`task verify`" + ` |
| verify+review | Review requires an independent reviewer not weaker than every implementer by model tier, then effort band, optional provider separation and current trusted-base evidence; the dormant configured path adds scoped owner ceiling exceptions | ateam review; .agents/contracts/reviewer.md |
` + indexMergeCheckRow + `
` + closeRuleRow + `

`

const defaultProtocolMD = `# Shared Work Protocol (Claude, Codex, and Gemini)

ATeam is an opinionated experimentation lab primarily grounded in Go development. Initialization uses the same templates for all project languages; it does not detect language support or create a Taskfile. Review and adapt these instructions before adoption. Go mutation checks require non-test .go source; verification targets belong to the adopting project. Shortcut scanning covers only .go, .ts, .js, .svelte, .sh, .yml, and .yaml files (excluding Go tests and .test.ts files). Other languages are not validated as complete workflows. Ponytail modes change injected guidance, not model settings or guaranteed compliance. Benchmark self-tests validate reference fixtures, not improved agent performance.

Canonical protocol for ` + "`ateam`" + ` and repositories bootstrapped with ` + "`ateam init`" + `. ` + "`AGENTS.md`" + ` and ` + "`CLAUDE.md`" + ` point here and inline the four workflow gates (**Clarify, Claim, Merge, Close**).

## 1. Hybrid State Architecture (GitHub + Gitea + OpenSpec + Local Cache)

Shared truth lives in the repository's versioned code, ` + "`openspec/`" + ` specs and changes, git worktrees, and the repository's remote forge (**GitHub** or **Gitea**), backed by a fast gitignored local cache (` + "`.ateam/state.json`" + `) so CLI lifecycle hooks execute in ` + "`<10ms`" + ` without blocking on network calls:

- **What is wanted & decided (GitHub & Gitea):** Issues with labels ` + "`type:epic`" + `, ` + "`status:needs-decision`" + `, ` + "`status:blocked`" + `, ` + "`class:flake`" + `.
` + protocolClaimState + `
- **Target state, task truth & holds (` + "`openspec/`" + `):** ` + "`openspec/specs/<capability>/spec.md`" + ` holds current truth; ` + "`openspec/changes/<id>/`" + ` holds in-flight changes validated by ` + "`task spec:check`" + ` and ` + "`task spec:queue`" + `.
- **Intentional shortcuts (` + "`// shortcut:`" + `):** Deferred complexity is marked inline in source code as ` + "`// shortcut: <the limit>, <when to upgrade>`" + ` and scanned live by ` + "`ateam state debt`" + `.
- **Session handoff & pre-compaction cache:** ` + "`ateam state handoff`" + ` snapshots live git branch/HEAD/dirty status, OpenSpec queue/holds, forge issues/PRs, and ` + "`// shortcut:`" + ` items alongside the 5 semantic handoff fields (` + "`Goal`" + `, ` + "`Addresses`" + `, ` + "`Constraints`" + `, ` + "`Done when`" + `, ` + "`Checks to run`" + `).

## 2. Work Lifecycle & Definition of Done

1. **Clarify:** Before mutating state on an underspecified request, inspect the tree first, then confirm Deliverable, Scope/non-goals, Definition of Done, and Stop boundary. For non-trivial work the default Stop boundary is proposal approval: a one-page ` + "`proposal.md`" + ` (≤ 60 lines; detail goes in ` + "`tasks.md`" + ` or ` + "`design.md`" + `) with an empty ` + "`Approved-by:`" + ` line, and a first task carrying ` + "`Hold: OPEN-Q proposal awaits owner approval`" + ` until the owner fills ` + "`Approved-by: <name> (<YYYY-MM-DD>)`" + ` and removes the hold in the same edit; the agent stops at the hold and hands off.
` + protocolClaimStep + `
3. **Execute (Ponytail Decision Ladder):** Take the smallest complete change that solves the task, reuse stdlib/existing code, add a focused test using an independent oracle for every non-trivial branch/parser, and tag deliberate limits with ` + "`// shortcut: <the limit>, <when to upgrade>`" + `.
` + protocolVerifyStep + `
` + protocolCloseStep + `

## 3. Mutation & Property-Based Testing (PBT) Discipline

1. **Independent oracle & wrong behavior first:** State the correct behavior and one plausible wrong behavior; compute expected outcomes from an independent oracle or reference model, never by repeating production logic.
2. **Non-destructive mutation check (` + "`task mutate:check`" + ` / ` + "`ateam mutate`" + `):** Never run ` + "`git checkout --`" + `, ` + "`git restore`" + `, ` + "`git stash`" + `, ` + "`git clean`" + `, or ` + "`git reset --hard`" + ` to undo a mutant. Use ` + "`task mutate:check`" + ` (` + "`go test -overlay`" + ` on a copy outside the repo) or ` + "`cp`" + ` + ` + "`shasum -a 256`" + ` before/after, and report ` + "`detection`" + `, ` + "`survivor`" + `, ` + "`invalid`" + `, or ` + "`inconclusive`" + `.
3. **Generated checks (Fuzz targets & PBT):** When touching interacting inputs, stated laws (round-trip, idempotence), or order-dependent histories, record whether ` + "`testing.F`" + ` / PBT is used or why named examples suffice, verify boundary reach and assertion activation, and compare histories against a test-owned reference model.
## Test-to-spec attribution

Run ` + "`" + `ateam test-spec` + "`" + ` (or ` + "`" + `task test:spec` + "`" + `) to deterministically validate every Go ` + "`" + `Test` + "`" + `, ` + "`" + `Fuzz` + "`" + `, and executable ` + "`" + `Example` + "`" + ` declaration and every Vitest ` + "`" + `it` + "`" + `/` + "`" + `test` + "`" + ` declaration. Attach one or more line comments directly to the declaration, naming existing repository-relative specifications and exact requirement headings:

` + "`" + `` + "`" + `` + "`" + `go
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestOpenSpecQueueAndCheck(t *testing.T) {
    // assertions
}
` + "`" + `` + "`" + `` + "`" + `

Multiple requirements use separate ` + "`" + `// spec:` + "`" + ` lines. Go subtests inherit the top-level declaration; each Vitest test needs its own immediately preceding comment, without inheritance from ` + "`" + `describe` + "`" + `. Missing, malformed, dangling, and escaping references fail the gate. Keep references current when requirements move or are renamed. The checker excludes benchmark functions and fixture source inside string literals; generated test source is not exempt. Dependency, VCS, worktree, and build-output directories are excluded.

A test may instead (or also) carry ` + "`" + `//aip:verifies <IDs>` + "`" + ` (no space after ` + "`" + `//` + "`" + `; the spaced form is a diagnostic), naming one or more space-separated scenario IDs. Each ID resolves when a line outside code fences in a canonical ` + "`" + `openspec/specs/*/spec.md` + "`" + ` or in-flight ` + "`" + `openspec/changes/<change>/specs/*/spec.md` + "`" + ` (never ` + "`" + `archive/` + "`" + `) is ` + "`" + `#### Scenario: <ID>` + "`" + ` followed by a space or end of line; an empty or unknown ID fails.

Vitest scanning requires Node and TypeScript installed at ui/node_modules/typescript in the target repository. Supported filenames end in .test or .spec plus .ts, .tsx, .js, or .jsx; .mjs/.cjs and other frameworks are outside this gate. Direct Vitest imports (including aliases), namespace calls, globals, modifiers, and parameterized declarations are recognized; arbitrary wrapper modules and runtime expansion counts are not covered. A blank line detaches a citation; block comments do not count as citation lines. Directory symlinks are not followed and symlink test files are rejected. The checker includes untracked supported tests and all Go build tags. It skips .git, .worktrees, node_modules, vendor, .svelte-kit, .enjoy-logs, ui/build, internal/web/dist, coverage, bin, benchmarks/runs, and .ateam. Other testdata files are not exempt.

Attribution validates syntax and reference existence, not whether assertions prove the requirement or whether all specifications have tests. Review must assess those claims separately. The current supported languages are Go and Vitest JavaScript/TypeScript; adopting projects must provide appropriate attribution enforcement for other frameworks and languages.

## 4. Configured model review

Install owner-approved comparisons and forge author/owner logins with ateam review setup --policy FILE. No universal model ranking is supplied. Run ateam review route before implementation and revalidate all actual model escalations. Native clients must be launched with the selected model/effort; role cards do not pin or attest runtime identity.

A separate read-only reviewer must outrank every actual implementer. Cross-provider review is optional and additional; reasoning effort is not model strength. If no stronger model exists, approval blocks without an explicit owner exception scoped to head/reviewer/policy. Independence, current evidence, configured checks and verdict remain required.

Use ateam review check --pr N --record-comment ID with optional --exception-comment ID. The gate reads policy only from the forge-verified current repository default branch, accepts only configured review/owner accounts, and refuses stale or known edited evidence. All paths require review. Shared logins, model/session claims and same-timestamp comment edits remain attestation limits; do not claim runtime or body authorship is proven.

`

var canonicalSkillBodies = map[string]string{
	"handoff": `---
name: handoff
description: >-
  Checkpoint current repository work for continuation or transfer across Claude, Codex, and Gemini.
  Reconciles live git/test/hold state and emits the 5-field pickup prompt.
---

# Checkpoint and Hand Off Work

Read [the shared protocol](../../protocol.md) first. A handoff records current truth in shared homes (GitHub/Gitea PR/issue comments and the structured ` + "`.ateam/state.json`" + ` snapshot) without creating competing prose handoff files in the repo tree.

## 1. Reconcile the Checkpoint

Call the ` + "`ateam_state`" + ` MCP tool (` + "`{\"action\": \"handoff\", ...}`" + `) or run ` + "`ateam state handoff`" + ` with the 5 semantic fields:

- ` + "`goal`" + `: PR/issue and the authorized outcome
- ` + "`addresses`" + `: PR, worktree, branch, checkpoint SHA, and evidence locations
- ` + "`constraints`" + `: Decision/hold pointers, local access limits, and write-ownership transfer status
- ` + "`done_when`" + `: Acceptance criteria and remaining verification gates
- ` + "`checks_to_run`" + `: Cheap reconciliation commands and remaining unrun validation

## 2. Return the 5-Field Pickup Prompt

Output the canonical 5-field block and instruct the next session to run ` + "`/pickup`" + ` (` + "`ateam state pickup`" + `):

` + "```text" + `
Goal: <PR/issue and authorized outcome>
Addresses: <PR, worktree, branch, checkpoint SHA, evidence locations>
Constraints: <Decision/hold pointers, local access limits, ownership transfer status>
Done when: <Acceptance criteria and remaining boundary>
Checks to run: <Reconciliation commands and remaining validation>
` + "```" + `
`,
	"pickup": `---
name: pickup
description: >-
  Resume a repository effort across Claude, Codex, or Gemini by reconciling live git/worktree state,
  GitHub/Gitea issues/PRs, Hold: items, and the last structured handoff checkpoint.
---

# Pick Up Repository Work

Read [the shared protocol](../../protocol.md) first. Treat any handoff block or cached ` + "`.ateam/state.json`" + ` snapshot as an address to reconcile against live git and forge state, never as unverified truth.

## 1. Locate and Measure

Prefer calling the ` + "`ateam_state`" + ` MCP tool (` + "`{\"action\": \"pickup\"}`" + `) or running ` + "`ateam state pickup`" + ` to inspect current branch, ` + "`HEAD`" + ` SHA, dirty status, worktrees, active ` + "`Hold:`" + ` items, ` + "`// shortcut:`" + ` debt, and any drift since the last handoff checkpoint SHA.

## 2. Verify Write Ownership

Confirm the previous writer has handed off or stopped before mutating files in an existing worktree. For unclaimed work, follow the Claim gate in [protocol.md](../../protocol.md).

## 3. Reconcile and Continue

Report verified worktree/branch/HEAD SHA, active holds/drift, write ownership status, and the next authorized step under the Ponytail decision ladder.
`,
	"ponytail": `---
name: ponytail
description: >-
  Lazy senior dev discipline: take the smallest complete change that solves the task, protect trust
  boundaries and tests, track // shortcut: debt, and end with skipped/risk lines. Levels: lite, full, ultra, off.
argument-hint: "[lite|full|ultra|off|review|audit|debt]"
---

# Ponytail Discipline Ladder

You are a lazy senior developer. The best code is the code never written. Solve the whole problem with the least new code, and end every reply with 1–2 lines: **what you skipped or did not check**, and **any risk the user must know**.

## 1. Before You Write

Read the task and the code it touches. List callers, tests, fixtures, config, and exports in scope.

## 2. The 6-Step Decision Ladder (Smallest Complete Change)

1. **Does it need to exist?** Skip unrequested features and options; name them in one line.
2. **Already in this codebase?** Reuse existing helpers, components, and patterns.
3. **Standard library or platform feature?** Prefer the project language’s standard library or native browser capabilities.
4. **An installed dependency?** Use what is already installed; never add a dependency for a few lines.
5. **Can it be one clear line?** Write flat, readable code.
6. **Otherwise:** Write the minimum complete change, test every non-trivial branch with an independent oracle proven via ` + "`task mutate:check`" + ` (or ` + "`cp`" + ` + SHA-256), use generated checks (` + "`testing.F`" + ` / PBT) when touching interacting inputs/laws/histories, and mark deliberate limits with ` + "`// shortcut: <the limit>, <when to upgrade>`" + `.
`,
	"preflight": `---
name: preflight
description: >-
  Select and execute Taskfile verification gates for the current diff and record reproducible evidence.
  Use before implementation pushes, handoffs, or concluding a task.
---

# Verify Changes (Preflight)

Read [the shared protocol](../../protocol.md) first. One ` + "`Taskfile.yml`" + ` entrypoint serves local verification, MCP (` + "`ateam_task`" + `), and CI.

## 1. Select Gates by Diff Scope

Inspect ` + "`git status --short`" + ` and ` + "`git diff --stat`" + `, run ` + "`task test:spec`" + ` for test/spec changes (syntax and reference existence, not semantic coverage), run ` + "`task mutate:check`" + ` for non-trivial refusals/branches, and run ` + "`task verify`" + ` (via ` + "`ateam_task`" + ` or CLI) before any push or completion.

## 2. Record Evidence

Record the exact command, exit status, tested ` + "`HEAD`" + ` SHA, and the 6-line testing block (` + "`Decision`" + `, ` + "`Rule`" + `, ` + "`Inputs`" + `, ` + "`Run`" + `, ` + "`Mutation`" + `, ` + "`Not covered`" + `) when touching parsers/laws/histories, ending with Ponytail's 1–2 line skipped/risk summary.
`,
	"review-pr": `---
name: review-pr
description: >-
  Review one pull request as an independent, read-only reviewer and post the record comment
  merge-check reads. The manual trigger for a cross-agent review.
---

# Review a Pull Request

Read [the shared protocol](../../protocol.md) and [the reviewer contract](../../contracts/reviewer.md) first. Review in a separate run that wrote none of the change. The deliverable stays read-only; the author makes corrections.

## 1. Pick the PR

Use the PR the user named. Otherwise list the open, non-draft PRs whose ` + "`implemented-by:`" + ` line does not name you and whose comments hold no record (` + "`reviewed-by:`" + ` with ` + "`Reviewed at <sha>`" + `) for the current head: ` + "`gh pr list --state open --json number,title,isDraft,headRefOid,body`" + `, then ` + "`gh pr view <n> --comments`" + ` (` + "`tea`" + ` on Gitea). One candidate: review it and say which. Zero or several: ask which PR to review, then stop.

## 2. State the Problem and Your Own Approach

Read the PR description and the issue it closes or addresses. State the problem in one sentence. Before opening the diff, decide how you would solve it: the files you would touch, the shape of the change, and what you would not build. The diff must not anchor you.

## 3. Review

Check out the PR head detached at ` + "`.worktrees/<agent>-review-pr-<n>`" + ` (` + "`git worktree add --detach .worktrees/<agent>-review-pr-<n> <head-sha>`" + `), never in the author's worktree. Read the complete base-to-head diff and the callers, tests, fixtures, and configuration it touches. Compare it with your approach: a difference is a finding only when the PR's choice is worse. Pragmatic, idiomatic, simple yet thorough code beats complex or clever code (the Ponytail ladder). For each finding, try to refute it, then report the failure mechanism, location, minimal correction, and how to verify. On a code PR, run ` + "`task verify`" + ` in the checkout and record the result verbatim.

## 4. Post the Record

Post one PR comment in the contract grammar (` + "`gh pr comment <n> --body-file <file>`" + `):

` + "`" + `` + "`" + `` + "`" + `text
reviewed-by: <agent> (<model>, <effort>)
Reviewed at <head-sha>
Verdict: APPROVE | REQUEST CHANGES
<findings, checks run, unavailable evidence, limitations>
` + "`" + `` + "`" + `` + "`" + `

Name your provider, exact model, and reasoning effort. Remove the review checkout (` + "`git worktree remove .worktrees/<agent>-review-pr-<n>`" + `) once the comment is posted. End your reply with 1–2 lines naming what you skipped or did not check and any residual risk.
`,
}

const agyHooksJSON = `{
  "ateam": {
    "SessionStart": [
      {
        "type": "command",
        "command": "ateam hook --event session-start --agent agy"
      }
    ],
    "PreInvocation": [
      {
        "type": "command",
        "command": "ateam hook --event before-prompt --agent agy"
      }
    ],
    "Stop": [
      {
        "type": "command",
        "command": "ateam hook --event turn-end --agent agy"
      }
    ]
  }
}
`

const agyPluginJSON = `{
  "name": "ateam"
}
`

const agyMCPConfigJSON = `{
  "mcpServers": {
    "ateam": {
      "command": "ateam",
      "args": ["mcp"]
    }
  }
}
`

const defaultAgentsReadmeMD = `# Agent & Platform Configuration (` + "`.agents/`" + `)

` + "`.agents/`" + ` holds the single set of instructions, workflow rules, and skills shared across **Claude Code**, **Codex CLI**, **Gemini CLI**, and **Antigravity (` + "`agy`" + `)**. The files under ` + "`.claude/`" + `, ` + "`.codex/`" + `, and ` + "`.gemini/`" + ` are thin pointers generated by ` + "`ateam init`" + ` and checked by ` + "`ateam doctor`" + `.

Optional model-review setup uses .agents/review-policy.json and .agents/contracts/{developer,reviewer}.md. Run ateam review setup --policy FILE to generate role cards and applicable pointers without overwriting custom files. Operators select the runtime model explicitly; doctor checks configured instruction drift, not runtime model identity.

Existing client configuration is preserved even with --force; merge missing MCP entries manually using https://github.com/C360Studio/ateam/blob/main/docs/mcp.md. MCP configuration belongs in .mcp.json for Claude, .codex/config.toml for Codex, and .gemini/settings.json for Gemini. Doctor checks local structure, not live connection or client trust. Initialization does not detect the project language or create a Taskfile. Adapt the Go-first instructions and supply project verification tasks; local parity checks do not certify live-agent integration. See the adoption scope in AGENTS.md and .agents/protocol.md.

| Capability | Claude Code | Codex CLI | Gemini CLI | Antigravity (` + "`agy`" + `) |
| --- | --- | --- | --- | --- |
| Instructions | ` + "`CLAUDE.md`" + ` (` + "`@AGENTS.md`" + `) | ` + "`AGENTS.md`" + ` | ` + "`.gemini/settings.json`" + ` (` + "`contextFileName: \"AGENTS.md\"`" + `) | ` + "`AGENTS.md`" + ` |
| Canonical Skills | ` + "`.agents/skills/<name>/SKILL.md`" + ` | ` + "`.agents/skills/<name>/SKILL.md`" + ` | ` + "`.agents/skills/<name>/SKILL.md`" + ` | ` + "`.agents/skills/<name>/SKILL.md`" + ` |
| Skill Adapters | ` + "`.claude/skills/<name>/SKILL.md`" + ` | Reads ` + "`.agents/skills/`" + ` directly | ` + "`.gemini/commands/<name>.toml`" + ` | Reads ` + "`.agents/skills/`" + ` directly |
| Hook Config | ` + "`.claude/settings.json`" + ` | ` + "`.codex/hooks.json`" + ` | ` + "`.gemini/settings.json`" + ` | ` + "`.agents/hooks.json`" + ` |
| MCP Config | ` + "`.mcp.json`" + ` | ` + "`.codex/config.toml`" + ` | ` + "`.gemini/settings.json`" + ` | ` + "`.agents/plugins/ateam/mcp_config.json`" + ` |
| Auto-Compaction / Context Cap | ` + "`.claude/settings.json`" + ` (` + "`autoCompactWindow: 500000`" + `) | ` + "`.codex/config.toml`" + ` (` + "`model_context_window = 500000`" + `, ` + "`model_auto_compact_token_limit = 450000`" + `) | ` + "`.gemini/settings.json`" + ` (` + "`model.chatCompression.contextPercentageThreshold: 0.5`" + `) | Built-in checkpointing + manual ` + "`/compact`" + `, ` + "`/context`" + `, ` + "`/btw`" + `, subagents |
`

func defaultTemplateFiles() map[string]string {
	m := map[string]string{
		".agents/contracts/developer.md":        developerContract,
		".agents/contracts/reviewer.md":         reviewerContract,
		"AGENTS.md":                             defaultAgentsMD,
		"CLAUDE.md":                             "# ATeam\n\n@AGENTS.md\n",
		".agents/README.md":                     defaultAgentsReadmeMD,
		".agents/protocol.md":                   defaultProtocolMD,
		".agents/rules-index.md":                defaultRulesIndexMD,
		".agents/hooks.json":                    agyHooksJSON,
		".agents/plugins/ateam/plugin.json":     agyPluginJSON,
		".agents/plugins/ateam/mcp_config.json": agyMCPConfigJSON,
		".claude/settings.json":                 claudeSettingsJSON,
		".codex/hooks.json":                     codexHooksJSON,
		".codex/config.toml":                    codexMCPConfig,
		".mcp.json":                             claudeMCPJSON,
		".gemini/settings.json":                 geminiSettingsJSON,
		"openspec/config.yaml":                  "schema: spec-driven\n",
		"openspec/changes/.gitkeep":             "# keep\n",
		"openspec/specs/.gitkeep":               "# keep\n",
	}
	for _, s := range canonicalSkills {
		body, ok := canonicalSkillBodies[s]
		if !ok {
			body = "---\nname: " + s + "\ndescription: Canonical " + s + " skill.\n---\n\n# " + s + "\n\n## 1. Workflow\n\nFollow `.agents/protocol.md`.\n"
		}
		m[".agents/skills/"+s+"/SKILL.md"] = body
		m[".claude/skills/"+s+"/SKILL.md"] = "---\nname: " + s + "\ndescription: Adapter for " + s + ".\n---\n\nRead [.agents/skills/" + s + "/SKILL.md](../../../.agents/skills/" + s + "/SKILL.md) fully and follow the canonical workflow.\n"
		m[".gemini/commands/"+s+".toml"] = "description = \"Run " + s + " workflow.\"\nprompt = \"Read .agents/skills/" + s + "/SKILL.md fully and follow the canonical workflow.\"\n"
	}
	return m
}

const agentsMergeCheckRow = `| ` + "`merge:check+review`" + ` | Non-draft code PRs (any file outside ` + "`.md`" + ` and ` + "`openspec/**/*.{md,yml,yaml,json}`" + `, plus ` + "`.claude/agents/`" + ` and ` + "`.agents/contracts/`" + `) carry ` + "`implemented-by:`" + ` and ` + "`reviewed-by:`" + ` each naming an agent (` + "`claude`" + `, ` + "`codex`" + `, ` + "`gemini`" + `) with its model and, once ` + "`.agents/review-policy.json`" + ` exists, native effort in parentheses (` + "`claude (opus, high)`" + `), the reviewer not weaker by tier then band (` + "`ateam review compare`" + `) and a different agent only when the policy sets ` + "`cross_provider`" + `, plus a PR comment review record (` + "`reviewed-by: <agent>`" + ` or ` + "`### Independent Cross-Agent Review`" + `) naming a PR commit with only clean ` + "`main`" + ` merges or docs-only commits after it, checked alongside active ` + "`main`" + ` ruleset and open ` + "`class:flake`" + ` issues by ` + "`task merge:check`" + ` | ` + "`scripts/merge-check.sh`" + `; ` + "`.agents/protocol.md`" + ` |`

const indexMergeCheckRow = `| ` + "`merge:check+review`" + ` | Non-draft code PRs (any file outside ` + "`.md`" + ` and ` + "`openspec/**/*.{md,yml,yaml,json}`" + `, plus ` + "`.claude/agents/`" + ` and ` + "`.agents/contracts/`" + `) carry ` + "`implemented-by:`" + ` and ` + "`reviewed-by:`" + ` each naming an agent (` + "`claude`" + `, ` + "`codex`" + `, ` + "`gemini`" + `) with its model and, once ` + "`.agents/review-policy.json`" + ` exists, native effort in parentheses (` + "`claude (opus, high)`" + `), the reviewer not weaker by tier then band (` + "`ateam review compare`" + `) and a different agent only when the policy sets ` + "`cross_provider`" + `, plus a PR comment review record (` + "`reviewed-by: <agent>`" + ` or ` + "`### Independent Cross-Agent Review`" + `) naming a PR commit with only clean ` + "`main`" + ` merges or docs-only commits after it, checked alongside active ` + "`main`" + ` ruleset and open ` + "`class:flake`" + ` issues by ` + "`task merge:check`" + ` | ` + "`scripts/merge-check.sh`" + ` |`

const protocolVerifyStep = `4. **Verify (4-Gate Definition of Done):** ` + "`task verify`" + ` green, ` + "`ateam doctor`" + ` parity green, ` + "`ateam bench --selftest`" + ` green, ` + "`implemented-by:`" + ` and a not-weaker ` + "`reviewed-by:`" + ` recorded (` + "`task merge:check -- <pr>`" + `) in a PR comment starting ` + "`reviewed-by: <agent> (<model>, <effort>)`" + ` and naming one PR commit as ` + "`Reviewed at <sha>`" + ` (template: ` + "`.agents/contracts/reviewer.md`" + `; a later commit that is not a clean merge of ` + "`main`" + ` or docs-only needs a new record), and Ponytail exit summary stated.`

const closeRuleRow = `| ` + "`review`" + ` | A PR archives its completed OpenSpec change before it is marked ready; once its required check is green on an up-to-date branch and the squash auto-merge (GitHub or Gitea) has landed, the claiming agent removes its worktree and remote branch (the ` + "`main`" + ` run is informational), and a review checkout is removed when its record is posted | ` + "`.agents/protocol.md`" + ` § Close; ` + "`ateam state pickup`" + ` worktree labels |`

const protocolClaimState = `- **Who has claimed what:** A draft PR on an agent-prefixed branch (` + "`claude/<topic>`" + `, ` + "`codex/<topic>`" + `, ` + "`gemini/<topic>`" + `) in its own worktree (` + "`git worktree list`" + `), with ` + "`Closes #n`" + ` or ` + "`Addresses #n`" + `.`

const protocolClaimStep = `2. **Claim:** Create or switch to an isolated worktree on an agent-prefixed branch (` + "`claude/…`" + `, ` + "`codex/…`" + `, ` + "`gemini/…`" + `), backed by a draft PR on GitHub or Gitea.`

const protocolCloseStep = `5. **Handoff or Land (Close):** At a session boundary or before context compaction, run ` + "`ateam state handoff`" + ` (or ` + "`/handoff`" + `) to record the 5-field pickup prompt. When a PR lands an OpenSpec change, the archive move (` + "`openspec/changes/<id>/`" + ` → ` + "`openspec/changes/archive/<date>-<id>/`" + `) is a commit on the branch before the PR is marked ready, and the change's ` + "`Hold: OPEN-Q proposal awaits owner approval`" + ` must be gone before that move; merging with a completed unarchived change is a Close-gate miss, not a follow-up. Mark the PR ready and arm auto-merge (` + "`gh pr merge --squash --auto`" + `; on Gitea, ` + "`POST /repos/{owner}/{repo}/pulls/{n}/merge`" + ` with ` + "`{\"do\":\"squash\",\"merge_when_checks_succeed\":true}`" + `); the merge fires when the required check is green on an up-to-date branch, and arming it past a previously red required job needs the cause fixed or an owner waiver. Once the merge has landed, remove the worktree you claimed and delete its remote branch. The ` + "`main`" + ` run is informational: a red one after a green up-to-date PR run is a tracked issue, not a reason to keep the worktree. Review checkouts (` + "`<agent>/review-pr-<n>`" + `, detached) are removed when the review record comment is posted. Pickup labels every merged and PR-less worktree; a finished worktree older than a day is yours to explain or remove.`
