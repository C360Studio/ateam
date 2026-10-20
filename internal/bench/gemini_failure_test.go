package bench

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func turnLimitFixture(message string) string {
	data, _ := json.MarshalIndent(map[string]any{"session_id": "fixture-session", "error": map[string]any{"type": "FatalTurnLimitedError", "message": message, "code": 53}}, "", "  ")
	return string(data)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Gemini fatal stderr evidence
func TestGeminiTurnLimitEnvelope(t *testing.T) {
	valid := turnLimitFixture("Reached max session turns.")
	want := GeminiFailureEvidence{SessionID: "fixture-session", Type: "FatalTurnLimitedError", Message: "Reached max session turns.", ExitCode: 53}
	for name, input := range map[string]string{
		"many-duplicates":  strings.Repeat(valid+"\n", 2000),
		"exact-byte-limit": strings.Repeat(" ", maxAdapterBytes-len(valid)-1) + "\n" + valid,
		"bare":             valid, "prefixed": "[ERROR] " + valid,
		"duplicate":  "Warning: fallback\n" + valid + "\n[ERROR] " + valid + "\n",
		"whitespace": "  " + valid + "  \nWarning: cleanup\n",
	} {
		t.Run(name, func(t *testing.T) {
			got := parseGeminiTurnLimit(input)
			if got == nil || *got != want {
				t.Fatalf("native turn-limit envelope lost: got=%+v", got)
			}
		})
	}
	for name, input := range map[string]string{
		"one-over-byte-limit": strings.Repeat(" ", maxAdapterBytes-len(valid)) + "\n" + valid,
		"empty":               "", "warning-only": "warning: " + valid,
		"wrong-code":        strings.ReplaceAll(valid, `"code": 53`, `"code": 1`),
		"wrong-type":        strings.ReplaceAll(valid, "FatalTurnLimitedError", "APIError"),
		"missing-session":   strings.ReplaceAll(valid, `"fixture-session"`, `""`),
		"missing-message":   strings.ReplaceAll(valid, `"Reached max session turns."`, `"  "`),
		"missing-code":      strings.ReplaceAll(valid, `"code": 53,`, ""),
		"unknown-field":     strings.ReplaceAll(valid, `"code": 53`, `"extra": 1, "code": 53`),
		"case-alias":        strings.ReplaceAll(valid, `"code": 53`, `"code": 52, "Code": 53`),
		"case-replacement":  strings.ReplaceAll(valid, `"session_id"`, `"SESSION_ID"`),
		"invalid-utf8":      strings.ReplaceAll(valid, "Reached max session turns.", string([]byte{0xff})),
		"duplicate-key":     strings.ReplaceAll(valid, `"code": 53`, `"code": 52, "code": 53`),
		"trailing-junk":     valid + " garbage\n",
		"adjacent":          valid + valid,
		"malformed-first":   "{broken\n" + valid,
		"malformed-last":    valid + "\n[ERROR] {",
		"conflicting":       valid + "\n" + turnLimitFixture("Different limit"),
		"unexpected-object": valid + "\n{}",
		"oversize":          strings.Repeat("w", maxAdapterBytes) + "\n" + valid,
	} {
		t.Run(name, func(t *testing.T) {
			if got := parseGeminiTurnLimit(input); got != nil {
				t.Fatalf("ambiguous or invalid envelope normalized: %+v", got)
			}
		})
	}
}

// The message is an independent oracle: its JSON escaping must not create a
// framing boundary, and a malformed earlier candidate must never be skipped.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Gemini fatal stderr evidence
func FuzzGeminiTurnLimitFraming(f *testing.F) {
	for _, message := range []string{"limit", "brace } and {", "newline\n[ERROR] {\n", "quote \" and slash \\"} {
		f.Add(message)
	}
	f.Fuzz(func(t *testing.T, message string) {
		if len(message) > 4096 || !utf8.ValidString(message) || strings.TrimSpace(message) == "" {
			t.Skip()
		}
		envelope := turnLimitFixture(message)
		got := parseGeminiTurnLimit("Warning: bounded fixture\n" + envelope + "\n[ERROR] " + envelope)
		if got == nil || got.Message != message || got.SessionID != "fixture-session" || got.Type != "FatalTurnLimitedError" || got.ExitCode != 53 {
			t.Fatal("escaped message or duplicate envelope changed reported evidence")
		}
		if parseGeminiTurnLimit("[ERROR] {invalid\n"+envelope) != nil {
			t.Fatal("malformed envelope was skipped")
		}
	})
}
