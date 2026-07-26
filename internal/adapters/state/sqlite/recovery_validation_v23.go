package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

func validateRecoveryV23Intake(ctx context.Context, tx *sql.Tx) error {
	if err := validateRecoveryV23IntakeChains(ctx, tx); err != nil {
		return err
	}
	type receiptRef struct {
		ref, authorizationRef string
	}
	rows, err := tx.QueryContext(ctx, `
SELECT ref, authorization_receipt_ref
FROM intake_receipts
ORDER BY actor_ref, project_ref, state_ref, revision`)
	if err != nil {
		return err
	}
	var receipts []receiptRef
	for rows.Next() {
		var receipt receiptRef
		if err := rows.Scan(&receipt.ref, &receipt.authorizationRef); err != nil {
			_ = rows.Close()
			return err
		}
		receipts = append(receipts, receipt)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	type intakeScope struct {
		actor, project, state string
	}
	chains := make(map[intakeScope][]application.IntakeRecord)
	var orderedScopes []intakeScope
	for _, ref := range receipts {
		record, err := readIntakeRecordByReceipt(ctx, tx, ref.ref)
		if err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_snapshot_invalid: %w", err)
		}
		scope := intakeScope{
			actor: record.ActorRef.String(), project: record.ProjectRef.String(),
			state: string(record.State.Ref()),
		}
		if _, found := chains[scope]; !found {
			orderedScopes = append(orderedScopes, scope)
		}
		chains[scope] = append(chains[scope], record)
		authorization, err := readAuthorizationReceipt(ctx, tx, ref.authorizationRef)
		if err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_authorization_invalid: %w", err)
		}
		request := authorization.Decision().Request()
		expectedAuthorizationRequestRef, err := application.IntakeAuthorizationRequestRef(
			record.Receipt.Operation, record.Receipt.RequestRef,
		)
		if err != nil ||
			authorization.Ref() != record.Receipt.AuthorizationReceiptRef ||
			request.RequestRef() != expectedAuthorizationRequestRef ||
			request.Principal().ActorRef != record.ActorRef ||
			request.ProjectRef() != record.ProjectRef ||
			request.Permission() != identity.PermissionGoalsCreate ||
			request.ResourceRef() != record.ProjectRef.String() ||
			authorization.Decision().Outcome() != identity.AuthorizationAllowed ||
			!identity.RoleAllows(authorization.Decision().Role(), identity.PermissionGoalsCreate) {
			return errors.New("sqlite.recovery_v23_intake_authorization_invalid")
		}
	}
	for _, scope := range orderedScopes {
		if err := application.ValidateIntakeChain(chains[scope]); err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_chain_invalid: %w", err)
		}
	}

	rows, err = tx.QueryContext(ctx, `
SELECT state_ref, actor_ref, project_ref
FROM intake_states
ORDER BY actor_ref, project_ref, state_ref`)
	if err != nil {
		return err
	}
	type stateScope struct{ state, actor, project string }
	var states []stateScope
	for rows.Next() {
		var state stateScope
		if err := rows.Scan(&state.state, &state.actor, &state.project); err != nil {
			_ = rows.Close()
			return err
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, scope := range states {
		actorRef, err := goal.NewActorRef(scope.actor)
		if err != nil {
			return errors.New("sqlite.recovery_v23_intake_scope_invalid")
		}
		projectRef, err := goal.NewProjectRef(scope.project)
		if err != nil {
			return errors.New("sqlite.recovery_v23_intake_scope_invalid")
		}
		if _, err := readCurrentIntakeRecord(
			ctx, tx, actorRef, projectRef, intake.Ref(scope.state),
		); err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_current_invalid: %w", err)
		}
	}
	return nil
}

func validateRecoveryV23IntakeChains(ctx context.Context, tx *sql.Tx) error {
	var invalid int
	err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM intake_states state
WHERE
 (SELECT COUNT(*) FROM intake_receipts receipt
  WHERE receipt.state_ref=state.state_ref
    AND receipt.actor_ref=state.actor_ref
    AND receipt.project_ref=state.project_ref) <> state.revision
 OR
 (SELECT COUNT(*) FROM intake_receipts receipt
  WHERE receipt.state_ref=state.state_ref
    AND receipt.actor_ref=state.actor_ref
    AND receipt.project_ref=state.project_ref
    AND receipt.operation='create'
    AND receipt.previous_revision=0
    AND receipt.revision=1) <> 1
 OR EXISTS (
  SELECT 1 FROM intake_receipts receipt
  WHERE receipt.state_ref=state.state_ref
    AND receipt.actor_ref=state.actor_ref
    AND receipt.project_ref=state.project_ref
    AND (
      receipt.revision>state.revision
      OR receipt.previous_revision<>receipt.revision-1
      OR (receipt.operation='create' AND receipt.revision<>1)
      OR (receipt.operation='apply' AND receipt.revision<=1)
    )
 )
 OR NOT EXISTS (
  SELECT 1 FROM intake_receipts receipt
  WHERE receipt.ref=state.receipt_ref
    AND receipt.state_ref=state.state_ref
    AND receipt.actor_ref=state.actor_ref
    AND receipt.project_ref=state.project_ref
    AND receipt.revision=state.revision
    AND receipt.state_digest=state.state_digest
    AND receipt.snapshot_json=state.snapshot_json
 )`).Scan(&invalid)
	if err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v23_intake_chain_invalid")
	}
	return nil
}
