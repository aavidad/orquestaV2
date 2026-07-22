package sqlite

import (
	"errors"
	"fmt"
	"slices"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func validateAttestationRecord(a application.AttestationRecord) error {
	if a.Ref.String() == "" || a.GoalRef.String() == "" || a.WorkItemRef.String() == "" || a.ExecutionRef.String() == "" || a.ArtifactRef.String() == "" || !validText(a.PolicyRef) || a.Policy != a.PolicyRef || a.StartedAt.IsZero() || a.FinishedAt.Before(a.StartedAt) || !a.AcceptedAt.Equal(a.FinishedAt) || (a.Kind != application.AttestationKindArtifactProvenance && a.Kind != application.AttestationKindRequiredTests) {
		return errors.New("sqlite.attestation_invalid")
	}
	return nil
}

func validateGoalRecordAttestationsAndReceipts(record application.GoalRecord, aggregate goal.Goal, items map[goal.WorkItemRef]goal.WorkItem, executions map[goal.ExecutionRef]application.ExecutionRecord, artifacts map[goal.ArtifactRef]application.ArtifactRecord) error {
	attestations := make(map[goal.AttestationRef]application.AttestationRecord, len(record.Attestations))
	for _, attestation := range record.Attestations {
		if err := validateGoalRecordAttestation(record, aggregate, items, executions, artifacts, attestations, attestation); err != nil {
			return err
		}
	}
	if err := validateGoalRecordItemReferences(items, artifacts, attestations); err != nil {
		return err
	}
	return validateGoalRecordConsumptionReceipts(record, aggregate, items, executions)
}

func validateTypedAttestationScope(record application.GoalRecord, item goal.WorkItem, execution application.ExecutionRecord, attestation application.AttestationRecord) error {
	if attestation.Kind == application.AttestationKindArtifactProvenance {
		return nil
	}
	changeIndex := slices.IndexFunc(record.ChangeSets, func(change application.ChangeSet) bool {
		return change.Ref == attestation.ChangeSetRef && change.GoalRef == attestation.GoalRef && change.WorkItemRef == attestation.WorkItemRef && change.ExecutionRef == attestation.ExecutionRef && change.ExecutionAttempt == attestation.ExecutionAttempt && change.PlanGeneration == attestation.PlanGeneration && change.AppSpecGeneration == attestation.AppSpecGeneration && change.SpecHash == attestation.SpecHash && change.Digest() == attestation.ChangeSetDigest
	})
	if changeIndex < 0 {
		return errors.New("sqlite.goal_record_test_attestation_evidence_invalid")
	}
	change := record.ChangeSets[changeIndex]
	bindingIndex := slices.IndexFunc(record.WorkspaceBindings, func(binding application.WorkspaceBinding) bool {
		return binding.Ref == change.WorkspaceRef && binding.GoalRef == attestation.GoalRef && binding.WorkItemRef == attestation.WorkItemRef && binding.ExecutionRef == attestation.ExecutionRef
	})
	passed := attestation.Verdict == application.AttestationVerdictPassed
	if bindingIndex < 0 || !application.RequiredTestsAttestationEvidenceMatches(record, item, execution, persistedTestSubject(item, attestation, change, record.WorkspaceBindings[bindingIndex]), attestation, passed) || !application.TestAttestationConsumptionMatches(record, attestation) {
		return errors.New("sqlite.goal_record_test_attestation_evidence_invalid")
	}
	if !passed && (execution.State != application.ExecutionFailed || execution.FailureCode != "test_attestor.required_tests_failed") {
		return errors.New("sqlite.goal_record_test_attestation_execution_invalid")
	}
	return nil
}
func persistedTestSubject(item goal.WorkItem, a application.AttestationRecord, change application.ChangeSet, binding application.WorkspaceBinding) ports.TestSubject {
	return ports.TestSubject{
		GoalRef: a.GoalRef, WorkItemRef: a.WorkItemRef, ExecutionRef: a.ExecutionRef, ExecutionAttempt: a.ExecutionAttempt,
		PlanGeneration: a.PlanGeneration, WorkItemGeneration: a.WorkItemGeneration, AppSpecGeneration: a.AppSpecGeneration, AppSpecHash: a.SpecHash,
		WorkspaceRef: binding.Ref, WorkspaceBindingDigest: binding.Digest(), ChangeSetRef: change.Ref, ChangeSetDigest: change.Digest(),
		RepositoryRef: change.RepositoryRef, ObjectFormat: change.ObjectFormat, BaseOID: change.BaseOID, ParentOID: change.ParentOID,
		HeadOID: change.HeadOID, TreeOID: change.TreeOID, DiffDigest: change.DiffDigest, WriteSetDigest: change.WriteSetDigest,
		RequiredTestsDigest: item.RequiredTestsDigest(), PolicyRef: a.PolicyRef, PolicyDigest: a.PolicyDigest,
	}
}

func validateGoalRecordAttestation(record application.GoalRecord, aggregate goal.Goal, items map[goal.WorkItemRef]goal.WorkItem, executions map[goal.ExecutionRef]application.ExecutionRecord, artifacts map[goal.ArtifactRef]application.ArtifactRecord, attestations map[goal.AttestationRef]application.AttestationRecord, a application.AttestationRecord) error {
	if err := validateAttestationRecord(a); err != nil {
		return err
	}
	item, itemFound := items[a.WorkItemRef]
	execution, executionFound := executions[a.ExecutionRef]
	_, artifactFound := artifacts[a.ArtifactRef]
	bound, boundFound := item.Execution()
	legacy := a.Kind == application.AttestationKindArtifactProvenance && a.ExecutionAttempt == 0
	artifactOccurrenceFound := slices.ContainsFunc(record.Artifacts, func(artifact application.ArtifactRecord) bool {
		return artifact.Stored.Ref == a.ArtifactRef && artifact.GoalRef == a.GoalRef && artifact.WorkItemRef == a.WorkItemRef && artifact.ExecutionRef == a.ExecutionRef
	})
	if !itemFound || !executionFound || !artifactFound || (!legacy && !artifactOccurrenceFound) || !boundFound || a.GoalRef != aggregate.Ref() || bound != a.ExecutionRef || execution.WorkItemRef != a.WorkItemRef || (!legacy && (execution.AttemptNo != a.ExecutionAttempt || execution.PlanGeneration != a.PlanGeneration || execution.AppSpecGeneration != a.AppSpecGeneration || execution.SpecHash != a.SpecHash || a.WorkItemGeneration > item.Revision())) {
		return errors.New("sqlite.goal_record_attestation_scope_invalid")
	}
	if err := validateTypedAttestationScope(record, item, execution, a); err != nil {
		return err
	}
	if !slices.Contains(item.Attestations(), a.Ref) {
		staged, ok := workItemStagedOutputExecution(item, executions, record)
		if !ok || staged.Ref != a.ExecutionRef {
			return errors.New("sqlite.goal_record_attestation_scope_invalid")
		}
	}
	if _, duplicate := attestations[a.Ref]; duplicate {
		return errors.New("sqlite.goal_record_attestation_duplicate")
	}
	attestations[a.Ref] = a
	return nil
}

func validateGoalRecordItemReferences(items map[goal.WorkItemRef]goal.WorkItem, artifacts map[goal.ArtifactRef]application.ArtifactRecord, attestations map[goal.AttestationRef]application.AttestationRecord) error {
	for _, item := range items {
		for _, ref := range item.Artifacts() {
			if artifact, found := artifacts[ref]; !found || artifact.WorkItemRef != item.Ref() {
				return errors.New("sqlite.goal_record_item_artifact_invalid")
			}
		}
		for _, ref := range item.Attestations() {
			if attestation, found := attestations[ref]; !found || attestation.WorkItemRef != item.Ref() {
				return errors.New("sqlite.goal_record_item_attestation_invalid")
			}
		}
	}
	return nil
}

func validateGoalRecordConsumptionReceipts(record application.GoalRecord, aggregate goal.Goal, items map[goal.WorkItemRef]goal.WorkItem, executions map[goal.ExecutionRef]application.ExecutionRecord) error {
	seen := make(map[string]struct{}, 3*len(record.ConsumptionReceipts))
	for _, receipt := range record.ConsumptionReceipts {
		item, itemFound := items[receipt.WorkItemRef]
		execution, executionFound := executions[receipt.ExecutionRef]
		if validateConsumptionReceipt(receipt) != nil || !itemFound || !executionFound || receipt.GoalRef != aggregate.Ref() || execution.WorkItemRef != receipt.WorkItemRef || receipt.PlanGeneration != execution.PlanGeneration || receipt.WorkItemGeneration > item.Revision() {
			return errors.New("sqlite.goal_record_receipt_scope_invalid")
		}
		scope := receipt.WorkItemRef.String()
		if receipt.Kind == application.ActionDeliverMailbox {
			scope = receipt.MailboxMessageRef.String()
		}
		keys := []string{"action:" + receipt.ActionRef, "token:" + receipt.ClaimToken, "fence:" + string(receipt.Kind) + ":" + scope + ":" + fmt.Sprint(receipt.Fence)}
		for _, key := range keys {
			if _, duplicate := seen[key]; duplicate {
				return errors.New("sqlite.goal_record_receipt_duplicate")
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}
