package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// flagSpec maps a flag name to whether it is a bool flag, which takes no value.
type flagSpec map[string]bool

// cliFlags is the flag inventory read from source: each FlagSet's flags, and for a set
// parsed from args[N:] rather than args, that N.
type cliFlags struct {
	sets      map[string]flagSpec
	parseFrom map[string]int
}

// flagSetAliases names FlagSets whose name is not the command path that reaches them.
var flagSetAliases = map[string]string{"spec-check": "state spec-check"}

// repoRoot is the repository root, found from this file so a test that changes directory cannot move it.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// flagSets reads every flag.NewFlagSet in the given directories' non-test Go files and the
// flags defined on that set in the same function, so the check never runs a command.
func flagSets(t *testing.T, dirs ...string) cliFlags {
	t.Helper()
	inv := cliFlags{sets: map[string]flagSpec{}, parseFrom: map[string]int{}}
	fset := token.NewFileSet()
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range f.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
					collectFlagSets(fn.Body, inv)
				}
			}
		}
	}
	return inv
}

// collectFlagSets records, for each `v := flag.NewFlagSet("<name>", …)` in body, the flags
// that body defines through v. Receivers are matched by their resolved declaration, not
// their spelling, so two `fs :=` in sibling switch cases stay separate sets.
func collectFlagSets(body *ast.BlockStmt, inv cliFlags) {
	type set struct {
		name string
		spec flagSpec
	}
	byObj := map[*ast.Object]set{}
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok || !isSelector(call.Fun, "flag", "NewFlagSet") || len(call.Args) == 0 {
			return true
		}
		name, ok := stringLit(call.Args[0])
		ident, isIdent := assign.Lhs[0].(*ast.Ident)
		if !ok || !isIdent || ident.Obj == nil {
			return true
		}
		if alias, found := flagSetAliases[name]; found {
			name = alias
		}
		spec := flagSpec{}
		inv.sets[name] = spec
		byObj[ident.Obj] = set{name, spec}
		return true
	})
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok || recv.Obj == nil {
			return true
		}
		s, known := byObj[recv.Obj]
		if !known {
			return true
		}
		if sel.Sel.Name == "Parse" {
			if len(call.Args) == 1 {
				if slice, ok := call.Args[0].(*ast.SliceExpr); ok && slice.Low != nil {
					if lit, ok := slice.Low.(*ast.BasicLit); ok && lit.Kind == token.INT {
						n, _ := strconv.Atoi(lit.Value)
						inv.parseFrom[s.name] = n
					}
				}
			}
			return true
		}
		spec := s.spec
		// The flag name is the first argument of fs.Bool/String/Int/Duration/Func and the
		// second of fs.BoolVar/StringVar/Var and the like.
		idx := 0
		if strings.HasSuffix(sel.Sel.Name, "Var") {
			idx = 1
		}
		if len(call.Args) <= idx {
			return true
		}
		if flagName, ok := stringLit(call.Args[idx]); ok {
			spec[flagName] = sel.Sel.Name == "Bool" || sel.Sel.Name == "BoolVar"
		}
		return true
	})
}

func isSelector(e ast.Expr, pkg, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == pkg && sel.Sel.Name == name
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}

