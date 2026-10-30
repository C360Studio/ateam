// Package hook normalizes Claude Code, Codex CLI, and Gemini CLI lifecycle hook events in under 10ms.
package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/bootstrap"
	"github.com/c360studio/ateam/internal/state"
)

var inspectClientMCP = bootstrap.InspectClientMCPProcesses

// Input represents the normalized JSON payload passed on stdin by Claude, Codex, Gemini, or Antigravity hooks.
type Input struct {
	HookEventName  string `json:"hook_event_name,omitempty"`
	Event          string `json:"event,omitempty"`
	Prompt         string `json:"prompt,omitempty"`
	UserPrompt     string `json:"user_prompt,omitempty"`
	Cwd            string `json:"cwd,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

// SystemMessageStep wraps a persistent system message for Antigravity injectSteps output.
type SystemMessageStep struct {
	SystemMessage string `json:"systemMessage"`
}

// InjectStep represents a single step in Antigravity's injectSteps hook response.
type InjectStep struct {
	SystemMessage *SystemMessageStep `json:"systemMessage,omitempty"`
}

// Output represents the JSON response returned on stdout to Claude, Codex, Gemini, or Antigravity hooks.
type Output struct {
	AdditionalContext string          `json:"additionalContext,omitempty"`
	Decision          string          `json:"decision,omitempty"`
	Reason            string          `json:"reason,omitempty"`
	HookSpecific      *SpecificOutput `json:"hookSpecificOutput,omitempty"`
	InjectSteps       []InjectStep    `json:"injectSteps,omitempty"`
	EmitEmptyObject   bool            `json:"-"`
}

// HasPayload reports whether the hook output should be encoded as JSON to stdout.
func (o Output) HasPayload() bool {
	return o.AdditionalContext != "" || o.Decision != "" || o.HookSpecific != nil || len(o.InjectSteps) > 0 || o.EmitEmptyObject
}

// SpecificOutput provides compatibility with Claude/Gemini hookSpecificOutput wrappers.
type SpecificOutput struct {
	HookEventName     string `json:"hookEventName,omitempty"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

var modeCommandRe = regexp.MustCompile(`(?i)(?:^|\s)/(?:ponytail|mode)\s+(on|full|lite|ultra|off)\b`)

// NormalizeEvent maps CLI-specific hook names (Claude, Codex, Gemini, Antigravity) to canonical event names.
func NormalizeEvent(flagEvent, payloadEvent string) string {
	raw := strings.TrimSpace(flagEvent)
	if raw == "" {
		raw = strings.TrimSpace(payloadEvent)
	}
	switch strings.ToLower(raw) {
	case "session-start", "sessionstart":
		return "session-start"
	case "before-prompt", "userpromptsubmit", "beforeagent", "preinvocation":
		return "before-prompt"
	case "pre-compact", "precompact", "precompress":
		return "pre-compact"
	case "turn-end", "stop", "afteragent", "sessionend", "postinvocation":
		return "turn-end"
	default:
		return ""
	}
}

// Handle processes a hook event against root without external subprocesses so latency stays <10ms.
func Handle(root, flagEvent string, stdin io.Reader) (Output, error) {
	return HandleWithAgent(root, flagEvent, "", stdin)
}

// HandleWithAgent processes a hook event with an optional explicit agent identity (--agent claude|codex|gemini|agy).
func HandleWithAgent(root, flagEvent, flagAgent string, stdin io.Reader) (Output, error) {
	return handle(root, flagEvent, flagAgent, stdin, state.ResolveEnvSessionID())
}

// Probe runs a hook event as no session. For session-start that means it writes neither the writer lease nor
// the MCP-checked marker, so ateam doctor times it without stealing a live session's lease (#204); other events
// carry no such guarantee.
func Probe(root, flagEvent string) (Output, error) {
	return handle(root, flagEvent, "", strings.NewReader(`{}`), "")
}

func handle(root, flagEvent, flagAgent string, stdin io.Reader, envSessionID string) (Output, error) {
	var in Input
	if stdin != nil {
		if data, err := io.ReadAll(io.LimitReader(stdin, 1<<20)); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			_ = json.Unmarshal(data, &in)
		}
	}
	ev := NormalizeEvent(flagEvent, firstNonEmpty(in.HookEventName, in.Event))
	if ev == "" {
		return Output{}, nil
	}
	st, err := state.Load(root)
	if err != nil {
		return Output{}, err
	}
	sessionID := firstNonEmpty(in.SessionID, in.ConversationID, envSessionID)
	st.CurrentSessionID = sessionID
	br, sha := state.ReadFastGitHead(root)
	wtHeadChanged := false
	if sha != "" {
		wtHeadChanged = st.SyncBranchHead(br, sha)
	}
	st.CurrentAgent = state.ResolveActiveAgentForSession(root, flagAgent, sessionID, st.Git.Branch, st.Writer)

	switch ev {
	case "session-start":
		if shortcuts, err := state.ScanShortcuts(root); err == nil {
			st.Shortcuts = shortcuts
		}
		var scanErrs []string
		if holds, err := state.ScanHolds(root); err == nil {
			st.Holds = holds
		} else {
			scanErrs = append(scanErrs, err.Error())
		}
		if queue, err := state.ScanOpenSpecQueue(root); err == nil {
			st.OpenSpecQueue = queue
		} else {
			scanErrs = append(scanErrs, err.Error())
		}
		st.HoldScanError = strings.Join(scanErrs, "; ")
		reconcileSessionStartHandoffs(root, flagAgent, br, sha, wtHeadChanged, st)
		ctxText := state.BuildPickupSummary(st)
		if sessionID != "" && !strings.Contains(ctxText, "WARNING: another writer may be active") {
			_, _ = state.WriteWriterLease(root, flagAgent, sessionID)
		}
		if buildWarning := checkSessionBuildProvenance(root, sha); buildWarning != "" {
			ctxText += "\n" + buildWarning
		}
		if mcpWarnings := checkSessionStaleMCPOnce(root, sessionID, sha); len(mcpWarnings) > 0 {
			ctxText += "\n" + strings.Join(mcpWarnings, "\n")
		}
		if st.Mode != state.ModeOff {
			ctxText += "\n\n" + LadderPrompt(st.Mode)
		}
		return buildOutputForAgent(flagAgent, "SessionStart", ctxText), nil

	case "before-prompt":
		_, _ = state.WriteWriterLease(root, flagAgent, sessionID)
		mcpWarnings := checkSessionStaleMCPOnce(root, sessionID, sha)
		// shortcut: Antigravity PreInvocation stdin omits prompt text, switch Ponytail mode via ateam state mode or /ponytail skill in agy
		promptText := firstNonEmpty(in.Prompt, in.UserPrompt)
		newMode, changed := detectModeSwitch(promptText)
		if !changed {
			if len(mcpWarnings) > 0 {
				return buildOutputForAgent(flagAgent, "UserPromptSubmit", strings.Join(mcpWarnings, "\n")), nil
			}
			return emptyOutputForAgent(flagAgent), nil
		}
		updated, err := state.Update(root, func(cur *state.State) error {
			cur.Mode = newMode
			return nil
		})
		if err != nil {
			return Output{}, err
		}
		var parts []string
		if len(mcpWarnings) > 0 {
			parts = append(parts, strings.Join(mcpWarnings, "\n"))
		}
		if updated.Mode != state.ModeOff {
			parts = append(parts, LadderPrompt(updated.Mode))
		}
		if len(parts) == 0 {
			return emptyOutputForAgent(flagAgent), nil
		}
		return buildOutputForAgent(flagAgent, "UserPromptSubmit", strings.Join(parts, "\n\n")), nil

	case "pre-compact":
		shortcuts, err := state.ScanShortcuts(root)
		if err != nil {
			return Output{}, err
		}
		holds, err := state.ScanHolds(root)
		if err != nil {
			return Output{}, err
		}
		heldBy := ""
		updated, err := state.Update(root, func(cur *state.State) error {
			cur.CurrentSessionID = sessionID
			if liveBr, liveSHA := state.ReadFastGitHead(root); liveBr != "" && liveSHA != "" {
				cur.SyncBranchHead(liveBr, liveSHA)
			} else if sha != "" {
				cur.SyncBranchHead(br, sha)
			}
			cur.CurrentAgent = state.ResolveActiveAgentForSession(root, flagAgent, sessionID, cur.Git.Branch, cur.Writer)
			cur.Shortcuts = shortcuts
			cur.Holds = holds
			cur.HoldScanError = ""
			curSlotKey := cur.CurrentHandoffKey()
			slotForCheck := ""
			curHandoff := cur.LastHandoff
			if cur.Handoffs != nil {
				curHandoff = cur.Handoffs[curSlotKey]
				slotForCheck = curSlotKey
			}
			sameSessionBranch := curHandoff != nil &&
				sessionID != "" &&
				curHandoff.SessionID == sessionID &&
				(curHandoff.Branch == "" || cur.Git.Branch == "" || curHandoff.Branch == cur.Git.Branch) &&
				!cur.HasNonHeadDriftHandoffStalenessFor(slotForCheck, curHandoff)
			if sameSessionBranch {
				curHandoff.Branch = cur.Git.Branch
				curHandoff.CheckpointSHA = cur.Git.HeadSHA
				curHandoff.Dirty = cur.Git.Dirty
			}
			cur.RefreshHandoffStaleness()
			if cur.Handoffs != nil {
				curHandoff = cur.Handoffs[curSlotKey]
			} else {
				curHandoff = cur.LastHandoff
			}
			foreignSession := curHandoff != nil && (curHandoff.SessionID == "" || sessionID == "" || curHandoff.SessionID != sessionID)
			if curHandoff.HoldsSlotAgainst(sessionID) {
				// #124: the slot is keyed agent@branch, so a fresh handoff here may be another
				// session's deliberate one; an auto-save must not replace it. Another auto-save
				// carries nothing worth keeping and is replaced as before.
				heldBy = curHandoff.SessionID
				if heldBy == "" {
					heldBy = "an unknown session"
				}
				return nil
			}
			if curHandoff == nil || curHandoff.Stale || foreignSession {
				cur.SetHandoff(state.Handoff{
					Goal:          state.AutoSaveGoal,
					Addresses:     fmt.Sprintf("branch %s, SHA %s", cur.Git.Branch, cur.Git.HeadSHA),
					Constraints:   fmt.Sprintf("%d active holds, %d shortcuts", len(cur.Holds), len(cur.Shortcuts)),
					DoneWhen:      "Run task verify and reconcile via /pickup",
					ChecksToRun:   "ateam state pickup && task verify",
					Agent:         cur.EffectiveAgent(),
					SessionID:     sessionID,
					Branch:        cur.Git.Branch,
					CheckpointSHA: cur.Git.HeadSHA,
					Dirty:         cur.Git.Dirty,
					AutoCompacted: true,
					UpdatedAt:     time.Now().UTC(),
				})
			} else {
				curHandoff.AutoCompacted = true
				curHandoff.UpdatedAt = time.Now().UTC()
				if cur.Handoffs != nil {
					cur.SetHandoff(*curHandoff)
				}
			}
			return nil
		})
		if err != nil {
			return Output{}, err
		}
		if heldBy != "" {
			return buildOutputForAgent(flagAgent, "PreCompact", fmt.Sprintf("No auto-handoff saved before compaction: handoff slot %s is held by %s and was not overwritten. Reconcile via /pickup.", updated.CurrentHandoffKey(), heldBy)), nil
		}
		return buildOutputForAgent(flagAgent, "PreCompact", "Auto-handoff checkpoint saved before compaction:\n"+state.FormatFiveFieldPrompt(*updated.LastHandoff)), nil

	case "turn-end":
		_, _ = state.WriteWriterLease(root, flagAgent, sessionID)
		holds, holdErr := state.ScanHolds(root)
		if _, err := state.Update(root, func(cur *state.State) error {
			if holdErr == nil {
				cur.Holds = holds
				cur.HoldScanError = ""
			} else {
				cur.HoldScanError = holdErr.Error()
			}
			return nil
		}); err != nil {
			return Output{}, err
		}
		if isAntigravityAgent(flagAgent) {
			return Output{Decision: "allow"}, nil
		}
		return Output{}, nil

	default:
		return Output{}, nil
	}
}

