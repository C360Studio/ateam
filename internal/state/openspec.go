package state

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/testspec"
	"unicode/utf8"
)

// HoldItem represents an active Hold: marker inside an open Markdown task.
type HoldItem struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	TaskNumber string `json:"task_number,omitempty"`
	Text       string `json:"text"`
}

// OpenSpecCaveat represents a flagged open task or Hold: line in an OpenSpec change.
type OpenSpecCaveat struct {
	Marker string `json:"marker"` // WONTDO, HALT, RED, BLOCKED, OPEN-Q
	Line   int    `json:"line"`
	Text   string `json:"text"`
	// Full is the normalized, untruncated task text, for stable identity (Text is for display).
	Full string `json:"-"`
}

// OpenSpecChange summarizes an in-flight OpenSpec change under openspec/changes/<name>.
type OpenSpecChange struct {
	Name           string           `json:"name"`
	CompletedTasks int              `json:"completed_tasks"`
	TotalTasks     int              `json:"total_tasks"`
	Caveats        []OpenSpecCaveat `json:"caveats,omitempty"`
	StrayHolds     []int            `json:"stray_holds,omitempty"`
	// ApprovedBy is the value of the proposal.md `Approved-by:` header, trimmed. It is ""
	// when the header is empty, absent, or proposal.md is missing; the change's
	// `Hold: OPEN-Q` task, not this field, is what keeps an unapproved change open.
	ApprovedBy string `json:"approved_by,omitempty"`
	// proposalLines is the line count of proposal.md (0 when missing), for the one-page warning.
	proposalLines int
	// designLines is the line count of design.md (0 when missing), for the design length warning.
	designLines int
}

// maxProposalLines is the one-page limit for proposal.md; spec:check warns, never fails, past it.
const maxProposalLines = 60

// maxDesignLines is the design-track limit for design.md; spec:check warns, never fails, past it.
const maxDesignLines = 120

// ParseProposal returns the first `Approved-by:` header value outside code fences and the
// physical line count (a final newline ends the last line; it does not start another).
func ParseProposal(data []byte) (approvedBy string, lines int) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if text == "" {
		return "", 0
	}
	lines = strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		lines++
	}
	found := false
	testspec.EachUnfencedLine(text, func(line string) {
		if found {
			return
		}
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "Approved-by:"); ok {
			approvedBy, found = strings.TrimSpace(v), true
		}
	})
	return approvedBy, lines
}

// IsCompletedUnarchived reports whether all tasks in the change are done and no caveats remain.
func (ch OpenSpecChange) IsCompletedUnarchived() bool {
	return ch.TotalTasks > 0 && ch.CompletedTasks == ch.TotalTasks && len(ch.Caveats) == 0
}

// OpenSpecScenario represents a scenario block (#### Scenario: <Name>) under a requirement.
type OpenSpecScenario struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps,omitempty"`
}

// OpenSpecRequirement represents a normative requirement (### Requirement: <Name>) in a spec.md.
type OpenSpecRequirement struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Spec      string             `json:"spec"`
	Statement string             `json:"statement"`
	Scenarios []OpenSpecScenario `json:"scenarios,omitempty"`
}

// OpenSpecSpec represents a canonical specification file under openspec/specs/<name>/spec.md.
type OpenSpecSpec struct {
	Name         string                `json:"name"`
	Path         string                `json:"path"`
	Purpose      string                `json:"purpose"`
	Requirements []OpenSpecRequirement `json:"requirements,omitempty"`
}

