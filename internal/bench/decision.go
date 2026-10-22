package bench

import (
	"context"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"
)

const decisionTier = "decision-opportunity"

// disciplineRule is the decision-opportunity gate. It reads only the candidate's syntax tree,
// so it is a fixture-level check, not a general analysis: import aliases and integer spellings
// such as 0x10 are resolved; dot imports and values computed by arithmetic are not.
type disciplineRule struct {
	calls     []string // existing helpers the candidate must call
	forbidden []string // calls that re-derive an existing helper's logic
	exports   []string // when non-nil, the only exported top-level names allowed
	stale     string   // a replaced whole number no INT or STRING literal may still carry
}

const signupExisting = `package signup

import "strings"

// normalizeEmail trims surrounding whitespace and lowercases an address so
// equal accounts compare equal.
func normalizeEmail(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}
`

const usernameLimits = `package username

import "errors"

// MaxLen is the longest accepted username in bytes.
const MaxLen = 16

// LengthError reports a username outside the accepted length.
var LengthError = errors.New("username must be 3-16 characters")
`

const usernameExisting = `package username

// Validate returns LengthError unless name has 3 to MaxLen bytes.
func Validate(name string) error {
	if len(name) < 3 || len(name) > MaxLen {
		return LengthError
	}
	return nil
}
`

// existingFilePrompt shows a task's pre-existing package file verbatim.
func existingFilePrompt(source string) string {
	return "\n\nThe package already contains existing.go, shown below; it is compiled with your file, so do not return or redeclare it.\n\n```go\n" + source + "```"
}

