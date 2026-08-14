package ports

import (
	"testing"
	"time"

	"orquesta/internal/goal"
)

func validAgentStopRequest(t *testing.T) AgentStopRequest {
	t.Helper()
	launch := validAgentLaunchReceipt(validAgentLaunchRequest(t))
	launch.LaunchActionFence = 11
	return AgentStopRequest{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, LaunchActionFence: 11,
		StopEffectAttemptRef: "effect-attempt:stop:1", StopActionFence: 13, SpecHash: launch.SpecHash,
		ProviderRef: launch.ProviderRef, ModelRef: launch.ModelRef, AgentRef: launch.AgentRef,
		ExternalRef: launch.ExternalRef, Mode: AgentStopCooperative, IdempotencyKey: "stop:1",
	}
}

func validAgentStopReceipt(request AgentStopRequest, status AgentStopStatus) AgentStopReceipt {
	receipt := AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, StopEffectAttemptRef: request.StopEffectAttemptRef,
		StopActionFence: request.StopActionFence, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: status,
	}
	if status == AgentStopped || status == AgentStopAlreadyStopped ||
		status == AgentStopAlreadyCompleted || status == AgentStopAlreadyFailed {
		receipt.ReceiptRef = "receipt:stop:1"
		receipt.ConfirmedAt = time.Unix(20, 0).UTC()
	}
	return receipt
}

func TestAgentStopCapabilitiesAdvertiseModesExactly(t *testing.T) {
	capabilities := AgentControlCapabilities{CooperativeStop: true}
	if !SupportsAgentStopMode(capabilities, AgentStopCooperative) ||
		SupportsAgentStopMode(capabilities, AgentStopForced) ||
		SupportsAgentStopMode(capabilities, AgentStopMode("kill")) {
		t.Fatalf("capability matching is not exact: %+v", capabilities)
	}
}

func TestAgentStopRequestRejectsEveryInvalidIdentityMutation(t *testing.T) {
	tests := map[string]func(*AgentStopRequest){
		"execution":       func(value *AgentStopRequest) { value.ExecutionRef = goal.ExecutionRef{} },
		"goal":            func(value *AgentStopRequest) { value.GoalRef = goal.GoalRef{} },
		"work item":       func(value *AgentStopRequest) { value.WorkItemRef = goal.WorkItemRef{} },
		"plan":            func(value *AgentStopRequest) { value.PlanGeneration = 0 },
		"app spec":        func(value *AgentStopRequest) { value.AppSpecGeneration = 0 },
		"attempt":         func(value *AgentStopRequest) { value.ExecutionAttempt = 0 },
		"launch fence":    func(value *AgentStopRequest) { value.LaunchActionFence = 0 },
		"effect attempt":  func(value *AgentStopRequest) { value.StopEffectAttemptRef = "" },
		"action fence":    func(value *AgentStopRequest) { value.StopActionFence = 0 },
		"hash missing":    func(value *AgentStopRequest) { value.SpecHash = "" },
		"hash invalid":    func(value *AgentStopRequest) { value.SpecHash = "ABC" },
		"provider":        func(value *AgentStopRequest) { value.ProviderRef = " provider:fake" },
		"model":           func(value *AgentStopRequest) { value.ModelRef = "" },
		"agent":           func(value *AgentStopRequest) { value.AgentRef = "agent:fake " },
		"external":        func(value *AgentStopRequest) { value.ExternalRef = " " },
		"mode":            func(value *AgentStopRequest) { value.Mode = AgentStopMode("kill") },
		"idempotency key": func(value *AgentStopRequest) { value.IdempotencyKey = "" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := validAgentStopRequest(t)
			mutate(&request)
			if code := AgentContractErrorCode(ValidateAgentStopRequest(request)); code == "" {
				t.Fatalf("invalid stop accepted: %+v", request)
			}
		})
	}
}

func TestAgentStopTargetRejectsEveryLaunchMismatch(t *testing.T) {
	launch := validAgentLaunchReceipt(validAgentLaunchRequest(t))
	launch.LaunchActionFence = 11
	base := validAgentStopRequest(t)
	otherExecution, _ := goal.NewExecutionRef("execution:other")
	otherGoal, _ := goal.NewGoalRef("goal:other")
	otherWork, _ := goal.NewWorkItemRef("work:other")
	tests := map[string]func(*AgentStopRequest){
		"execution": func(value *AgentStopRequest) { value.ExecutionRef = otherExecution },
		"goal":      func(value *AgentStopRequest) { value.GoalRef = otherGoal },
		"work item": func(value *AgentStopRequest) { value.WorkItemRef = otherWork },
		"plan":      func(value *AgentStopRequest) { value.PlanGeneration++ },
		"app spec":  func(value *AgentStopRequest) { value.AppSpecGeneration++ },
		"attempt":   func(value *AgentStopRequest) { value.ExecutionAttempt++ },
		"launch fence": func(value *AgentStopRequest) {
			value.LaunchActionFence++
		},
		"hash": func(value *AgentStopRequest) {
			value.SpecHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		"provider": func(value *AgentStopRequest) { value.ProviderRef = "provider:other" },
		"model":    func(value *AgentStopRequest) { value.ModelRef = "model:other" },
		"agent":    func(value *AgentStopRequest) { value.AgentRef = "agent:other" },
		"external": func(value *AgentStopRequest) { value.ExternalRef = "external:other" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			if code := AgentContractErrorCode(ValidateAgentStopTarget(launch, request)); code == "" {
				t.Fatalf("mismatched stop target accepted: %+v", request)
			}
		})
	}
}

