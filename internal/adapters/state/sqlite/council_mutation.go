package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (repository *Repository) RecordCouncilContribution(
	ctx context.Context, state application.CouncilContributionState,
) error {
	if err := validateCouncilContributionState(state); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(tx *sql.Tx) error {
		current, err := requireCouncilMutationFrontier(ctx, tx, state.Claim.Action.GoalRef,
			state.Claim.Action.WorkItemRef, state.ExpectedGoalRevision, state.ExpectedItemRevision)
		if err != nil {
			return err
		}
		round, found := councilRoundFor(current.CouncilRounds,
			application.CouncilSubjectDigest(state.Fact.SubjectDigest))
		role, councilExecution := councilExecutionRole(state.Execution)
		if !found || !councilExecution || state.Claim.Action.Kind != application.ActionObserveAgent ||
			state.Execution.Ref != state.Claim.Action.ExecutionRef || state.Execution.State != application.ExecutionSucceeded ||
			state.Execution.GoalRef != state.Claim.Action.GoalRef ||
			state.Execution.WorkItemRef != state.Claim.Action.WorkItemRef ||
			role != state.Fact.Role || state.Execution.AttemptNo != state.Fact.ExecutionAttempt ||
			state.Execution.LaunchReceiptRef != state.Fact.LaunchReceiptRef ||
			state.Execution.ExternalRef != state.Fact.ExternalRef ||
			state.Artifact.Kind != application.ArtifactKindCouncilContribution ||
			state.Artifact.Stored.MediaType != council.ContributionMediaType ||
			state.Artifact.ExecutionRef != state.Execution.Ref ||
			state.Artifact.Stored.Ref.String() != state.Fact.ArtifactRef ||
			state.Artifact.Stored.Digest != state.Fact.ArtifactDigest ||
			state.Fact.SubjectDigest != string(round.SubjectDigest) {
			return invalid(errors.New("sqlite.council_contribution_scope_invalid"))
		}
		if err := application.ValidatePersistedCouncilSubject(current, round.Subject); err != nil {
			return conflict(err)
		}
		for _, prior := range current.CouncilFacts {
			if prior.SubjectDigest == state.Fact.SubjectDigest &&
				(prior.Role == state.Fact.Role || prior.IdempotencyKey == state.Fact.IdempotencyKey) {
				return conflict(errors.New("sqlite.council_contribution_duplicate"))
			}
		}
		facts := append(councilFactsFor(current.CouncilFacts, round.SubjectDigest), state.Fact)
		decision, err := council.Evaluate(round.Subject, facts)
		if err != nil {
			return invalid(err)
		}
		if !councilDecisionMatches(round, decision, state.Decision) {
			return invalid(errors.New("sqlite.council_decision_mismatch"))
		}
		if err := updateExecutionCAS(ctx, tx, state.Execution, application.ExecutionRunning); err != nil {
			return err
		}
		if err := insertArtifact(ctx, tx, state.Artifact); err != nil {
			return err
		}
		if err := insertCouncilFact(ctx, tx, round, state.Fact, state.Artifact.OccurrenceRef, state.OperationAt); err != nil {
			return err
		}
		if state.Decision != nil {
			if err := insertCouncilDecision(ctx, tx, round, *state.Decision); err != nil {
				return err
			}
		}
		if state.BudgetSettlement != nil {
			if err := insertBudgetSettlement(ctx, tx, *state.BudgetSettlement); err != nil {
				return err
			}
		}
		if err := completeClaim(ctx, tx, state.Claim, state.OperationAt, "", false); err != nil {
			return err
		}
		return insertEvents(ctx, tx, state.Events)
	})
}

