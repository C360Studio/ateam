package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// fakeFiles serves one change at one head and records the writes.
type fakeFiles struct {
	tree     map[string]state.ForgeFile
	writeErr error
	calls    int
	writes   []state.ForgeFile
	branch   string
}

func (f *fakeFiles) Head(_ context.Context, branch string) (string, error) {
	f.calls++
	if branch != "claude/alpha" {
		return "", fmt.Errorf("branch %s: %w", branch, state.ErrForgeNotFound)
	}
	return "headsha", nil
}

func (f *fakeFiles) Read(_ context.Context, ref, path string) (state.ForgeFile, error) {
	f.calls++
	if ref != "headsha" {
		return state.ForgeFile{}, fmt.Errorf("read %s at %q, want the head commit", path, ref)
	}
	if file, ok := f.tree[path]; ok {
		return file, nil
	}
	return state.ForgeFile{Path: path, Missing: true}, nil
}

func (f *fakeFiles) Write(_ context.Context, branch, _ string, files []state.ForgeFile) error {
	f.calls++
	f.branch = branch
	f.writes = files
	return f.writeErr
}

func (f *fakeFiles) Login(context.Context) (string, error) {
	f.calls++
	return "Ann Owner", nil
}

const (
	approveProposal = "# Alpha\n\nApproved-by:\n\n## Why\n\nBecause.\n"
	approveDesign   = "# Design\n\nD1.\n"
	approveTasks    = "## 1. Approval\n\n- [ ] 1.1 Owner approves. Hold: OPEN-Q proposal awaits owner approval\n- [ ] 1.2 Build it.\n"
	// approveDigest is sha256 over "proposal.md <len>\n<proposal without its Approved-by line>design.md <len>\n<design>",
	// the #272 format, computed outside Go for these fixtures.
	approveDigest = "1cb65e42708c"
)

func approveTree(tasks string) map[string]state.ForgeFile {
	return map[string]state.ForgeFile{
		"openspec/changes/alpha/proposal.md": {Path: "openspec/changes/alpha/proposal.md", Content: []byte(approveProposal), SHA: "blob-p"},
		"openspec/changes/alpha/design.md":   {Path: "openspec/changes/alpha/design.md", Content: []byte(approveDesign), SHA: "blob-d"},
		"openspec/changes/alpha/tasks.md":    {Path: "openspec/changes/alpha/tasks.md", Content: []byte(tasks), SHA: "blob-t"},
	}
}

func approveServer(t *testing.T, files *fakeFiles) (*Server, http.Handler) {
	t.Helper()
	srv, err := NewServer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv.forgeFiles = func(string, state.ForgeConfig) (state.ForgeFiles, error) { return files, nil }
	srv.now = func() time.Time { return time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC) }
	return srv, srv.Handler()
}

func postApprove(srv *Server, h http.Handler, body string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/openspec/approve", bytes.NewReader([]byte(body)))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ATeam-Repo", srv.currentRepo())
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestApproveWritesBoundHeaderAndDropsHold(t *testing.T) {
	files := &fakeFiles{tree: approveTree(approveTasks)}
	srv, h := approveServer(t, files)
	rec := postApprove(srv, h, `{"change":"alpha","branch":"claude/alpha"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve = %d: %s", rec.Code, rec.Body.String())
	}
	if files.branch != "claude/alpha" || len(files.writes) != 2 {
		t.Fatalf("writes on %q = %+v, want proposal and tasks on claude/alpha", files.branch, files.writes)
	}
	p, tk := files.writes[0], files.writes[1]
	if p.Path != "openspec/changes/alpha/proposal.md" || p.SHA != "blob-p" || tk.Path != "openspec/changes/alpha/tasks.md" || tk.SHA != "blob-t" {
		t.Errorf("write order or blob shas = %s@%s, %s@%s; want proposal.md@blob-p then tasks.md@blob-t", p.Path, p.SHA, tk.Path, tk.SHA)
	}
	wantP := strings.Replace(approveProposal, "Approved-by:\n", "Approved-by: Ann Owner (2026-11-10) sha256:"+approveDigest+"\n", 1)
	if string(p.Content) != wantP {
		t.Errorf("proposal =\n%s\nwant\n%s", p.Content, wantP)
	}
	if strings.Contains(string(tk.Content), "Hold: OPEN-Q") || !strings.Contains(string(tk.Content), "- [ ] 1.2 Build it.") {
		t.Errorf("tasks = %q, want the approval hold gone and the other task kept", tk.Content)
	}
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["approved_by"] != "Ann Owner" || body["head"] != "headsha" {
		t.Errorf("response = %v", body)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestApproveRefusals(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		tasks     string
		writeErr  error
		mutate    func(*http.Request)
		want      int
		forgeUsed bool
		wrote     bool
	}{
		{name: "stale blob", body: `{"change":"alpha","branch":"claude/alpha"}`, tasks: approveTasks,
			writeErr: fmt.Errorf("tasks.md: %w", state.ErrForgeConflict), want: http.StatusConflict, forgeUsed: true, wrote: true},
		{name: "no approval hold", body: `{"change":"alpha","branch":"claude/alpha"}`, tasks: "- [x] 1.1 Done.\n",
			want: http.StatusBadRequest, forgeUsed: true},
		{name: "default branch", body: `{"change":"alpha","branch":"main"}`, tasks: approveTasks, want: http.StatusBadRequest},
		{name: "unknown branch", body: `{"change":"alpha","branch":"claude/gone"}`, tasks: approveTasks,
			want: http.StatusBadRequest, forgeUsed: true},
		{name: "path in change name", body: `{"change":"../alpha","branch":"claude/alpha"}`, tasks: approveTasks, want: http.StatusBadRequest},
		{name: "cross origin", body: `{"change":"alpha","branch":"claude/alpha"}`, tasks: approveTasks,
			mutate: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, want: http.StatusForbidden},
		{name: "other forge error", body: `{"change":"alpha","branch":"claude/alpha"}`, tasks: approveTasks,
			writeErr: errors.New("HTTP 500"), want: http.StatusBadGateway, forgeUsed: true, wrote: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := &fakeFiles{tree: approveTree(tc.tasks), writeErr: tc.writeErr}
			srv, h := approveServer(t, files)
			rec := postApprove(srv, h, tc.body, tc.mutate)
			if rec.Code != tc.want {
				t.Fatalf("approve = %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
			if (files.calls > 0) != tc.forgeUsed {
				t.Errorf("forge calls = %d, want used=%v", files.calls, tc.forgeUsed)
			}
			if (files.writes != nil) != tc.wrote {
				t.Errorf("writes = %+v, want wrote=%v", files.writes, tc.wrote)
			}
		})
	}
}
