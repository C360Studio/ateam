package bench

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const codexTestThread = `{"id":2,"result":{"thread":{"id":"thread-1","ephemeral":true},"model":"gpt-6-luna","modelProvider":"benchmark_openai","reasoningEffort":"low","approvalPolicy":"never","sandbox":{"type":"readOnly","networkAccess":false},"instructionSources":[]}}`
const codexTestFirstUsage = `{"total":{"inputTokens":100,"cachedInputTokens":80,"outputTokens":10,"reasoningOutputTokens":5,"totalTokens":110},"last":{"inputTokens":100,"cachedInputTokens":80,"outputTokens":10,"reasoningOutputTokens":5,"totalTokens":110}}`
const codexTestFinalUsage = `{"total":{"inputTokens":200,"cachedInputTokens":150,"cacheWriteInputTokens":0,"outputTokens":40,"reasoningOutputTokens":25,"totalTokens":240},"last":{"inputTokens":100,"cachedInputTokens":70,"outputTokens":30,"reasoningOutputTokens":20,"totalTokens":130}}`

func codexTestNotification(method, extra string) string {
	return `{"method":"` + method + `","params":{"threadId":"thread-1","turnId":"turn-1",` + extra + `}}`
}

func codexTestTranscript() []string {
	return []string{
		`{"id":1,"result":{"userAgent":"fixture-codex"}}`,
		`{"method":"remoteControl/status/changed","params":{"status":"disabled"}}`,
		codexTestThread,
		`{"method":"thread/started","params":{"thread":{"id":"thread-1"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1","status":"inProgress","items":[]}}}`,
		codexTestNotification("turn/started", `"turn":{"id":"turn-1","status":"inProgress","items":[]}`),
		codexTestNotification("item/started", `"item":{"id":"reason-1","type":"reasoning"}`),
		codexTestNotification("item/reasoning/summaryTextDelta", `"itemId":"reason-1","delta":"thinking"`),
		codexTestNotification("item/completed", `"item":{"id":"reason-1","type":"reasoning"}`),
		codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+codexTestFirstUsage),
		codexTestNotification("item/started", `"item":{"id":"message-1","type":"agentMessage","phase":"final_answer","text":""}`),
		codexTestNotification("item/agentMessage/delta", `"itemId":"message-1","delta":"ignored text delta"`),
		codexTestNotification("item/completed", `"item":{"id":"message-1","type":"agentMessage","phase":"final_answer","text":"{\"source\":\"package fixture\\n\"}"}`),
		codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+codexTestFinalUsage),
		codexTestNotification("turn/completed", `"turn":{"id":"turn-1","status":"completed","items":[]}`),
		`{"method":"thread/status/changed","params":{"threadId":"thread-1","status":{"type":"idle"}}}`,
	}
}

func TestCodexOutputNativeEvidenceAndAccounting(t *testing.T) {
	r, err := parseCodexOutput(strings.Join(codexTestTranscript(), "\n") + "\n")
	if err != nil || r.Source != "package fixture\n" || r.Usage == nil {
		t.Fatalf("valid native transcript rejected: %+v %v", r, err)
	}
	// Two requests used (100 input,10 output) and (100 input,30 output).
	// Snapshots overlap; adding 110 and 240, or adding subsets, is wrong.
	if r.Usage.InputTokens != 200 || r.Usage.OutputTokens != 40 || *r.Usage.CachedInputTokens != 150 || *r.Usage.ReasoningOutputTokens != 25 {
		t.Fatalf("cumulative usage was summed or subsets counted twice: %+v", r.Usage)
	}
	n := r.Native
	if n.SessionID != "thread-1" || n.Terminal != "completed" || n.UsageStatus != "reported" || n.EventCount != 16 || n.Codex.IdentitySource != "cli-thread-start" || n.Codex.TurnID != "turn-1" {
		t.Fatalf("native lifecycle evidence lost: %+v", n)
	}
	if n.ReportedModels != nil || *n.ResolvedModel != "gpt-6-luna" || *n.ResolvedReasoning != "low" || *n.Codex.ModelProvider != "benchmark_openai" {
		t.Fatalf("CLI resolved identity was lost or mislabeled as backend evidence: %+v", n)
	}
	if !reflect.DeepEqual(n.ObservedItemTypes, []string{"agentMessage", "reasoning"}) || !reflect.DeepEqual(n.CompletedItemTypes, n.ObservedItemTypes) || *n.Codex.Last.TotalTokens != 130 {
		t.Fatalf("native item or last-request evidence lost: %+v", n)
	}
}

