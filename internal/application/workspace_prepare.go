package application

import (
	"context"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func newChangeSetRef(ctx context.Context, ids IDGenerator) (ports.ChangeSetRef, error) {
	value, err := ids.NewID(ctx, "change-set")
	if err != nil {
		return ports.ChangeSetRef{}, err
	}
	return ports.NewChangeSetRef(value)
}

func (orchestrator *Orchestrator) processPrepareWorkspace(ctx context.Context, claim ActionClaim) error {
	if orchestrator.workspaceManager == nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, "workspace.manager_unavailable")
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	if err := validateClaimedRecord(claim, record, ActionPrepareWorkspace); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	if err := validateClaimedEffect(claim, orchestrator.clock.Now()); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	item, itemFound := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := executionForAction(record, claim.Action)
	authority, authorityFound := workItemAuthorityFor(record.WorkItemAuthorities, claim.Action.WorkItemRef)
	if !itemFound || !executionFound || !authorityFound || validateWorkItemAuthority(record.Goal, authority) != nil {
		return orchestrator.quarantine(ctx, claim, "application.workspace_scope_invalid")
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, err.Error())
	}
	attempt, err := orchestrator.beginEffectAttempt(ctx, claim, orchestrator.clock.Now())
	if err != nil {
		return err
	}
	request := prepareWorkspaceRequest(record, item, execution, authority, claim, attempt)
	if workspacePrepareTargetDigest(record.Goal, item, execution) != claim.Action.EffectIntent.TargetDigest {
		return orchestrator.quarantine(ctx, claim, "application.effect_target_mismatch")
	}
	prepared, prepareErr := orchestrator.workspaceManager.Prepare(ctx, request)
	if prepareErr != nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, workspaceErrorCode(prepareErr, "workspace.prepare_failed"))
	}
	if err := ports.ValidateWorkspacePrepared(request, prepared); err != nil {
		return orchestrator.quarantine(ctx, claim, ports.WorkspaceContractErrorCode(err))
	}
	now := orchestrator.clock.Now().UTC()
	externalReceipt, err := effectReceipt(claim, attempt, prepared.ReceiptRef, EffectStatusPrepared, unknownUsage(), now)
	if err != nil {
		return err
	}
	binding := WorkspaceBinding{
		Ref: prepared.WorkspaceRef, PrincipalRef: authority.PrincipalRef,
		ActorRef: record.Goal.Actor(), ProjectRef: record.Goal.Project(), RepositoryRef: prepared.RepositoryRef,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		WriteSet: request.WriteSet, WriteSetDigest: prepared.WriteSetDigest,
		TargetRef: prepared.TargetRef, BaseOID: prepared.BaseOID, ObjectFormat: prepared.ObjectFormat,
		AdapterRef: prepared.AdapterRef, EffectIntentRef: claim.Action.EffectIntent.Ref,
		EffectAttemptRef: attempt.Ref, EffectFence: claim.Fence, ReceiptRef: externalReceipt.Ref,
		PreparedAt: prepared.PreparedAt,
	}
	if err := ValidateWorkspaceBinding(binding); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	execution.ExecutionWorkspaceRef = binding.Ref
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return err
	}
	next, err := orchestrator.launchAction(policy, record.Goal, item, execution, authority, now, now)
	if err != nil {
		return err
	}
	return orchestrator.state.RecordWorkspacePrepared(ctx, WorkspacePreparedState{
		Claim: claim, Execution: execution, Binding: binding, NextAction: next, EffectReceipt: externalReceipt,
		Event: EventRecord{Ref: "event:workspace-prepared:" + execution.Ref.String(), Kind: "workspace.prepared",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: now},
		OperationAt: now,
	})
}

func prepareWorkspaceRequest(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	authority WorkItemAuthority, claim ActionClaim, attempt EffectAttempt,
) ports.WorkspacePrepareRequest {
	writeSet := workItemWriteSet(item)
	return ports.WorkspacePrepareRequest{
		WorkspaceRef: execution.ExecutionWorkspaceRef, PrincipalRef: authority.PrincipalRef,
		ActorRef: record.Goal.Actor(), ProjectRef: record.Goal.Project(), RepositoryRef: execution.RepositoryRef,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, AppSpecHash: execution.SpecHash,
		WriteSet: writeSet, WriteSetDigest: ports.WorkspaceWriteSetDigest(writeSet),
		IntentRef: claim.Action.EffectIntent.Ref, AttemptRef: attempt.Ref, ActionFence: claim.Fence,
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey, PreparedAt: claim.Action.EffectIntent.CreatedAt,
	}
}
