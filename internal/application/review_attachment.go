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
		if facts != 1 {
			return errors.New("review.persisted_assessment_invalid")
		}
	case ExecutionQueued, ExecutionDispatching, ExecutionRunning, ExecutionFailed, ExecutionStopped:
		if facts != 0 {
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
		binding, bindingFound := workspaceBindingForExecution(record, author.Ref)
		if !changeFound || !bindingFound || change.WorkspaceRef != binding.Ref {
			continue
		}
		for _, attestation := range record.Attestations {
			if attestation.Kind != AttestationKindRequiredTests ||
				attestation.Verdict != AttestationVerdictPassed || attestation.ExecutionRef != author.Ref ||
				attestation.ChangeSetRef != change.Ref {
				continue
			}
			subject, err := reviewSubjectFromEvidence(item, author, binding, change, attestation)
			if err != nil || subject.Digest() != participant.ReviewSubjectDigest {
				continue
			}
			attachment = ReviewAttachment{Author: author, Change: change, Subject: subject}
			matches++
		}
	}
	if matches != 1 {
		return ReviewAttachment{}, errors.New("review.attachment_persisted_invalid")
	}
	return attachment, nil
}
