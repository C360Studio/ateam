package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/bootstrap"
	"github.com/c360studio/ateam/internal/review"
	"github.com/c360studio/ateam/internal/state"
)

func runReview(root string, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: ateam review setup --policy FILE | route [--developer-models IDS] | check --pr N --record-comment ID [--exception-comment ID] | compare --implementer AGENT:MODEL:EFFORT... --reviewer AGENT:MODEL:EFFORT | gate [PR]")
		return 2
	}
	var err error
	switch args[0] {
	case "compare":
		return reviewCompare(root, args[1:])
	case "gate":
		return reviewGate(root, args[1:])
	case "setup":
		err = reviewSetup(root, args[1:])
	case "route":
		err = reviewRoute(root, args[1:])
	case "check":
		err = reviewCheck(root, args[1:])
	default:
		err = fmt.Errorf("unknown review command %q", args[0])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 1
	}
	return 0
}

// runSpecs collects a repeatable agent:model:effort flag.
type runSpecs []review.Run

func (r *runSpecs) String() string { return fmt.Sprint(*r) }

func (r *runSpecs) Set(s string) error {
	run, err := review.ParseRunSpec(s)
	if err == nil {
		*r = append(*r, run)
	}
	return err
}

// reviewCompare exits 0 when the reviewer is not weaker than every implementer,
// 1 on a finding, and 2 when it cannot decide (ADR 0002 rule 2).
func reviewCompare(root string, args []string) int {
	fs := flag.NewFlagSet("review compare", flag.ContinueOnError)
	var implementers, reviewers runSpecs
	fs.Var(&implementers, "implementer", "implementing run as agent:model:effort; repeat for several")
	fs.Var(&reviewers, "reviewer", "reviewing run as agent:model:effort")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || len(implementers) == 0 || len(reviewers) != 1 {
		fmt.Fprintln(os.Stderr, "ateam review: compare requires one or more --implementer and exactly one --reviewer")
		return 2
	}
	data, err := readReviewFile(filepath.Join(root, ".agents", "review-policy.json"))
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("compare: skipped: no policy")
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 2
	}
	policy, err := review.ParsePolicy(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: .agents/review-policy.json: %v\n", err)
		return 2
	}
	findings, notes := policy.Compare(implementers, reviewers[0])
	for _, n := range notes {
		fmt.Println("compare: note: " + n)
	}
	for _, f := range findings {
		fmt.Println("compare: finding: " + f)
	}
	if len(findings) > 0 {
		return 1
	}
	fmt.Println("compare: pass: the reviewer is not weaker than any implementer")
	return 0
}

// reviewGate is the merge check (scripts/merge-check.sh runs it): exit 0 pass,
// 1 fail, 2 unavailable.
func reviewGate(root string, args []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	actions := os.Getenv("GITHUB_ACTIONS") == "true"
	cfg, source, err := gateForge(ctx, root, actions)
	if err != nil {
		fmt.Printf("merge-check: unavailable: %v\n", err)
		return 2
	}
	repo := cfg.Owner + "/" + cfg.Repo
	if repo == "/" {
		repo = "(resolved by gh)"
	}
	// The host is not printed: a self-hosted forge's name stays out of public logs.
	fmt.Printf("merge-check: forge=%s repo=%s (from %s)\n", cfg.Provider, repo, source)
	if cfg.Provider == string(state.ForgeGitea) {
		token, err := reviewToken(ctx, state.ForgeConfig{Provider: state.ForgeGitea})
		if err != nil {
			fmt.Printf("merge-check: unavailable: %v\n", err)
			return 2
		}
		cfg.Token = token
	}
	pr := ""
	if len(args) > 0 {
		pr = args[0]
	}
	opts := review.GateOptions{PR: pr, Actions: actions, Event: os.Getenv("GITHUB_EVENT_NAME"), PolicyPath: filepath.Join(root, ".agents", "review-policy.json")}
	return review.Gate(ctx, os.Stdout, cfg, opts)
}

// gateForge picks the forge the merge check reads: ATEAM_FORGE when set; under
// Actions, GITHUB_SERVER_URL (Gitea Actions sets it to the Gitea root) with
// GITHUB_REPOSITORY; otherwise origin. With none of them it is GitHub through
// gh, which resolves the repository itself, as the shell script always did.
func gateForge(ctx context.Context, root string, actions bool) (review.ForgeConfig, string, error) {
	explicit := strings.ToLower(strings.TrimSpace(os.Getenv("ATEAM_FORGE")))
	if explicit != "" && explicit != "github" && explicit != "gitea" {
		return review.ForgeConfig{}, "", fmt.Errorf("ATEAM_FORGE=%s is neither github nor gitea", explicit)
	}
	server := strings.TrimRight(strings.TrimSpace(os.Getenv("GITHUB_SERVER_URL")), "/")
	if actions && server != "" {
		provider, source := "gitea", "GITHUB_SERVER_URL"
		if server == "https://github.com" {
			provider = "github"
		}
		if explicit != "" {
			provider, source = explicit, "ATEAM_FORGE"
		}
		owner, repo, ok := strings.Cut(os.Getenv("GITHUB_REPOSITORY"), "/")
		if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
			return review.ForgeConfig{}, "", fmt.Errorf("GITHUB_REPOSITORY is not owner/repo")
		}
		return review.ForgeConfig{Provider: provider, BaseURL: server, Owner: owner, Repo: repo}, source, nil
	}
	source := "origin"
	if explicit != "" {
		source = "ATEAM_FORGE"
	}
	remote, err := review.GitRemote(ctx, root)
	if err != nil {
		remote = ""
	}
	cfg := state.DetectForge(ctx, root, remote)
	switch cfg.Provider {
	case state.ForgeGitHub, state.ForgeGitea:
		return review.ForgeConfig{Provider: string(cfg.Provider), BaseURL: cfg.BaseURL, Owner: cfg.Owner, Repo: cfg.Repo}, source, nil
	}
	return review.ForgeConfig{Provider: "github"}, "default: no origin forge", nil
}

func readReviewFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("review file exceeds 1 MiB")
	}
	return data, nil
}

func reviewSetup(root string, args []string) error {
	fs := flag.NewFlagSet("review setup", flag.ContinueOnError)
	file := fs.String("policy", "", "owner-approved policy JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" || fs.NArg() != 0 {
		return fmt.Errorf("setup requires --policy FILE and no positional arguments")
	}
	data, err := readReviewFile(*file)
	if err != nil {
		return err
	}
	written, err := bootstrap.ConfigureReview(root, data)
	if err != nil {
		return fmt.Errorf("setup wrote %v before failure: %w", written, err)
	}
	printJSON(map[string]any{"written": written, "notice": "Policy is a project declaration; routing does not launch agents or attest their actual models."})
	return nil
}

func reviewRoute(root string, args []string) error {
	fs := flag.NewFlagSet("review route", flag.ContinueOnError)
	models := fs.String("developer-models", "", "comma-separated actual implementation model IDs; defaults to configured developer")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("route accepts flags only")
	}
	data, err := readReviewFile(filepath.Join(root, ".agents", "review-policy.json"))
	if err != nil {
		return fmt.Errorf("review routing is unconfigured: %w", err)
	}
	p, err := review.ParsePolicy(data)
	if err != nil {
		return err
	}
	if !p.HasRoutes() {
		return fmt.Errorf("review routing is unconfigured: the policy declares model tiers only (ADR 0002)")
	}
	selected := p.Routes.Reviewer
	devIDs := []string{p.Routes.Developer.Model}
	if *models != "" {
		devIDs = strings.Split(*models, ",")
	}
	provider := func(id string) string {
		for _, m := range p.Models {
			if m.ID == id {
				return m.Provider
			}
		}
		return ""
	}
	devs := make([]review.Run, 0, len(devIDs))
	for i, id := range devIDs {
		id = strings.TrimSpace(id)
		devs = append(devs, review.Run{ID: fmt.Sprintf("planned-developer-%d", i), Model: id, Provider: provider(id), Effort: p.Routes.Developer.Effort})
	}
	rev := review.Run{ID: "planned-reviewer", Model: selected.Model, Provider: provider(selected.Model), Effort: selected.Effort}
	if err := review.Eligible(p, devs, rev); err != nil {
		return fmt.Errorf("routing blocked; no automatic downgrade or same-tier fallback: %w", err)
	}
	printJSON(map[string]any{"developer": p.Routes.Developer, "implementation_models": devIDs, "reviewer": rev, "policy_digest": review.PolicyDigest(data), "notice": "Selection only. Launch a separate read-only reviewer with this model and record the actual runtime identity; no model was launched."})
	return nil
}

