package testspec

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Rust test discovery is a Go-native lexer, not a Rust parser (design D1 of test-spec-rust-384): it tokenises
// comments, literals, attributes and braces well enough to find attributed functions without a Rust toolchain.

type rustKind int

const (
	rustIdent rustKind = iota
	rustPunct
	rustLiteral
	rustLineComment
	rustBlockComment
)

type rustToken struct {
	kind      rustKind
	text      string
	line, end int
}

func scanRust(root, rel string) []declaration {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return []declaration{{File: rel, Line: 1, Error: "read Rust: " + err.Error()}}
	}
	return scanRustSource(rel, data)
}

// scanRustSource selects functions whose outer attributes include a path ending in `test` (design D2) and
// attaches the contiguous comment and attribute run above them (D3). Unsupported forms fail closed (D4).
func scanRustSource(rel string, data []byte) []declaration {
	toks, err := lexRust(data)
	if err != nil {
		return []declaration{{File: rel, Line: err.line, Error: "parse Rust: " + err.msg}}
	}
	type runElem struct {
		tok  rustToken
		test bool
	}
	var out []declaration
	var run []runElem
	var mods []string // one entry per open brace; a module name or ""
	lastCode := 0     // end line of the last code token, to drop trailing comments
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if t.kind == rustLineComment || t.kind == rustBlockComment {
			if len(run) == 0 && t.line == lastCode {
				continue
			}
			run = append(run, runElem{tok: t})
			continue
		}
		if t.kind == rustPunct && t.text == "#" {
			open, inner := i+1, false
			if open < len(toks) && toks[open].text == "!" {
				open, inner = open+1, true
			}
			if open < len(toks) && toks[open].kind == rustPunct && toks[open].text == "[" {
				closeAt := matchRust(toks, open, "[", "]")
				if closeAt < 0 {
					return []declaration{{File: rel, Line: t.line, Error: "parse Rust: unterminated attribute"}}
				}
				if inner {
					run, lastCode = nil, toks[closeAt].end
				} else {
					attr := rustToken{kind: rustPunct, text: "#[", line: t.line, end: toks[closeAt].end}
					run = append(run, runElem{tok: attr, test: rustTestPath(toks[open+1 : closeAt])})
				}
				i = closeAt
				continue
			}
		}
		isTest := false
		for _, e := range run {
			isTest = isTest || e.test
		}
		if isTest {
			d := rustTestItem(rel, toks, i, mods)
			if d.Error == "" {
				prev := t.line
				for k := len(run) - 1; k >= 0; k-- {
					e := run[k].tok
					if e.end+1 < prev {
						break
					}
					if e.kind == rustLineComment || e.kind == rustBlockComment {
						d.Comments = append([]comment{{e.line, e.text}}, d.Comments...)
					}
					prev = e.line
				}
			}
			out = append(out, d)
		}
		run = nil
		switch {
		case t.kind == rustPunct && t.text == "{":
			name := ""
			if i >= 2 && toks[i-2].kind == rustIdent && toks[i-2].text == "mod" && toks[i-1].kind == rustIdent {
				name = toks[i-1].text
			}
			mods = append(mods, name)
		case t.kind == rustPunct && t.text == "}":
			if len(mods) == 0 {
				return []declaration{{File: rel, Line: t.line, Error: "parse Rust: unbalanced closing brace"}}
			}
			mods = mods[:len(mods)-1]
		}
		lastCode = t.end
	}
	if len(mods) > 0 {
		return []declaration{{File: rel, Line: 1, Error: "parse Rust: unclosed brace"}}
	}
	return out
}

// rustTestPath reports whether attribute tokens start with a path whose last segment is `test`.
func rustTestPath(toks []rustToken) bool {
	last := ""
	for _, t := range toks {
		if t.kind == rustIdent {
			last = t.text
			continue
		}
		if t.kind == rustPunct && t.text == ":" {
			continue
		}
		break
	}
	return last == "test"
}