func (repository *Repository) RecordCouncilExecutionReplaced(
	ctx context.Context, state application.CouncilExecutionReplacedState,
) error {
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(tx *sql.Tx) error {
		current, err := requireCouncilMutationFrontier(ctx, tx, state.Claim.Action.GoalRef,
			state.Claim.Action.WorkItemRef, state.ExpectedGoalRevision, state.ExpectedItemRevision)
		if err != nil {
			return err
		}
		if !validCouncilReplacement(current, state) {
			return invalid(errors.New("sqlite.council_replacement_invalid"))
		}
		expected := failedClaimExpectedExecutionState(state.Claim)
		if err := updateExecutionCAS(ctx, tx, state.FailedExecution, expected); err != nil {
			return err
		}
		if state.BudgetSettlement != nil {
			if err := insertBudgetSettlement(ctx, tx, *state.BudgetSettlement); err != nil {
				return err
			}
		}
		if err := completeClaim(ctx, tx, state.Claim, state.OperationAt, state.FailedExecution.FailureCode, false); err != nil {
			return err
		}
		if err := insertExecution(ctx, tx, state.ReplacementExecution); err != nil {
			return err
		}
		if err := insertAction(ctx, tx, state.NextAction); err != nil {
			return err
		}
		return insertEvents(ctx, tx, state.Events)
	})
}

func (repository *Repository) RecordCouncilExecutionFailed(
	ctx context.Context, state application.CouncilExecutionFailedState,
) error {
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(tx *sql.Tx) error {
		current, err := requireCouncilMutationFrontier(ctx, tx, state.Claim.Action.GoalRef,
			state.Claim.Action.WorkItemRef, state.ExpectedGoalRevision, state.ExpectedItemRevision)
		if err != nil {
			return err
		}
		role, ok := councilExecutionRole(state.Execution)
		round, found := councilRoundFor(current.CouncilRounds, state.Execution.CouncilSubjectDigest)
		expected := failedClaimExpectedExecutionState(state.Claim)
		stored, exists := executionFor(current.Executions, state.Execution.Ref)
		if !ok || !found || role == "" || !exists || stored.State != expected ||
			state.Execution.Ref != state.Claim.Action.ExecutionRef ||
			state.Execution.GoalRef != state.Claim.Action.GoalRef ||
			state.Execution.WorkItemRef != state.Claim.Action.WorkItemRef ||
			state.Execution.State != application.ExecutionFailed || state.Execution.FailureCode == "" ||
			state.Execution.ReviewSubjectDigest != "" || state.Execution.CouncilSubjectDigest != round.SubjectDigest ||
			(state.Claim.Action.Kind != application.ActionLaunchAgent &&
				state.Claim.Action.Kind != application.ActionObserveAgent) ||
			len(councilFactsForRole(current.CouncilFacts, round.SubjectDigest, role)) != 0 {
			return invalid(errors.New("sqlite.council_execution_failure_invalid"))
		}
		if state.Goal.Ref() != current.Goal.Ref() ||
			(state.AuthorExecution.Ref.String() != "" && state.AuthorExecution.State != application.ExecutionFailed) ||
			(state.AuthorExecution.Ref.String() == "" && len(state.CleanupControls) == 0 && state.ResolvedCleanup == nil) {
			return invalid(errors.New("sqlite.council_failure_cleanup_invalid"))
		}
		if state.Goal.Revision() != state.ExpectedGoalRevision {
			if err := updateGoalCAS(ctx, tx, state.Goal, state.ExpectedGoalRevision); err != nil {
				return err
			}
		}
		item, itemFound := state.Goal.WorkItem(state.Claim.Action.WorkItemRef)
		if !itemFound {
			return invalid(errors.New("sqlite.council_work_item_missing"))
		}
		if item.Revision() != state.ExpectedItemRevision {
			if err := updateWorkItemCAS(ctx, tx, item, state.ExpectedItemRevision); err != nil {
				return err
			}
		}
		if err := updateExecutionCAS(ctx, tx, state.Execution, expected); err != nil {
			return err
		}
		if state.AuthorExecution.Ref.String() != "" {
			if err := updateExecutionCAS(ctx, tx, state.AuthorExecution, application.ExecutionAwaitingIntegration); err != nil {
				return err
			}
		}
		for _, retirement := range state.RetiredPeers {
			storedPeer, peerFound := executionFor(current.Executions, retirement.Execution.Ref)
			if !peerFound || storedPeer.State != retirement.ExpectedState ||
				!isCouncilExecutionForSubject(retirement.Execution, round.SubjectDigest) ||
				retirement.Execution.State != application.ExecutionFailed {
				return invalid(errors.New("sqlite.council_retirement_invalid"))
			}
			if err := updateExecutionCAS(ctx, tx, retirement.Execution, retirement.ExpectedState); err != nil {
				return err
			}
		}
		for _, control := range state.CleanupControls {
			if !application.IsCouncilCleanupControl(control) ||
				application.ValidatePersistedControlRecord(control) != nil {
				return invalid(errors.New("sqlite.council_cleanup_control_invalid"))
			}
			if err := validateCouncilCleanupControlAuthority(ctx, tx, control); err != nil {
				return invalid(err)
			}
			if err := insertControl(ctx, tx, control); err != nil {
				return err
			}
		}
		if state.ResolvedCleanup != nil {
			if !application.IsCouncilCleanupControl(*state.ResolvedCleanup) ||
				state.ResolvedCleanup.Status != application.ControlConfirmed ||
				state.ResolvedCleanup.ExecutionRef != state.Execution.Ref ||
				state.ResolvedCleanup.ExecutionAttempt != state.Execution.AttemptNo {
				return invalid(errors.New("sqlite.council_cleanup_resolution_invalid"))
			}
			if err := updateControl(ctx, tx, *state.ResolvedCleanup); err != nil {
				return err
			}
		}
		for _, action := range state.CleanupActions {
			if err := insertAction(ctx, tx, action); err != nil {
				return err
			}
		}
		if state.BudgetSettlement != nil {
			if err := insertBudgetSettlement(ctx, tx, *state.BudgetSettlement); err != nil {
				return err
			}
		}
		if err := completeClaim(ctx, tx, state.Claim, state.OperationAt, state.Execution.FailureCode, false); err != nil {
			return err
		}
		for _, actionRef := range state.RetireActionRefs {
			if err := settleRetiredLaunchReservation(ctx, tx, actionRef, state.OperationAt); err != nil {
				return err
			}
			if err := consumeRetiredAction(ctx, tx, actionRef,
				"council-round:"+state.Execution.Ref.String(), state.OperationAt); err != nil {
				return err
			}
		}
		return insertEvents(ctx, tx, state.Events)
	})
}

