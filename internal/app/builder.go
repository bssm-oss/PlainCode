package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/bssm-oss/PlainCode/internal/backend/core"
	"github.com/bssm-oss/PlainCode/internal/config"
	"github.com/bssm-oss/PlainCode/internal/contextpack"
	"github.com/bssm-oss/PlainCode/internal/graph"
	"github.com/bssm-oss/PlainCode/internal/hooks"
	"github.com/bssm-oss/PlainCode/internal/policy"
	"github.com/bssm-oss/PlainCode/internal/receipt"
	"github.com/bssm-oss/PlainCode/internal/spec/ast"
	"github.com/bssm-oss/PlainCode/internal/workspace/fsguard"
	"github.com/bssm-oss/PlainCode/internal/workspace/patch"
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
	SpecID    string         `json:"spec_id"`
	BuildID   string         `json:"build_id"`
	Status    string         `json:"status"` // success, failed, skipped
	Receipt   *receipt.Receipt `json:"receipt,omitempty"`
	Error     string         `json:"error,omitempty"`
	Skipped   bool           `json:"skipped,omitempty"`
	SkipReason string        `json:"skip_reason,omitempty"`
}

// Builder orchestrates the full build pipeline.
type Builder struct {
	cfg       *config.ProjectConfig
	registry  *core.Registry
	store     *receipt.Store
	hooks     *hooks.Runner
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
	// 1. Load specs
	specDir := filepath.Join(b.projectDir, b.cfg.Project.SpecDir)
	loaded, err := LoadSpecs(specDir)
	if err != nil {
		return nil, fmt.Errorf("loading specs: %w", err)
	}

	// Report parse errors
	for _, le := range loaded.Errors {
		fmt.Fprintf(nil, "  warning: %s: %v\n", le.Path, le.Err) // TODO: proper logger
	}

	if len(loaded.Specs) == 0 {
		return nil, fmt.Errorf("no specs found in %s", specDir)
	}

	// 2. Determine which specs to build
	var toBuild []*graph.Node

	if opts.SpecID != "" {
		// Single spec mode
		node := loaded.Graph.Node(opts.SpecID)
		if node == nil {
			return nil, fmt.Errorf("spec not found: %s", opts.SpecID)
		}
		node.IsDirty = true // always build when explicitly requested
		toBuild = []*graph.Node{node}
	} else {
		// All dirty specs
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

	// 3. Build each spec in dependency order
	var results []BuildResult
	for _, node := range toBuild {
		result := b.buildSpec(ctx, node.Spec, loaded, opts)
		results = append(results, result)

		if result.Status == "failed" && !opts.DryRun {
			// Stop on first failure (fail-fast)
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

	// Dry-run: just validate
	if opts.DryRun {
		result.Status = "success"
		result.SkipReason = "dry-run"
		return result
	}

	// a. Context pack
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

	// b. Select backend
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

	// c. Resolve policy
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

	// d. Pre-exec hooks
	b.hooks.Run(ctx, hooks.PreExec, map[string]string{
		"PLAINCODE_SPEC_ID": spec.ID,
		"PLAINCODE_BUILD_ID": buildID,
	})

	// e. Execute backend
	maxTurns := spec.Budget.MaxTurns
	if maxTurns == 0 {
		maxTurns = 10
	}
	maxCost := spec.Budget.MaxCostUSD
	if maxCost == 0 {
		maxCost = 5.0
	}

	execReq := &core.ExecRequest{
		BuildID: buildID,
		PromptPack: core.PromptPack{
			SpecBody:      ctxPack.SpecBody,
			AgentsRules:   ctxPack.AgentRules,
			OwnedFiles:    ctxPack.OwnedFiles,
			SharedFiles:   ctxPack.SharedFiles,
			ReadonlyFiles: ctxPack.ReadonlyFiles,
		},
		WorkspaceDir:    b.projectDir,
		ApprovalProfile: core.ApprovalProfile(profile.Name),
		Budget: core.Budget{
			MaxTurns:   maxTurns,
			MaxCostUSD: maxCost,
		},
	}

	execResult, err := backend.Execute(ctx, execReq, nil)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("backend execution: %v", err)
		return result
	}

	// f. Validate patches against ownership
	var patchPaths []string
	for _, p := range execResult.Patches {
		patchPaths = append(patchPaths, p.Path())
	}

	if err := fsguard.ValidatePatch(spec.ID, patchPaths, loaded.Ownership); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("ownership violation: %v", err)
		b.hooks.Run(ctx, hooks.OnError, map[string]string{
			"PLAINCODE_ERROR": err.Error(),
		})
		return result
	}

	// g. Apply patches
	applier := patch.NewApplier(b.projectDir)
	if err := applier.ApplyAll(execResult.Patches); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("applying patches: %v", err)
		return result
	}

	// h. Post-apply hooks
	b.hooks.Run(ctx, hooks.PostApply, map[string]string{
		"PLAINCODE_SPEC_ID":  spec.ID,
		"PLAINCODE_BUILD_ID": buildID,
	})

	// i. Collect changed file paths
	var changedFiles []string
	for _, op := range applier.Applied() {
		changedFiles = append(changedFiles, op.Path())
	}

	// j. Save receipt
	r := &receipt.Receipt{
		BuildID:         buildID,
		SpecID:          spec.ID,
		SpecHash:        spec.Hash,
		BackendID:       backend.ID(),
		ApprovalProfile: profile.Name,
		ChangedFiles:    changedFiles,
		InputTokens:     execResult.Usage.InputTokens,
		OutputTokens:    execResult.Usage.OutputTokens,
		CostUSD:         execResult.Usage.CostUSD,
		DurationMS:      time.Since(startedAt).Milliseconds(),
		Status:          "success",
		StartedAt:       startedAt,
		CompletedAt:     time.Now(),
	}

	if !opts.SkipTests && spec.Tests.Command != "" {
		// TODO: Run tests and update receipt with test results
		// For now, mark as untested
		r.TestsRun = 0
	}

	if err := b.store.Save(r); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("saving receipt: %v", err)
		return result
	}

	// k. On-receipt hooks
	b.hooks.Run(ctx, hooks.OnReceipt, map[string]string{
		"PLAINCODE_BUILD_ID": buildID,
		"PLAINCODE_STATUS":   "success",
	})

	result.Status = "success"
	result.Receipt = r
	return result
}
