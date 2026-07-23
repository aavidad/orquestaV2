package application

import (
	"context"
	"errors"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

func (orchestrator *Orchestrator) recordReviewerObservation(ctx context.Context, claim ActionClaim,
	record GoalRecord, item goal.WorkItem, execution ExecutionRecord, observation ports.AgentObservation, at time.Time,
) error {
	role, ok := reviewerRole(execution)
	if !ok {
		return &StateError{Code: StateConflict}
	}
	payload, err := review.DecodeArtifact(observation.Content)
	if err != nil || payload.Role != role || payload.SubjectDigest != execution.ReviewSubjectDigest {
		diagnostic, diagnosticErr := orchestrator.publishReviewDiagnostic(
			ctx, record, item, execution, observation.Content, "review.assessment_invalid", at,
		)
		if diagnosticErr != nil {
			return orchestrator.replaceReviewerExecution(ctx, claim, record, execution,
				"artifact.store_failed", at, observation.Usage, int64(len(observation.Content)), false)
		}
		return orchestrator.replaceReviewerExecutionWithDiagnostic(ctx, claim, record, execution,
			"review.assessment_invalid", at, observation.Usage, int64(len(observation.Content)), false, diagnostic)
	}
	for _, prior := range record.Reviews {
		if prior.SubjectDigest == payload.SubjectDigest && prior.Role == role {
			return orchestrator.quarantine(ctx, claim, "review.duplicate_role")
		}
	}
	stored, err := orchestrator.publishTestArtifact(ctx, ports.PutArtifactRequest{
		MediaType: review.AssessmentMediaType, Content: observation.Content,
	})
	if err != nil {
		return orchestrator.replaceReviewerExecution(ctx, claim, record, execution,
			"artifact.store_failed", at, observation.Usage, int64(len(observation.Content)), false)
	}
	attachment, attachmentErr := reviewAttached(record, item, execution, orchestrator.testAttestationPolicy)
	author, change := attachment.Author, attachment.Change
	launchIntent, intentFound := effectIntentByRef(record.EffectIntents, execution.EffectIntentRef)
	if attachmentErr != nil || author.State != ExecutionAwaitingIntegration || !intentFound ||
		launchIntent.ActionKind != ActionLaunchAgent || launchIntent.Subject.GoalRef != record.Goal.Ref() ||
		launchIntent.Subject.WorkItemRef != item.Ref() || launchIntent.Subject.ExecutionRef != execution.Ref {
		return orchestrator.quarantine(ctx, claim, "review.subject_mismatch")
	}
	reviewFact := ReviewRecord{
		Ref: "review:" + execution.Ref.String(), GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(),
		ChangeSetRef: change.Ref, SubjectDigest: payload.SubjectDigest, Role: role, Verdict: payload.Verdict,
		ReviewerExecutionRef: execution.Ref, ReviewerExecutionAttempt: execution.AttemptNo,
		LaunchReceiptRef: execution.LaunchReceiptRef, PrincipalRef: launchIntent.ProposedBy,
		AgentRef: execution.AgentRef, ExternalRef: execution.ExternalRef,
		AssessmentArtifactRef: stored.Ref.String(), AssessmentDigest: stored.Digest,
		RecordedAt: at.UTC(),
	}
	assessment, err := reviewFact.Assessment()
	if err != nil {
		return orchestrator.quarantine(ctx, claim, "review.assessment_invalid")
	}
	_ = assessment
	artifact := ArtifactRecord{
		OccurrenceRef: "artifact-occurrence:review:" + execution.Ref.String(), Kind: ArtifactKindReviewAssessment,
		Stored: stored, GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AppSpecGeneration: execution.AppSpecGeneration,
		SpecHash: execution.SpecHash, CreatedAt: at.UTC(),
	}
	execution.State, execution.FinishedAt = ExecutionSucceeded, at.UTC()
	candidate := record
	candidate.Reviews = append(append([]ReviewRecord(nil), record.Reviews...), reviewFact)
	_, gate, err := reviewGateForChange(candidate, author, change, orchestrator.testAttestationPolicy)
	if err != nil {
		return orchestrator.quarantine(ctx, claim, "review.subject_mismatch")
	}
	aggregate := record.Goal
	if gate.Status == review.GateChangesRequested {
		aggregate, err = aggregate.InterruptWorkItem(aggregate.Revision(), item.Revision(), item.Ref(), author.Ref,
			goal.WorkItemInterruptExecutionFailed, lifecycleTime(at, aggregate, item))
		if err != nil {
			return err
		}
		author.State, author.FinishedAt = ExecutionFailed, at.UTC()
		author.FailureCode = string(goal.ReplanCauseReviewChangesRequested)
	}
	settlement, err := settlementFor(record, execution, observation.Usage, int64(len(observation.Content)), at)
	if err != nil {
		return err
	}
	events := []EventRecord{{Ref: "event:review-assessed:" + execution.Ref.String(), Kind: "review.assessed",
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at.UTC()}}
	if gate.Status == review.GateChangesRequested {
		events = append(events, EventRecord{Ref: "event:review-changes-requested:" + change.Ref.String(),
			Kind: string(goal.ReplanCauseReviewChangesRequested), GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
			ExecutionRef: author.Ref, OccurredAt: at.UTC()})
	}
	return orchestrator.state.RecordReviewAssessed(ctx, ReviewAssessedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: aggregate, ReviewerExecution: execution, AuthorExecution: author,
		Artifact: artifact, Review: reviewFact, BudgetSettlement: settlement, Events: events, OperationAt: at.UTC(),
	})
}

