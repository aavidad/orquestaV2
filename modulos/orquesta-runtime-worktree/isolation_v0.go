package orquestaruntimeworktree

import (
	"context"
	"strings"
)

const (
	WorktreeIsolationSchemaVersionV0 = "worktree_isolation.v0"
	WorktreeIsolationModeIsolatedV0  = "isolated"
)

type WorktreeIsolationRequestV0 struct {
	IsolationRef         string   `json:"isolation_ref"`
	ProjectRef           string   `json:"project_ref"`
	WorktreeRef          string   `json:"worktree_ref"`
	BranchRef            string   `json:"branch_ref"`
	ProjectWorkDir       string   `json:"project_work_dir"`
	Isolated             bool     `json:"isolated"`
	BaselineRef          string   `json:"baseline_ref,omitempty"`
	IgnorePrefixes       []string `json:"ignore_prefixes,omitempty"`
	AllowPartialSnapshot bool     `json:"allow_partial_snapshot,omitempty"`
}

type WorktreeIsolationV0 struct {
	SchemaVersion string             `json:"schema_version"`
	IsolationRef  string             `json:"isolation_ref"`
	ProjectRef    string             `json:"project_ref"`
	WorktreeRef   string             `json:"worktree_ref"`
	BranchRef     string             `json:"branch_ref"`
	Mode          string             `json:"mode"`
	BaselineRef   string             `json:"baseline_ref"`
	Snapshot      WorktreeSnapshotV0 `json:"snapshot"`
	EvidenceRefs  []string           `json:"evidence_refs,omitempty"`
}

func PrepareIsolatedWorktreeV0(
	ctx context.Context,
	request WorktreeIsolationRequestV0,
) (WorktreeIsolationV0, []WorktreeIssueV0) {
	request = normalizeWorktreeIsolationRequestV0(request)
	if issues := validateWorktreeIsolationRequestV0(request); len(issues) > 0 {
		return WorktreeIsolationV0{}, issues
	}
	snapshot, issues := CaptureWorktreeSnapshotV0(ctx, WorktreeSnapshotRequestV0{
		SnapshotRef:    worktreeIsolationBaselineRefV0(request),
		ProjectWorkDir: request.ProjectWorkDir,
		IgnorePrefixes: request.IgnorePrefixes,
		AllowPartial:   request.AllowPartialSnapshot,
	})
	if len(issues) > 0 {
		return WorktreeIsolationV0{}, issues
	}
	return WorktreeIsolationV0{
		SchemaVersion: WorktreeIsolationSchemaVersionV0,
		IsolationRef:  request.IsolationRef,
		ProjectRef:    request.ProjectRef,
		WorktreeRef:   request.WorktreeRef,
		BranchRef:     request.BranchRef,
		Mode:          WorktreeIsolationModeIsolatedV0,
		BaselineRef:   snapshot.SnapshotRef,
		Snapshot:      snapshot,
		EvidenceRefs:  worktreeIsolationEvidenceRefsV0(snapshot),
	}, nil
}

func normalizeWorktreeIsolationRequestV0(
	request WorktreeIsolationRequestV0,
) WorktreeIsolationRequestV0 {
	request.IsolationRef = strings.TrimSpace(request.IsolationRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.BranchRef = strings.TrimSpace(request.BranchRef)
	request.ProjectWorkDir = strings.TrimSpace(request.ProjectWorkDir)
	request.BaselineRef = strings.TrimSpace(request.BaselineRef)
	request.IgnorePrefixes, _ = normalizeWorktreePathListV0(request.IgnorePrefixes, false)
	return request
}

func validateWorktreeIsolationRequestV0(
	request WorktreeIsolationRequestV0,
) []WorktreeIssueV0 {
	var issues []WorktreeIssueV0
	for _, item := range []struct {
		field string
		value string
	}{
		{field: "isolation_ref", value: request.IsolationRef},
		{field: "project_ref", value: request.ProjectRef},
		{field: "worktree_ref", value: request.WorktreeRef},
		{field: "branch_ref", value: request.BranchRef},
	} {
		issues = append(issues, validateWorktreeOpaqueRefV0(item.field, item.value, true)...)
	}
	issues = append(issues, validateWorktreeOpaqueRefV0("baseline_ref", request.BaselineRef, false)...)
	if !request.Isolated {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "isolated"))
	}
	return issues
}

func worktreeIsolationBaselineRefV0(request WorktreeIsolationRequestV0) string {
	if request.BaselineRef != "" {
		return request.BaselineRef
	}
	return request.IsolationRef + "-baseline"
}

func worktreeIsolationEvidenceRefsV0(snapshot WorktreeSnapshotV0) []string {
	refs := []string{"evidence-ref-worktree-isolation-v0"}
	refs = append(refs, worktreeSnapshotBudgetEvidenceRefsV0(snapshot.ExclusionReceipts)...)
	return compactWorktreeStringsV0(refs)
}
