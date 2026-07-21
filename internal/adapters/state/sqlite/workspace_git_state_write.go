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
		return insertEvent(ctx, tx, s.Event)
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
