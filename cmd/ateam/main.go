// Package main provides the ateam CLI entrypoint.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/bootstrap"
	"github.com/c360studio/ateam/internal/hook"
	"github.com/c360studio/ateam/internal/mcpserver"
	"github.com/c360studio/ateam/internal/mutate"
	"github.com/c360studio/ateam/internal/state"
	"github.com/c360studio/ateam/internal/taskrunner"
	"github.com/c360studio/ateam/internal/testspec"
	"github.com/c360studio/ateam/internal/web"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage()
		return 0
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam: %v\n", err)
		return 1
	}
	cwd = findRepoRoot(cwd)

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "review":
		return runReview(cwd, subArgs)
	case "hook":
		return runHook(cwd, subArgs)
	case "mcp":
		return runMCP(cwd)
	case "state":
		return runState(cwd, subArgs)
	case "task":
		return runTask(cwd, subArgs)
	case "init":
		return runInit(cwd, subArgs)
	case "doctor":
		return runDoctor(cwd, subArgs)
	case "bench":
		return runBench(cwd, subArgs)
	case "test-spec":
		return runTestSpec(cwd, subArgs)
	case "mutate":
		return runMutate(cwd, subArgs)
	case "serve":
		return runServe(cwd, subArgs)
	case "version", "--version", "-v":
		fmt.Println(state.VersionString())
		return 0
	default:
		fmt.Fprintf(os.Stderr, "ateam: unknown command %q\n", cmd)
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Println(`ateam - Universal Agentic Harness CLI, MCP Server, Benchmark & PM Board

Usage:
  ateam <command> [flags]

Commands:
  hook [--event <ev>] Execute <10ms normalized lifecycle hook (session-start, before-prompt, pre-compact, turn-end)
  mcp                 Start the 2-tool stdio MCP server (<300 schema tokens)
  state <subcmd>      Inspect or mutate repo state (status, pickup, handoff, holds, debt, spec-queue, spec-check, sync, mode, board, herdr)
  task [name]         List or execute standardized Taskfile.yml targets
  init [flags]        Bootstrap repo with .agents, .claude, .codex, .gemini, openspec, and optional GitHub/Gitea labels
  review <subcmd>     Configure model routes and validate commit-bound independent review
  doctor [--check]    Verify multi-agent parity, MCP schema token budget, and hook latency
  bench [flags]       Offline self-tests or paired experiments (run, rescore, compare, export)
  test-spec          Check Go and Vitest test-to-spec citations offline
  mutate [flags]      Run non-destructive Go -overlay mutation check (-pkg, -test, -file, -mutant, -expect/-expect-text)
  serve [flags]       Start embedded SvelteKit PM Board HTTP server (default 127.0.0.1:8420)`)
}

func runHook(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	evFlag := fs.String("event", "", "normalized hook event name")
	agFlag := fs.String("agent", "", "agent name (claude, codex, gemini, agy)")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	ev := *evFlag
	if ev == "" && fs.NArg() > 0 {
		ev = fs.Arg(0)
	}
	out, err := hook.HandleWithAgent(repoRoot, ev, *agFlag, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam hook: %v\n", err)
		return 1
	}
	if out.HasPayload() {
		_ = json.NewEncoder(os.Stdout).Encode(out)
	}
	return 0
}

func runMCP(repoRoot string) int {
	if err := mcpserver.Serve(context.Background(), repoRoot, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "ateam mcp: %v\n", err)
		return 1
	}
	return 0
}

func runState(repoRoot string, args []string) int {
	sub := "status"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "spec-queue", "spec_queue":
		return runSpecQueue(repoRoot)
	case "spec-check", "spec_check":
		return runSpecCheck(repoRoot)
	case "holds":
		return runStateHolds(repoRoot)
	case "debt":
		return runStateDebt(repoRoot)
	case "handoff":
		return runStateHandoff(repoRoot, args[1:])
	case "mode":
		return runStateMode(repoRoot, args[1:])
	case "sync":
		return runStateSync(repoRoot, args[1:])
	case "pickup":
		return runStatePickup(repoRoot, args[1:])
	case "read-handoff":
		return runStateReadHandoff(repoRoot, args[1:])
	case "prune":
		return runStatePrune(repoRoot, args[1:])
	case "status", "board":
		return runStateSnapshot(repoRoot, sub)
	case "herdr":
		return runStateHerdr(args[1:])
	case "-h", "--help", "help":
		fmt.Print(stateUsage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "ateam state: unknown subcommand %q; run 'ateam state --help' for subcommands\n", sub)
		return 1
	}
}

