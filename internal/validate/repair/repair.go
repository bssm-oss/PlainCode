// Package repair implements the failure analysis and repair loop.
//
// When a build produces code that fails tests or doesn't meet coverage targets,
// the repair loop:
// 1. Classifies the failure (spec issue vs implementation bug vs missing test)
// 2. Generates a repair prompt with failure context
// 3. Re-executes the backend with the repair prompt
// 4. Validates the result again
// 5. Repeats up to the configured retry limit
package repair

import "fmt"

// FailureKind classifies what went wrong.
type FailureKind int

const (
	// TestFailure means existing tests failed on the generated code.
	TestFailure FailureKind = iota

	// CoverageGap means tests pass but coverage is below target.
	CoverageGap

	// OwnershipViolation means the backend tried to modify files outside its scope.
	OwnershipViolation

	// BuildError means the generated code doesn't compile.
	BuildError

	// UnknownFailure is an unclassified failure.
	UnknownFailure
)

// String returns a human-readable name for the failure kind.
func (k FailureKind) String() string {
	switch k {
	case TestFailure:
		return "test_failure"
	case CoverageGap:
		return "coverage_gap"
	case OwnershipViolation:
		return "ownership_violation"
	case BuildError:
		return "build_error"
	default:
		return "unknown"
	}
}

// Failure describes a specific build failure.
type Failure struct {
	Kind    FailureKind
	Message string
	Details string // test output, compiler errors, etc.
}

// RepairContext holds all information needed for a repair attempt.
type RepairContext struct {
	OriginalPrompt string
	Failures       []Failure
	Attempt        int
	MaxAttempts    int
}

// ShouldRetry returns true if there are attempts remaining.
func (rc *RepairContext) ShouldRetry() bool {
	return rc.Attempt < rc.MaxAttempts
}

// Classify analyzes build results and returns categorized failures.
//
// TODO: Implement with actual test output parsing, coverage report analysis,
// and ownership violation detection.
func Classify(testOutput string, testPassed bool, coveragePercent float64, coverageTarget float64, ownershipErrors []string) []Failure {
	var failures []Failure

	if len(ownershipErrors) > 0 {
		for _, err := range ownershipErrors {
			failures = append(failures, Failure{
				Kind:    OwnershipViolation,
				Message: err,
			})
		}
	}

	if !testPassed {
		failures = append(failures, Failure{
			Kind:    TestFailure,
			Message: "tests failed",
			Details: testOutput,
		})
	}

	if testPassed && coveragePercent < coverageTarget {
		failures = append(failures, Failure{
			Kind:    CoverageGap,
			Message: fmt.Sprintf("coverage %.1f%% below target %.1f%%", coveragePercent*100, coverageTarget*100),
		})
	}

	return failures
}

// BuildRepairPrompt generates a prompt for the backend to fix the failures.
//
// TODO: Implement with structured failure context and specific repair instructions
// based on failure kind.
func BuildRepairPrompt(ctx RepairContext) string {
	var prompt string
	prompt += fmt.Sprintf("# Repair Attempt %d/%d\n\n", ctx.Attempt+1, ctx.MaxAttempts)
	prompt += "The previous build produced failures that need to be fixed.\n\n"

	for i, f := range ctx.Failures {
		prompt += fmt.Sprintf("## Failure %d: %s\n\n", i+1, f.Kind)
		prompt += f.Message + "\n\n"
		if f.Details != "" {
			prompt += "```\n" + f.Details + "\n```\n\n"
		}
	}

	prompt += "Please fix the issues above. " +
		"Do not modify files outside the owned/shared scope. " +
		"Ensure all tests pass and coverage meets the target.\n"

	return prompt
}
