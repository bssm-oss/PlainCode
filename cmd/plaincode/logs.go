package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bssm-oss/PlainCode/internal/config"
	pruntime "github.com/bssm-oss/PlainCode/internal/runtime"
)

func cmdLogs(args []string) {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	specID := fs.String("spec", "", "Spec ID to inspect")
	events := fs.Bool("events", false, "Show runtime event history instead of the main log")
	outputJSON := fs.Bool("json", false, "Output events as JSON")
	pathOnly := fs.Bool("path", false, "Print the stored artifact path only")
	tail := fs.Int("tail", 0, "Show only the last N lines/events")
	_ = fs.Parse(args)

	if *specID == "" {
		fmt.Fprintln(os.Stderr, "usage: plaincode logs --spec <id> [--events] [--tail N]")
		os.Exit(1)
	}

	dir, _ := os.Getwd()
	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	store := pruntime.NewStore(filepath.Join(dir, cfg.Project.StateDir))
	if *events {
		if *pathOnly {
			fmt.Println(store.EventPath(*specID))
			return
		}
		items, err := store.ReadEvents(*specID, *tail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading runtime events: %v\n", err)
			os.Exit(1)
		}
		if *outputJSON {
			data, _ := json.MarshalIndent(items, "", "  ")
			fmt.Println(string(data))
			return
		}
		for _, item := range items {
			fmt.Printf("%s  %s  %s\n", item.Timestamp.Format("2006-01-02 15:04:05"), item.Kind, item.Message)
			if len(item.Fields) == 0 {
				continue
			}
			keys := make([]string, 0, len(item.Fields))
			for key := range item.Fields {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				value := item.Fields[key]
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return
	}

	logPath := store.LogPath(*specID)
	if *pathOnly {
		fmt.Println(logPath)
		return
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading runtime log: %v\n", err)
		os.Exit(1)
	}
	text := string(data)
	if *tail > 0 {
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		if len(lines) > *tail {
			lines = lines[len(lines)-*tail:]
		}
		text = strings.Join(lines, "\n")
		if text != "" {
			text += "\n"
		}
	}
	fmt.Print(text)
}
