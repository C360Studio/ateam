package state

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// Fake credentials only; no real token is read or printed.
const (
	fakeRead  = "fake-read"
	fakeWrite = "fake-write"
)

// spec: openspec/specs/harness/spec.md#Requirement: Forge token roles
func TestForgeTokenRoleResolution(t *testing.T) {
	for _, tc := range []struct {
		name, read, write   string
		wantRead, wantWrite string
	}{
		{name: "neither set leaves the forge fallback"},
		{name: "separate tokens", read: fakeRead, write: fakeWrite, wantRead: fakeRead, wantWrite: fakeWrite},
		{name: "read only serves writes", read: fakeRead, wantRead: fakeRead, wantWrite: fakeRead},
		{name: "write only serves reads", write: fakeWrite, wantRead: fakeWrite, wantWrite: fakeWrite},
		{name: "whitespace read is unset", read: " \t", write: fakeWrite, wantRead: fakeWrite, wantWrite: fakeWrite},
		{name: "values are trimmed", read: " " + fakeRead + "\n", write: "\t" + fakeWrite + " ", wantRead: fakeRead, wantWrite: fakeWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATEAM_TOKEN_READ", tc.read)
			t.Setenv("ATEAM_TOKEN_WRITE", tc.write)
			if got := ReadToken(); got != tc.wantRead {
				t.Errorf("ReadToken() = %q, want %q", got, tc.wantRead)
			}
			if got := WriteToken(); got != tc.wantWrite {
				t.Errorf("WriteToken() = %q, want %q", got, tc.wantWrite)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Forge token roles
func TestGiteaForgeTokenRoles(t *testing.T) {
	for _, tc := range []struct {
		name, read, write, gitea string
		wantGet, wantPost        string
	}{
		{name: "separate role tokens", read: fakeRead, write: fakeWrite, gitea: "fake-gitea", wantGet: "token " + fakeRead, wantPost: "token " + fakeWrite},
		{name: "single role token", write: fakeWrite, gitea: "fake-gitea", wantGet: "token " + fakeWrite, wantPost: "token " + fakeWrite},
		{name: "legacy fallback unchanged", gitea: "fake-gitea", wantGet: "token fake-gitea", wantPost: "token fake-gitea"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATEAM_TOKEN_READ", tc.read)
			t.Setenv("ATEAM_TOKEN_WRITE", tc.write)
			t.Setenv("GITEA_TOKEN", tc.gitea)
			t.Setenv("ATEAM_GITEA_TOKEN", "")
			var mu sync.Mutex
			auth := map[string]map[string]bool{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				if auth[r.Method] == nil {
					auth[r.Method] = map[string]bool{}
				}
				auth[r.Method][r.Header.Get("Authorization")] = true
				mu.Unlock()
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/o/r/pulls" && r.URL.Query().Get("state") == "open":
					_, _ = w.Write([]byte(`[{"number":5,"title":"t","state":"open","head":{"ref":"claude/x"}}]`))
				case r.Method == http.MethodGet:
					_, _ = w.Write([]byte(`[]`))
				default:
					w.WriteHeader(http.StatusCreated)
				}
			}))
			defer srv.Close()
			cfg := ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"}

			if _, err := SyncForgeBoard(context.Background(), t.TempDir(), cfg); err != nil {
				t.Fatalf("sync: %v", err)
			}
			mu.Lock()
			syncGets := auth[http.MethodGet]
			auth = map[string]map[string]bool{}
			mu.Unlock()
			if want := map[string]bool{tc.wantGet: true}; !reflect.DeepEqual(syncGets, want) {
				t.Errorf("sync GET credentials = %v, want only %q", keys(syncGets), tc.wantGet)
			}

			if posted, err := PostPRHandoffComment(context.Background(), t.TempDir(), cfg, "claude/x", "handoff"); err != nil || !posted {
				t.Fatalf("comment: posted=%v err=%v", posted, err)
			}
			if _, err := BootstrapForgeLabels(context.Background(), t.TempDir(), cfg); err != nil {
				t.Fatalf("labels: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if want := map[string]bool{tc.wantPost: true}; !reflect.DeepEqual(auth[http.MethodPost], want) {
				t.Errorf("write POST credentials = %v, want only %q", keys(auth[http.MethodPost]), tc.wantPost)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Forge token roles
func TestGitHubForgeTokenRoles(t *testing.T) {
	for _, tc := range []struct {
		name, read, write   string
		wantRead, wantWrite string
	}{
		{name: "separate role tokens", read: fakeRead, write: fakeWrite, wantRead: fakeRead, wantWrite: fakeWrite},
		{name: "single role token", read: fakeRead, wantRead: fakeRead, wantWrite: fakeRead},
		{name: "gh resolution unchanged", wantRead: "fake-inherited", wantWrite: "fake-inherited"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATEAM_TOKEN_READ", tc.read)
			t.Setenv("ATEAM_TOKEN_WRITE", tc.write)
			t.Setenv("GH_TOKEN", "fake-inherited")
			log := installForgeCLI(t, "gh", "printf 'auth %s %s %s\\n' \"$GH_TOKEN\" \"$1\" \"$2\" >> \"$FORGE_CALL_LOG\"\n"+ghEmptyLists)
			cfg := ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"}
			if _, err := SyncForgeBoard(context.Background(), t.TempDir(), cfg); err != nil {
				t.Fatalf("sync: %v", err)
			}
			if posted, err := PostPRHandoffComment(context.Background(), t.TempDir(), cfg, "claude/x", "handoff"); err != nil || !posted {
				t.Fatalf("comment: posted=%v err=%v", posted, err)
			}
			if _, err := BootstrapForgeLabels(context.Background(), t.TempDir(), cfg); err != nil {
				t.Fatalf("labels: %v", err)
			}
			data, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, line := range strings.Split(string(data), "\n") {
				f := strings.Fields(line)
				if len(f) != 4 || f[0] != "auth" {
					continue
				}
				want := tc.wantRead
				if f[3] == "comment" || f[3] == "create" {
					want = tc.wantWrite
				}
				if f[1] != want {
					t.Errorf("gh %s %s ran with the wrong role token (got %q, want %q)", f[2], f[3], f[1], want)
				}
				seen[f[2]+" "+f[3]] = true
			}
			for _, call := range []string{"issue list", "pr list", "pr comment", "label create"} {
				if !seen[call] {
					t.Errorf("gh %s never ran; seen %v", call, keys(seen))
				}
			}
		})
	}
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
