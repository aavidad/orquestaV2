package application

import (
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/review"
)

// ReviewAttachment is the single relation between the WorkItem authority
// execution and an attached, read-only reviewer participant.
type ReviewAttachment struct {
	Author  ExecutionRecord
	Change  ChangeSet
	Subject review.Subject
}

func authorBound(record GoalRecord, item goal.WorkItem) (ExecutionRecord, bool) {
	ref, bound := item.Execution()
	if !bound {
		return ExecutionRecord{}, false
	}
	author, found := executionByRef(record.Executions, ref)
	if !found || author.Purpose != ExecutionPurposeAuthor || author.GoalRef != record.Goal.Ref() ||
		author.WorkItemRef != item.Ref() {
		return ExecutionRecord{}, false
	}
	return author, true
}

func reviewAttached(record GoalRecord, item goal.WorkItem, participant ExecutionRecord,
	policy TestAttestationPolicy,
) (ReviewAttachment, error) {
	if !isReviewerExecution(participant) || participant.GoalRef != record.Goal.Ref() ||
		participant.WorkItemRef != item.Ref() || participant.ReviewSubjectDigest == "" {
		return ReviewAttachment{}, errors.New("review.attachment_participant_invalid")
	}
	author, found := authorBound(record, item)
	change, changeFound := changeForAuthor(record, author)
	if !found || !changeFound || author.Ref == participant.Ref ||
		participant.PlanGeneration != author.PlanGeneration ||
		participant.AppSpecGeneration != author.AppSpecGeneration || participant.SpecHash != author.SpecHash ||
		participant.RepositoryRef != author.RepositoryRef ||
		participant.ExecutionWorkspaceRef != author.ExecutionWorkspaceRef {
		return ReviewAttachment{}, errors.New("review.attachment_author_invalid")
	}
	subject, err := buildReviewSubject(record, item, author, change, policy)
	if err != nil || subject.Digest() != participant.ReviewSubjectDigest {
		return ReviewAttachment{}, errors.New("review.attachment_subject_invalid")
	}
	return ReviewAttachment{Author: author, Change: change, Subject: subject}, nil
}

// ResolveReviewAttachment exposes the same invariant to state adapters. It
// does not confer authority or mutate the WorkItem binding.
func ResolveReviewAttachment(record GoalRecord, item goal.WorkItem, participant ExecutionRecord,
	policy TestAttestationPolicy,
) (ReviewAttachment, error) {
	return reviewAttached(record, item, participant, policy)
}

// ValidatePersistedReviewerBinding proves a reviewer that has not emitted an
// assessment is still attached to the exact durable author/change/PASS subject
// and to the canonical launch intent. Recovery must not wait until dispatch to
// discover a corrupted queued reviewer.
func ValidatePersistedReviewerBinding(record GoalRecord, item goal.WorkItem,
	participant ExecutionRecord,
) error {
	attachment, err := persistedReviewAttachment(record, item, participant)
	if err != nil {
		return errors.New("review.persisted_attachment_invalid")
	}
	facts := 0
	for _, fact := range record.Reviews {
		if fact.ReviewerExecutionRef != participant.Ref {
			continue
		}
		role, reviewer := reviewerRole(participant)
		if !reviewer || fact.GoalRef != participant.GoalRef || fact.WorkItemRef != participant.WorkItemRef ||
			fact.ChangeSetRef != attachment.Change.Ref || fact.SubjectDigest != participant.ReviewSubjectDigest ||
			fact.Role != role || fact.ReviewerExecutionAttempt != participant.AttemptNo ||
			fact.LaunchReceiptRef != participant.LaunchReceiptRef || fact.AgentRef != participant.AgentRef ||
			fact.ExternalRef != participant.ExternalRef {
			return errors.New("review.persisted_assessment_invalid")
		}
		facts++
	}
	switch participant.State {
	case ExecutionSucceeded:
		if facts != 1 && (facts != 0 || !persistedReviewerSettledDuringCancel(record, item, participant)) {
			return errors.New("review.persisted_assessment_invalid")
		}
	case ExecutionQueued, ExecutionDispatching, ExecutionRunning, ExecutionFailed, ExecutionStopped:
		if facts != 0 {
			return errors.New("review.persisted_assessment_invalid")
		}
	case ExecutionCanceled:
		if facts != 0 || !persistedReviewerCancelCausal(record, item, participant) {
			return errors.New("review.persisted_assessment_invalid")
		}
	default:
		return errors.New("review.persisted_state_invalid")
	}
	phase, found := phaseForWorkItem(record.Goal, item)
	role, reviewer := reviewerRole(participant)
	if !found || !reviewer {
		return errors.New("review.persisted_launch_invalid")
	}
	request, err := reviewerAgentLaunchRequestFromAttachment(
		record, item, participant, phase, role, attachment,
	)
	if err != nil {
		return errors.New("review.persisted_launch_invalid")
	}
	actionRef := "action:launch:" + participant.Ref.String()
	authority, authorityFound := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	var intent EffectIntent
	intentMatches := 0
	for _, candidate := range record.EffectIntents {
		if candidate.ActionRef == actionRef && candidate.ActionKind == ActionLaunchAgent &&
			candidate.Kind == EffectKindAgentLaunch {
			intent, intentMatches = candidate, intentMatches+1
		}
	}
	if !authorityFound || intentMatches != 1 || intent.Ref != "effect-intent:"+actionRef ||
		intent.Subject != effectSubject(record.Goal, item, participant) ||
		intent.ProposedBy != authority.PrincipalRef || intent.Permission != authority.Permission ||
		intent.Authority.Ref() != authority.AuthorizationReceipt.Ref() ||
		intent.TargetDigest != reviewerLaunchTargetDigest(request) || intent.IdempotencyKey != participant.IdempotencyKey {
		return errors.New("review.persisted_launch_invalid")
	}
	return nil
}

