package application

import "time"

func consumptionReceipt(claim ActionClaim, outcome ActionConsumptionOutcome, code string,
	at time.Time,
) ActionConsumptionReceipt {
	return ActionConsumptionReceipt{
		ActionRef: claim.Action.Ref, Kind: claim.Action.Kind,
		GoalRef: claim.Action.GoalRef, WorkItemRef: claim.Action.WorkItemRef,
		ExecutionRef: claim.Action.ExecutionRef, ChangeRef: claim.Action.ChangeRef,
		PlanGeneration:     claim.Action.PlanGeneration,
		WorkItemGeneration: claim.Action.WorkItemGeneration, Fence: claim.Fence,
		DeliveryAttempt: claim.DeliveryAttempt, ClaimToken: claim.Token,
		WorkerRef: claim.WorkerRef, Outcome: outcome, ErrorCode: code, ConsumedAt: at.UTC(),
	}
}
