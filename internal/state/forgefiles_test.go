package state

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// forgeCall is one forge request as a recorded fake saw it.
type forgeCall struct {
	Op      string // head, read, write, login
	Path    string
	Ref     string
	SHA     string
	Content string
}

// forgeBackend is a recorded forge: files at headSHA, an optional path whose write is refused as stale, and the
// calls the client made.
type forgeBackend struct {
	files    ForgeFiles
	calls    func() []forgeCall
	conflict func(path string)
}

const (
	fakeHead   = "1111111111111111111111111111111111111111"
	fakeBranch = "claude/alpha"
)

var fakeTree = map[string]struct{ content, sha string }{
	"openspec/changes/alpha/proposal.md": {"# Alpha\n\nApproved-by:\n", "blob-proposal"},
	"openspec/changes/alpha/tasks.md":    {"- [ ] 1.1 Approve. Hold: OPEN-Q proposal awaits owner approval\n", "blob-tasks"},
}

// githubBackend fakes gh: it answers from fakeTree and logs each call as JSON lines, PUT bodies included.
func githubBackend(t *testing.T) forgeBackend {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	tree := filepath.Join(dir, "tree")
	for p, f := range fakeTree {
		answer, _ := json.Marshal(map[string]string{"content": base64.StdEncoding.EncodeToString([]byte(f.content)), "encoding": "base64", "sha": f.sha})
		writeRepoFile(t, tree, p, string(answer))
	}
	t.Setenv("FORGE_CALL_LOG", log)
	t.Setenv("FAKE_TREE", tree)
	t.Setenv("FAKE_CONFLICT", "")
	t.Setenv("GH_TOKEN", "")
	writeFake(t, filepath.Join(dir, "gh"), `
case "$*" in
"api repos/o/r/git/ref/heads/`+fakeBranch+`") echo "head|`+fakeBranch+`" >> "$FORGE_CALL_LOG"; echo '{"object":{"sha":"`+fakeHead+`"}}';;
"api repos/o/r/git/ref/heads/"*) echo '{"message":"Not Found","status":"404"}'; exit 1;;
"api repos/o/r/contents/"*"?ref="*)
  rest=${2#repos/o/r/contents/}; p=${rest%%\?ref=*}; ref=${rest#*\?ref=}
  echo "read|$p|$ref" >> "$FORGE_CALL_LOG"
  if [ -f "$FAKE_TREE/$p" ]; then cat "$FAKE_TREE/$p"; else echo '{"message":"Not Found","status":"404"}'; exit 1; fi;;
"api -X PUT repos/o/r/contents/"*" --input -")
  p=${4#repos/o/r/contents/}; body=$(cat)
  echo "write|$p|$body" >> "$FORGE_CALL_LOG"
  if [ "$p" = "$FAKE_CONFLICT" ]; then echo '{"message":"is at x but expected y","status":"409"}'; exit 1; fi
  echo '{"content":{}}';;
"api user") echo "login" >> "$FORGE_CALL_LOG"; echo '{"login":"ann"}';;
*) echo "unexpected|$*" >> "$FORGE_CALL_LOG"; exit 2;;
esac
`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	files, err := NewForgeFiles(t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"})
	if err != nil {
		t.Fatal(err)
	}
	return forgeBackend{
		files:    files,
		conflict: func(p string) { t.Setenv("FAKE_CONFLICT", p) },
		calls: func() []forgeCall {
			data, _ := os.ReadFile(log)
			var out []forgeCall
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				f := strings.SplitN(line, "|", 3)
				switch f[0] {
				case "head":
					out = append(out, forgeCall{Op: "head", Ref: f[1]})
				case "read":
					out = append(out, forgeCall{Op: "read", Path: f[1], Ref: f[2]})
				case "write":
					var b map[string]string
					if err := json.Unmarshal([]byte(f[2]), &b); err != nil {
						t.Fatalf("PUT body %q: %v", f[2], err)
					}
					content, _ := base64.StdEncoding.DecodeString(b["content"])
					out = append(out, forgeCall{Op: "write", Path: f[1], Ref: b["branch"], SHA: b["sha"], Content: string(content)})
				case "login":
					out = append(out, forgeCall{Op: "login"})
				default:
					t.Fatalf("gh fake saw %q", line)
				}
			}
			return out
		},
	}
}

// giteaBackend records requests on an httptest server shaped like Gitea's API v1.
func giteaBackend(t *testing.T) forgeBackend {
	var mu sync.Mutex
	var calls []forgeCall
	conflict := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		const repo = "/api/v1/repos/o/r"
		switch {
		case r.Method == http.MethodGet && r.URL.Path == repo+"/branches/"+fakeBranch:
			calls = append(calls, forgeCall{Op: "head", Ref: fakeBranch})
			_, _ = io.WriteString(w, `{"commit":{"id":"`+fakeHead+`"}}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, repo+"/branches/"):
			http.Error(w, `{"message":"branch not found"}`, http.StatusNotFound)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, repo+"/contents/"):
			p := strings.TrimPrefix(r.URL.Path, repo+"/contents/")
			calls = append(calls, forgeCall{Op: "read", Path: p, Ref: r.URL.Query().Get("ref")})
			f, ok := fakeTree[p]
			if !ok {
				http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"content": base64.StdEncoding.EncodeToString([]byte(f.content)), "encoding": "base64", "sha": f.sha})
		case r.Method == http.MethodPost && r.URL.Path == repo+"/contents":
			var body struct {
				Branch string `json:"branch"`
				Files  []struct {
					Operation, Path, Content, SHA string
				} `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("POST contents body: %v", err)
			}
			stale := false
			for _, f := range body.Files {
				content, _ := base64.StdEncoding.DecodeString(f.Content)
				if f.Operation != "update" {
					t.Errorf("operation = %q, want update", f.Operation)
				}
				calls = append(calls, forgeCall{Op: "write", Path: f.Path, Ref: body.Branch, SHA: f.SHA, Content: string(content)})
				stale = stale || f.Path == conflict
			}
			if stale {
				http.Error(w, `{"message":"sha does not match"}`, http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/user":
			calls = append(calls, forgeCall{Op: "login"})
			_, _ = io.WriteString(w, `{"login":"ann"}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "", http.StatusTeapot)
		}
	}))
	t.Cleanup(srv.Close)
	files, err := NewForgeFiles("", ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"})
	if err != nil {
		t.Fatal(err)
	}
	return forgeBackend{
		files:    files,
		conflict: func(p string) { mu.Lock(); conflict = p; mu.Unlock() },
		calls: func() []forgeCall {
			mu.Lock()
			defer mu.Unlock()
			return append([]forgeCall(nil), calls...)
		},
	}
}

var forgeBackends = map[string]func(*testing.T) forgeBackend{"github": githubBackend, "gitea": giteaBackend}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestForgeFilesReadAtHeadAndWriteWithReadSHA(t *testing.T) {
	for name, backend := range forgeBackends {
		t.Run(name, func(t *testing.T) {
			b := backend(t)
			ctx := context.Background()
			head, err := b.files.Head(ctx, fakeBranch)
			if err != nil || head != fakeHead {
				t.Fatalf("Head = %q, %v; want %s", head, err, fakeHead)
			}
			prop, err := b.files.Read(ctx, head, "openspec/changes/alpha/proposal.md")
			if err != nil || string(prop.Content) != fakeTree[prop.Path].content || prop.SHA != "blob-proposal" {
				t.Fatalf("Read proposal = %+v, %v", prop, err)
			}
			design, err := b.files.Read(ctx, head, "openspec/changes/alpha/design.md")
			if err != nil || !design.Missing {
				t.Fatalf("Read design = %+v, %v; want Missing", design, err)
			}
			login, err := b.files.Login(ctx)
			if err != nil || login != "ann" {
				t.Fatalf("Login = %q, %v", login, err)
			}
			prop.Content = []byte("approved\n")
			tasks := ForgeFile{Path: "openspec/changes/alpha/tasks.md", Content: []byte("done\n"), SHA: "blob-tasks"}
			if err := b.files.Write(ctx, fakeBranch, "msg", []ForgeFile{prop, tasks}); err != nil {
				t.Fatalf("Write: %v", err)
			}
			want := []forgeCall{
				{Op: "head", Ref: fakeBranch},
				{Op: "read", Path: "openspec/changes/alpha/proposal.md", Ref: fakeHead},
				{Op: "read", Path: "openspec/changes/alpha/design.md", Ref: fakeHead},
				{Op: "login"},
				{Op: "write", Path: "openspec/changes/alpha/proposal.md", Ref: fakeBranch, SHA: "blob-proposal", Content: "approved\n"},
				{Op: "write", Path: "openspec/changes/alpha/tasks.md", Ref: fakeBranch, SHA: "blob-tasks", Content: "done\n"},
			}
			got := b.calls()
			if len(got) != len(want) {
				t.Fatalf("calls = %+v\nwant %+v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("call %d = %+v, want %+v", i, got[i], want[i])
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestForgeFilesStaleSHAIsConflict(t *testing.T) {
	for name, backend := range forgeBackends {
		t.Run(name, func(t *testing.T) {
			b := backend(t)
			b.conflict("openspec/changes/alpha/tasks.md")
			files := []ForgeFile{
				{Path: "openspec/changes/alpha/proposal.md", Content: []byte("p"), SHA: "blob-proposal"},
				{Path: "openspec/changes/alpha/tasks.md", Content: []byte("t"), SHA: "old"},
			}
			err := b.files.Write(context.Background(), fakeBranch, "msg", files)
			if !errors.Is(err, ErrForgeConflict) {
				t.Fatalf("Write error = %v, want ErrForgeConflict", err)
			}
			// GitHub commits per file, so the proposal has landed and the error must say so (#181 D2).
			if name == "github" && !strings.Contains(err.Error(), "wrote openspec/changes/alpha/proposal.md") {
				t.Errorf("Write error = %q, want it to name the proposal already written", err)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestForgeFilesUnknownBranch(t *testing.T) {
	for name, backend := range forgeBackends {
		t.Run(name, func(t *testing.T) {
			b := backend(t)
			if _, err := b.files.Head(context.Background(), "claude/missing"); !errors.Is(err, ErrForgeNotFound) {
				t.Fatalf("Head on an unknown branch = %v, want ErrForgeNotFound", err)
			}
		})
	}
}
