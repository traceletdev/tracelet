package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tracelet/internal/ci"
	"tracelet/internal/config"
	"tracelet/internal/fix"
	"tracelet/internal/hud"
	"tracelet/internal/lint"
	"tracelet/internal/probe"
	"tracelet/internal/reporters"
)

type globalFlags struct {
	configPath string
	quiet      bool
	ciMode     bool
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "init":
		runInit()
	case "lint":
		runLint(os.Args[2:])
	case "fix":
		runFix(os.Args[2:])
	case "probe":
		runProbe(os.Args[2:])
	case "ci":
		runCI(os.Args[2:])
	case "hud":
		runHUD(os.Args[2:])
	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("tracelet <command>\n\nCommands:\n  init\n  lint\n  fix\n  probe\n  ci\n  hud")
}

func runInit() {
	// Write default config if not present
	defaultCfg := config.DefaultConfig()
	if _, err := os.Stat("tracelet.config.json"); err == nil {
		fmt.Println("tracelet.config.json already exists")
		return
	}
	f, err := os.Create("tracelet.config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create config: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(defaultCfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write config: %v\n", err)
		os.Exit(1)
	}
	// Ensure .tracelet dir exists
	_ = os.MkdirAll(".tracelet", 0o755)
	// Create .gitkeep to help track directory
	_ = os.WriteFile(".tracelet/.gitkeep", []byte(""), 0o644)
	fmt.Println("Initialized tracelet.config.json and .tracelet/")
}

func runLint(args []string) {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	var (
		format   = fs.String("format", "table", "output format: table|json")
		scope    = fs.String("scope", "all", "lint scope: all|changed")
		filePath = fs.String("file", "", "target file for focused lint")
		cfgPath  = fs.String("config", "", "path to config")
		quiet    = fs.Bool("quiet", false, "suppress non-critical logs")
		ci       = fs.Bool("ci", false, "CI mode (non-zero exit on warnings)")
	)
	_ = fs.Parse(args)

	gl := globalFlags{configPath: *cfgPath, quiet: *quiet, ciMode: *ci}

	cfg, err := config.Load(gl.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(2)
	}

	req := lint.Request{
		Scope:  *scope,
		File:   *filePath,
		Config: cfg,
	}

	results, stats := lint.Run(req)

	switch strings.ToLower(*format) {
	case "json":
		reporters.PrintJSON(os.Stdout, results, stats)
	default:
		reporters.PrintTable(os.Stdout, results, stats)
	}

	// Exit codes: 0 ok, 1 warnings (if --ci), 2 errors
	hasError := results.HasLevel("error")
	hasWarn := results.HasLevel("warn")
	if hasError {
		os.Exit(2)
	}
	if gl.ciMode && hasWarn {
		os.Exit(1)
	}
}

func runFix(args []string) {
	fs := flag.NewFlagSet("fix", flag.ExitOnError)
	var (
		ruleID  = fs.String("rule", "", "fix specific rule (unoptimized-image, font-display)")
		apply   = fs.Bool("apply", false, "apply fixes (default: dry-run)")
		cfgPath = fs.String("config", "", "path to config")
	)
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(2)
	}

	// Run lint first to find issues
	req := lint.Request{Scope: "all", Config: cfg}
	results, _ := lint.Run(req)

	// Filter by rule if specified
	toFix := results
	if *ruleID != "" {
		filtered := lint.Results{}
		for _, r := range results {
			if r.RuleID == *ruleID {
				filtered = append(filtered, r)
			}
		}
		toFix = filtered
	}

	// Group by file
	fixesByFile := make(map[string][]lint.Results)
	for _, r := range toFix {
		if r.RuleID == "unoptimized-image" || r.RuleID == "font-display" {
			if r.File != "" {
				fixesByFile[r.File] = append(fixesByFile[r.File], lint.Results{r})
			}
		}
	}

	if len(fixesByFile) == 0 {
		fmt.Println("No fixable issues found")
		return
	}

	var applied int
	for filePath, fileResults := range fixesByFile {
		for _, fileRes := range fileResults {
			if len(fileRes) == 0 {
				continue
			}
			ruleID := fileRes[0].RuleID
			if *apply {
				f, err := fix.ApplyFix(ruleID, filePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error fixing %s: %v\n", filePath, err)
					continue
				}
				fmt.Println(fix.FormatFixResult(f))
				if f.Applied {
					applied++
				}
			} else {
				fmt.Printf("Would fix %s in %s (use --apply to apply)\n", ruleID, filePath)
			}
		}
	}

	if !*apply {
		fmt.Printf("\nRun with --apply to apply fixes\n")
	} else {
		fmt.Printf("\nApplied %d fix(es)\n", applied)
	}
}