func reviewCheck(root string, args []string) error {
	fs := flag.NewFlagSet("review check", flag.ContinueOnError)
	prNumber := fs.Int("pr", 0, "pull request number")
	recordID := fs.Int("record-comment", 0, "comment containing ateam-review followed by record JSON")
	exceptionID := fs.Int("exception-comment", 0, "optional owner comment containing ateam-review-exception followed by exception JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *prNumber <= 0 || *recordID <= 0 || *exceptionID < 0 || fs.NArg() != 0 {
		return fmt.Errorf("check requires positive --pr and --record-comment IDs and optional positive --exception-comment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	remote, err := review.GitRemote(ctx, root)
	if err != nil {
		return err
	}
	cfg := state.DetectForge(ctx, root, remote)
	if cfg.Provider != state.ForgeGitHub && cfg.Provider != state.ForgeGitea {
		return fmt.Errorf("review check requires a GitHub or Gitea origin")
	}
	token, err := reviewToken(ctx, cfg)
	if err != nil {
		return err
	}
	fc := review.ForgeConfig{Provider: string(cfg.Provider), BaseURL: cfg.BaseURL, Owner: cfg.Owner, Repo: cfg.Repo, Token: token}
	client := &http.Client{Timeout: 10 * time.Second}
	pr, err := review.ReadPullRequest(ctx, client, fc, *prNumber)
	if err != nil {
		return err
	}
	data, err := review.BaselinePolicy(ctx, root, pr.Base, pr.Head)
	if err != nil {
		return err
	}
	policy, err := review.ParsePolicy(data)
	if err != nil {
		return fmt.Errorf("trusted policy: %w", err)
	}
	if !policy.HasRoutes() {
		return fmt.Errorf("trusted policy declares model tiers only; review check needs models and routes (ADR 0002)")
	}
	comment, err := review.ReadComment(ctx, client, fc, *prNumber, *recordID)
	if err != nil {
		return err
	}
	payload, err := reviewCommentPayload(comment.Body, "ateam-review")
	if err != nil {
		return err
	}
	record, err := review.ParseRecord(payload)
	if err != nil {
		return err
	}
	var exception *review.Exception
	exceptionAuthor := ""
	var exceptionComment review.Comment
	if *exceptionID != 0 {
		c, err := review.ReadComment(ctx, client, fc, *prNumber, *exceptionID)
		if err != nil {
			return err
		}
		b, err := reviewCommentPayload(c.Body, "ateam-review-exception")
		if err != nil {
			return err
		}
		e, err := review.ParseException(b)
		if err != nil {
			return err
		}
		exception = &e
		exceptionAuthor = c.Author
		exceptionComment = c
	}
	repo := string(cfg.Provider) + ":" + strings.TrimRight(cfg.BaseURL, "/") + "/" + cfg.Owner + "/" + cfg.Repo
	expected := review.Expected{Repository: repo, Base: pr.Base, Head: pr.Head, PolicyDigest: review.PolicyDigest(data), RecordAuthor: comment.Author}
	if err := review.Validate(policy, record, expected, exception, exceptionAuthor); err != nil {
		return err
	}
	// Comments authenticate original accounts, not every later editor. Refuse
	// changed evidence within this check; timestamps also reject known edits.
	currentComment, err := review.ReadComment(ctx, client, fc, *prNumber, *recordID)
	if err != nil {
		return err
	}
	if currentComment != comment {
		return fmt.Errorf("review comment changed during validation; post fresh evidence")
	}
	if *exceptionID != 0 {
		currentException, err := review.ReadComment(ctx, client, fc, *prNumber, *exceptionID)
		if err != nil {
			return err
		}
		if currentException != exceptionComment {
			return fmt.Errorf("exception comment changed during validation; obtain fresh owner approval")
		}
	}
	// Detect movement during this check; merge protection must rerun against its head.
	latest, err := review.ReadPullRequest(ctx, client, fc, *prNumber)
	if err != nil {
		return err
	}
	if latest.Base != pr.Base || latest.Head != pr.Head {
		return fmt.Errorf("PR changed during review validation; rerun on its current head")
	}
	printJSON(map[string]any{"approved": true, "head": pr.Head, "base": pr.Base, "policy_digest": expected.PolicyDigest, "record_comment": *recordID, "record_author": comment.Author, "exception_author": exceptionAuthor, "notice": "Forge authenticates original comment accounts. Known edits are rejected, but timestamp granularity can hide same-tick edits. Body authorship, model identity, session independence and checks remain attestations; this does not merge or configure branch protection."})
	return nil
}

func reviewCommentPayload(body, marker string) ([]byte, error) {
	first, rest, ok := strings.Cut(strings.TrimSpace(body), "\n")
	if !ok || strings.TrimSpace(first) != marker {
		return nil, fmt.Errorf("comment must start with %s on its own line, followed by JSON", marker)
	}
	return []byte(strings.TrimSpace(rest)), nil
}

func reviewToken(ctx context.Context, cfg state.ForgeConfig) (string, error) {
	// Review evidence and the merge check only read the forge, so they take the read role (#160).
	if token := state.ReadToken(); token != "" {
		return token, nil
	}
	if cfg.Provider == state.ForgeGitea {
		if token := strings.TrimSpace(os.Getenv("GITEA_TOKEN")); token != "" {
			return token, nil
		}
		return strings.TrimSpace(os.Getenv("ATEAM_GITEA_TOKEN")), nil
	}
	if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		return token, nil
	}
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token, nil
	}
	// Never send credentials from the default gh account to a configured endpoint.
	if cfg.BaseURL != "https://api.github.com" {
		return "", nil
	}
	cmd := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", "github.com")
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	} // Public read-only repositories do not need a token.
	return strings.TrimSpace(string(out)), nil
}

// forgeTokenWarnings nags when forge reads and writes would authenticate with one credential: neither role token set
// (the forge's single token serves both), only one set, or both set to the same value. It names variables, never
// token values, and is a warning only so solo setups keep passing doctor --check (#160).
func forgeTokenWarnings() []string {
	if state.ReadToken() != state.WriteToken() {
		return nil
	}
	return []string{"forge reads and writes share one token; set ATEAM_TOKEN_READ and ATEAM_TOKEN_WRITE to separate tokens scoped to each role"}
}
