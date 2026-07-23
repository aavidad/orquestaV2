package application

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

const executionSessionDerivationVersion = "orquesta.execution-session-authority.v1"

// ExecutionSessionAuthoritySource is the read-only authority boundary used by
// authentication adapters. Implementations must return only a currently live,
// exact execution; absence and revoked/superseded executions fail closed.
type ExecutionSessionAuthoritySource interface {
	ExecutionSessionAuthority(context.Context, goal.ExecutionRef, string) (ports.ExecutionSessionAuthority, error)
}

// DeriveExecutionSessionAuthority creates material-free identities from one
// exact immutable execution tuple. No database row or credential alias is
// needed: restart reproduces the same refs from canonical Goal state.
func DeriveExecutionSessionAuthority(
	request ports.ExecutionSessionEnsureRequest,
	authenticationMethod string,
) (ports.ExecutionSessionAuthority, error) {
	if err := ValidateExecutionSessionEnsureRequest(request); err != nil ||
		strings.TrimSpace(authenticationMethod) != authenticationMethod || authenticationMethod == "" ||
		strings.ContainsAny(authenticationMethod, "\x00\r\n") {
		return ports.ExecutionSessionAuthority{}, errors.New("application.execution_session_invalid")
	}
	digest := sha256.New()
	writeExecutionSessionField(digest, executionSessionDerivationVersion)
	writeExecutionSessionField(digest, request.ProjectRef.String())
	writeExecutionSessionField(digest, request.GoalRef.String())
	writeExecutionSessionField(digest, request.WorkItemRef.String())
	writeExecutionSessionField(digest, request.ExecutionRef.String())
	writeExecutionSessionUint(digest, request.ExecutionAttempt)
	writeExecutionSessionField(digest, request.ReplacesExecutionRef.String())
	writeExecutionSessionUint(digest, uint64(request.PlanGeneration))
	writeExecutionSessionUint(digest, uint64(request.AppSpecGeneration))
	writeExecutionSessionField(digest, request.SpecHash)
	writeExecutionSessionField(digest, authenticationMethod)
	suffix := hex.EncodeToString(digest.Sum(nil))

	sessionRef, err := ports.NewExecutionSessionRef("execution-session:sha256:" + suffix)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, errors.New("application.execution_session_invalid")
	}
	principalRef, err := identity.NewPrincipalRef("principal:execution:sha256:" + suffix)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, errors.New("application.execution_session_invalid")
	}
	actorRef, err := goal.NewActorRef("actor:execution:sha256:" + suffix)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, errors.New("application.execution_session_invalid")
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindService, authenticationMethod,
	)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, errors.New("application.execution_session_invalid")
	}
	return ports.ExecutionSessionAuthority{
		SessionRef: sessionRef, ServicePrincipal: principal, Request: request,
	}, nil
}

func ValidateExecutionSessionEnsureRequest(request ports.ExecutionSessionEnsureRequest) error {
	if request.ProjectRef.String() == "" || request.GoalRef.String() == "" ||
		request.WorkItemRef.String() == "" || request.ExecutionRef.String() == "" ||
		request.ExecutionAttempt == 0 || request.PlanGeneration == 0 ||
		request.AppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(request.SpecHash) ||
		(request.ExecutionAttempt == 1 && request.ReplacesExecutionRef.String() != "") ||
		(request.ExecutionAttempt > 1 && request.ReplacesExecutionRef.String() == "") {
		return errors.New("application.execution_session_invalid")
	}
	return nil
}

func ExecutionSessionRequest(
	aggregate goal.Goal,
	execution ExecutionRecord,
) ports.ExecutionSessionEnsureRequest {
	return ports.ExecutionSessionEnsureRequest{
		ProjectRef: aggregate.Project(), GoalRef: aggregate.Ref(), WorkItemRef: execution.WorkItemRef,
		ExecutionRef: execution.Ref, ExecutionAttempt: execution.AttemptNo,
		ReplacesExecutionRef: execution.ReplacesExecutionRef, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
	}
}

func SameExecutionSessionAuthority(
	left ports.ExecutionSessionAuthority,
	right ports.ExecutionSessionAuthority,
) bool {
	return left.SessionRef == right.SessionRef &&
		left.ServicePrincipal == right.ServicePrincipal &&
		left.Request == right.Request
}

func writeExecutionSessionField(digest hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write([]byte(value))
}

func writeExecutionSessionUint(digest hash.Hash, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = digest.Write(encoded[:])
}
