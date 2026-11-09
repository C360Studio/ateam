package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The card ateam init writes stays static and small, and each skill carries the
// card paragraph once, behind the T3 condition (#304).
// spec: openspec/specs/harness/spec.md#Requirement: T3 Code in-thread confidence card
func TestT3CardTemplate(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatalf("init: %v", err)
	}
	card, err := os.ReadFile(filepath.Join(root, t3CardPath))
	if err != nil {
		t.Fatal(err)
	}
	if violations := t3CardViolations(string(card)); len(violations) != 0 {
		t.Fatalf("generated card: %v", violations)
	}
	for _, tc := range []struct{ dir, skill, kind, link string }{
		{root, "pickup", "Pickup", "../handoff/t3-card.html"},
		{root, "handoff", "Handoff", "t3-card.html"},
		// The repository's own canonical skills carry the same paragraph.
		{filepath.Join("..", ".."), "pickup", "Pickup", "../handoff/t3-card.html"},
		{filepath.Join("..", ".."), "handoff", "Handoff", "t3-card.html"},
	} {
		body, err := os.ReadFile(filepath.Join(tc.dir, ".agents", "skills", tc.skill, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if n := strings.Count(text, "html_render"); n != 2 {
			t.Errorf("%s/%s: html_render named %d times, want 2 (condition and call)", tc.dir, tc.skill, n)
		}
		if n := strings.Count(text, t3CardParagraph(tc.kind, tc.link)); n != 1 {
			t.Errorf("%s/%s: card paragraph appears %d times, want 1", tc.dir, tc.skill, n)
		}
		if !strings.HasPrefix(t3CardParagraph(tc.kind, tc.link), t3CardCondition) {
			t.Errorf("card paragraph does not begin with the T3 condition")
		}
	}
	repoCard, err := os.ReadFile(filepath.Join("..", "..", t3CardPath))
	if err != nil {
		t.Fatal(err)
	}
	if string(repoCard) != t3CardHTML {
		t.Errorf("%s differs from the bootstrap template", t3CardPath)
	}
}

// The static checker itself names each limit a card can break.
// spec: openspec/specs/harness/spec.md#Requirement: T3 Code in-thread confidence card
func TestT3CardViolations(t *testing.T) {
	cases := []struct{ name, mutate, want string }{
		{"script", strings.Replace(t3CardHTML, "</body>", "<script>x()</script></body>", 1), "<script"},
		{"remote url", strings.Replace(t3CardHTML, "</body>", `<img src="https://x/y.png"></body>`, 1), "http"},
		{"too big", strings.Replace(t3CardHTML, "</body>", strings.Repeat("x", 8192)+"</body>", 1), "8192"},
		{"row missing", strings.Replace(t3CardHTML, "{{MERGE}}", "", 1), "{{MERGE}}"},
		{"region missing", strings.Replace(t3CardHTML, "Agent-written (not verified)", "Notes", 1), "Agent-written"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(t3CardViolations(tc.mutate), "; ")
			if !strings.Contains(got, tc.want) {
				t.Fatalf("violations %q do not name %q", got, tc.want)
			}
		})
	}
}

// t3CardViolations lists every way a card breaks the Requirement's static limits.
func t3CardViolations(card string) []string {
	var out []string
	lower := strings.ToLower(card)
	if strings.Contains(lower, "<script") {
		out = append(out, "contains <script")
	}
	if strings.Contains(lower, "http") {
		out = append(out, "contains http")
	}
	if len(card) >= 8192 {
		out = append(out, "not under 8192 bytes")
	}
	for _, name := range []string{"{{PROPOSAL}}", "{{CLAIM}}", "{{GATES}}", "{{REVIEW}}", "{{MERGE}}", "{{GOAL}}", "{{DONE_WHEN}}"} {
		if !strings.Contains(card, name) {
			out = append(out, "missing "+name)
		}
	}
	if !strings.Contains(card, "Agent-written (not verified)") {
		out = append(out, "missing Agent-written region")
	}
	return out
}
