package ports

import (
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

func validAgentLaunchRequest(t *testing.T) AgentLaunchRequest {
	t.Helper()
	executionRef, _ := goal.NewExecutionRef("execution:1")
	goalRef, _ := goal.NewGoalRef("goal:1")
	workItemRef, _ := goal.NewWorkItemRef("work:1")
	actorRef, _ := goal.NewActorRef("actor:local-owner")
	projectRef, _ := goal.NewProjectRef("project:default")
	colocacion, _ := NewAgentPlacementRef("placement:test")
	return AgentLaunchRequest{
		ExecutionRef:         executionRef,
		ReferenciaColocacion: colocacion,
		GoalRef:              goalRef,
		WorkItemRef:          workItemRef,
		PlanGeneration:       2,
		AppSpecGeneration:    3,
		ExecutionAttempt:     1,
		SpecHash:             "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ActorRef:             actorRef,
		ProjectRef:           projectRef,
		Objective:            "produce artifact",
		PhaseRef:             "phase-instance:build",
		PhaseKey:             "phase:build",
		PhaseTemplateRef:     "phase-template:program",
		PhaseInputRefs:       []string{"input:app-spec"},
		PhaseCriterionRefs:   []string{"criterion:tests-green"},
		RoleKey:              "role:worker",
		SkillRefs:            []string{"skill:go"},
		ToolRefs:             []string{"tool:go-test"},
		CapabilityRefs:       []string{"capability:code-review"},
		WriteSet:             []string{"internal/ports"},
		OutputContract:       string(goal.OutputContractEvidenceBundle),
		ArtifactMediaType:    "text/markdown",
		IdempotencyKey:       "launch:1",
		MaxOutputBytes:       1024,
		BudgetDemand: governance.BudgetDemand{
			Ref: "demand:work:1", Resources: governance.ResourceVector{
				Tokens: 1_000, ActiveTimeNS: int64(time.Minute), ProcessSlots: 1, DiskBytes: 1024,
			},
		},
		SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort:     governance.ReasoningEffortMedium,
	}
}

func validAgentLaunchReceipt(request AgentLaunchRequest) AgentLaunchReceipt {
	return AgentLaunchReceipt{
		ExecutionRef:      request.ExecutionRef,
		GoalRef:           request.GoalRef,
		WorkItemRef:       request.WorkItemRef,
		PlanGeneration:    request.PlanGeneration,
		AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt:  request.ExecutionAttempt,
		SpecHash:          request.SpecHash,
		ProviderRef:       "provider:fake",
		ModelRef:          "model:fake",
		AgentRef:          "agent:fake",
		ExternalRef:       "external:1",
		IdempotencyKey:    request.IdempotencyKey,
		ReceiptRef:        "receipt:launch:1",
		AcceptedAt:        time.Unix(10, 0).UTC(),
	}
}

func TestAgentPlacementRefIsOpaqueAndCanonical(t *testing.T) {
	ref, err := NewAgentPlacementRef("placement:codex:account-1")
	if err != nil || ref.String() != "placement:codex:account-1" {
		t.Fatalf("NewAgentPlacementRef() = %q, %v", ref.String(), err)
	}
	for _, invalid := range []string{"", " placement:one", "placement:one ", "placement:\x00"} {
		if _, err := NewAgentPlacementRef(invalid); AgentContractErrorCode(err) != "agent.placement_ref_invalid" {
			t.Fatalf("NewAgentPlacementRef(%q) no cerró el contrato: %v", invalid, err)
		}
	}
}

func TestAgentCapabilitiesValidateAndMatchNeutralRequirements(t *testing.T) {
	requirements := AgentRequirements{
		RoleKey:        "role:worker",
		SkillRefs:      []string{"skill:go"},
		ToolRefs:       []string{"tool:go-test"},
		CapabilityRefs: []string{"capability:patch"},
	}
	restricted := AgentCapabilities{
		ProviderRef: "provider:fake", ModelRef: "model:fake", AgentRef: "agent:fake",
		RoleKeys: []string{"role:worker", "role:reviewer"}, SkillRefs: []string{"skill:go"},
		ToolRefs: []string{"tool:go-test"}, CapabilityRefs: []string{"capability:patch"},
	}
	if err := ValidateAgentCapabilities(restricted); err != nil {
		t.Fatalf("ValidateAgentCapabilities() error = %v", err)
	}
	if !MatchAgentCapabilities(restricted, requirements) {
		t.Fatal("matching restricted capabilities rejected")
	}
	restricted.ToolRefs = nil
	if MatchAgentCapabilities(restricted, requirements) {
		t.Fatal("missing required tool accepted")
	}

	unrestricted := AgentCapabilities{
		ProviderRef: "provider:codex", ModelRef: "codex-default", AgentRef: "agent:codex", Unrestricted: true,
	}
	if !MatchAgentCapabilities(unrestricted, requirements) {
		t.Fatal("valid requirements rejected by unrestricted adapter")
	}
}

func TestAgentCapabilitiesRejectNonCanonicalOrDuplicateSets(t *testing.T) {
	valid := AgentCapabilities{
		ProviderRef: "provider:fake", ModelRef: "model:fake", AgentRef: "agent:fake",
		RoleKeys: []string{"role:worker"}, SkillRefs: []string{"skill:go"},
		ToolRefs: []string{"tool:test"}, CapabilityRefs: []string{"capability:patch"},
	}
	tests := map[string]func(*AgentCapabilities){
		"provider":       func(value *AgentCapabilities) { value.ProviderRef = " provider:fake" },
		"model":          func(value *AgentCapabilities) { value.ModelRef = "" },
		"agent":          func(value *AgentCapabilities) { value.AgentRef = "agent:fake " },
		"duplicate role": func(value *AgentCapabilities) { value.RoleKeys = []string{"role:worker", "role:worker"} },
		"invalid skill":  func(value *AgentCapabilities) { value.SkillRefs = []string{" skill:go"} },
		"duplicate tool": func(value *AgentCapabilities) { value.ToolRefs = []string{"tool:test", "tool:test"} },
		"invalid cap":    func(value *AgentCapabilities) { value.CapabilityRefs = []string{""} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if ValidateAgentCapabilities(candidate) == nil {
				t.Fatalf("invalid capabilities accepted: %+v", candidate)
			}
		})
	}
}

