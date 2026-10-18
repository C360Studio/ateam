// Package main provides the ateam CLI entrypoint.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/bench"
	"github.com/c360studio/ateam/internal/bootstrap"
	"github.com/c360studio/ateam/internal/hook"
	"github.com/c360studio/ateam/internal/mcpserver"
	"github.com/c360studio/ateam/internal/state"
	"github.com/c360studio/ateam/internal/taskrunner"
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
	case "serve":
		return runServe(cwd, subArgs)
	case "version", "--version", "-v":
		fmt.Println("ateam v0.1.0")
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
  hook [--event <ev>] Execute <5ms normalized lifecycle hook (session-start, before-prompt, pre-compact, turn-end)
  mcp                 Start the 2-tool stdio MCP server (<300 schema tokens)
  state <subcmd>      Inspect or mutate repo state (status, pickup, handoff, holds, debt, spec-queue, spec-check, sync, mode, board)
  task [name]         List or execute standardized Taskfile.yml targets
  init [flags]        Bootstrap repo with .agents, .claude, .codex, .gemini, openspec, and optional GitHub/Gitea labels
  doctor [--check]    Verify multi-agent parity, MCP schema token budget, and hook latency
  bench [flags]       Run Ponytail + Handoff benchmark suite (--selftest, --rescore)
  serve [flags]       Start embedded SvelteKit PM Board HTTP server (default :8420)`)
}

func runHook(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	evFlag := fs.String("event", "", "normalized hook event name")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	ev := *evFlag
	if ev == "" && fs.NArg() > 0 {
		ev = fs.Arg(0)
	}
	out, err := hook.Handle(repoRoot, ev, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam hook: %v\n", err)
		return 1
	}
	if out.AdditionalContext != "" || out.Decision != "" || out.HookSpecific != nil {
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
		st, err := state.Refresh(context.Background(), repoRoot, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ateam state pickup: %v\n", err)
			return 1
		}
		fmt.Println(state.BuildPickupSummary(st))
		return 0
	case "status", "board":
		return runStateSnapshot(repoRoot, sub)
	default:
		fmt.Fprintf(os.Stderr, "ateam state: unknown subcommand %q\n", sub)
		return 1
	}
}

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

func runStateHandoff(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("state handoff", flag.ContinueOnError)
	goal := fs.String("goal", "", "1-sentence goal")
	addresses := fs.String("addresses", "", "issues/PRs/files in play")
	constraints := fs.String("constraints", "", "constraints or active holds")
	doneWhen := fs.String("done-when", "", "definition of done")
	checks := fs.String("checks", "task verify", "checks to run")
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
	ho := state.Handoff{
		Goal:          *goal,
		Addresses:     *addresses,
		Constraints:   *constraints,
		DoneWhen:      *doneWhen,
		ChecksToRun:   *checks,
		Branch:        st.Git.Branch,
		CheckpointSHA: st.Git.HeadSHA,
		Dirty:         st.Git.Dirty,
		UpdatedAt:     time.Now().UTC(),
	}
	if strings.TrimSpace(ho.Addresses) == "" {
		ho.Addresses = fmt.Sprintf("branch %s, SHA %s", st.Git.Branch, st.Git.HeadSHA)
	}
	st.LastHandoff = &ho
	_ = state.Save(repoRoot, st)
	fmt.Print(state.FormatFiveFieldPrompt(ho))
	return 0
}

func runStateMode(repoRoot string, args []string) int {
	st, err := state.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state mode: %v\n", err)
		return 1
	}
	if len(args) > 0 {
		mode, ok := state.NormalizeMode(args[0])
		if !ok {
			fmt.Fprintf(os.Stderr, "ateam state mode: invalid mode %q\n", args[0])
			return 1
		}
		st.Mode = mode
		_ = state.Save(repoRoot, st)
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
		cfg.Provider = state.ForgeProvider(*forge)
	}
	cards, err := state.SyncForgeBoard(ctx, repoRoot, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state sync: %v\n", err)
		return 1
	}
	if len(cards) > 0 {
		st.Cards = cards
		_ = state.Save(repoRoot, st)
	}
	printJSON(map[string]any{"synced": len(cards), "forge": cfg})
	return 0
}

func runStateSnapshot(repoRoot, sub string) int {
	st, err := state.Refresh(context.Background(), repoRoot, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state: %v\n", err)
		return 1
	}
	if sub == "board" {
		printJSON(st.Cards)
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
	force := fs.Bool("force", false, "overwrite existing harness files")
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
	written, err := bootstrap.InitRepo(context.Background(), *dir, bootstrap.Options{
		Force:          *force,
		BootstrapForge: *ghLabels || *giteaLabels,
		ForgeOverride:  forgeOverride,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam init: %v\n", err)
		return 1
	}
	printJSON(map[string]any{"dir": *dir, "written": written})
	return 0
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
	t0 := time.Now()
	_, _ = hook.Handle(repoRoot, "session-start", strings.NewReader(`{}`))
	hookUs := time.Since(t0).Microseconds()
	if hookUs > 50000 {
		violations = append(violations, fmt.Sprintf("hook latency exceeded threshold: %d us", hookUs))
	}
	passed := len(violations) == 0
	printJSON(map[string]any{
		"passed":            passed,
		"mcp_schema_tokens": tokens,
		"mcp_schema_bytes":  bytesLen,
		"hook_latency_us":   hookUs,
		"violations":        violations,
	})
	if !passed {
		return 1
	}
	return 0
}

func runBench(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	selfTest := fs.Bool("selftest", false, "run offline good/bad fixture and mutation checks")
	rescorePath := fs.String("rescore", "", "rescore an existing JSON benchmark artifact")
	markdown := fs.Bool("markdown", false, "print human-readable markdown table")
	save := fs.Bool("save", false, "write benchmarks/runs/latest.json")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	_ = selfTest
	if *rescorePath != "" {
		data, err := os.ReadFile(*rescorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ateam bench --rescore: %v\n", err)
			return 1
		}
		var rep bench.SuiteReport
		if err := json.Unmarshal(data, &rep); err != nil {
			fmt.Fprintf(os.Stderr, "ateam bench --rescore parse: %v\n", err)
			return 1
		}
		if *markdown {
			fmt.Print(bench.FormatMarkdown(rep))
		} else {
			printJSON(rep)
		}
		if !rep.SelfTestPassed {
			return 1
		}
		return 0
	}
	rep, err := bench.RunSelfTest(context.Background())
	if *save {
		_ = bench.SaveRun(repoRoot, rep)
	}
	if *markdown {
		fmt.Print(bench.FormatMarkdown(rep))
	} else {
		printJSON(rep)
	}
	if err != nil || !rep.SelfTestPassed {
		return 1
	}
	return 0
}

func runServe(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", ":8420", "HTTP listen address")
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
	fmt.Printf("ATeam PM Board listening on http://localhost%s\n", *addr)
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