func runProbe(args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	var (
		profile = fs.String("profile", "desktop", "desktop|mobile")
		runs    = fs.Int("runs", 1, "number of runs")
		outPath = fs.String("out", "", "write JSON to file")
		verbose = fs.Bool("verbose", false, "log each run and include samples in JSON")
	)
	// Allow URL before flags by peeling off first non-flag as candidate
	urlCandidate := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		urlCandidate = args[0]
		args = args[1:]
	}
	_ = fs.Parse(args)
	url := urlCandidate
	if fs.NArg() >= 1 {
		url = fs.Arg(0)
	}
	if url == "" {
		fmt.Fprintln(os.Stderr, "usage: tracelet probe <url> [--profile desktop|mobile] [--runs 1] [--out file]")
		os.Exit(2)
	}
	req := probe.Request{URL: url, Profile: probe.Profile(*profile), Runs: *runs, Verbose: *verbose}
	resp := probe.Run(req)

	// Warn if fallback stub metrics were used
	if resp.Note != "" && strings.Contains(resp.Note, "fallback") {
		fmt.Fprintf(os.Stderr, "warning: %s\n", resp.Note)
	}

	if *verbose && len(resp.Samples) > 0 {
		for i, s := range resp.Samples {
			fmt.Fprintf(os.Stderr, "run %d: TTFB=%dms FCP=%dms LCP=%dms CLS=%.3f TBT=%dms FSI=%dms\n", i+1, s.TTFBms, s.FCPms, s.LCPms, s.CLS, s.TBTms, s.FSIms)
		}
	}

	if *outPath != "" {
		// ensure parent directory exists
		if dir := filepath.Dir(*outPath); dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create out file: %v\n", err)
			os.Exit(2)
		}
		defer f.Close()
		_ = probe.WriteJSON(f, resp)
		fmt.Println("wrote:", *outPath)
		return
	}
	_ = probe.WriteJSON(os.Stdout, resp)
}

func runCI(args []string) {
	fs := flag.NewFlagSet("ci", flag.ExitOnError)
	var (
		compare = fs.String("compare", "", "path to baseline JSON to compare against")
		write   = fs.String("write-baseline", "", "write current results to baseline path and exit")
		format  = fs.String("format", "markdown", "markdown|json")
		cfgPath = fs.String("config", "", "path to config")
	)
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(2)
	}
	results, st := lint.Run(lint.Request{Scope: "all", Config: cfg})

	if *write != "" {
		bl := ci.Baseline{Results: results, Stats: st}
		if err := ci.SaveBaseline(*write, bl); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write baseline: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("baseline written:", *write)
		return
	}

	var base *ci.Baseline
	var diff *ci.Diff
	if *compare != "" {
		if bl, err := ci.LoadBaseline(*compare); err == nil {
			base = &bl
			d := ci.ComputeDiff(bl.Results, results)
			diff = &d
		}
	}

	switch strings.ToLower(*format) {
	case "json":
		reporters.PrintJSON(os.Stdout, results, st)
	default:
		reporters.PrintMarkdown(os.Stdout, results, st, base, diff)
	}

	if results.HasLevel("error") {
		os.Exit(2)
	}
}

func runHUD(args []string) {
	fs := flag.NewFlagSet("hud", flag.ExitOnError)
	var (
		port    = fs.Int("port", 3111, "port to listen on")
		cfgPath = fs.String("config", "", "path to config")
	)
	_ = fs.Parse(args)
	if err := hud.Start(*port, *cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "hud error: %v\n", err)
		os.Exit(2)
	}
}
