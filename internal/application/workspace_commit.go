package application

import (
	"context"

	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) processCommitChange(ctx context.Context, claim ActionClaim) error {
	if orchestrator.versionControl == nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, "version_control.unavailable")
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	if err := validateClaimedRecord(claim, record, ActionCommitChange); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	if err := validateClaimedEffect(claim, orchestrator.clock.Now()); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, _ := executionForAction(record, claim.Action)
	binding, found := workspaceBindingForExecution(record, execution.Ref)
	if !found || binding.Ref != execution.ExecutionWorkspaceRef {
		return orchestrator.quarantine(ctx, claim, "application.workspace_binding_missing")
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, err.Error())
	}
	attempt, err := orchestrator.beginEffectAttempt(ctx, claim, orchestrator.clock.Now())
	if err != nil {
		return err
	}
	parentChangeRef, err := parentChangeRef(record, item)
	if err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	request := ports.CommitRequest{
		ChangeSetRef: claim.Action.ChangeRef, WorkspaceRef: binding.Ref,
		PrincipalRef: binding.PrincipalRef, ActorRef: binding.ActorRef,
		ProjectRef: binding.ProjectRef, RepositoryRef: binding.RepositoryRef,
		GoalRef: binding.GoalRef, WorkItemRef: binding.WorkItemRef, ExecutionRef: binding.ExecutionRef,
		ExecutionAttempt: binding.ExecutionAttempt, PlanGeneration: binding.PlanGeneration,
		AppSpecGeneration: binding.AppSpecGeneration, AppSpecHash: binding.SpecHash,
		BaseOID: binding.BaseOID, ObjectFormat: binding.ObjectFormat,
		WriteSet: binding.WriteSet, WriteSetDigest: binding.WriteSetDigest, ParentChangeRef: parentChangeRef,
		IntentRef: claim.Action.EffectIntent.Ref, AttemptRef: attempt.Ref, ActionFence: claim.Fence,
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey, CommittedAt: claim.Action.EffectIntent.CreatedAt,
	}
	if commitChangeTargetDigest(binding, request.ChangeSetRef, execution) != claim.Action.EffectIntent.TargetDigest {
		return orchestrator.quarantine(ctx, claim, "application.effect_target_mismatch")
	}
	result, commitErr := orchestrator.versionControl.Commit(ctx, request)
	if commitErr != nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, versionControlErrorCode(commitErr, "version_control.commit_failed"))
	}
	if err := ports.ValidateCommitResult(request, result); err != nil {
		return orchestrator.quarantine(ctx, claim, ports.VersionControlContractErrorCode(err))
	}
	now := orchestrator.clock.Now().UTC()
	externalReceipt, err := effectReceipt(claim, attempt, result.ReceiptRef, EffectStatusCommitted, unknownUsage(), now)
	if err != nil {
		return err
	}
	change := changeSetFromCommit(binding, result, claim, attempt, externalReceipt)
	if err := ValidateChangeSet(change); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	execution.State = ExecutionAwaitingIntegration
	return orchestrator.state.RecordChangeCommitted(ctx, ChangeCommittedState{
		Claim: claim, Execution: execution, ChangeSet: change, EffectReceipt: externalReceipt,
		Event: EventRecord{Ref: "event:change-committed:" + change.Ref.String(), Kind: "change.committed",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: now},
		OperationAt: now,
	})
}

func changeSetFromCommit(binding WorkspaceBinding, result ports.CommitResult, claim ActionClaim,
	attempt EffectAttempt, receipt EffectReceipt,
) ChangeSet {
	return ChangeSet{
		Ref: result.ChangeSetRef, WorkspaceRef: result.WorkspaceRef,
		PrincipalRef: binding.PrincipalRef, ActorRef: binding.ActorRef,
		ProjectRef: binding.ProjectRef, RepositoryRef: result.RepositoryRef,
		GoalRef: binding.GoalRef, WorkItemRef: binding.WorkItemRef, ExecutionRef: result.ExecutionRef,
		ExecutionAttempt: binding.ExecutionAttempt, PlanGeneration: binding.PlanGeneration,
		AppSpecGeneration: binding.AppSpecGeneration, SpecHash: binding.SpecHash,
		BaseOID: result.BaseOID, ParentOID: result.ParentOID, HeadOID: result.HeadOID, TreeOID: result.TreeOID,
		ObjectFormat: result.ObjectFormat, DiffDigest: result.DiffDigest,
		ChangedPaths: result.ChangedPaths, WriteSet: binding.WriteSet, WriteSetDigest: result.WriteSetDigest,
		ParentChangeRef: result.ParentChangeRef, EffectIntentRef: claim.Action.EffectIntent.Ref,
		EffectAttemptRef: attempt.Ref, EffectFence: claim.Fence,
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey,
		AdapterRef:     result.AdapterRef, ReceiptRef: receipt.Ref, CommittedAt: result.CommittedAt,
	}
}
