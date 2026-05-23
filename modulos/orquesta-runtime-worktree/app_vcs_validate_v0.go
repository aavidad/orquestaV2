package orquestaruntimeworktree

import "strings"

func normalizeAppVCSRequestV0(request AppVCSRequestV0) AppVCSRequestV0 {
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.Action = AppVCSActionV0(strings.TrimSpace(string(request.Action)))
	request.AppRef = strings.TrimSpace(request.AppRef)
	request.RepoRef = strings.TrimSpace(request.RepoRef)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.BranchRef = strings.TrimSpace(request.BranchRef)
	request.ProjectWorkDir = strings.TrimSpace(request.ProjectWorkDir)
	request.CommitMessage = strings.TrimSpace(request.CommitMessage)
	request.RemoteName = strings.TrimSpace(request.RemoteName)
	request.RemoteBranch = strings.TrimSpace(request.RemoteBranch)
	request.CommitPaths, _ = normalizeWorktreePathListV0(request.CommitPaths, false)
	return request
}

func validateAppVCSRequestV0(request AppVCSRequestV0) []AppVCSIssueV0 {
	var issues []AppVCSIssueV0
	switch request.Action {
	case AppVCSActionPrepareRepoV0, AppVCSActionCommitV0, AppVCSActionPushV0:
	default:
		issues = append(issues, appVCSIssueV0(AppVCSIssueInvalidRequestV0, "action"))
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"app_ref", request.AppRef},
		{"repo_ref", request.RepoRef},
	} {
		issues = append(issues, appVCSOpaqueRefIssuesV0(item.field, item.value, true)...)
	}
	issues = append(issues, appVCSOpaqueRefIssuesV0("worktree_ref", request.WorktreeRef, false)...)
	issues = append(issues, appVCSOpaqueRefIssuesV0("branch_ref", request.BranchRef, false)...)
	if issue := validateWorktreeRootV0(request.ProjectWorkDir); issue != nil {
		issues = append(issues, appVCSIssueV0(AppVCSIssueInvalidRequestV0, "project_work_dir"))
	}
	if request.Action == AppVCSActionCommitV0 && request.CommitMessage == "" {
		issues = append(issues, appVCSIssueV0(AppVCSIssueInvalidRequestV0, "commit_message"))
	}
	if request.Action == AppVCSActionPushV0 && !request.AllowPush {
		issues = append(issues, appVCSIssueV0(AppVCSIssueInvalidRequestV0, "allow_push"))
	}
	return issues
}

func appVCSOpaqueRefIssuesV0(field string, value string, required bool) []AppVCSIssueV0 {
	worktreeIssues := validateWorktreeOpaqueRefV0(field, value, required)
	out := make([]AppVCSIssueV0, 0, len(worktreeIssues))
	for _, issue := range worktreeIssues {
		out = append(out, appVCSIssueV0(AppVCSIssueInvalidRequestV0, issue.Field, issue.Evidence...))
	}
	return out
}

func appVCSIssueV0(code AppVCSIssueCodeV0, field string, evidence ...string) AppVCSIssueV0 {
	return AppVCSIssueV0{
		Code:       code,
		MessageKey: "orquesta.runtime.worktree." + string(code),
		Field:      strings.TrimSpace(field),
		Evidence:   compactWorktreeStringsV0(evidence),
	}
}