func persistedReviewerCancelCausal(
	record GoalRecord,
	item goal.WorkItem,
	participant ExecutionRecord,
) bool {
	if participant.FailureCode != "application.execution_canceled" || participant.FinishedAt.IsZero() {
		return false
	}
	_, found := persistedReviewerCancelControl(record, item, participant, true)
	return found
}

func persistedReviewerSettledDuringCancel(
	record GoalRecord,
	item goal.WorkItem,
	participant ExecutionRecord,
) bool {
	if participant.State != ExecutionSucceeded || participant.FinishedAt.IsZero() {
		return false
	}
	control, found := persistedReviewerCancelControl(record, item, participant, false)
	if !found {
		return false
	}
	stopActionRef := "action:stop:" + control.Ref + ":" + participant.Ref.String()
	stopEffects := 0
	stopReceiptRef := ""
	for _, receipt := range record.EffectReceipts {
		if receipt.ActionRef != stopActionRef || receipt.Subject.ExecutionRef != participant.Ref ||
			receipt.Subject.GoalRef != participant.GoalRef ||
			receipt.Subject.WorkItemRef != participant.WorkItemRef ||
			receipt.Status != EffectStatusAlreadyCompleted ||
			receipt.ConfirmedAt.Before(control.RequestedAt) ||
			receipt.ConfirmedAt.After(participant.FinishedAt) {
			continue
		}
		stopReceiptRef = receipt.Ref
		stopEffects++
	}
	if stopEffects != 1 {
		return false
	}
	stopConsumptions, observations := 0, 0
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.ExecutionRef != participant.Ref || receipt.GoalRef != participant.GoalRef ||
			receipt.WorkItemRef != participant.WorkItemRef ||
			receipt.Outcome != ActionConsumedCompleted || receipt.ErrorCode != "" {
			continue
		}
		switch {
		case receipt.Kind == ActionStopAgent && receipt.ActionRef == stopActionRef &&
			receipt.EffectReceiptRef == stopReceiptRef:
			stopConsumptions++
		case receipt.Kind == ActionObserveAgent &&
			receipt.ActionRef == "action:observe:"+participant.Ref.String() &&
			receipt.EffectReceiptRef == "" && receipt.ConsumedAt.Equal(participant.FinishedAt):
			observations++
		}
	}
	return stopConsumptions == 1 && observations == 1
}

