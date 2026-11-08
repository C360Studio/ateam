package state

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeSettings(t *testing.T, root, body string) {
	t.Helper()
	p := filepath.Join(root, ".agents", "ateam.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestLoadSettingsMissingFile(t *testing.T) {
	s, ok, err := LoadSettings(t.TempDir())
	if err != nil || ok || !reflect.DeepEqual(s, Settings{}) {
		t.Fatalf("missing file: got (%+v, %v, %v), want zero settings, ok=false, nil error", s, ok, err)
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestLoadSettings(t *testing.T) {
	off := false
	cases := []struct {
		name    string
		body    string
		want    Settings
		wantErr string // substring; "" means the file must parse
	}{
		{name: "version only", body: `{"version":1}`, want: Settings{Version: 1}},
		{name: "every key", body: `{"version":1,"merge_method":"rebase","default_branch":"trunk","commit_statuses":false,
			"forges":[{"provider":"GitHub","owner":"o","repo":"r"},{"provider":"gitea","base_url":"https://git.example/","owner":"o","repo":"r"}]}`,
			want: Settings{Version: 1, MergeMethod: "rebase", DefaultBranch: "trunk", CommitStatuses: &off, Forges: []ForgeConfig{
				{Provider: "github", BaseURL: "https://api.github.com", Owner: "o", Repo: "r"},
				{Provider: "gitea", BaseURL: "https://git.example", Owner: "o", Repo: "r"},
			}}},
		{name: "merge", body: `{"version":1,"merge_method":"merge"}`, want: Settings{Version: 1, MergeMethod: "merge"}},
		{name: "squash", body: `{"version":1,"merge_method":"squash"}`, want: Settings{Version: 1, MergeMethod: "squash"}},
		{name: "nested branch", body: `{"version":1,"default_branch":"release/v1"}`, want: Settings{Version: 1, DefaultBranch: "release/v1"}},
		{name: "unknown key", body: `{"version":1,"required_checks":["ci"]}`, wantErr: `unknown field "required_checks"`},
		{name: "version 2", body: `{"version":2}`, wantErr: "version must be 1"},
		{name: "missing version", body: `{"merge_method":"merge"}`, wantErr: "version must be 1"},
		{name: "null", body: `null`, wantErr: "version must be 1"},
		{name: "bad merge method", body: `{"version":1,"merge_method":"rebase-merge"}`, wantErr: "merge_method"},
		{name: "upper-case merge method", body: `{"version":1,"merge_method":"Squash"}`, wantErr: "merge_method"},
		{name: "trailing data", body: `{"version":1} {"version":1}`, wantErr: "trailing data"},
		{name: "trailing garbage", body: `{"version":1}x`, wantErr: "trailing data"},
		{name: "option-like branch", body: `{"version":1,"default_branch":"--all"}`, wantErr: "default_branch"},
		{name: "range branch", body: `{"version":1,"default_branch":"a..b"}`, wantErr: "default_branch"},
		{name: "spaced branch", body: `{"version":1,"default_branch":"my branch"}`, wantErr: "default_branch"},
		{name: "invalid forge", body: `{"version":1,"forges":[{"provider":"bitbucket","owner":"o","repo":"r"}]}`, wantErr: "forges[0]"},
		{name: "forge with credentials", body: `{"version":1,"forges":[{"provider":"gitea","base_url":"https://u:p@git.example","owner":"o","repo":"r"}]}`, wantErr: "forges[0]"},
		{name: "duplicate forge", body: `{"version":1,"forges":[{"provider":"github","owner":"o","repo":"r"},{"provider":"github","owner":"x","repo":"y"}]}`, wantErr: "duplicate provider"},
		{name: "wrong type", body: `{"version":1,"commit_statuses":"no"}`, wantErr: "commit_statuses"},
		{name: "oversized", body: `{"version":1,"default_branch":"` + strings.Repeat("a", 64<<10) + `"}`, wantErr: "exceeds 65536 bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeSettings(t, root, c.body)
			s, ok, err := LoadSettings(root)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) || !strings.Contains(err.Error(), ".agents/ateam.json") {
					t.Fatalf("got err %v, want one naming .agents/ateam.json and %q", err, c.wantErr)
				}
				return
			}
			if err != nil || !ok {
				t.Fatalf("got (%v, ok=%v), want parsed", err, ok)
			}
			if !reflect.DeepEqual(s, c.want) {
				t.Fatalf("got %+v, want %+v", s, c.want)
			}
		})
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestCommitStatusesEnabledPrecedence(t *testing.T) {
	// Truth table written out by hand from D4: env "0"/"1" wins, else the file, else on.
	want := map[string]map[string]bool{
		"unset": {"absent": true, "true": true, "false": false, "invalid": true},
		"0":     {"absent": false, "true": false, "false": false, "invalid": false},
		"1":     {"absent": true, "true": true, "false": true, "invalid": true},
		"yes":   {"absent": true, "true": true, "false": false, "invalid": true},
	}
	files := map[string]string{
		"absent":  "",
		"true":    `{"version":1,"commit_statuses":true}`,
		"false":   `{"version":1,"commit_statuses":false}`,
		"invalid": `{"version":1,"commit_statuses":false,"bogus":1}`,
	}
	for env, row := range want {
		for file, expect := range row {
			t.Run(env+"/"+file, func(t *testing.T) {
				if env == "unset" {
					t.Setenv("ATEAM_COMMIT_STATUSES", "")
					os.Unsetenv("ATEAM_COMMIT_STATUSES")
				} else {
					t.Setenv("ATEAM_COMMIT_STATUSES", env)
				}
				root := t.TempDir()
				if files[file] != "" {
					writeSettings(t, root, files[file])
				}
				if got := CommitStatusesEnabled(root); got != expect {
					t.Fatalf("env %s, file %s: CommitStatusesEnabled = %v, want %v", env, file, got, expect)
				}
			})
		}
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestCommittedForgesAuthoritative(t *testing.T) {
	newRepo := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "https://github.com/detected/repo.git"}} {
			cmd := exec.Command("git", args...)
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v: %s", args, err, out)
			}
		}
		return root
	}
	detected := ForgeConfig{Provider: ForgeGitHub, BaseURL: "https://api.github.com", Owner: "detected", Repo: "repo"}
	committedGitHub := ForgeConfig{Provider: ForgeGitHub, BaseURL: "https://api.github.com", Owner: "team", Repo: "app"}
	committedGitea := ForgeConfig{Provider: ForgeGitea, BaseURL: "https://git.example", Owner: "team", Repo: "app"}
	uiEdited := ForgeConfig{Provider: ForgeGitea, BaseURL: "https://other.example", Owner: "ui", Repo: "edit"}

	t.Run("committed list beats detection, ATEAM_FORGE and a board edit", func(t *testing.T) {
		t.Setenv("ATEAM_FORGE", "gitea")
		t.Setenv("GITEA_URL", "https://env.example")
		root := newRepo(t)
		if _, err := Update(root, func(st *State) error { return st.UpsertForge(uiEdited) }); err != nil {
			t.Fatal(err)
		}
		writeSettings(t, root, `{"version":1,"forges":[{"provider":"github","owner":"team","repo":"app"}]}`)
		st, err := Refresh(context.Background(), root, false)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(st.Forges, []ForgeConfig{committedGitHub}) || st.Forge != committedGitHub || !st.ForgesConfigured {
			t.Fatalf("got forge %+v, forges %+v, configured %v; want only the committed %+v", st.Forge, st.Forges, st.ForgesConfigured, committedGitHub)
		}
	})
	t.Run("detection picks the active committed entry", func(t *testing.T) {
		t.Setenv("ATEAM_FORGE", "")
		t.Setenv("GITEA_URL", "")
		root := newRepo(t)
		writeSettings(t, root, `{"version":1,"forges":[{"provider":"gitea","base_url":"https://git.example","owner":"team","repo":"app"},{"provider":"github","owner":"team","repo":"app"}]}`)
		st, err := Refresh(context.Background(), root, false)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(st.Forges, []ForgeConfig{committedGitea, committedGitHub}) || st.Forge != committedGitHub {
			t.Fatalf("got forge %+v, forges %+v; want github active within the committed list", st.Forge, st.Forges)
		}
	})
	for name, settings := range map[string]string{"no file": "", "no forges key": `{"version":1}`, "invalid file": `{"version":1,"forges":[{"provider":"svn"}]}`} {
		t.Run(name+" keeps detection", func(t *testing.T) {
			t.Setenv("ATEAM_FORGE", "")
			t.Setenv("GITEA_URL", "")
			root := newRepo(t)
			if settings != "" {
				writeSettings(t, root, settings)
			}
			st, err := Refresh(context.Background(), root, false)
			if err != nil {
				t.Fatal(err)
			}
			if st.Forge != detected || st.ForgesConfigured {
				t.Fatalf("got forge %+v, configured %v; want detected %+v, unconfigured", st.Forge, st.ForgesConfigured, detected)
			}
		})
	}
}