func isCouncilExecutionForSubject(execution application.ExecutionRecord,
	subject application.CouncilSubjectDigest,
) bool {
	_, ok := councilExecutionRole(execution)
	return ok && execution.CouncilSubjectDigest == subject && execution.ReviewSubjectDigest == ""
}

func validateCouncilCleanupControlAuthority(ctx context.Context, tx *sql.Tx,
	record application.ControlRecord,
) error {
	var matches int
	err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM executions participant
JOIN executions author ON author.goal_ref=participant.goal_ref
 AND author.work_item_ref=participant.work_item_ref AND author.purpose='author'
JOIN effect_intents intent ON intent.ref=author.effect_intent_ref
JOIN work_item_authorities authority ON authority.goal_ref=participant.goal_ref
 AND authority.work_item_ref=participant.work_item_ref
WHERE participant.goal_ref=? AND participant.work_item_ref=? AND participant.ref=?
 AND participant.purpose IN ('council_proposer','council_critic','council_arbiter')
 AND intent.action_kind='launch_agent' AND intent.kind='agent_launch'
 AND intent.project_ref=? AND intent.goal_ref=participant.goal_ref
 AND intent.work_item_ref=participant.work_item_ref AND intent.execution_ref=author.ref
 AND intent.plan_generation=author.plan_generation
 AND intent.app_spec_generation=author.app_spec_generation AND intent.spec_hash=author.spec_hash
 AND intent.proposed_by_ref=? AND intent.permission=? AND intent.authority_receipt_ref=?
 AND authority.principal_ref=intent.proposed_by_ref AND authority.permission=intent.permission
 AND authority.authorization_receipt_ref=intent.authority_receipt_ref`,
		record.GoalRef.String(), record.WorkItemRef.String(), record.ExecutionRef.String(),
		record.ProjectRef.String(), record.PrincipalRef.String(), string(identity.PermissionGoalsCreate),
		record.AuthorizationReceipt.Ref(),
	).Scan(&matches)
	if err != nil {
		return err
	}
	if matches != 1 {
		return errors.New("sqlite.council_cleanup_authority_invalid")
	}
	return nil
}

func insertCouncilFact(
	ctx context.Context, tx *sql.Tx, round application.CouncilRoundRecord,
	fact council.ContributionFact, occurrenceRef string, recordedAt time.Time,
) error {
	evidence, err := json.Marshal(fact.Contribution.Evidence)
	if err != nil {
		return invalid(err)
	}
	if recordedAt.IsZero() {
		return invalid(errors.New("sqlite.council_fact_time_invalid"))
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO council_facts(
ref,round_ref,goal_ref,work_item_ref,council_subject_digest,role,ballot,contribution_schema,body,evidence_json,
contribution_digest,execution_ref,execution_attempt,launch_receipt_ref,external_ref,artifact_occurrence_ref,
artifact_ref,artifact_digest,idempotency_key,fact_digest,recorded_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"council-fact:"+fact.ExecutionRef, round.Ref, round.GoalRef.String(), round.WorkItemRef.String(),
		fact.SubjectDigest, string(fact.Role), string(fact.Ballot), fact.Contribution.Schema,
		fact.Contribution.Body, string(evidence), fact.Contribution.Digest(), fact.ExecutionRef,
		int64(fact.ExecutionAttempt), fact.LaunchReceiptRef, fact.ExternalRef, occurrenceRef,
		fact.ArtifactRef, fact.ArtifactDigest, fact.IdempotencyKey, fact.Digest(), requiredTime(recordedAt))
	return mapDatabaseError(err)
}

func insertCouncilDecision(
	ctx context.Context, tx *sql.Tx, round application.CouncilRoundRecord, record application.CouncilDecisionRecord,
) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO council_decisions(
ref,round_ref,goal_ref,work_item_ref,subject_digest,outcome,decision_digest,recorded_at)
VALUES(?,?,?,?,?,?,?,?)`, record.Ref, round.Ref, round.GoalRef.String(), round.WorkItemRef.String(),
		string(record.SubjectDigest), string(record.Decision.Outcome), string(record.DecisionDigest),
		requiredTime(record.RecordedAt))
	return mapDatabaseError(err)
}

