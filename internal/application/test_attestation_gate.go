package application

import (
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func requiredTestsPassForChange(
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	change ChangeSet,
	policy TestAttestationPolicy,
) (AttestationRecord, bool) {
	if len(item.RequiredTests()) == 0 {
		return AttestationRecord{}, false
	}
	binding, found := workspaceBindingForExecution(record, execution.Ref)
	if !found {
		return AttestationRecord{}, false
	}
	subject, err := buildTestSubject(record.Goal, item, execution, binding, change, policy)
	if err != nil {
		return AttestationRecord{}, false
	}
	historicalGeneration, historical := historicalIntegratedWorkItemGeneration(record, change)
	var matched AttestationRecord
	count := 0
	for _, attestation := range record.Attestations {
		if attestation.Kind != AttestationKindRequiredTests || attestation.ChangeSetRef != change.Ref ||
			attestation.ExecutionRef != execution.Ref {
			continue
		}
		candidateSubject := subject
		if historical {
			candidateSubject.WorkItemGeneration = historicalGeneration
		}
		if RequiredTestsAttestationMatches(record, item, execution, candidateSubject, attestation) {
			matched, count = attestation, count+1
		}
	}
	return matched, count == 1
}

func anyRequiredTestsPassForChange(
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	change ChangeSet,
) bool {
	for _, attestation := range record.Attestations {
		if attestation.Kind != AttestationKindRequiredTests || attestation.Verdict != AttestationVerdictPassed {
			continue
		}
		policy := TestAttestationPolicy{Ref: attestation.PolicyRef, Digest: attestation.PolicyDigest}
		if _, passed := requiredTestsPassForChange(record, item, execution, change, policy); passed {
			return true
		}
	}
	return false
}

func historicalIntegratedWorkItemGeneration(record GoalRecord, change ChangeSet) (goal.Revision, bool) {
	var generation goal.Revision
	count := 0
	for _, integration := range record.IntegrationReceipts {
		if integration.ChangeRef != change.Ref || integration.Status != ports.IntegrationStatusIntegrated {
			continue
		}
		intent, found := testEffectIntentByRef(record.EffectIntents, integration.EffectIntentRef)
		if !found || intent.ActionKind != ActionIntegrateChange {
			continue
		}
		for _, receipt := range record.ConsumptionReceipts {
			if receipt.Kind != ActionIntegrateChange || receipt.ActionRef != intent.ActionRef ||
				receipt.GoalRef != change.GoalRef || receipt.WorkItemRef != change.WorkItemRef ||
				receipt.ExecutionRef != change.ExecutionRef || receipt.ChangeRef != change.Ref ||
				receipt.WorkItemGeneration == 0 || receipt.Outcome != ActionConsumedCompleted ||
				receipt.EffectReceiptRef != integration.EffectReceiptRef {
				continue
			}
			generation, count = receipt.WorkItemGeneration, count+1
		}
	}
	return generation, count == 1
}

func RequiredTestsAttestationMatches(
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	subject ports.TestSubject,
	attestation AttestationRecord,
) bool {
	return RequiredTestsAttestationEvidenceMatches(record, item, execution, subject, attestation, true) &&
		TestAttestationConsumptionMatches(record, attestation)
}

func RequiredTestsAttestationEvidenceMatches(
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	subject ports.TestSubject,
	attestation AttestationRecord,
	passed bool,
) bool {
	wantVerdict, wantStatus := AttestationVerdictFailed, EffectStatusAttestedFailed
	portVerdict := ports.TestAttestationFailed
	if passed {
		wantVerdict, wantStatus = AttestationVerdictPassed, EffectStatusAttestedPassed
		portVerdict = ports.TestAttestationPassed
	}
	if attestation.Kind != AttestationKindRequiredTests || attestation.Verdict != wantVerdict ||
		attestation.GoalRef != subject.GoalRef || attestation.WorkItemRef != subject.WorkItemRef ||
		attestation.ExecutionRef != subject.ExecutionRef || attestation.ExecutionAttempt != subject.ExecutionAttempt ||
		attestation.PlanGeneration != subject.PlanGeneration ||
		attestation.WorkItemGeneration != subject.WorkItemGeneration ||
		attestation.AppSpecGeneration != subject.AppSpecGeneration || attestation.SpecHash != subject.AppSpecHash ||
		attestation.SubjectDigest != ports.TestSubjectDigest(subject) ||
		attestation.WorkspaceBindingDigest != subject.WorkspaceBindingDigest ||
		attestation.ChangeSetRef != subject.ChangeSetRef || attestation.ChangeSetDigest != subject.ChangeSetDigest ||
		attestation.RequiredTestsDigest != subject.RequiredTestsDigest || attestation.PolicyRef != subject.PolicyRef ||
		attestation.PolicyDigest != subject.PolicyDigest || attestation.ManifestArtifactRef.String() == "" ||
		attestation.ReportArtifactRef.String() == "" || attestation.ArtifactRef != attestation.ReportArtifactRef ||
		!validApplicationRef(attestation.AttestorRef) || !validApplicationRef(attestation.ReceiptRef) ||
		attestation.StartedAt.IsZero() || attestation.FinishedAt.Before(attestation.StartedAt) ||
		!validApplicationRef(attestation.EffectIntentRef) || !validApplicationRef(attestation.EffectAttemptRef) ||
		attestation.EffectFence == 0 || !validApplicationRef(attestation.EffectReceiptRef) {
		return false
	}
	if ports.ValidateRequiredTestOutcomes(item.RequiredTests(), portVerdict, attestation.Tests) != nil ||
		!testArtifactOccurrenceMatches(record, attestation.ManifestArtifactRef, ArtifactKindTestSubjectManifest,
			execution, attestation.WorkItemGeneration) ||
		!testArtifactOccurrenceMatches(record, attestation.ReportArtifactRef, ArtifactKindTestReport,
			execution, attestation.WorkItemGeneration) {
		return false
	}
	intent, intentFound := testEffectIntentByRef(record.EffectIntents, attestation.EffectIntentRef)
	attempt, attemptFound := testEffectAttemptByRef(record.EffectAttempts, attestation.EffectAttemptRef)
	receipt, receiptFound := testEffectReceiptByRef(record.EffectReceipts, attestation.EffectReceiptRef)
	return intentFound && attemptFound && receiptFound && intent.Kind == EffectKindAttestTest &&
		intent.ActionKind == ActionAttestTest && intent.TargetDigest == attestation.SubjectDigest &&
		intent.Subject.ExecutionRef == execution.Ref && attempt.IntentRef == intent.Ref &&
		attempt.ActionFence == attestation.EffectFence && receipt.IntentRef == intent.Ref &&
		receipt.AttemptRef == attempt.Ref && receipt.ActionFence == attestation.EffectFence &&
		receipt.Status == wantStatus && receipt.ExternalRef == attestation.ReceiptRef
}

func testArtifactOccurrenceMatches(
	record GoalRecord,
	ref goal.ArtifactRef,
	kind ArtifactKind,
	execution ExecutionRecord,
	workItemGeneration goal.Revision,
) bool {
	count := 0
	for _, artifact := range record.Artifacts {
		if artifact.Stored.Ref != ref || artifact.Kind != kind || artifact.ExecutionRef != execution.Ref {
			continue
		}
		wantMediaType := ports.TestAttestationReportMediaType
		if kind == ArtifactKindTestSubjectManifest {
			wantMediaType = ports.TestSubjectManifestMediaType
		}
		if !validApplicationRef(artifact.OccurrenceRef) || artifact.GoalRef != execution.GoalRef ||
			artifact.WorkItemRef != execution.WorkItemRef || artifact.ExecutionAttempt != execution.AttemptNo ||
			artifact.PlanGeneration != execution.PlanGeneration || artifact.WorkItemGeneration != workItemGeneration ||
			artifact.AppSpecGeneration != execution.AppSpecGeneration || artifact.SpecHash != execution.SpecHash ||
			artifact.CreatedAt.IsZero() || !validEffectDigest(artifact.Stored.Digest) ||
			artifact.Stored.Size <= 0 || artifact.Stored.MediaType != wantMediaType {
			return false
		}
		count++
	}
	return count == 1
}

func TestAttestationConsumptionMatches(record GoalRecord, attestation AttestationRecord) bool {
	intent, found := testEffectIntentByRef(record.EffectIntents, attestation.EffectIntentRef)
	if !found {
		return false
	}
	count := 0
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.Kind == ActionAttestTest && receipt.ActionRef == intent.ActionRef &&
			receipt.GoalRef == attestation.GoalRef && receipt.WorkItemRef == attestation.WorkItemRef &&
			receipt.ExecutionRef == attestation.ExecutionRef && receipt.ChangeRef == attestation.ChangeSetRef &&
			receipt.WorkItemGeneration == attestation.WorkItemGeneration && receipt.Fence == attestation.EffectFence &&
			receipt.Outcome == ActionConsumedCompleted && receipt.EffectReceiptRef == attestation.EffectReceiptRef {
			count++
		}
	}
	return count == 1
}

func testEffectIntentByRef(records []EffectIntent, ref string) (EffectIntent, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return EffectIntent{}, false
}

func testEffectAttemptByRef(records []EffectAttempt, ref string) (EffectAttempt, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return EffectAttempt{}, false
}

func testEffectReceiptByRef(records []EffectReceipt, ref string) (EffectReceipt, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return EffectReceipt{}, false
}