func TestCodexThreadWarningDoesNotRequireActiveTurn(t *testing.T) {
	// Allowlisted warning envelope/message from the first Luna canary, with
	// thread ID and generated config path anonymized; timestamp is omitted.
	warning := `{"method":"warning","params":{"threadId":"thread-1","message":"Under-development features enabled: skip_host_skill_discovery. Under-development features are incomplete and may behave unpredictably. To suppress this warning, set \u0060suppress_unstable_features_warning = true\u0060 in /generated-workspace/.codex/config.toml."}}`
	for _, position := range []int{4, 13, 15} {
		t.Run(fmt.Sprintf("position-%d", position), func(t *testing.T) {
			lines := codexTestTranscript()
			withWarning := append([]string{}, lines[:position]...)
			withWarning = append(withWarning, warning)
			withWarning = append(withWarning, lines[position:]...)
			r, err := parseCodexOutput(strings.Join(withWarning, "\n"))
			if err != nil || r.Source != "package fixture\n" || r.Usage == nil || r.Usage.InputTokens != 200 || len(r.Native.Warnings) != 0 {
				t.Fatalf("native thread warning rejected or changed accounting: %+v %v", r, err)
			}
		})
	}
	for name, warning := range map[string]string{
		"foreign thread":     `{"method":"warning","params":{"threadId":"other-thread","message":"warning"}}`,
		"missing thread":     `{"method":"warning","params":{"message":"warning"}}`,
		"null thread":        `{"method":"warning","params":{"threadId":null,"message":"warning"}}`,
		"missing message":    `{"method":"warning","params":{"threadId":"thread-1"}}`,
		"null message":       `{"method":"warning","params":{"threadId":"thread-1","message":null}}`,
		"wrong message type": `{"method":"warning","params":{"threadId":"thread-1","message":7}}`,
		"empty message":      `{"method":"warning","params":{"threadId":"thread-1","message":" "}}`,
		"foreign turn":       `{"method":"warning","params":{"threadId":"thread-1","turnId":"other-turn","message":"warning"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			lines := codexTestTranscript()
			lines = append(append(append([]string{}, lines[:13]...), warning), lines[13:]...)
			if r, err := parseCodexOutput(strings.Join(lines, "\n")); err == nil || r.Source != "" {
				t.Fatalf("unscoped or malformed warning accepted: %+v %v", r, err)
			}
		})
	}
	t.Run("incomplete warning retains archived evidence", func(t *testing.T) {
		lines := codexTestTranscript()[:4]
		before, err := parseCodexOutput(strings.Join(lines, "\n"))
		if err == nil {
			t.Fatal("truncated baseline unexpectedly complete")
		}
		before.Native.EventCount++
		after, err := parseCodexOutput(strings.Join(append(lines, warning), "\n"))
		if err == nil || !reflect.DeepEqual(before, after) {
			t.Fatalf("warning changed archived incomplete response evidence: before=%+v after=%+v err=%v", before, after, err)
		}
	})
	t.Run("warning cannot establish thread identity", func(t *testing.T) {
		lines := codexTestTranscript()
		lines = append(append(append([]string{}, lines[:2]...), warning), lines[2:]...)
		if r, err := parseCodexOutput(strings.Join(lines, "\n")); err == nil || r.Source != "" || r.Native.Codex.ThreadID != "" {
			t.Fatalf("warning established unvalidated thread identity: %+v %v", r, err)
		}
	})
}

func TestCodexAccountRateLimitsDiagnostic(t *testing.T) {
	// Allowlisted native event from the second Luna canary; timestamp omitted.
	captured := `{"method":"account/rateLimits/updated","params":{"rateLimits":{"limitId":"codex","limitName":null,"normalModelSlug":null,"primary":null,"secondary":null,"credits":null,"individualLimit":null,"spendControlReached":null,"planType":null,"rateLimitReachedType":null}}}`
	full := `{"method":"account/rateLimits/updated","params":{"rateLimits":{"limitId":"codex","limitName":"fixture","normalModelSlug":"not-model-proof","primary":{"usedPercent":100,"windowDurationMins":60,"resetsAt":123},"secondary":{"usedPercent":0},"credits":{"balance":"1.25","hasCredits":true,"unlimited":false},"individualLimit":{"limit":"2","remainingPercent":50,"resetsAt":123,"used":"1"},"spendControlReached":false,"planType":"unknown","rateLimitReachedType":"rate_limit_reached"}}}`
	baseline, err := parseCodexOutput(strings.Join(codexTestTranscript(), "\n"))
	if err != nil {
		t.Fatal(err)
	}
	baseline.Native.EventCount++
	for _, event := range []string{captured, full, `{"method":"account/rateLimits/updated","params":{"rateLimits":{}}}`} {
		for _, position := range []int{2, 4, 13, 15, 16} {
			lines := codexTestTranscript()
			lines = append(append(append([]string{}, lines[:position]...), event), lines[position:]...)
			got, err := parseCodexOutput(strings.Join(lines, "\n"))
			if err != nil || !reflect.DeepEqual(got, baseline) {
				t.Fatalf("account diagnostic changed native evidence at %d: %+v %v", position, got, err)
			}
		}
	}
	// The failed canary stopped after usage and the diagnostic, before terminal
	// completion. Its persisted incomplete response must remain replayable.
	prefix := codexTestTranscript()[:14]
	before, err := parseCodexOutput(strings.Join(prefix, "\n"))
	if err == nil {
		t.Fatal("truncated baseline unexpectedly complete")
	}
	before.Native.EventCount++
	after, err := parseCodexOutput(strings.Join(append(prefix, captured), "\n"))
	if err == nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("account diagnostic changed archived incomplete evidence: %+v %v", after, err)
	}
	for name, params := range map[string]string{
		"missing snapshot": `{}`, "null snapshot": `{"rateLimits":null}`, "array snapshot": `{"rateLimits":[]}`,
		"string snapshot": `{"rateLimits":"unknown"}`, "number snapshot": `{"rateLimits":0}`, "boolean snapshot": `{"rateLimits":false}`,
		"duplicate nested key": `{"rateLimits":{"primary":null,"primary":{}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			lines := append(codexTestTranscript(), `{"method":"account/rateLimits/updated","params":`+params+`}`)
			if got, err := parseCodexOutput(strings.Join(lines, "\n")); err == nil || got.Source != "" {
				t.Fatalf("malformed account diagnostic accepted: %+v %v", got, err)
			}
		})
	}
	// Optional nested quota metadata is opaque: it never influences the run.
	// These examples cover the complete outer JSON type boundary; a generated
	// nested schema matrix would test fields deliberately excluded from evidence.
	opaque := `{"method":"account/rateLimits/updated","params":{"rateLimits":{"newField":{"model":"untrusted-model","usage":{"inputTokens":999}},"credits":"future-metadata"}}}`
	got, err := parseCodexOutput(strings.Join(append(codexTestTranscript(), opaque), "\n"))
	if err != nil || !reflect.DeepEqual(got, baseline) {
		t.Fatalf("opaque quota metadata affected benchmark evidence: %+v %v", got, err)
	}
	h := newCodexHistory()
	if err := h.consume([]byte(captured)); err == nil {
		t.Fatal("account diagnostic accepted before initialization")
	}
	h = newCodexHistory()
	if err := h.consume([]byte(codexTestTranscript()[0])); err != nil {
		t.Fatal(err)
	}
	if err := h.consume([]byte(captured)); err != nil || h.response.Native.Codex.ThreadID != "" || h.response.Native.ResolvedModel != nil || h.response.Usage != nil {
		t.Fatalf("account diagnostic established benchmark identity or usage: %+v %v", h.response, err)
	}
}

