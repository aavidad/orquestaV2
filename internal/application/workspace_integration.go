package application

import (
	"context"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) processIntegrateChange(ctx context.Context, claim ActionClaim) error {
	if orchestrator.versionControl == nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, "version_control.unavailable")
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	if err := validateClaimedRecord(claim, record, ActionIntegrateChange); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	if err := validateClaimedEffect(claim, orchestrator.clock.Now()); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, _ := executionForAction(record, claim.Action)
	change, found := changeSetByRef(record, claim.Action.ChangeRef)
	if !found || integrationTargetDigest(change, claim.Action.ExpectedTargetOID) != claim.Action.EffectIntent.TargetDigest {
		return orchestrator.quarantine(ctx, claim, "application.effect_target_mismatch")
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, err.Error())
	}
	attempt, err := orchestrator.beginEffectAttempt(ctx, claim, orchestrator.clock.Now())
	if err != nil {
		return err
	}
	result, err := orchestrator.executeIntegration(ctx, claim, record, change, attempt)
	if err != nil {
		return err
	}
	now := orchestrator.clock.Now().UTC()
	effectStatus, observationStatus, candidateTreeOID, conflictDigest := integrationFactStatus(result)
	externalReceipt, err := effectReceipt(claim, attempt, result.ReceiptRef, effectStatus, unknownUsage(), now)
	if err != nil {
		return err
	}
	observation, integration := integrationFacts(
		claim, attempt, result, externalReceipt, observationStatus, candidateTreeOID, conflictDigest,
	)
	if ValidateMergeObservation(observation) != nil || ValidateIntegrationReceipt(integration) != nil {
		return orchestrator.quarantine(ctx, claim, "application.integration_fact_invalid")
	}
	transition, err := orchestrator.advanceIntegration(ctx, claim, record, item, execution, change, result.Status, now)
	if err != nil {
		return err
	}
	return orchestrator.state.RecordIntegrationResult(ctx, IntegrationResultState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: transition.goal, Execution: transition.execution, Observation: observation, Integration: integration,
		EffectReceipt: externalReceipt, NewExecutions: transition.newExecutions, NewActions: transition.newActions,
		Events: transition.events, OperationAt: now,
	})
}

func (orchestrator *Orchestrator) executeIntegration(ctx context.Context, claim ActionClaim,
	record GoalRecord, change ChangeSet, attempt EffectAttempt,
) (ports.IntegrationResult, error) {
	previewRequest := ports.IntegrationPreviewRequest{
		ChangeSetRef: change.Ref, RepositoryRef: change.RepositoryRef, SourceOID: change.HeadOID,
		TargetRef: workspaceTargetRef(record, change), TargetOID: claim.Action.ExpectedTargetOID,
		ObjectFormat: change.ObjectFormat, IdempotencyKey: "preview:" + claim.Action.EffectIntent.IdempotencyKey,
		RequestedAt: claim.Action.EffectIntent.CreatedAt,
	}
	preview, err := orchestrator.versionControl.PreviewIntegration(ctx, previewRequest)
	if err != nil {
		return ports.IntegrationResult{}, orchestrator.requeueWorkspaceEffect(
			ctx, claim, versionControlErrorCode(err, "version_control.preview_failed"),
		)
	}
	if err := ports.ValidateIntegrationPreview(previewRequest, preview); err != nil {
		return ports.IntegrationResult{}, orchestrator.quarantine(ctx, claim, ports.VersionControlContractErrorCode(err))
	}
	request := ports.IntegrationRequest{
		ChangeSetRef: change.Ref, RepositoryRef: change.RepositoryRef,
		PrincipalRef: claim.Action.EffectIntent.ProposedBy, ProjectRef: change.ProjectRef,
		SourceOID: change.HeadOID, TargetRef: preview.TargetRef, ExpectedTargetOID: claim.Action.ExpectedTargetOID,
		ObjectFormat: change.ObjectFormat, IntentRef: claim.Action.EffectIntent.Ref, AttemptRef: attempt.Ref,
		ActionFence: claim.Fence, IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey,
		RequestedAt: claim.Action.EffectIntent.CreatedAt,
	}
	result, err := orchestrator.versionControl.Integrate(ctx, request)
	if err != nil {
		return ports.IntegrationResult{}, orchestrator.requeueWorkspaceEffect(
			ctx, claim, versionControlErrorCode(err, "version_control.integrate_failed"),
		)
	}
	if err := ports.ValidateIntegrationResult(request, result); err != nil {
		return ports.IntegrationResult{}, orchestrator.quarantine(ctx, claim, ports.VersionControlContractErrorCode(err))
	}
	return result, nil
}

