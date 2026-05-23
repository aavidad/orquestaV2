package orquestaruntimeworktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const appVCSCommandTimeoutV0 = 10 * time.Second

type GitAppVCSConnectorV0 struct {
	CommandTimeout time.Duration
}

func (connector GitAppVCSConnectorV0) ExecuteAppVCSV0(
	ctx context.Context,
	request AppVCSRequestV0,
) (AppVCSResultV0, []AppVCSIssueV0) {
	request = normalizeAppVCSRequestV0(request)
	if issues := validateAppVCSRequestV0(request); len(issues) > 0 {
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, nil), issues
	}
	if ctx == nil {
		ctx = context.Background()
	}
	switch request.Action {
	case AppVCSActionPrepareRepoV0:
		return connector.prepareRepoV0(ctx, request)
	case AppVCSActionCommitV0:
		return connector.commitRepoV0(ctx, request)
	case AppVCSActionPushV0:
		return connector.pushRepoV0(ctx, request)
	default:
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, nil),
			[]AppVCSIssueV0{appVCSIssueV0(AppVCSIssueInvalidRequestV0, "action")}
	}
}

func (connector GitAppVCSConnectorV0) prepareRepoV0(
	ctx context.Context,
	request AppVCSRequestV0,
) (AppVCSResultV0, []AppVCSIssueV0) {
	head, issue := connector.gitOutputV0(ctx, request.ProjectWorkDir, "rev-parse", "HEAD")
	if issue != nil {
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, nil), []AppVCSIssueV0{*issue}
	}
	paths, issues := connector.changedPathsV0(ctx, request.ProjectWorkDir)
	if len(issues) > 0 {
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, nil), issues
	}
	result := newAppVCSResultV0(request, AppVCSStatusCompletedV0, paths)
	result.CommitRef = strings.TrimSpace(head)
	result.CommitShortRef = shortCommitRefV0(result.CommitRef)
	return result, nil
}

func (connector GitAppVCSConnectorV0) commitRepoV0(
	ctx context.Context,
	request AppVCSRequestV0,
) (AppVCSResultV0, []AppVCSIssueV0) {
	paths, issues := connector.changedPathsV0(ctx, request.ProjectWorkDir)
	if len(issues) > 0 {
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, nil), issues
	}
	if len(paths) == 0 {
		return connector.cleanResultV0(ctx, request)
	}
	addArgs := append([]string{"add", "-A", "--"}, request.CommitPaths...)
	if len(request.CommitPaths) == 0 {
		addArgs = []string{"add", "-A"}
	}
	if _, issue := connector.gitOutputV0(ctx, request.ProjectWorkDir, addArgs...); issue != nil {
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, paths), []AppVCSIssueV0{*issue}
	}
	if _, issue := connector.gitOutputV0(ctx, request.ProjectWorkDir, "commit", "-m", request.CommitMessage); issue != nil {
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, paths), []AppVCSIssueV0{*issue}
	}
	result, prepareIssues := connector.prepareRepoV0(ctx, request)
	if len(prepareIssues) > 0 {
		return result, prepareIssues
	}
	result.ChangedPaths = paths
	if request.AllowPush {
		return connector.pushFromResultV0(ctx, request, result)
	}
	return result, nil
}

func (connector GitAppVCSConnectorV0) pushRepoV0(
	ctx context.Context,
	request AppVCSRequestV0,
) (AppVCSResultV0, []AppVCSIssueV0) {
	result, issues := connector.prepareRepoV0(ctx, request)
	if len(issues) > 0 {
		return result, issues
	}
	return connector.pushFromResultV0(ctx, request, result)
}

