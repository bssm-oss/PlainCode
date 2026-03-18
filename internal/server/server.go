// Package server implements the Forge HTTP daemon.
//
// The daemon exposes a JSON API for builds, traces, and provider management,
// plus an SSE event stream for real-time build progress.
//
// Endpoints:
//
//	POST /build          - Trigger a spec build
//	POST /change         - Submit an implementation change request
//	POST /takeover       - Start a takeover pipeline
//	POST /coverage       - Run coverage analysis
//	GET  /builds/:id     - Get build receipt
//	GET  /events         - SSE event stream
//	GET  /providers      - List configured backends
//	GET  /policies       - List approval profiles
//	GET  /health         - Health check
//	GET  /openapi.json   - OpenAPI 3.1 specification
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Server is the Forge HTTP daemon.
type Server struct {
	addr string
	mux  *http.ServeMux
}

// New creates a new server.
func New(addr string) *Server {
	s := &Server{
		addr: addr,
		mux:  http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /providers", s.handleProviders)
	s.mux.HandleFunc("GET /policies", s.handlePolicies)
	s.mux.HandleFunc("GET /openapi.json", s.handleOpenAPI)

	// TODO: Implement these endpoints
	s.mux.HandleFunc("POST /build", s.handleNotImplemented)
	s.mux.HandleFunc("POST /change", s.handleNotImplemented)
	s.mux.HandleFunc("POST /takeover", s.handleNotImplemented)
	s.mux.HandleFunc("POST /coverage", s.handleNotImplemented)
	s.mux.HandleFunc("GET /builds/", s.handleNotImplemented)
	s.mux.HandleFunc("GET /events", s.handleNotImplemented)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	fmt.Printf("PlainCode daemon listening on %s\n", s.addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /health")
	fmt.Println("  GET  /providers")
	fmt.Println("  GET  /policies")
	fmt.Println("  GET  /openapi.json")
	fmt.Println("  POST /build         (not yet implemented)")
	fmt.Println("  POST /change        (not yet implemented)")
	fmt.Println("  POST /takeover      (not yet implemented)")
	fmt.Println("  POST /coverage      (not yet implemented)")
	fmt.Println("  GET  /builds/:id    (not yet implemented)")
	fmt.Println("  GET  /events        (not yet implemented)")
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "0.1.0-dev",
	})
}

func (s *Server) handleProviders(w http.ResponseWriter, _ *http.Request) {
	// TODO: Return actual configured providers
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": []string{},
		"note":      "Provider listing not yet connected to registry",
	})
}

func (s *Server) handlePolicies(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"profiles": []string{"plan", "patch", "workspace-auto", "sandbox-auto", "full-trust"},
	})
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	// TODO: Generate from actual route definitions
	writeJSON(w, http.StatusOK, map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "PlainCode API",
			"version": "0.1.0-dev",
		},
		"paths": map[string]any{
			"/health":    map[string]any{"get": map[string]any{"summary": "Health check"}},
			"/providers": map[string]any{"get": map[string]any{"summary": "List backends"}},
			"/policies":  map[string]any{"get": map[string]any{"summary": "List approval profiles"}},
			"/build":     map[string]any{"post": map[string]any{"summary": "Trigger spec build"}},
		},
	})
}

func (s *Server) handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"error":   "not_implemented",
		"message": fmt.Sprintf("%s %s is not yet implemented", r.Method, r.URL.Path),
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
