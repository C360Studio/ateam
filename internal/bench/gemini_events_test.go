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

func TestGeminiRecordedOutput(t *testing.T) {
	for _, tc := range []struct {
		name, terminal, usage string
		wantError             bool
		input, output         int64
	}{
		{"success", "completed", "reported", false, 103, 25},
		{"missing", "completed", "missing", false, 0, 0},
		{"partial", "completed", "partial", false, 0, 0},
		{"failure", "failed", "reported", true, 41, 7},
		{"retried", "completed", "partial", false, 0, 0},
		{"failed-defaults", "failed", "partial", true, 0, 0},
		{"interrupted", "incomplete", "missing", true, 0, 0},
		{"malformed", "incomplete", "invalid", true, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "gemini", tc.name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			r, err := parseGeminiOutput(string(data))
			if (err != nil) != tc.wantError || r.Native.Terminal != tc.terminal || r.Native.UsageStatus != tc.usage {
				t.Fatalf("response=%+v native=%+v error=%v", r, r.Native, err)
			}
			if tc.usage == "reported" {
				if r.Usage == nil || r.Usage.InputTokens != tc.input || r.Usage.OutputTokens != tc.output {
					t.Fatalf("native counts were normalized incorrectly: %+v", r.Usage)
				}
			} else if r.Usage != nil {
				t.Fatalf("invented complete usage: %+v", r.Usage)
			}
			if r.Native.ResolvedModel != nil || r.Native.ResolvedReasoning != nil {
				t.Fatal("invented resolved backend model or reasoning")
			}
			if tc.name == "success" {
				if r.Source != "package safepath\n" || *r.Usage.CachedInputTokens != 80 || *r.Usage.ReasoningOutputTokens != 15 {
					t.Fatalf("lost source or usage detail: %+v", r)
				}
				if !reflect.DeepEqual(r.Native.ReportedModels, []string{"gemini-3-flash-preview"}) {
					t.Fatalf("lost model evidence: %+v", r.Native)
				}
				if !reflect.DeepEqual(r.Native.ObservedItemTypes, []string{"read_file"}) || !reflect.DeepEqual(r.Native.CompletedItemTypes, []string{"read_file"}) {
					t.Fatalf("tool evidence was inferred or dropped: %+v", r.Native)
				}
			}
		})
	}
}

func geminiTestOutput(tokens string) string {
	return `{"session_id":"test-session","response":"{\"source\":\"package safepath\"}","stats":{"models":{"gemini-3-flash-preview":{"api":{"totalRequests":1,"totalErrors":0},"tokens":` + tokens + `}}}}`
}

const geminiGoodTokens = `{"prompt":100,"candidates":10,"total":125,"cached":80,"thoughts":15,"tool":0,"input":20}`

func TestGeminiRefusesAmbiguousOutput(t *testing.T) {
	good := geminiTestOutput(geminiGoodTokens)
	for name, output := range map[string]string{
		"duplicate report":   good + good,
		"trailing malformed": good + `{`,
		"duplicate response": strings.Replace(good, `"response":`, `"response":"first","response":`, 1),
		"duplicate count":    geminiTestOutput(`{"prompt":1,"prompt":2}`),
		"fractional count":   geminiTestOutput(`{"prompt":1.5}`),
		"negative count":     geminiTestOutput(`{"prompt":-1}`),
		"too large":          geminiTestOutput(`{"prompt":1000000000001}`),
		"cache exceeds":      geminiTestOutput(`{"prompt":1,"cached":2}`),
		"uncached mismatch":  geminiTestOutput(`{"prompt":10,"cached":2,"input":9}`),
		"total mismatch":     geminiTestOutput(strings.Replace(geminiGoodTokens, `"total":125`, `"total":126`, 1)),
		"empty model":        strings.Replace(good, `"gemini-3-flash-preview"`, `""`, 1),
		"bad tools":          strings.Replace(good, `"models":`, `"tools":{"byName":{"read_file":{"count":1,"success":2,"fail":0}}},"models":`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseGeminiOutput(output)
			if err == nil || r.Usage != nil || r.Native.UsageStatus != "invalid" {
				t.Fatalf("ambiguous report supplied accountable usage: %+v error=%v", r, err)
			}
		})
	}
	for _, message := range []string{`{}`, `{"source":""}`, `{"source":"one","source":"two"}`, `{"source":"x","extra":true}`, `null`} {
		encoded, _ := json.Marshal(message)
		output := strings.Replace(good, `"{\"source\":\"package safepath\"}"`, string(encoded), 1)
		r, err := parseGeminiOutput(output)
		if err == nil || r.Usage == nil {
			t.Fatalf("invalid final source must fail without losing usage: %+v error=%v", r, err)
		}
	}
}