const stateUsage = `Usage:
  ateam state [subcommand] [flags]

Subcommands:
  status       Refresh state and print it as JSON (default)
  board        Refresh state with a forge sync and print the PM board as JSON
  pickup       Refresh state and print the pickup summary (--agent, --session)
  handoff      Record the five-field handoff checkpoint (see ateam state handoff -h)
  read-handoff <agent>@<branch>
               Print another slot's handoff without changing state
  prune        Remove your clean, merged worktrees and their branches (--dry-run, --agent)
  holds        List unticked Hold: items as JSON
  debt         List // shortcut: comments as JSON
  spec-queue   Show in-flight OpenSpec changes with their holds and caveats
  spec-check   Validate OpenSpec changes and fail on a Hold: outside every task
  sync         Sync issues and PRs from the configured forges
  mode [mode]  Show or set the harness mode
  herdr        herdr integration (on-event, open-pane)
`

func runSpecQueue(repoRoot string) int {
	changes, err := state.ScanOpenSpecQueue(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	out, _ := state.FormatOpenSpecQueue(changes)
	fmt.Print(out)
	return 0
}

func runSpecCheck(repoRoot string) int {
	out, code, err := state.CheckOpenSpec(context.Background(), repoRoot)
	if code == 2 {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if code != 0 || err != nil {
		if out != "" {
			fmt.Fprint(os.Stderr, out)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		return 1
	}
	fmt.Print(out)
	return 0
}

func runStateHolds(repoRoot string) int {
	holds, err := state.ScanHolds(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state holds: %v\n", err)
		return 1
	}
	printJSON(holds)
	return 0
}

func runStateDebt(repoRoot string) int {
	items, err := state.ScanShortcuts(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state debt: %v\n", err)
		return 1
	}
	printJSON(items)
	return 0
}

func runStatePickup(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("state pickup", flag.ContinueOnError)
	agent := fs.String("agent", "", "optional agent identifier (claude, codex, gemini, agy)")
	session := fs.String("session", "", "optional session identifier")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	st, err := state.RefreshWithBestEffortSync(context.Background(), repoRoot, 30*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state pickup: %v\n", err)
		return 1
	}
	st.CurrentSessionID = state.ResolveActiveSessionID(repoRoot, *session)
	st.CurrentAgent = state.ResolveActiveAgentForSession(repoRoot, *agent, st.CurrentSessionID, st.Git.Branch, st.Writer)
	state.ReconcilePickup(repoRoot, st, *agent, st.CurrentSessionID)
	fmt.Println(state.BuildPickupSummary(st))
	return 0
}

// runStatePrune removes the caller's finished worktrees and their branches, listing every other worktree with the reason it was kept (#229).
func runStatePrune(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("state prune", flag.ContinueOnError)
	agent := fs.String("agent", "", "agent whose worktrees to prune (default: the resolved caller)")
	dryRun := fs.Bool("dry-run", false, "print what would be removed without changing anything")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx := context.Background()
	st, err := state.RefreshWithBestEffortSync(ctx, repoRoot, 30*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state prune: %v\n", err)
		return 1
	}
	st.ReconcileWorktrees()
	who := state.ResolveActiveAgentForSession(repoRoot, *agent, state.ResolveActiveSessionID(repoRoot, ""), st.Git.Branch, st.Writer)
	if who == "" {
		fmt.Fprintln(os.Stderr, "ateam state prune: cannot resolve the caller's agent; pass --agent")
		return 2
	}
	failed := state.Prune(ctx, repoRoot, st, who, *dryRun, os.Stdout)
	if !*dryRun {
		// Refresh so the cache no longer lists the removed worktrees.
		if _, err := state.Refresh(ctx, repoRoot, false); err != nil {
			fmt.Fprintf(os.Stderr, "ateam state prune: refresh: %v\n", err)
			failed = true
		}
	}
	if failed {
		return 1
	}
	return 0
}

// runStateReadHandoff prints one slot's five fields; it never refreshes, syncs, or writes state (#196).
func runStateReadHandoff(repoRoot string, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: ateam state read-handoff <agent>@<branch>")
		return 2
	}
	out, err := state.ReadHandoff(repoRoot, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state read-handoff: %v\n", err)
		return 1
	}
	fmt.Print(out)
	return 0
}

func runStateHandoff(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("state handoff", flag.ContinueOnError)
	goal := fs.String("goal", "", "1-sentence goal")
	addresses := fs.String("addresses", "", "issues/PRs/files in play")
	constraints := fs.String("constraints", "", "constraints or active holds")
	doneWhen := fs.String("done-when", "", "definition of done")
	checks := fs.String("checks", "task verify", "checks to run")
	agent := fs.String("agent", "", "optional agent identifier (claude, codex, gemini, agy)")
	session := fs.String("session", "", "optional session identifier")
	noPost := fs.Bool("no-post", false, "skip posting handoff comment to open PR on forge")
	force := fs.Bool("force", false, "replace a fresh handoff another session holds in this agent@branch slot")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if strings.TrimSpace(*goal) == "" {
		fmt.Fprintln(os.Stderr, "ateam state handoff: --goal is required")
		return 1
	}
	st, err := state.Refresh(context.Background(), repoRoot, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state handoff: %v\n", err)
		return 1
	}
	sessionID := state.ResolveActiveSessionID(repoRoot, *session)
	agentName := state.ResolveActiveAgentForSession(repoRoot, *agent, sessionID, st.Git.Branch, st.Writer)
	ho := state.Handoff{
		Goal:          *goal,
		Addresses:     *addresses,
		Constraints:   *constraints,
		DoneWhen:      *doneWhen,
		ChecksToRun:   *checks,
		Agent:         agentName,
		SessionID:     sessionID,
		Branch:        st.Git.Branch,
		CheckpointSHA: st.Git.HeadSHA,
		Dirty:         st.Git.Dirty,
		UpdatedAt:     time.Now().UTC(),
	}
	if strings.TrimSpace(ho.Addresses) == "" {
		ho.Addresses = fmt.Sprintf("branch %s, SHA %s", st.Git.Branch, st.Git.HeadSHA)
	}
	if _, err := state.Update(repoRoot, func(cur *state.State) error {
		if err := cur.SetHandoffGuarded(ho, *force); err != nil {
			return err
		}
		cur.CurrentAgent = agentName
		cur.Writer = nil
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "ateam state handoff: %v\n", err)
		var held *state.HandoffSlotHeldError
		if errors.As(err, &held) {
			return 2
		}
		return 1
	}
	_ = state.ClearWriterLease(repoRoot)
	if strings.TrimSpace(ho.SessionID) == "" {
		fmt.Fprintf(os.Stderr, "ateam state handoff: warning: %s\n", state.HandoffNoSessionWarning(ho))
	}
	prompt := state.FormatFiveFieldPrompt(ho)
	if !*noPost && st.Forge.Provider != state.ForgeLocal {
		if _, err := state.PostPRHandoffComment(context.Background(), repoRoot, st.Forge, ho.Branch, prompt); err != nil {
			fmt.Fprintf(os.Stderr, "ateam state handoff: forge PR comment skipped: %v\n", err)
		}
	}
	fmt.Print(prompt)
	return 0
}

func runStateMode(repoRoot string, args []string) int {
	if len(args) > 0 {
		mode, ok := state.NormalizeMode(args[0])
		if !ok {
			fmt.Fprintf(os.Stderr, "ateam state mode: invalid mode %q\n", args[0])
			return 1
		}
		st, err := state.Update(repoRoot, func(cur *state.State) error {
			cur.Mode = mode
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ateam state mode: %v\n", err)
			return 1
		}
		printJSON(map[string]string{"mode": string(st.Mode)})
		return 0
	}
	st, err := state.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state mode: %v\n", err)
		return 1
	}
	printJSON(map[string]string{"mode": string(st.Mode)})
	return 0
}

func runStateSync(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("state sync", flag.ContinueOnError)
	forge := fs.String("forge", "", "optional forge override (github or gitea)")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	st, err := state.Refresh(ctx, repoRoot, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state sync: %v\n", err)
		return 1
	}
	cfg := st.Forge
	if *forge != "" {
		p := state.ForgeProvider(strings.ToLower(strings.TrimSpace(*forge)))
		if resolved := st.ResolveForge(p); resolved.Provider != "" {
			cfg = resolved
		} else {
			cfg.Provider = p
		}
	}
	cards, err := state.SyncForgeBoard(ctx, repoRoot, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state sync: %v\n", err)
		return 1
	}
	if _, err := state.Update(repoRoot, func(cur *state.State) error {
		cur.MergeForgeCards(cards, cfg.Provider)
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "ateam state sync: %v\n", err)
		return 1
	}
	printJSON(map[string]any{"synced": len(cards), "forge": cfg})
	return 0
}

func runStateSnapshot(repoRoot, sub string) int {
	var (
		st  *state.State
		err error
	)
	if sub == "board" {
		st, err = state.RefreshWithBestEffortSync(context.Background(), repoRoot, 30*time.Second)
	} else {
		st, err = state.Refresh(context.Background(), repoRoot, false)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state: %v\n", err)
		return 1
	}
	if sub == "board" {
		printJSON(st.BoardView())
		return 0
	}
	printJSON(st)
	return 0
}

func runTask(repoRoot string, args []string) int {
	if len(args) == 0 || args[0] == "list" {
		targets, err := taskrunner.ListTargets(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ateam task list: %v\n", err)
			return 1
		}
		printJSON(targets)
		return 0
	}
	target := args[0]
	extra := args[1:]
	if target == "run" && len(args) > 1 {
		target = args[1]
		extra = args[2:]
	}
	out, err := taskrunner.Run(context.Background(), repoRoot, target, extra)
	fmt.Print(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam task run %s: %v\n", target, err)
		return 1
	}
	return 0
}

func runInit(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	dir := fs.String("dir", repoRoot, "target directory to bootstrap")
	force := fs.Bool("force", false, "overwrite harness documents/adapters; preserve existing client configuration")
	ghLabels := fs.Bool("github", false, "bootstrap canonical labels on GitHub via gh")
	giteaLabels := fs.Bool("gitea", false, "bootstrap canonical labels on Gitea")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	forgeOverride := ""
	if *ghLabels {
		forgeOverride = "github"
	} else if *giteaLabels {
		forgeOverride = "gitea"
	}
	res, err := bootstrap.InitRepo(context.Background(), *dir, bootstrap.Options{
		Force:          *force,
		BootstrapForge: *ghLabels || *giteaLabels,
		ForgeOverride:  forgeOverride,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam init: %v\n", err)
		if len(res.BackedUp) > 0 {
			fmt.Fprintf(os.Stderr, "ateam init: backed up before failing: %s\n", strings.Join(res.BackedUp, ", "))
		}
		return 1
	}
	printJSON(map[string]any{"dir": *dir, "written": res.Written, "backed_up": res.BackedUp, "next_steps": "Existing client configuration is preserved, even with --force; replaced hand-written harness documents are listed under backed_up. Run ateam doctor --check and follow https://github.com/C360Studio/ateam/blob/main/docs/mcp.md to merge missing registration and verify client discovery."})
	return 0
}

const (
	hookLatencyBudgetUs = 10000
	hookLatencyMaxUs    = 25000
)

// hookLatencyFindings maps a measured hook latency to a doctor warning (above the target budget) and a --check
// violation (above the hard ceiling); empty strings mean no finding.
func hookLatencyFindings(hookUs int64) (warning, violation string) {
	if hookUs > hookLatencyBudgetUs {
		warning = fmt.Sprintf("hook latency %d us exceeds <%dms (%d us) target budget", hookUs, hookLatencyBudgetUs/1000, hookLatencyBudgetUs)
	}
	if hookUs > hookLatencyMaxUs {
		violation = fmt.Sprintf("hook latency %d us exceeded %d us hard ceiling (target budget <%d us)", hookUs, hookLatencyMaxUs, hookLatencyBudgetUs)
	}
	return warning, violation
}

func runDoctor(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	_ = fs.Bool("check", false, "exit non-zero if any parity check fails")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	violations := bootstrap.CheckParity(repoRoot)
	tokens, bytesLen, _ := mcpserver.EstimateSchemaTokens()
	if tokens >= mcpserver.MaxSchemaTokens {
		violations = append(violations, fmt.Sprintf("MCP schema budget exceeded: %d tokens >= %d", tokens, mcpserver.MaxSchemaTokens))
	}
	_, _ = hook.Probe(repoRoot, "session-start")
	var hookUs int64
	for i := 0; i < 3; i++ {
		t0 := time.Now()
		_, _ = hook.Probe(repoRoot, "session-start")
		d := time.Since(t0).Microseconds()
		if i == 0 || d < hookUs {
			hookUs = d
		}
	}
	_, fullRev, _ := state.BuildRevision()
	_, headSHA := state.ReadFastGitHead(repoRoot)
	buildProvenance, warnings := bootstrap.BuildProvenanceStatus(context.Background(), repoRoot, fullRev, headSHA)
	warnings = append(warnings, bootstrap.CheckUnarchivedOpenSpecChanges(repoRoot)...)
	latencyWarning, latencyViolation := hookLatencyFindings(hookUs)
	if latencyWarning != "" {
		warnings = append(warnings, latencyWarning)
	}
	if latencyViolation != "" {
		violations = append(violations, latencyViolation)
	}
	servePIDs, procWarnings := bootstrap.InspectRunningServeProcesses(context.Background())
	warnings = append(warnings, procWarnings...)
	mcpProcs, mcpWarnings := bootstrap.InspectRunningMCPProcesses(context.Background(), repoRoot, headSHA)
	warnings = append(warnings, mcpWarnings...)
	diskBuild, diskWarnings := bootstrap.DiskBuildStatus(context.Background(), repoRoot)
	warnings = append(warnings, diskWarnings...)
	warnings = append(warnings, forgeTokenWarnings()...)

	adopterSkills, skillAliases := bootstrap.SkillInventory(repoRoot)

	passed := len(violations) == 0
	printJSON(map[string]any{
		"passed":                 passed,
		"build_commit":           state.BuildLabel(),
		"build_provenance":       buildProvenance,
		"disk_build":             diskBuild,
		"head_commit":            headSHA,
		"running_serve_pids":     servePIDs,
		"running_mcp_processes":  mcpProcs,
		"adopter_skills":         adopterSkills,
		"skill_aliases":          skillAliases,
		"hook_sources":           bootstrap.HookSources(repoRoot),
		"warnings":               warnings,
		"mcp_schema_tokens":      tokens,
		"mcp_connection_checked": false,
		"mcp_schema_bytes":       bytesLen,
		"hook_latency_us":        hookUs,
		"hook_latency_budget_us": hookLatencyBudgetUs,
		"violations":             violations,
	})
	if !passed {
		return 1
	}
	return 0
}

func runMutate(repoRoot string, args []string) int {
	opts, err := mutate.ParseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam mutate: %v\n", err)
		return 2
	}
	_, code := mutate.Run(context.Background(), repoRoot, opts, os.Stdout, os.Stderr)
	return code
}

func runServe(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8420", "HTTP listen address (default loopback)")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	srv, err := web.NewServer(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam serve: %v\n", err)
		return 1
	}
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	listenURL := *addr
	if strings.HasPrefix(listenURL, ":") {
		listenURL = "127.0.0.1" + listenURL
	}
	fmt.Printf("ATeam PM Board listening on http://%s\n", listenURL)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "ateam serve: %v\n", err)
		return 1
	}
	return 0
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func findRepoRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start
		}
		dir = parent
	}
}

func runTestSpec(root string, args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "ateam test-spec: no arguments accepted")
		return 2
	}
	ds, err := testspec.Check(context.Background(), root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam test-spec: %v\n", err)
		return 2
	}
	if len(ds) > 0 {
		fmt.Fprint(os.Stderr, testspec.Format(ds))
		return 1
	}
	fmt.Println("test-spec: all discovered supported tests have valid citations")
	return 0
}
