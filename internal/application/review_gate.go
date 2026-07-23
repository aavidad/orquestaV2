package application

import (
	"errors"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/review"
)

// ValidatePersistedCouncilSubject proves a durable Council subject still
// derives from one exact V18 PASS subject/gate and immutable WorkItem policy.
func ValidatePersistedCouncilSubject(record GoalRecord, subject council.Subject) error {
	if _, err := council.NewSubject(subject); err != nil {
		return errors.New("council.persisted_subject_invalid")
	}
	item, found := persistedCouncilWorkItem(record, subject)
	if !found || record.Goal.Ref().String() != subject.GoalRef ||
		record.Goal.Project().String() != subject.ProjectRef ||
		record.Goal.AppSpec().Generation() != goal.AppSpecGeneration(subject.AppSpecGeneration) {
		return errors.New("council.persisted_subject_invalid")
	}
	author, found := persistedCouncilAuthor(record, subject)
	if !found {
		return errors.New("council.persisted_subject_invalid")
	}
	change, found := persistedCouncilChange(record, subject)
	policy, policyFound := item.CouncilPolicy()
	if !found || !policyFound || policy != subject.Policy ||
		!persistedCouncilGenerationMatches(item, subject) || change.ExecutionRef != author.Ref {
		return errors.New("council.persisted_subject_invalid")
	}
	matches := 0
	for _, attestation := range record.Attestations {
		if attestation.Kind != AttestationKindRequiredTests || attestation.Verdict != AttestationVerdictPassed || attestation.ExecutionRef != author.Ref || attestation.ChangeSetRef != change.Ref {
			continue
		}
		reviewSubject, gate, err := persistedCouncilReviewGate(record, item, author, change, attestation)
		if err != nil {
			continue
		}
		candidate, err := councilSubjectFromReviewGate(record, item, author, change, policy, reviewSubject, gate)
		if err == nil && candidate == subject {
			matches++
		}
	}
	if matches != 1 {
		return errors.New("council.persisted_subject_invalid")
	}
	return nil
}

func persistedCouncilGenerationMatches(item goal.WorkItem, subject council.Subject) bool {
	generation := goal.Revision(subject.WorkItemGeneration)
	if item.Revision() == generation {
		return item.State() == goal.WorkItemStateRunning
	}
	return item.Revision() == generation+1 &&
		(item.State() == goal.WorkItemStateSucceeded || item.State() == goal.WorkItemStateSuperseded)
}

func persistedCouncilReviewGate(record GoalRecord, item goal.WorkItem, author ExecutionRecord,
	change ChangeSet, attestation AttestationRecord,
) (review.Subject, review.Gate, error) {
	binding, found := workspaceBindingForExecution(record, author.Ref)
	if !found || !TestAttestationConsumptionMatches(record, attestation) {
		return review.Subject{}, review.Gate{}, errors.New("council.persisted_subject_invalid")
	}
	subject, err := reviewSubjectFromEvidence(item, author, binding, change, attestation)
	if err != nil {
		return review.Subject{}, review.Gate{}, errors.New("council.persisted_subject_invalid")
	}
	assessments := make([]review.Assessment, 0, 2)
	for _, fact := range record.Reviews {
		if fact.ChangeSetRef != change.Ref || fact.SubjectDigest != subject.Digest() {
			continue
		}
		assessment, assessmentErr := fact.Assessment()
		if assessmentErr != nil {
			return review.Subject{}, review.Gate{}, errors.New("council.persisted_subject_invalid")
		}
		assessments = append(assessments, assessment)
	}
	gate, err := review.EvaluateGate(subject, assessments)
	if err != nil {
		return review.Subject{}, review.Gate{}, errors.New("council.persisted_subject_invalid")
	}
	return subject, gate, nil
}

func persistedCouncilWorkItem(record GoalRecord, subject council.Subject) (goal.WorkItem, bool) {
	for _, item := range record.Goal.WorkItems() {
		if item.Ref().String() == subject.WorkItemRef {
			return item, true
		}
	}
	return goal.WorkItem{}, false
}

func persistedCouncilAuthor(record GoalRecord, subject council.Subject) (ExecutionRecord, bool) {
	for _, candidate := range record.Executions {
		if candidate.Purpose == ExecutionPurposeAuthor && candidate.GoalRef.String() == subject.GoalRef &&
			candidate.WorkItemRef.String() == subject.WorkItemRef &&
			candidate.PlanGeneration == goal.PlanGeneration(subject.PlanGeneration) &&
			candidate.AppSpecGeneration == goal.AppSpecGeneration(subject.AppSpecGeneration) &&
			candidate.SpecHash == subject.SpecHash {
			return candidate, true
		}
	}
	return ExecutionRecord{}, false
}

func persistedCouncilChange(record GoalRecord, subject council.Subject) (ChangeSet, bool) {
	for _, candidate := range record.ChangeSets {
		if candidate.Ref.String() == subject.ChangeSetRef {
			return candidate, true
		}
	}
	return ChangeSet{}, false
}

