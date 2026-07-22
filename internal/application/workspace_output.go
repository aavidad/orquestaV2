package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) stageExecutionOutput(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	execution ExecutionRecord,
	observation ports.AgentObservation,
	transitionAt time.Time,
) error {
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found || execution.ExecutionWorkspaceRef.String() == "" {
		return &StateError{Code: StateConflict}
	}
	binding, found := workspaceBindingForExecution(record, execution.Ref)
	if !found {
		return &StateError{Code: StateConflict}
	}
	putRequest := ports.PutArtifactRequest{MediaType: observation.MediaType, Content: observation.Content}
	stored, err := orchestrator.artifacts.Put(ctx, putRequest)
	if err != nil {
		return orchestrator.requeue(ctx, claim, execution, "artifact.store_failed")
	}
	if err := ports.ValidateStoredArtifact(putRequest, stored); err != nil {
		return orchestrator.failGoalAt(ctx, claim, record, ports.ArtifactContractErrorCode(err), transitionAt)
	}
	attestationRef, err := goal.NewAttestationRef("attestation:execution:" + execution.Ref.String())
	if err != nil {
		return err
	}
	authority, found := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	if !found {
		return errors.New("application.work_item_authority_missing")
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return err
	}
	changeRef, err := newChangeSetRef(ctx, orchestrator.ids)
	if err != nil {
		return err
	}
	execution.State = ExecutionAwaitingCommit
	execution.LastObservedAt = transitionAt
	execution.ProviderObservedAt = observation.ObservedAt.UTC()
	settlement, err := settlementFor(record, execution, observation.Usage, int64(len(observation.Content)), transitionAt)
	if err != nil {
		return err
	}
	next, err := orchestrator.commitChangeAction(policy, record.Goal, item, execution, binding, changeRef, authority, transitionAt)
	if err != nil {
		return err
	}
	return orchestrator.state.RecordExecutionOutputReady(ctx, ExecutionOutputReadyState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Execution:   execution,
		Artifact:    artifactProvenanceRecord(stored, record.Goal, item, execution, transitionAt),
		Attestation: artifactProvenanceAttestation(attestationRef, stored.Ref, record.Goal, item, execution, transitionAt),
		NextAction:  next, BudgetSettlement: settlement,
		Event: EventRecord{Ref: "event:execution-output-ready:" + execution.Ref.String(), Kind: "execution.output_ready",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: transitionAt},
		OperationAt: transitionAt,
	})
}
