package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
)

func (r *Repository) RecordWorkspacePrepared(ctx context.Context, s application.WorkspacePreparedState) error {
	if application.ValidateWorkspaceBinding(s.Binding) != nil ||
		s.Binding.ReceiptRef != s.EffectReceipt.Ref {
		return invalid(errors.New("sqlite.workspace_binding_invalid"))
	}
	return r.mutate(ctx, s.Claim, s.OperationAt, func(tx *sql.Tx) error {
		if err := updateExecutionCAS(ctx, tx, s.Execution, application.ExecutionQueued); err != nil {
			return err
		}
		if err := completeClaimWithEffect(ctx, tx, s.Claim, s.OperationAt, "", false, &s.EffectReceipt); err != nil {
			return err
		}
		if err := insertWorkspaceBinding(ctx, tx, s.Binding); err != nil {
			return err
		}
		if err := insertAction(ctx, tx, s.NextAction); err != nil {
			return err
		}
		return insertEvent(ctx, tx, s.Event)
	})
}
func (r *Repository) RecordExecutionOutputReady(ctx context.Context, s application.ExecutionOutputReadyState) error {
	return r.mutate(ctx, s.Claim, s.OperationAt, func(tx *sql.Tx) error {
		if err := updateExecutionCAS(ctx, tx, s.Execution, application.ExecutionRunning); err != nil {
			return err
		}
		if err := insertArtifact(ctx, tx, s.Artifact); err != nil {
			return err
		}
		if err := insertAttestation(ctx, tx, s.Attestation); err != nil {
			return err
		}
		if s.BudgetSettlement != nil {
			if err := insertBudgetSettlement(ctx, tx, *s.BudgetSettlement); err != nil {
				return err
			}
		}
		if err := completeClaim(ctx, tx, s.Claim, s.OperationAt, "", false); err != nil {
			return err
		}
		if err := insertAction(ctx, tx, s.NextAction); err != nil {
			return err
		}
		return insertEvent(ctx, tx, s.Event)
	})
}
func (r *Repository) RecordChangeCommitted(ctx context.Context, s application.ChangeCommittedState) error {
	if application.ValidateChangeSet(s.ChangeSet) != nil ||
		s.ChangeSet.ReceiptRef != s.EffectReceipt.Ref {
		return invalid(errors.New("sqlite.change_set_invalid"))
	}
	return r.mutate(ctx, s.Claim, s.OperationAt, func(tx *sql.Tx) error {
		if err := updateExecutionCAS(ctx, tx, s.Execution, application.ExecutionAwaitingCommit); err != nil {
			return err
		}
		if err := completeClaimWithEffect(ctx, tx, s.Claim, s.OperationAt, "", false, &s.EffectReceipt); err != nil {
			return err
		}
		if err := insertChangeSet(ctx, tx, s.ChangeSet); err != nil {
			return err
		}
		if err := requireChangeAttestationAction(ctx, tx, s); err != nil {
			return err
		}
		if s.NextAction != nil {
			if err := insertAction(ctx, tx, *s.NextAction); err != nil {
				return err
			}
		}
		return insertEvent(ctx, tx, s.Event)
	})
}

func requireChangeAttestationAction(ctx context.Context, tx *sql.Tx, s application.ChangeCommittedState) error {
	var requiredTests int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM work_item_required_tests WHERE goal_ref=? AND work_item_ref=?`,
		s.ChangeSet.GoalRef.String(), s.ChangeSet.WorkItemRef.String()).Scan(&requiredTests); err != nil {
		return mapDatabaseError(err)
	}
	if (requiredTests > 0) != (s.NextAction != nil) {
		return invalid(errors.New("sqlite.change_attestation_action_required"))
	}
	if s.NextAction == nil {
		return nil
	}
	next := *s.NextAction
	if next.Kind != application.ActionAttestTest || next.GoalRef != s.ChangeSet.GoalRef ||
		next.WorkItemRef != s.ChangeSet.WorkItemRef || next.ExecutionRef != s.ChangeSet.ExecutionRef ||
		next.ChangeRef != s.ChangeSet.Ref || next.PlanGeneration != s.ChangeSet.PlanGeneration ||
		next.WorkItemGeneration != s.Claim.Action.WorkItemGeneration ||
		next.AvailableAt.Before(s.ChangeSet.CommittedAt) {
		return invalid(errors.New("sqlite.change_attestation_action_invalid"))
	}
	return nil
}

func (r *Repository) RecordTestAttested(ctx context.Context, s application.TestAttestedState) error {
	if s.Claim.Action.Kind != application.ActionAttestTest || s.ExpectedGoalRevision == 0 || s.ExpectedItemRevision == 0 {
		return invalid(errors.New("sqlite.test_attested_state_invalid"))
	}
	return r.mutate(ctx, s.Claim, s.OperationAt, func(tx *sql.Tx) error {
		if err := updateGoalCAS(ctx, tx, s.Goal, s.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateGoalWorkItems(ctx, tx, s.Goal); err != nil {
			return err
		}
		if err := updateExecutionCAS(ctx, tx, s.Execution, application.ExecutionAwaitingAttestation); err != nil {
			return err
		}
		if err := insertArtifact(ctx, tx, s.ManifestArtifact); err != nil {
			return err
		}
		if err := insertArtifact(ctx, tx, s.ReportArtifact); err != nil {
			return err
		}
		if err := completeClaimWithEffect(ctx, tx, s.Claim, s.OperationAt, "", false, &s.EffectReceipt); err != nil {
			return err
		}
		if err := insertAttestation(ctx, tx, s.Attestation); err != nil {
			return err
		}
		if err := insertScheduled(ctx, tx, s.NewExecutions, s.NewActions); err != nil {
			return err
		}
		if err := insertEvents(ctx, tx, s.Events); err != nil {
			return err
		}
		_, err := readGoalRecord(ctx, tx, s.Goal.Ref().String())
		return err
	})
}
func (r *Repository) RecordIntegrationResult(ctx context.Context, s application.IntegrationResultState) error {
	if application.ValidateMergeObservation(s.Observation) != nil ||
		application.ValidateIntegrationReceipt(s.Integration) != nil ||
		s.Integration.EffectReceiptRef != s.EffectReceipt.Ref {
		return invalid(errors.New("sqlite.integration_fact_invalid"))
	}
	return r.mutate(ctx, s.Claim, s.OperationAt, func(tx *sql.Tx) error {
		if err := updateGoalCAS(ctx, tx, s.Goal, s.ExpectedGoalRevision); err != nil {
			return err
		}
		if err := updateGoalWorkItems(ctx, tx, s.Goal); err != nil {
			return err
		}
		if err := updateExecutionCAS(ctx, tx, s.Execution, application.ExecutionAwaitingIntegration); err != nil {
			return err
		}
		if err := insertMergeObservation(ctx, tx, s.Observation); err != nil {
			return err
		}
		if err := completeClaimWithEffect(ctx, tx, s.Claim, s.OperationAt, "", false, &s.EffectReceipt); err != nil {
			return err
		}
		if err := insertIntegrationReceipt(ctx, tx, s.Integration, s.Observation.Ref); err != nil {
			return err
		}
		if err := insertScheduled(ctx, tx, s.NewExecutions, s.NewActions); err != nil {
			return err
		}
		return insertEvents(ctx, tx, s.Events)
	})
}