func validateCouncilContributionState(state application.CouncilContributionState) error {
	if validateClaim(state.Claim) != nil || state.ExpectedGoalRevision == 0 || state.ExpectedItemRevision == 0 ||
		state.OperationAt.IsZero() || validateExecution(state.Execution) != nil ||
		validateArtifactRecord(state.Artifact) != nil {
		return errors.New("sqlite.council_contribution_invalid")
	}
	if _, err := council.NewContributionFact(state.Fact); err != nil {
		return err
	}
	return nil
}

func requireCouncilMutationFrontier(
	ctx context.Context, tx *sql.Tx, goalRef goal.GoalRef, itemRef goal.WorkItemRef,
	expectedGoal goal.Revision, expectedItem goal.Revision,
) (application.GoalRecord, error) {
	current, err := readGoalRecord(ctx, tx, goalRef.String())
	if err != nil {
		return application.GoalRecord{}, err
	}
	item, found := current.Goal.WorkItem(itemRef)
	if !found || current.Goal.Revision() != expectedGoal || item.Revision() != expectedItem {
		return application.GoalRecord{}, conflict(errors.New("sqlite.council_revision_conflict"))
	}
	return current, nil
}

func validCouncilLaunchSet(
	round application.CouncilRoundRecord, executions []application.ExecutionRecord, actions []application.ActionRecord,
) bool {
	if len(executions) != 3 || len(actions) != 3 {
		return false
	}
	seen := map[council.Role]bool{}
	for i, execution := range executions {
		role, ok := councilExecutionRole(execution)
		if !ok || seen[role] || validateExecution(execution) != nil || validateAction(actions[i]) != nil ||
			execution.State != application.ExecutionQueued || execution.GoalRef != round.GoalRef ||
			execution.WorkItemRef != round.WorkItemRef || execution.ReviewSubjectDigest != "" ||
			execution.CouncilSubjectDigest != round.SubjectDigest ||
			execution.ArtifactMediaType != council.ContributionMediaType ||
			actions[i].Kind != application.ActionLaunchAgent || actions[i].ExecutionRef != execution.Ref ||
			actions[i].Ref != "action:launch:"+execution.Ref.String() {
			return false
		}
		seen[role] = true
	}
	return len(seen) == 3
}