// rustTestItem reads the item starting at toks[i], which follows a test attribute.
func rustTestItem(rel string, toks []rustToken, i int, mods []string) declaration {
	d := declaration{File: rel, Line: toks[i].line}
	j := i
	if j < len(toks) && toks[j].text == "pub" {
		j++
		if j < len(toks) && toks[j].text == "(" {
			if j = matchRust(toks, j, "(", ")"); j < 0 {
				d.Error = "parse Rust: unterminated visibility"
				return d
			}
			j++
		}
	}
	for j < len(toks) && toks[j].kind == rustIdent && (toks[j].text == "const" || toks[j].text == "async" || toks[j].text == "unsafe" || toks[j].text == "extern") {
		j++
		if toks[j-1].text == "extern" && j < len(toks) && toks[j].kind == rustLiteral {
			j++
		}
	}
	if j >= len(toks) || toks[j].kind != rustIdent || toks[j].text != "fn" {
		d.Error = "test attribute is not on a fn item"
		return d
	}
	if j+1 >= len(toks) || toks[j+1].kind != rustIdent {
		d.Error = "test fn name is not a literal identifier (macro-generated tests are unsupported; cite a hand-written wrapper test)"
		return d
	}
	var path []string
	for _, m := range mods {
		if m != "" {
			path = append(path, m)
		}
	}
	d.Name = strings.Join(append(path, strings.TrimPrefix(toks[j+1].text, "r#")), "::")
	depth := 0
	for k := j + 2; k < len(toks); k++ {
		if toks[k].kind != rustPunct {
			continue
		}
		switch toks[k].text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case ";":
			if depth == 0 {
				d.Error = d.Name + ": test fn has no body"
				return d
			}
		case "{":
			if depth == 0 {
				end := matchRust(toks, k, "{", "}")
				if end < 0 {
					d.Error = d.Name + ": unclosed test fn body"
					return d
				}
				d.End = toks[end].end
				return d
			}
		}
	}
	d.Error = d.Name + ": test fn has no body"
	return d
}

// matchRust returns the index of the punct closing the one at toks[open], or -1.
func matchRust(toks []rustToken, open int, left, right string) int {
	depth := 0
	for k := open; k < len(toks); k++ {
		if toks[k].kind != rustPunct {
			continue
		}
		switch toks[k].text {
		case left:
			depth++
		case right:
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

type rustLexError struct {
	line int
	msg  string
}

func lexRust(data []byte) ([]rustToken, *rustLexError) {
	src := string(data)
	var toks []rustToken
	line := 1
	i := 0
	// advance moves i to end, counting newlines.
	advance := func(end int) {
		line += strings.Count(src[i:end], "\n")
		i = end
	}
	fail := func(startLine int, what string) *rustLexError {
		return &rustLexError{startLine, "unterminated " + what}
	}
	for i < len(src) {
		c := src[i]
		start, startLine := i, line
		switch {
		case c == '\n' || c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			advance(i + 1)
		case strings.HasPrefix(src[i:], "//"):
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				end = len(src) - i
			}
			toks = append(toks, rustToken{rustLineComment, strings.TrimRight(src[i:i+end], "\r"), line, line})
			i += end
		case strings.HasPrefix(src[i:], "/*"):
			depth, k := 0, i
			for k < len(src) {
				if strings.HasPrefix(src[k:], "/*") {
					depth, k = depth+1, k+2
				} else if strings.HasPrefix(src[k:], "*/") {
					depth, k = depth-1, k+2
					if depth == 0 {
						break
					}
				} else {
					k++
				}
			}
			if depth != 0 {
				return nil, fail(startLine, "block comment")
			}
			advance(k)
			toks = append(toks, rustToken{rustBlockComment, src[start:k], startLine, line})
		case c == '"':
			k, ok := rustQuoted(src, i+1, '"')
			if !ok {
				return nil, fail(startLine, "string")
			}
			advance(k)
			toks = append(toks, rustToken{rustLiteral, src[start:k], startLine, line})
		case c == '\'':
			// A char literal closes within one escape or rune; anything else is a lifetime or label.
			if k, ok := rustCharEnd(src, i+1); ok {
				advance(k)
				toks = append(toks, rustToken{rustLiteral, src[start:k], startLine, line})
			} else {
				toks = append(toks, rustToken{rustPunct, "'", line, line})
				i++
			}
		case c == '_' || c >= 0x80 || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)):
			if rustIdentEnd(src, i) == i { // an invalid or non-identifier rune
				_, size := utf8.DecodeRuneInString(src[i:])
				toks = append(toks, rustToken{rustPunct, src[i : i+size], line, line})
				i += size
				continue
			}
			kind, end, unterminated := lexRustWord(src, i)
			if unterminated != "" {
				return nil, fail(startLine, unterminated)
			}
			advance(end)
			toks = append(toks, rustToken{kind, src[start:end], startLine, line})
		default:
			toks = append(toks, rustToken{rustPunct, string(c), line, line})
			i++
		}
	}
	return toks, nil
}

