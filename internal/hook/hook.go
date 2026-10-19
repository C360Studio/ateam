// Package hook normalizes Claude Code, Codex CLI, and Gemini CLI lifecycle hook events in under 5ms.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// Input represents the normalized JSON payload passed on stdin by Claude, Codex, or Gemini hooks.
type Input struct {
	HookEventName string `json:"hook_event_name,omitempty"`
	Event         string `json:"event,omitempty"`
	Prompt        string `json:"prompt,omitempty"`
	UserPrompt    string `json:"user_prompt,omitempty"`
	Cwd           string `json:"cwd,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
}

// Output represents the JSON response returned on stdout to Claude, Codex, or Gemini hooks.
type Output struct {
	AdditionalContext string          `json:"additionalContext,omitempty"`
	Decision          string          `json:"decision,omitempty"`
	Reason            string          `json:"reason,omitempty"`
	HookSpecific      *SpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// SpecificOutput provides compatibility with Claude/Gemini hookSpecificOutput wrappers.
type SpecificOutput struct {
	HookEventName     string `json:"hookEventName,omitempty"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

var modeCommandRe = regexp.MustCompile(`(?i)(?:^|\s)/(?:ponytail|mode)\s+(lite|full|ultra|off)\b`)

// NormalizeEvent maps CLI-specific hook names (Claude, Codex, Gemini) to canonical event names.
func NormalizeEvent(flagEvent, payloadEvent string) string {
	raw := strings.TrimSpace(flagEvent)
	if raw == "" {
		raw = strings.TrimSpace(payloadEvent)
	}
	switch strings.ToLower(raw) {
	case "session-start", "sessionstart":
		return "session-start"
	case "before-prompt", "userpromptsubmit", "beforeagent":
		return "before-prompt"
	case "pre-compact", "precompact", "precompress":
		return "pre-compact"
	case "turn-end", "stop", "afteragent", "sessionend":
		return "turn-end"
	default:
		return ""
	}
}

// Handle processes a hook event against root without external subprocesses so latency stays <5ms.
func Handle(root, flagEvent string, stdin io.Reader) (Output, error) {
	return HandleWithAgent(root, flagEvent, "", stdin)
}

// HandleWithAgent processes a hook event with an optional explicit agent identity (--agent claude|codex|gemini).
func HandleWithAgent(root, flagEvent, flagAgent string, stdin io.Reader) (Output, error) {
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
	st.CurrentSessionID = strings.TrimSpace(in.SessionID)
	br, sha := state.ReadFastGitHead(root)
	if sha != "" {
		st.Git.Branch = br
		st.Git.HeadSHA = sha
	}

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
		ctxText := state.BuildPickupSummary(st)
		if st.Mode != state.ModeOff {
			ctxText += "\n\n" + LadderPrompt(st.Mode)
		}
		return buildOutput("SessionStart", ctxText), nil

	case "before-prompt":
		_, _ = state.WriteWriterLease(root, flagAgent, in.SessionID)
		promptText := firstNonEmpty(in.Prompt, in.UserPrompt)
		newMode, changed := detectModeSwitch(promptText)
		if !changed {
			return Output{}, nil
		}
		updated, err := state.Update(root, func(cur *state.State) error {
			cur.Mode = newMode
			return nil
		})
		if err != nil {
			return Output{}, err
		}
		if updated.Mode == state.ModeOff {
			return Output{}, nil
		}
		return buildOutput("UserPromptSubmit", LadderPrompt(updated.Mode)), nil

	case "pre-compact":
		shortcuts, err := state.ScanShortcuts(root)
		if err != nil {
			return Output{}, err
		}
		holds, err := state.ScanHolds(root)
		if err != nil {
			return Output{}, err
		}
		updated, err := state.Update(root, func(cur *state.State) error {
			if sha != "" {
				cur.Git.Branch = br
				cur.Git.HeadSHA = sha
			}
			cur.Shortcuts = shortcuts
			cur.Holds = holds
			cur.HoldScanError = ""
			if cur.LastHandoff == nil {
				cur.LastHandoff = &state.Handoff{
					Goal:          "Auto-saved before context compaction",
					Addresses:     fmt.Sprintf("branch %s, SHA %s", cur.Git.Branch, cur.Git.HeadSHA),
					Constraints:   fmt.Sprintf("%d active holds, %d shortcuts", len(cur.Holds), len(cur.Shortcuts)),
					DoneWhen:      "Run task verify and reconcile via /pickup",
					ChecksToRun:   "ateam state pickup && task verify",
					Branch:        cur.Git.Branch,
					CheckpointSHA: cur.Git.HeadSHA,
					Dirty:         cur.Git.Dirty,
					AutoCompacted: true,
					UpdatedAt:     time.Now().UTC(),
				}
			} else {
				cur.LastHandoff.AutoCompacted = true
				cur.LastHandoff.UpdatedAt = time.Now().UTC()
			}
			return nil
		})
		if err != nil {
			return Output{}, err
		}
		return buildOutput("PreCompact", "Auto-handoff checkpoint saved before compaction:\n"+state.FormatFiveFieldPrompt(*updated.LastHandoff)), nil

	case "turn-end":
		_, _ = state.WriteWriterLease(root, flagAgent, in.SessionID)
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
		return Output{}, nil

	default:
		return Output{}, nil
	}
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
	return state.ModeFull, false
}

// LadderPrompt returns the compact (~110 token) Ponytail decision ladder for the active level.
func LadderPrompt(mode state.DisciplineMode) string {
	if mode == state.ModeOff {
		return ""
	}
	levelLine := "Full discipline: smallest complete change; no unrequested features/wrappers."
	switch mode {
	case state.ModeLite:
		levelLine = "Lite discipline: build what was asked; name the smaller option in one line."
	case state.ModeUltra:
		levelLine = "Ultra discipline: question any requirement not justified by the need before building."
	}
	return fmt.Sprintf("PONYTAIL MODE [%s] — %s\n"+
		"Ladder: 1) Need to exist? 2) Already in codebase? 3) Stdlib/platform? 4) Existing dep? 5) Clear 1-liner? 6) Minimum complete change.\n"+
		"Guardrails: Never cut trust-boundary validation, error handling, security, or tests on branches/parsers. Tag limits with `// shortcut: <limit>, <when to upgrade>`. End reply with 1-2 lines: skipped/unchecked + residual risk.",
		mode, levelLine)
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
