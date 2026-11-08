package state

import (
	"os"
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