func buildReviewSubject(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	change ChangeSet, policy TestAttestationPolicy,
) (review.Subject, error) {
	if execution.Purpose != ExecutionPurposeAuthor || execution.LaunchReceiptRef == "" || execution.ExternalRef == "" {
		return review.Subject{}, errors.New("review.subject_invalid")
	}
	binding, found := workspaceBindingForExecution(record, execution.Ref)
	if !found {
		return review.Subject{}, errors.New("review.subject_invalid")
	}
	attestation, passed := requiredTestsPassForChange(record, item, execution, change, policy)
	if !passed {
		return review.Subject{}, errors.New("review.subject_invalid")
	}
	return reviewSubjectFromEvidence(item, execution, binding, change, attestation)
}

func reviewSubjectFromEvidence(item goal.WorkItem, execution ExecutionRecord, binding WorkspaceBinding,
	change ChangeSet, attestation AttestationRecord,
) (review.Subject, error) {
	if execution.Purpose != ExecutionPurposeAuthor || execution.LaunchReceiptRef == "" || execution.ExternalRef == "" ||
		attestation.Verdict != AttestationVerdictPassed || attestation.ExecutionRef != execution.Ref ||
		attestation.ChangeSetRef != change.Ref || attestation.WorkspaceBindingDigest != binding.Digest() ||
		attestation.ChangeSetDigest != change.Digest() || attestation.RequiredTestsDigest != item.RequiredTestsDigest() {
		return review.Subject{}, errors.New("review.subject_invalid")
	}
	return review.NewSubject(review.Subject{
		GoalRef: execution.GoalRef.String(), WorkItemRef: execution.WorkItemRef.String(),
		AuthorExecutionRef: execution.Ref.String(), AuthorExecutionAttempt: execution.AttemptNo,
		PlanGeneration: uint64(execution.PlanGeneration), WorkItemGeneration: uint64(attestation.WorkItemGeneration),
		AppSpecGeneration: uint64(execution.AppSpecGeneration), SpecHash: execution.SpecHash,
		AuthorLaunchReceiptRef: execution.LaunchReceiptRef, AuthorExternalRef: execution.ExternalRef,
		WorkspaceBindingDigest: binding.Digest(),
		ChangeSetRef:           change.Ref.String(), ChangeSetDigest: change.Digest(), TreeOID: change.TreeOID,
		DiffDigest: change.DiffDigest, WriteSetDigest: change.WriteSetDigest,
		RequiredTestsDigest: attestation.RequiredTestsDigest, TestAttestationRef: attestation.Ref.String(),
		TestSubjectDigest: attestation.SubjectDigest, TestPolicyDigest: attestation.PolicyDigest,
	})
}

func reviewGateForChange(record GoalRecord, execution ExecutionRecord, change ChangeSet,
	policy TestAttestationPolicy,
) (review.Subject, review.Gate, error) {
	if execution.Purpose != ExecutionPurposeAuthor {
		return review.Subject{}, review.Gate{}, errors.New("review.subject_mismatch")
	}
	item, found := record.Goal.WorkItem(change.WorkItemRef)
	if !found {
		return review.Subject{}, review.Gate{}, errors.New("review.subject_mismatch")
	}
	subject, err := buildReviewSubject(record, item, execution, change, policy)
	if err != nil {
		return review.Subject{}, review.Gate{}, errors.New("review.subject_mismatch")
	}
	assessments := make([]review.Assessment, 0, 2)
	for _, fact := range record.Reviews {
		if fact.ChangeSetRef != change.Ref {
			continue
		}
		assessments = append(assessments, review.Assessment{
			SubjectDigest: fact.SubjectDigest, Role: fact.Role, Verdict: fact.Verdict,
			ReviewerExecutionRef: fact.ReviewerExecutionRef.String(), ReviewerExecutionAttempt: fact.ReviewerExecutionAttempt,
			LaunchReceiptRef: fact.LaunchReceiptRef, ReviewerExternalRef: fact.ExternalRef,
			AssessmentArtifactRef: fact.AssessmentArtifactRef,
			AssessmentDigest:      fact.AssessmentDigest, RecordedAt: fact.RecordedAt,
		})
	}
	gate, err := review.EvaluateGate(subject, assessments)
	if err != nil {
		return subject, review.Gate{}, err
	}
	return subject, gate, nil
}

func reviewGateAllowsIntegration(record GoalRecord, execution ExecutionRecord, change ChangeSet,
	policy TestAttestationPolicy,
) (string, error) {
	_, gate, err := reviewGateForChange(record, execution, change, policy)
	if err != nil {
		return "", err
	}
	if gate.Status == review.GateApproved {
		return gate.Digest, nil
	}
	if gate.Status == review.GateChangesRequested {
		return "", errors.New("review.changes_requested")
	}
	return "", errors.New("review.required")
}

