package state

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ShortcutItem represents an inline // shortcut: <limit>, <when to upgrade> marker in source code.
type ShortcutItem struct {
	File           string `json:"file"`
	Line           int    `json:"line"`
	Limit          string `json:"limit"`
	RevisitTrigger string `json:"revisit_trigger"`
	Raw            string `json:"raw"`
}

var shortcutPattern = regexp.MustCompile(`(?i)^\s*(?://|#|--)\s*shortcut:\s*(.+)$`)

var skippedDirs = map[string]bool{
	".git":              true,
	".ateam":            true,
	".agents":           true,
	".claude":           true,
	".codex":            true,
	".gemini":           true,
	".worktrees":        true,
	".enjoy-logs":       true,
	".svelte-kit":       true,
	"node_modules":      true,
	"vendor":            true,
	"dist":              true,
	"build":             true,
	"bin":               true,
	"coverage":          true,
	"playwright-report": true,
	"test-results":      true,
	"openspec":          true,
	"docs":              true,
	"testdata":          true,
	"benchmarks":        true,
}

var sourceExts = map[string]bool{
	".go":     true,
	".ts":     true,
	".js":     true,
	".svelte": true,
	".sh":     true,
	".yml":    true,
	".yaml":   true,
}

// ScanShortcuts walks root and extracts all // shortcut: <limit>, <when to upgrade> comments.
func ScanShortcuts(root string) ([]ShortcutItem, error) {
	var items []ShortcutItem
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !sourceExts[filepath.Ext(d.Name())] {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") || strings.HasSuffix(d.Name(), ".test.ts") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		found, err := scanFileShortcuts(path, rel)
		if err == nil {
			items = append(items, found...)
		}
		return nil
	})
	return items, err
}

func scanFileShortcuts(absPath, relPath string) ([]ShortcutItem, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	if !containsShortcutFold(data) {
		return nil, nil
	}

	var out []ShortcutItem
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if !containsShortcutFold(scanner.Bytes()) {
			continue
		}
		line := scanner.Text()
		m := shortcutPattern.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		body := strings.TrimSpace(m[1])
		// Ignore template/regex definitions that contain literal "<the limit>" or "<limit>".
		if strings.Contains(body, "<the limit>") || strings.Contains(body, "<limit>") {
			continue
		}
		limit, trigger := body, "not specified"
		if parts := strings.SplitN(body, ",", 2); len(parts) == 2 {
			limit = strings.TrimSpace(parts[0])
			trigger = strings.TrimSpace(parts[1])
		}
		out = append(out, ShortcutItem{
			File:           relPath,
			Line:           lineNo,
			Limit:          limit,
			RevisitTrigger: trigger,
			Raw:            body,
		})
	}
	return out, scanner.Err()
}

// containsShortcutFold reports whether data contains "shortcut:" in any ASCII case without copying data:
// session-start scans every source file, and lowering each one cost a full allocation per file (#207).
func containsShortcutFold(data []byte) bool {
	const word = "shortcut"
	for i := 0; ; {
		j := bytes.IndexByte(data[i:], ':')
		if j < 0 {
			return false
		}
		j += i
		if j >= len(word) && bytes.EqualFold(data[j-len(word):j], []byte(word)) {
			return true
		}
		i = j + 1
	}
}