func integrationFactStatus(result ports.IntegrationResult) (EffectStatus, ports.MergeStatus, string, string) {
	switch result.Status {
	case ports.IntegrationStatusIntegrated:
		return EffectStatusIntegrated, ports.MergeStatusClean, result.TreeOID, ""
	case ports.IntegrationStatusConflicted:
		return EffectStatusConflicted, ports.MergeStatusConflicted, "", result.ConflictDigest
	default:
		return EffectStatusStale, ports.MergeStatusStale, "", result.ConflictDigest
	}
}

func integrationFacts(claim ActionClaim, attempt EffectAttempt, result ports.IntegrationResult,
	externalReceipt EffectReceipt, status ports.MergeStatus, candidateTreeOID, conflictDigest string,
) (MergeObservation, IntegrationReceipt) {
	observation := MergeObservation{
		Ref:       "merge-observation:" + claim.Action.Ref + ":" + strconv.FormatUint(claim.Fence, 10),
		ChangeRef: result.ChangeSetRef, RepositoryRef: result.RepositoryRef, SourceOID: result.SourceOID,
		TargetRef: result.TargetRef, TargetOID: result.TargetBeforeOID, ObjectFormat: result.ObjectFormat,
		Status: status, CandidateTreeOID: candidateTreeOID, ConflictDigest: conflictDigest,
		AdapterRef: result.AdapterRef, ObservedAt: result.RecordedAt,
	}
	receipt := IntegrationReceipt{
		Ref: "integration-receipt:" + claim.Action.Ref, ChangeRef: result.ChangeSetRef,
		RepositoryRef: result.RepositoryRef, SourceOID: result.SourceOID, TargetRef: result.TargetRef,
		TargetBeforeOID: result.TargetBeforeOID, TargetAfterOID: result.TargetAfterOID,
		TreeOID: result.TreeOID, ObjectFormat: result.ObjectFormat, Status: result.Status,
		MarkerRef: result.MarkerRef, ConflictDigest: result.ConflictDigest,
		EffectIntentRef: claim.Action.EffectIntent.Ref, EffectAttemptRef: attempt.Ref,
		EffectFence: claim.Fence, EffectReceiptRef: externalReceipt.Ref,
		AdapterRef: result.AdapterRef, RecordedAt: result.RecordedAt,
	}
	return observation, receipt
}

type integrationTransition struct {
	goal          goal.Goal
	execution     ExecutionRecord
	newExecutions []ExecutionRecord
	newActions    []ActionRecord
	events        []EventRecord
}

func (orchestrator *Orchestrator) advanceIntegration(ctx context.Context, claim ActionClaim,
	record GoalRecord, item goal.WorkItem, execution ExecutionRecord, change ChangeSet,
	status ports.IntegrationStatus, at time.Time,
) (integrationTransition, error) {
	if status != ports.IntegrationStatusIntegrated {
		aggregate, err := record.Goal.InterruptWorkItem(
			record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref,
			goal.WorkItemInterruptExecutionFailed, at,
		)
		if err != nil {
			return integrationTransition{}, err
		}
		execution.State, execution.FinishedAt = ExecutionFailed, at
		execution.FailureCode = "version_control.integration_" + string(status)
		events := []EventRecord{{Ref: "event:change-pending:" + change.Ref.String() + ":" + string(status),
			Kind: "change." + string(status), GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
			ExecutionRef: execution.Ref, OccurredAt: at}}
		return integrationTransition{goal: aggregate, execution: execution, events: events}, nil
	}
	if !record.Goal.ChildHandoffsResolved(item.Ref()) {
		return integrationTransition{}, orchestrator.requeueWorkspaceEffect(ctx, claim, "application.child_handoffs_pending")
	}
	artifact, attestation, found := executionEvidence(record, execution.Ref)
	if !found {
		return integrationTransition{}, &StateError{Code: StateConflict}
	}
	aggregate, err := record.Goal.SucceedWorkItem(record.Goal.Revision(), item.Revision(), item.Ref(),
		[]goal.ArtifactRef{artifact.Stored.Ref}, []goal.AttestationRef{attestation.Ref}, at)
	if err != nil {
		return integrationTransition{}, err
	}
	if outcome, closable := aggregate.ClosableOutcome(); closable {
		aggregate, err = aggregate.Close(aggregate.Revision(), outcome, at)
		if err != nil {
			return integrationTransition{}, err
		}
	}
	execution.State, execution.FinishedAt = ExecutionSucceeded, at
	executions, actions, events, err := orchestrator.scheduleHistoricalReady(
		ctx, record, aggregate, replaceExecution(record.Executions, execution), at,
	)
	if err != nil {
		return integrationTransition{}, err
	}
	events = append([]EventRecord{{Ref: "event:change-integrated:" + change.Ref.String(), Kind: "change.integrated",
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at}}, events...)
	return integrationTransition{
		goal: aggregate, execution: execution, newExecutions: executions, newActions: actions, events: events,
	}, nil
}