func TestAgentContractRejectsInvalidPlanMetadata(t *testing.T) {
	tests := map[string]func(*AgentLaunchRequest){
		"missing plan generation": func(request *AgentLaunchRequest) { request.PlanGeneration = 0 },
		"missing spec generation": func(request *AgentLaunchRequest) { request.AppSpecGeneration = 0 },
		"missing attempt":         func(request *AgentLaunchRequest) { request.ExecutionAttempt = 0 },
		"missing phase ref":       func(request *AgentLaunchRequest) { request.PhaseRef = "" },
		"missing phase":           func(request *AgentLaunchRequest) { request.PhaseKey = "" },
		"missing template":        func(request *AgentLaunchRequest) { request.PhaseTemplateRef = "" },
		"invalid input":           func(request *AgentLaunchRequest) { request.PhaseInputRefs = []string{" input:bad"} },
		"duplicate criterion":     func(request *AgentLaunchRequest) { request.PhaseCriterionRefs = []string{"criterion:a", "criterion:a"} },
		"missing role":            func(request *AgentLaunchRequest) { request.RoleKey = "" },
		"invalid skill":           func(request *AgentLaunchRequest) { request.SkillRefs = []string{" skill:go"} },
		"duplicate tool":          func(request *AgentLaunchRequest) { request.ToolRefs = []string{"tool:test", "tool:test"} },
		"invalid capability":      func(request *AgentLaunchRequest) { request.CapabilityRefs = []string{""} },
		"unknown output":          func(request *AgentLaunchRequest) { request.OutputContract = "unknown" },
		"empty write scope":       func(request *AgentLaunchRequest) { request.WriteSet = []string{""} },
		"unclean write scope":     func(request *AgentLaunchRequest) { request.WriteSet = []string{"internal/../ports"} },
		"duplicate scope":         func(request *AgentLaunchRequest) { request.WriteSet = []string{"internal/ports", "internal/ports"} },
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
	receipt := validAgentLaunchReceipt(request)
	if err := ValidateAgentLaunchReceipt(request, receipt); err != nil {
		t.Fatalf("ValidateAgentLaunchReceipt() error = %v", err)
	}
	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		SpecHash:     request.SpecHash,
		Status:       AgentCompleted,
		MediaType:    "text/markdown",
		Content:      []byte("artifact"),
		Usage:        governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if err := ValidateAgentObservation(observation, request.MaxOutputBytes); err != nil {
		t.Fatalf("ValidateAgentObservation() error = %v", err)
	}
}

func TestAgentContractRejectsIdentityMismatchAndFalseTerminalState(t *testing.T) {
	request := validAgentLaunchRequest(t)
	otherExecution, _ := goal.NewExecutionRef("execution:other")
	receipt := validAgentLaunchReceipt(request)
	receipt.ExecutionRef = otherExecution
	if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != "agent.receipt_execution_mismatch" {
		t.Fatalf("receipt error code = %q", code)
	}

	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		SpecHash:     request.SpecHash,
		Status:       AgentCompleted,
		ErrorCode:    "provider.failed",
		Usage:        governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code == "" {
		t.Fatal("contradictory terminal observation accepted")
	}
}

func TestAgentContractFailureDispositionIsClosedAndFailedOnly(t *testing.T) {
	request := validAgentLaunchRequest(t)
	valid := AgentObservation{
		ExecutionRef:       request.ExecutionRef,
		SpecHash:           request.SpecHash,
		Status:             AgentFailed,
		FailureDisposition: AgentFailureDispositionTerminalSecurity,
		ErrorCode:          "provider.security_failure",
		Usage:              governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt:         time.Unix(11, 0).UTC(),
	}
	if err := ValidateAgentObservation(valid, request.MaxOutputBytes); err != nil {
		t.Fatalf("terminal security observation rejected: %v", err)
	}

	conflict := valid
	conflict.Status = AgentRunning
	conflict.ErrorCode = ""
	if code := AgentContractErrorCode(ValidateAgentObservation(conflict, request.MaxOutputBytes)); code != "agent.observation_failure_disposition_conflict" {
		t.Fatalf("non-failed disposition code = %q", code)
	}

	unknown := valid
	unknown.FailureDisposition = AgentFailureDisposition("provider_retry_policy")
	if code := AgentContractErrorCode(ValidateAgentObservation(unknown, request.MaxOutputBytes)); code != "agent.observation_failure_disposition_invalid" {
		t.Fatalf("unknown disposition code = %q", code)
	}
}

func TestAgentContractRejectsReceiptSpecHashMismatchAndInvalidObservationHash(t *testing.T) {
	request := validAgentLaunchRequest(t)
	receipt := validAgentLaunchReceipt(request)
	receipt.SpecHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != "agent.receipt_spec_hash_mismatch" {
		t.Fatalf("receipt error code = %q", code)
	}

	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef,
		SpecHash:     "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Status:       AgentRunning,
		Usage:        governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
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
			receipt := validAgentLaunchReceipt(request)
			receipt.SpecHash = testCase.specHash
			if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != testCase.wantCode {
				t.Fatalf("receipt spec hash code = %q, want %q", code, testCase.wantCode)
			}
		})
	}
}