func sameHandoffValue(a, b *state.Handoff) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ca, cb := *a, *b
	ca.UpdatedAt = time.Time{}
	cb.UpdatedAt = time.Time{}
	return ca == cb && a.UpdatedAt.Equal(b.UpdatedAt)
}

func reconcileSessionStartHandoffs(root, flagAgent, br, sha string, wtHeadChanged bool, st *state.State) {
	beforeHandoffs := len(st.Handoffs)
	var prevLast *state.Handoff
	if st.LastHandoff != nil {
		cp := *st.LastHandoff
		prevLast = &cp
	}
	if st.Handoffs != nil {
		st.PruneStaleHandoffs()
	} else {
		st.RefreshHandoffStaleness()
	}
	lastChanged := !sameHandoffValue(prevLast, st.LastHandoff)
	if wtHeadChanged || (st.Handoffs != nil && (len(st.Handoffs) != beforeHandoffs || lastChanged)) {
		if updated, err := state.Update(root, func(cur *state.State) error {
			if liveBr, liveSHA := state.ReadFastGitHead(root); liveBr != "" && liveSHA != "" {
				cur.SyncBranchHead(liveBr, liveSHA)
			} else if sha != "" && !cur.UpdatedAt.After(st.UpdatedAt) {
				cur.SyncBranchHead(br, sha)
			}
			cur.CurrentSessionID = st.CurrentSessionID
			cur.CurrentAgent = state.ResolveActiveAgentForSession(root, flagAgent, cur.CurrentSessionID, cur.Git.Branch, cur.Writer)
			if cur.Handoffs != nil {
				cur.PruneStaleHandoffs()
			} else {
				cur.RefreshHandoffStaleness()
			}
			return nil
		}); err == nil && updated != nil {
			st.Git = updated.Git
			st.Handoffs = updated.Handoffs
			st.LastHandoff = updated.LastHandoff
			st.CurrentAgent = updated.CurrentAgent
		}
	}
}