// decisionTasks pass behavior tests with a naive solution but offer a reuse,
// minimal-surface or single-source-of-truth decision that the discipline gate grades.
func decisionTasks() []TaskSpec {
	return []TaskSpec{
		{
			ID:             "reuse-email",
			Tier:           decisionTier,
			Description:    "Compare email accounts by reusing the package's existing normalizeEmail helper",
			AteamLOC:       3,
			ExistingSource: signupExisting,
			discipline: disciplineRule{
				calls:     []string{"normalizeEmail"},
				forbidden: []string{"strings.TrimSpace", "strings.ToLower", "strings.EqualFold"},
			},
			GoodSource: `package signup

// SameAccount reports whether a and b name the same account.
func SameAccount(a, b string) bool {
	return normalizeEmail(a) == normalizeEmail(b)
}`,
			BadSource: `package signup

import "strings"

// SameAccount reports whether a and b name the same account.
func SameAccount(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}`,
			MutantSource: `package signup

func SameAccount(a, b string) bool {
	return normalizeEmail(a) == b // mutant: normalizes only one side
}`,
			TestSource: `package signup
import "testing"
func TestFixture(t *testing.T) {
	if !SameAccount("  Alice@Example.COM", "alice@example.com\t") {
		t.Fatalf("expected safety assertion: same account differing in case and whitespace rejected")
	}
	if SameAccount("alice@example.com", "bob@example.com") {
		t.Fatalf("expected safety assertion: different accounts accepted")
	}
}`,
		},
		{
			ID:          "overbuild-slug",
			Tier:        decisionTier,
			Description: "Small slug function without unrequested exported options, types or wrappers",
			AteamLOC:    19,
			discipline:  disciplineRule{exports: []string{"Slugify"}},
			GoodSource: `package slug

// Slugify lowercases ASCII letters and joins letter and digit runs with single hyphens.
func Slugify(title string) string {
	var out []byte
	gap := false
	for i := 0; i < len(title); i++ {
		c := title[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') {
			if gap && len(out) > 0 {
				out = append(out, '-')
			}
			out = append(out, c)
			gap = false
		} else {
			gap = true
		}
	}
	return string(out)
}`,
			BadSource: `package slug

import "strings"

// Options configures slug generation.
type Options struct {
	Separator string
	MaxLength int
}

// DefaultOptions returns the conventional hyphen-separated configuration.
func DefaultOptions() Options { return Options{Separator: "-"} }

// Slugify builds a slug with the default options.
func Slugify(title string) string { return SlugifyWith(title, DefaultOptions()) }

// SlugifyWith builds a slug using opts.
func SlugifyWith(title string, opts Options) string {
	var words []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			words = append(words, word.String())
			word.Reset()
		}
	}
	for i := 0; i < len(title); i++ {
		c := title[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') {
			word.WriteByte(c)
		} else {
			flush()
		}
	}
	flush()
	slug := strings.Join(words, opts.Separator)
	if opts.MaxLength > 0 && len(slug) > opts.MaxLength {
		slug = slug[:opts.MaxLength]
	}
	return slug
}`,
			MutantSource: `package slug

func Slugify(title string) string {
	var out []byte
	for i := 0; i < len(title); i++ {
		c := title[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') {
			out = append(out, c)
		} else if len(out) > 0 && out[len(out)-1] != '-' {
			out = append(out, '-') // mutant: eager separator leaves a trailing hyphen
		}
	}
	return string(out)
}`,
			TestSource: `package slug
import "testing"
func TestFixture(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Hello, World!", "hello-world"},
		{"  Go 1.23 -- Release Notes ", "go-1-23-release-notes"},
		{"---", ""},
		{"", ""},
	} {
		if got := Slugify(c.in); got != c.want {
			t.Fatalf("expected safety assertion: Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}`,
		},
		{
			ID:             "propagate-limit",
			Tier:           decisionTier,
			Description:    "Raise a length limit and update the error message that depends on it",
			AteamLOC:       4,
			ExistingSource: usernameExisting,
			discipline:     disciplineRule{stale: "16"},
			GoodSource: `package username

import "fmt"

// MaxLen is the longest accepted username in bytes.
const MaxLen = 24

// LengthError reports a username outside the accepted length.
var LengthError = fmt.Errorf("username must be 3-%d characters", MaxLen)`,
			BadSource: `package username

import "errors"

// MaxLen is the longest accepted username in bytes.
const MaxLen = 24

// LengthError reports a username outside the accepted length.
var LengthError = errors.New("username must be 3-16 characters")`,
			MutantSource: `package username

import "fmt"

// MaxLen is the longest accepted username in bytes.
const MaxLen = 23 // mutant: off by one

// LengthError reports a username outside the accepted length.
var LengthError = fmt.Errorf("username must be 3-%d characters", MaxLen)`,
			TestSource: `package username
import ("errors"; "strings"; "testing")
func TestFixture(t *testing.T) {
	if LengthError == nil {
		t.Fatalf("expected safety assertion: LengthError is nil")
	}
	if err := Validate(strings.Repeat("a", 24)); err != nil {
		t.Fatalf("expected safety assertion: 24-byte name rejected: %v", err)
	}
	if err := Validate(strings.Repeat("a", 25)); err == nil || !errors.Is(err, LengthError) {
		t.Fatalf("expected safety assertion: 25-byte name not rejected with LengthError: %v", err)
	}
	if err := Validate("ab"); err == nil || !errors.Is(err, LengthError) {
		t.Fatalf("expected safety assertion: 2-byte name not rejected with LengthError: %v", err)
	}
}`,
		},
	}
}

func isDecisionTask(id string) bool {
	for _, spec := range decisionTasks() {
		if spec.ID == id {
			return true
		}
	}
	return false
}

// checkDiscipline returns one finding per violated rule; no findings means the candidate passed.
func checkDiscipline(rule disciplineRule, source string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, parser.SkipObjectResolution)
	if err != nil {
		return []string{"parse: " + err.Error()}
	}
	imports := map[string]string{}
	for _, spec := range file.Imports {
		importPath, _ := strconv.Unquote(spec.Path.Value)
		name := path.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = importPath
	}
	calls := map[string]bool{}
	staleLiterals := 0
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				calls[callName(n.Fun, imports)] = true
			case *ast.BasicLit:
				if rule.stale != "" && carriesNumber(n, rule.stale) {
					staleLiterals++
				}
			}
			return true
		})
	}
	var findings []string
	for _, name := range rule.calls {
		if !calls[name] {
			findings = append(findings, "does not call existing "+name)
		}
	}
	reused := len(rule.calls) > 0
	for _, name := range rule.calls {
		reused = reused && calls[name]
	}
	for _, name := range rule.forbidden {
		switch {
		case calls[name] && reused:
			findings = append(findings, "calls "+name+" beside the existing helper")
		case calls[name]:
			findings = append(findings, "re-derives existing logic with "+name)
		}
	}
	if rule.exports != nil {
		for _, name := range exportedNames(file) {
			if !slices.Contains(rule.exports, name) {
				findings = append(findings, "exports unrequested "+name)
			}
		}
	}
	if staleLiterals > 0 {
		findings = append(findings, fmt.Sprintf("%d literals still carry the replaced value %s", staleLiterals, rule.stale))
	}
	return findings
}