// lexRustWord lexes the identifier-like token at i: an identifier, a raw identifier, or a prefixed raw, byte or C
// string or byte char. It returns the token kind and end index, or the name of an unterminated literal.
func lexRustWord(src string, i int) (rustKind, int, string) {
	k := rustIdentEnd(src, i)
	word := src[i:k]
	if word == "r" || word == "br" || word == "cr" {
		hashes := k
		for hashes < len(src) && src[hashes] == '#' {
			hashes++
		}
		if hashes < len(src) && src[hashes] == '"' {
			closer := "\"" + strings.Repeat("#", hashes-k)
			end := strings.Index(src[hashes+1:], closer)
			if end < 0 {
				return 0, 0, "raw string"
			}
			return rustLiteral, hashes + 1 + end + len(closer), ""
		}
		if word == "r" && hashes == k+1 {
			if m := rustIdentEnd(src, hashes); m > hashes {
				return rustIdent, m, "" // raw identifier r#name
			}
		}
	}
	if (word == "b" || word == "c") && k < len(src) && src[k] == '"' {
		end, ok := rustQuoted(src, k+1, '"')
		if !ok {
			return 0, 0, "string"
		}
		return rustLiteral, end, ""
	}
	if word == "b" && k < len(src) && src[k] == '\'' {
		if end, ok := rustCharEnd(src, k+1); ok {
			return rustLiteral, end, ""
		}
	}
	return rustIdent, k, ""
}

func rustIdentEnd(src string, i int) int {
	for i < len(src) {
		r, size := utf8.DecodeRuneInString(src[i:])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		i += size
	}
	return i
}

// rustQuoted returns the index after the closing quote of a string body starting at i.
func rustQuoted(src string, i int, quote byte) (int, bool) {
	for i < len(src) {
		switch src[i] {
		case '\\':
			i += 2
		case quote:
			return i + 1, true
		default:
			i++
		}
	}
	return 0, false
}

// rustCharEnd returns the index after a char literal body starting at i ('a', '\n', '\u{1F600}'), if any.
func rustCharEnd(src string, i int) (int, bool) {
	if i >= len(src) || src[i] == '\n' || src[i] == '\'' {
		return 0, false
	}
	if src[i] == '\\' {
		if i+1 >= len(src) {
			return 0, false
		}
		k := i + 2
		if src[i+1] == 'u' && k < len(src) && src[k] == '{' {
			end := strings.IndexByte(src[k:], '}')
			if end < 0 {
				return 0, false
			}
			k += end + 1
		} else if src[i+1] == 'x' {
			k += 2
		}
		if k < len(src) && src[k] == '\'' {
			return k + 1, true
		}
		return 0, false
	}
	_, size := utf8.DecodeRuneInString(src[i:])
	if i+size < len(src) && src[i+size] == '\'' {
		return i + size + 1, true
	}
	return 0, false
}
