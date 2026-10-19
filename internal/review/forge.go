package review

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ForgeConfig explicitly identifies the API and credentials used for evidence reads.
// BaseURL is the GitHub API root or the Gitea installation root.
type ForgeConfig struct {
	Provider string
	BaseURL  string
	Owner    string
	Repo     string
	Token    string
}

// PullRequest contains the current immutable revision identities and draft state.
type PullRequest struct {
	Base   string
	Head   string
	Draft  bool
	Author string
}

// Comment contains the forge-authenticated author and exact, untrusted body.
type Comment struct {
	Author    string
	Body      string
	CreatedAt string
	UpdatedAt string
}

var forgeSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
var forgeSegment = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

const maxForgeResponse = 1 << 20

// ReadPullRequest reads current revision identities without executing forge CLIs.
func ReadPullRequest(ctx context.Context, client *http.Client, cfg ForgeConfig, number int) (PullRequest, error) {
	if number <= 0 {
		return PullRequest{}, fmt.Errorf("pull request number must be positive")
	}
	root, err := forgeRoot(cfg)
	if err != nil {
		return PullRequest{}, err
	}
	var payload struct {
		Number int    `json:"number"`
		State  string `json:"state"`
		Merged *bool  `json:"merged"`
		Base   struct {
			SHA  string `json:"sha"`
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
		Draft *bool `json:"draft"`
		User  struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := readForgeJSON(ctx, client, cfg, fmt.Sprintf("%s/pulls/%d", root, number), &payload); err != nil {
		return PullRequest{}, err
	}
	if payload.State != "open" || payload.Merged == nil || *payload.Merged || payload.Number != number || !forgeSHA.MatchString(payload.Base.SHA) || !forgeSHA.MatchString(payload.Head.SHA) || payload.Draft == nil || strings.TrimSpace(payload.User.Login) == "" {
		return PullRequest{}, fmt.Errorf("pull request response has missing or invalid open state, identity, revisions, author, or draft state")
	}
	if err := verifyDefaultBase(ctx, client, cfg, root, payload.Base.Ref, payload.Base.Repo.FullName, payload.Base.SHA); err != nil {
		return PullRequest{}, err
	}
	return PullRequest{Base: payload.Base.SHA, Head: payload.Head.SHA, Draft: *payload.Draft, Author: payload.User.Login}, nil
}

// verifyDefaultBase anchors policy lookup to the repository's current default
// branch. A PR-controlled target branch must never select its own trust policy.
func verifyDefaultBase(ctx context.Context, client *http.Client, cfg ForgeConfig, root, ref, repository, sha string) error {
	var repo struct {
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := readForgeJSON(ctx, client, cfg, root, &repo); err != nil {
		return err
	}
	expected := cfg.Owner + "/" + cfg.Repo
	if repo.FullName != expected || repository != expected || repo.DefaultBranch == "" || ref != repo.DefaultBranch {
		return fmt.Errorf("pull request must target the configured repository default branch")
	}
	for _, segment := range strings.Split(repo.DefaultBranch, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("invalid repository default branch")
		}
	}
	var branch struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
			ID  string `json:"id"`
		} `json:"commit"`
	}
	if err := readForgeJSON(ctx, client, cfg, root+"/branches/"+url.PathEscape(repo.DefaultBranch), &branch); err != nil {
		return err
	}
	current := branch.Commit.SHA
	if cfg.Provider == "gitea" {
		current = branch.Commit.ID
	}
	if branch.Name != repo.DefaultBranch || !forgeSHA.MatchString(current) || current != sha {
		return fmt.Errorf("pull request base does not match the current default branch revision")
	}
	return nil
}

