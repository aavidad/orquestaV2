package application

import (
	"context"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) processCommitChange(ctx context.Context, claim ActionClaim) error {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	if err := validateClaimedRecord(claim, record, ActionCommitChange); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	if priorEffectAttemptBlocksDispatch(record, claim.Action) {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	if orchestrator.versionControl == nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, "version_control.unavailable")
	}
	if err := validateClaimedEffect(claim, orchestrator.clock.Now()); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, _ := executionForAction(record, claim.Action)
	if len(item.RequiredTests()) > 0 && (orchestrator.testAttestor == nil ||
		ValidateTestAttestationPolicy(orchestrator.testAttestationPolicy) != nil) {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, "test_attestor.unavailable")
	}
	binding, found := workspaceBindingForExecution(record, execution.Ref)
	if !found || binding.Ref != execution.ExecutionWorkspaceRef {
		return orchestrator.quarantine(ctx, claim, "application.workspace_binding_missing")
	}
	if commitChangeTargetDigest(binding, claim.Action.ChangeRef, execution) != claim.Action.EffectIntent.TargetDigest {
		return orchestrator.quarantine(ctx, claim, "application.effect_target_mismatch")
	}
	parentChangeRef, err := parentChangeRef(record, item)
	if err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	authority, authorityFound := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	if len(item.RequiredTests()) > 0 && !authorityFound {
		return orchestrator.quarantine(ctx, claim, "application.work_item_authority_missing")
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return err
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, err.Error())
	}
	attempt, err := orchestrator.beginNewEffectAttempt(ctx, claim, orchestrator.clock.Now())
	if err != nil {
		return err
	}
	return orchestrator.commitChange(ctx, claim, record, item, execution, binding, parentChangeRef, authority, policy, attempt)
}

func (orchestrator *Orchestrator) commitChange(ctx context.Context, claim ActionClaim, record GoalRecord, item goal.WorkItem, execution ExecutionRecord, binding WorkspaceBinding, parentChangeRef ports.ChangeSetRef, authority WorkItemAuthority, policy effectPolicySnapshot, attempt EffectAttempt) error {
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
	effectCtx, cancel := orchestrator.actionCallContext(ctx, claim)
	result, commitErr := orchestrator.versionControl.Commit(effectCtx, request)
	cancel()
	if commitErr != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	if err := ports.ValidateCommitResult(request, result); err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	now := orchestrator.clock.Now().UTC()
	externalReceipt, err := effectReceipt(claim, attempt, result.ReceiptRef, EffectStatusCommitted, unknownUsage(), now)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	change := changeSetFromCommit(binding, result, claim, attempt, externalReceipt)
	if err := ValidateChangeSet(change); err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	execution.State = ExecutionAwaitingAttestation
	var next *ActionRecord
	if len(item.RequiredTests()) > 0 {
		action, actionErr := orchestrator.attestTestAction(
			policy, record.Goal, item, execution, binding, change, authority, now,
		)
		if actionErr != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		next = &action
	}
	err = orchestrator.state.RecordChangeCommitted(ctx, ChangeCommittedState{
		Claim: claim, Execution: execution, ChangeSet: change, NextAction: next, EffectReceipt: externalReceipt,
		Event: EventRecord{Ref: "event:change-committed:" + change.Ref.String(), Kind: "change.committed",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: now},
		OperationAt: now,
	})
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	return nil
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