func persistedReviewerCancelControl(
	record GoalRecord,
	item goal.WorkItem,
	participant ExecutionRecord,
	exactRequestTime bool,
) (ControlRecord, bool) {
	if item.State() == goal.WorkItemStateRunning {
		if !item.CancelRequested() {
			return ControlRecord{}, false
		}
	} else if item.State() != goal.WorkItemStateCanceled {
		return ControlRecord{}, false
	}
	var selected ControlRecord
	matches := 0
	for _, control := range record.Controls {
		targetsItem := control.Target == ControlTargetGoal ||
			control.Target == ControlTargetWorkItem && control.WorkItemRef == item.Ref()
		causalTime := !participant.FinishedAt.Before(control.RequestedAt)
		if exactRequestTime {
			causalTime = participant.FinishedAt.Equal(control.RequestedAt)
		}
		if control.Status == ControlConfirmed && item.State() == goal.WorkItemStateCanceled {
			causalTime = causalTime && !participant.FinishedAt.After(control.ConfirmedAt)
		}
		if item.State() == goal.WorkItemStateRunning && control.Status != ControlRequested {
			continue
		}
		if control.Operation != ControlCancel ||
			(control.Status != ControlRequested && control.Status != ControlConfirmed) ||
			control.GoalRef != participant.GoalRef || !targetsItem || !causalTime ||
			ValidatePersistedControlRecord(control) != nil {
			continue
		}
		selected, matches = control, matches+1
	}
	return selected, matches == 1
}

// ValidatePersistedReviewRecord centralizes the pure aggregate invariant used
// by durable adapters. Persistence owns atomicity, not a second review model.
func ValidatePersistedReviewRecord(record GoalRecord, item goal.WorkItem,
	fact ReviewRecord,
) (review.Subject, review.Assessment, error) {
	reviewer, reviewerFound := executionByRef(record.Executions, fact.ReviewerExecutionRef)
	attachment, attachmentErr := persistedReviewAttachment(record, item, reviewer)
	intent, intentFound := effectIntentByRef(record.EffectIntents, reviewer.EffectIntentRef)
	authority, authorityFound := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	role, roleFound := reviewerRole(reviewer)
	artifact, artifactFound := artifactForReview(record.Artifacts, fact.AssessmentArtifactRef)
	if !reviewerFound || attachmentErr != nil || !intentFound || !authorityFound || !roleFound || !artifactFound ||
		fact.GoalRef != record.Goal.Ref() || fact.WorkItemRef != item.Ref() || fact.ChangeSetRef != attachment.Change.Ref ||
		fact.Role != role || fact.SubjectDigest != reviewer.ReviewSubjectDigest || fact.Ref != "review:"+reviewer.Ref.String() ||
		fact.ReviewerExecutionAttempt != reviewer.AttemptNo || fact.LaunchReceiptRef != reviewer.LaunchReceiptRef ||
		fact.AgentRef != reviewer.AgentRef || fact.ExternalRef != reviewer.ExternalRef ||
		fact.PrincipalRef != intent.ProposedBy || fact.PrincipalRef != authority.PrincipalRef ||
		intent.Permission != authority.Permission || intent.Authority.Ref() != authority.AuthorizationReceipt.Ref() ||
		reviewer.State != ExecutionSucceeded || !fact.RecordedAt.Equal(reviewer.FinishedAt) ||
		!fact.RecordedAt.Equal(artifact.CreatedAt) || artifact.Kind != ArtifactKindReviewAssessment ||
		artifact.ExecutionRef != reviewer.Ref || artifact.GoalRef != fact.GoalRef || artifact.WorkItemRef != fact.WorkItemRef ||
		artifact.ExecutionAttempt != reviewer.AttemptNo || artifact.PlanGeneration != reviewer.PlanGeneration ||
		artifact.AppSpecGeneration != reviewer.AppSpecGeneration || artifact.SpecHash != reviewer.SpecHash ||
		artifact.WorkItemGeneration != goal.Revision(attachment.Subject.WorkItemGeneration) ||
		artifact.Stored.Digest != fact.AssessmentDigest || artifact.Stored.Ref.String() != fact.AssessmentArtifactRef {
		return review.Subject{}, review.Assessment{}, errors.New("review.persisted_record_invalid")
	}
	assessment, err := fact.Assessment()
	if err != nil {
		return review.Subject{}, review.Assessment{}, errors.New("review.persisted_record_invalid")
	}
	return attachment.Subject, assessment, nil
}

func artifactForReview(artifacts []ArtifactRecord, ref string) (ArtifactRecord, bool) {
	for _, artifact := range artifacts {
		if artifact.Stored.Ref.String() == ref {
			return artifact, true
		}
	}
	return ArtifactRecord{}, false
}

