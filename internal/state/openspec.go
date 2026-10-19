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
}

var (
	taskLineRe   = regexp.MustCompile(`^(\s*)- \[(.)\](.*)$`)
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
			if len(words) > 0 {
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
			joined := strings.TrimSpace(strings.Join(t.holdLines, " "))
			if t.number != "" {
				joined = t.number + " " + joined
			}
			cleaned := cleanTaskText(joined)
			change.Caveats = append(change.Caveats, OpenSpecCaveat{
				Marker: "BLOCKED",
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

func cleanTaskText(s string) string {
	s = normalizeTaskText(s)
	if len(s) > 104 {
		i := 104
		for i > 0 && !utf8.RuneStart(s[i]) {
			i--
		}
		s = s[:i]
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
		data, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				out = append(out, OpenSpecChange{Name: name})
				continue
			}
			return nil, fmt.Errorf("queue unavailable: cannot read %s", rel)
		}
		ch, _, err := ParseTasksMarkdown(rel, data)
		if err != nil {
			return nil, err
		}
		ch.Name = name
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
	for _, ch := range changes {
		fmt.Fprintf(&b, "  %-42s %d/%d\n", ch.Name, ch.CompletedTasks, ch.TotalTasks)
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
	fmt.Fprintf(&b, "  %d change(s) in flight.\n", len(changes))
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
		// Run openspec CLI validation if node_modules/.bin/openspec is present.
		localBin := filepath.Join(root, "node_modules", ".bin", "openspec")
		if st, err := os.Stat(localBin); err == nil && !st.IsDir() {
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
	fmt.Fprintf(&out, "holds: ok (%d tasks.md read)\n", readCount)
	return out.String(), 0, nil
}
