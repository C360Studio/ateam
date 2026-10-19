package bootstrap

const hookSettingsJSON = `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event session-start"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event before-prompt"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event pre-compact"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event turn-end"
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

const geminiSettingsJSON = `{
  "contextFileName": "AGENTS.md",
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event session-start"
          }
        ]
      }
    ],
    "BeforeAgent": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event before-prompt"
          }
        ]
      }
    ],
    "PreCompress": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event pre-compact"
          }
        ]
      }
    ],
    "AfterAgent": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ateam hook --event turn-end"
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

const defaultAgentsMD = `# Repository Agent Instructions

Guidance for coding agents. Codex and Gemini load this file as ` + "`AGENTS.md`" + `; ` + "`CLAUDE.md`" + ` imports it (` + "`@AGENTS.md`" + `), keeping a single copy strictly under 24,000 bytes.

## Ponytail decision ladder

You are a lazy senior developer. The best code is the code never written. Solve the whole problem with the least new code, and end every reply with 1–2 lines stating **what you skipped or did not check** and **any residual risk**.

1. **Does it need to exist?** Skip unrequested features, options, and abstractions.
2. **Already in this codebase?** Reuse existing helpers, components, and conventions.
3. **Standard library or platform feature?** Prefer Go stdlib and native platform features.
4. **Installed dependency?** Use what is already in ` + "`go.mod`" + ` / ` + "`package.json`" + `; never add a dependency for a few lines.
5. **Readable at a glance?** Prefer flat, obvious code over clever one-liners.
6. **Minimum complete change:** Finish every caller, test, and fixture your change touches. Mark any intentional deferral with ` + "`// shortcut: <the limit>, <when to upgrade>`" + `.

## Rules and what enforces them

### 1. Ponytail decision ladder and code discipline

| Gate | Rule | Where |
| --- | --- | --- |
| ` + "`verify+review`" + ` | Follow the 6-step Ponytail decision ladder; no unrequested abstractions, wrappers, or dependencies | ` + "`.agents/skills/ponytail/SKILL.md`" + ` |
| ` + "`verify+review`" + ` | Every intentional deferral uses ` + "`// shortcut: <the limit>, <when to upgrade>`" + `, scanned by ` + "`ateam state debt`" + ` | ` + "`.agents/skills/ponytail/SKILL.md`" + ` |
| ` + "`verify+review`" + ` | Non-trivial logic (branches, parsers, state transitions, boundaries) ships with a test proven able to fail | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify`" + ` | Go packages pass ` + "`gofmt`" + `, ` + "`go vet`" + `, and ` + "`revive`" + ` with zero warnings | ` + "`Taskfile.yml`" + ` |
| ` + "`review`" + ` | Agent replies end with 1–2 lines stating what was skipped/not checked and any residual risk | ` + "`.agents/skills/ponytail/SKILL.md`" + ` |

### 2. Hooks, MCP budget, and multi-agent parity

