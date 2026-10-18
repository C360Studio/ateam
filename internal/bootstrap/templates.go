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

func defaultTemplateFiles() map[string]string {
	m := map[string]string{
		"AGENTS.md":                 defaultAgentsMD,
		"CLAUDE.md":                 "# ATeam\n\n@AGENTS.md\n",
		".agents/README.md":         "# Agent Profiles\n\nManaged by `ateam init` and verified by `ateam doctor`.\n",
		".agents/protocol.md":       "# Shared Work Protocol\n\nClarify, Claim, Merge, Close.\n",
		".agents/rules-index.md":    defaultRulesIndexMD,
		".claude/settings.json":     hookSettingsJSON,
		".codex/hooks.json":         hookSettingsJSON,
		".gemini/settings.json":     geminiSettingsJSON,
		"openspec/config.yaml":      "schema: spec-driven\n",
		"openspec/changes/.gitkeep": "# keep\n",
		"openspec/specs/.gitkeep":   "# keep\n",
	}
	for _, s := range canonicalSkills {
		m[".agents/skills/"+s+"/SKILL.md"] = "---\nname: " + s + "\ndescription: Canonical " + s + " skill.\n---\n\n# " + s + "\n\nFollow `.agents/protocol.md`.\n"
		m[".claude/skills/"+s+"/SKILL.md"] = "---\nname: " + s + "\ndescription: Adapter for " + s + ".\n---\n\nRead [.agents/skills/" + s + "/SKILL.md](../../../.agents/skills/" + s + "/SKILL.md) fully and follow the canonical workflow.\n"
		m[".gemini/commands/"+s+".toml"] = "description = \"Run " + s + " workflow.\"\nprompt = \"Read .agents/skills/" + s + "/SKILL.md fully and follow the canonical workflow.\"\n"
	}
	return m
}