func TestAgentReceiptRejectsEveryCausalAndAdapterIdentityMismatch(t *testing.T) {
	request := validAgentLaunchRequest(t)
	otherGoal, _ := goal.NewGoalRef("goal:other")
	otherWorkItem, _ := goal.NewWorkItemRef("work:other")
	tests := map[string]struct {
		mutate   func(*AgentLaunchReceipt)
		wantCode string
	}{
		"goal":      {func(value *AgentLaunchReceipt) { value.GoalRef = otherGoal }, "agent.receipt_goal_mismatch"},
		"work item": {func(value *AgentLaunchReceipt) { value.WorkItemRef = otherWorkItem }, "agent.receipt_work_item_mismatch"},
		"plan":      {func(value *AgentLaunchReceipt) { value.PlanGeneration++ }, "agent.receipt_plan_generation_mismatch"},
		"app spec":  {func(value *AgentLaunchReceipt) { value.AppSpecGeneration++ }, "agent.receipt_app_spec_generation_mismatch"},
		"attempt":   {func(value *AgentLaunchReceipt) { value.ExecutionAttempt++ }, "agent.receipt_execution_attempt_mismatch"},
		"provider":  {func(value *AgentLaunchReceipt) { value.ProviderRef = "" }, "agent.receipt_provider_ref_required"},
		"model":     {func(value *AgentLaunchReceipt) { value.ModelRef = "" }, "agent.receipt_model_ref_required"},
		"agent":     {func(value *AgentLaunchReceipt) { value.AgentRef = "" }, "agent.receipt_agent_ref_required"},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			receipt := validAgentLaunchReceipt(request)
			testCase.mutate(&receipt)
			if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != testCase.wantCode {
				t.Fatalf("receipt error code = %q, want %q", code, testCase.wantCode)
			}
		})
	}
}

