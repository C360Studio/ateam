package state

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	Op      string // head, read, write, login; GitHub also base, commit, ref
	Path    string
	Ref     string
	SHA     string
	Content string
}

// forgeBackend is a recorded forge serving fakeTree at fakeHead. moved makes the forge behave as if design.md
// changed after the read: GitHub's branch is no longer at fakeHead, and Gitea's design.md sha no longer matches.
type forgeBackend struct {
	files ForgeFiles
	calls func() []forgeCall
	moved func()
}

const (
	fakeHead   = "1111111111111111111111111111111111111111"
	fakeBranch = "claude/alpha"
)

var fakeTree = map[string]struct{ content, sha string }{
	"openspec/changes/alpha/proposal.md": {"# Alpha\n\nApproved-by:\n", "blob-proposal"},
	"openspec/changes/alpha/tasks.md":    {"- [ ] 1.1 Approve. Hold: OPEN-Q proposal awaits owner approval\n", "blob-tasks"},
	"openspec/changes/alpha/design.md":   {"# Design\n", "blob-design"},
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
	t.Setenv("FAKE_MOVED", "")
	t.Setenv("GH_TOKEN", "")
	writeFake(t, filepath.Join(dir, "gh"), `
case "$*" in
"api repos/o/r/git/ref/heads/`+fakeBranch+`") echo "head|`+fakeBranch+`" >> "$FORGE_CALL_LOG"; echo '{"object":{"sha":"`+fakeHead+`"}}';;
"api repos/o/r/git/ref/heads/"*) echo '{"message":"Not Found","status":"404"}'; exit 1;;
"api repos/o/r/contents/"*"?ref="*)
  rest=${2#repos/o/r/contents/}; p=${rest%%\?ref=*}; ref=${rest#*\?ref=}
  echo "read|$p|$ref" >> "$FORGE_CALL_LOG"
  if [ -f "$FAKE_TREE/$p" ]; then cat "$FAKE_TREE/$p"; else echo '{"message":"Not Found","status":"404"}'; exit 1; fi;;
"api repos/o/r/git/commits/"*) echo "base|${2#repos/o/r/git/commits/}" >> "$FORGE_CALL_LOG"; echo '{"tree":{"sha":"tree-base"}}';;
"api -X POST repos/o/r/git/trees --input -") printf '%s|%s\n' tree "$(cat)" >> "$FORGE_CALL_LOG"; echo '{"sha":"tree-new"}';;
"api -X POST repos/o/r/git/commits --input -") printf '%s|%s\n' commit "$(cat)" >> "$FORGE_CALL_LOG"; echo '{"sha":"commit-new"}';;
"api -X PATCH repos/o/r/git/refs/heads/`+fakeBranch+` --input -")
  printf '%s|%s\n' ref "$(cat)" >> "$FORGE_CALL_LOG"
  if [ -n "$FAKE_MOVED" ]; then echo '{"message":"Update is not a fast forward","status":"422"}'; exit 1; fi
  echo '{"object":{"sha":"commit-new"}}';;
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
		files: files,
		moved: func() { t.Setenv("FAKE_MOVED", "1") },
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
				case "base":
					out = append(out, forgeCall{Op: "base", Ref: f[1]})
				case "tree":
					var b struct {
						BaseTree string `json:"base_tree"`
						Tree     []struct{ Path, Mode, Type, Content string }
					}
					if err := json.Unmarshal([]byte(strings.Join(f[1:], "|")), &b); err != nil {
						t.Fatalf("tree body %q: %v", line, err)
					}
					for _, e := range b.Tree {
						if b.BaseTree != "tree-base" || e.Mode != "100644" || e.Type != "blob" {
							t.Errorf("tree entry %+v on base %q, want a 100644 blob on tree-base", e, b.BaseTree)
						}
						out = append(out, forgeCall{Op: "write", Path: e.Path, Content: e.Content})
					}
				case "commit":
					var b struct {
						Tree    string   `json:"tree"`
						Parents []string `json:"parents"`
					}
					if err := json.Unmarshal([]byte(strings.Join(f[1:], "|")), &b); err != nil || len(b.Parents) != 1 {
						t.Fatalf("commit body %q: %v", line, err)
					}
					out = append(out, forgeCall{Op: "commit", Ref: b.Parents[0], SHA: b.Tree})
				case "ref":
					var b struct {
						SHA   string `json:"sha"`
						Force bool   `json:"force"`
					}
					if err := json.Unmarshal([]byte(strings.Join(f[1:], "|")), &b); err != nil {
						t.Fatalf("ref body %q: %v", line, err)
					}
					out = append(out, forgeCall{Op: "ref", Ref: fakeBranch, SHA: b.SHA, Content: fmt.Sprintf("force=%v", b.Force)})
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
	moved := false
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
				stale = stale || (moved && strings.HasSuffix(f.Path, "/design.md"))
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
		files: files,
		moved: func() { mu.Lock(); moved = true; mu.Unlock() },
		calls: func() []forgeCall {
			mu.Lock()
			defer mu.Unlock()
			return append([]forgeCall(nil), calls...)
		},
	}
}

var forgeBackends = map[string]func(*testing.T) forgeBackend{"github": githubBackend, "gitea": giteaBackend}

func approvalWrite(t *testing.T, b forgeBackend) (files, unchanged []ForgeFile) {
	t.Helper()
	ctx := context.Background()
	read := map[string]ForgeFile{}
	for _, name := range []string{"proposal.md", "tasks.md", "design.md"} {
		f, err := b.files.Read(ctx, fakeHead, "openspec/changes/alpha/"+name)
		if err != nil || f.Missing {
			t.Fatalf("Read %s = %+v, %v", name, f, err)
		}
		read[name] = f
	}
	p, tk := read["proposal.md"], read["tasks.md"]
	p.Content, tk.Content = []byte("approved\n"), []byte("done\n")
	return []ForgeFile{p, tk}, []ForgeFile{read["design.md"]}
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestForgeFilesReadAtHeadAndWriteOneCommit(t *testing.T) {
	for name, backend := range forgeBackends {
		t.Run(name, func(t *testing.T) {
			b := backend(t)
			ctx := context.Background()
			head, err := b.files.Head(ctx, fakeBranch)
			if err != nil || head != fakeHead {
				t.Fatalf("Head = %q, %v; want %s", head, err, fakeHead)
			}
			absent, err := b.files.Read(ctx, head, "openspec/changes/alpha/notes.md")
			if err != nil || !absent.Missing {
				t.Fatalf("Read absent file = %+v, %v; want Missing", absent, err)
			}
			login, err := b.files.Login(ctx)
			if err != nil || login != "ann" {
				t.Fatalf("Login = %q, %v", login, err)
			}
			files, unchanged := approvalWrite(t, b)
			if err := b.files.Write(ctx, fakeBranch, head, "msg", files, unchanged); err != nil {
				t.Fatalf("Write: %v", err)
			}
			var reads, writes []forgeCall
			var guard []forgeCall
			for _, c := range b.calls() {
				switch {
				case c.Op == "read":
					if c.Ref != fakeHead {
						t.Errorf("read %s at %q, want the head commit", c.Path, c.Ref)
					}
					reads = append(reads, c)
				case c.Op == "write" && strings.HasSuffix(c.Path, "/design.md"):
					guard = append(guard, c)
				case c.Op == "write":
					writes = append(writes, forgeCall{Op: c.Op, Path: c.Path, Content: c.Content})
				case c.Op == "base", c.Op == "commit", c.Op == "ref":
					guard = append(guard, c)
				}
			}
			wantWrites := []forgeCall{
				{Op: "write", Path: "openspec/changes/alpha/proposal.md", Content: "approved\n"},
				{Op: "write", Path: "openspec/changes/alpha/tasks.md", Content: "done\n"},
			}
			if len(writes) != 2 || writes[0] != wantWrites[0] || writes[1] != wantWrites[1] {
				t.Errorf("written files = %+v, want %+v", writes, wantWrites)
			}
			// The guard that makes the commit fail when the approved revision moved, per forge.
			var wantGuard []forgeCall
			if name == "github" {
				wantGuard = []forgeCall{
					{Op: "base", Ref: fakeHead},
					{Op: "commit", Ref: fakeHead, SHA: "tree-new"},
					{Op: "ref", Ref: fakeBranch, SHA: "commit-new", Content: "force=false"},
				}
			} else {
				wantGuard = []forgeCall{{Op: "write", Path: "openspec/changes/alpha/design.md", Ref: fakeBranch, SHA: "blob-design", Content: "# Design\n"}}
			}
			if len(guard) != len(wantGuard) {
				t.Fatalf("guard calls = %+v, want %+v", guard, wantGuard)
			}
			for i := range wantGuard {
				if guard[i] != wantGuard[i] {
					t.Errorf("guard call %d = %+v, want %+v", i, guard[i], wantGuard[i])
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestForgeFilesDesignMovedIsConflict(t *testing.T) {
	for name, backend := range forgeBackends {
		t.Run(name, func(t *testing.T) {
			b := backend(t)
			files, unchanged := approvalWrite(t, b)
			b.moved()
			err := b.files.Write(context.Background(), fakeBranch, fakeHead, "msg", files, unchanged)
			if !errors.Is(err, ErrForgeConflict) {
				t.Fatalf("Write after design.md moved = %v, want ErrForgeConflict", err)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestGiteaFilesAbsentInputThatAppearsIsConflict(t *testing.T) {
	b := giteaBackend(t)
	files, _ := approvalWrite(t, b)
	// design.md read as absent, but the branch has one now.
	appeared := []ForgeFile{{Path: "openspec/changes/alpha/design.md", Missing: true}}
	if err := b.files.Write(context.Background(), fakeBranch, fakeHead, "msg", files, appeared); !errors.Is(err, ErrForgeConflict) {
		t.Fatalf("Write = %v, want ErrForgeConflict", err)
	}
	for _, c := range b.calls() {
		if c.Op == "write" {
			t.Fatalf("wrote %+v after the absent input appeared", c)
		}
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
