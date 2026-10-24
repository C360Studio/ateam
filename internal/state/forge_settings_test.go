package state

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func installForgeCLI(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	t.Setenv("FORGE_CALL_LOG", log)
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$FORGE_CALL_LOG\"\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestConfiguredGitHubTarget(t *testing.T) {
	log := installForgeCLI(t, "gh", "case \"$*\" in *comments*) exit 1;; esac\nprintf '[]'\n")
	_, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "chosen", Repo: "target"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	calls := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(calls) != 6 {
		t.Fatalf("calls = %q, want open+closed issue (each with fallback) and PR lists", calls)
	}
	for _, call := range calls {
		if !strings.Contains(call, "--repo github.com/chosen/target") {
			t.Errorf("configured target missing from call: %s", call)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestConfiguredGitHubHostRefusal(t *testing.T) {
	log := installForgeCLI(t, "gh", "printf '[]'\n")
	_, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, BaseURL: "https://github.example/api/v3", Owner: "chosen", Repo: "target"})
	if err == nil {
		t.Fatal("unsupported GitHub host was accepted")
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("gh invoked for unsupported host")
	}
}

type refusedForgeTransport struct{}

func (refusedForgeTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader("forbidden"))}, nil
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestExplicitGiteaFailureDoesNotFallback(t *testing.T) {
	log := installForgeCLI(t, "tea", "printf '[]'\n")
	original := http.DefaultTransport
	http.DefaultTransport = refusedForgeTransport{}
	t.Cleanup(func() { http.DefaultTransport = original })
	_, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: "https://forge.example", Owner: "chosen", Repo: "target"})
	if err == nil {
		t.Fatal("explicit Gitea failure was hidden by fallback")
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("tea fallback invoked for explicit target")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestForgeConfigurationValidation(t *testing.T) {
	valid := ForgeConfig{Provider: ForgeGitea, BaseURL: "https://forge.example", Owner: "chosen", Repo: "target"}
	for _, tc := range []struct{ name, field, value string }{
		{"scheme", "url", "file:///tmp/forge"}, {"userinfo", "url", "https://user:pass@forge.example"}, {"query", "url", "https://forge.example?token=secret"}, {"fragment", "url", "https://forge.example#fragment"}, {"relative", "url", "/forge"},
		{"slash owner", "owner", "../other"}, {"dot owner", "owner", ".."}, {"query repo", "repo", "repo?x=y"}, {"encoded slash", "repo", "a%2fb"}, {"backslash", "repo", `a\b`}, {"space", "owner", "two words"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			switch tc.field {
			case "url":
				cfg.BaseURL = tc.value
			case "owner":
				cfg.Owner = tc.value
			case "repo":
				cfg.Repo = tc.value
			}
			st := &State{Forge: valid, Forges: []ForgeConfig{valid}}
			before := &State{Forge: valid, Forges: []ForgeConfig{valid}}
			if err := st.UpsertForge(cfg); err == nil {
				t.Errorf("invalid configuration accepted: %+v", cfg)
			}
			if !reflect.DeepEqual(st, before) {
				t.Error("invalid configuration mutated state")
			}
		})
	}
	st := &State{}
	if err := st.UpsertForge(valid); err != nil {
		t.Fatalf("valid configuration rejected: %v", err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestResolveForgeDoesNotRelabel(t *testing.T) {
	cfg := ForgeConfig{Provider: ForgeGitHub, Owner: "chosen", Repo: "target"}
	st := &State{Forge: cfg, Forges: []ForgeConfig{cfg}}
	for _, p := range []ForgeProvider{"unknown", ForgeGitea} {
		if got := st.ResolveForge(p); got != (ForgeConfig{}) {
			t.Errorf("unconfigured provider %q borrowed config: %+v", p, got)
		}
	}
	if got := st.ResolveForge(""); got != cfg {
		t.Errorf("default config = %+v", got)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestForgeRemovalPersistsAcrossRefresh(t *testing.T) {
	t.Setenv("ATEAM_FORGE", "")
	t.Setenv("GITEA_URL", "")
	t.Setenv("GITHUB_API_URL", "")
	root := t.TempDir()
	for _, args := range [][]string{{"init"}, {"remote", "add", "origin", "https://github.com/origin/repo.git"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	st, err := Refresh(context.Background(), root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Forges) != 1 {
		t.Fatalf("origin not detected: %+v", st.Forges)
	}
	if _, err := Update(root, func(st *State) error { st.RemoveForge(ForgeGitHub); return nil }); err != nil {
		t.Fatal(err)
	}
	log := installForgeCLI(t, "gh", "printf '[]'\n")
	for _, syncRemote := range []bool{false, true} {
		st, err = Refresh(context.Background(), root, syncRemote)
		if err != nil {
			t.Fatal(err)
		}
		if len(st.Forges) != 0 || st.Forge.Provider != ForgeLocal {
			t.Fatalf("removed origin reappeared: %+v / %+v", st.Forge, st.Forges)
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("removed forge was synced")
	}
	cfg := ForgeConfig{Provider: ForgeGitHub, Owner: "replacement", Repo: "repo"}
	if _, err := Update(root, func(st *State) error { return st.UpsertForge(cfg) }); err != nil {
		t.Fatal(err)
	}
	st, err = Refresh(context.Background(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	if st.Forge.Owner != "replacement" {
		t.Fatalf("explicit target replaced by origin: %+v", st.Forge)
	}
	calls, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(calls), "--repo github.com/replacement/repo") != 4 {
		t.Fatalf("refresh did not sync configured target: %s", calls)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestAutomaticForgeTracksOriginChanges(t *testing.T) {
	t.Setenv("ATEAM_FORGE", "")
	t.Setenv("GITEA_URL", "")
	t.Setenv("GITHUB_API_URL", "")
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init")
	git("remote", "add", "origin", "https://gitea.example/other/repo.git")
	if _, err := Refresh(context.Background(), root, false); err != nil {
		t.Fatal(err)
	}
	log := installForgeCLI(t, "gh", "printf '[]'\n")
	for _, owner := range []string{"original", "changed"} {
		git("remote", "set-url", "origin", "https://github.com/"+owner+"/repo.git")
		st, err := Refresh(context.Background(), root, true)
		if err != nil {
			t.Fatal(err)
		}
		want := ForgeConfig{Provider: ForgeGitHub, BaseURL: "https://api.github.com", Owner: owner, Repo: "repo"}
		if st.Forge != want || st.ResolveForge(ForgeGitHub) != want {
			t.Fatalf("automatic target did not follow origin: primary=%+v, configured=%+v, want=%+v", st.Forge, st.ResolveForge(ForgeGitHub), want)
		}
		other := ForgeConfig{Provider: ForgeGitea, BaseURL: "https://gitea.example", Owner: "other", Repo: "repo"}
		if st.ResolveForge(ForgeGitea) != other || len(st.Forges) != 2 {
			t.Fatalf("other provider changed: %+v", st.Forges)
		}
		if st.ForgesConfigured {
			t.Fatal("automatic refresh became explicit configuration")
		}
	}
	for _, args := range [][]string{{"remote", "remove", "origin"}, {"remote", "add", "origin", filepath.Join(root, "local-repository")}} {
		git(args...)
		st, err := Refresh(context.Background(), root, false)
		if err != nil {
			t.Fatal(err)
		}
		if st.Forge.Provider != ForgeLocal {
			t.Fatalf("local origin retained automatic remote target: %+v", st.Forge)
		}
		if st.ResolveForge(ForgeGitHub).Owner != "changed" || len(st.Forges) != 2 {
			t.Fatalf("cached provider entries disappeared: %+v", st.Forges)
		}
		if _, err := Refresh(context.Background(), root, true); err != nil {
			t.Fatal(err)
		}
		calls, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		if len(strings.Split(strings.TrimSpace(string(calls)), "\n")) != 8 {
			t.Fatalf("local origin triggered remote sync: %s", calls)
		}
	}
	pinned := ForgeConfig{Provider: ForgeGitHub, BaseURL: "https://api.github.com", Owner: "pinned", Repo: "repo"}
	if _, err := Update(root, func(st *State) error { return st.UpsertForge(pinned) }); err != nil {
		t.Fatal(err)
	}
	git("remote", "set-url", "origin", "https://github.com/ignored/repo.git")
	st, err := Refresh(context.Background(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	if st.Forge != pinned {
		t.Fatalf("explicit configuration followed changed origin: %+v", st.Forge)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	calls := strings.Split(strings.TrimSpace(string(data)), "\n")
	expected := []string{"original", "original", "original", "original", "changed", "changed", "changed", "changed", "pinned", "pinned", "pinned", "pinned"}
	if len(calls) != len(expected) {
		t.Fatalf("calls = %q, want six issue/PR requests", calls)
	}
	for i, owner := range expected {
		if !strings.Contains(calls[i], "--repo github.com/"+owner+"/repo") {
			t.Errorf("sync history call %d = %q, want target %s", i, calls[i], owner)
		}
	}
}