func TestGeminiReportedUsageMissingAndAggregation(t *testing.T) {
	for _, field := range []string{"prompt", "candidates", "thoughts", "total", "tool"} {
		var tokens map[string]any
		if err := json.Unmarshal([]byte(geminiGoodTokens), &tokens); err != nil {
			t.Fatal(err)
		}
		delete(tokens, field)
		raw, _ := json.Marshal(tokens)
		r, err := parseGeminiOutput(geminiTestOutput(string(raw)))
		if err != nil || r.Usage != nil || r.Native.UsageStatus != "partial" {
			t.Fatalf("missing %s was invented: %+v %v", field, r, err)
		}
	}
	for _, value := range []string{"null", "{}"} {
		r, err := parseGeminiOutput(geminiTestOutput(value))
		if err != nil || r.Usage != nil || r.Native.UsageStatus != "partial" {
			t.Fatalf("absent counts invented: %+v %v", r, err)
		}
	}
	// Role totals repeat model totals and must not be charged a second time.
	output := strings.Replace(geminiTestOutput(geminiGoodTokens), `"tokens":`, `"roles":{"main":{"tokens":`+geminiGoodTokens+`}},"tokens":`, 1)
	r, err := parseGeminiOutput(output)
	if err != nil || r.Usage.InputTokens != 100 || r.Usage.OutputTokens != 25 {
		t.Fatalf("role counts doubled: %+v %v", r, err)
	}
	// Multiple models are retained to let the fixed-model runner reject routing.
	output = strings.Replace(geminiTestOutput(geminiGoodTokens), `"models":{`, `"models":{"other-model":{"api":{"totalRequests":1,"totalErrors":0},"tokens":`+geminiGoodTokens+`},`, 1)
	r, err = parseGeminiOutput(output)
	if err != nil || r.Usage.InputTokens != 200 || r.Usage.OutputTokens != 50 || !reflect.DeepEqual(r.Native.ReportedModels, []string{"gemini-3-flash-preview", "other-model"}) {
		t.Fatalf("lost model usage: %+v native=%+v %v", r, r.Native, err)
	}
	// Aggregate bounds also apply across individually valid model counts.
	huge := `{"prompt":1000000000000,"candidates":0,"total":1000000000000,"thoughts":0,"tool":0}`
	output = strings.Replace(geminiTestOutput(huge), `"models":{`, `"models":{"other-model":{"api":{"totalRequests":1,"totalErrors":0},"tokens":`+huge+`},`, 1)
	if r, err = parseGeminiOutput(output); err == nil || r.Usage != nil {
		t.Fatal("aggregate overflow accepted")
	}
}

// Object order is irrelevant, while two terminal reports are never one outcome.
func TestGeminiOutputFieldOrder(t *testing.T) {
	fields := []string{`"session_id":"s"`, `"response":"{\"source\":\"package safepath\"}"`, `"stats":{"models":{"model":{"api":{"totalRequests":1,"totalErrors":0},"tokens":` + geminiGoodTokens + `}}}`}
	checked := 0
	for a := 0; a < 3; a++ {
		for b := 0; b < 3; b++ {
			for c := 0; c < 3; c++ {
				if a == b || a == c || b == c {
					continue
				}
				checked++
				r, err := parseGeminiOutput("{" + strings.Join([]string{fields[a], fields[b], fields[c]}, ",") + "}")
				if err != nil || r.Usage == nil || r.Usage.InputTokens != 100 || r.Usage.OutputTokens != 25 {
					t.Fatalf("order %d%d%d: %+v %v", a, b, c, r, err)
				}
			}
		}
	}
	if checked != 6 {
		t.Fatalf("order assertions did not activate: %d", checked)
	}
}

