// Package main is the daemon entrypoint for forge (plaincode serve).
//
// The daemon exposes an HTTP API with JSON endpoints and SSE event stream,
// enabling IDE plugins, CI pipelines, and web dashboards to interact
// with Forge programmatically.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bssm-oss/PlainCode/internal/server"
)

func main() {
	addr := flag.String("addr", ":8420", "Listen address")
	flag.Parse()

	srv := server.New(*addr)
	fmt.Fprintf(os.Stderr, "Starting PlainCode daemon...\n")
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
