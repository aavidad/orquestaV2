package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
)

func (repository *Repository) RecordReviewAssessed(ctx context.Context, state application.ReviewAssessedState) error {
	if _, err := state.Review.Assessment(); err != nil {
		return invalid(err)
	}
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(tx *sql.Tx) error {
		if err := updateGoalCAS(ctx, tx, state.Goal, state.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateGoalWorkItems(ctx, tx, state.Goal); err != nil {
			return err
		}
		if err := updateExecutionCAS(ctx, tx, state.ReviewerExecution, application.ExecutionRunning); err != nil {
			return err
		}
		if state.AuthorExecution.State == application.ExecutionFailed {
			if err := updateExecutionCAS(ctx, tx, state.AuthorExecution, application.ExecutionAwaitingIntegration); err != nil {
				return err
			}
		}
		if err := insertArtifact(ctx, tx, state.Artifact); err != nil {
			return err
		}
		if err := insertReviewRecord(ctx, tx, state.Review); err != nil {
			return err
		}
		if state.AutoOpenCouncil != nil {
			if _, created, err := openCouncilRoundTx(ctx, tx, *state.AutoOpenCouncil, state.OperationAt); err != nil {
				return err
			} else if !created {
				return conflict(errors.New("sqlite.council_auto_open_replay_conflict"))
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

func (repository *Repository) RecordReviewExecutionReplaced(ctx context.Context,
	state application.ReviewExecutionReplacedState,
) error {
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(tx *sql.Tx) error {
		expected := application.ExecutionRunning
		if state.Claim.Action.Kind == application.ActionLaunchAgent {
			expected = application.ExecutionDispatching
		}
		if err := updateExecutionCAS(ctx, tx, state.FailedExecution, expected); err != nil {
			return err
		}
		if state.BudgetSettlement != nil {
			if err := insertBudgetSettlement(ctx, tx, *state.BudgetSettlement); err != nil {
				return err
			}
		}
		if state.DiagnosticArtifact != nil {
			if err := insertArtifact(ctx, tx, *state.DiagnosticArtifact); err != nil {
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

func (repository *Repository) RecordReviewExecutionFailed(ctx context.Context,
	state application.ReviewExecutionFailedState,
) error {
	return repository.mutate(ctx, state.Claim, state.OperationAt, func(tx *sql.Tx) error {
		if state.Goal.Revision() != state.ExpectedGoalRevision {
			if err := updateGoalCAS(ctx, tx, state.Goal, state.ExpectedGoalRevision); err != nil {
				return err
			}
		}
		item, found := state.Goal.WorkItem(state.Claim.Action.WorkItemRef)
		if !found {
			return invalid(errors.New("sqlite.review_work_item_missing"))
		}
		if item.Revision() != state.ExpectedItemRevision {
			if err := updateWorkItemCAS(ctx, tx, item, state.ExpectedItemRevision); err != nil {
				return err
			}
		}
		expected := application.ExecutionRunning
		if state.Claim.Action.Kind == application.ActionLaunchAgent {
			expected = application.ExecutionDispatching
		}
		if err := updateExecutionCAS(ctx, tx, state.Execution, expected); err != nil {
			return err
		}
		if state.AuthorExecution.Ref.String() != "" {
			if err := updateExecutionCAS(ctx, tx, state.AuthorExecution, application.ExecutionAwaitingIntegration); err != nil {
				return err
			}
		}
		for _, retirement := range state.RetiredReviewers {
			if err := updateExecutionCAS(ctx, tx, retirement.Execution, retirement.ExpectedState); err != nil {
				return err
			}
		}
		for _, control := range state.CleanupControls {
			if !application.IsReviewCleanupControl(control) ||
				application.ValidatePersistedControlRecord(control) != nil {
				return invalid(errors.New("sqlite.review_cleanup_control_invalid"))
			}
			if err := validateReviewCleanupControlAuthority(ctx, tx, control); err != nil {
				return invalid(err)
			}
			if err := insertControl(ctx, tx, control); err != nil {
				return err
			}
		}
		if state.ResolvedCleanup != nil {
			if !application.IsReviewCleanupControl(*state.ResolvedCleanup) ||
				state.ResolvedCleanup.Status != application.ControlConfirmed ||
				state.ResolvedCleanup.ExecutionRef != state.Execution.Ref ||
				state.ResolvedCleanup.ExecutionAttempt != state.Execution.AttemptNo {
				return invalid(errors.New("sqlite.review_cleanup_resolution_invalid"))
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
		if state.DiagnosticArtifact != nil {
			if err := insertArtifact(ctx, tx, *state.DiagnosticArtifact); err != nil {
				return err
			}
		}
		if state.BudgetSettlement != nil {
			if err := insertBudgetSettlement(ctx, tx, *state.BudgetSettlement); err != nil {
				return err
			}
		}
		if err := completeClaimWithEffect(ctx, tx, state.Claim, state.OperationAt,
			state.Execution.FailureCode, state.QuarantineClaim, state.EffectReceipt); err != nil {
			return err
		}
		for _, actionRef := range state.RetireActionRefs {
			if err := settleRetiredLaunchReservation(ctx, tx, actionRef, state.OperationAt); err != nil {
				return err
			}
			if err := consumeRetiredAction(ctx, tx, actionRef,
				"review-round:"+state.Execution.Ref.String(), state.OperationAt); err != nil {
				return err
			}
		}
		return insertEvents(ctx, tx, state.Events)
	})
}

func insertReviewRecord(ctx context.Context, tx *sql.Tx, fact application.ReviewRecord) error {
	if fact.GoalRef.String() == "" || fact.WorkItemRef.String() == "" || fact.ChangeSetRef.String() == "" ||
		fact.ReviewerExecutionRef.String() == "" || fact.PrincipalRef.String() == "" || fact.ExternalRef == "" {
		return invalid(errors.New("sqlite.review_record_invalid"))
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO review_records(
ref,goal_ref,work_item_ref,change_set_ref,subject_digest,role,verdict,reviewer_execution_ref,
reviewer_execution_attempt,launch_receipt_ref,principal_ref,agent_ref,external_ref,assessment_artifact_ref,assessment_digest,recorded_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, fact.Ref, fact.GoalRef.String(), fact.WorkItemRef.String(), fact.ChangeSetRef.String(),
		fact.SubjectDigest, string(fact.Role), string(fact.Verdict), fact.ReviewerExecutionRef.String(),
		int64(fact.ReviewerExecutionAttempt), fact.LaunchReceiptRef, fact.PrincipalRef.String(), fact.AgentRef, fact.ExternalRef,
		fact.AssessmentArtifactRef, fact.AssessmentDigest, requiredTime(fact.RecordedAt))
	return mapDatabaseError(err)
}
