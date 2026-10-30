package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestOpenSpecQueueAndCheck(t *testing.T) {
	// Hide any host openspec so the skip notice below does not depend on the machine (#205).
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "openspec", "changes", "setup-01")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tasksMD := `# Tasks

This header explains Hold: markers above the first section heading.

## 1. Implementation

- [x] 1.1 Completed task mentioning halt condition that already passed.
- [ ] 1.2 Implement core state parser.
  - Hold: waiting on PR #48 to merge.
- [~] 1.3 Deliberately skipped legacy adapter.
- [ ] 1.4 If pre-v1 window closes, halt: record the missed window.
- [ ] 1.5 RED — integration suite is failing on port check.
- [ ] 1.6 **STILL OPEN** — decide whether Gitea webhook rides v0.1.
`
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(tasksMD), 0o644); err != nil {
		t.Fatal(err)
	}

	changes, err := ScanOpenSpecQueue(dir)
	if err != nil {
		t.Fatalf("ScanOpenSpecQueue: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(changes))
	}
	ch := changes[0]
	if ch.CompletedTasks != 1 || ch.TotalTasks != 6 {
		t.Errorf("counts = %d/%d, want 1/6", ch.CompletedTasks, ch.TotalTasks)
	}
	if len(ch.Caveats) != 5 {
		t.Fatalf("got %d caveats, want 5: %+v", len(ch.Caveats), ch.Caveats)
	}

	wantMarkers := []string{"BLOCKED", "WONTDO", "HALT", "RED", "OPEN-Q"}
	for i, want := range wantMarkers {
		if ch.Caveats[i].Marker != want {
			t.Errorf("caveat[%d] marker = %q, want %q (%+v)", i, ch.Caveats[i].Marker, want, ch.Caveats[i])
		}
	}

	out, code, err := CheckOpenSpec(context.Background(), dir)
	if err != nil || code != 0 {
		t.Fatalf("CheckOpenSpec clean failed: code=%d err=%v out=%s", code, err, out)
	}
	if !strings.Contains(out, "holds: ok (1 tasks.md read)") {
		t.Errorf("unexpected check output: %q", out)
	}
	if !strings.Contains(out, "openspec: strict validation skipped") {
		t.Errorf("check without node_modules/.bin/openspec does not say strict validation was skipped: %q", out)
	}

	// #127: with the validator installed (a stub that passes), no skip notice.
	binDir := filepath.Join(dir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "openspec"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code, err = CheckOpenSpec(context.Background(), dir)
	if err != nil || code != 0 || strings.Contains(out, "strict validation skipped") {
		t.Fatalf("CheckOpenSpec with validator installed: code=%d err=%v out=%q", code, err, out)
	}
	if err := os.RemoveAll(filepath.Join(dir, "node_modules")); err != nil {
		t.Fatal(err)
	}

	// Add a stray Hold: below the ## heading outside any task block.
	strayMD := tasksMD + "\nHold: stray hold outside every task.\n"
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(strayMD), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code, err = CheckOpenSpec(context.Background(), dir)
	if err == nil || code != 1 {
		t.Fatalf("expected exit code 1 for stray hold, got code=%d err=%v out=%s", code, err, out)
	}
	if !strings.Contains(out, "Hold: outside every task") {
		t.Errorf("missing stray hold message in %q", out)
	}

	// Write invalid UTF-8 and verify exit code 2.
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, code, err = CheckOpenSpec(context.Background(), dir)
	if err == nil || code != 2 {
		t.Fatalf("expected exit code 2 for invalid UTF-8, got code=%d err=%v", code, err)
	}

	// #48: Completed unarchived change (N/N, N > 0, 0 caveats) prints "done, not archived".
	formatted, hasCaveats := FormatOpenSpecQueue([]OpenSpecChange{
		{Name: "active-change", CompletedTasks: 1, TotalTasks: 3},
		{Name: "finished-change", CompletedTasks: 4, TotalTasks: 4},
	})
	if hasCaveats {
		t.Fatalf("expected hasCaveats=false when no caveats are present")
	}
	if !strings.Contains(formatted, "done, not archived — move openspec/changes/finished-change to openspec/changes/archive/<date>-finished-change") {
		t.Fatalf("expected done, not archived notice in FormatOpenSpecQueue, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "1 change(s) in flight, 1 completed (done, not archived).") {
		t.Fatalf("expected split in-flight and completed counts in FormatOpenSpecQueue, got:\n%s", formatted)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestCleanTaskTextKeepsUTF8AtTheCut(t *testing.T) {
	got := cleanTaskText("a" + strings.Repeat("é", 60))
	if !utf8.ValidString(got) || !strings.HasSuffix(got, "…") || len(strings.TrimSuffix(got, "…")) > 104 {
		t.Fatalf("cleanTaskText = %q (len %d), want valid UTF-8 truncated at <=104 bytes with trailing ellipsis", got, len(got))
	}

	wordCut := cleanTaskText("- [ ] Hold: owner authorization required before bridge installation, downloaded-code execution or adapter setup")
	wantWordCut := "Hold: owner authorization required before bridge installation, downloaded-code execution or adapter…"
	if wordCut != wantWordCut {
		t.Fatalf("cleanTaskText word boundary = %q, want %q", wordCut, wantWordCut)
	}

	longToken := cleanTaskText("Hold: waiting for acknowledgement " + strings.Repeat("supercalifragilisticexpialidocious", 3))
	wantLongToken := "Hold: waiting for acknowledgement…"
	if longToken != wantLongToken {
		t.Fatalf("cleanTaskText long token boundary = %q, want %q", longToken, wantLongToken)
	}
	if cleanTaskText(got) != got || cleanTaskText(wordCut) != wordCut || cleanTaskText(longToken) != longToken {
		t.Fatalf("expected cleanTaskText to be idempotent on already-truncated text")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestHoldDisplaySinglePrefixAndWordBoundaryInPickup(t *testing.T) {
	tasksMD := `# Tasks

## 1. Live Verification

- [ ] Run A/B live benchmark
  - Hold: owner authorization for live API spend; run after gate 1 self-test is green and verified in staging.
- [ ] Hold: owner authorization required before bridge installation, downloaded-code execution or adapter setup
- [ ] 1.2 Implement core state parser.
  - Hold: waiting on PR #48 to merge.
`
	ch, holds, err := ParseTasksMarkdown("openspec/changes/ab-live-bench/tasks.md", []byte(tasksMD))
	if err != nil {
		t.Fatalf("ParseTasksMarkdown: %v", err)
	}
	if len(holds) != 3 {
		t.Fatalf("got %d holds, want 3: %+v", len(holds), holds)
	}
	wantHolds := []string{
		"Hold: owner authorization for live API spend; run after gate 1 self-test is green and verified in…",
		"Hold: owner authorization required before bridge installation, downloaded-code execution or adapter…",
		"Hold: waiting on PR #48 to merge.",
	}
	for i, want := range wantHolds {
		if holds[i].Text != want {
			t.Errorf("holds[%d].Text = %q, want %q", i, holds[i].Text, want)
		}
		if ch.Caveats[i].Text != want {
			t.Errorf("caveats[%d].Text = %q, want %q", i, ch.Caveats[i].Text, want)
		}
	}
	if holds[0].TaskNumber != "" || holds[1].TaskNumber != "" || holds[2].TaskNumber != "1.2" {
		t.Errorf("unexpected task numbers: %q, %q, %q", holds[0].TaskNumber, holds[1].TaskNumber, holds[2].TaskNumber)
	}

	st := &State{
		Mode: ModeOn,
		Holds: []HoldItem{
			holds[0],
			{
				File: "openspec/changes/cached/tasks.md",
				Line: 16,
				Text: "Run Hold: owner authorization for live API spend; run after gate 1 self-test is green and verified in staging.",
			},
			{
				File: "openspec/changes/cached/tasks.md",
				Line: 20,
				Text: "Hold: Hold: owner authorization required before bridge installation, downloaded-code execution or adapter setup",
			},
		},
	}
	summary := BuildPickupSummary(st)
	for _, bad := range []string{"Run Hold:", "Hold: Hold:", "is gr\n", "or adapte\n"} {
		if strings.Contains(summary, bad) {
			t.Fatalf("pickup summary contained untidy hold text %q:\n%s", bad, summary)
		}
	}
	for _, wantLine := range []string{
		"- openspec/changes/ab-live-bench/tasks.md:6 " + wantHolds[0],
		"- openspec/changes/cached/tasks.md:16 " + wantHolds[0],
		"- openspec/changes/cached/tasks.md:20 " + wantHolds[1],
	} {
		if !strings.Contains(summary, wantLine) {
			t.Fatalf("pickup summary missing expected hold line %q:\n%s", wantLine, summary)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestProposalApprovedByHeader(t *testing.T) {
	cases := []struct {
		name     string
		proposal *string // nil: no proposal.md
		want     string
	}{
		{name: "filled", proposal: ptr("# Change\n\nApproved-by: Coby (2026-10-26)\n\n## Why\n"), want: "Coby (2026-10-26)"},
		{name: "empty", proposal: ptr("# Change\n\nApproved-by:\n\n## Why\n"), want: ""},
		{name: "empty with trailing space", proposal: ptr("Approved-by:   \n"), want: ""},
		{name: "missing header", proposal: ptr("# Change\n\n## Why\n\nApproved by the owner later.\n"), want: ""},
		{name: "missing proposal", proposal: nil, want: ""},
		{name: "fenced example is not the header", proposal: ptr("```\nApproved-by: <name> (<YYYY-MM-DD>)\n```\nApproved-by:\n"), want: ""},
		{name: "longer outer fence hides inner example", proposal: ptr("````markdown\n```\nApproved-by: Example (2026-10-26)\n```\n````\n"), want: ""},
		{name: "tilde fence ignores backtick lines", proposal: ptr("~~~\n```\nApproved-by: Example (2026-10-26)\n~~~\n"), want: ""},
		{name: "fence line with trailing text does not close", proposal: ptr("```\n``` not a close\nApproved-by: Example (2026-10-26)\n```\n"), want: ""},
		{name: "real header after outer fence", proposal: ptr("````\n```\n````\nApproved-by: Coby (2026-10-26)\n"), want: "Coby (2026-10-26)"},
		{name: "first header wins", proposal: ptr("Approved-by: Ann (2026-01-02)\nApproved-by: Bob (2026-03-04)\n"), want: "Ann (2026-01-02)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			changeDir := filepath.Join(dir, "openspec", "changes", "c1")
			if err := os.MkdirAll(changeDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("## 1. Tasks\n\n- [ ] 1.1 Do it\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.proposal != nil {
				if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte(*tc.proposal), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			changes, err := ScanOpenSpecQueue(dir)
			if err != nil {
				t.Fatalf("ScanOpenSpecQueue: %v", err)
			}
			if len(changes) != 1 {
				t.Fatalf("got %d changes, want 1", len(changes))
			}
			if changes[0].ApprovedBy != tc.want {
				t.Fatalf("ApprovedBy = %q, want %q", changes[0].ApprovedBy, tc.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestOpenQApprovalHoldKeepsChangeOpen(t *testing.T) {
	tasks := "## 1. Implementation\n\n" +
		"- [ ] 1.1 Owner approves the proposal\n" +
		"  Hold: OPEN-Q proposal awaits owner approval\n" +
		"- [x] 1.2 Parser\n" +
		"- [x] 1.3 Docs\n"
	ch, holds, err := ParseTasksMarkdown("openspec/changes/c1/tasks.md", []byte(tasks))
	if err != nil {
		t.Fatal(err)
	}
	if ch.IsCompletedUnarchived() {
		t.Fatalf("change held for approval reads as completed unarchived: %+v", ch)
	}
	if len(holds) != 1 || holds[0].Text != "Hold: OPEN-Q proposal awaits owner approval" || holds[0].TaskNumber != "1.1" {
		t.Fatalf("holds = %+v, want the approval hold on task 1.1", holds)
	}
	ch.Name = "c1"
	out, flagged := FormatOpenSpecQueue([]OpenSpecChange{ch})
	if !flagged || strings.Contains(out, "done, not archived") || !strings.Contains(out, "OPEN-Q   L4     Hold: OPEN-Q proposal awaits owner approval") {
		t.Fatalf("queue output does not flag the approval hold:\n%s", out)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestHoldMarkerNamesItsLeadingCaveat(t *testing.T) {
	for _, tc := range []struct{ hold, want string }{
		{"Hold: OPEN-Q proposal awaits owner approval", "OPEN-Q"},
		{"Hold: HALT until the owner rules on #12", "HALT"},
		{"Hold: RED: main CI is failing", "RED"},
		{"Hold: waiting on #12", "BLOCKED"},
		{"Hold: waiting on the failing OPEN-Q from #12", "BLOCKED"},
	} {
		tasks := "## 1. Tasks\n\n- [ ] 1.1 Work\n  " + tc.hold + "\n"
		ch, _, err := ParseTasksMarkdown("openspec/changes/c1/tasks.md", []byte(tasks))
		if err != nil {
			t.Fatal(err)
		}
		if len(ch.Caveats) != 1 || ch.Caveats[0].Marker != tc.want || ch.Caveats[0].Text != tc.hold {
			t.Errorf("%q: caveats = %+v, want one %s caveat", tc.hold, ch.Caveats, tc.want)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestProposalLengthWarning(t *testing.T) {
	// Fixtures are not strict-valid; keep a host openspec on PATH out of a length check (#205).
	t.Setenv("PATH", t.TempDir())
	for _, tc := range []struct {
		name     string
		proposal string
		warn     bool
	}{
		{"60 lines", strings.Repeat("line\n", 60), false},
		{"60 lines, no final newline", strings.Repeat("line\n", 59) + "line", false},
		{"61 lines", strings.Repeat("line\n", 61), true},
		{"61 lines, no final newline", strings.Repeat("line\n", 60) + "line", true},
		{"60 lines plus a trailing blank line", strings.Repeat("line\n", 60) + "\n", true},
	} {
		dir := t.TempDir()
		changeDir := filepath.Join(dir, "openspec", "changes", "long")
		if err := os.MkdirAll(changeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("## 1. Tasks\n\n- [ ] 1.1 Do it\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte(tc.proposal), 0o644); err != nil {
			t.Fatal(err)
		}
		out, code, err := CheckOpenSpec(context.Background(), dir)
		if code != 0 || err != nil {
			t.Fatalf("%s: CheckOpenSpec code=%d err=%v; a long proposal must warn, not fail", tc.name, code, err)
		}
		wantLine := "warning: openspec/changes/long/proposal.md has 61 lines, over the one-page limit of 60"
		if got := strings.Contains(out, wantLine); got != tc.warn {
			t.Fatalf("%s: warning present = %v, want %v; output:\n%s", tc.name, got, tc.warn, out)
		}
		if !tc.warn && strings.Contains(out, "warning:") {
			t.Fatalf("%s: unexpected warning:\n%s", tc.name, out)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Opt-in design track
func TestDesignLengthWarning(t *testing.T) {
	// Fixtures are not strict-valid; keep a host openspec on PATH out of a length check (#205).
	t.Setenv("PATH", t.TempDir())
	const absent = "\x00absent"
	for _, tc := range []struct {
		name   string
		design string
		warn   bool
	}{
		{"120 lines", strings.Repeat("line\n", 120), false},
		{"121 lines", strings.Repeat("line\n", 121), true},
		{"121 lines, no final newline", strings.Repeat("line\n", 120) + "line", true},
		{"no design.md", absent, false},
	} {
		dir := t.TempDir()
		changeDir := filepath.Join(dir, "openspec", "changes", "deep")
		if err := os.MkdirAll(changeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("## 1. Tasks\n\n- [ ] 1.1 Do it\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if tc.design != absent {
			if err := os.WriteFile(filepath.Join(changeDir, "design.md"), []byte(tc.design), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		out, code, err := CheckOpenSpec(context.Background(), dir)
		if code != 0 || err != nil {
			t.Fatalf("%s: CheckOpenSpec code=%d err=%v; a long design must warn, not fail", tc.name, code, err)
		}
		wantLine := "warning: openspec/changes/deep/design.md has 121 lines, over the limit of 120"
		if got := strings.Contains(out, wantLine); got != tc.warn {
			t.Fatalf("%s: warning present = %v, want %v; output:\n%s", tc.name, got, tc.warn, out)
		}
		if !tc.warn && strings.Contains(out, "warning:") {
			t.Fatalf("%s: unexpected warning:\n%s", tc.name, out)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestCheckOpenSpecFindsValidatorOnPATH(t *testing.T) {
	const pass, fail = "exit 0\n", "echo strict-failed\nexit 1\n"
	const versioned = "[ \"$1\" = --version ] && echo 1.7.0\nexit 0\n"
	for _, tc := range []struct {
		name, local, onPath string // "" means that validator is absent
		wantCode            int
		wantSkip            bool
		wantOK              string // "local" or "path": which binary the success line names (#264)
		wantVersion         string
	}{
		{"neither installed", "", "", 0, true, "", ""},
		{"PATH only, passes", "", versioned, 0, false, "path", "1.7.0"},
		{"PATH only, fails", "", fail, 1, false, "", ""},
		{"local wins over PATH", pass, fail, 0, false, "local", "version unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "openspec", "changes"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			pathDir := t.TempDir()
			t.Setenv("PATH", pathDir)
			if tc.onPath != "" {
				writeFake(t, filepath.Join(pathDir, "openspec"), tc.onPath)
			}
			if tc.local != "" {
				binDir := filepath.Join(dir, "node_modules", ".bin")
				if err := os.MkdirAll(binDir, 0o755); err != nil {
					t.Fatal(err)
				}
				writeFake(t, filepath.Join(binDir, "openspec"), tc.local)
			}
			out, code, err := CheckOpenSpec(context.Background(), dir)
			if code != tc.wantCode || (err != nil) != (tc.wantCode != 0) {
				t.Fatalf("code=%d err=%v, want code %d; output:\n%s", code, err, tc.wantCode, out)
			}
			if tc.wantCode != 0 && !strings.Contains(out, "strict-failed") {
				t.Errorf("failing validator output not returned: %q", out)
			}
			if got := strings.Contains(out, "strict validation skipped"); got != tc.wantSkip {
				t.Errorf("skip notice = %v, want %v; output:\n%s", got, tc.wantSkip, out)
			}
			wantLine := ""
			switch tc.wantOK {
			case "path":
				wantLine = "openspec: strict ok (" + filepath.Join(pathDir, "openspec") + ", " + tc.wantVersion + ", 0 changes)\n"
			case "local":
				wantLine = "openspec: strict ok (" + filepath.Join(dir, "node_modules", ".bin", "openspec") + ", " + tc.wantVersion + ", 0 changes)\n"
			}
			if wantLine != "" && !strings.Contains(out, wantLine) {
				t.Errorf("success line missing, want %q; output:\n%s", wantLine, out)
			}
			if wantLine == "" && strings.Contains(out, "strict ok") {
				t.Errorf("success line printed without a passing validator; output:\n%s", out)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestCheckOpenSpecIgnoresValidatorInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "openspec", "changes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A planted ./openspec reached only through a "." PATH entry must not run (exec.ErrDot).
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "openspec"), []byte("#!/bin/sh\necho planted\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	t.Setenv("PATH", ".")
	out, code, err := CheckOpenSpec(context.Background(), dir)
	if code != 0 || err != nil || strings.Contains(out, "planted") || !strings.Contains(out, "strict validation skipped") {
		t.Fatalf("code=%d err=%v; a ./openspec must be skipped, not run; output:\n%s", code, err, out)
	}
}
