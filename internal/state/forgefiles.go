package state

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrForgeConflict reports that a file changed on the forge after it was read: the contents API rejected the
// blob sha the write sent (#181 D3).
var ErrForgeConflict = errors.New("file changed on the forge since it was read")

// ErrForgeNotFound reports a branch the forge does not have.
var ErrForgeNotFound = errors.New("not found on the forge")

// ForgeFile is one file read at a commit with the blob sha the contents API needs to replace it. Missing is set
// when the commit has no such file.
type ForgeFile struct {
	Path    string
	Content []byte
	SHA     string
	Missing bool
}

// ForgeFiles reads and writes repository files through the forge, never a local worktree (#181 D1 A).
type ForgeFiles interface {
	// Head resolves a branch to its head commit sha.
	Head(ctx context.Context, branch string) (string, error)
	// Read returns path at commit ref.
	Read(ctx context.Context, ref, path string) (ForgeFile, error)
	// Write commits files on branch in one commit and returns ErrForgeConflict, writing nothing, when the
	// approved revision moved after it was read at base: on GitHub when the branch is no longer at base, on
	// Gitea when any of files or unchanged no longer has its read blob sha (#181 review, OQ2 revised).
	Write(ctx context.Context, branch, base, message string, files, unchanged []ForgeFile) error
	// Login names the account behind the write token.
	Login(ctx context.Context) (string, error)
}

const forgeFileTimeout = 15 * time.Second

// NewForgeFiles returns the GitHub or Gitea file client for cfg; root is the checkout gh resolves {owner}/{repo}
// from when cfg names none.
func NewForgeFiles(root string, cfg ForgeConfig) (ForgeFiles, error) {
	switch cfg.Provider {
	case ForgeGitHub:
		repo := "{owner}/{repo}"
		if cfg.Owner != "" || cfg.Repo != "" {
			validated, err := normalizeForgeConfig(cfg)
			if err != nil {
				return nil, err
			}
			repo = validated.Owner + "/" + validated.Repo
		}
		return ghFiles{root: root, repo: repo}, nil
	case ForgeGitea:
		validated, err := normalizeForgeConfig(cfg)
		if err != nil {
			return nil, err
		}
		return giteaFiles{
			base:   strings.TrimRight(validated.BaseURL, "/") + "/api/v1",
			repo:   "/repos/" + validated.Owner + "/" + validated.Repo,
			client: &http.Client{Timeout: forgeFileTimeout},
		}, nil
	}
	return nil, fmt.Errorf("forge %q cannot write files", cfg.Provider)
}

// contentsFile is the contents API answer for one file on both forges.
type contentsFile struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	SHA      string `json:"sha"`
}

func decodeContents(path string, data []byte) (ForgeFile, error) {
	var c contentsFile
	if err := json.Unmarshal(data, &c); err != nil || c.SHA == "" {
		return ForgeFile{}, fmt.Errorf("%s: unreadable contents answer", path)
	}
	if c.Encoding != "base64" {
		return ForgeFile{}, fmt.Errorf("%s: unexpected encoding %q", path, c.Encoding)
	}
	// GitHub wraps the base64 at 60 columns.
	content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(c.Content, "\n", ""))
	if err != nil {
		return ForgeFile{}, fmt.Errorf("%s: undecodable content", path)
	}
	return ForgeFile{Path: path, Content: content, SHA: c.SHA}, nil
}

// escapePath escapes each segment of a repository path for a URL, keeping the slashes.
func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

type ghFiles struct {
	root string
	repo string
}

// api runs gh api with the role token and returns the body and the HTTP status gh reports. gh prints an error
// body with a "status" field on stdout and exits 1, so a non-2xx answer is read from there.
func (g ghFiles) api(ctx context.Context, token string, stdin []byte, args ...string) ([]byte, int, error) {
	cctx, cancel := context.WithTimeout(ctx, forgeFileTimeout)
	defer cancel()
	cmd := ghCommand(cctx, token, append([]string{"api"}, args...)...)
	cmd.Dir = g.root
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	out, err := cmd.Output()
	if err == nil {
		return out, http.StatusOK, nil
	}
	var body struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(out, &body) == nil && body.Status != "" {
		var code int
		if _, scanErr := fmt.Sscanf(body.Status, "%d", &code); scanErr == nil {
			return out, code, nil
		}
	}
	return nil, 0, ghRunError(err)
}

