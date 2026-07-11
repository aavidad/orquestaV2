package orquestaruntimeworktree

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const GoalWorkspaceSchemaVersionV0 = "goal_workspace.v0"

type GoalWorkspaceRequestV0 struct {
	RunRef        string `json:"run_ref"`
	GoalRef       string `json:"goal_ref"`
	ProjectRef    string `json:"project_ref"`
	WorktreeRef   string `json:"worktree_ref"`
	SourceWorkDir string `json:"source_work_dir"`
	WorkspaceRoot string `json:"workspace_root"`
	BaseRevision  string `json:"base_revision,omitempty"`
}

type GoalWorkspaceV0 struct {
	SchemaVersion  string   `json:"schema_version"`
	RunRef         string   `json:"run_ref"`
	GoalRef        string   `json:"goal_ref"`
	ProjectRef     string   `json:"project_ref"`
	WorktreeRef    string   `json:"worktree_ref"`
	WorkspaceID    string   `json:"workspace_id"`
	ProjectWorkDir string   `json:"project_work_dir"`
	BaseRevision   string   `json:"base_revision"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type GoalWorkspaceProvisionerPortV0 interface {
	PrepareGoalWorkspaceV0(context.Context, GoalWorkspaceRequestV0) (GoalWorkspaceV0, []WorktreeIssueV0)
	ResolveGoalWorkspaceV0(context.Context, GoalWorkspaceRequestV0) (GoalWorkspaceV0, []WorktreeIssueV0)
}

type GitGoalWorkspaceProvisionerV0 struct {
	VCS GitAppVCSConnectorV0
}

func (connector GitGoalWorkspaceProvisionerV0) PrepareGoalWorkspaceV0(
	ctx context.Context,
	request GoalWorkspaceRequestV0,
) (GoalWorkspaceV0, []WorktreeIssueV0) {
	request, issues := normalizeGoalWorkspaceRequestV0(request)
	if len(issues) > 0 {
		return GoalWorkspaceV0{}, issues
	}
	base, issue := connector.resolveGoalWorkspaceBaseV0(ctx, request)
	if issue != nil {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{*issue}
	}
	request.BaseRevision = base
	want := goalWorkspaceFromRequestV0(request)
	if existing, found, loadIssues := loadGoalWorkspaceManifestV0(goalWorkspaceManifestPathV0(request)); found || len(loadIssues) > 0 {
		if len(loadIssues) > 0 {
			return GoalWorkspaceV0{}, loadIssues
		}
		return connector.validateExistingGoalWorkspaceV0(ctx, want, existing)
	}
	if err := os.MkdirAll(filepath.Dir(goalWorkspaceManifestPathV0(request)), 0o700); err != nil {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("workspace_root")}
	}
	lock, err := os.OpenFile(goalWorkspaceLockPathV0(request), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("workspace_lock")}
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lock.Close()
		return GoalWorkspaceV0{}, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceLockedV0, "goal_ref")}
	}
	defer func() {
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
	}()

	if existing, found, loadIssues := loadGoalWorkspaceManifestV0(goalWorkspaceManifestPathV0(request)); found || len(loadIssues) > 0 {
		if len(loadIssues) > 0 {
			return GoalWorkspaceV0{}, loadIssues
		}
		return connector.validateExistingGoalWorkspaceV0(ctx, want, existing)
	}
	if _, statErr := os.Stat(want.ProjectWorkDir); statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "project_work_dir")}
	}
	if err := os.MkdirAll(filepath.Dir(want.ProjectWorkDir), 0o700); err != nil {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("workspace_root")}
	}
	if _, gitIssue := connector.VCS.gitOutputV0(ctx, request.SourceWorkDir, "worktree", "add", "--detach", want.ProjectWorkDir, want.BaseRevision); gitIssue != nil {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{goalWorkspaceGitIssueV0("git.worktree_add", *gitIssue)}
	}
	if writeIssues := writeGoalWorkspaceManifestV0(goalWorkspaceManifestPathV0(request), want); len(writeIssues) > 0 {
		_, _ = connector.VCS.gitOutputV0(ctx, request.SourceWorkDir, "worktree", "remove", "--force", want.ProjectWorkDir)
		return GoalWorkspaceV0{}, writeIssues
	}
	return want, nil
}

func (connector GitGoalWorkspaceProvisionerV0) ResolveGoalWorkspaceV0(
	ctx context.Context,
	request GoalWorkspaceRequestV0,
) (GoalWorkspaceV0, []WorktreeIssueV0) {
	request, issues := normalizeGoalWorkspaceRequestV0(request)
	if len(issues) > 0 {
		return GoalWorkspaceV0{}, issues
	}
	existing, found, loadIssues := loadGoalWorkspaceManifestV0(goalWorkspaceManifestPathV0(request))
	if len(loadIssues) > 0 {
		return GoalWorkspaceV0{}, loadIssues
	}
	if !found {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueInvalidRequestV0, "goal_workspace_manifest")}
	}
	return connector.validateExistingGoalWorkspaceV0(ctx, existing, existing)
}

func (connector GitGoalWorkspaceProvisionerV0) resolveGoalWorkspaceBaseV0(
	ctx context.Context,
	request GoalWorkspaceRequestV0,
) (string, *WorktreeIssueV0) {
	ref := request.BaseRevision
	if ref == "" {
		ref = "HEAD"
	}
	base, issue := connector.VCS.gitOutputV0(ctx, request.SourceWorkDir, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if issue != nil {
		result := goalWorkspaceGitIssueV0("git.rev_parse", *issue)
		return "", &result
	}
	return strings.TrimSpace(base), nil
}

func (connector GitGoalWorkspaceProvisionerV0) validateExistingGoalWorkspaceV0(
	ctx context.Context,
	want GoalWorkspaceV0,
	existing GoalWorkspaceV0,
) (GoalWorkspaceV0, []WorktreeIssueV0) {
	if existing.SchemaVersion != GoalWorkspaceSchemaVersionV0 ||
		existing.RunRef != want.RunRef || existing.GoalRef != want.GoalRef ||
		existing.ProjectRef != want.ProjectRef || existing.WorktreeRef != want.WorktreeRef ||
		existing.WorkspaceID != want.WorkspaceID || existing.ProjectWorkDir != want.ProjectWorkDir ||
		(want.BaseRevision != "" && existing.BaseRevision != want.BaseRevision) {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "goal_workspace_manifest")}
	}
	head, issue := connector.VCS.gitOutputV0(ctx, existing.ProjectWorkDir, "rev-parse", "--verify", "HEAD^{commit}")
	if issue != nil || strings.TrimSpace(head) == "" {
		return GoalWorkspaceV0{}, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "project_work_dir")}
	}
	return existing, nil
}

func normalizeGoalWorkspaceRequestV0(request GoalWorkspaceRequestV0) (GoalWorkspaceRequestV0, []WorktreeIssueV0) {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.SourceWorkDir = strings.TrimSpace(request.SourceWorkDir)
	request.WorkspaceRoot = strings.TrimSpace(request.WorkspaceRoot)
	request.BaseRevision = strings.TrimSpace(request.BaseRevision)
	var issues []WorktreeIssueV0
	for _, item := range []struct{ field, value string }{
		{"run_ref", request.RunRef}, {"goal_ref", request.GoalRef},
		{"project_ref", request.ProjectRef}, {"worktree_ref", request.WorktreeRef},
	} {
		issues = append(issues, validateWorktreeOpaqueRefV0(item.field, item.value, true)...)
	}
	source, sourceErr := filepath.Abs(request.SourceWorkDir)
	root, rootErr := filepath.Abs(request.WorkspaceRoot)
	if request.SourceWorkDir == "" || sourceErr != nil {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "source_work_dir"))
	} else {
		request.SourceWorkDir = filepath.Clean(source)
	}
	if request.WorkspaceRoot == "" || rootErr != nil {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "workspace_root"))
	} else {
		request.WorkspaceRoot = filepath.Clean(root)
	}
	if sourceErr == nil && rootErr == nil && pathInsideOrEqualV0(request.SourceWorkDir, request.WorkspaceRoot) {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "workspace_root"))
	}
	if strings.ContainsAny(request.BaseRevision, "\x00\r\n") {
		issues = append(issues, worktreeIssueV0(WorktreeIssueInvalidRequestV0, "base_revision"))
	}
	return request, issues
}

func goalWorkspaceFromRequestV0(request GoalWorkspaceRequestV0) GoalWorkspaceV0 {
	id := goalWorkspaceIDV0(request.RunRef, request.GoalRef)
	return GoalWorkspaceV0{
		SchemaVersion:  GoalWorkspaceSchemaVersionV0,
		RunRef:         request.RunRef,
		GoalRef:        request.GoalRef,
		ProjectRef:     request.ProjectRef,
		WorktreeRef:    request.WorktreeRef,
		WorkspaceID:    id,
		ProjectWorkDir: filepath.Join(request.WorkspaceRoot, "workspaces", id),
		BaseRevision:   request.BaseRevision,
		EvidenceRefs:   []string{"evidence-ref-goal-workspace-physical-isolation", "evidence-ref-goal-workspace-base:" + request.BaseRevision},
	}
}

func goalWorkspaceIDV0(runRef string, goalRef string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(runRef) + "\x00" + strings.TrimSpace(goalRef)))
	return hex.EncodeToString(digest[:16])
}

func goalWorkspaceManifestPathV0(request GoalWorkspaceRequestV0) string {
	return filepath.Join(request.WorkspaceRoot, "manifests", goalWorkspaceIDV0(request.RunRef, request.GoalRef)+".json")
}

func goalWorkspaceLockPathV0(request GoalWorkspaceRequestV0) string {
	return filepath.Join(request.WorkspaceRoot, "manifests", goalWorkspaceIDV0(request.RunRef, request.GoalRef)+".lock")
}

func loadGoalWorkspaceManifestV0(path string) (GoalWorkspaceV0, bool, []WorktreeIssueV0) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return GoalWorkspaceV0{}, false, nil
	}
	if err != nil {
		return GoalWorkspaceV0{}, false, []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("goal_workspace_manifest")}
	}
	var workspace GoalWorkspaceV0
	if json.Unmarshal(raw, &workspace) != nil {
		return GoalWorkspaceV0{}, false, []WorktreeIssueV0{worktreeIssueV0(WorktreeIssueWorkspaceConflictV0, "goal_workspace_manifest")}
	}
	return workspace, true, nil
}

func writeGoalWorkspaceManifestV0(path string, workspace GoalWorkspaceV0) []WorktreeIssueV0 {
	raw, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("goal_workspace_manifest")}
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".goal-workspace-*.tmp")
	if err != nil {
		return []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("goal_workspace_manifest")}
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if chmodErr := tmp.Chmod(0o600); chmodErr != nil {
		_ = tmp.Close()
		return []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("goal_workspace_manifest")}
	}
	if _, err = tmp.Write(raw); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err != nil {
		return []WorktreeIssueV0{goalWorkspaceFilesystemIssueV0("goal_workspace_manifest")}
	}
	return nil
}

func pathInsideOrEqualV0(parent string, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}

func goalWorkspaceFilesystemIssueV0(field string) WorktreeIssueV0 {
	return worktreeIssueV0(WorktreeIssueFilesystemV0, field)
}

func goalWorkspaceGitIssueV0(field string, issue AppVCSIssueV0) WorktreeIssueV0 {
	return worktreeIssueV0(WorktreeIssueFilesystemV0, field, issue.Evidence...)
}