func TestCodexThreadStartValidatedBeforeInference(t *testing.T) {
	id, err := validateCodexThreadStart([]byte(codexTestThread), "gpt-6-luna", "low")
	if err != nil || id != "thread-1" {
		t.Fatalf("native thread configuration rejected: %s %v", id, err)
	}
	for name, data := range map[string]string{
		"wrong model":          strings.Replace(codexTestThread, "gpt-6-luna", "other-model", 1),
		"missing model":        strings.Replace(codexTestThread, `"model":"gpt-6-luna",`, "", 1),
		"wrong provider":       strings.Replace(codexTestThread, "benchmark_openai", "openai", 1),
		"wrong reasoning":      strings.Replace(codexTestThread, `"low"`, `"high"`, 1),
		"null reasoning":       strings.Replace(codexTestThread, `"low"`, `null`, 1),
		"persistent thread":    strings.Replace(codexTestThread, `"ephemeral":true`, `"ephemeral":false`, 1),
		"missing ephemeral":    strings.Replace(codexTestThread, `,"ephemeral":true`, "", 1),
		"missing thread":       strings.Replace(codexTestThread, `"thread-1"`, `""`, 1),
		"approval enabled":     strings.Replace(codexTestThread, `"never"`, `"on-request"`, 1),
		"writable sandbox":     strings.Replace(codexTestThread, `"readOnly"`, `"workspaceWrite"`, 1),
		"network enabled":      strings.Replace(codexTestThread, `"networkAccess":false`, `"networkAccess":true`, 1),
		"unknown network":      strings.Replace(codexTestThread, `,"networkAccess":false`, "", 1),
		"ambient instructions": strings.Replace(codexTestThread, `"instructionSources":[]`, `"instructionSources":["AGENTS.md"]`, 1),
		"unknown instructions": strings.Replace(codexTestThread, `,"instructionSources":[]`, "", 1),
		"wrong id":             strings.Replace(codexTestThread, `"id":2`, `"id":1`, 1),
		"duplicate key":        strings.Replace(codexTestThread, `"id":2`, `"id":1,"id":2`, 1),
		"RPC error":            strings.Replace(codexTestThread, `"id":2`, `"id":2,"error":{"code":-1}`, 1),
		"native request":       strings.Replace(codexTestThread, `"id":2`, `"id":2,"method":"approval"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if id, err := validateCodexThreadStart([]byte(data), "gpt-6-luna", "low"); err == nil || id != "" {
				t.Fatalf("unsafe or unknown native configuration permitted inference: %q %v", id, err)
			}
		})
	}
	lines := codexTestTranscript()
	lines[2] = strings.Replace(codexTestThread, `"model":"gpt-6-luna",`, "", 1)
	r, err := parseCodexOutput(strings.Join(lines, "\n"))
	if err != nil || r.Native.ResolvedModel != nil || r.Native.Codex.Model != nil || r.Native.ReportedModels != nil {
		t.Fatalf("missing native identity was invented: %+v %v", r.Native, err)
	}
}

func TestCodexOutputRejectsInvalidHistories(t *testing.T) {
	for name, mutate := range map[string]func([]string) []string{
		"missing initialize":           func(lines []string) []string { return lines[1:] },
		"missing thread start":         func(lines []string) []string { return append(lines[:3], lines[4:]...) },
		"missing turn acknowledgement": func(lines []string) []string { return append(lines[:4], lines[5:]...) },
		"missing item start":           func(lines []string) []string { return append(lines[:10], lines[11:]...) },
		"duplicate initialize":         func(lines []string) []string { return append([]string{lines[0]}, lines...) },
		"duplicate turn completion":    func(lines []string) []string { return append(lines, lines[14]) },
		"truncated before completion":  func(lines []string) []string { return lines[:14] },
		"request after completion": func(lines []string) []string {
			return append(lines, `{"id":22,"method":"item/commandExecution/requestApproval","params":{}}`)
		},
		"foreign thread": func(lines []string) []string {
			lines[13] = strings.Replace(lines[13], "thread-1", "thread-2", 1)
			return lines
		},
		"foreign turn": func(lines []string) []string {
			lines[13] = strings.Replace(lines[13], "turn-1", "turn-2", 1)
			return lines
		},
		"failed turn": func(lines []string) []string {
			lines[14] = strings.Replace(lines[14], `"completed"`, `"failed"`, 1)
			return lines
		},
		"interrupted turn": func(lines []string) []string {
			lines[14] = strings.Replace(lines[14], `"completed"`, `"interrupted"`, 1)
			return lines
		},
		"completion error": func(lines []string) []string {
			lines[14] = strings.Replace(lines[14], `"status":"completed"`, `"status":"completed","error":{"message":"failed"}`, 1)
			return lines
		},
		"rerouted": func(lines []string) []string {
			lines[13] = codexTestNotification("model/rerouted", `"fromModel":"gpt-6-luna","toModel":"other","reason":"highRiskCyberActivity"`)
			return lines
		},
		"retry error": func(lines []string) []string {
			lines[13] = codexTestNotification("error", `"willRetry":true,"error":{"message":"retrying"}`)
			return lines
		},
		"tool item": func(lines []string) []string {
			lines[10] = strings.Replace(lines[10], "agentMessage", "commandExecution", 1)
			return lines
		},
		"auxiliary agent": func(lines []string) []string {
			lines[10] = strings.Replace(lines[10], "agentMessage", "collabToolCall", 1)
			return lines
		},
		"remote control enabled": func(lines []string) []string {
			lines[1] = strings.Replace(lines[1], "disabled", "enabled", 1)
			return lines
		},
		"premature thread close": func(lines []string) []string {
			lines[13] = codexTestNotification("thread/closed", `"extra":null`)
			return lines
		},
		"unknown notification": func(lines []string) []string {
			lines[13] = codexTestNotification("future/toolExecution", `"extra":null`)
			return lines
		},
		"duplicate JSON key": func(lines []string) []string {
			lines[13] = strings.Replace(lines[13], `"totalTokens":240`, `"totalTokens":240,"totalTokens":241`, 1)
			return lines
		},
		"malformed JSON": func(lines []string) []string { return append(lines, `{`) },
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseCodexOutput(strings.Join(mutate(codexTestTranscript()), "\n"))
			if err == nil || r.Source != "" || r.Usage != nil {
				t.Fatalf("invalid history yielded a graded or accountable success: %+v %v", r, err)
			}
		})
	}
}

func TestCodexTokenPresenceAndBoundaries(t *testing.T) {
	fields := []string{"inputTokens", "cachedInputTokens", "outputTokens", "reasoningOutputTokens", "totalTokens"}
	checked := 0
	for mask := 0; mask < 32; mask++ {
		for _, absent := range []string{"missing", "null"} {
			t.Run(fmt.Sprintf("%s/%05b", absent, mask), func(t *testing.T) {
				total := map[string]any{}
				for i, field := range fields {
					if mask&(1<<i) != 0 {
						total[field] = 0
					} else if absent == "null" {
						total[field] = nil
					}
				}
				data, _ := json.Marshal(map[string]any{"total": total, "last": total})
				lines := codexTestTranscript()
				lines[9] = codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+string(data))
				lines[13] = lines[9]
				r, err := parseCodexOutput(strings.Join(lines, "\n"))
				if err != nil || (r.Usage != nil) != (mask == 31) || r.Native.Codex.Total.CacheWriteInputTokens != nil {
					t.Fatalf("missing counters invented or complete zero counts lost: mask=%05b %+v %v", mask, r, err)
				}
				checked++
			})
		}
	}
	if checked != 64 {
		t.Fatalf("presence assertions not executed: %d", checked)
	}
	for name, change := range map[string]string{
		"negative": `"inputTokens":-1`, "over bound": `"inputTokens":1000000000001`,
		"fraction": `"inputTokens":1.5`, "overflow": `"inputTokens":9223372036854775808`,
		"total mismatch": `"inputTokens":201`, "decreased cumulative": `"inputTokens":99`,
	} {
		t.Run(name, func(t *testing.T) {
			lines := codexTestTranscript()
			lines[13] = strings.Replace(lines[13], `"inputTokens":200`, change, 1)
			if r, err := parseCodexOutput(strings.Join(lines, "\n")); err == nil || r.Usage != nil || r.Source != "" {
				t.Fatalf("invalid counters accepted: %+v %v", r, err)
			}
		})
	}
	for name, usage := range map[string]string{
		"cache exceeds input":       strings.Replace(codexTestFinalUsage, `"cachedInputTokens":150`, `"cachedInputTokens":201`, 1),
		"cache write exceeds input": strings.Replace(codexTestFinalUsage, `"cacheWriteInputTokens":0`, `"cacheWriteInputTokens":201`, 1),
		"reasoning exceeds output":  strings.Replace(codexTestFinalUsage, `"reasoningOutputTokens":25`, `"reasoningOutputTokens":41`, 1),
		"last exceeds total":        `{"total":{"inputTokens":200,"cachedInputTokens":150,"outputTokens":40,"reasoningOutputTokens":25,"totalTokens":240},"last":{"inputTokens":200,"cachedInputTokens":151,"outputTokens":30,"reasoningOutputTokens":20,"totalTokens":230}}`,
		"cumulative decreases":      strings.Replace(codexTestFinalUsage, `"cachedInputTokens":150`, `"cachedInputTokens":79`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			lines := codexTestTranscript()
			lines[13] = codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+usage)
			if r, err := parseCodexOutput(strings.Join(lines, "\n")); err == nil || r.Usage != nil {
				t.Fatalf("inconsistent subset/history accepted: %+v %v", r, err)
			}
		})
	}
}

func TestCodexExactTokenBoundary(t *testing.T) {
	usage := `{"total":{"inputTokens":1000000000000,"cachedInputTokens":0,"outputTokens":0,"reasoningOutputTokens":0,"totalTokens":1000000000000},"last":{"inputTokens":0,"cachedInputTokens":0,"outputTokens":0,"reasoningOutputTokens":0,"totalTokens":0}}`
	lines := codexTestTranscript()
	lines[9] = codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+usage)
	lines[13] = lines[9]
	r, err := parseCodexOutput(strings.Join(lines, "\n"))
	if err != nil || r.Usage == nil || r.Usage.InputTokens != 1000000000000 {
		t.Fatalf("exact native count boundary rejected: %+v %v", r, err)
	}
}

func TestCodexCumulativeUsageCannotDisappear(t *testing.T) {
	first := `{"total":{"inputTokens":500,"cachedInputTokens":300,"cacheWriteInputTokens":0,"outputTokens":50,"reasoningOutputTokens":20,"totalTokens":550},"last":null}`
	last := `{"total":{"inputTokens":100,"cachedInputTokens":50,"outputTokens":10,"reasoningOutputTokens":5,"totalTokens":110},"last":null}`
	for name, middle := range map[string]string{
		"null snapshot":  `null`,
		"empty snapshot": `{}`,
		"null counters":  `{"inputTokens":null,"cachedInputTokens":null,"outputTokens":null,"reasoningOutputTokens":null,"totalTokens":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			lines := codexTestTranscript()
			lines[9] = codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+first)
			lines[13] = codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+last)
			withMiddle := append([]string{}, lines[:10]...)
			withMiddle = append(withMiddle, codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":{"total":`+middle+`,"last":null}`))
			withMiddle = append(withMiddle, lines[10:]...)
			r, err := parseCodexOutput(strings.Join(withMiddle, "\n"))
			if err == nil || r.Source != "" || r.Usage != nil || r.Native.UsageStatus != "invalid" {
				t.Fatalf("missing snapshot erased cumulative accounting history: %+v %v", r, err)
			}
		})
	}
	// Losing even one previously reported category must fail at that update,
	// before a later lower value can be mistaken for a new accounting baseline.
	for _, field := range []string{"inputTokens", "cachedInputTokens", "cacheWriteInputTokens", "outputTokens", "reasoningOutputTokens", "totalTokens"} {
		t.Run("missing "+field, func(t *testing.T) {
			var total map[string]any
			if err := json.Unmarshal([]byte(`{"inputTokens":500,"cachedInputTokens":300,"cacheWriteInputTokens":0,"outputTokens":50,"reasoningOutputTokens":20,"totalTokens":550}`), &total); err != nil {
				t.Fatal(err)
			}
			delete(total, field)
			middle, _ := json.Marshal(map[string]any{"total": total, "last": nil})
			lines := codexTestTranscript()
			lines[9] = codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+first)
			lines[13] = codexTestNotification("thread/tokenUsage/updated", `"tokenUsage":`+string(middle))
			r, err := parseCodexOutput(strings.Join(lines, "\n"))
			if err == nil || r.Usage != nil || r.Native.UsageStatus != "invalid" {
				t.Fatalf("previously known %s silently disappeared: %+v %v", field, r, err)
			}
		})
	}
}