func (g ghFiles) Head(ctx context.Context, branch string) (string, error) {
	out, code, err := g.api(ctx, ReadToken(), nil, "repos/"+g.repo+"/git/ref/heads/"+escapePath(branch))
	if err != nil {
		return "", err
	}
	if code == http.StatusNotFound {
		return "", fmt.Errorf("branch %s: %w", branch, ErrForgeNotFound)
	}
	if code != http.StatusOK {
		return "", fmt.Errorf("branch %s: HTTP %d", branch, code)
	}
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(out, &ref); err != nil || ref.Object.SHA == "" {
		return "", fmt.Errorf("branch %s: unreadable ref", branch)
	}
	return ref.Object.SHA, nil
}

func (g ghFiles) Read(ctx context.Context, ref, path string) (ForgeFile, error) {
	out, code, err := g.api(ctx, ReadToken(), nil, "repos/"+g.repo+"/contents/"+escapePath(path)+"?ref="+url.QueryEscape(ref))
	if err != nil {
		return ForgeFile{}, fmt.Errorf("%s: %w", path, err)
	}
	if code == http.StatusNotFound {
		return ForgeFile{Path: path, Missing: true}, nil
	}
	if code != http.StatusOK {
		return ForgeFile{}, fmt.Errorf("%s: HTTP %d", path, code)
	}
	return decodeContents(path, out)
}

