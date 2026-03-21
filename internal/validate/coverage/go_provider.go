// Package coverage also provides language-specific implementations.
// This file implements the Go coverage provider using `go test -coverprofile`.
package coverage

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// GoProvider implements coverage collection for Go projects.
type GoProvider struct{}

// NewGoProvider creates a Go coverage provider.
func NewGoProvider() *GoProvider {
	return &GoProvider{}
}

// Language returns "go".
func (g *GoProvider) Language() string { return "go" }

// RunUnit runs `go test -coverprofile` and parses the results.
func (g *GoProvider) RunUnit(ctx context.Context, dir string) (*CoverageReport, error) {
	profilePath := filepath.Join(dir, ".plaincode", "coverage", "unit.out")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return nil, fmt.Errorf("creating coverage directory: %w", err)
	}

	// Safe execution via os/exec arg array — no shell
	cmd := exec.CommandContext(ctx, "go", "test", "./...",
		"-coverprofile="+profilePath,
		"-covermode=atomic",
	)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go test failed: %w\n%s", err, string(output))
	}

	report, err := parseCoverProfile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("parsing coverage profile: %w", err)
	}
	report.Language = "go"

	return report, nil
}

// RunIntegration runs Go integration coverage using `go build -cover`.
// NOTE: Requires Go 1.20+.
func (g *GoProvider) RunIntegration(ctx context.Context, dir string) (*CoverageReport, error) {
	// TODO: Implement integration coverage with `go build -cover`
	// This requires building the binary with coverage instrumentation,
	// running it, then collecting the GOCOVERDIR output.
	return nil, fmt.Errorf("Go integration coverage not yet implemented")
}

// FindGaps identifies functions and code paths lacking test coverage.
func (g *GoProvider) FindGaps(report *CoverageReport) []CoverageGap {
	var gaps []CoverageGap

	for path, fileCov := range report.Files {
		for _, r := range fileCov.UncoveredRanges {
			gaps = append(gaps, CoverageGap{
				File:        path,
				LineRange:   r,
				Description: fmt.Sprintf("lines %d-%d not covered", r.Start, r.End),
			})
		}
	}

	return gaps
}

// parseCoverProfile reads a Go coverage profile and produces a report.
// Format: mode: <mode>\n<file>:<start.col>,<end.col> <stmts> <count>
func parseCoverProfile(path string) (*CoverageReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading profile: %w", err)
	}

	report := &CoverageReport{
		Files: make(map[string]FileCoverage),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()

		// Skip mode line
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse: file:start.col,end.col stmts count
		colonIdx := strings.LastIndex(line, ":")
		if colonIdx < 0 {
			continue
		}

		file := line[:colonIdx]
		rest := line[colonIdx+1:]

		parts := strings.Fields(rest)
		if len(parts) < 3 {
			continue
		}

		stmts, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		fc := report.Files[file]
		fc.Path = file
		fc.TotalLines += stmts

		if count > 0 {
			fc.CoveredLines += stmts
		} else {
			// Parse line range from position
			positions := strings.Split(parts[0], ",")
			if len(positions) == 2 {
				startLine := parseLineNum(positions[0])
				endLine := parseLineNum(positions[1])
				if startLine > 0 && endLine > 0 {
					fc.UncoveredRanges = append(fc.UncoveredRanges, LineRange{
						Start: startLine,
						End:   endLine,
					})
				}
			}
		}

		report.Files[file] = fc
		report.TotalLines += stmts
		if count > 0 {
			report.CoveredLines += stmts
		}
	}

	if report.TotalLines > 0 {
		report.Percentage = float64(report.CoveredLines) / float64(report.TotalLines)
	}

	return report, nil
}

// parseLineNum extracts the line number from "line.col" format.
func parseLineNum(pos string) int {
	dotIdx := strings.Index(pos, ".")
	if dotIdx < 0 {
		n, _ := strconv.Atoi(pos)
		return n
	}
	n, _ := strconv.Atoi(pos[:dotIdx])
	return n
}
