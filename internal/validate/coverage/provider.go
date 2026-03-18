package coverage

import "context"

// CoverageReport holds the results of a coverage run.
type CoverageReport struct {
	Language    string
	TotalLines  int
	CoveredLines int
	Percentage  float64
	Files       map[string]FileCoverage
}

// FileCoverage holds per-file coverage data.
type FileCoverage struct {
	Path         string
	TotalLines   int
	CoveredLines int
	UncoveredRanges []LineRange
}

// LineRange represents a contiguous range of lines.
type LineRange struct {
	Start int
	End   int
}

// CoverageGap identifies a specific area lacking test coverage.
type CoverageGap struct {
	File        string
	Function    string
	LineRange   LineRange
	Description string
}

// Provider abstracts language-specific coverage tools.
type Provider interface {
	// Language returns the language this provider handles.
	Language() string

	// RunUnit runs unit tests and collects coverage.
	RunUnit(ctx context.Context, dir string) (*CoverageReport, error)

	// RunIntegration runs integration tests and collects coverage.
	RunIntegration(ctx context.Context, dir string) (*CoverageReport, error)

	// FindGaps identifies areas that lack test coverage.
	FindGaps(report *CoverageReport) []CoverageGap
}
