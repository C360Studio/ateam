# Design: `t3-confidence-card-304`

## Context

Inventory taken at `cf49651`; every line below was opened this session.

- **Pickup skill.** `.agents/skills/pickup/SKILL.md:29-37` § 3 reports four facts: worktree, branch and HEAD; holds and drift; write ownership and evidence; the next step. Line 27 is already host-conditional prose ("Inside herdr (`HERDR_ENV=1`) …"), which is the precedent for a host paragraph. The adopter template is shorter (`internal/bootstrap/templates.go:432-448`, § 3 is one sentence at `:446`).
- **Handoff skill.** `.agents/skills/handoff/SKILL.md` § 2 emits the 5-field block. Its adopter mirror is at `templates.go:399-424`. The skill map is built by `defaultTemplateFiles()` (`templates.go:597`), and the thin adapters (`.claude/skills/*/SKILL.md`, `.gemini/commands/*.toml`) only point at the canonical file, so they do not change.
- **Sizes.** The canonical skills are 2,328 bytes (pickup) and 2,265 bytes (handoff). #200 tracks instruction-reading cost, so text every host reads matters.
- **T3 detection.** ateam already keys T3 on `T3CODE_THREAD_ID` (`cmd/ateam/reviewlaunch.go:59`; spec `openspec/specs/harness/spec.md:973`). `docs/adoption.md:211-218` records where it reaches:
  - Claude: in the process env, verified live 2026-11-07.
  - Codex: only in the shells Codex runs.
  - Antigravity: never, because T3 isolates it.
  - The model itself does not see env vars without running a shell.
- **The T3 tool.** `html_render` (T3 `apps/server/src/mcp/toolkits/html/tools.ts:78-104` at main `4f0b74e`; `:73-106` at `32a0531`) takes:
  - `html`: a complete document of 1 to 512,000 chars;
  - `title`: 1 to 200 chars;
  - `height`: an int from 80 to 2000.
  - All three are required. Its companion `html_preview` returns `contentHeight` but costs a headless-browser round trip. The tool's rules allow inline `<style>` and `<script>`, and remote URLs load as-is. It defaults to a dark theme and a 728 px column.
- **Card sources.** #302's statuses (`ateam/proposal`, `ateam/handoff`, `ateam/review-record`) and pickup's worktree line (`… open PR #n; review record at <sha>, <VERDICT>, <agent> (<model>, <effort>)`) supply the Proposal, Claim and Review rows. Gates and Merge come from `gh pr checks` / `gh pr view` when the agent ran them this turn.

## Goals / Non-Goals

The goal is one honest card per boundary, only in T3, with nothing extra for other hosts. For non-goals, see `proposal.md`.

## Decisions

### D1. Where the card HTML lives

- **A (recommended): one shared file, `.agents/skills/handoff/t3-card.html`.** Both skills name it in one sentence and read it only under the T3 condition. Each skill grows by about 400 bytes instead of about 3 KB, so the cost on non-T3 hosts is that sentence. `ateam init` writes the file; doctor does not require it (corrected at implementation). It forecloses nothing; a later `ateam state card` (Go) could render the same file.
- **B: inline the HTML in both SKILL.md files**, as the issue first sketched. Every pickup and handoff on every host would then read about 3 KB × 2 of HTML it never uses. That works against #200, and it is two copies to keep in sync.
- **Adopter seam:** an adopter who never uses T3 sees one extra conditional sentence per skill and one inert file. If they delete the file, the T3 card cannot render; doctor does not check it, the same as other non-required template files (corrected at implementation: the approved draft claimed doctor parity covered it).

### D2. The trigger

- **A (recommended): a single condition, `html_render` is among your tools and `printenv T3CODE_THREAD_ID` is non-empty.** Checking the tool first costs nothing. Only then does the agent spend one shell call, which also works for Codex, whose model sees the id only through a shell. Requiring both avoids misfiring on a different host that happens to expose a tool named `html_render`.
- **B: the env var alone.** That costs a shell call on every host at every pickup and handoff.
- **C: the tool alone.** That costs nothing, but it would trigger on any host that ships a tool with that name.

### D3. Height and honesty

- **Height:** the card has a fixed layout of five rows plus a labelled free-text region, so the skill passes a fixed `height` (about 420; the value is measured at implementation and written in the template's header comment) and skips `html_preview`. On mobile, T3 scrolls past the given height, so long handoff text cannot break the card.
- **Escaping:** the skill tells the agent to escape `& < > " '` in every inserted value.
- **Rows:** each row is a state word plus a value. A fact the agent did not read this turn is written `not checked` and never guessed. Handoff goal and done-when appear only inside the region headed "Agent-written (not verified)".
- **Template content:** no `<script>` and no remote URL, even though T3 permits both. The card then cannot fetch or run anything, and the bootstrap test can check that statically.

## Risks / Trade-offs

- **Agent-filled HTML.** The card is only as good as the agent's escaping and its honesty about `not checked`. A Go renderer would be stronger, but it is a non-goal here; `pickup --json` plus a renderer is the upgrade path if the hand render shows drift.
- **Upstream changes.** T3 can change the tool's name or parameters upstream. The trigger requires the tool to be present, so a rename makes the card silently absent rather than broken.
- **Verification is by hand.** The owner renders the card once in T3 on web and mobile; CI cannot render it.

## Migration Plan

None. The change is additive, and non-T3 hosts get identical behaviour.

## Open Questions

1. **OQ1 (owner ruling):** keep the card in one shared file (D1 A, recommended), or inline it in both skills (D1 B)?
2. **OQ2 (owner ruling):** for the trigger, require both the tool and the env var (D2 A, recommended), or use the env var alone (D2 B)?
3. **OQ3 (owner ruling):** you do the web and mobile hand render (recommended, since it needs your T3 devices), or waive it in favour of an `html_preview` screenshot taken from a T3 thread?