func TestGeminiOutputSizeBoundary(t *testing.T) {
	valid := geminiTestOutput(geminiGoodTokens)
	exact := valid + strings.Repeat(" ", maxAdapterBytes-len(valid))
	if _, err := parseGeminiOutput(exact); err != nil {
		t.Fatalf("exact 1 MiB boundary rejected: %v", err)
	}
	if _, err := parseGeminiOutput(exact + " "); err == nil {
		t.Fatal("accepted output over 1 MiB")
	}
	if err := uniqueJSON([]byte(strings.Repeat("[", 66) + "0" + strings.Repeat("]", 66))); err == nil {
		t.Fatal("accepted excessive nesting")
	}
}

// Generated count/presence combinations check native limits without inferring absent fields.
func FuzzGeminiNativeUsage(f *testing.F) {
	f.Add(int64(100), int64(10), int64(80), int64(15), int64(3), uint8(127))
	f.Add(int64(1e12), int64(0), int64(1e12), int64(0), int64(0), uint8(127))
	f.Add(int64(1e12+1), int64(0), int64(0), int64(0), int64(0), uint8(127))
	f.Add(int64(0), int64(0), int64(0), int64(0), int64(0), uint8(127))
	f.Add(int64(100), int64(10), int64(80), int64(15), int64(0), uint8(5))
	f.Add(int64(-1), int64(10), int64(0), int64(15), int64(0), uint8(127))
	f.Fuzz(func(t *testing.T, prompt, candidate, cache, thought, tool int64, present uint8) {
		// Keep derived fixture arithmetic inside int64 while still reaching each boundary.
		for _, v := range []int64{prompt, candidate, cache, thought, tool} {
			if v < -2e12 || v > 2e12 {
				return
			}
		}
		total := prompt + candidate + thought + tool
		values := []int64{prompt, candidate, cache, thought, tool, total, prompt - cache}
		keys := []string{"prompt", "candidates", "cached", "thoughts", "tool", "total", "input"}
		parts := []string{}
		invalid := false
		for i, v := range values {
			if present&(1<<i) == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%q:%d", keys[i], v))
			invalid = invalid || v < 0 || v > 1e12
		}
		invalid = invalid || (present&5 == 5 && cache > prompt)
		r, err := parseGeminiOutput(geminiTestOutput("{" + strings.Join(parts, ",") + "}"))
		if (err != nil) != invalid {
			t.Fatalf("native count bounds: mask=%d values=%v err=%v", present, values, err)
		}
		// Required primary fields are prompt,candidates,thoughts,tool,total (mask59).
		if !invalid && present&59 == 59 {
			if r.Usage == nil || r.Usage.InputTokens != prompt+tool || r.Usage.OutputTokens != candidate+thought {
				t.Fatalf("native normalization changed: %+v", r.Usage)
			}
		} else if r.Usage != nil {
			t.Fatalf("invented complete usage: %+v", r.Usage)
		}
	})
}

func TestGeminiMalformedModelEvidenceDeterministic(t *testing.T) {
	output := strings.Replace(geminiTestOutput(geminiGoodTokens), `"models":{`, `"models":{"broken":{"tokens":{"prompt":-1}},`, 1)
	first, firstErr := parseGeminiOutput(output)
	if firstErr == nil {
		t.Fatal("malformed model unexpectedly accepted")
	}
	for i := 0; i < 1000; i++ {
		actual, err := parseGeminiOutput(output)
		if err == nil || err.Error() != firstErr.Error() || !reflect.DeepEqual(actual, first) {
			t.Fatalf("malformed evidence depends on map iteration: first=%+v actual=%+v", first.Native, actual.Native)
		}
	}
}

func TestGeminiToolEvidenceRequiresNativeCounters(t *testing.T) {
	good := geminiTestOutput(geminiGoodTokens)
	for _, tc := range []struct {
		tools               string
		observed, completed []string
	}{
		{`null`, nil, nil},
		{`{}`, nil, nil},
		{`{"byName":{}}`, []string{}, []string{}},
		{`{"byName":{"configured_only":{}}}`, []string{}, []string{}},
		{`{"byName":{"failed_call":{"count":1,"success":0,"fail":1}}}`, []string{"failed_call"}, []string{"failed_call"}},
	} {
		r, err := parseGeminiOutput(strings.Replace(good, `"models":`, `"tools":`+tc.tools+`,"models":`, 1))
		if err != nil || !reflect.DeepEqual(r.Native.ObservedItemTypes, tc.observed) || !reflect.DeepEqual(r.Native.CompletedItemTypes, tc.completed) {
			t.Fatalf("tool evidence differs from counters: %+v %v", r.Native, err)
		}
	}
}