func checkSessionBuildProvenance(root, headSHA string) string {
	shortBuild, _, _ := state.BuildRevision()
	return sessionBuildWarning(root, headSHA, shortBuild)
}

// sessionBuildWarning compares the running build with HEAD by string only. A module install's tag is
// labelled, not resolved: the hook spawns no subprocess (#150, #251); doctor resolves it.
func sessionBuildWarning(root, headSHA, shortBuild string) string {
	headSHA = strings.TrimSpace(headSHA)
	if headSHA == "" || headSHA == "uncommitted" {
		return ""
	}
	if !bootstrap.IsAteamCheckout(root) {
		return ""
	}
	if shortBuild != "" && shortBuild != "dev" && shortBuild != "unknown" && state.SameCommitSHA(shortBuild, headSHA) {
		return ""
	}
	return bootstrap.FormatStaleBinaryWarning(root, shortBuild, headSHA, bootstrap.RunningBinaryPath())
}

func checkSessionStaleMCPOnce(root, sessionID, headSHA string) []string {
	sessionID = strings.TrimSpace(sessionID)
	headSHA = strings.TrimSpace(headSHA)
	if sessionID == "" || headSHA == "" {
		return nil
	}
	// Only the ateam checkout's HEAD says anything about an MCP server's build (#151).
	if !bootstrap.IsAteamCheckout(root) {
		return nil
	}
	if state.HasMCPCheckedSession(root, sessionID) {
		return nil
	}
	_, warnings := inspectClientMCP(context.Background(), root, headSHA, state.ResolveClientPID())
	_ = state.RecordMCPCheckedSession(root, sessionID)
	return warnings
}