| Gate | Rule | Where |
| --- | --- | --- |
| ` + "`verify`" + ` | ` + "`AGENTS.md`" + ` stays under 24,000 bytes, ` + "`CLAUDE.md`" + ` only imports ` + "`@AGENTS.md`" + `, and rule row counts match ` + "`.agents/rules-index.md`" + ` | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | Canonical skills in ` + "`.agents/skills/*/SKILL.md`" + ` have matching thin adapters in ` + "`.claude/skills/`" + ` and ` + "`.gemini/commands/`" + ` | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | Hook configs in ` + "`.claude/settings.json`" + `, ` + "`.codex/hooks.json`" + `, and ` + "`.gemini/settings.json`" + ` map native events to ` + "`ateam hook`" + ` | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | ` + "`ateam mcp`" + ` exposes only the 2 multiplexed tools (` + "`ateam_state`" + `, ` + "`ateam_task`" + `) with ` + "`<300`" + ` total schema tokens | ` + "`.agents/README.md`" + ` |
| ` + "`verify`" + ` | ` + "`ateam hook`" + ` executes normalized lifecycle events in ` + "`<5ms`" + ` with zero external runtime dependencies | ` + "`.agents/README.md`" + ` |

### 3. State, OpenSpec, Forge (GitHub/Gitea), and Definition of Done

| Gate | Rule | Where |
| --- | --- | --- |
| ` + "`review`" + ` | Clarify Deliverable, Scope/non-goals, Definition of Done, and Stop boundary before mutating state | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify+review`" + ` | Shared state syncs with GitHub (` + "`gh`" + `/API) or Gitea (` + "`tea`" + `/API v1) and git worktrees, cached in gitignored ` + "`.ateam/state.json`" + ` | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify`" + ` | OpenSpec changes/specs validate via ` + "`spec:check`" + `, and ` + "`Hold:`" + ` / caveats (` + "`HALT`" + `, ` + "`RED`" + `, ` + "`BLOCKED`" + `, ` + "`WONTDO`" + `, ` + "`OPEN-Q`" + `) are parsed by ` + "`spec:queue`" + ` | ` + "`openspec/`" + ` |
| ` + "`verify`" + ` | ` + "`ateam bench --selftest`" + ` verifies good/bad reference fixtures and mutation detection before any live benchmark run | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify`" + ` | ` + "`task verify`" + ` runs formatting, OpenSpec check, lint, unit/parity tests, benchmark self-test, and UI checks on a clean tree | ` + "`Taskfile.yml`" + ` |
`

const defaultRulesIndexMD = `# Rules Enforcement Index

## 1. Ponytail decision ladder and code discipline

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| ` + "`verify+review`" + ` | Follow the 6-step Ponytail decision ladder; no unrequested abstractions, wrappers, or dependencies | ` + "`ateam hook`" + ` |
| ` + "`verify+review`" + ` | Every intentional deferral uses ` + "`// shortcut: <the limit>, <when to upgrade>`" + `, scanned by ` + "`ateam state debt`" + ` | ` + "`ateam state debt`" + ` |
| ` + "`verify+review`" + ` | Non-trivial logic (branches, parsers, state transitions, boundaries) ships with a test proven able to fail | ` + "`ateam bench --selftest`" + ` |
| ` + "`verify`" + ` | Go packages pass ` + "`gofmt`" + `, ` + "`go vet`" + `, and ` + "`revive`" + ` with zero warnings | ` + "`Taskfile.yml`" + ` |
| ` + "`review`" + ` | Agent replies end with 1–2 lines stating what was skipped/not checked and any residual risk | review |

## 2. Hooks, MCP budget, and multi-agent parity

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| ` + "`verify`" + ` | ` + "`AGENTS.md`" + ` stays under 24,000 bytes, ` + "`CLAUDE.md`" + ` only imports ` + "`@AGENTS.md`" + `, and rule row counts match ` + "`.agents/rules-index.md`" + ` | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | Canonical skills in ` + "`.agents/skills/*/SKILL.md`" + ` have matching thin adapters in ` + "`.claude/skills/`" + ` and ` + "`.gemini/commands/`" + ` | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | Hook configs in ` + "`.claude/settings.json`" + `, ` + "`.codex/hooks.json`" + `, and ` + "`.gemini/settings.json`" + ` map native events to ` + "`ateam hook`" + ` | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | ` + "`ateam mcp`" + ` exposes only the 2 multiplexed tools (` + "`ateam_state`" + `, ` + "`ateam_task`" + `) with ` + "`<300`" + ` total schema tokens | ` + "`ateam doctor`" + ` |
| ` + "`verify`" + ` | ` + "`ateam hook`" + ` executes normalized lifecycle events in ` + "`<5ms`" + ` with zero external runtime dependencies | ` + "`ateam doctor`" + ` |

## 3. State, OpenSpec, Forge (GitHub/Gitea), and Definition of Done

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| ` + "`review`" + ` | Clarify Deliverable, Scope/non-goals, Definition of Done, and Stop boundary before mutating state | ` + "`.agents/protocol.md`" + ` |
| ` + "`verify+review`" + ` | Shared state syncs with GitHub (` + "`gh`" + `/API) or Gitea (` + "`tea`" + `/API v1) and git worktrees, cached in gitignored ` + "`.ateam/state.json`" + ` | ` + "`ateam state pickup`" + ` |
| ` + "`verify`" + ` | OpenSpec changes/specs validate via ` + "`spec:check`" + `, and ` + "`Hold:`" + ` / caveats (` + "`HALT`" + `, ` + "`RED`" + `, ` + "`BLOCKED`" + `, ` + "`WONTDO`" + `, ` + "`OPEN-Q`" + `) are parsed by ` + "`spec:queue`" + ` | ` + "`ateam state spec-check`" + ` |
| ` + "`verify`" + ` | ` + "`ateam bench --selftest`" + ` verifies good/bad reference fixtures and mutation detection before any live benchmark run | ` + "`ateam bench --selftest`" + ` |
| ` + "`verify`" + ` | ` + "`task verify`" + ` runs formatting, OpenSpec check, lint, unit/parity tests, benchmark self-test, and UI checks on a clean tree | ` + "`task verify`" + ` |
`

const defaultProtocolMD = `# Shared Work Protocol (Claude, Codex, and Gemini)

Canonical protocol for ` + "`ateam`" + ` and repositories bootstrapped with ` + "`ateam init`" + `. ` + "`AGENTS.md`" + ` and ` + "`CLAUDE.md`" + ` point here and inline the four workflow gates (**Clarify, Claim, Merge, Close**).