func TestAgentReceiptRequiresCausalFieldsEvenWhenRequestIsInvalid(t *testing.T) {
	tests := map[string]struct {
		mutate   func(*AgentLaunchRequest, *AgentLaunchReceipt)
		wantCode string
	}{
		"execution": {func(request *AgentLaunchRequest, receipt *AgentLaunchReceipt) {
			request.ExecutionRef, receipt.ExecutionRef = goal.ExecutionRef{}, goal.ExecutionRef{}
		}, "agent.receipt_execution_ref_required"},
		"goal": {func(request *AgentLaunchRequest, receipt *AgentLaunchReceipt) {
			request.GoalRef, receipt.GoalRef = goal.GoalRef{}, goal.GoalRef{}
		}, "agent.receipt_goal_ref_required"},
		"work item": {func(request *AgentLaunchRequest, receipt *AgentLaunchReceipt) {
			request.WorkItemRef, receipt.WorkItemRef = goal.WorkItemRef{}, goal.WorkItemRef{}
		}, "agent.receipt_work_item_ref_required"},
		"plan": {func(request *AgentLaunchRequest, receipt *AgentLaunchReceipt) {
			request.PlanGeneration, receipt.PlanGeneration = 0, 0
		}, "agent.receipt_plan_generation_required"},
		"app spec": {func(request *AgentLaunchRequest, receipt *AgentLaunchReceipt) {
			request.AppSpecGeneration, receipt.AppSpecGeneration = 0, 0
		}, "agent.receipt_app_spec_generation_required"},
		"attempt": {func(request *AgentLaunchRequest, receipt *AgentLaunchReceipt) {
			request.ExecutionAttempt, receipt.ExecutionAttempt = 0, 0
		}, "agent.receipt_execution_attempt_required"},
		"idempotency": {func(request *AgentLaunchRequest, receipt *AgentLaunchReceipt) {
			request.IdempotencyKey, receipt.IdempotencyKey = "", ""
		}, "agent.receipt_idempotency_key_required"},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			request := validAgentLaunchRequest(t)
			receipt := validAgentLaunchReceipt(request)
			testCase.mutate(&request, &receipt)
			if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != testCase.wantCode {
				t.Fatalf("receipt required error code = %q, want %q", code, testCase.wantCode)
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
		Usage:        governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt:   time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code != "agent.observation_output_too_large" {
		t.Fatalf("output cap error code = %q", code)
	}
}

func TestAgentContractValidatesBudgetRiskEffortAndUsageWithoutInference(t *testing.T) {
	request := validAgentLaunchRequest(t)
	mutations := map[string]func(*AgentLaunchRequest){
		"budget": func(value *AgentLaunchRequest) { value.BudgetDemand.Resources.Tokens = -1 },
		"criticality": func(value *AgentLaunchRequest) {
			value.SecurityCriticality = governance.SecurityCriticality("inferred")
		},
		"effort": func(value *AgentLaunchRequest) { value.ReasoningEffort = governance.ReasoningEffort("auto") },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := request
			mutate(&candidate)
			if AgentContractErrorCode(ValidateAgentLaunchRequest(candidate)) == "" {
				t.Fatal("invalid governance metadata accepted")
			}
		})
	}

	receipt := validAgentLaunchReceipt(request)
	receipt.ReceiptRef = ""
	if code := AgentContractErrorCode(ValidateAgentLaunchReceipt(request, receipt)); code != "agent.receipt_ref_required" {
		t.Fatalf("missing receipt ref code = %q", code)
	}
	observation := AgentObservation{
		ExecutionRef: request.ExecutionRef, SpecHash: request.SpecHash, Status: AgentRunning,
		Usage: governance.ResourceUsage{
			Resources: governance.ResourceVector{Tokens: 1}, Quality: governance.UsageQualityUnknown,
		},
		ObservedAt: time.Unix(11, 0).UTC(),
	}
	if code := AgentContractErrorCode(ValidateAgentObservation(observation, request.MaxOutputBytes)); code != "agent.observation_usage_invalid" {
		t.Fatalf("invalid usage code = %q", code)
	}
}
