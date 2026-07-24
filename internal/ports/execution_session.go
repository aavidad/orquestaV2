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
	SessionRef       ExecutionSessionRef
	ServicePrincipal identity.Principal
	Request          ExecutionSessionEnsureRequest
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
