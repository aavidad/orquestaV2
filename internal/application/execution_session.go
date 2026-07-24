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

var errExecutionSessionInvalid = errors.New("application.execution_session_invalid")

type ExecutionSessionAuthoritySource interface {
	ExecutionSessionAuthority(context.Context, goal.ExecutionRef, string) (ports.ExecutionSessionAuthority, error)
}

func DeriveExecutionSessionAuthority(
	request ports.ExecutionSessionEnsureRequest,
	authenticationMethod string,
) (ports.ExecutionSessionAuthority, error) {
	if err := ValidateExecutionSessionEnsureRequest(request); err != nil ||
		strings.TrimSpace(authenticationMethod) != authenticationMethod || authenticationMethod == "" ||
		strings.ContainsAny(authenticationMethod, "\x00\r\n") {
		return ports.ExecutionSessionAuthority{}, errExecutionSessionInvalid
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

	sessionRef, sessionErr := ports.NewExecutionSessionRef("execution-session:sha256:" + suffix)
	principalRef, principalErr := identity.NewPrincipalRef("principal:execution:sha256:" + suffix)
	actorRef, actorErr := goal.NewActorRef("actor:execution:sha256:" + suffix)
	if sessionErr != nil || principalErr != nil || actorErr != nil {
		return ports.ExecutionSessionAuthority{}, errExecutionSessionInvalid
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindService, authenticationMethod,
	)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, errExecutionSessionInvalid
	}
	return ports.ExecutionSessionAuthority{SessionRef: sessionRef, ServicePrincipal: principal, Request: request}, nil
}

func ValidateExecutionSessionEnsureRequest(request ports.ExecutionSessionEnsureRequest) error {
	if request.ProjectRef.String() == "" || request.GoalRef.String() == "" ||
		request.WorkItemRef.String() == "" || request.ExecutionRef.String() == "" ||
		request.ExecutionAttempt == 0 || request.PlanGeneration == 0 ||
		request.AppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(request.SpecHash) ||
		(request.ExecutionAttempt == 1 && request.ReplacesExecutionRef.String() != "") ||
		(request.ExecutionAttempt > 1 && request.ReplacesExecutionRef.String() == "") {
		return errExecutionSessionInvalid
	}
	return nil
}

func ExecutionSessionRequest(
	aggregate goal.Goal,
	execution ExecutionRecord,
) ports.ExecutionSessionEnsureRequest {
	return ports.ExecutionSessionEnsureRequest{
		ProjectRef: aggregate.Project(), GoalRef: aggregate.Ref(), WorkItemRef: execution.WorkItemRef,
		ExecutionRef: execution.Ref, ExecutionAttempt: execution.AttemptNo, ReplacesExecutionRef: execution.ReplacesExecutionRef,
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
	}
}

func writeExecutionSessionField(digest hash.Hash, value string) {
	_ = binary.Write(digest, binary.BigEndian, uint64(len(value)))
	_, _ = digest.Write([]byte(value))
}

func writeExecutionSessionUint(digest hash.Hash, value uint64) {
	_ = binary.Write(digest, binary.BigEndian, value)
}
