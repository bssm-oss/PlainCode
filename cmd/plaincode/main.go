// Package main is the CLI entrypoint for forge.
//
// Forge is a spec-first build orchestrator that compiles Markdown specs
// into real code using pluggable AI backends, validates the output,
// and enforces deterministic file ownership.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bssm-oss/PlainCode/internal/app"
	"github.com/bssm-oss/PlainCode/internal/config"
	"github.com/bssm-oss/PlainCode/internal/spec/parser"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "init":
		cmdInit(args)
	case "build":
		cmdBuild(args)
	case "change":
		cmdChange(args)
	case "takeover":
		cmdTakeover(args)
	case "coverage":
		cmdCoverage(args)
	case "providers":
		cmdProviders(args)
	case "agents":
		cmdAgents(args)
	case "trace":
		cmdTrace(args)
	case "explain":
		cmdExplain(args)
	case "serve":
		cmdServe(args)
	case "version":
		fmt.Printf("plaincode %s\n", version)
	case "parse-spec":
		cmdParseSpec(args) // development/debug command
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`plaincode — spec-first multi-agent build orchestrator

Usage: plaincode <command> [options]

Core Commands:
  init                        Initialize a new PlainCode project
  build [--spec <id>]         Build specs into code
  change -m "description"     Fix implementation bug (not spec change)
  takeover <file|package>     Extract spec from existing code
  coverage                    Run coverage analysis and gap filling

Inspection Commands:
  providers list|doctor       Manage AI backends
  agents list                 List AGENTS.md and skills
  trace <build-id>            Inspect build receipt and trace
  explain <spec-id>           Explain spec dependencies and ownership

Platform Commands:
  serve                       Start HTTP daemon (OpenAPI + SSE)

Development Commands:
  parse-spec <file>           Parse and dump a spec file (debug)
  version                     Print version

`)
}

// cmdInit initializes a new PlainCode project in the current directory.
func cmdInit(args []string) {
	dir, _ := os.Getwd()

	// Check if plaincode.yaml already exists
	if _, err := os.Stat(filepath.Join(dir, "plaincode.yaml")); err == nil {
		fmt.Fprintln(os.Stderr, "plaincode.yaml already exists in this directory")
		os.Exit(1)
	}

	// Create directory structure
	dirs := []string{
		"spec",
		".plaincode",
		".plaincode/builds",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "creating directory %s: %v\n", d, err)
			os.Exit(1)
		}
	}

	// Write default config
	if err := config.WriteDefault(dir); err != nil {
		fmt.Fprintf(os.Stderr, "writing plaincode.yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Initialized PlainCode project:")
	fmt.Println("  plaincode.yaml  — project configuration")
	fmt.Println("  spec/       — spec files directory")
	fmt.Println("  .plaincode/     — state directory (add to .gitignore)")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Create a spec:   spec/my-feature.md")
	fmt.Println("  2. Build it:        plaincode build --spec my-feature")
}

// cmdBuild builds one or all specs through the full pipeline.
func cmdBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	specID := fs.String("spec", "", "Build a specific spec by ID")
	dryRun := fs.Bool("dry-run", false, "Parse and validate only, don't execute")
	outputJSON := fs.Bool("json", false, "Output results as JSON")
	skipTests := fs.Bool("skip-tests", false, "Skip test execution")
	skipCoverage := fs.Bool("skip-coverage", false, "Skip coverage analysis")
	_ = fs.Parse(args)

	dir, _ := os.Getwd()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	// Build registry from plaincode.yaml providers config.
	// Falls back to mock backend if no providers match the default.
	registry := app.BuildRegistry(cfg)

	builder := app.NewBuilder(cfg, registry, dir)

	opts := app.BuildOptions{
		SpecID:       *specID,
		DryRun:       *dryRun,
		SkipTests:    *skipTests,
		SkipCoverage: *skipCoverage,
		JSONOutput:   *outputJSON,
		MaxRetries:   cfg.Defaults.RetryLimit,
	}

	results, err := builder.Build(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		os.Exit(1)
	}

	if *outputJSON {
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Human-readable output
	for _, r := range results {
		if r.Skipped {
			fmt.Printf("  [skip] %s\n", r.SkipReason)
			continue
		}
		status := "OK"
		if r.Status == "failed" {
			status = "FAIL"
		}
		fmt.Printf("  [%s] %s (build %s)\n", status, r.SpecID, r.BuildID)
		if r.Error != "" {
			fmt.Printf("    error: %s\n", r.Error)
		}
		if r.Receipt != nil {
			fmt.Printf("    backend: %s\n", r.Receipt.BackendID)
			fmt.Printf("    files:   %v\n", r.Receipt.ChangedFiles)
			fmt.Printf("    cost:    $%.4f\n", r.Receipt.CostUSD)
			fmt.Printf("    time:    %dms\n", r.Receipt.DurationMS)
		}
	}

	// Exit with error if any build failed
	for _, r := range results {
		if r.Status == "failed" {
			os.Exit(1)
		}
	}
}