// ValidatePersistedIntegrationReviewGate is used by durable adapters during
// recovery. Historical completed V17 integrations are not passed here; every
// live V18 integration action must carry the exact approved pair gate.
func ValidatePersistedIntegrationReviewGate(record GoalRecord, action ActionRecord) error {
	item, itemFound := record.Goal.WorkItem(action.WorkItemRef)
	execution, executionFound := executionByRef(record.Executions, action.ExecutionRef)
	change, changeFound := changeSetByRef(record, action.ChangeRef)
	if !itemFound || !executionFound || !changeFound || action.Kind != ActionIntegrateChange ||
		action.GoalRef != record.Goal.Ref() || change.WorkItemRef != item.Ref() ||
		change.ExecutionRef != execution.Ref || action.ReviewGateDigest == "" {
		return errors.New("review.integration_gate_invalid")
	}
	var gateDigest string
	var gatePolicy TestAttestationPolicy
	matches := 0
	for _, attestation := range record.Attestations {
		if attestation.Kind != AttestationKindRequiredTests || attestation.Verdict != AttestationVerdictPassed ||
			attestation.ExecutionRef != execution.Ref || attestation.ChangeSetRef != change.Ref {
			continue
		}
		policy := TestAttestationPolicy{Ref: attestation.PolicyRef, Digest: attestation.PolicyDigest}
		candidate, err := reviewGateAllowsIntegration(record, execution, change, policy)
		if err == nil {
			gateDigest, gatePolicy, matches = candidate, policy, matches+1
		}
	}
	var intent EffectIntent
	intentMatches := 0
	for _, candidate := range record.EffectIntents {
		if candidate.ActionRef == action.Ref && candidate.ActionKind == ActionIntegrateChange {
			intent, intentMatches = candidate, intentMatches+1
		}
	}
	if matches != 1 || gateDigest != action.ReviewGateDigest || intentMatches != 1 {
		return errors.New("review.integration_gate_invalid")
	}
	if action.CouncilResolution == nil && intent.CouncilResolution == nil {
		if intent.TargetDigest == integrationTargetDigest(change, action.ExpectedTargetOID, gateDigest) {
			return nil
		}
		return errors.New("review.integration_gate_invalid")
	}
	resolution, resolutionErr := councilIntegrationResolution(record, item, execution, change, gateDigest, gatePolicy)
	if resolutionErr != nil || !councilResolutionEqual(action.CouncilResolution, resolution) ||
		!councilResolutionEqual(intent.CouncilResolution, resolution) ||
		intent.TargetDigest != integrationTargetDigest(change, action.ExpectedTargetOID, gateDigest, resolution) {
		return errors.New("review.integration_gate_invalid")
	}
	return nil
}

func changeForAuthor(record GoalRecord, execution ExecutionRecord) (ChangeSet, bool) {
	var result ChangeSet
	found := false
	for _, change := range record.ChangeSets {
		if change.ExecutionRef != execution.Ref {
			continue
		}
		if found {
			return ChangeSet{}, false
		}
		result, found = change, true
	}
	return result, found
}

// FailedReviewPreservesCandidate validates the only review failures allowed to
// retain an immutable, non-integrated candidate for Director replan.
func FailedReviewPreservesCandidate(record GoalRecord, item goal.WorkItem, author ExecutionRecord,
	change ChangeSet,
) bool {
	subject, err := ResolvePersistedReviewSubject(record, item, author, change)
	if err != nil {
		return false
	}
	digest := subject.Digest()
	latest := make(map[review.Role]ExecutionRecord, 2)
	for _, participant := range record.Executions {
		role, reviewer := reviewerRole(participant)
		if !reviewer || participant.ReviewSubjectDigest != digest {
			continue
		}
		if previous, found := latest[role]; !found || participant.AttemptNo > previous.AttemptNo {
			latest[role] = participant
		}
	}
	unavailable := false
	for _, role := range []review.Role{review.RolePrimary, review.RoleAdversarial} {
		participant, found := latest[role]
		if !found {
			return false
		}
		switch participant.State {
		case ExecutionSucceeded:
		case ExecutionFailed, ExecutionStopped:
			unavailable = true
		default:
			return false
		}
	}
	if author.FailureCode == "review.unavailable" {
		return unavailable
	}
	if author.FailureCode != string(goal.ReplanCauseReviewChangesRequested) {
		return false
	}
	assessments := make([]review.Assessment, 0, 2)
	for _, fact := range record.Reviews {
		if fact.SubjectDigest != digest || fact.ChangeSetRef != change.Ref {
			continue
		}
		assessment, assessmentErr := fact.Assessment()
		if assessmentErr != nil {
			return false
		}
		assessments = append(assessments, assessment)
	}
	gate, err := review.EvaluateGate(subject, assessments)
	return err == nil && gate.Status == review.GateChangesRequested
}
