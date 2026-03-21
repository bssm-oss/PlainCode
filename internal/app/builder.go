package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
	"github.com/bssm-oss/PlainCode/internal/config"
	"github.com/bssm-oss/PlainCode/internal/contextpack"
	"github.com/bssm-oss/PlainCode/internal/graph"
	"github.com/bssm-oss/PlainCode/internal/hooks"
	"github.com/bssm-oss/PlainCode/internal/policy"
	"github.com/bssm-oss/PlainCode/internal/receipt"
	"github.com/bssm-oss/PlainCode/internal/spec/ast"
	vcoverage "github.com/bssm-oss/PlainCode/internal/validate/coverage"
	"github.com/bssm-oss/PlainCode/internal/validate/repair"
	vtest "github.com/bssm-oss/PlainCode/internal/validate/test"
	"github.com/bssm-oss/PlainCode/internal/workspace/fsguard"
	"github.com/bssm-oss/PlainCode/internal/workspace/patch"
	prompttpl "github.com/bssm-oss/PlainCode/prompts"
	"github.com/google/uuid"
)

// BuildOptions controls the build pipeline behavior.
type BuildOptions struct {
	SpecID       string // empty = build all dirty specs
	DryRun       bool
	SkipTests    bool
	SkipCoverage bool
	JSONOutput   bool
	MaxRetries   int
}

// BuildResult holds the outcome of a full build pipeline run.
type BuildResult struct {
	SpecID     string           `json:"spec_id"`
	BuildID    string           `json:"build_id"`
	Status     string           `json:"status"` // success, failed, skipped
	Receipt    *receipt.Receipt `json:"receipt,omitempty"`
	Error      string           `json:"error,omitempty"`
	Skipped    bool             `json:"skipped,omitempty"`
	SkipReason string           `json:"skip_reason,omitempty"`
}

type repairAttemptArtifact struct {
	Attempt  int              `json:"attempt"`
	Prompt   string           `json:"prompt"`
	Failures []repair.Failure `json:"failures"`
}

type buildArtifacts struct {
	TestResult     *vtest.Result             `json:"test_result,omitempty"`
	CoverageReport *vcoverage.CoverageReport `json:"coverage_report,omitempty"`
	RepairAttempts []repairAttemptArtifact   `json:"repair_attempts,omitempty"`
}

// Builder orchestrates the full build pipeline.
type Builder struct {
	cfg        *config.ProjectConfig
	registry   *core.Registry
	store      *receipt.Store
	hooks      *hooks.Runner
	projectDir string
}

// NewBuilder creates a build pipeline orchestrator.
func NewBuilder(cfg *config.ProjectConfig, registry *core.Registry, projectDir string) *Builder {
	stateDir := filepath.Join(projectDir, cfg.Project.StateDir)
	return &Builder{
		cfg:        cfg,
		registry:   registry,
		store:      receipt.NewStore(stateDir),
		hooks:      hooks.NewRunner(projectDir),
		projectDir: projectDir,
	}
}

// Build runs the full build pipeline.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) ([]BuildResult, error) {
	specDir := filepath.Join(b.projectDir, b.cfg.Project.SpecDir)
	loaded, err := LoadSpecs(specDir)
	if err != nil {
		return nil, fmt.Errorf("loading specs: %w", err)
	}

	for _, le := range loaded.Errors {
		fmt.Fprintf(nil, "  warning: %s: %v\n", le.Path, le.Err) // TODO: proper logger
	}

	if len(loaded.Specs) == 0 {
		return nil, fmt.Errorf("no specs found in %s", specDir)
	}

	var toBuild []*graph.Node
	if opts.SpecID != "" {
		node := loaded.Graph.Node(opts.SpecID)
		if node == nil {
			return nil, fmt.Errorf("spec not found: %s", opts.SpecID)
		}
		node.IsDirty = true
		toBuild = []*graph.Node{node}
	} else {
		hashes, _ := b.store.SpecHashes()
		loaded.Graph.MarkDirty(hashes)
		toBuild, err = loaded.Graph.DirtyNodes()
		if err != nil {
			return nil, fmt.Errorf("computing dirty nodes: %w", err)
		}
	}

	if len(toBuild) == 0 {
		return []BuildResult{{
			Status:     "skipped",
			Skipped:    true,
			SkipReason: "no dirty specs",
		}}, nil
	}

	var results []BuildResult
	for _, node := range toBuild {
		result := b.buildSpec(ctx, node.Spec, loaded, opts)
		results = append(results, result)
		if result.Status == "failed" && !opts.DryRun {
			break
		}
	}

	return results, nil
}

