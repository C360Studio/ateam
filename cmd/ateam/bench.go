package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/c360studio/ateam/internal/bench"
)

func runBench(repoRoot string, args []string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if len(args) > 0 {
		switch args[0] {
		case "run", "rescore", "compare", "export":
			return runExperimentCLI(ctx, repoRoot, args)
		}
	}
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	selfTest := fs.Bool("selftest", false, "run offline reference fixtures (no agent calls)")
	rescore := fs.String("rescore", "", "validate a self-test artifact and rerun its installed fixture suite")
	markdown := fs.Bool("markdown", false, "print a human-readable report")
	save := fs.Bool("save", false, "save a unique artifact in benchmarks/runs")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 0 || (*selfTest && *rescore != "") {
		fmt.Fprintln(os.Stderr, "bench: unexpected arguments or conflicting modes")
		return 1
	}
	if *rescore != "" {
		data, err := os.ReadFile(*rescore)
		if err != nil {
			return benchError(err)
		}
		if _, err := bench.ParseSelfTestReport(data); err != nil {
			return benchError(err)
		}
	}
	report, err := bench.RunSelfTest(ctx)
	if *save {
		path, saveErr := bench.SaveSelfTestRun(repoRoot, report)
		if saveErr != nil {
			return benchError(saveErr)
		}
		fmt.Fprintln(os.Stderr, "Saved:", path)
	}
	if *markdown {
		fmt.Print(bench.FormatMarkdown(report))
	} else {
		printJSON(report)
	}
	if err != nil {
		return benchError(err)
	}
	return 0
}

func runExperimentCLI(ctx context.Context, root string, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ateam bench run <experiment.json> [--dry-run] [--output dir] | rescore <run.json> | compare <run.json> [--json] | export <run.json> (--output new-dir | --check existing-dir) [--replay rescore-run.json]")
		return 1
	}
	switch args[0] {
	case "export":
		return runReportExportCLI(ctx, args)
	case "run":
		fs := flag.NewFlagSet("bench run", flag.ContinueOnError)
		dry := fs.Bool("dry-run", false, "validate and display resolved configuration without adapter calls")
		output := fs.String("output", filepath.Join(root, "benchmarks", "runs"), "run artifact directory")
		if err := fs.Parse(args[2:]); err != nil {
			return 1
		}
		if fs.NArg() != 0 {
			return benchError(fmt.Errorf("unexpected run arguments"))
		}
		path, err := filepath.Abs(args[1])
		if err != nil {
			return benchError(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return benchError(err)
		}
		config, err := bench.ParseExperiment(data)
		if err != nil {
			return benchError(err)
		}
		resolved, err := bench.ResolveExperimentContext(ctx, config, filepath.Dir(path))
		if err != nil {
			return benchError(err)
		}
		if *dry {
			printJSON(resolved)
			return 0
		}
		if config.IsNative() {
			report, checkErr := bench.RunSelfTest(ctx)
			evidence, saveErr := bench.SaveSelfTestRun(root, report)
			if saveErr != nil {
				return benchError(saveErr)
			}
			fmt.Fprintln(os.Stderr, "Preflight self-test:", evidence)
			if checkErr != nil {
				return benchError(checkErr)
			}
		}
		artifact, err := bench.RunExperiment(ctx, resolved, *output)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Partial evidence:", artifact)
			return benchError(err)
		}
		fmt.Println(artifact)
		return 0
	case "rescore":
		if len(args) != 2 {
			return benchError(fmt.Errorf("rescore requires one run artifact"))
		}
		path, err := bench.RescoreExperiment(ctx, args[1])
		if err != nil {
			return benchError(err)
		}
		fmt.Println(path)
		return 0
	case "compare":
		fs := flag.NewFlagSet("bench compare", flag.ContinueOnError)
		asJSON := fs.Bool("json", false, "print machine-readable comparison")
		if err := fs.Parse(args[2:]); err != nil {
			return 1
		}
		if fs.NArg() != 0 {
			return benchError(fmt.Errorf("unexpected compare arguments"))
		}
		run, err := bench.LoadExperimentRun(args[1])
		if err != nil {
			return benchError(err)
		}
		comparison, err := bench.CompareExperiment(run)
		if err != nil {
			return benchError(err)
		}
		if *asJSON {
			printJSON(comparison)
		} else {
			fmt.Print(bench.FormatComparison(comparison))
		}
		return 0
	}
	return 1
}

func runReportExportCLI(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("bench export", flag.ContinueOnError)
	output := fs.String("output", "", "create a report in a new directory")
	check := fs.String("check", "", "check existing generated files without writing")
	replay := fs.String("replay", "", "associate an existing rescore artifact without regrading")
	if err := fs.Parse(args[2:]); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		return benchError(fmt.Errorf("unexpected export arguments"))
	}
	destinations := 0
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "output" || f.Name == "check" {
			destinations++
		}
	})
	if destinations != 1 || (*output == "") == (*check == "") {
		return benchError(fmt.Errorf("export requires exactly one nonempty --output or --check directory"))
	}
	var err error
	if *output != "" {
		err = bench.ExportExperimentReport(ctx, args[1], *replay, *output)
	} else {
		err = bench.CheckExperimentReport(ctx, args[1], *replay, *check)
	}
	if err != nil {
		return benchError(err)
	}
	return 0
}

func benchError(err error) int { fmt.Fprintln(os.Stderr, "ateam bench:", err); return 1 }
