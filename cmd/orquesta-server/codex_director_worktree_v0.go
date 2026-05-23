package main

import (
	"context"
	"path/filepath"
	"strings"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type codexDirectorWorktreeIsolationSummaryV0 struct {
	SchemaVersion string   `json:"schema_version,omitempty"`
	IsolationRef  string   `json:"isolation_ref,omitempty"`
	ProjectRef    string   `json:"project_ref,omitempty"`
	WorktreeRef   string   `json:"worktree_ref,omitempty"`
	BranchRef     string   `json:"branch_ref,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	BaselineRef   string   `json:"baseline_ref,omitempty"`
	SnapshotFiles int      `json:"snapshot_files,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

func codexDirectorDefaultBranchRefV0(waveRef string) string {
	return codexDirectorDefaultRefV0("", "branch", waveRef)
}

func codexDirectorPrepareWorktreeIsolationV0(
	ctx context.Context,
	config codexDirectorWaveConfigV0,
) (codexDirectorWorktreeIsolationSummaryV0, []orquestadirectoroperativo.OperationalDirectorIssueV0) {
	isolation, issues := orquestaruntimeworktree.PrepareIsolatedWorktreeV0(
		ctx,
		orquestaruntimeworktree.WorktreeIsolationRequestV0{
			IsolationRef:   codexDirectorDefaultRefV0("", "worktree-isolation", config.Wave.WaveRef),
			ProjectRef:     config.ProjectRef,
			WorktreeRef:    config.WorktreeRef,
			BranchRef:      config.BranchRef,
			ProjectWorkDir: config.Wave.ProjectWorkDir,
			Isolated:       true,
			BaselineRef:    codexDirectorDefaultRefV0("", "worktree-baseline", config.Wave.WaveRef),
			IgnorePrefixes: codexDirectorWorktreeIgnorePrefixesV0(config),
		},
	)
	summary := codexDirectorWorktreeIsolationSummaryFromV0(isolation)
	if len(issues) == 0 {
		return summary, nil
	}
	return summary, codexDirectorWorktreeIssuesV0(issues)
}

func codexDirectorWorktreeIsolationSummaryFromV0(
	isolation orquestaruntimeworktree.WorktreeIsolationV0,
) codexDirectorWorktreeIsolationSummaryV0 {
	if isolation.SchemaVersion == "" {
		return codexDirectorWorktreeIsolationSummaryV0{}
	}
	return codexDirectorWorktreeIsolationSummaryV0{
		SchemaVersion: isolation.SchemaVersion,
		IsolationRef:  isolation.IsolationRef,
		ProjectRef:    isolation.ProjectRef,
		WorktreeRef:   isolation.WorktreeRef,
		BranchRef:     isolation.BranchRef,
		Mode:          isolation.Mode,
		BaselineRef:   isolation.BaselineRef,
		SnapshotFiles: len(isolation.Snapshot.Files),
		EvidenceRefs:  append([]string(nil), isolation.EvidenceRefs...),
	}
}

func codexDirectorWorktreeIssuesV0(
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) []orquestadirectoroperativo.OperationalDirectorIssueV0 {
	out := make([]orquestadirectoroperativo.OperationalDirectorIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestadirectoroperativo.OperationalDirectorIssueV0{
			Code:    "worktree_isolation_invalid",
			Field:   issue.Field,
			Message: string(issue.Code),
		})
	}
	return out
}

func codexDirectorWorktreeIgnorePrefixesV0(config codexDirectorWaveConfigV0) []string {
	rel, ok := codexWaveRelativePathV0(config.Wave.ProjectWorkDir, config.Wave.RuntimeWorkDir)
	if !ok {
		return nil
	}
	return []string{rel}
}

func codexWaveRelativePathV0(root string, path string) (string, bool) {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
