package orquestaruntimeworktree

import "strings"

var worktreeDefaultControlPrefixesV0 = []string{
	".orquesta-runtime",
	".orquesta-codex-runtime",
	".orquesta-local",
	".orquesta-local-runtime",
	".orquesta-control",
	".orquesta-server",
	".orquesta-guardian",
	".orquesta-logs",
	".orquesta-local-state",
	".orquesta-state",
	".orquesta-purged",
	".orquesta-smoke-work",
	".orquesta-runs",
	".orquesta-worktrees",
	".git",
	".agents",
	".codex",
	".codex-docker-home",
	".codex-sandbox-workspace",
	"orquesta-runtime",
	"orquesta-codex-runtime",
	"orquesta-local-runtime",
	"logs",
	"tmp",
	".cache",
	"backups",
	"certs",
}

var worktreeDefaultControlFileNamesV0 = map[string]bool{
	"agent_ack.json":                     true,
	"agent_packet.json":                  true,
	"agent_prompt.txt":                   true,
	"agent_shutdown_checkpoint_ack.json": true,
	"codex_last_message.txt":             true,
	"codex_stderr.log":                   true,
	"codex_stdout.log":                   true,
	"director_decisions.json":            true,
	"orquesta_shutdown_request.json":     true,
}

func DefaultWorktreeControlIgnorePrefixesV0() []string {
	return append([]string(nil), worktreeDefaultControlPrefixesV0...)
}

func IsWorktreeControlPathV0(value string) bool {
	return worktreeControlPathV0(value)
}

func normalizeWorktreeIgnorePrefixesV0(values []string) []string {
	paths, _ := normalizeWorktreePathListV0(values, false)
	return compactWorktreeStringsV0(append(paths, worktreeDefaultControlPrefixesV0...))
}

func worktreeControlPathV0(value string) bool {
	_, ok := worktreeLocalArtifactPolicyMatchV0(value)
	return ok
}

func splitWorktreeProductAndControlPathsV0(values []string) ([]string, []string) {
	product := make([]string, 0, len(values))
	control := make([]string, 0)
	for _, value := range values {
		pathValue, ok := normalizeWorktreeRelPathV0(value, false)
		if !ok {
			continue
		}
		if worktreeControlPathV0(pathValue) {
			control = append(control, pathValue)
			continue
		}
		product = append(product, pathValue)
	}
	return compactWorktreeStringsV0(product), compactWorktreeStringsV0(control)
}

func worktreeControlPathIssuesV0(values []string) []WorktreeIssueV0 {
	_, control := splitWorktreeProductAndControlPathsV0(values)
	issues := make([]WorktreeIssueV0, 0, len(control))
	for _, item := range control {
		issues = append(issues, worktreeIssueV0(WorktreeIssueControlPathV0, item, "control_path_excluded"))
	}
	return issues
}

func worktreeControlPathIssueEvidenceV0(issues []WorktreeIssueV0) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Code == WorktreeIssueControlPathV0 && strings.TrimSpace(issue.Field) != "" {
			out = append(out, issue.Field)
		}
	}
	return compactWorktreeStringsV0(out)
}