func TestCodexFinalSourceAndUnknownUsage(t *testing.T) {
	for _, source := range []string{`{}`, `null`, `{"source":""}`, `{"Source":"package x"}`, `{"source":"x","extra":true}`, `{"source":"x","source":"y"}`, "```json\n{}\n```"} {
		encoded, _ := json.Marshal(source)
		lines := codexTestTranscript()
		lines[12] = codexTestNotification("item/completed", `"item":{"id":"message-1","type":"agentMessage","phase":"final_answer","text":`+string(encoded)+`}`)
		r, err := parseCodexOutput(strings.Join(lines, "\n"))
		if err == nil || r.Source != "" || r.Usage == nil {
			t.Fatalf("bad final source accepted or usage lost: %+v %v", r, err)
		}
	}
	lines := codexTestTranscript()
	lines = append(lines[:13], lines[14:]...)
	lines = append(lines[:9], lines[10:]...)
	r, err := parseCodexOutput(strings.Join(lines, "\n"))
	if err != nil || r.Usage != nil || r.Native.UsageStatus != "missing" || r.Source != "package fixture\n" {
		t.Fatalf("absent usage invented: %+v %v", r, err)
	}
	for _, text := range []string{"", " ", "null", "[]", strings.Repeat(" ", maxAdapterBytes+1)} {
		if r, err := parseCodexOutput(text); err == nil || r.Source != "" || r.Usage != nil {
			t.Fatalf("empty/oversized output accepted: %+v %v", r, err)
		}
	}
}

