package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Native totals must retain provider counts; adding cached/reasoning subsets twice is wrong.
func TestCodexRecordedEvents(t *testing.T) {
	for _, tc := range []struct {
		name, terminal, usage string
		wantError             bool
		input, output         int64
	}{
		{"success", "completed", "reported", false, 100, 25},
		{"missing", "completed", "missing", false, 0, 0},
		{"partial", "completed", "partial", false, 0, 0},
		{"failure", "failed", "reported", true, 41, 7},
		{"interrupted", "incomplete", "missing", true, 0, 0},
		{"malformed", "incomplete", "invalid", true, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "codex", tc.name+".jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			r, err := parseCodexEvents(string(data))
			if (err != nil) != tc.wantError || r.Native.Terminal != tc.terminal || r.Native.UsageStatus != tc.usage {
				t.Fatalf("response=%+v native=%+v error=%v", r, r.Native, err)
			}
			if tc.usage == "reported" {
				if r.Usage == nil || r.Usage.InputTokens != tc.input || r.Usage.OutputTokens != tc.output {
					t.Fatalf("native primary counts changed: %+v", r.Usage)
				}
			} else if r.Usage != nil {
				t.Fatalf("invented complete usage: %+v", r.Usage)
			}
			if r.Native.ResolvedModel != nil {
				t.Fatal("invented resolved model")
			}
			if tc.name == "success" {
				if r.Source != "package safepath\n" || *r.Usage.CachedInputTokens != 80 || *r.Usage.ReasoningOutputTokens != 15 || *r.Native.ReportedUsage.CacheWriteInputTokens != 2 {
					t.Fatalf("lost source or usage detail: %+v", r)
				}
				if !reflect.DeepEqual(r.Native.ObservedItemTypes, []string{"agent_message", "command_execution", "reasoning"}) || !reflect.DeepEqual(r.Native.CompletedItemTypes, []string{"agent_message", "command_execution"}) {
					t.Fatalf("tool evidence was inferred or dropped: %+v", r.Native)
				}
			}
		})
	}
}

func codexTestEvents(usage string) string {
	return "{\"type\":\"thread.started\",\"thread_id\":\"test-session\"}\n" +
		"{\"type\":\"turn.started\"}\n" +
		`{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"{\"source\":\"package safepath\"}"}}` + "\n" +
		`{"type":"turn.completed","usage":` + usage + "}\n"
}

// One turn has one terminal usage snapshot; accepting replayed terminals would inflate totals.
func TestCodexRefusesAmbiguousEvents(t *testing.T) {
	good := codexTestEvents(`{"input_tokens":10,"output_tokens":5}`)
	for name, stream := range map[string]string{
		"duplicate terminal": good + `{"type":"turn.completed","usage":{"input_tokens":20,"output_tokens":10}}`,
		"trailing malformed": good + `{`,
		"terminal first":     `{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}`,
		"duplicate type":     strings.Replace(good, `"type":"turn.started"`, `"type":"turn.started","type":"turn.started"`, 1),
		"duplicate count":    codexTestEvents(`{"input_tokens":1,"input_tokens":2,"output_tokens":1}`),
		"unknown usage":      codexTestEvents(`{"input_tokens":1,"output_tokens":1,"billing_tokens":2}`),
		"fractional usage":   codexTestEvents(`{"input_tokens":1.5,"output_tokens":1}`),
		"negative usage":     codexTestEvents(`{"input_tokens":-1,"output_tokens":1}`),
		"cache exceeds":      codexTestEvents(`{"input_tokens":1,"output_tokens":1,"cached_input_tokens":2}`),
		"reasoning exceeds":  codexTestEvents(`{"input_tokens":1,"output_tokens":1,"reasoning_output_tokens":2}`),
		"unknown event":      strings.Replace(good, `"turn.started"`, `"turn.mystery"`, 1),
		"repeated turn":      strings.Replace(good, `{"type":"turn.started"}`, "{\"type\":\"turn.started\"}\n{\"type\":\"turn.started\"}", 1),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseCodexEvents(stream)
			if err == nil || r.Usage != nil {
				t.Fatalf("ambiguous stream supplied accountable usage: %+v error=%v", r, err)
			}
		})
	}
	for _, message := range []string{`{}`, `{"source":""}`, `{"source":"one","source":"two"}`, `{"source":"x","extra":true}`, `null`} {
		encoded, _ := json.Marshal(message)
		stream := strings.Replace(good, `"{\"source\":\"package safepath\"}"`, string(encoded), 1)
		r, err := parseCodexEvents(stream)
		if err == nil || r.Usage == nil {
			t.Fatalf("invalid final source must fail without losing valid usage: %+v error=%v", r, err)
		}
	}
}

