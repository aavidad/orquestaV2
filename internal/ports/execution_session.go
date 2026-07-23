package ports

import (
	"context"
	"errors"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// ExecutionSessionRef is an opaque, execution-scoped authentication session.
// It is authority metadata, never credential material or a transport endpoint.
type ExecutionSessionRef string

func NewExecutionSessionRef(value string) (ExecutionSessionRef, error) {
	if strings.TrimSpace(value) != value || !strings.HasPrefix(value, "execution-session:") ||
		len(value) == len("execution-session:") || strings.ContainsAny(value, "\x00\r\n/\\") {
		return "", errors.New("execution_session.ref_invalid")
	}
	return ExecutionSessionRef(value), nil
}

func (ref ExecutionSessionRef) String() string { return string(ref) }

// ExecutionSessionEnsureRequest is the complete immutable execution identity
// from which an adapter may materialize an ephemeral child credential.
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

// ExecutionSessionAuthority is the material-free authority projection. Every
// field is either persisted by the Goal aggregate or deterministically derived
// from that persisted tuple.
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

// ExecutionSessionBroker owns credential materialization outside durable Goal
// state. Ensure is causally idempotent for an equal request. Revocation belongs
// to durable execution lifecycle: credential material alone grants no authority.
type ExecutionSessionBroker interface {
	Ensure(context.Context, ExecutionSessionEnsureRequest) (ExecutionSessionReceipt, error)
}