func councilDecisionMatches(
	round application.CouncilRoundRecord, evaluated council.Decision, record *application.CouncilDecisionRecord,
) bool {
	if evaluated.Outcome == council.OutcomePending {
		return record == nil
	}
	return record != nil && record.RoundRef == round.Ref && record.SubjectDigest == round.SubjectDigest &&
		record.Decision.SubjectDigest == evaluated.SubjectDigest && record.Decision.Outcome == evaluated.Outcome &&
		record.Decision.Digest == evaluated.Digest && reflect.DeepEqual(record.Decision.Dissent, evaluated.Dissent) &&
		record.DecisionDigest == application.CouncilSubjectDigest(evaluated.Digest) &&
		validText(record.Ref) && !record.RecordedAt.IsZero()
}

func validCouncilReplacement(current application.GoalRecord, state application.CouncilExecutionReplacedState) bool {
	failed, replacement, action := state.FailedExecution, state.ReplacementExecution, state.NextAction
	role, ok := councilExecutionRole(failed)
	round, found := councilRoundFor(current.CouncilRounds, failed.CouncilSubjectDigest)
	stored, exists := executionFor(current.Executions, failed.Ref)
	expected := application.ExecutionRunning
	if state.Claim.Action.Kind == application.ActionLaunchAgent {
		expected = application.ExecutionDispatching
	}
	return ok && found && exists && stored.State == expected && failed.Ref == state.Claim.Action.ExecutionRef &&
		failed.GoalRef == state.Claim.Action.GoalRef && failed.WorkItemRef == state.Claim.Action.WorkItemRef &&
		failed.State == application.ExecutionFailed && failed.FailureCode != "" &&
		failed.ReviewSubjectDigest == "" && failed.CouncilSubjectDigest == round.SubjectDigest &&
		replacement.State == application.ExecutionQueued && replacement.ReplacesExecutionRef == failed.Ref &&
		replacement.Purpose == failed.Purpose && replacement.GoalRef == failed.GoalRef &&
		replacement.WorkItemRef == failed.WorkItemRef && replacement.PlanGeneration == failed.PlanGeneration &&
		replacement.AppSpecGeneration == failed.AppSpecGeneration && replacement.SpecHash == failed.SpecHash &&
		replacement.ArtifactMediaType == failed.ArtifactMediaType &&
		replacement.CouncilSubjectDigest == failed.CouncilSubjectDigest && replacement.ReviewSubjectDigest == "" &&
		replacement.AttemptNo == failed.AttemptNo+1 && replacement.AttemptNo <= replacement.MaxExecutionAttempts &&
		validateExecution(replacement) == nil && validateAction(action) == nil &&
		action.Kind == application.ActionLaunchAgent && action.Ref == "action:launch:"+replacement.Ref.String() &&
		action.GoalRef == replacement.GoalRef && action.WorkItemRef == replacement.WorkItemRef &&
		action.ExecutionRef == replacement.Ref && action.PlanGeneration == replacement.PlanGeneration &&
		action.WorkItemGeneration == state.Claim.Action.WorkItemGeneration &&
		len(councilFactsForRole(current.CouncilFacts, round.SubjectDigest, role)) == 0
}