func ValidatePersistedReviewCleanupControl(record GoalRecord, control ControlRecord) error {
	item, itemFound := record.Goal.WorkItem(control.WorkItemRef)
	reviewer, reviewerFound := executionByRef(record.Executions, control.ExecutionRef)
	author, authorFound := authorBound(record, item)
	intent, intentFound := effectIntentByRef(record.EffectIntents, author.EffectIntentRef)
	authority, authorityFound := workItemAuthorityFor(record.WorkItemAuthorities, control.WorkItemRef)
	_, reviewerRoleFound := reviewerRole(reviewer)
	if !IsReviewCleanupControl(control) || !itemFound || !reviewerFound || !authorFound || !intentFound ||
		!authorityFound || !reviewerRoleFound || control.GoalRef != record.Goal.Ref() ||
		reviewer.GoalRef != control.GoalRef || reviewer.WorkItemRef != control.WorkItemRef ||
		reviewer.AttemptNo != control.ExecutionAttempt || author.Purpose != ExecutionPurposeAuthor ||
		intent.ActionKind != ActionLaunchAgent || intent.Subject.ProjectRef != record.Goal.Project() ||
		intent.Subject.GoalRef != record.Goal.Ref() || intent.Subject.WorkItemRef != author.WorkItemRef ||
		intent.Subject.ExecutionRef != author.Ref || intent.Subject.PlanGeneration != author.PlanGeneration ||
		intent.Subject.AppSpecGeneration != author.AppSpecGeneration || intent.Subject.SpecHash != author.SpecHash ||
		intent.ProposedBy != control.PrincipalRef || intent.Authority.Ref() != control.AuthorizationReceipt.Ref() ||
		authority.PrincipalRef != control.PrincipalRef ||
		authority.AuthorizationReceipt.Ref() != control.AuthorizationReceipt.Ref() {
		return errors.New("review.cleanup_control_invalid")
	}
	return nil
}

// persistedReviewAttachment resolves the author from immutable evidence, not
// from the WorkItem's current lifecycle revision. This keeps completed and
// retried reviewer attempts valid after unrelated WorkItems progress while
// still requiring one exact author/change/binding/PASS subject.
func persistedReviewAttachment(record GoalRecord, item goal.WorkItem,
	participant ExecutionRecord,
) (ReviewAttachment, error) {
	if !isReviewerExecution(participant) || participant.GoalRef != record.Goal.Ref() ||
		participant.WorkItemRef != item.Ref() || participant.ReviewSubjectDigest == "" {
		return ReviewAttachment{}, errors.New("review.attachment_persisted_invalid")
	}
	var attachment ReviewAttachment
	matches := 0
	for _, author := range record.Executions {
		if author.Purpose != ExecutionPurposeAuthor || author.GoalRef != participant.GoalRef ||
			author.WorkItemRef != participant.WorkItemRef || author.Ref == participant.Ref ||
			author.PlanGeneration != participant.PlanGeneration ||
			author.AppSpecGeneration != participant.AppSpecGeneration || author.SpecHash != participant.SpecHash ||
			author.RepositoryRef != participant.RepositoryRef ||
			author.ExecutionWorkspaceRef != participant.ExecutionWorkspaceRef {
			continue
		}
		change, changeFound := changeForAuthor(record, author)
		if !changeFound {
			continue
		}
		subject, err := ResolvePersistedReviewSubject(record, item, author, change)
		if err == nil && subject.Digest() == participant.ReviewSubjectDigest {
			attachment, matches = ReviewAttachment{Author: author, Change: change, Subject: subject}, matches+1
		}
	}
	if matches != 1 {
		return ReviewAttachment{}, errors.New("review.attachment_persisted_invalid")
	}
	return attachment, nil
}

// ResolvePersistedReviewSubject reconstructs one exact immutable review
// subject from author/change/binding/PASS evidence after lifecycle progress.
func ResolvePersistedReviewSubject(record GoalRecord, item goal.WorkItem, author ExecutionRecord,
	change ChangeSet,
) (review.Subject, error) {
	binding, bindingFound := workspaceBindingForExecution(record, author.Ref)
	var subject review.Subject
	matches := 0
	for _, pass := range record.Attestations {
		if pass.Kind != AttestationKindRequiredTests || pass.Verdict != AttestationVerdictPassed ||
			pass.ExecutionRef != author.Ref || pass.ChangeSetRef != change.Ref {
			continue
		}
		candidate, err := reviewSubjectFromEvidence(item, author, binding, change, pass)
		if err == nil {
			subject, matches = candidate, matches+1
		}
	}
	if author.Purpose != ExecutionPurposeAuthor || !bindingFound || change.ExecutionRef != author.Ref ||
		change.WorkspaceRef != binding.Ref || matches != 1 {
		return review.Subject{}, errors.New("review.subject_persisted_invalid")
	}
	return subject, nil
}