// Generated histories use a test-owned state oracle: every item ID has one start
// and one completion, and final turn completion occurs only when all are closed.
func TestCodexGeneratedItemHistories(t *testing.T) {
	checked := 0
	for count := 1; count <= 5; count++ {
		for reverse := 0; reverse < 2; reverse++ {
			lines := codexTestTranscript()
			prefix := append([]string{}, lines[:6]...)
			open := map[string]bool{}
			for i := 0; i < count; i++ {
				id := fmt.Sprintf("reason-generated-%d", i)
				open[id] = true
				prefix = append(prefix, codexTestNotification("item/started", `"item":{"id":"`+id+`","type":"reasoning"}`))
			}
			for step := 0; step < count; step++ {
				i := step
				if reverse == 1 {
					i = count - 1 - step
				}
				id := fmt.Sprintf("reason-generated-%d", i)
				delete(open, id)
				prefix = append(prefix, codexTestNotification("item/completed", `"item":{"id":"`+id+`","type":"reasoning"}`))
			}
			if len(open) != 0 {
				t.Fatal("reference history not completed")
			}
			valid := append(append([]string{}, prefix...), lines[9:]...)
			if r, err := parseCodexOutput(strings.Join(valid, "\n")); err != nil || r.Source == "" {
				t.Fatalf("valid independently modeled history rejected: %+v %v", r, err)
			}
			// Duplicate one completion: the oracle has no matching open item.
			invalid := append(append([]string{}, prefix...), prefix[len(prefix)-1])
			invalid = append(invalid, lines[9:]...)
			if r, err := parseCodexOutput(strings.Join(invalid, "\n")); err == nil || r.Source != "" {
				t.Fatalf("closed item completed twice: %+v %v", r, err)
			}
			checked++
		}
	}
	if checked != 10 {
		t.Fatalf("generated histories did not reach assertions: %d", checked)
	}
}