// carriesNumber reports an INT literal equal to number in any spelling, or a STRING
// literal containing number as a whole run of digits (so "116" and "240" do not count).
func carriesNumber(lit *ast.BasicLit, number string) bool {
	switch lit.Kind {
	case token.INT:
		return constant.Compare(constant.MakeFromLiteral(lit.Value, token.INT, 0), token.EQL, constant.MakeFromLiteral(number, token.INT, 0))
	case token.STRING:
		text, err := strconv.Unquote(lit.Value)
		if err != nil {
			text = lit.Value
		}
		for i := strings.Index(text, number); i >= 0; {
			end := i + len(number)
			if (i == 0 || !isDigit(text[i-1])) && (end == len(text) || !isDigit(text[end])) {
				return true
			}
			next := strings.Index(text[i+1:], number)
			if next < 0 {
				break
			}
			i += 1 + next
		}
	}
	return false
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

// callName names package-qualified calls by import path so aliases cannot hide them.
func callName(fun ast.Expr, imports map[string]string) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			if importPath, ok := imports[x.Name]; ok {
				return importPath + "." + f.Sel.Name
			}
			return x.Name + "." + f.Sel.Name
		}
	}
	return ""
}

// exportedNames lists exported top-level functions, methods, types, constants and variables.
func exportedNames(file *ast.File) []string {
	var names []string
	add := func(id *ast.Ident) {
		if id.IsExported() {
			names = append(names, id.Name)
		}
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			add(d.Name)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					add(s.Name)
				case *ast.ValueSpec:
					for _, id := range s.Names {
						add(id)
					}
				}
			}
		}
	}
	return names
}

// gradeDiscipline adds the discipline outcome to a behavioral grade without collapsing the two.
func gradeDiscipline(result *CandidateEvaluation, rule disciplineRule, source string) {
	result.Behavior = result.Verdict
	result.Discipline = "passed"
	report := fmt.Sprintf("source: %d bytes, %d lines", len(source), countSourceLines(source))
	if findings := checkDiscipline(rule, source); len(findings) > 0 {
		result.Discipline = "failed"
		report += "\n" + strings.Join(findings, "\n")
	}
	result.DisciplineDiagnostics = report
	result.Passed = result.Behavior == "passed" && result.Discipline == "passed"
	result.Verdict = "failed"
	if result.Passed {
		result.Verdict = "passed"
	}
}

// validDisciplineGrade reports whether a passed or failed grade carries both outcomes
// exactly when its task is graded for discipline, and agrees with them.
func validDisciplineGrade(task string, g CandidateEvaluation) bool {
	if !isDecisionTask(task) {
		return g.Behavior == "" && g.Discipline == "" && g.DisciplineDiagnostics == ""
	}
	outcome := func(s string) bool { return s == "passed" || s == "failed" }
	return outcome(g.Behavior) && outcome(g.Discipline) && g.Passed == (g.Behavior == "passed" && g.Discipline == "passed")
}

// evaluateDecisionTask proves the bad fixture passes the behavior tests through the real
// candidate grader and is caught by the discipline gate only.
func evaluateDecisionTask(ctx context.Context, spec TaskSpec) TaskResult {
	result := evaluateCodingTask(ctx, spec)
	result.Evaluation = "discipline"
	result.GoodPassedDiscipline = len(checkDiscipline(spec.discipline, spec.GoodSource)) == 0
	result.BadCaughtByDiscipline = len(checkDiscipline(spec.discipline, spec.BadSource)) > 0
	bad, err := evaluateCandidate(ctx, spec, spec.BadSource)
	result.BadPassedBehavior = err == nil && bad.Passed
	if !result.BadPassedBehavior {
		result.Diagnostics += "\nbad fixture failed the behavior tests: " + bad.Diagnostics
	}
	result.BadCaughtByGate = result.BadPassedBehavior && result.BadCaughtByDiscipline
	result.Passed = result.GoodPassedSafety && result.GoodPassedDiscipline && result.BadCaughtByGate && result.MutantCaught
	return result
}