func (g ghFiles) Write(ctx context.Context, branch, base, message string, files, _ []ForgeFile) error {
	// One commit on top of base through the git data API; the non-force ref update is refused unless the branch
	// is still at base, so any push after the read (design.md included) is a conflict and nothing lands.
	out, code, err := g.api(ctx, ReadToken(), nil, "repos/"+g.repo+"/git/commits/"+base)
	if err != nil || code != http.StatusOK {
		return ghWriteError("read base commit", code, err)
	}
	var commit struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if err := json.Unmarshal(out, &commit); err != nil || commit.Tree.SHA == "" {
		return fmt.Errorf("read base commit: unreadable answer")
	}
	type entry struct {
		Path    string `json:"path"`
		Mode    string `json:"mode"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	tree := struct {
		BaseTree string  `json:"base_tree"`
		Tree     []entry `json:"tree"`
	}{BaseTree: commit.Tree.SHA}
	for _, f := range files {
		tree.Tree = append(tree.Tree, entry{Path: f.Path, Mode: "100644", Type: "blob", Content: string(f.Content)})
	}
	newTree, err := g.create(ctx, "git/trees", tree)
	if err != nil {
		return fmt.Errorf("create tree: %w", err)
	}
	newCommit, err := g.create(ctx, "git/commits", map[string]any{"message": message, "tree": newTree, "parents": []string{base}})
	if err != nil {
		return fmt.Errorf("create commit: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{"sha": newCommit, "force": false})
	_, code, err = g.api(ctx, WriteToken(), payload, "-X", "PATCH", "repos/"+g.repo+"/git/refs/heads/"+escapePath(branch), "--input", "-")
	if err == nil && code == http.StatusUnprocessableEntity { // "Update is not a fast forward"
		return fmt.Errorf("branch %s moved after %s: %w", branch, sha7(base), ErrForgeConflict)
	}
	if err != nil || code != http.StatusOK {
		return ghWriteError("update branch", code, err)
	}
	return nil
}

// create POSTs a git data object and returns its sha.
func (g ghFiles) create(ctx context.Context, kind string, body any) (string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	out, code, err := g.api(ctx, WriteToken(), payload, "-X", "POST", "repos/"+g.repo+"/"+kind, "--input", "-")
	if err != nil {
		return "", err
	}
	if code != http.StatusOK && code != http.StatusCreated {
		return "", fmt.Errorf("HTTP %d", code)
	}
	var obj struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(out, &obj); err != nil || obj.SHA == "" {
		return "", fmt.Errorf("unreadable answer")
	}
	return obj.SHA, nil
}

func ghWriteError(step string, code int, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	return fmt.Errorf("%s: HTTP %d", step, code)
}

func (g ghFiles) Login(ctx context.Context) (string, error) {
	out, code, err := g.api(ctx, WriteToken(), nil, "user")
	if err != nil {
		return "", err
	}
	if code != http.StatusOK {
		return "", fmt.Errorf("login: HTTP %d", code)
	}
	return decodeLogin(out)
}

func decodeLogin(data []byte) (string, error) {
	var u struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(data, &u); err != nil || strings.TrimSpace(u.Login) == "" {
		return "", fmt.Errorf("login: unreadable user")
	}
	return u.Login, nil
}

type giteaFiles struct {
	base   string // https://host/api/v1
	repo   string // /repos/<owner>/<repo>
	client *http.Client
}

func (g giteaFiles) do(ctx context.Context, method, path, token string, body []byte) ([]byte, int, error) {
	cctx, cancel := context.WithTimeout(ctx, forgeFileTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, method, g.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("bad request")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("unreachable")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("unreadable response")
	}
	return data, resp.StatusCode, nil
}

func (g giteaFiles) Head(ctx context.Context, branch string) (string, error) {
	// shortcut: the branch goes into the path unescaped, as Gitea's router matches slashes in branch names; escape it
	// if a branch with other reserved characters is ever approved.
	data, code, err := g.do(ctx, http.MethodGet, g.repo+"/branches/"+branch, giteaToken(ReadToken()), nil)
	if err != nil {
		return "", err
	}
	if code == http.StatusNotFound {
		return "", fmt.Errorf("branch %s: %w", branch, ErrForgeNotFound)
	}
	if code != http.StatusOK {
		return "", fmt.Errorf("branch %s: HTTP %d", branch, code)
	}
	var b struct {
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &b); err != nil || b.Commit.ID == "" {
		return "", fmt.Errorf("branch %s: unreadable branch", branch)
	}
	return b.Commit.ID, nil
}

func (g giteaFiles) Read(ctx context.Context, ref, path string) (ForgeFile, error) {
	data, code, err := g.do(ctx, http.MethodGet, g.repo+"/contents/"+escapePath(path)+"?ref="+url.QueryEscape(ref), giteaToken(ReadToken()), nil)
	if err != nil {
		return ForgeFile{}, fmt.Errorf("%s: %w", path, err)
	}
	if code == http.StatusNotFound {
		return ForgeFile{Path: path, Missing: true}, nil
	}
	if code != http.StatusOK {
		return ForgeFile{}, fmt.Errorf("%s: HTTP %d", path, code)
	}
	return decodeContents(path, data)
}

func (g giteaFiles) Write(ctx context.Context, branch, _, message string, files, unchanged []ForgeFile) error {
	type change struct {
		Operation string `json:"operation"`
		Path      string `json:"path"`
		Content   string `json:"content"`
		SHA       string `json:"sha"`
	}
	req := struct {
		Branch  string   `json:"branch"`
		Message string   `json:"message"`
		Files   []change `json:"files"`
	}{Branch: branch, Message: message}
	var names []string
	for _, f := range files {
		req.Files = append(req.Files, change{Operation: "update", Path: f.Path, Content: base64.StdEncoding.EncodeToString(f.Content), SHA: f.SHA})
		names = append(names, f.Path)
	}
	// Gitea's contents API takes no base commit, so each unchanged input rides along as an identical update
	// whose sha the server checks in the same commit.
	for _, f := range unchanged {
		if f.Missing {
			// shortcut: an absent input (no design.md) is rechecked here, not in the commit, so one added between
			// this read and the write lands unguarded; spec:check then fails on the stale digest. Close it if Gitea
			// gains a base-commit option for POST contents.
			now, err := g.Head(ctx, branch)
			if err != nil {
				return err
			}
			got, err := g.Read(ctx, now, f.Path)
			if err != nil {
				return err
			}
			if !got.Missing {
				return fmt.Errorf("%s appeared after the read: %w", f.Path, ErrForgeConflict)
			}
			continue
		}
		req.Files = append(req.Files, change{Operation: "update", Path: f.Path, Content: base64.StdEncoding.EncodeToString(f.Content), SHA: f.SHA})
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, code, err := g.do(ctx, http.MethodPost, g.repo+"/contents", giteaToken(WriteToken()), payload)
	switch {
	case err != nil:
	// Gitea's status for a stale sha is unverified (#181 design Risks); both mean the write was refused unchanged.
	case code == http.StatusConflict || code == http.StatusUnprocessableEntity:
		err = ErrForgeConflict
	case code != http.StatusOK && code != http.StatusCreated:
		err = fmt.Errorf("HTTP %d", code)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", strings.Join(names, ", "), err)
	}
	return nil
}

func (g giteaFiles) Login(ctx context.Context) (string, error) {
	data, code, err := g.do(ctx, http.MethodGet, "/user", giteaToken(WriteToken()), nil)
	if err != nil {
		return "", err
	}
	if code != http.StatusOK {
		return "", fmt.Errorf("login: HTTP %d", code)
	}
	return decodeLogin(data)
}
