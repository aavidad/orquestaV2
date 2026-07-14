package ports

import (
	"testing"
	"time"

	"orquesta/internal/goal"
)

func validAgentLaunchRequest(t *testing.T) AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:1")
	goalRef, _ := goal.NewGoalRef("goal:1")
	workItemRef, _ := goal.NewWorkItemRef("work:1")
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")
	return AgentLaunchRequest{
		ExecutionRef:      executionRef,
		GoalRef:           goalRef,
		WorkItemRef:       workItemRef,
		ActorRef:          actorRef,
		ProjectRef:        projectRef,
		Objective:         "produce artifact",
		ArtifactMediaType: "text/markdown",
		IdempotencyKey:    "launch:1",
		MaxOutputBytes:    1024,
	}
}

func TestAgentContractAcceptsCausalLaunchAndTerminalObservation(t *testing.T) {
	request := validAgentLaunchRequest(t)
	if err := ValidateAgentLaunchRequest(request); err != nil {
		t.Fatalf("ValidateAgentLaunchRequest() error = %v", err)
	}
	receipt := AgentLaunchReceipt{
		ExecutionRef:   request.ExecutionRef,
		ProviderRef:    "provider:fake",
		ExternalRef:    "external:1",
		IdempotencyKey: request.IdempotencyKey,
		AcceptedAt:     time.Unix(10, 0).UTC(),
	}
	if err := ValidateAgentLaunchReceipt(request, receipt); err != nil {
		t.Fatalf("ValidateAgentLaunchReceipt() error = %v", err)
	}
	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		Status:       AgentCompleted,
		MediaType:    "text/markdown",
		Content:      []byte("artifact"),
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if err := ValidateAgentObservation(observation, request.MaxOutputBytes); err != nil {
		t.Fatalf("ValidateAgentObservation() error = %v", err)
	}
}

func TestAgentContractRejectsIdentityMismatchAndFalseTerminalState(t *testing.T) {
	request := validAgentLaunchRequest(t)
	otherExecution, _ := goal.NewExecutionRef("execution:other")
	receipt := AgentLaunchReceipt{
		ExecutionRef:   otherExecution,
		ProviderRef:    "provider:fake",
		ExternalRef:    "external:1",
		IdempotencyKey: request.IdempotencyKey,
		AcceptedAt:     time.Unix(10, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != "agent.receipt_execution_mismatch" {
		t.Fatalf("receipt error code = %q", code)
	}

	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		Status:       AgentCompleted,
		ErrorCode:    "provider.failed",
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code == "" {
		t.Fatal("contradictory terminal observation accepted")
	}
}

func TestAgentContractCapsOutputBeforeApplication(t *testing.T) {
	request := validAgentLaunchRequest(t)
	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		Status:       AgentCompleted,
		MediaType:    "text/plain",
		Content:      make([]byte, request.MaxOutputBytes+1),
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code != "agent.observation_output_too_large" {
		t.Fatalf("output cap error code = %q", code)
	}
}
