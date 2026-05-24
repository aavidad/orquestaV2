package orquestaruntimeworktree

import "fmt"

const WorktreeSnapshotSchemaVersionV0 = "worktree_snapshot.v0"

type WorktreeIssueCodeV0 string

const (
	WorktreeIssueInvalidRequestV0  WorktreeIssueCodeV0 = "worktree_request_invalida"
	WorktreeIssueFilesystemV0      WorktreeIssueCodeV0 = "worktree_filesystem_error"
	WorktreeIssueOutsideWriteSetV0 WorktreeIssueCodeV0 = "outside_write_set"
	WorktreeIssueRemovedPathV0     WorktreeIssueCodeV0 = "removed_path"
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
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type WorktreeSnapshotV0 struct {
	SchemaVersion string                   `json:"schema_version"`
	SnapshotRef   string                   `json:"snapshot_ref"`
	Files         []WorktreeSnapshotFileV0 `json:"files"`
}

type WorktreeSnapshotRequestV0 struct {
	SnapshotRef    string   `json:"snapshot_ref"`
	ProjectWorkDir string   `json:"project_work_dir"`
	IgnorePrefixes []string `json:"ignore_prefixes,omitempty"`
}

type WorktreeVerifyRequestV0 struct {
	Baseline       WorktreeSnapshotV0 `json:"baseline"`
	ProjectWorkDir string             `json:"project_work_dir"`
	WriteSet       []string           `json:"write_set"`
	IgnorePrefixes []string           `json:"ignore_prefixes,omitempty"`
}

type WorktreeVerifyResultV0 struct {
	OK              bool     `json:"ok"`
	ChangedPaths    []string `json:"changed_paths,omitempty"`
	AddedPaths      []string `json:"added_paths,omitempty"`
	RemovedPaths    []string `json:"removed_paths,omitempty"`
	OutsideWriteSet []string `json:"outside_write_set,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
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
