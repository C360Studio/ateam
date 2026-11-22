package testspec

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const rustDangling = "// spec: openspec/specs/demo/spec.md#Requirement: Missing\n"

// rustSuite holds the four #384 cases: attributed #[test], attributed #[tokio::test], uncited, dangling.
const rustSuite = citation + `#[test]
fn cited() {}

` + citation + `#[tokio::test(flavor = "multi_thread")]
async fn cited_async() {
    let s = "}";
}

#[test]
fn uncited() {}

` + rustDangling + `#[test]
fn dangling() {}
`

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for path, text := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// spec: test-attribution / Rust test declarations
func TestRustInlineAndIntegrationTests(t *testing.T) {
	root := fixture(t, "package p\n", "### Requirement: Works\n")
	inline := "pub fn add() {}\n\n#[cfg(test)]\nmod tests {\n    use super::*;\n\n" + rustSuite + "}\n"
	writeTree(t, root, map[string]string{
		"crates/core/src/lib.rs":               inline,
		"crates/core/tests/wire.rs":            rustSuite,
		"firmware/pilot/src/main.rs":           "fn main() {}\n#[test]\nfn outside_workspace() {}\n",
		"target/debug/build/gen.rs":            "#[test]\nfn generated() {}\n",
		"firmware/pilot/target/release/gen.rs": "#[test]\nfn nested_generated() {}\n",
		"crates/core/src/fixtures.rs":          "const SRC: &str = r#\"say \"\n#[test]\nfn fake() {}\n\"#;\nconst ESC: &str = r\"\\\";\n// #[test] fn also_fake() {}\n/* #[test] /* nested */ fn fake_block() {} */\n",
		"crates/core/src/odd.rs":               "#[ /* note */ test]\nfn odd_comment() {}\n#[r#test]\nfn odd_raw() {}\n#[tokio:: /* x */ r#test]\nasync fn odd_path() {}\n",
		"crates/core/src/lifetimes.rs":         "fn f<'a>(x: &'a str) -> char { let _ = '\\''; let _ = b'}'; 'x' }\n",
	})
	ds, err := Check(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"crates/core/src/lib.rs:18: tests::uncited: missing spec citation",
		"crates/core/src/lib.rs:20: tests::dangling: requirement resolves 0 times: openspec/specs/demo/spec.md#Requirement: Missing",
		"crates/core/src/odd.rs:2: odd_comment: missing spec citation",
		"crates/core/src/odd.rs:4: odd_raw: missing spec citation",
		"crates/core/src/odd.rs:6: odd_path: missing spec citation",
		"crates/core/tests/wire.rs:12: uncited: missing spec citation",
		"crates/core/tests/wire.rs:14: dangling: requirement resolves 0 times: openspec/specs/demo/spec.md#Requirement: Missing",
		"firmware/pilot/src/main.rs:3: outside_workspace: missing spec citation",
	}, "\n") + "\n"
	if got := Format(ds); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// spec: test-attribution / Rust test declarations
