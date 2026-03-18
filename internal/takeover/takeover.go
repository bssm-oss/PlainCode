// Package takeover implements the code-to-spec extraction pipeline
// with round-trip verification (Takeover v2).
//
// The takeover process:
//  1. Analyze existing code (public API, imports, tests)
//  2. Generate a spec draft
//  3. Verify by round-trip: delete code → rebuild from spec → compare
//  4. Compute confidence score
//  5. Promote to managed-by-spec only if above threshold
//
// This package coordinates the pipeline; actual code analysis uses
// Go's ast package (for Go) or Tree-sitter (future, for multi-language).
package takeover

import (
	"fmt"
)

// DefaultConfidenceThreshold is the minimum score for auto-promotion.
const DefaultConfidenceThreshold = 0.9

// WarningThreshold is the minimum score for promotion with warnings.
const WarningThreshold = 0.7

// Target describes what code to take over.
type Target struct {
	// Path is the file or directory to analyze.
	Path string

	// Package is the Go package path (optional, for Go targets).
	Package string

	// Language is the target language (auto-detected if empty).
	Language string
}

// AnalysisResult holds the output of code analysis.
type AnalysisResult struct {
	// PublicAPI lists exported functions, types, and interfaces.
	PublicAPI []APISymbol

	// Imports lists package imports.
	Imports []string

	// TestFiles lists discovered test files.
	TestFiles []string

	// TestCount is the number of test functions found.
	TestCount int

	// BaselineCoverage is the current test coverage percentage.
	BaselineCoverage float64

	// BaselineTestsPassed is the number of passing tests.
	BaselineTestsPassed int

	// BaselineTestsFailed is the number of failing tests.
	BaselineTestsFailed int

	// Comments holds significant code comments and doc strings.
	Comments []string

	// SourceFiles lists all source files in the target.
	SourceFiles []string
}

// APISymbol represents an exported symbol in the public API.
type APISymbol struct {
	Kind      string // "func", "type", "interface", "const", "var"
	Name      string
	Signature string // function signature or type definition
}

// VerificationResult holds the outcome of round-trip verification.
type VerificationResult struct {
	// TestPassRate is the fraction of tests passing on rebuilt code.
	TestPassRate float64

	// APIMatchRate is the fraction of public API symbols preserved.
	APIMatchRate float64

	// CoverageRatio is rebuilt coverage / baseline coverage.
	CoverageRatio float64

	// BehaviorMatch is a measure of behavioral equivalence (0-1).
	BehaviorMatch float64

	// MissingSymbols lists API symbols not found in rebuilt code.
	MissingSymbols []string

	// NewSymbols lists unexpected symbols in rebuilt code.
	NewSymbols []string
}

// Score computes the overall confidence score from verification results.
// Weights: test=0.4, api=0.3, coverage=0.15, behavior=0.15
func (v *VerificationResult) Score() float64 {
	return 0.4*v.TestPassRate +
		0.3*v.APIMatchRate +
		0.15*v.CoverageRatio +
		0.15*v.BehaviorMatch
}

// Decision returns the takeover decision based on the confidence score.
func (v *VerificationResult) Decision(threshold float64) TakeoverDecision {
	score := v.Score()
	if score >= threshold {
		return DecisionPromote
	}
	if score >= WarningThreshold {
		return DecisionPromoteWithWarnings
	}
	return DecisionReject
}

// TakeoverDecision represents the outcome of a takeover attempt.
type TakeoverDecision int

const (
	DecisionPromote TakeoverDecision = iota
	DecisionPromoteWithWarnings
	DecisionReject
)

// String returns a human-readable decision name.
func (d TakeoverDecision) String() string {
	switch d {
	case DecisionPromote:
		return "promote"
	case DecisionPromoteWithWarnings:
		return "promote_with_warnings"
	case DecisionReject:
		return "reject"
	default:
		return "unknown"
	}
}

// Pipeline orchestrates the full takeover process.
type Pipeline struct {
	projectDir string
	specDir    string
	threshold  float64
}

// NewPipeline creates a takeover pipeline.
func NewPipeline(projectDir, specDir string, threshold float64) *Pipeline {
	if threshold <= 0 {
		threshold = DefaultConfidenceThreshold
	}
	return &Pipeline{
		projectDir: projectDir,
		specDir:    specDir,
		threshold:  threshold,
	}
}

// Run executes the full takeover pipeline for a target.
//
// TODO: Implement each stage:
// 1. Analyze(target) → AnalysisResult
// 2. GenerateSpec(analysis) → spec file
// 3. Verify(spec) → VerificationResult
// 4. Decide(verification) → TakeoverDecision
// 5. Promote(spec) if decided
func (p *Pipeline) Run(target Target) error {
	fmt.Printf("Takeover target: %s\n", target.Path)
	fmt.Println("[not yet implemented] Full takeover pipeline")
	fmt.Println("Stages planned:")
	fmt.Println("  1. Code analysis (public API, imports, tests)")
	fmt.Println("  2. Spec draft generation")
	fmt.Println("  3. Round-trip verification")
	fmt.Println("  4. Confidence scoring")
	fmt.Println("  5. Promotion decision")
	return nil
}
