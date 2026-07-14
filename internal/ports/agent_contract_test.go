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
		ExecutionRef:       executionRef,
		GoalRef:            goalRef,
		WorkItemRef:        workItemRef,
		SpecHash:           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ActorRef:           actorRef,
		ProjectRef:         projectRef,
		Objective:          "produce artifact",
		PhaseRef:           "phase-instance:build",
		PhaseKey:           "phase:build",
		PhaseTemplateRef:   "phase-template:program",
		PhaseInputRefs:     []string{"input:app-spec"},
		PhaseCriterionRefs: []string{"criterion:tests-green"},
		RoleKey:            "role:worker",
		SkillRefs:          []string{"skill:go"},
		ToolRefs:           []string{"tool:go-test"},
		CapabilityRefs:     []string{"capability:code-review"},
		WriteSet:           []string{"internal/ports"},
		OutputContract:     string(goal.OutputContractEvidenceBundle),
		ArtifactMediaType:  "text/markdown",
		IdempotencyKey:     "launch:1",
		MaxOutputBytes:     1024,
	}
}

func TestAgentContractRejectsInvalidPlanMetadata(t *testing.T) {
	tests := map[string]func(*AgentLaunchRequest){
		"missing phase ref":   func(request *AgentLaunchRequest) { request.PhaseRef = "" },
		"missing phase":       func(request *AgentLaunchRequest) { request.PhaseKey = "" },
		"missing template":    func(request *AgentLaunchRequest) { request.PhaseTemplateRef = "" },
		"invalid input":       func(request *AgentLaunchRequest) { request.PhaseInputRefs = []string{" input:bad"} },
		"duplicate criterion": func(request *AgentLaunchRequest) { request.PhaseCriterionRefs = []string{"criterion:a", "criterion:a"} },
		"missing role":        func(request *AgentLaunchRequest) { request.RoleKey = "" },
		"invalid skill":       func(request *AgentLaunchRequest) { request.SkillRefs = []string{" skill:go"} },
		"duplicate tool":      func(request *AgentLaunchRequest) { request.ToolRefs = []string{"tool:test", "tool:test"} },
		"invalid capability":  func(request *AgentLaunchRequest) { request.CapabilityRefs = []string{""} },
		"unknown output":      func(request *AgentLaunchRequest) { request.OutputContract = "unknown" },
		"empty write scope":   func(request *AgentLaunchRequest) { request.WriteSet = []string{""} },
		"unclean write scope": func(request *AgentLaunchRequest) { request.WriteSet = []string{"internal/../ports"} },
		"duplicate scope":     func(request *AgentLaunchRequest) { request.WriteSet = []string{"internal/ports", "internal/ports"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := validAgentLaunchRequest(t)
			mutate(&request)
			if code := AgentContractErrorCode(ValidateAgentLaunchRequest(request)); code == "" {
				t.Fatalf("invalid metadata accepted: %+v", request)
			}
		})
	}
}

func TestAgentContractRejectsMissingOrNonCanonicalSpecHash(t *testing.T) {
	tests := map[string]string{
		"missing":   "",
		"short":     "0123456789abcdef",
		"prefixed":  "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"uppercase": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeF",
		"non-hex":   "g123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	for name, specHash := range tests {
		t.Run(name, func(t *testing.T) {
			request := validAgentLaunchRequest(t)
			request.SpecHash = specHash
			if code := AgentContractErrorCode(ValidateAgentLaunchRequest(request)); code == "" {
				t.Fatalf("invalid spec hash accepted: %q", specHash)
			}
		})
	}
}

func TestAgentContractAcceptsCausalLaunchAndTerminalObservation(t *testing.T) {
	request := validAgentLaunchRequest(t)
	if err := ValidateAgentLaunchRequest(request); err != nil {
		t.Fatalf("ValidateAgentLaunchRequest() error = %v", err)
	}
	receipt := AgentLaunchReceipt{
		ExecutionRef:   request.ExecutionRef,
		SpecHash:       request.SpecHash,
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
		SpecHash:     request.SpecHash,
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
		SpecHash:       request.SpecHash,
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
		SpecHash:     request.SpecHash,
		Status:       AgentCompleted,
		ErrorCode:    "provider.failed",
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code == "" {
		t.Fatal("contradictory terminal observation accepted")
	}
}

func TestAgentContractRejectsReceiptSpecHashMismatchAndInvalidObservationHash(t *testing.T) {
	request := validAgentLaunchRequest(t)
	receipt := AgentLaunchReceipt{
		ExecutionRef:   request.ExecutionRef,
		SpecHash:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ProviderRef:    "provider:fake",
		ExternalRef:    "external:1",
		IdempotencyKey: request.IdempotencyKey,
		AcceptedAt:     time.Unix(10, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != "agent.receipt_spec_hash_mismatch" {
		t.Fatalf("receipt error code = %q", code)
	}

	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		SpecHash:     "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Status:       AgentRunning,
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code != "agent.observation_spec_hash_invalid" {
		t.Fatalf("observation error code = %q", code)
	}
}

func TestAgentContractClassifiesReceiptSpecHashBeforeCausalMismatch(t *testing.T) {
	request := validAgentLaunchRequest(t)
	for _, testCase := range []struct {
		name     string
		specHash string
		wantCode string
	}{
		{name: "required", specHash: "", wantCode: "agent.receipt_spec_hash_required"},
		{name: "invalid", specHash: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", wantCode: "agent.receipt_spec_hash_invalid"},
		{name: "mismatch", specHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", wantCode: "agent.receipt_spec_hash_mismatch"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			receipt := AgentLaunchReceipt{
				ExecutionRef: request.ExecutionRef, SpecHash: testCase.specHash,
				ProviderRef: "provider:fake", ExternalRef: "external:1",
				IdempotencyKey: request.IdempotencyKey, AcceptedAt: time.Unix(10, 0).UTC(),
			}
			if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != testCase.wantCode {
				t.Fatalf("receipt spec hash code = %q, want %q", code, testCase.wantCode)
			}
		})
	}
}

func TestAgentContractCapsOutputBeforeApplication(t *testing.T) {
	request := validAgentLaunchRequest(t)
	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		SpecHash:     request.SpecHash,
		Status:       AgentCompleted,
		MediaType:    "text/plain",
		Content:      make([]byte, request.MaxOutputBytes+1),
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code != "agent.observation_output_too_large" {
		t.Fatalf("output cap error code = %q", code)
	}
}