// buildSpec runs the pipeline for a single spec.
func (b *Builder) buildSpec(ctx context.Context, spec *ast.Spec, loaded *LoadResult, opts BuildOptions) BuildResult {
	buildID := uuid.New().String()[:8]
	startedAt := time.Now()
	result := BuildResult{
		SpecID:  spec.ID,
		BuildID: buildID,
	}

	if opts.DryRun {
		result.Status = "success"
		result.SkipReason = "dry-run"
		return result
	}

	assembler := contextpack.NewAssembler(b.projectDir, filepath.Join(b.projectDir, b.cfg.Project.SpecDir))
	importedSpecs := make(map[string]*ast.Spec)
	for _, impID := range spec.Imports {
		if s, ok := loaded.Specs[impID]; ok {
			importedSpecs[impID] = s
		}
	}
	ctxPack, err := assembler.Assemble(spec, importedSpecs)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("assembling context: %v", err)
		return result
	}

	preferred := spec.Backend.Preferred
	if len(preferred) == 0 {
		preferred = []string{b.cfg.Defaults.Backend}
	}
	backend, err := b.registry.Select(preferred, b.cfg.Defaults.Backend)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("selecting backend: %v", err)
		return result
	}

	approvalName := spec.Approval
	if approvalName == "" {
		approvalName = b.cfg.Defaults.Approval
	}
	profile, err := policy.Resolve(approvalName)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("resolving policy: %v", err)
		return result
	}

	ctxPack.PolicySummary = b.policySummary(profile)
	ctxPack.BackendCapabilities = b.capabilitySummary(backend.Capabilities())

	if _, err := b.hooks.Run(ctx, hooks.PreExec, map[string]string{
		"PLAINCODE_SPEC_ID":  spec.ID,
		"PLAINCODE_BUILD_ID": buildID,
	}); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("pre-exec hook failed: %v", err)
		return result
	}

	maxTurns := spec.Budget.MaxTurns
	if maxTurns == 0 {
		maxTurns = 10
	}
	maxCost := spec.Budget.MaxCostUSD
	if maxCost == 0 {
		maxCost = 5.0
	}

	retryLimit := opts.MaxRetries
	if retryLimit < 0 {
		retryLimit = 0
	}
	managed := managedPaths(spec.ManagedFiles.Owned, spec.ManagedFiles.Shared, spec.ManagedFiles.Readonly)
	artifacts := &buildArtifacts{}
	var lastFailures []repair.Failure
	var failingTests string
	var coverageGaps string
	var latestError error
	var changedFiles []string
	var testRuns int
	var testsPassed int
	var testsFailed int
	var coverageAfter float64

	for attempt := 0; attempt <= retryLimit; attempt++ {
		attemptPack := *ctxPack
		attemptPack.FailingTests = failingTests
		attemptPack.CoverageGaps = coverageGaps
		prompt := b.buildPrompt(&attemptPack, attempt, retryLimit, lastFailures)

		execReq := &core.ExecRequest{
			BuildID: buildID,
			PromptPack: core.PromptPack{
				SpecBody:      prompt,
				AgentsRules:   attemptPack.AgentRules,
				OwnedFiles:    attemptPack.OwnedFiles,
				SharedFiles:   attemptPack.SharedFiles,
				ReadonlyFiles: attemptPack.ReadonlyFiles,
			},
			WorkspaceDir:    b.projectDir,
			ApprovalProfile: core.ApprovalProfile(profile.Name),
			Budget: core.Budget{
				MaxTurns:   maxTurns,
				MaxCostUSD: maxCost,
			},
		}

		execResult, execErr := backend.Execute(ctx, execReq, nil)
		if execErr != nil {
			latestError = fmt.Errorf("backend execution: %w", execErr)
			break
		}

		ops := append([]core.PatchOp{}, execResult.Patches...)
		ops = append(ops, staleOwnedDeleteOps(spec.ManagedFiles.Owned, execResult.Patches)...)
		rollbackState, stateErr := captureWorkspaceState(b.projectDir, append(managed, patchPathsForSnapshot(ops)...))
		if stateErr != nil {
			latestError = fmt.Errorf("capturing workspace state: %w", stateErr)
			break
		}

		var patchTargetPaths []string
		for _, op := range ops {
			patchTargetPaths = append(patchTargetPaths, op.Path())
		}
		if err := fsguard.ValidatePatch(spec.ID, patchTargetPaths, loaded.Ownership); err != nil {
			latestError = fmt.Errorf("ownership violation: %w", err)
			lastFailures = repair.Classify("", true, 0, 0, []string{err.Error()})
			artifacts.RepairAttempts = append(artifacts.RepairAttempts, repairAttemptArtifact{
				Attempt:  attempt + 1,
				Prompt:   prompt,
				Failures: lastFailures,
			})
			failingTests = ""
			coverageGaps = ""
			if attempt < retryLimit {
				continue
			}
			break
		}

		applier := patch.NewApplier(b.projectDir)
		if err := applier.ApplyAll(ops); err != nil {
			_ = rollbackState.Restore()
			latestError = fmt.Errorf("applying patches: %w", err)
			break
		}

		if _, err := b.hooks.Run(ctx, hooks.PostApply, map[string]string{
			"PLAINCODE_SPEC_ID":  spec.ID,
			"PLAINCODE_BUILD_ID": buildID,
		}); err != nil {
			_ = rollbackState.Restore()
			latestError = fmt.Errorf("post-apply hook failed: %w", err)
			break
		}

		changedFiles = changedFilesFromOps(applier.Applied())
		failingTests = ""
		coverageGaps = ""
		artifacts.TestResult = nil
		artifacts.CoverageReport = nil
		testRuns, testsPassed, testsFailed = 0, 0, 0
		coverageAfter = 0

		if !opts.SkipTests && spec.Tests.Command != "" {
			if _, err := b.hooks.Run(ctx, hooks.PreTest, map[string]string{
				"PLAINCODE_SPEC_ID":  spec.ID,
				"PLAINCODE_BUILD_ID": buildID,
			}); err != nil {
				_ = rollbackState.Restore()
				latestError = fmt.Errorf("pre-test hook failed: %w", err)
				break
			}

			testResult, err := vtest.NewRunner().Run(ctx, b.projectDir, spec.Tests.Command)
			if err != nil {
				_ = rollbackState.Restore()
				latestError = fmt.Errorf("running tests: %w", err)
				break
			}
			artifacts.TestResult = testResult
			testRuns = 1
			if testResult.Passed {
				testsPassed = 1
			} else {
				testsFailed = 1
			}

			if _, err := b.hooks.Run(ctx, hooks.PostTest, map[string]string{
				"PLAINCODE_SPEC_ID":   spec.ID,
				"PLAINCODE_BUILD_ID":  buildID,
				"PLAINCODE_TEST_EXIT": fmt.Sprintf("%d", testResult.ExitCode),
			}); err != nil {
				_ = rollbackState.Restore()
				latestError = fmt.Errorf("post-test hook failed: %w", err)
				break
			}

			if !testResult.Passed {
				_ = rollbackState.Restore()
				failingTests = strings.TrimSpace(testResult.Stdout + "\n" + testResult.Stderr)
				lastFailures = repair.Classify(failingTests, false, 0, spec.Coverage.Target, nil)
				artifacts.RepairAttempts = append(artifacts.RepairAttempts, repairAttemptArtifact{
					Attempt:  attempt + 1,
					Prompt:   prompt,
					Failures: lastFailures,
				})
				latestError = fmt.Errorf("tests failed: exit code %d", testResult.ExitCode)
				if attempt < retryLimit {
					continue
				}
				break
			}
		}

		if shouldRunCoverage(spec, opts) {
			report, err := vcoverage.NewGoProvider().RunUnit(ctx, b.projectDir)
			if err != nil {
				_ = rollbackState.Restore()
				latestError = fmt.Errorf("running coverage: %w", err)
				break
			}
			artifacts.CoverageReport = report
			coverageAfter = report.Percentage
			if report.Percentage < spec.Coverage.Target {
				_ = rollbackState.Restore()
				gaps := vcoverage.NewGoProvider().FindGaps(report)
				coverageGaps = formatCoverageGaps(gaps)
				lastFailures = repair.Classify("", true, report.Percentage, spec.Coverage.Target, nil)
				artifacts.RepairAttempts = append(artifacts.RepairAttempts, repairAttemptArtifact{
					Attempt:  attempt + 1,
					Prompt:   prompt,
					Failures: lastFailures,
				})
				latestError = fmt.Errorf("coverage %.2f below target %.2f", report.Percentage, spec.Coverage.Target)
				if attempt < retryLimit {
					continue
				}
				break
			}
		}

		r := &receipt.Receipt{
			BuildID:         buildID,
			SpecID:          spec.ID,
			SpecHash:        spec.Hash,
			BackendID:       backend.ID(),
			ApprovalProfile: profile.Name,
			ChangedFiles:    changedFiles,
			TestsRun:        testRuns,
			TestsPassed:     testsPassed,
			TestsFailed:     testsFailed,
			CoverageAfter:   coverageAfter,
			Retries:         attempt,
			InputTokens:     execResult.Usage.InputTokens,
			OutputTokens:    execResult.Usage.OutputTokens,
			CostUSD:         execResult.Usage.CostUSD,
			DurationMS:      time.Since(startedAt).Milliseconds(),
			Status:          "success",
			StartedAt:       startedAt,
			CompletedAt:     time.Now(),
		}

		if err := b.saveArtifacts(buildID, artifacts); err != nil {
			_ = rollbackState.Restore()
			result.Status = "failed"
			result.Error = fmt.Sprintf("saving artifacts: %v", err)
			return result
		}
		if err := b.store.Save(r); err != nil {
			_ = rollbackState.Restore()
			result.Status = "failed"
			result.Error = fmt.Sprintf("saving receipt: %v", err)
			return result
		}

		if _, err := b.hooks.Run(ctx, hooks.OnReceipt, map[string]string{
			"PLAINCODE_BUILD_ID": buildID,
			"PLAINCODE_STATUS":   "success",
		}); err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("on-receipt hook failed: %v", err)
			return result
		}

		result.Status = "success"
		result.Receipt = r
		return result
	}

	failedReceipt := &receipt.Receipt{
		BuildID:         buildID,
		SpecID:          spec.ID,
		SpecHash:        spec.Hash,
		BackendID:       backend.ID(),
		ApprovalProfile: profile.Name,
		ChangedFiles:    nil,
		TestsRun:        testRuns,
		TestsPassed:     testsPassed,
		TestsFailed:     testsFailed,
		CoverageAfter:   coverageAfter,
		Retries:         len(artifacts.RepairAttempts),
		DurationMS:      time.Since(startedAt).Milliseconds(),
		Status:          "failed",
		Error:           errorString(latestError),
		StartedAt:       startedAt,
		CompletedAt:     time.Now(),
	}

	if err := b.saveArtifacts(buildID, artifacts); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("saving artifacts: %v", err)
		return result
	}
	if err := b.store.Save(failedReceipt); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("saving receipt: %v", err)
		return result
	}
	_, _ = b.hooks.Run(ctx, hooks.OnError, map[string]string{
		"PLAINCODE_ERROR": failedReceipt.Error,
	})

	result.Status = "failed"
	result.Error = failedReceipt.Error
	result.Receipt = failedReceipt
	return result
}

