package orquestaruntimeworktree

import "fmt"

const WorktreeSnapshotSchemaVersionV0 = "worktree_snapshot.v0"
const WorktreeDefaultGoFileLineBudgetV0 = 300

const WorktreeDefaultSnapshotMaxFilesV0 = 20000
const WorktreeDefaultSnapshotMaxFileBytesV0 int64 = 8 * 1024 * 1024
const WorktreeDefaultSnapshotMaxTotalBytesV0 int64 = 128 * 1024 * 1024

type WorktreeIssueCodeV0 string

const (
	WorktreeIssueInvalidRequestV0        WorktreeIssueCodeV0 = "worktree_request_invalida"
	WorktreeIssueFilesystemV0            WorktreeIssueCodeV0 = "worktree_filesystem_error"
	WorktreeIssueOutsideWriteSetV0       WorktreeIssueCodeV0 = "outside_write_set"
	WorktreeIssueRemovedPathV0           WorktreeIssueCodeV0 = "removed_path"
	WorktreeIssueTruncatedPathV0         WorktreeIssueCodeV0 = "truncated_path"
	WorktreeIssueRenamedOrMovedV0        WorktreeIssueCodeV0 = "renamed_or_moved_path"
	WorktreeIssueReplacedLargeV0         WorktreeIssueCodeV0 = "replaced_large_delta"
	WorktreeIssueAckFilesMismatchV0      WorktreeIssueCodeV0 = "ack_files_mismatch"
	WorktreeIssueGoLineBudgetV0          WorktreeIssueCodeV0 = "go_file_line_budget_exceeded"
	WorktreeIssueControlPathV0           WorktreeIssueCodeV0 = "control_path"
	WorktreeIssueSnapshotFileTooLargeV0  WorktreeIssueCodeV0 = "worktree_snapshot_file_too_large"
	WorktreeIssueSnapshotTooManyFilesV0  WorktreeIssueCodeV0 = "worktree_snapshot_too_many_files"
	WorktreeIssueSnapshotTooLargeV0      WorktreeIssueCodeV0 = "worktree_snapshot_too_large"
	WorktreeIssueSnapshotUnreadableV0    WorktreeIssueCodeV0 = "worktree_snapshot_unreadable"
	WorktreeIssueGitOutputTooLargeV0     WorktreeIssueCodeV0 = "git_output_too_large"
	WorktreeIssueGitStatusTooManyPathsV0 WorktreeIssueCodeV0 = "git_status_too_many_paths"
	WorktreeIssueGitCommandTimeoutV0     WorktreeIssueCodeV0 = "git_command_timeout"
	WorktreeIssueWorkspaceConflictV0     WorktreeIssueCodeV0 = "goal_workspace_conflict"
	WorktreeIssueWorkspaceLockedV0       WorktreeIssueCodeV0 = "goal_workspace_locked"
)

type WorktreeIssueV0 struct {
	Code       WorktreeIssueCodeV0 `json:"code"`
	MessageKey string              `json:"message_key"`
	Field      string              `json:"field,omitempty"`
	Evidence   []string            `json:"evidence,omitempty"`
}

func (issue WorktreeIssueV0) Error() string {
	if issue.Field == "" {
		return string(issue.Code)
	}
	return fmt.Sprintf("%s: %s", issue.Code, issue.Field)
}