func TestRustSelectionAndAttachment(t *testing.T) {
	cases := []struct {
		name, src string
		want      []declaration
	}{
		{"stacked attributes and doc comment", citation + "#[test]\n/// Doc line.\n#[ignore]\n#[should_panic(expected = \"]\")]\npub(crate) fn stacked() {\n}\n",
			[]declaration{{File: "x.rs", Line: 6, End: 7, Start: 2, Name: "stacked", Comments: []comment{{1, strings.TrimSuffix(citation, "\n")}, {3, "/// Doc line."}}}}},
		{"citation between attributes", "#[test]\n" + citation + "fn between() {}\n",
			[]declaration{{File: "x.rs", Line: 3, End: 3, Start: 1, Name: "between", Comments: []comment{{2, strings.TrimSuffix(citation, "\n")}}}}},
		{"blank line detaches", citation + "\n#[test]\nfn detached() {}\n",
			[]declaration{{File: "x.rs", Line: 4, End: 4, Start: 3, Name: "detached"}}},
		{"trailing code comment does not attach", "let x = 1; " + citation + "#[test]\nfn trailing() {}\n",
			[]declaration{{File: "x.rs", Line: 3, End: 3, Start: 2, Name: "trailing"}}},
		{"runtime path ending in test", "#[async_std::test]\nasync fn rt() {}\n#[sqlx :: test]\nasync unsafe fn db() {}\n",
			[]declaration{{File: "x.rs", Line: 2, End: 2, Start: 1, Name: "rt"}, {File: "x.rs", Line: 4, End: 4, Start: 3, Name: "db"}}},
		{"non-test attributes", "#[testing]\nfn a() {}\n#[test_case(1)]\nfn b() {}\n#[cfg(test)]\nmod m { fn c() {} }\n#[cfg_attr(test, test)]\nfn d() {}\n", nil},
		{"raw identifier and nested mods", "mod a {\n    mod b {\n        #[test]\n        fn r#match() {\n        }\n    }\n}\n",
			[]declaration{{File: "x.rs", Line: 4, End: 5, Start: 3, Name: "a::b::match"}}},
		{"inner attribute is not a test", "#![test]\nfn notatest() {}\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := scanRustSource("x.rs", []byte(tc.src))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got  %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

// spec: test-attribution / Rust test declarations
func TestRustFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct{ src, want string }{
		"macro-generated name":          {"macro_rules! t {\n    ($n:ident) => {\n        #[test]\n        fn $n() {}\n    };\n}\n", "macro-generated tests are unsupported"},
		"attribute on non-fn":           {"#[test]\nstruct S;\n", "test attribute is not on a fn item"},
		"no body":                       {"#[test]\nfn decl();\n", "decl: test fn has no body"},
		"unterminated string":           {"#[test]\nfn s() { let _ = \"open; }\n", "parse Rust: unterminated string"},
		"unterminated raw":              {"const X: &str = r##\"x\"#;\n", "parse Rust: unterminated raw string"},
		"unclosed block":                {"/* /* */\n", "parse Rust: unterminated block comment"},
		"unbalanced braces":             {"fn a() {}\n}\n", "parse Rust: unbalanced closing brace"},
		"unclosed brace":                {"mod a {\n", "parse Rust: unclosed brace"},
		"attribute at EOF":              {"#[test]\n", "test attribute is not on a fn item"},
		"attribute then comment at EOF": {"fn a() {}\n#[test]\n// trailing\n", "test attribute is not on a fn item"},
	} {
		t.Run(name, func(t *testing.T) {
			got := scanRustSource("x.rs", []byte(tc.src))
			if len(got) != 1 || !strings.Contains(got[0].Error, tc.want) {
				t.Fatalf("got %+v, want one error containing %q", got, tc.want)
			}
		})
	}
}

// spec: test-attribution / Rust test declarations
func TestRustBaseScopeUsesClosingBrace(t *testing.T) {
	ds := scanRustSource("x.rs", []byte("#[test]\nfn long() {\n    let a = 1;\n    let b = 2;\n}\n\nfn other() {}\n"))
	if len(ds) != 1 || ds[0].Line != 2 || ds[0].End != 5 {
		t.Fatalf("got %+v, want long at lines 2-5", ds)
	}
	if !(lineSet{lines: map[int]bool{4: true}}).touches(ds[0]) {
		t.Fatal("a change inside the body did not touch the test")
	}
	if (lineSet{lines: map[int]bool{7: true}}).touches(ds[0]) {
		t.Fatal("a change after the closing brace touched the test")
	}
}

// spec: test-attribution / Rust test declarations
// spec: openspec/specs/test-attribution/spec.md#Requirement: Diff-scoped attribution
func TestRustBaseScopeCoversAttributes(t *testing.T) {
	for _, tc := range []struct {
		name, before, after, want string
	}{
		{"adding only a test attribute fails", "fn hidden() {}\n", "#[test]\nfn hidden() {}\n", "src/lib.rs:2: hidden: missing spec citation\n"},
		{"adding a cited test attribute passes", "fn hidden() {}\n", citation + "#[test]\nfn hidden() {}\n", ""},
		{"editing a detached header passes", "// License 2025\n\n#[test]\nfn legacy_header() {}\n", "// License 2026\n\n#[test]\nfn legacy_header() {}\n", ""},
		{"editing an attached comment fails", "// Checks parsing.\n#[test]\nfn noted() {}\n", "// Checks framing.\n#[test]\nfn noted() {}\n", "src/lib.rs:3: noted: missing spec citation\n"},
		{"removing the sole citation fails", citation + "#[test]\nfn kept() {}\n", "#[test]\nfn kept() {}\n", "src/lib.rs:2: kept: missing spec citation\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t, "package p\n", "### Requirement: Works\n")
			writeTree(t, root, map[string]string{"src/lib.rs": tc.before, "src/legacy.rs": "#[test]\nfn legacy() {}\n"})
			git(t, root, "init", "-q", "-b", "main")
			git(t, root, "add", ".")
			git(t, root, "commit", "-q", "-m", "base")
			writeTree(t, root, map[string]string{"src/lib.rs": tc.after})
			ds, err := Check(context.Background(), root, "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			if got := Format(ds); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

// spec: test-attribution / Rust test declarations
func FuzzScanRust(f *testing.F) {
	for _, seed := range []string{
		rustSuite, "#[test]\nfn a() {}\n", "r#\"", "'", "'\\u{1F600}'", "b'\\''", "/* /* */ */", "#[", "fn $x",
		"mod m { #[tokio::test] async fn t() {} }", "r#fn", "\"\\", "é#[test]fn x(){}",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		first := scanRustSource("x.rs", data)
		if again := scanRustSource("x.rs", data); !reflect.DeepEqual(first, again) {
			t.Fatalf("nondeterministic: %+v vs %+v", first, again)
		}
		lines := strings.Count(string(data), "\n") + 1
		for _, d := range first {
			if d.Line < 1 || d.Line > lines {
				t.Fatalf("line %d outside 1..%d: %+v", d.Line, lines, d)
			}
			if d.Error == "" && (d.Name == "" || d.End < d.Line || d.End > lines) {
				t.Fatalf("bad declaration span: %+v", d)
			}
		}
	})
}
