package orquestamcp

import (
	"context"
	"strings"
)

const (
	MCPAppVCSToolNameV0              = "orquesta.app_vcs.v0"
	MCPAppVCSToolVersionV0           = "v0"
	MCPAppVCSResourceURIV0           = "orquesta://contracts/app-vcs/v0"
	MCPAppVCSHTTPPathV0              = "/api/v0/apps/vcs"
	MCPAppVCSActionPrepareRepoV0     = "prepare_repo"
	MCPAppVCSActionCommitV0          = "commit"
	MCPAppVCSActionPushV0            = "push"
	MCPAppVCSEstadoOKV0              = "ok"
	MCPAppVCSEstadoErrorV0           = "error"
	MCPAppVCSExecutorUnavailableV0   = "app_vcs_executor_no_disponible"
	MCPAppVCSActionUnsupportedV0     = "app_vcs_action_no_soportada"
	MCPAppVCSCommitMessageRequiredV0 = "app_vcs_commit_message_requerido"
	MCPAppVCSPushOptInRequiredV0     = "app_vcs_push_opt_in_requerido"
	MCPAppVCSRequiredRefMissingV0    = "app_vcs_ref_requerida"
)

type MCPAppVCSToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAppVCSToolInputV0 struct {
	RequestID     string   `json:"request_id,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
	Action        string   `json:"action"`
	AppRef        string   `json:"app_ref"`
	RepoRef       string   `json:"repo_ref"`
	WorktreeRef   string   `json:"worktree_ref,omitempty"`
	BranchRef     string   `json:"branch_ref,omitempty"`
	CommitMessage string   `json:"commit_message,omitempty"`
	CommitPaths   []string `json:"commit_paths,omitempty"`
	AllowPush     bool     `json:"allow_push,omitempty"`
}

type MCPAppVCSToolResultV0 struct {
	Estado         string                 `json:"estado"`
	RequestID      string                 `json:"request_id,omitempty"`
	CorrelationID  string                 `json:"correlation_id,omitempty"`
	Action         string                 `json:"action,omitempty"`
	Status         string                 `json:"status,omitempty"`
	AppRef         string                 `json:"app_ref,omitempty"`
	RepoRef        string                 `json:"repo_ref,omitempty"`
	WorktreeRef    string                 `json:"worktree_ref,omitempty"`
	BranchRef      string                 `json:"branch_ref,omitempty"`
	CommitRef      string                 `json:"commit_ref,omitempty"`
	CommitShortRef string                 `json:"commit_short_ref,omitempty"`
	ChangedPaths   []string               `json:"changed_paths,omitempty"`
	PushPending    bool                   `json:"push_pending,omitempty"`
	Retryable      bool                   `json:"retryable,omitempty"`
	EvidenceRefs   []string               `json:"evidence_refs,omitempty"`
	Errores        []MCPValidationIssueV0 `json:"errores_publicos,omitempty"`
}

type MCPAppVCSExecutorPortV0 interface {
	Execute(context.Context, MCPAppVCSToolInputV0) (MCPAppVCSToolResultV0, error)
}

func MCPAppVCSDescriptorV0() MCPAppVCSToolDescriptorV0 {
	return MCPAppVCSToolDescriptorV0{
		Name:        MCPAppVCSToolNameV0,
		Version:     MCPAppVCSToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,action:prepare_repo|commit|push,app_ref,repo_ref,worktree_ref?,branch_ref?,commit_message?,commit_paths?,allow_push?}",
		Output:      "ok:{status,commit_ref?,changed_paths?,push_pending?,retryable?,evidence_refs?}|error:{errores_publicos}",
		ResourceURI: MCPAppVCSResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"solo transporta refs opacas de app repo worktree y branch",
			"commit local y push opt-in delegan en conector inyectado",
			"sin rutas locales DB HOME proveedor modelo ni runtime en payload publico",
		},
	}
}

func NormalizeMCPAppVCSInputV0(input MCPAppVCSToolInputV0) MCPAppVCSToolInputV0 {
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	input.AppRef = strings.TrimSpace(input.AppRef)
	input.RepoRef = strings.TrimSpace(input.RepoRef)
	input.WorktreeRef = strings.TrimSpace(input.WorktreeRef)
	input.BranchRef = strings.TrimSpace(input.BranchRef)
	input.CommitMessage = strings.TrimSpace(input.CommitMessage)
	input.CommitPaths = compactStringsMCPV0(input.CommitPaths)
	return input
}

func ValidateMCPAppVCSInputV0(input MCPAppVCSToolInputV0) []MCPValidationIssueV0 {
	input = NormalizeMCPAppVCSInputV0(input)
	var issues []MCPValidationIssueV0
	if input.AppRef == "" {
		issues = append(issues, newMCPAppVCSIssueV0(MCPAppVCSRequiredRefMissingV0, "app_ref"))
	}
	if input.RepoRef == "" {
		issues = append(issues, newMCPAppVCSIssueV0(MCPAppVCSRequiredRefMissingV0, "repo_ref"))
	}
	switch input.Action {
	case MCPAppVCSActionPrepareRepoV0:
	case MCPAppVCSActionCommitV0:
		if input.CommitMessage == "" {
			issues = append(issues, newMCPAppVCSIssueV0(MCPAppVCSCommitMessageRequiredV0, "commit_message"))
		}
	case MCPAppVCSActionPushV0:
		if !input.AllowPush {
			issues = append(issues, newMCPAppVCSIssueV0(MCPAppVCSPushOptInRequiredV0, "allow_push"))
		}
	default:
		issues = append(issues, newMCPAppVCSIssueV0(MCPAppVCSActionUnsupportedV0, "action"))
	}
	return issues
}

func NewMCPAppVCSErrorResultV0(
	input MCPAppVCSToolInputV0,
	code string,
	field string,
	message string,
) MCPAppVCSToolResultV0 {
	input = NormalizeMCPAppVCSInputV0(input)
	return MCPAppVCSToolResultV0{
		Estado:        MCPAppVCSEstadoErrorV0,
		RequestID:     input.RequestID,
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:        input.Action,
		AppRef:        input.AppRef,
		RepoRef:       input.RepoRef,
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func newMCPAppVCSIssueV0(code string, field string) MCPValidationIssueV0 {
	return MCPValidationIssueV0{
		Code:    strings.TrimSpace(code),
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(code),
	}
}
