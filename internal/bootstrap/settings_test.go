package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTeamSettings(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "ateam.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestDoctorTeamSettings(t *testing.T) {
	t.Run("absent file reports nothing", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		if v := CheckParity(root); len(v) != 0 {
			t.Fatalf("violations without .agents/ateam.json: %v", v)
		}
		if w := CheckMergeMethodDocs(root); len(w) != 0 {
			t.Fatalf("warnings without .agents/ateam.json: %v", w)
		}
	})
	t.Run("invalid file fails the check", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		writeTeamSettings(t, root, `{"version":2}`)
		requireViolation(t, CheckParity(root), ".agents/ateam.json", "version must be 1")
	})
	t.Run("valid file passes the check", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		writeTeamSettings(t, root, `{"version":1,"default_branch":"trunk"}`)
		if v := CheckParity(root); len(v) != 0 {
			t.Fatalf("violations for a valid file: %v", v)
		}
		if w := CheckMergeMethodDocs(root); len(w) != 0 {
			t.Fatalf("warned without merge_method: %v", w)
		}
	})
	t.Run("protocol naming another method warns", func(t *testing.T) {
		root, _ := initWithPolicy(t, "") // protocol written with the squash command
		writeTeamSettings(t, root, `{"version":1,"merge_method":"merge"}`)
		w := CheckMergeMethodDocs(root)
		if len(w) != 1 || !strings.Contains(w[0], "`gh pr merge --merge --auto`") || !strings.Contains(w[0], "ateam init --force") {
			t.Fatalf("got warnings %q, want one naming gh pr merge --merge --auto and ateam init --force", w)
		}
		if v := CheckParity(root); len(v) != 0 {
			t.Fatalf("a stale protocol is a warning, not a violation: %v", v)
		}
	})
	t.Run("protocol naming the method is quiet", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		writeTeamSettings(t, root, `{"version":1,"merge_method":"squash"}`)
		if w := CheckMergeMethodDocs(root); len(w) != 0 {
			t.Fatalf("warned although protocol names gh pr merge --squash --auto: %v", w)
		}
	})
}

// initWithSettings runs ateam init in a fresh directory holding policy and settings ("" for none) and returns
// the directory, every generated template file as written, and the init error.
func initWithSettings(t *testing.T, policy, settings string) (string, map[string]string, error) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if policy != "" {
		if err := os.WriteFile(filepath.Join(root, reviewPolicyPath), []byte(policy), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if settings != "" {
		writeTeamSettings(t, root, settings)
	}
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		return root, nil, err
	}
	got := map[string]string{}
	for rel := range defaultTemplateFiles() {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		got[rel] = string(data)
	}
	return root, got, nil
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestInitMergeMethod(t *testing.T) {
	defaults := defaultTemplateFiles()
	closeFiles := map[string]bool{"AGENTS.md": true, ".agents/rules-index.md": true, ".agents/protocol.md": true}
	for _, c := range []struct {
		name, settings  string
		rule, gh, gitea string // "" for byte-identical output
	}{
		{name: "no file"},
		{name: "no merge_method", settings: `{"version":1}`},
		{name: "squash", settings: `{"version":1,"merge_method":"squash"}`},
		{name: "merge", settings: `{"version":1,"merge_method":"merge"}`,
			rule: "the merge-commit auto-merge (GitHub or Gitea) has landed", gh: "`gh pr merge --merge --auto`", gitea: "`{\"do\":\"merge\",\"merge_when_checks_succeed\":true}`"},
		{name: "rebase", settings: `{"version":1,"merge_method":"rebase"}`,
			rule: "the rebase auto-merge (GitHub or Gitea) has landed", gh: "`gh pr merge --rebase --auto`", gitea: "`{\"do\":\"rebase\",\"merge_when_checks_succeed\":true}`"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, got, err := initWithSettings(t, "", c.settings)
			if err != nil {
				t.Fatal(err)
			}
			for rel, want := range defaults {
				if c.rule == "" || !closeFiles[rel] {
					if got[rel] != want {
						t.Errorf("%s differs from defaultTemplateFiles()", rel)
					}
				}
			}
			if w := CheckMergeMethodDocs(root); len(w) != 0 {
				t.Errorf("doctor warns on the protocol init just wrote: %v", w)
			}
			if c.rule == "" {
				return
			}
			for _, rel := range []string{"AGENTS.md", ".agents/rules-index.md"} {
				if !strings.Contains(got[rel], c.rule) {
					t.Errorf("%s lacks %q", rel, c.rule)
				}
			}
			proto := got[".agents/protocol.md"]
			if !strings.Contains(proto, "("+c.gh+"; on Gitea, `POST /repos/{owner}/{repo}/pulls/{n}/merge` with "+c.gitea+")") {
				t.Errorf("protocol Close step lacks %s and %s", c.gh, c.gitea)
			}
			for rel := range closeFiles {
				for _, stale := range []string{"--squash", `"do":"squash"`, "squash auto-merge"} {
					if strings.Contains(got[rel], stale) {
						t.Errorf("%s still names %s", rel, stale)
					}
				}
			}
		})
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestInitMergeMethodAIPUnaffected(t *testing.T) {
	aipPolicy := `{"version":1,"review_record":"aip",` + recordTiers + `}`
	_, want, err := initWithSettings(t, aipPolicy, "")
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := initWithSettings(t, aipPolicy, `{"version":1,"merge_method":"rebase"}`)
	if err != nil {
		t.Fatal(err)
	}
	for rel := range want {
		if got[rel] != want[rel] {
			t.Errorf("%s: merge_method changed review_record: aip output", rel)
		}
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestInitInvalidSettingsFails(t *testing.T) {
	if _, _, err := initWithSettings(t, "", `{"version":1,"merge_method":"fast-forward-only"}`); err == nil || !strings.Contains(err.Error(), ".agents/ateam.json") {
		t.Fatalf("init err = %v, want one naming .agents/ateam.json", err)
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestMergeMethodSwapTextsAppearOnce(t *testing.T) {
	for _, c := range []struct{ text, sub string }{
		{closeRuleRow, "squash auto-merge"},
		{protocolCloseStep, "gh pr merge --squash --auto"},
		{protocolCloseStep, `{"do":"squash",`},
	} {
		if n := strings.Count(c.text, c.sub); n != 1 {
			t.Errorf("%q appears %d times, want 1", c.sub, n)
		}
	}
	// Every "squash" in the Close constants is one the replacer rewrites.
	if n := strings.Count(closeRuleRow+protocolCloseStep, "squash"); n != 3 {
		t.Errorf("Close constants name squash %d times, want 3 (one per swapped substring)", n)
	}
	for rel, text := range defaultTemplateFiles() {
		if want := map[string]int{"AGENTS.md": 1, ".agents/rules-index.md": 1, ".agents/protocol.md": 2}[rel]; strings.Count(text, "squash") != want {
			t.Errorf("%s names squash %d times, want %d", rel, strings.Count(text, "squash"), want)
		}
	}
}