// commandTokens splits a quoted command on spaces, keeping a <placeholder with spaces> whole.
func commandTokens(cmd string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	for _, r := range cmd {
		switch {
		case r == '<':
			depth++
		case r == '>' && depth > 0:
			depth--
		}
		if r == ' ' && depth == 0 {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// checkQuotedCommand reports why Go flag parsing would reject cmd (which starts with
// "ateam "), or "" when it parses. A span ending on a value flag is a reference to that flag.
func checkQuotedCommand(cmd string, inv cliFlags) string {
	toks := commandTokens(strings.TrimPrefix(cmd, "ateam "))
	var spec flagSpec
	path, lead := 0, 0
	for k := min(2, len(toks)); k >= 1; k-- {
		name := strings.Join(toks[:k], " ")
		if s, ok := inv.sets[name]; ok {
			spec, path = s, k
			// A set parsed from args[N:] sees args starting at the last path token (bench
			// run/compare/export take args[0] = subcommand, args[1] = artifact), so N-1
			// positionals come before its flags.
			if n := inv.parseFrom[name]; n > 0 {
				lead = n - 1
			}
			break
		}
	}
	if path+lead > len(toks) {
		return ""
	}
	path += lead
	rest := toks[path:]
	if spec == nil {
		// shortcut: a path without a FlagSet is not checked against run's dispatch, so a misspelled
		// subcommand passes; upgrade when one ships.
		spec = flagSpec{}
		if len(toks) > 0 && strings.HasPrefix(toks[0], "-") {
			spec = topLevelFlags
		}
		for path < len(toks) && !strings.HasPrefix(toks[path], "-") {
			path++
		}
		rest = toks[path:]
	}
	positional := ""
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			if positional == "" {
				positional = tok
			}
			continue
		}
		if tok == "--" {
			return ""
		}
		if positional != "" {
			return fmt.Sprintf("flag %s after positional %s is never parsed", tok, positional)
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(tok, "-"), "=")
		isBool, known := spec[name]
		if !known {
			return fmt.Sprintf("flag %s is not defined", tok)
		}
		if isBool || hasValue {
			continue
		}
		if i+1 == len(rest) {
			return ""
		}
		if strings.HasPrefix(rest[i+1], "-") {
			return fmt.Sprintf("flag %s takes a value, got %s", tok, rest[i+1])
		}
		i++
	}
	return ""
}

// topLevelFlags are the bare `ateam -h|--help|-v|--version` forms run accepts before any command.
var topLevelFlags = flagSpec{"h": true, "help": true, "v": true, "version": true}

var quotedCommand = regexp.MustCompile("`(ateam [^`]+)`")

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestQuotedCommandsParse(t *testing.T) {
	root := repoRoot(t)
	inv := flagSets(t, filepath.Join(root, "cmd", "ateam"), filepath.Join(root, "internal", "mutate"))

	files := []string{filepath.Join(root, "AGENTS.md"), filepath.Join(root, ".agents", "protocol.md")}
	for _, pattern := range []string{".agents/contracts/*.md", ".agents/skills/*/SKILL.md"} {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	checked := 0
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, path)
		for _, m := range quotedCommand.FindAllStringSubmatch(string(body), -1) {
			checked++
			if why := checkQuotedCommand(m[1], inv); why != "" {
				t.Errorf("%s: `%s`: %s", rel, m[1], why)
			}
		}
	}
	if checked < 20 {
		t.Fatalf("checked %d quoted commands, want at least 20: the extraction is broken", checked)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestCheckQuotedCommand(t *testing.T) {
	inv := flagSets(t, filepath.Join(repoRoot(t), "cmd", "ateam"), filepath.Join(repoRoot(t), "internal", "mutate"))
	sets := inv.sets
	run, compare := sets["bench run"], sets["bench compare"]
	if !run["dry-run"] || run["output"] || len(run) != 2 {
		t.Fatalf("bench run flags = %v, want exactly bool dry-run and value output", run)
	}
	if _, ok := compare["dry-run"]; ok || !compare["json"] {
		t.Fatalf("bench compare flags = %v, want bool json and no dry-run", compare)
	}
	wantFrom := map[string]int{"bench run": 2, "bench compare": 2, "bench export": 2}
	if fmt.Sprint(inv.parseFrom) != fmt.Sprint(wantFrom) {
		t.Fatalf("parseFrom = %v, want %v: a new args[N:] parse needs its lead checked", inv.parseFrom, wantFrom)
	}
	if spec := sets["review request"]; !spec["dry-run"] || spec["return-to"] {
		t.Fatalf("review request flags = %v, want bool dry-run and value return-to", spec)
	}
	if !sets["doctor"]["check"] {
		t.Fatalf("doctor flags = %v, want bool check", sets["doctor"])
	}
	if _, ok := sets["init"]["check"]; ok {
		t.Fatal("init defines check; the --check rejection case below no longer tests anything")
	}

	for _, tc := range []struct {
		cmd, want string
	}{
		{"ateam review request <pr> --return-to <your id>", "flag --return-to after positional <pr> is never parsed"},
		{"ateam review request --dry-run 425 --return-to probe", "flag --return-to after positional 425 is never parsed"},
		{"ateam init --check", "flag --check is not defined"},
		{"ateam review request --reviewer-session --return-to <id> <n>", "flag --reviewer-session takes a value, got --return-to"},
		{"ateam state next --bogus", "flag --bogus is not defined"},
		{"ateam review request --return-to <your id> <pr>", ""},
		{"ateam review request --reviewer-session", ""},
		{"ateam review notify --record <record file> --return-to=<id> <n>", ""},
		{"ateam doctor --check", ""},
		{"ateam state mode <on|off>", ""},
		{"ateam --help", ""},
		{"ateam bench compare <run.json> --json", ""},
		{"ateam bench run <experiment.json> --dry-run --output <dir>", ""},
		{"ateam bench export <run.json> --check <dir>", ""},
		{"ateam bench compare <run.json> --dry-run", "flag --dry-run is not defined"},
		{"ateam bench compare <run.json> extra --json", "flag --json after positional extra is never parsed"},
	} {
		if got := checkQuotedCommand(tc.cmd, inv); got != tc.want {
			t.Errorf("checkQuotedCommand(%q) = %q, want %q", tc.cmd, got, tc.want)
		}
	}
}
