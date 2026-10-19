// Package testspec checks source test citations against repository specifications.
package testspec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Diagnostic identifies an invalid test citation.
type Diagnostic struct {
	File    string
	Line    int
	Message string
}
type comment struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}
type declaration struct {
	File     string    `json:"file"`
	Line     int       `json:"line"`
	Name     string    `json:"name"`
	Comments []comment `json:"comments"`
	Error    string    `json:"error"`
}

//go:embed vitest.mjs
var vitestScanner string

// Check checks supported test sources under root. Build tags do not exclude Go tests.
func Check(ctx context.Context, root string) ([]Diagnostic, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	var declarations []declaration
	var jsFiles []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".worktrees", "node_modules", "vendor", ".svelte-kit", ".enjoy-logs":
				return filepath.SkipDir
			}
			switch rel {
			case "ui/build", "internal/web/dist", "coverage", "bin", "benchmarks/runs", ".ateam":
				return filepath.SkipDir
			}
			return nil
		}
		goTest := strings.HasSuffix(rel, "_test.go")
		jsTest := false
		for _, ext := range []string{"ts", "tsx", "js", "jsx"} {
			if strings.HasSuffix(rel, ".test."+ext) || strings.HasSuffix(rel, ".spec."+ext) {
				jsTest = true
			}
		}
		if !goTest && !jsTest {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			declarations = append(declarations, declaration{File: rel, Line: 1, Error: "symlink test source is not supported"})
			return nil
		}
		if goTest {
			declarations = append(declarations, scanGo(root, rel)...)
		} else {
			jsFiles = append(jsFiles, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover tests: %w", err)
	}
	if len(jsFiles) > 0 {
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		args := append([]string{"--input-type=module", "-", root}, jsFiles...)
		cmd := exec.CommandContext(cctx, "node", args...)
		cmd.Dir = root
		cmd.Stdin = strings.NewReader(vitestScanner)
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("scan Vitest tests (requires node and ui/node_modules/typescript): %w", err)
		}
		var ds []declaration
		if err := json.Unmarshal(out, &ds); err != nil {
			return nil, fmt.Errorf("decode Vitest scan: %w", err)
		}
		declarations = append(declarations, ds...)
	}
	var diagnostics []Diagnostic
	for _, d := range declarations {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if d.Error != "" {
			diagnostics = append(diagnostics, Diagnostic{d.File, d.Line, d.Error})
			continue
		}
		found := false
		for _, c := range d.Comments {
			text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if !strings.HasPrefix(text, "spec:") {
				continue
			}
			found = true
			ref := strings.TrimSpace(strings.TrimPrefix(text, "spec:"))
			if err := resolve(root, ref); err != nil {
				diagnostics = append(diagnostics, Diagnostic{d.File, c.Line, d.Name + ": " + err.Error()})
			}
		}
		if !found {
			diagnostics = append(diagnostics, Diagnostic{d.File, d.Line, d.Name + ": missing spec citation"})
		}
	}
	sort.Slice(diagnostics, func(i, j int) bool {
		a, b := diagnostics[i], diagnostics[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Message < b.Message
	})
	return diagnostics, nil
}

func scanGo(root, rel string) []declaration {
	fset := token.NewFileSet()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return []declaration{{File: rel, Line: 1, Error: "read Go: " + err.Error()}}
	}
	file, err := parser.ParseFile(fset, rel, data, parser.ParseComments)
	if err != nil {
		return []declaration{{File: rel, Line: 1, Error: "parse Go: " + err.Error()}}
	}
	examples := map[string]bool{}
	for _, example := range doc.Examples(file) {
		if example.Output != "" || example.EmptyOutput {
			examples["Example"+example.Name] = true
		}
	}
	var out []declaration
	for _, node := range file.Decls {
		fn, ok := node.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Body == nil {
			continue
		}
		name := fn.Name.Name
		selected := (testName(name, "Test") && name != "TestMain") || testName(name, "Fuzz")
		if testName(name, "Example") {
			selected = examples[name]
		}
		if !selected {
			continue
		}
		d := declaration{File: rel, Line: fset.Position(fn.Pos()).Line, Name: name}
		if fn.Doc != nil && fset.Position(fn.Doc.End()).Line+1 == d.Line {
			for _, c := range fn.Doc.List {
				d.Comments = append(d.Comments, comment{fset.Position(c.Pos()).Line, c.Text})
			}
		}
		out = append(out, d)
	}
	return out
}

func testName(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(name, prefix)
	if suffix == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(suffix)
	return !unicode.IsLower(r)
}

func resolve(root, ref string) error {
	path, heading, ok := strings.Cut(ref, "#")
	if !ok || !strings.HasPrefix(heading, "Requirement: ") || strings.TrimSpace(strings.TrimPrefix(heading, "Requirement: ")) == "" {
		return fmt.Errorf("malformed spec citation")
	}
	if filepath.IsAbs(path) || strings.Contains(path, "\\") || filepath.ToSlash(filepath.Clean(path)) != path || strings.Contains(path, ":") || strings.HasPrefix(path, "../") {
		return fmt.Errorf("unsafe spec path")
	}
	parts := strings.Split(path, "/")
	canonical := len(parts) == 4 && parts[0] == "openspec" && parts[1] == "specs" && parts[3] == "spec.md"
	inflight := len(parts) == 6 && parts[0] == "openspec" && parts[1] == "changes" && parts[2] != "archive" && parts[3] == "specs" && parts[5] == "spec.md"
	if !canonical && !inflight {
		return fmt.Errorf("expected canonical or in-flight spec.md")
	}
	full, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return fmt.Errorf("resolve spec: %w", err)
	}
	relative, err := filepath.Rel(root, full)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("spec path escapes repository")
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("spec is not UTF-8")
	}
	count := 0
	var fence byte
	fenceLength := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimLeft(line, " ")
		indentation := len(line) - len(trimmed)
		if indentation <= 3 && len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
			n := 0
			for n < len(trimmed) && trimmed[n] == trimmed[0] {
				n++
			}
			if n >= 3 {
				if fence == 0 {
					fence = trimmed[0]
					fenceLength = n
					continue
				}
				if trimmed[0] == fence && n >= fenceLength && strings.TrimSpace(trimmed[n:]) == "" {
					fence = 0
					fenceLength = 0
					continue
				}
			}
		}
		if fence == 0 && line == "### "+heading {
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("requirement resolves %d times: %s", count, ref)
	}
	return nil
}

// Format renders diagnostics in their deterministic check order.
func Format(ds []Diagnostic) string {
	var out strings.Builder
	for _, d := range ds {
		fmt.Fprintf(&out, "%s:%d: %s\n", d.File, d.Line, d.Message)
	}
	return out.String()
}
