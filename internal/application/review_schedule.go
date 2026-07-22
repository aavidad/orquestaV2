package application

import (
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/review"
)

func (orchestrator *Orchestrator) scheduleIndependentReviews(
	record GoalRecord,
	item goal.WorkItem,
	author ExecutionRecord,
	subject review.Subject,
	at time.Time,
) ([]ExecutionRecord, []ActionRecord, []EventRecord, error) {
	authority, found := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	if !found {
		return nil, nil, nil, errors.New("application.work_item_authority_missing")
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return nil, nil, nil, err
	}
	executions := make([]ExecutionRecord, 0, 2)
	actions := make([]ActionRecord, 0, 2)
	events := make([]EventRecord, 0, 2)
	for _, role := range []review.Role{review.RolePrimary, review.RoleAdversarial} {
		execution, err := reviewerExecution(record.Goal, item, author, subject.Digest(), role, at,
			orchestrator.maxExecutionAttempts, orchestrator.maxOutputBytes)
		if err != nil {
			return nil, nil, nil, err
		}
		scheduled := record
		scheduled.Executions = append(append([]ExecutionRecord(nil), record.Executions...), executions...)
		scheduled.Executions = append(scheduled.Executions, execution)
		change, changeFound := changeForAuthor(record, author)
		if !changeFound {
			return nil, nil, nil, errors.New("review.change_missing")
		}
		attachment := ReviewAttachment{Author: author, Change: change, Subject: subject}
		action, err := orchestrator.reviewerLaunchAction(policy, scheduled, item, execution, authority, at, at, attachment)
		if err != nil {
			return nil, nil, nil, err
		}
		executions = append(executions, execution)
		actions = append(actions, action)
		events = append(events, EventRecord{
			Ref: "event:review-queued:" + execution.Ref.String(), Kind: "review.queued",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at.UTC(),
		})
	}
	return executions, actions, events, nil
}

func (orchestrator *Orchestrator) reviewerLaunchAction(policy effectPolicySnapshot, record GoalRecord,
	item goal.WorkItem, execution ExecutionRecord, authority WorkItemAuthority, at, availableAt time.Time,
	attachment ReviewAttachment,
) (ActionRecord, error) {
	phase, found := phaseForWorkItem(record.Goal, item)
	if !found {
		return ActionRecord{}, errors.New("application.phase_missing")
	}
	role, reviewer := reviewerRole(execution)
	if !reviewer || attachment.Author.Ref.String() == "" || attachment.Subject.Digest() != execution.ReviewSubjectDigest {
		return ActionRecord{}, errors.New("review.attachment_invalid")
	}
	request, err := reviewerAgentLaunchRequestFromAttachment(record, item, execution, phase, role, attachment)
	if err != nil {
		return ActionRecord{}, err
	}
	actionRef := "action:launch:" + execution.Ref.String()
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: authority.AuthorizationReceipt.Decision().Request().RequestRef(),
		RequestFingerprint: effectAdmissionFingerprint(actionRef, authority.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionLaunchAgent, Kind: EffectKindAgentLaunch,
		Subject: effectSubject(record.Goal, item, execution), ProposedBy: authority.PrincipalRef,
		Permission: authority.Permission, Authority: authority.AuthorizationReceipt, Demand: item.BudgetDemand(),
		SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
		PolicyHash: policy.PolicyHash, PolicyRevision: policy.PolicyRevision, QuotaRetryDelay: policy.QuotaRetryDelay,
		ApprovalTTL: policy.ApprovalTTL, TargetDigest: reviewerLaunchTargetDigest(request),
		IdempotencyKey: execution.IdempotencyKey, CreatedAt: at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{Ref: actionRef, Kind: ActionLaunchAgent,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: availableAt,
	}, authority.Source, at)
}

func reviewerExecution(aggregate goal.Goal, item goal.WorkItem, author ExecutionRecord,
	subjectDigest string, role review.Role, at time.Time, maxAttempts uint64, maxOutputBytes int64,
) (ExecutionRecord, error) {
	purpose := ExecutionPurposePrimaryReview
	if role == review.RoleAdversarial {
		purpose = ExecutionPurposeAdversarialReview
	}
	ref, err := goal.NewExecutionRef("execution:review:" + fingerprintFields(
		"orquesta.review-execution.v1", aggregate.Ref().String(), item.Ref().String(), subjectDigest, string(role), "1",
	))
	if err != nil {
		return ExecutionRecord{}, err
	}
	return ExecutionRecord{
		Ref: ref, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), AttemptNo: 1,
		MaxExecutionAttempts: maxAttempts, PlanGeneration: author.PlanGeneration,
		AppSpecGeneration: author.AppSpecGeneration, SpecHash: author.SpecHash,
		State: ExecutionQueued, Purpose: purpose, ReviewSubjectDigest: subjectDigest,
		RepositoryRef: author.RepositoryRef, ExecutionWorkspaceRef: author.ExecutionWorkspaceRef,
		ArtifactMediaType: review.AssessmentMediaType, IdempotencyKey: "execution:" + ref.String(),
		MaxOutputBytes: maxOutputBytes, CreatedAt: at.UTC(),
	}, nil
}

func reviewerRole(execution ExecutionRecord) (review.Role, bool) {
	switch execution.Purpose {
	case ExecutionPurposePrimaryReview:
		return review.RolePrimary, true
	case ExecutionPurposeAdversarialReview:
		return review.RoleAdversarial, true
	default:
		return "", false
	}
}

func isReviewerExecution(execution ExecutionRecord) bool {
	_, ok := reviewerRole(execution)
	return ok
}