func (b *Builder) buildPrompt(pack *contextpack.ContextPack, attempt, retryLimit int, failures []repair.Failure) string {
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(prompttpl.BuildSystem))
	sb.WriteString("\n\n")
	sb.WriteString(strings.TrimSpace(pack.ToPrompt()))

	if attempt > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(repair.BuildRepairPrompt(repair.RepairContext{
			OriginalPrompt: sb.String(),
			Failures:       failures,
			Attempt:        attempt,
			MaxAttempts:    retryLimit + 1,
		}))
	}

	return sb.String()
}

func (b *Builder) saveArtifacts(buildID string, artifacts *buildArtifacts) error {
	if artifacts.TestResult != nil {
		data, err := json.MarshalIndent(artifacts.TestResult, "", "  ")
		if err != nil {
			return err
		}
		if err := b.store.SaveArtifact(buildID, "tests.json", data); err != nil {
			return err
		}
	}
	if artifacts.CoverageReport != nil {
		data, err := json.MarshalIndent(artifacts.CoverageReport, "", "  ")
		if err != nil {
			return err
		}
		if err := b.store.SaveArtifact(buildID, "coverage.json", data); err != nil {
			return err
		}
	}
	if len(artifacts.RepairAttempts) > 0 {
		data, err := json.MarshalIndent(artifacts.RepairAttempts, "", "  ")
		if err != nil {
			return err
		}
		if err := b.store.SaveArtifact(buildID, "repair.json", data); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) policySummary(profile policy.Profile) string {
	return fmt.Sprintf("name=%s tools=%s file_write=%s shell=%s network=%s",
		profile.Name, profile.Tools, profile.FileWrite, profile.Shell, profile.Network)
}

func (b *Builder) capabilitySummary(caps core.CapabilitySet) string {
	return fmt.Sprintf("structured=%t streaming=%t mcp=%t tools=%t patches=%t json_schema=%t",
		caps.StructuredOutput, caps.StreamingEvents, caps.MCP, caps.Tools, caps.FilePatchMode, caps.JsonSchemaOutput)
}

func shouldRunCoverage(spec *ast.Spec, opts BuildOptions) bool {
	return !opts.SkipCoverage && spec.Language == "go" && spec.Coverage.Target > 0
}

func staleOwnedDeleteOps(owned []string, ops []core.PatchOp) []core.PatchOp {
	desired := make(map[string]struct{})
	for _, op := range ops {
		switch patch := op.(type) {
		case core.WriteFile:
			desired[filepath.Clean(patch.FilePath)] = struct{}{}
		case core.ApplyDiff:
			desired[filepath.Clean(patch.FilePath)] = struct{}{}
		case core.RenameFile:
			desired[filepath.Clean(patch.To)] = struct{}{}
		}
	}

	var deletes []core.PatchOp
	for _, path := range owned {
		clean := filepath.Clean(path)
		if _, ok := desired[clean]; ok {
			continue
		}
		deletes = append(deletes, core.DeleteFile{FilePath: clean})
	}
	return deletes
}

func changedFilesFromOps(ops []core.PatchOp) []string {
	var files []string
	for _, op := range ops {
		switch patch := op.(type) {
		case core.RenameFile:
			files = append(files, filepath.Clean(patch.From), filepath.Clean(patch.To))
		default:
			files = append(files, filepath.Clean(op.Path()))
		}
	}
	return uniquePaths(files)
}

func formatCoverageGaps(gaps []vcoverage.CoverageGap) string {
	if len(gaps) == 0 {
		return ""
	}
	lines := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		lines = append(lines, fmt.Sprintf("- %s:%d-%d %s", gap.File, gap.LineRange.Start, gap.LineRange.End, gap.Description))
	}
	return strings.Join(lines, "\n")
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
