package application

import (
	"context"
	"encoding/binary"
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
	if request.ProjectRef.String() == "" || request.GoalRef.String() == "" ||
		request.WorkItemRef.String() == "" || request.ExecutionRef.String() == "" ||
		request.ExecutionAttempt == 0 || request.PlanGeneration == 0 ||
		request.AppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(request.SpecHash) ||
		(request.ExecutionAttempt == 1 && request.ReplacesExecutionRef.String() != "") ||
		(request.ExecutionAttempt > 1 && request.ReplacesExecutionRef.String() == "") ||
		strings.TrimSpace(authenticationMethod) != authenticationMethod || authenticationMethod == "" ||
		strings.ContainsAny(authenticationMethod, "\x00\r\n") {
		return ports.ExecutionSessionAuthority{}, errExecutionSessionInvalid
	}
	digest := fingerprintDigest(executionSessionDerivationVersion,
		request.ProjectRef.String(), request.GoalRef.String(),
		request.WorkItemRef.String(), request.ExecutionRef.String())
	writeExecutionSessionUint(digest, request.ExecutionAttempt)
	writeFingerprintField(digest, request.ReplacesExecutionRef.String())
	writeExecutionSessionUint(digest, uint64(request.PlanGeneration))
	writeExecutionSessionUint(digest, uint64(request.AppSpecGeneration))
	writeFingerprintField(digest, request.SpecHash)
	writeFingerprintField(digest, authenticationMethod)
	suffix := fingerprintHex(digest)
	artifactSuffix := executionSessionAccessSuffix("artifact-access", suffix)
	mcpSuffix := executionSessionAccessSuffix("mcp-access", suffix)
	mailboxSuffix := executionSessionAccessSuffix("mailbox-endpoint", suffix)

	sessionRef, sessionErr := ports.NewExecutionSessionRef("execution-session:sha256:" + suffix)
	artifactAccessRef, artifactErr := ports.NewExecutionArtifactAccessRef("artifact-access:execution:sha256:" + artifactSuffix)
	mcpAccessRef, mcpErr := ports.NewExecutionMCPAccessRef("mcp-access:execution:sha256:" + mcpSuffix)
	mailboxEndpointRef, mailboxErr := ports.NewExecutionMailboxEndpointRef("mailbox-endpoint:execution:sha256:" + mailboxSuffix)
	principalRef, principalErr := identity.NewPrincipalRef("principal:execution:sha256:" + suffix)
	actorRef, actorErr := goal.NewActorRef("actor:execution:sha256:" + suffix)
	if sessionErr != nil || artifactErr != nil || mcpErr != nil || mailboxErr != nil ||
		principalErr != nil || actorErr != nil {
		return ports.ExecutionSessionAuthority{}, errExecutionSessionInvalid
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindService, authenticationMethod,
	)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, errExecutionSessionInvalid
	}
	return ports.ExecutionSessionAuthority{
		SessionRef: sessionRef, ArtifactAccessRef: artifactAccessRef, MCPAccessRef: mcpAccessRef,
		MailboxEndpointRef: mailboxEndpointRef, ServicePrincipal: principal, Request: request,
	}, nil
}

func executionSessionAccessSuffix(domain, authoritySuffix string) string {
	return fingerprintHex(fingerprintDigest(
		"orquesta.execution-session-access.v1", domain, authoritySuffix,
	))
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

func writeExecutionSessionUint(digest hash.Hash, value uint64) {
	_ = binary.Write(digest, binary.BigEndian, value)
}