// The reference history is thread -> turn -> answer -> terminal; all other permutations fail.
func TestCodexEventHistoryPermutations(t *testing.T) {
	lines := strings.Split(strings.TrimSpace(codexTestEvents(`null`)), "\n")
	checked := 0
	for a := 0; a < 4; a++ {
		for b := 0; b < 4; b++ {
			for c := 0; c < 4; c++ {
				for d := 0; d < 4; d++ {
					if a == b || a == c || a == d || b == c || b == d || c == d {
						continue
					}
					checked++
					_, err := parseCodexEvents(strings.Join([]string{lines[a], lines[b], lines[c], lines[d]}, "\n"))
					wantSuccess := a == 0 && b == 1 && c == 2 && d == 3
					if (err == nil) != wantSuccess {
						t.Fatalf("history %d%d%d%d: error=%v", a, b, c, d, err)
					}
				}
			}
		}
	}
	if checked != 24 {
		t.Fatalf("history assertions did not activate: %d", checked)
	}
	t.Logf("checked all %d event orderings", checked)
}

func TestCodexEventSizeBoundary(t *testing.T) {
	valid := codexTestEvents(`null`)
	exact := valid + strings.Repeat(" ", maxAdapterBytes-len(valid))
	if _, err := parseCodexEvents(exact); err != nil {
		t.Fatalf("exact 1 MiB boundary rejected: %v", err)
	}
	if _, err := parseCodexEvents(exact + " "); err == nil {
		t.Fatal("accepted output over 1 MiB")
	}
	if err := uniqueJSON([]byte(strings.Repeat("[", 66) + "0" + strings.Repeat("]", 66))); err == nil {
		t.Fatal("accepted excessive nesting")
	}
}

// Generated presence/count combinations check the requirement directly, including both limits.
func FuzzCodexNativeUsage(f *testing.F) {
	f.Add(int64(100), int64(25), int64(80), int64(15), uint8(15))
	f.Add(int64(1e12), int64(1e12), int64(1e12), int64(1e12), uint8(15))
	f.Add(int64(1e12+1), int64(0), int64(0), int64(0), uint8(15))
	f.Add(int64(0), int64(0), int64(0), int64(0), uint8(15))
	f.Add(int64(100), int64(25), int64(80), int64(15), uint8(5))
	f.Add(int64(-1), int64(25), int64(0), int64(0), uint8(15))
	f.Fuzz(func(t *testing.T, input, output, cache, reasoning int64, present uint8) {
		values := []int64{input, output, cache, reasoning}
		keys := []string{"input_tokens", "output_tokens", "cached_input_tokens", "reasoning_output_tokens"}
		parts := []string{}
		invalid := false
		for i, v := range values {
			if present&(1<<i) == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%q:%d", keys[i], v))
			invalid = invalid || v < 0 || v > 1e12
		}
		invalid = invalid || (present&5 == 5 && cache > input) || (present&10 == 10 && reasoning > output)
		r, err := parseCodexEvents(codexTestEvents("{" + strings.Join(parts, ",") + "}"))
		if (err != nil) != invalid {
			t.Fatalf("usage validation disagrees with native count bounds: mask=%d values=%v err=%v", present, values, err)
		}
		if !invalid && present&3 == 3 {
			if r.Usage == nil || r.Usage.InputTokens != input || r.Usage.OutputTokens != output {
				t.Fatalf("native primary counts changed: %+v", r.Usage)
			}
		} else if r.Usage != nil {
			t.Fatalf("invented complete usage: %+v", r.Usage)
		}
	})
}
