package ports

import (
	"context"
	"errors"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// ExecutionSessionRef is opaque authority metadata, never credential material or an endpoint.
type ExecutionSessionRef string

func NewExecutionSessionRef(value string) (ExecutionSessionRef, error) {
	if strings.TrimSpace(value) != value || !strings.HasPrefix(value, "execution-session:") ||
		len(value) == len("execution-session:") || strings.ContainsAny(value, "\x00\r\n/\\") {
		return "", errors.New("execution_session.ref_invalid")
	}
	return ExecutionSessionRef(value), nil
}

func (ref ExecutionSessionRef) String() string { return string(ref) }

// ExecutionArtifactAccessRef identifies the execution-scoped artifact access
// authority. It is opaque metadata, never a credential or filesystem path.
type ExecutionArtifactAccessRef string

func NewExecutionArtifactAccessRef(value string) (ExecutionArtifactAccessRef, error) {
	if !validExecutionSessionAuthorityRef(value, "artifact-access:execution:sha256:") {
		return "", errors.New("execution_session.artifact_access_ref_invalid")
	}
	return ExecutionArtifactAccessRef(value), nil
}

func (ref ExecutionArtifactAccessRef) String() string { return string(ref) }

// ExecutionMCPAccessRef identifies the execution-scoped MCP access authority.
type ExecutionMCPAccessRef string

func NewExecutionMCPAccessRef(value string) (ExecutionMCPAccessRef, error) {
	if !validExecutionSessionAuthorityRef(value, "mcp-access:execution:sha256:") {
		return "", errors.New("execution_session.mcp_access_ref_invalid")
	}
	return ExecutionMCPAccessRef(value), nil
}

func (ref ExecutionMCPAccessRef) String() string { return string(ref) }

// ExecutionMailboxEndpointRef identifies the execution-scoped mailbox endpoint.
type ExecutionMailboxEndpointRef string

func NewExecutionMailboxEndpointRef(value string) (ExecutionMailboxEndpointRef, error) {
	if !validExecutionSessionAuthorityRef(value, "mailbox-endpoint:execution:sha256:") {
		return "", errors.New("execution_session.mailbox_endpoint_ref_invalid")
	}
	return ExecutionMailboxEndpointRef(value), nil
}

func (ref ExecutionMailboxEndpointRef) String() string { return string(ref) }

func validExecutionSessionAuthorityRef(value, prefix string) bool {
	if strings.TrimSpace(value) != value || !strings.HasPrefix(value, prefix) ||
		len(value) == len(prefix) || strings.ContainsAny(value, "\x00\r\n/\\") {
		return false
	}
	suffix := strings.TrimPrefix(value, prefix)
	if len(suffix) != 64 {
		return false
	}
	for _, character := range suffix {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return false
		}
	}
	return true
}

// ExecutionSessionEnsureRequest is the immutable identity used to materialize an ephemeral credential.
type ExecutionSessionEnsureRequest struct {
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	WorkItemRef          goal.WorkItemRef
	ExecutionRef         goal.ExecutionRef
	ExecutionAttempt     uint64
	ReplacesExecutionRef goal.ExecutionRef
	PlanGeneration       goal.PlanGeneration
	AppSpecGeneration    goal.AppSpecGeneration
	SpecHash             string
}

// ExecutionSessionAuthority is persisted or deterministically derived, never secret material.
type ExecutionSessionAuthority struct {
	SessionRef         ExecutionSessionRef
	ArtifactAccessRef  ExecutionArtifactAccessRef
	MCPAccessRef       ExecutionMCPAccessRef
	MailboxEndpointRef ExecutionMailboxEndpointRef
	ServicePrincipal   identity.Principal
	Request            ExecutionSessionEnsureRequest
}

type ExecutionSessionReceipt struct {
	Authority ExecutionSessionAuthority
	EnsuredAt time.Time
	Replayed  bool
}

// ExecutionSessionBroker materializes causally idempotent credentials outside Goal state.
type ExecutionSessionBroker interface {
	Ensure(context.Context, ExecutionSessionEnsureRequest) (ExecutionSessionReceipt, error)
	Revoke(context.Context, ExecutionSessionEnsureRequest) error
}