var (
	taskLineRe   = regexp.MustCompile(`^(\s*)- \[(.)\](.*)$`)
	taskNumberRe = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*\.?$`)
	haltWordRe   = regexp.MustCompile(`(?i)\bhalt(s|ed|ing)?\b`)
	redWordRe    = regexp.MustCompile(`(?i)\b(red|failed|failing)\b`)
	holdWordRe   = regexp.MustCompile(`(?i)\b(hold|blocked|blocking)\b`)
	wontdoWordRe = regexp.MustCompile(`(?i)(\bdeliberate|\bnot done\b|\bwont ?do\b)`)
	openQRe      = regexp.MustCompile(`(?i)still open`)
	multiSpaceRe = regexp.MustCompile(`\s+`)
)

type parsedTask struct {
	line      int
	open      bool
	tilde     bool
	first     string
	depth     int
	number    string
	holdLine  int
	holdLines []string
}

// ClassifyTaskLine returns the caveat marker (WONTDO, HALT, RED, BLOCKED, OPEN-Q, or "") for an open task's first line.
func ClassifyTaskLine(line string) string {
	if strings.Contains(line, "- [~]") {
		return "WONTDO"
	}
	switch {
	case haltWordRe.MatchString(line):
		return "HALT"
	case redWordRe.MatchString(line):
		return "RED"
	case holdWordRe.MatchString(line):
		return "BLOCKED"
	case wontdoWordRe.MatchString(line):
		return "WONTDO"
	case openQRe.MatchString(line):
		return "OPEN-Q"
	default:
		return ""
	}
}

// ParseTasksMarkdown parses a tasks.md file according to the SemEngine spec-queue specification.
func ParseTasksMarkdown(relPath string, data []byte) (OpenSpecChange, []HoldItem, error) {
	if !utf8.Valid(data) {
		return OpenSpecChange{}, nil, fmt.Errorf("queue unavailable: cannot read %s", relPath)
	}
	var (
		tasks      []*parsedTask
		strays     []int
		cur        *parsedTask
		belowFirst bool
		doneCount  int
		totalCount int
	)

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for idx, raw := range lines {
		lineNo := idx + 1
		trimmedLeft := strings.TrimLeft(raw, " \t")
		depth := len(raw) - len(trimmedLeft)

		if m := taskLineRe.FindStringSubmatch(raw); len(m) == 4 {
			box := m[2]
			rest := strings.TrimSpace(m[3])
			words := strings.Fields(rest)
			num := ""
			if len(words) > 0 && taskNumberRe.MatchString(words[0]) {
				num = words[0]
			}
			isOpen := box == " " || box == "~"
			totalCount++
			if box == "x" || box == "X" {
				doneCount++
			}
			cur = &parsedTask{
				line:   lineNo,
				open:   isOpen,
				tilde:  box == "~",
				first:  raw,
				depth:  depth,
				number: num,
			}
			tasks = append(tasks, cur)
		} else if cur != nil && strings.TrimSpace(raw) != "" && depth <= cur.depth {
			cur = nil
		}

		if cur == nil {
			if belowFirst && strings.Contains(raw, "Hold:") {
				strays = append(strays, lineNo)
			}
		} else if cur.holdLine > 0 {
			cur.holdLines = append(cur.holdLines, strings.TrimSpace(raw))
		} else if pos := strings.Index(raw, "Hold:"); pos >= 0 {
			cur.holdLine = lineNo
			cur.holdLines = append(cur.holdLines, strings.TrimSpace(raw[pos:]))
		}

		if strings.HasPrefix(raw, "## ") {
			belowFirst = true
		}
	}

	change := OpenSpecChange{
		CompletedTasks: doneCount,
		TotalTasks:     totalCount,
		StrayHolds:     strays,
	}
	var holds []HoldItem
	for _, t := range tasks {
		if !t.open {
			continue
		}
		firstMarker := ClassifyTaskLine(t.first)
		if t.holdLine > 0 && firstMarker == "BLOCKED" {
			firstMarker = ""
		}
		if firstMarker != "" {
			change.Caveats = append(change.Caveats, OpenSpecCaveat{
				Marker: firstMarker,
				Line:   t.line,
				Text:   cleanTaskText(t.first),
				Full:   normalizeTaskText(t.first),
			})
		}
		if t.holdLine > 0 {
			joined := normalizeHoldPrefix(strings.Join(t.holdLines, " "))
			cleaned := cleanTaskText(joined)
			change.Caveats = append(change.Caveats, OpenSpecCaveat{
				Marker: holdMarker(joined),
				Line:   t.holdLine,
				Text:   cleaned,
				Full:   normalizeTaskText(joined),
			})
			holds = append(holds, HoldItem{
				File:       relPath,
				Line:       t.holdLine,
				TaskNumber: t.number,
				Text:       cleaned,
			})
		}
	}
	return change, holds, nil
}

// holdMarker returns the caveat a hold carries: its leading HALT, RED, or OPEN-Q token (the protocol writes the
// approval hold as `Hold: OPEN-Q proposal awaits owner approval`), else BLOCKED. Only the leading token counts, so a
// hold that merely mentions a failing check stays BLOCKED.
func holdMarker(joined string) string {
	words := strings.Fields(strings.TrimPrefix(joined, "Hold:"))
	if len(words) > 0 {
		switch tok := strings.ToUpper(strings.TrimRight(words[0], ":,;")); tok {
		case "HALT", "RED", "OPEN-Q":
			return tok
		}
	}
	return "BLOCKED"
}

func normalizeHoldPrefix(s string) string {
	s = normalizeTaskText(s)
	if pos := strings.Index(s, "Hold:"); pos >= 0 {
		s = strings.TrimSpace(s[pos:])
	}
	for strings.HasPrefix(s, "Hold:") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "Hold:"))
	}
	if s == "" {
		return "Hold:"
	}
	return "Hold: " + s
}

func cleanTaskText(s string) string {
	s = normalizeTaskText(s)
	if strings.HasSuffix(s, "…") && len(strings.TrimSuffix(s, "…")) <= 104 {
		return s
	}
	if len(s) > 104 {
		i := 104
		for i > 0 && !utf8.RuneStart(s[i]) {
			i--
		}
		if s[i] != ' ' {
			if sp := strings.LastIndexByte(s[:i], ' '); sp > 0 {
				i = sp
			}
		}
		s = strings.TrimRight(s[:i], " \t") + "…"
	}
	return s
}

func normalizeTaskText(s string) string {
	s = taskLineRe.ReplaceAllString(s, "$3")
	s = strings.ReplaceAll(s, "**", "")
	return multiSpaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
}

// ScanOpenSpecQueue scans openspec/changes/*/tasks.md in deterministic order.
func ScanOpenSpecQueue(root string) ([]OpenSpecChange, error) {
	changesDir := filepath.Join(root, "openspec", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("queue unavailable: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var out []OpenSpecChange
	for _, name := range names {
		rel := filepath.ToSlash(filepath.Join("openspec", "changes", name, "tasks.md"))
		abs := filepath.Join(root, filepath.FromSlash(rel))
		var ch OpenSpecChange
		data, err := os.ReadFile(abs)
		switch {
		case err == nil:
			if ch, _, err = ParseTasksMarkdown(rel, data); err != nil {
				return nil, err
			}
		case !os.IsNotExist(err):
			return nil, fmt.Errorf("queue unavailable: cannot read %s", rel)
		}
		ch.Name = name
		propRel := filepath.ToSlash(filepath.Join("openspec", "changes", name, "proposal.md"))
		prop, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(propRel)))
		switch {
		case err == nil:
			ch.ApprovedBy, ch.proposalLines = ParseProposal(prop)
		case !os.IsNotExist(err):
			return nil, fmt.Errorf("queue unavailable: cannot read %s", propRel)
		}
		designRel := filepath.ToSlash(filepath.Join("openspec", "changes", name, "design.md"))
		design, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(designRel)))
		switch {
		case err == nil:
			// ParseProposal's physical line count applies to any Markdown file; the header is ignored.
			_, ch.designLines = ParseProposal(design)
		case !os.IsNotExist(err):
			return nil, fmt.Errorf("queue unavailable: cannot read %s", designRel)
		}
		out = append(out, ch)
	}
	return out, nil
}

// ScanHolds scans openspec/changes/*/tasks.md for active Hold: lines in open tasks, failing closed on read or parse errors.
func ScanHolds(root string) ([]HoldItem, error) {
	var all []HoldItem
	changesDir := filepath.Join(root, "openspec", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read openspec changes: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("openspec", "changes", e.Name(), "tasks.md"))
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		ch, holds, err := ParseTasksMarkdown(rel, data)
		if err != nil {
			return nil, err
		}
		if len(ch.StrayHolds) > 0 {
			return nil, fmt.Errorf("%s:%d: Hold: outside task block", rel, ch.StrayHolds[0])
		}
		all = append(all, holds...)
	}
	return all, nil
}

// FormatOpenSpecQueue formats the human-readable spec:queue output and returns whether any caveat was found.
func FormatOpenSpecQueue(changes []OpenSpecChange) (string, bool) {
	var b strings.Builder
	b.WriteString("\nopenspec queue — why each in-flight change is still open\n")
	b.WriteString("-------------------------------------------------------\n\n")
	if len(changes) == 0 {
		b.WriteString("  (queue is empty)\n\n")
		b.WriteString("-------------------------------------------------------\n")
		b.WriteString("  0 change(s) in flight.\n\n")
		return b.String(), false
	}
	foundAny := false
	inFlightCount := 0
	completedCount := 0
	for _, ch := range changes {
		fmt.Fprintf(&b, "  %-42s %d/%d\n", ch.Name, ch.CompletedTasks, ch.TotalTasks)
		if ch.IsCompletedUnarchived() {
			completedCount++
			fmt.Fprintf(&b, "      %-8s done, not archived — move openspec/changes/%s to openspec/changes/archive/<date>-%s\n\n", "done", ch.Name, ch.Name)
			continue
		}
		inFlightCount++
		if len(ch.Caveats) == 0 {
			fmt.Fprintf(&b, "      %-8s no halt/hold/deliberate marker in the open tasks\n\n", "ok")
			continue
		}
		foundAny = true
		for _, c := range ch.Caveats {
			fmt.Fprintf(&b, "      %-8s L%-5d %s\n", c.Marker, c.Line, c.Text)
		}
		b.WriteString("\n")
	}
	b.WriteString("-------------------------------------------------------\n")
	if completedCount > 0 {
		fmt.Fprintf(&b, "  %d change(s) in flight, %d completed (done, not archived).\n", inFlightCount, completedCount)
	} else {
		fmt.Fprintf(&b, "  %d change(s) in flight.\n", len(changes))
	}
	if foundAny {
		b.WriteString("  Read the flagged lines before treating any fraction above as \"almost done\".\n")
	}
	b.WriteString("\n")
	return b.String(), foundAny
}

// CheckOpenSpec validates OpenSpec specs/changes and fails on any stray Hold: outside task blocks.
// Returns (output, exitCode, error).
func CheckOpenSpec(ctx context.Context, root string) (string, int, error) {
	// 1. Validate openspec/config.yaml and openspec/specs/*/spec.md exist if openspec/ is present.
	strictSkipped := false
	openspecDir := filepath.Join(root, "openspec")
	if info, err := os.Stat(openspecDir); err == nil && info.IsDir() {
		cfgPath := filepath.Join(openspecDir, "config.yaml")
		if _, err := os.Stat(cfgPath); err != nil {
			return "", 1, fmt.Errorf("openspec/config.yaml missing")
		}
		specsDir := filepath.Join(openspecDir, "specs")
		if entries, err := os.ReadDir(specsDir); err == nil {
			for _, e := range entries {
				if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
					continue
				}
				specFile := filepath.Join(specsDir, e.Name(), "spec.md")
				data, err := os.ReadFile(specFile)
				if err != nil {
					return "", 1, fmt.Errorf("missing spec file %s: %w", specFile, err)
				}
				if !strings.Contains(string(data), "## Purpose") || !strings.Contains(string(data), "## Requirements") {
					return "", 1, fmt.Errorf("%s: spec must contain '## Purpose' and '## Requirements' sections", specFile)
				}
			}
		}
		// Run openspec CLI validation if node_modules/.bin/openspec is present; say so when it
		// is not, so a green check does not read as strict validation (#127).
		localBin := filepath.Join(root, "node_modules", ".bin", "openspec")
		strictSkipped = true
		if st, err := os.Stat(localBin); err == nil && !st.IsDir() {
			strictSkipped = false
			cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(cctx, localBin, "validate", "--all", "--strict", "--no-interactive")
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				return string(out), 1, fmt.Errorf("openspec validate failed: %w", err)
			}
		}
	}

	// 2. Scan changes/*/tasks.md for unreadable files (exit 2) or stray Hold: lines (exit 1).
	changes, err := ScanOpenSpecQueue(root)
	if err != nil {
		return "", 2, err
	}
	var out strings.Builder
	misplaced := false
	readCount := 0
	for _, ch := range changes {
		rel := filepath.ToSlash(filepath.Join("openspec", "changes", ch.Name, "tasks.md"))
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			continue
		}
		readCount++
		for _, lineNo := range ch.StrayHolds {
			fmt.Fprintf(&out, "%s:%d: Hold: outside every task; task spec:queue cannot show it. Put it in the task it stops.\n", rel, lineNo)
			misplaced = true
		}
	}
	if misplaced {
		return out.String(), 1, fmt.Errorf("stray Hold: outside task block")
	}
	for _, ch := range changes {
		if ch.proposalLines > maxProposalLines {
			fmt.Fprintf(&out, "warning: openspec/changes/%s/proposal.md has %d lines, over the one-page limit of %d; move detail to tasks.md or design.md\n", ch.Name, ch.proposalLines, maxProposalLines)
		}
		if ch.designLines > maxDesignLines {
			fmt.Fprintf(&out, "warning: openspec/changes/%s/design.md has %d lines, over the limit of %d; cut to the decisions the owner must approve\n", ch.Name, ch.designLines, maxDesignLines)
		}
	}
	if strictSkipped {
		out.WriteString("openspec: strict validation skipped (node_modules/.bin/openspec not installed; run npm ci at the repository root)\n")
	}
	fmt.Fprintf(&out, "holds: ok (%d tasks.md read)\n", readCount)
	return out.String(), 0, nil
}

// ParseSpecMarkdown parses a canonical openspec/specs/<name>/spec.md file into Purpose, Requirements, and Scenarios.
func ParseSpecMarkdown(specName, relPath string, data []byte) (OpenSpecSpec, error) {
	if !utf8.Valid(data) {
		return OpenSpecSpec{}, fmt.Errorf("cannot read %s: invalid UTF-8", relPath)
	}
	spec := OpenSpecSpec{
		Name: specName,
		Path: relPath,
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	section := ""
	var purposeLines []string
	var curReq *OpenSpecRequirement
	var curScen *OpenSpecScenario

	flushScenario := func() {
		if curReq != nil && curScen != nil {
			curReq.Scenarios = append(curReq.Scenarios, *curScen)
			curScen = nil
		}
	}
	flushReq := func() {
		flushScenario()
		if curReq != nil {
			curReq.Statement = strings.TrimSpace(curReq.Statement)
			spec.Requirements = append(spec.Requirements, *curReq)
			curReq = nil
		}
	}

	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "## ") {
			flushReq()
			heading := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			switch {
			case strings.HasPrefix(heading, "purpose"):
				section = "purpose"
			case strings.HasPrefix(heading, "requirements"):
				section = "requirements"
			default:
				section = heading
			}
			continue
		}
		if section == "purpose" {
			if trimmed != "" {
				purposeLines = append(purposeLines, trimmed)
			}
			continue
		}
		if section != "requirements" {
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			flushReq()
			reqTitle := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			reqTitle = strings.TrimSpace(strings.TrimPrefix(reqTitle, "Requirement:"))
			curReq = &OpenSpecRequirement{
				ID:   fmt.Sprintf("%s:%d", specName, len(spec.Requirements)+1),
				Name: reqTitle,
				Spec: specName,
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#### ") && curReq != nil {
			flushScenario()
			scenTitle := strings.TrimSpace(strings.TrimPrefix(trimmed, "#### "))
			scenTitle = strings.TrimSpace(strings.TrimPrefix(scenTitle, "Scenario:"))
			curScen = &OpenSpecScenario{
				Name: scenTitle,
			}
			continue
		}
		if curScen != nil {
			if step, ok := strings.CutPrefix(trimmed, "- "); ok {
				step = strings.ReplaceAll(strings.TrimSpace(step), "**", "")
				curScen.Steps = append(curScen.Steps, step)
			}
			continue
		}
		if curReq != nil && trimmed != "" {
			if curReq.Statement != "" {
				curReq.Statement += " "
			}
			curReq.Statement += trimmed
		}
	}
	flushReq()
	spec.Purpose = strings.Join(purposeLines, " ")
	return spec, nil
}

// ScanOpenSpecSpecs scans openspec/specs/*/spec.md in deterministic order.
func ScanOpenSpecSpecs(root string) ([]OpenSpecSpec, error) {
	specsDir := filepath.Join(root, "openspec", "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read openspec specs: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var out []OpenSpecSpec
	for _, name := range names {
		rel := filepath.ToSlash(filepath.Join("openspec", "specs", name, "spec.md"))
		abs := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		sp, err := ParseSpecMarkdown(name, rel, data)
		if err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, nil
}