// ReadComment retrieves one pinned comment and verifies its parent pull request.
// GitHub binds issue_url to the API parent; Gitea binds pull_request_url to
// the installation HTML pull URL. Known edits cannot prove the latest editor.
func ReadComment(ctx context.Context, client *http.Client, cfg ForgeConfig, prNumber, commentID int) (Comment, error) {
	if prNumber <= 0 || commentID <= 0 {
		return Comment{}, fmt.Errorf("pull request and comment IDs must be positive")
	}
	root, err := forgeRoot(cfg)
	if err != nil {
		return Comment{}, err
	}
	var payload struct {
		ID             int    `json:"id"`
		IssueURL       string `json:"issue_url"`
		PullRequestURL string `json:"pull_request_url"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
		Body           string `json:"body"`
		User           struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := readForgeJSON(ctx, client, cfg, fmt.Sprintf("%s/issues/comments/%d", root, commentID), &payload); err != nil {
		return Comment{}, err
	}
	parent := payload.IssueURL
	expectedParent := fmt.Sprintf("%s/issues/%d", root, prNumber)
	if cfg.Provider == "gitea" {
		parent = payload.PullRequestURL
		expectedParent = fmt.Sprintf("%s/%s/%s/pulls/%d", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo, prNumber)
	}
	if payload.ID != commentID || parent != expectedParent || strings.TrimSpace(payload.User.Login) == "" || strings.TrimSpace(payload.Body) == "" {
		return Comment{}, fmt.Errorf("comment response has missing or mismatched ID, pull request, author, or body")
	}
	created, createdErr := time.Parse(time.RFC3339Nano, payload.CreatedAt)
	updated, updatedErr := time.Parse(time.RFC3339Nano, payload.UpdatedAt)
	if createdErr != nil || updatedErr != nil || created.IsZero() || !created.Equal(updated) {
		return Comment{}, fmt.Errorf("comment is edited or lacks trustworthy creation/update timestamps; publish a new comment")
	}
	return Comment{Author: payload.User.Login, Body: payload.Body, CreatedAt: payload.CreatedAt, UpdatedAt: payload.UpdatedAt}, nil
}

func forgeRoot(cfg ForgeConfig) (string, error) {
	if cfg.Provider != "github" && cfg.Provider != "gitea" {
		return "", fmt.Errorf("unsupported review forge %q", cfg.Provider)
	}
	for _, segment := range []string{cfg.Owner, cfg.Repo} {
		if !forgeSegment.MatchString(segment) || segment == "." || segment == ".." {
			return "", fmt.Errorf("review forge requires valid owner and repository")
		}
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" || base.Opaque != "" {
		return "", fmt.Errorf("review forge requires an absolute base URL without credentials, query, or fragment")
	}
	// Cleartext is allowed only on literal loopback addresses for local fixtures.
	loopback := net.ParseIP(base.Hostname())
	if base.Scheme != "https" && !(base.Scheme == "http" && loopback != nil && loopback.IsLoopback()) {
		return "", fmt.Errorf("review forge requires HTTPS")
	}
	if strings.Contains(base.EscapedPath(), "%") {
		return "", fmt.Errorf("review forge base URL must not contain escaped path segments")
	}
	for _, segment := range strings.Split(base.Path, "/") {
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("review forge base URL contains relative path segments")
		}
	}
	root := strings.TrimRight(base.String(), "/")
	if cfg.Provider == "gitea" {
		root += "/api/v1"
	}
	return root + "/repos/" + cfg.Owner + "/" + cfg.Repo, nil
}

func readForgeJSON(ctx context.Context, client *http.Client, cfg ForgeConfig, endpoint string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create review evidence request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if cfg.Token != "" {
		scheme := "Bearer "
		if cfg.Provider == "gitea" {
			scheme = "token "
		}
		req.Header.Set("Authorization", scheme+cfg.Token)
	}
	if client == nil {
		client = http.DefaultClient
	}
	safeClient := *client
	// Even same-host redirects are unnecessary for pinned evidence and can change
	// the resource being authenticated. Never forward credentials on a redirect.
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := safeClient.Do(req)
	if err != nil {
		return fmt.Errorf("read review evidence: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("review evidence API returned HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxForgeResponse+1))
	if err != nil {
		return fmt.Errorf("read review evidence body: %w", err)
	}
	if len(raw) > maxForgeResponse {
		return fmt.Errorf("review evidence response exceeds %d bytes", maxForgeResponse)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode review evidence: %w", err)
	}
	return nil
}