## 1. Hybrid State Architecture (GitHub + Gitea + OpenSpec + Local Cache)

Shared truth lives in the repository's versioned code, ` + "`openspec/`" + ` specs and changes, git worktrees, and the repository's remote forge (**GitHub** or **Gitea**), backed by a fast gitignored local cache (` + "`.ateam/state.json`" + `) so CLI lifecycle hooks execute in ` + "`<5ms`" + ` without blocking on network calls:

- **What is wanted & decided (GitHub & Gitea):** Issues with labels ` + "`type:epic`" + `, ` + "`status:needs-decision`" + `, ` + "`status:blocked`" + `, ` + "`class:flake`" + `.
- **Who has claimed what:** A draft PR on an agent-prefixed branch (` + "`claude/<topic>`" + `, ` + "`codex/<topic>`" + `, ` + "`gemini/<topic>`" + `) in its own worktree (` + "`git worktree list`" + `), with ` + "`Closes #n`" + ` or ` + "`Addresses #n`" + `.
- **Target state, task truth & holds (` + "`openspec/`" + `):** ` + "`openspec/specs/<capability>/spec.md`" + ` holds current truth; ` + "`openspec/changes/<id>/`" + ` holds in-flight changes validated by ` + "`task spec:check`" + ` and ` + "`task spec:queue`" + `.
- **Intentional shortcuts (` + "`// shortcut:`" + `):** Deferred complexity is marked inline in source code as ` + "`// shortcut: <the limit>, <when to upgrade>`" + ` and scanned live by ` + "`ateam state debt`" + `.
- **Session handoff & pre-compaction cache:** ` + "`ateam state handoff`" + ` snapshots live git branch/HEAD/dirty status, OpenSpec queue/holds, forge issues/PRs, and ` + "`// shortcut:`" + ` items alongside the 5 semantic handoff fields (` + "`Goal`" + `, ` + "`Addresses`" + `, ` + "`Constraints`" + `, ` + "`Done when`" + `, ` + "`Checks to run`" + `).

## 2. Work Lifecycle & Definition of Done

1. **Clarify:** Before mutating state on an underspecified request, inspect the tree first, then confirm Deliverable, Scope/non-goals, Definition of Done, and Stop boundary.
2. **Claim:** Create or switch to an isolated worktree on an agent-prefixed branch (` + "`claude/…`" + `, ` + "`codex/…`" + `, ` + "`gemini/…`" + `), backed by a draft PR on GitHub or Gitea.
3. **Execute (Ponytail Decision Ladder):** Take the smallest complete change that solves the task, reuse stdlib/existing code, add a focused test for every non-trivial branch/parser, and tag deliberate limits with ` + "`// shortcut: <the limit>, <when to upgrade>`" + `.
4. **Verify (4-Gate Definition of Done):** ` + "`task verify`" + ` green, ` + "`ateam doctor`" + ` parity green, ` + "`ateam bench --selftest`" + ` green, and Ponytail exit summary stated.
5. **Handoff or Land:** At a session boundary or before context compaction, run ` + "`ateam state handoff`" + ` to record the 5-field pickup prompt.
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
3. **Standard library or platform feature?** Prefer Go stdlib or native browser capabilities.
4. **An installed dependency?** Use what is already installed; never add a dependency for a few lines.
5. **Can it be one clear line?** Write flat, readable code.
6. **Otherwise:** Write the minimum complete change, test every non-trivial branch, and mark deliberate limits with ` + "`// shortcut: <the limit>, <when to upgrade>`" + `.
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

Inspect ` + "`git status --short`" + ` and ` + "`git diff --stat`" + ` and run ` + "`task verify`" + ` (via ` + "`ateam_task`" + ` or CLI) before any push or completion.

## 2. Record Evidence

Record the exact command, exit status, and tested ` + "`HEAD`" + ` SHA, and end with Ponytail's 1–2 line skipped/risk summary.
`,
}

func defaultTemplateFiles() map[string]string {
	m := map[string]string{
		"AGENTS.md":                 defaultAgentsMD,
		"CLAUDE.md":                 "# ATeam\n\n@AGENTS.md\n",
		".agents/README.md":         "# Agent Profiles\n\nManaged by `ateam init` and verified by `ateam doctor`.\n",
		".agents/protocol.md":       defaultProtocolMD,
		".agents/rules-index.md":    defaultRulesIndexMD,
		".claude/settings.json":     hookSettingsJSON,
		".codex/hooks.json":         hookSettingsJSON,
		".gemini/settings.json":     geminiSettingsJSON,
		"openspec/config.yaml":      "schema: spec-driven\n",
		"openspec/changes/.gitkeep": "# keep\n",
		"openspec/specs/.gitkeep":   "# keep\n",
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