func isAntigravityAgent(agent string) bool {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "agy", "antigravity":
		return true
	default:
		return false
	}
}

func emptyOutputForAgent(agent string) Output {
	if isAntigravityAgent(agent) {
		return Output{EmitEmptyObject: true}
	}
	return Output{}
}

func buildOutputForAgent(agent, eventName, contextText string) Output {
	contextText = strings.TrimSpace(contextText)
	if contextText == "" {
		return emptyOutputForAgent(agent)
	}
	if isAntigravityAgent(agent) {
		return Output{
			InjectSteps: []InjectStep{
				{
					SystemMessage: &SystemMessageStep{
						SystemMessage: contextText,
					},
				},
			},
		}
	}
	return buildOutput(eventName, contextText)
}

func detectModeSwitch(prompt string) (state.DisciplineMode, bool) {
	lower := strings.ToLower(strings.TrimSpace(prompt))
	if lower == "stop ponytail" || lower == "normal mode" {
		return state.ModeOff, true
	}
	if m := modeCommandRe.FindStringSubmatch(prompt); len(m) == 2 {
		if mode, ok := state.NormalizeMode(m[1]); ok {
			return mode, true
		}
	}
	return state.ModeOn, false
}

// LadderPrompt returns the compact (~110 token) Ponytail decision ladder, or "" when the mode is off.
func LadderPrompt(mode state.DisciplineMode) string {
	if mode == state.ModeOff {
		return ""
	}
	return fmt.Sprintf("PONYTAIL MODE [%s] — Full discipline: smallest complete change; no unrequested features/wrappers.\n"+
		"Ladder: 1) Need to exist? 2) Already in codebase? 3) Stdlib/platform? 4) Existing dep? 5) Clear 1-liner? 6) Minimum complete change.\n"+
		"Guardrails: Never cut trust-boundary validation, error handling, security, or tests on branches/parsers. Tag limits with `// shortcut: <limit>, <when to upgrade>`. End conclusions and handoffs with 1-2 lines: skipped/unchecked + residual risk.",
		mode)
}

func buildOutput(eventName, contextText string) Output {
	contextText = strings.TrimSpace(contextText)
	if contextText == "" {
		return Output{}
	}
	return Output{
		AdditionalContext: contextText,
		HookSpecific: &SpecificOutput{
			HookEventName:     eventName,
			AdditionalContext: contextText,
		},
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
