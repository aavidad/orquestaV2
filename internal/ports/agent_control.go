package ports

import (
	"fmt"
	"strings"
	"time"

	"orquesta/internal/goal"
)

type AgentStopMode string

const (
	AgentStopCooperative AgentStopMode = "cooperative"
	AgentStopForced      AgentStopMode = "forced"
)

type AgentStopStatus string

const (
	AgentStopPending          AgentStopStatus = "pending"
	AgentStopped              AgentStopStatus = "stopped"
	AgentStopAlreadyStopped   AgentStopStatus = "already_stopped"
	AgentStopAlreadyCompleted AgentStopStatus = "already_completed"
	AgentStopAlreadyFailed    AgentStopStatus = "already_failed"
	AgentStopUnsupported      AgentStopStatus = "unsupported"
)

type AgentControlCapabilities struct {
	CooperativeStop bool
	ForcedStop      bool
}

type AgentStopRequest struct {
	ExecutionRef      goal.ExecutionRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	ExecutionAttempt  uint64
	SpecHash          string
	ProviderRef       string
	ModelRef          string
	AgentRef          string
	ExternalRef       string
	Mode              AgentStopMode
	IdempotencyKey    string
}

type AgentStopReceipt struct {
	ExecutionRef      goal.ExecutionRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	ExecutionAttempt  uint64
	SpecHash          string
	ProviderRef       string
	ModelRef          string
	AgentRef          string
	ExternalRef       string
	Mode              AgentStopMode
	IdempotencyKey    string
	Status            AgentStopStatus
	ReceiptRef        string
	ConfirmedAt       time.Time
}

func SupportsAgentStopMode(capabilities AgentControlCapabilities, mode AgentStopMode) bool {
	switch mode {
	case AgentStopCooperative:
		return capabilities.CooperativeStop
	case AgentStopForced:
		return capabilities.ForcedStop
	default:
		return false
	}
}

func ValidateAgentStopRequest(request AgentStopRequest) error {
	switch {
	case request.ExecutionRef.String() == "":
		return stopContractError("execution_ref_required")
	case request.GoalRef.String() == "":
		return stopContractError("goal_ref_required")
	case request.WorkItemRef.String() == "":
		return stopContractError("work_item_ref_required")
	case request.PlanGeneration == 0:
		return stopContractError("plan_generation_required")
	case request.AppSpecGeneration == 0:
		return stopContractError("app_spec_generation_required")
	case request.ExecutionAttempt == 0:
		return stopContractError("execution_attempt_required")
	case request.SpecHash == "":
		return stopContractError("spec_hash_required")
	case !goal.IsCanonicalAppSpecHash(request.SpecHash):
		return stopContractError("spec_hash_invalid")
	case !validAgentIdentityRef(request.ProviderRef):
		return stopContractError("provider_ref_required")
	case !validAgentIdentityRef(request.ModelRef):
		return stopContractError("model_ref_required")
	case !validAgentIdentityRef(request.AgentRef):
		return stopContractError("agent_ref_required")
	case strings.TrimSpace(request.ExternalRef) == "":
		return stopContractError("external_ref_required")
	case request.Mode != AgentStopCooperative && request.Mode != AgentStopForced:
		return stopContractError("mode_invalid")
	case strings.TrimSpace(request.IdempotencyKey) == "":
		return stopContractError("idempotency_key_required")
	default:
		return nil
	}
}

// ValidateAgentStopTarget binds a stop to the exact launch accepted by the
// adapter. The stop idempotency key and mode are deliberately new control data.
func ValidateAgentStopTarget(launch AgentLaunchReceipt, stop AgentStopRequest) error {
	if err := ValidateAgentStopRequest(stop); err != nil {
		return err
	}
	checks := []struct {
		matches bool
		field   string
	}{
		{stop.ExecutionRef == launch.ExecutionRef, "execution"},
		{stop.GoalRef == launch.GoalRef, "goal"},
		{stop.WorkItemRef == launch.WorkItemRef, "work_item"},
		{stop.PlanGeneration == launch.PlanGeneration, "plan_generation"},
		{stop.AppSpecGeneration == launch.AppSpecGeneration, "app_spec_generation"},
		{stop.ExecutionAttempt == launch.ExecutionAttempt, "execution_attempt"},
		{stop.SpecHash == launch.SpecHash, "spec_hash"},
		{stop.ProviderRef == launch.ProviderRef, "provider_ref"},
		{stop.ModelRef == launch.ModelRef, "model_ref"},
		{stop.AgentRef == launch.AgentRef, "agent_ref"},
		{stop.ExternalRef == launch.ExternalRef, "external_ref"},
	}
	for _, check := range checks {
		if !check.matches {
			return stopContractError(check.field + "_mismatch")
		}
	}
	return nil
}

func ValidateAgentStopReceipt(request AgentStopRequest, receipt AgentStopReceipt) error {
	if err := ValidateAgentStopRequest(request); err != nil {
		return err
	}
	checks := []struct {
		matches bool
		field   string
	}{
		{receipt.ExecutionRef == request.ExecutionRef, "execution"},
		{receipt.GoalRef == request.GoalRef, "goal"},
		{receipt.WorkItemRef == request.WorkItemRef, "work_item"},
		{receipt.PlanGeneration == request.PlanGeneration, "plan_generation"},
		{receipt.AppSpecGeneration == request.AppSpecGeneration, "app_spec_generation"},
		{receipt.ExecutionAttempt == request.ExecutionAttempt, "execution_attempt"},
		{receipt.SpecHash == request.SpecHash, "spec_hash"},
		{receipt.ProviderRef == request.ProviderRef, "provider_ref"},
		{receipt.ModelRef == request.ModelRef, "model_ref"},
		{receipt.AgentRef == request.AgentRef, "agent_ref"},
		{receipt.ExternalRef == request.ExternalRef, "external_ref"},
		{receipt.Mode == request.Mode, "mode"},
		{receipt.IdempotencyKey == request.IdempotencyKey, "idempotency"},
	}
	for _, check := range checks {
		if !check.matches {
			return stopReceiptError(check.field + "_mismatch")
		}
	}

	switch receipt.Status {
	case AgentStopped, AgentStopAlreadyStopped, AgentStopAlreadyCompleted, AgentStopAlreadyFailed:
		if strings.TrimSpace(receipt.ReceiptRef) == "" {
			return stopReceiptError("receipt_ref_required")
		}
		if receipt.ConfirmedAt.IsZero() {
			return stopReceiptError("confirmed_at_required")
		}
	case AgentStopPending, AgentStopUnsupported:
		if receipt.ReceiptRef != "" || !receipt.ConfirmedAt.IsZero() {
			return stopReceiptError("unconfirmed_has_confirmation")
		}
	default:
		return stopReceiptError(fmt.Sprintf("status_invalid:%s", receipt.Status))
	}
	return nil
}

func stopContractError(suffix string) error {
	return &AgentContractError{Code: "agent.stop_" + suffix}
}

func stopReceiptError(suffix string) error {
	return &AgentContractError{Code: "agent.stop_receipt_" + suffix}
}