type WorktreeSnapshotFileV0 struct {
	Path      string `json:"path"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
	LineCount int    `json:"line_count,omitempty"`
}

type WorktreeSnapshotReadBudgetV0 struct {
	MaxFiles      int   `json:"max_files,omitempty"`
	MaxFileBytes  int64 `json:"max_file_bytes,omitempty"`
	MaxTotalBytes int64 `json:"max_total_bytes,omitempty"`
}

type WorktreeSnapshotV0 struct {
	SchemaVersion     string                                    `json:"schema_version"`
	SnapshotRef       string                                    `json:"snapshot_ref"`
	ReadBudget        WorktreeSnapshotReadBudgetV0              `json:"read_budget,omitempty"`
	Files             []WorktreeSnapshotFileV0                  `json:"files"`
	OmittedPaths      []string                                  `json:"omitted_paths,omitempty"`
	ExclusionReceipts []WorktreeLocalArtifactExclusionReceiptV0 `json:"exclusion_receipts,omitempty"`
}

type WorktreeSnapshotRequestV0 struct {
	SnapshotRef    string   `json:"snapshot_ref"`
	ProjectWorkDir string   `json:"project_work_dir"`
	IgnorePrefixes []string `json:"ignore_prefixes,omitempty"`
	MaxFiles       int      `json:"max_files,omitempty"`
	MaxFileBytes   int64    `json:"max_file_bytes,omitempty"`
	MaxTotalBytes  int64    `json:"max_total_bytes,omitempty"`
	AllowPartial   bool     `json:"allow_partial,omitempty"`
}

type WorktreeVerifyRequestV0 struct {
	Baseline                   WorktreeSnapshotV0                   `json:"baseline"`
	ProjectWorkDir             string                               `json:"project_work_dir"`
	WriteSet                   []string                             `json:"write_set"`
	AckFiles                   []string                             `json:"ack_files,omitempty"`
	DestructiveAuthorizations  []WorktreeDestructiveAuthorizationV0 `json:"destructive_authorizations,omitempty"`
	IgnorePrefixes             []string                             `json:"ignore_prefixes,omitempty"`
	MaxSnapshotFiles           int                                  `json:"max_snapshot_files,omitempty"`
	MaxSnapshotFileBytes       int64                                `json:"max_snapshot_file_bytes,omitempty"`
	MaxSnapshotTotalBytes      int64                                `json:"max_snapshot_total_bytes,omitempty"`
	AllowPartialSnapshot       bool                                 `json:"allow_partial_snapshot,omitempty"`
	StrictGoLineBudget         bool                                 `json:"strict_go_line_budget,omitempty"`
	MaxGoFileLines             int                                  `json:"max_go_file_lines,omitempty"`
	AcceptedPartitionFollowups []string                             `json:"accepted_partition_followups,omitempty"`
}

type WorktreeDestructiveAuthorizationKindV0 string

const (
	WorktreeDestructiveAuthorizationRenameV0   WorktreeDestructiveAuthorizationKindV0 = "renamed_or_moved"
	WorktreeDestructiveAuthorizationRemoveV0   WorktreeDestructiveAuthorizationKindV0 = "removed"
	WorktreeDestructiveAuthorizationTruncateV0 WorktreeDestructiveAuthorizationKindV0 = "truncated"
	WorktreeDestructiveAuthorizationReplaceV0  WorktreeDestructiveAuthorizationKindV0 = "replaced_large_delta"
)

type WorktreeDestructiveAuthorizationV0 struct {
	Kind         WorktreeDestructiveAuthorizationKindV0 `json:"kind"`
	Path         string                                 `json:"path,omitempty"`
	PreviousPath string                                 `json:"previous_path,omitempty"`
	CurrentPath  string                                 `json:"current_path,omitempty"`
}

type WorktreeVerifyResultV0 struct {
	OK                     bool                                      `json:"ok"`
	ChangedPaths           []string                                  `json:"changed_paths,omitempty"`
	AddedPaths             []string                                  `json:"added_paths,omitempty"`
	RemovedPaths           []string                                  `json:"removed_paths,omitempty"`
	TruncatedPaths         []string                                  `json:"truncated_paths,omitempty"`
	RenamedOrMovedPaths    []string                                  `json:"renamed_or_moved_paths,omitempty"`
	ReplacedLargeDelta     []string                                  `json:"replaced_large_delta,omitempty"`
	OutsideWriteSet        []string                                  `json:"outside_write_set,omitempty"`
	AckFilesNotChanged     []string                                  `json:"ack_files_not_changed,omitempty"`
	UnreportedChangedPaths []string                                  `json:"unreported_changed_paths,omitempty"`
	DestructiveChanges     []WorktreeDestructiveChangeV0             `json:"destructive_changes,omitempty"`
	GoLineBudgetViolations []WorktreeGoLineBudgetViolationV0         `json:"go_line_budget_violations,omitempty"`
	EvidenceRefs           []string                                  `json:"evidence_refs,omitempty"`
	ExclusionReceipts      []WorktreeLocalArtifactExclusionReceiptV0 `json:"exclusion_receipts,omitempty"`
}

type WorktreeLocalArtifactExclusionReceiptV0 struct {
	Category   string `json:"category"`
	ReasonCode string `json:"reason_code"`
	Count      int    `json:"count"`
}

type WorktreeDestructiveChangeV0 struct {
	Kind          string `json:"kind"`
	Path          string `json:"path,omitempty"`
	PreviousPath  string `json:"previous_path,omitempty"`
	CurrentPath   string `json:"current_path,omitempty"`
	BaselineSize  int64  `json:"baseline_size,omitempty"`
	CurrentSize   int64  `json:"current_size,omitempty"`
	BaselineLines int    `json:"baseline_lines,omitempty"`
	CurrentLines  int    `json:"current_lines,omitempty"`
}

type WorktreeGoLineBudgetViolationV0 struct {
	Path          string `json:"path"`
	BaselineLines int    `json:"baseline_lines,omitempty"`
	CurrentLines  int    `json:"current_lines"`
	MaxLines      int    `json:"max_lines"`
	Reason        string `json:"reason"`
}

func worktreeIssueV0(
	code WorktreeIssueCodeV0,
	field string,
	evidence ...string,
) WorktreeIssueV0 {
	return WorktreeIssueV0{
		Code:       code,
		MessageKey: "orquesta.runtime.worktree." + string(code),
		Field:      field,
		Evidence:   compactWorktreeStringsV0(evidence),
	}
}