func TestAgentStopReceiptStatusAndConfirmationAreExact(t *testing.T) {
	request := validAgentStopRequest(t)
	for _, status := range []AgentStopStatus{
		AgentStopped, AgentStopAlreadyStopped, AgentStopAlreadyCompleted,
		AgentStopAlreadyFailed, AgentStopPending, AgentStopUnsupported,
	} {
		receipt := validAgentStopReceipt(request, status)
		if err := ValidateAgentStopReceipt(request, receipt); err != nil {
			t.Fatalf("status %q rejected: %v", status, err)
		}
	}

	missingRef := validAgentStopReceipt(request, AgentStopped)
	missingRef.ReceiptRef = ""
	if code := AgentContractErrorCode(ValidateAgentStopReceipt(request, missingRef)); code != "agent.stop_receipt_receipt_ref_required" {
		t.Fatalf("missing confirmation ref code = %q", code)
	}
	missingTime := validAgentStopReceipt(request, AgentStopped)
	missingTime.ConfirmedAt = time.Time{}
	if code := AgentContractErrorCode(ValidateAgentStopReceipt(request, missingTime)); code != "agent.stop_receipt_confirmed_at_required" {
		t.Fatalf("missing confirmation time code = %q", code)
	}
	falseConfirmation := validAgentStopReceipt(request, AgentStopUnsupported)
	falseConfirmation.ReceiptRef = "receipt:false"
	if code := AgentContractErrorCode(ValidateAgentStopReceipt(request, falseConfirmation)); code != "agent.stop_receipt_unconfirmed_has_confirmation" {
		t.Fatalf("false confirmation code = %q", code)
	}
	invalid := validAgentStopReceipt(request, AgentStopStatus("completed"))
	if code := AgentContractErrorCode(ValidateAgentStopReceipt(request, invalid)); code == "" {
		t.Fatal("invalid stop status accepted")
	}
}

func TestAgentStopReceiptRejectsEveryCausalMutation(t *testing.T) {
	request := validAgentStopRequest(t)
	otherExecution, _ := goal.NewExecutionRef("execution:other")
	otherGoal, _ := goal.NewGoalRef("goal:other")
	otherWork, _ := goal.NewWorkItemRef("work:other")
	tests := map[string]func(*AgentStopReceipt){
		"execution": func(value *AgentStopReceipt) { value.ExecutionRef = otherExecution },
		"goal":      func(value *AgentStopReceipt) { value.GoalRef = otherGoal },
		"work item": func(value *AgentStopReceipt) { value.WorkItemRef = otherWork },
		"plan":      func(value *AgentStopReceipt) { value.PlanGeneration++ },
		"app spec":  func(value *AgentStopReceipt) { value.AppSpecGeneration++ },
		"attempt":   func(value *AgentStopReceipt) { value.ExecutionAttempt++ },
		"effect attempt": func(value *AgentStopReceipt) {
			value.StopEffectAttemptRef = "effect-attempt:stop:other"
		},
		"action fence": func(value *AgentStopReceipt) { value.StopActionFence++ },
		"hash": func(value *AgentStopReceipt) {
			value.SpecHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		"provider":    func(value *AgentStopReceipt) { value.ProviderRef = "provider:other" },
		"model":       func(value *AgentStopReceipt) { value.ModelRef = "model:other" },
		"agent":       func(value *AgentStopReceipt) { value.AgentRef = "agent:other" },
		"external":    func(value *AgentStopReceipt) { value.ExternalRef = "external:other" },
		"mode":        func(value *AgentStopReceipt) { value.Mode = AgentStopForced },
		"idempotency": func(value *AgentStopReceipt) { value.IdempotencyKey = "stop:other" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			receipt := validAgentStopReceipt(request, AgentStopped)
			mutate(&receipt)
			if code := AgentContractErrorCode(ValidateAgentStopReceipt(request, receipt)); code == "" {
				t.Fatalf("mismatched stop receipt accepted: %+v", receipt)
			}
		})
	}
}