func (orchestrator *Orchestrator) replaceReviewerExecution(ctx context.Context, claim ActionClaim,
	record GoalRecord, execution ExecutionRecord, code string, at time.Time, usage governance.ResourceUsage,
	diskBytes int64, definitelyUnapplied bool,
) error {
	return orchestrator.replaceReviewerExecutionWithDiagnostic(
		ctx, claim, record, execution, code, at, usage, diskBytes, definitelyUnapplied, nil,
	)
}

func (orchestrator *Orchestrator) replaceReviewerExecutionWithDiagnostic(ctx context.Context, claim ActionClaim,
	record GoalRecord, execution ExecutionRecord, code string, at time.Time, usage governance.ResourceUsage,
	diskBytes int64, definitelyUnapplied bool, diagnostic *ArtifactRecord,
) error {
	item, found := record.Goal.WorkItem(execution.WorkItemRef)
	if !found || !isReviewerExecution(execution) || item.State() != goal.WorkItemStateRunning {
		return &StateError{Code: StateConflict}
	}
	at = lifecycleTime(at, record.Goal, item)
	execution.State, execution.FailureCode, execution.FinishedAt = ExecutionFailed, stableFailureCode(code), at.UTC()
	settlement, err := settlementForExecutionAttempt(record, claim, execution, usage, diskBytes, at, definitelyUnapplied)
	if err != nil {
		return err
	}
	if execution.AttemptNo >= execution.MaxExecutionAttempts {
		retired, actionRefs, cleanupControls, cleanupActions, cleanupEvents, cleanupErr :=
			orchestrator.reviewCleanupPlan(record, item, execution.Ref, execution.ReviewSubjectDigest,
				claim.Action.Ref, at)
		if cleanupErr != nil {
			return cleanupErr
		}
		aggregate, author := record.Goal, ExecutionRecord{}
		if !reviewCleanupStillActive(record, execution, retired) {
			var interruptErr error
			aggregate, author, interruptErr = orchestrator.interruptAuthorForReviewFailure(
				record, item, at, "review.unavailable",
			)
			if interruptErr != nil {
				return interruptErr
			}
		}
		events := append([]EventRecord{{Ref: "event:review-failed:" + execution.Ref.String(), Kind: "review.failed",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at.UTC()}},
			cleanupEvents...)
		return orchestrator.state.RecordReviewExecutionFailed(ctx, ReviewExecutionFailedState{
			Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
			Execution: execution, Goal: aggregate, AuthorExecution: author,
			RetiredReviewers: retired, RetireActionRefs: actionRefs,
			CleanupControls: cleanupControls, CleanupActions: cleanupActions, DiagnosticArtifact: diagnostic,
			BudgetSettlement: settlement, Events: events, OperationAt: at.UTC(),
		})
	}
	replacement, err := reviewerReplacementExecution(execution, at)
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
	retryRecord := record
	retryRecord.Executions = append(append([]ExecutionRecord(nil), record.Executions...), replacement)
	attachment, attachmentErr := reviewAttached(retryRecord, item, replacement, orchestrator.testAttestationPolicy)
	if attachmentErr != nil {
		return attachmentErr
	}
	action, err := orchestrator.reviewerLaunchAction(policy, retryRecord, item, replacement, authority, at,
		at.Add(executionRetryBackoff(orchestrator.observationDelay, execution.AttemptNo, orchestrator.executionTimeout)), attachment)
	if err != nil {
		return err
	}
	events := []EventRecord{
		{Ref: "event:review-attempt-failed:" + execution.Ref.String(), Kind: "review.attempt_failed",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at.UTC()},
		{Ref: "event:review-queued:" + replacement.Ref.String(), Kind: "review.queued",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacement.Ref, OccurredAt: at.UTC()},
	}
	return orchestrator.state.RecordReviewExecutionReplaced(ctx, ReviewExecutionReplacedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		FailedExecution: execution, ReplacementExecution: replacement, NextAction: action,
		DiagnosticArtifact: diagnostic,
		BudgetSettlement:   settlement, Events: events, OperationAt: at.UTC(),
	})
}

func reviewerReplacementExecution(execution ExecutionRecord, at time.Time) (ExecutionRecord, error) {
	role, ok := reviewerRole(execution)
	if !ok {
		return ExecutionRecord{}, errors.New("review.execution_invalid")
	}
	ref, err := goal.NewExecutionRef("execution:review:" + fingerprintFields("orquesta.review-execution.v1",
		execution.GoalRef.String(), execution.WorkItemRef.String(), execution.ReviewSubjectDigest, string(role),
		strconv.FormatUint(execution.AttemptNo+1, 10)))
	if err != nil {
		return ExecutionRecord{}, err
	}
	return ExecutionRecord{
		Ref: ref, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		AttemptNo: execution.AttemptNo + 1, MaxExecutionAttempts: execution.MaxExecutionAttempts,
		ReplacesExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		State: ExecutionQueued, Purpose: execution.Purpose, ReviewSubjectDigest: execution.ReviewSubjectDigest,
		RepositoryRef: execution.RepositoryRef, ExecutionWorkspaceRef: execution.ExecutionWorkspaceRef,
		ArtifactMediaType: review.AssessmentMediaType, IdempotencyKey: "execution:" + ref.String(),
		MaxOutputBytes: execution.MaxOutputBytes, CreatedAt: at.UTC(),
	}, nil
}
