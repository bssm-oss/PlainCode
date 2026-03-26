package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bssm-oss/PlainCode/internal/app"
	"github.com/bssm-oss/PlainCode/internal/config"
	pruntime "github.com/bssm-oss/PlainCode/internal/runtime"
)

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	specID := fs.String("spec", "", "Spec ID to run")
	mode := fs.String("mode", "", "Runtime mode override: auto, process, docker")
	buildFirst := fs.Bool("build", false, "Build the spec before starting it")
	outputJSON := fs.Bool("json", false, "Output status as JSON")
	wait := fs.Duration("wait", 10*time.Second, "How long to wait for the service health check")
	_ = fs.Parse(args)

	if *specID == "" {
		fmt.Fprintln(os.Stderr, "usage: plaincode run --spec <id> [--build] [--mode auto|process|docker]")
		os.Exit(1)
	}

	dir, _ := os.Getwd()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	if *buildFirst {
		registry := app.BuildRegistry(cfg)
		builder := app.NewBuilder(cfg, registry, dir)
		results, err := builder.Build(context.Background(), app.BuildOptions{
			SpecID:     *specID,
			MaxRetries: cfg.Defaults.RetryLimit,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
			os.Exit(1)
		}
		if len(results) == 0 {
			fmt.Fprintln(os.Stderr, "build produced no result")
			os.Exit(1)
		}
		last := results[len(results)-1]
		if last.Status == "failed" {
			fmt.Fprintf(os.Stderr, "build failed: %s\n", last.Error)
			os.Exit(1)
		}
	}

	spec, err := app.LoadSingleSpec(filepath.Join(dir, cfg.Project.SpecDir), *specID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading spec: %v\n", err)
		os.Exit(1)
	}

	manager := pruntime.NewManager(dir, filepath.Join(dir, cfg.Project.StateDir))
	state, err := manager.Start(context.Background(), spec, pruntime.StartOptions{
		Mode:          *mode,
		HealthTimeout: *wait,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		os.Exit(1)
	}

	if *outputJSON {
		data, _ := json.MarshalIndent(state, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Started %s via %s\n", state.SpecID, state.Mode)
	fmt.Printf("  status:  %s\n", state.Status)
	fmt.Printf("  health:  %s\n", state.Health)
	if state.Mode == pruntime.ModeProcess {
		fmt.Printf("  pid:     %d\n", state.PID)
		fmt.Printf("  command: %v\n", state.Command)
	}
	if state.Mode == pruntime.ModeDocker {
		fmt.Printf("  image:   %s\n", state.Image)
		fmt.Printf("  container: %s\n", state.ContainerName)
	}
	if state.HealthcheckURL != "" {
		fmt.Printf("  check:   %s\n", state.HealthcheckURL)
	}
	if state.LogPath != "" {
		fmt.Printf("  log:     %s\n", state.LogPath)
	}
	if state.EventPath != "" {
		fmt.Printf("  events:  %s\n", state.EventPath)
	}
}

func cmdStop(args []string) {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	specID := fs.String("spec", "", "Spec ID to stop")
	outputJSON := fs.Bool("json", false, "Output status as JSON")
	_ = fs.Parse(args)

	if *specID == "" {
		fmt.Fprintln(os.Stderr, "usage: plaincode stop --spec <id>")
		os.Exit(1)
	}

	dir, _ := os.Getwd()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	manager := pruntime.NewManager(dir, filepath.Join(dir, cfg.Project.StateDir))
	state, err := manager.Stop(context.Background(), *specID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stop failed: %v\n", err)
		os.Exit(1)
	}
	if *outputJSON {
		data, _ := json.MarshalIndent(state, "", "  ")
		fmt.Println(string(data))
		return
	}
	fmt.Printf("Stopped %s\n", *specID)
}

func cmdStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	specID := fs.String("spec", "", "Spec ID to inspect")
	outputJSON := fs.Bool("json", false, "Output status as JSON")
	_ = fs.Parse(args)

	dir, _ := os.Getwd()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	manager := pruntime.NewManager(dir, filepath.Join(dir, cfg.Project.StateDir))
	var states []*pruntime.State
	if *specID != "" {
		state, err := manager.Status(context.Background(), *specID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
			os.Exit(1)
		}
		states = []*pruntime.State{state}
	} else {
		states, err = manager.List(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
			os.Exit(1)
		}
	}

	if *outputJSON {
		if states == nil {
			states = []*pruntime.State{}
		}
		data, _ := json.MarshalIndent(states, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(states) == 0 {
		fmt.Println("No managed services found.")
		return
	}
	for _, state := range states {
		fmt.Printf("%s\n", state.SpecID)
		fmt.Printf("  mode:    %s\n", state.Mode)
		fmt.Printf("  status:  %s\n", state.Status)
		fmt.Printf("  health:  %s\n", state.Health)
		if state.Mode == pruntime.ModeProcess && state.PID != 0 {
			fmt.Printf("  pid:     %d\n", state.PID)
		}
		if state.Mode == pruntime.ModeDocker && state.ContainerName != "" {
			fmt.Printf("  container: %s\n", state.ContainerName)
		}
		if state.HealthcheckURL != "" {
			fmt.Printf("  check:   %s\n", state.HealthcheckURL)
		}
		if state.LogPath != "" {
			fmt.Printf("  log:     %s\n", state.LogPath)
		}
		if state.EventPath != "" {
			fmt.Printf("  events:  %s\n", state.EventPath)
		}
		if state.Error != "" {
			fmt.Printf("  error:   %s\n", state.Error)
		}
	}
}