func TestGeminiAPIErrorUsageIsUnknown(t *testing.T) {
	for _, tc := range []struct{ name, api, tokens string }{
		{"successful retry", `{"totalRequests":2,"totalErrors":1}`, geminiGoodTokens},
		{"failed call zero defaults", `{"totalRequests":1,"totalErrors":1}`, `{"prompt":0,"candidates":0,"total":0,"thoughts":0,"cached":0,"tool":0,"input":0}`},
		{"missing api", `null`, geminiGoodTokens},
		{"missing error count", `{"totalRequests":1}`, geminiGoodTokens},
		{"missing request count", `{"totalErrors":0}`, geminiGoodTokens},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := strings.Replace(geminiTestOutput(tc.tokens), `"api":{"totalRequests":1,"totalErrors":0}`, `"api":`+tc.api, 1)
			r, err := parseGeminiOutput(output)
			if err != nil || r.Usage != nil || r.Native.UsageStatus != "partial" || r.Native.ReportedUsage == nil {
				t.Fatalf("API errors or absent error counters supplied complete usage: %+v native=%+v err=%v", r.Usage, r.Native, err)
			}
			var expected *GeminiAPIUsage
			if err := json.Unmarshal([]byte(tc.api), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r.Native.ReportedUsage.API["gemini-3-flash-preview"], expected) {
				t.Fatal("lost native API counters")
			}
		})
	}
}

func TestGeminiInvalidAPICounters(t *testing.T) {
	for _, api := range []string{
		`{"totalRequests":1,"totalErrors":2}`,
		`{"totalRequests":1,"totalErrors":-1}`,
		`{"totalRequests":-1,"totalErrors":0}`,
		`{"totalRequests":1000000000001,"totalErrors":0}`,
		`{"totalRequests":0,"totalErrors":0}`,
	} {
		output := strings.Replace(geminiTestOutput(geminiGoodTokens), `"api":{"totalRequests":1,"totalErrors":0}`, `"api":`+api, 1)
		r, err := parseGeminiOutput(output)
		if err == nil || r.Usage != nil || r.Native.UsageStatus != "invalid" {
			t.Fatalf("invalid API counters accepted: %s %+v %v", api, r.Native, err)
		}
	}
}

func TestGeminiAPICounterBoundaries(t *testing.T) {
	values := []int64{-1, 0, 1, 2, 1e12, 1e12 + 1}
	checked := 0
	for _, requests := range values {
		for _, errors := range values {
			for present := 0; present < 4; present++ {
				checked++
				api := map[string]int64{}
				invalid := false
				if present&1 != 0 {
					api["totalRequests"] = requests
					invalid = invalid || requests < 0 || requests > 1e12 || requests == 0
				}
				if present&2 != 0 {
					api["totalErrors"] = errors
					invalid = invalid || errors < 0 || errors > 1e12
				}
				invalid = invalid || (present == 3 && errors > requests)
				data, _ := json.Marshal(api)
				output := strings.Replace(geminiTestOutput(geminiGoodTokens), `"api":{"totalRequests":1,"totalErrors":0}`, `"api":`+string(data), 1)
				r, err := parseGeminiOutput(output)
				if (err != nil) != invalid {
					t.Fatalf("API boundary requests=%d errors=%d present=%d: %v", requests, errors, present, err)
				}
				complete := !invalid && present == 3 && requests > 0 && errors == 0
				if (r.Usage != nil) != complete {
					t.Fatalf("unknown API accounting fabricated: requests=%d errors=%d present=%d usage=%+v", requests, errors, present, r.Usage)
				}
			}
		}
	}
	if checked != 144 {
		t.Fatalf("boundary assertions did not activate: %d", checked)
	}
}
