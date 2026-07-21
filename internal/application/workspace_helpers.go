package application

import (
	"context"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func workspaceBindingForExecution(record GoalRecord, executionRef goal.ExecutionRef) (WorkspaceBinding, bool) {
	for _, binding := range record.WorkspaceBindings {
		if binding.ExecutionRef == executionRef {
			return binding, true
		}
	}
	return WorkspaceBinding{}, false
}

func changeSetByRef(record GoalRecord, ref ports.ChangeSetRef) (ChangeSet, bool) {
	for _, change := range record.ChangeSets {
		if change.Ref == ref {
			return change, true
		}
	}
	return ChangeSet{}, false
}

func parentChangeRef(record GoalRecord, item goal.WorkItem) (ports.ChangeSetRef, error) {
	parentItem, rework := item.ReworkOf()
	if !rework {
		return ports.ChangeSetRef{}, nil
	}
	for index := len(record.ChangeSets) - 1; index >= 0; index-- {
		if record.ChangeSets[index].WorkItemRef == parentItem {
			return record.ChangeSets[index].Ref, nil
		}
	}
	return ports.ChangeSetRef{}, errors.New("application.parent_change_ref_missing")
}

func workspaceTargetRef(record GoalRecord, change ChangeSet) string {
	for _, binding := range record.WorkspaceBindings {
		if binding.Ref == change.WorkspaceRef {
			return binding.TargetRef
		}
	}
	return ""
}

func executionEvidence(record GoalRecord, executionRef goal.ExecutionRef) (ArtifactRecord, AttestationRecord, bool) {
	for _, attestation := range record.Attestations {
		if attestation.ExecutionRef != executionRef {
			continue
		}
		for _, artifact := range record.Artifacts {
			if artifact.Stored.Ref == attestation.ArtifactRef {
				return artifact, attestation, true
			}
		}
	}
	return ArtifactRecord{}, AttestationRecord{}, false
}

func (orchestrator *Orchestrator) requeueWorkspaceEffect(ctx context.Context, claim ActionClaim, code string) error {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	execution, found := executionForAction(record, claim.Action)
	if !found {
		return orchestrator.quarantine(ctx, claim, "application.execution_not_found")
	}
	return orchestrator.requeue(ctx, claim, execution, code)
}

func workspaceErrorCode(err error, fallback string) string {
	if code := ports.WorkspaceContractErrorCode(err); code != "" {
		return code
	}
	return fallback
}

func versionControlErrorCode(err error, fallback string) string {
	if code := ports.VersionControlContractErrorCode(err); code != "" {
		return code
	}
	return fallback
}