func (connector GitAppVCSConnectorV0) cleanResultV0(
	ctx context.Context,
	request AppVCSRequestV0,
) (AppVCSResultV0, []AppVCSIssueV0) {
	head, issue := connector.gitOutputV0(ctx, request.ProjectWorkDir, "rev-parse", "HEAD")
	if issue != nil {
		return newAppVCSResultV0(request, AppVCSStatusFailedV0, nil), []AppVCSIssueV0{*issue}
	}
	result := newAppVCSResultV0(request, AppVCSStatusCleanV0, nil)
	result.CommitRef = strings.TrimSpace(head)
	result.CommitShortRef = shortCommitRefV0(result.CommitRef)
	return result, nil
}

func (connector GitAppVCSConnectorV0) pushFromResultV0(
	ctx context.Context,
	request AppVCSRequestV0,
	result AppVCSResultV0,
) (AppVCSResultV0, []AppVCSIssueV0) {
	args := []string{"push"}
	if request.RemoteName != "" && request.RemoteBranch != "" {
		args = append(args, request.RemoteName, "HEAD:"+request.RemoteBranch)
	}
	if _, issue := connector.gitOutputV0(ctx, request.ProjectWorkDir, args...); issue != nil {
		result.Status = AppVCSStatusPendingPushV0
		result.PushPending = true
		result.Retryable = true
		return result, []AppVCSIssueV0{appVCSIssueV0(AppVCSIssuePushPendingV0, "push", issue.Evidence...)}
	}
	return result, nil
}

func (connector GitAppVCSConnectorV0) changedPathsV0(
	ctx context.Context,
	repo string,
) ([]string, []AppVCSIssueV0) {
	raw, issue := connector.gitOutputRawV0(ctx, repo, "status", "--porcelain", "--untracked-files=all")
	if issue != nil {
		return nil, []AppVCSIssueV0{*issue}
	}
	return parseAppVCSStatusPathsV0(raw), nil
}

func (connector GitAppVCSConnectorV0) gitOutputV0(
	ctx context.Context,
	repo string,
	args ...string,
) (string, *AppVCSIssueV0) {
	out, issue := connector.gitOutputRawV0(ctx, repo, args...)
	return strings.TrimSpace(out), issue
}

func (connector GitAppVCSConnectorV0) gitOutputRawV0(
	ctx context.Context,
	repo string,
	args ...string,
) (string, *AppVCSIssueV0) {
	timeout := connector.CommandTimeout
	if timeout <= 0 {
		timeout = appVCSCommandTimeoutV0
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}
	evidence := strings.TrimSpace(string(out))
	if runCtx.Err() != nil {
		evidence = fmt.Sprintf("%s timeout: %s", strings.Join(args, " "), evidence)
	}
	issue := appVCSIssueV0(AppVCSIssueGitErrorV0, gitIssueFieldV0(args), evidence)
	return "", &issue
}

func parseAppVCSStatusPathsV0(raw string) []string {
	set := map[string]struct{}{}
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		path := filepath.ToSlash(strings.TrimSpace(line[3:]))
		if parts := strings.Split(path, " -> "); len(parts) == 2 {
			path = strings.TrimSpace(parts[1])
		}
		if path != "" {
			set[path] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for path := range set {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func gitIssueFieldV0(args []string) string {
	if len(args) == 0 {
		return "git"
	}
	return "git." + strings.TrimSpace(args[0])
}

func shortCommitRefV0(ref string) string {
	if len(ref) <= 12 {
		return ref
	}
	return ref[:12]
}

func newAppVCSResultV0(
	request AppVCSRequestV0,
	status AppVCSStatusV0,
	paths []string,
) AppVCSResultV0 {
	return AppVCSResultV0{
		SchemaVersion: AppVCSResultSchemaVersionV0,
		Status:        status,
		Action:        request.Action,
		AppRef:        request.AppRef,
		RepoRef:       request.RepoRef,
		WorktreeRef:   request.WorktreeRef,
		BranchRef:     request.BranchRef,
		ChangedPaths:  compactWorktreeStringsV0(paths),
		EvidenceRefs:  []string{"evidence-ref-app-vcs-v0"},
	}
}
