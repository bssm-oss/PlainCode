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
	pruntime "github.com/bssm-oss/PlainCode/internal/runtime"
	"github.com/bssm-oss/PlainCode/internal/validate/speccheck"
)

func cmdTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	specID := fs.String("spec", "", "Spec ID to verify")
	outputJSON := fs.Bool("json", false, "Output verification result as JSON")
	mode := fs.String("mode", "", "Runtime mode override for HTTP oracles: auto, process, docker")
	skipCommand := fs.Bool("skip-command", false, "Skip tests.command and only run parsed spec oracles")
	keepRunning := fs.Bool("keep-running", false, "Keep the runtime running if plaincode test starts it")
	_ = fs.Parse(args)

	if *specID == "" {
		fmt.Fprintln(os.Stderr, "usage: plaincode test --spec <id>")
		os.Exit(1)
	}

	dir, _ := os.Getwd()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	spec, err := app.LoadSingleSpec(filepath.Join(dir, cfg.Project.SpecDir), *specID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading spec: %v\n", err)
		os.Exit(1)
	}

	runtimeManager := pruntime.NewManager(dir, filepath.Join(dir, cfg.Project.StateDir))
	checker := speccheck.New(runtimeManager)
	result, err := checker.Run(context.Background(), spec, dir, speccheck.Options{
		RuntimeMode: *mode,
		SkipCommand: *skipCommand,
		KeepRunning: *keepRunning,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "spec verification failed: %v\n", err)
		os.Exit(1)
	}

	if *outputJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		printSpecTestResult(result)
	}

	if !result.Passed {
		os.Exit(1)
	}
}

func printSpecTestResult(result *speccheck.Result) {
	status := "OK"
	if !result.Passed {
		status = "FAIL"
	}
	fmt.Printf("[%s] %s\n", status, result.SpecID)
	if result.CommandResult != nil {
		cmdStatus := "PASS"
		if !result.CommandResult.Passed {
			cmdStatus = "FAIL"
		}
		fmt.Printf("  tests.command: %s", cmdStatus)
		if result.CommandResult.ExitCode != 0 {
			fmt.Printf(" (exit %d)", result.CommandResult.ExitCode)
		}
		fmt.Println()
	}
	if result.StartedRuntime && result.RuntimeState != nil {
		fmt.Printf("  runtime: %s (started by plaincode test)\n", result.RuntimeState.Mode)
	} else if result.RuntimeState != nil {
		fmt.Printf("  runtime: %s (reused existing service)\n", result.RuntimeState.Mode)
	}
	for _, oracle := range result.Oracles {
		label := "PASS"
		if !oracle.Passed {
			label = "FAIL"
		}
		fmt.Printf("  oracle: %s  %s\n", label, oracle.Raw)
		if oracle.Error != "" {
			fmt.Printf("    error: %s\n", oracle.Error)
		}
	}
	if len(result.IgnoredOracles) > 0 {
		fmt.Printf("  ignored oracles: %d\n", len(result.IgnoredOracles))
	}
	if len(result.Errors) > 0 {
		fmt.Println("  errors:")
		for _, item := range result.Errors {
			fmt.Printf("    - %s\n", item)
		}
		if result.RuntimeState != nil {
			if result.RuntimeState.LogPath != "" {
				fmt.Printf("  log: %s\n", result.RuntimeState.LogPath)
			}
			if result.RuntimeState.EventPath != "" {
				fmt.Printf("  events: %s\n", result.RuntimeState.EventPath)
			}
		}
	}
}
