package orquestaruntimeworktree

const AppVCSResultSchemaVersionV0 = "app_vcs_result.v0"

type AppVCSActionV0 string

const (
	AppVCSActionPrepareRepoV0 AppVCSActionV0 = "prepare_repo"
	AppVCSActionReviewRepoV0  AppVCSActionV0 = "review_repo"
	AppVCSActionCommitV0      AppVCSActionV0 = "commit"
	AppVCSActionPushV0        AppVCSActionV0 = "push"
)

type AppVCSStatusV0 string

const (
	AppVCSStatusCompletedV0   AppVCSStatusV0 = "completed"
	AppVCSStatusCleanV0       AppVCSStatusV0 = "clean"
	AppVCSStatusPendingPushV0 AppVCSStatusV0 = "pending_push"
	AppVCSStatusFailedV0      AppVCSStatusV0 = "failed"
)

type AppVCSIssueCodeV0 string

const (
	AppVCSIssueInvalidRequestV0        AppVCSIssueCodeV0 = "app_vcs_request_invalida"
	AppVCSIssueGitErrorV0              AppVCSIssueCodeV0 = "app_vcs_git_error"
	AppVCSIssuePushPendingV0           AppVCSIssueCodeV0 = "app_vcs_push_pending"
	AppVCSIssueControlPathV0           AppVCSIssueCodeV0 = "app_vcs_control_path"
	AppVCSIssueGitOutputTooLargeV0     AppVCSIssueCodeV0 = "git_output_too_large"
	AppVCSIssueGitStatusTooManyPathsV0 AppVCSIssueCodeV0 = "git_status_too_many_paths"
	AppVCSIssueGitCommandTimeoutV0     AppVCSIssueCodeV0 = "git_command_timeout"
)

type AppVCSRequestV0 struct {
	RequestID      string         `json:"request_id,omitempty"`
	CorrelationID  string         `json:"correlation_id,omitempty"`
	Action         AppVCSActionV0 `json:"action"`
	AppRef         string         `json:"app_ref"`
	RepoRef        string         `json:"repo_ref"`
	WorktreeRef    string         `json:"worktree_ref,omitempty"`
	BranchRef      string         `json:"branch_ref,omitempty"`
	ProjectWorkDir string         `json:"project_work_dir,omitempty"`
	CommitMessage  string         `json:"commit_message,omitempty"`
	CommitPaths    []string       `json:"commit_paths,omitempty"`
	AllowPush      bool           `json:"allow_push,omitempty"`
	RemoteName     string         `json:"remote_name,omitempty"`
	RemoteBranch   string         `json:"remote_branch,omitempty"`
}

type AppVCSResultV0 struct {
	SchemaVersion     string                                    `json:"schema_version"`
	Status            AppVCSStatusV0                            `json:"status"`
	Action            AppVCSActionV0                            `json:"action"`
	AppRef            string                                    `json:"app_ref,omitempty"`
	RepoRef           string                                    `json:"repo_ref,omitempty"`
	WorktreeRef       string                                    `json:"worktree_ref,omitempty"`
	BranchRef         string                                    `json:"branch_ref,omitempty"`
	CommitRef         string                                    `json:"commit_ref,omitempty"`
	CommitShortRef    string                                    `json:"commit_short_ref,omitempty"`
	ChangedPaths      []string                                  `json:"changed_paths,omitempty"`
	PushPending       bool                                      `json:"push_pending,omitempty"`
	Retryable         bool                                      `json:"retryable,omitempty"`
	EvidenceRefs      []string                                  `json:"evidence_refs,omitempty"`
	ExclusionReceipts []WorktreeLocalArtifactExclusionReceiptV0 `json:"exclusion_receipts,omitempty"`
	Issues            []AppVCSIssueV0                           `json:"issues,omitempty"`
}

type AppVCSIssueV0 struct {
	Code       AppVCSIssueCodeV0 `json:"code"`
	MessageKey string            `json:"message_key"`
	Field      string            `json:"field,omitempty"`
	Evidence   []string          `json:"evidence,omitempty"`
}