func cmdChange(args []string) {
	fs := flag.NewFlagSet("change", flag.ExitOnError)
	msg := fs.String("m", "", "Change description")
	_ = fs.Parse(args)
	if *msg == "" {
		fmt.Fprintln(os.Stderr, "usage: plaincode change -m \"description\"")
		os.Exit(1)
	}
	// TODO: Implementation change request (not spec fix)
	fmt.Printf("[not yet implemented] Change request: %s\n", *msg)
}

func cmdTakeover(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: plaincode takeover <file|package>")
		os.Exit(1)
	}
	// TODO: Takeover v2 pipeline
	fmt.Printf("[not yet implemented] Takeover target: %s\n", args[0])
}

func cmdCoverage(args []string) {
	// TODO: Coverage analysis with language-specific providers
	fmt.Println("[not yet implemented] Coverage analysis")
}

func cmdProviders(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: plaincode providers <list|doctor>")
		os.Exit(1)
	}

	dir, _ := os.Getwd()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	registry := app.BuildRegistry(cfg)

	switch args[0] {
	case "list":
		ids := registry.List()
		fmt.Printf("Registered backends (%d):\n", len(ids))
		for _, id := range ids {
			b, _ := registry.Get(id)
			caps := b.Capabilities()
			fmt.Printf("  %-20s structured=%v mcp=%v tools=%v\n", id, caps.StructuredOutput, caps.MCP, caps.Tools)
		}
		fmt.Printf("\nDefault: %s\n", cfg.Defaults.Backend)
	case "doctor":
		fmt.Println("Health check:")
		results := registry.HealthCheckAll()
		for id, err := range results {
			if err != nil {
				fmt.Printf("  %-20s FAIL: %v\n", id, err)
			} else {
				fmt.Printf("  %-20s OK\n", id)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown providers subcommand: %s\n", args[0])
	}
}

func cmdAgents(args []string) {
	// TODO: List AGENTS.md, SKILL.md, .agents/skills/
	fmt.Println("[not yet implemented] Agent and skill listing")
}

func cmdTrace(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: plaincode trace <build-id>")
		os.Exit(1)
	}
	// TODO: Load and display build receipt
	fmt.Printf("[not yet implemented] Trace for build: %s\n", args[0])
}

func cmdExplain(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: plaincode explain <spec-id>")
		os.Exit(1)
	}
	// TODO: Show spec dependencies, ownership, backend preferences
	fmt.Printf("[not yet implemented] Explain spec: %s\n", args[0])
}

func cmdServe(args []string) {
	// TODO: HTTP daemon with OpenAPI + SSE
	fmt.Println("[not yet implemented] Starting forge daemon on :8420")
	fmt.Println("Endpoints planned:")
	fmt.Println("  POST /build      POST /change     POST /takeover")
	fmt.Println("  POST /coverage   GET  /builds/:id GET  /events (SSE)")
	fmt.Println("  GET  /providers  GET  /policies   GET  /health")
	fmt.Println("  GET  /openapi.json")
}

// cmdParseSpec is a debug command that parses a spec file and dumps the result.
func cmdParseSpec(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: plaincode parse-spec <file>")
		os.Exit(1)
	}

	spec, err := parser.ParseFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
