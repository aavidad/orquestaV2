package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

func validateRecoveryV23Dossiers(ctx context.Context, tx *sql.Tx) error {
	var invalid int
	if err := tx.QueryRowContext(ctx, `
SELECT
 (SELECT COUNT(*)
  FROM intake_dossiers dossier
  LEFT JOIN intake_dossier_generation_receipts receipt
    ON receipt.ref=dossier.generation_receipt_ref
  WHERE receipt.ref IS NULL
     OR receipt.actor_ref<>dossier.actor_ref
     OR receipt.project_ref<>dossier.project_ref
     OR receipt.state_ref<>dossier.state_ref
     OR receipt.state_revision<>dossier.state_revision
     OR receipt.state_digest<>dossier.state_digest
     OR receipt.source_intake_receipt_ref<>dossier.source_intake_receipt_ref
     OR receipt.dossier_ref<>dossier.ref
     OR receipt.dossier_digest<>dossier.dossier_digest
     OR receipt.plan_digest<>dossier.plan_digest)
 +
 (SELECT COUNT(*)
  FROM intake_dossier_generation_receipts receipt
  LEFT JOIN intake_dossiers dossier ON dossier.ref=receipt.dossier_ref
  WHERE dossier.ref IS NULL
     OR receipt.actor_ref<>dossier.actor_ref
     OR receipt.project_ref<>dossier.project_ref
     OR receipt.state_ref<>dossier.state_ref
     OR receipt.state_revision<>dossier.state_revision
     OR receipt.state_digest<>dossier.state_digest
     OR receipt.source_intake_receipt_ref<>dossier.source_intake_receipt_ref
     OR receipt.dossier_digest<>dossier.dossier_digest
     OR receipt.plan_digest<>dossier.plan_digest)
 +
 (SELECT COUNT(*)
  FROM intake_dossiers dossier
  LEFT JOIN intake_receipts source
    ON source.ref=dossier.source_intake_receipt_ref
  WHERE source.ref IS NULL
     OR source.actor_ref<>dossier.actor_ref
     OR source.project_ref<>dossier.project_ref
     OR source.state_ref<>dossier.state_ref
     OR source.revision<>dossier.state_revision
     OR source.state_digest<>dossier.state_digest)`).Scan(&invalid); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v23_intake_dossier_binding_invalid")
	}
	rows, err := tx.QueryContext(ctx, `
SELECT ref, authorization_receipt_ref
FROM intake_dossier_generation_receipts
ORDER BY actor_ref, project_ref, request_ref, ref`)
	if err != nil {
		return err
	}
	type generationReceiptRef struct {
		ref, authorizationRef string
	}
	var receipts []generationReceiptRef
	for rows.Next() {
		var receipt generationReceiptRef
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
	for _, locator := range receipts {
		record, err := readIntakeDossierRecordByReceipt(ctx, tx, locator.ref)
		if err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_dossier_snapshot_invalid: %w", err)
		}
		source, err := readIntakeRecordByReceipt(
			ctx, tx, record.Dossier.SourceIntakeReceiptRef(),
		)
		if err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_dossier_source_invalid: %w", err)
		}
		rebuilt, err := application.BuildIntakeDossier(
			source,
			record.Dossier.Plan(),
			application.IntakeDossierInput{
				Statement: record.Dossier.Statement(),
				Objective: record.Dossier.Objective(),
				Sections:  record.Dossier.Sections(),
				Diagrams:  record.Dossier.Diagrams(),
				RiskRefs:  record.Dossier.RiskRefs(),
			},
		)
		if err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_dossier_source_invalid: %w", err)
		}
		if rebuilt.Ref() != record.Dossier.Ref() ||
			rebuilt.Digest() != record.Dossier.Digest() {
			return errors.New("sqlite.recovery_v23_intake_dossier_source_invalid")
		}
		authorization, err := readAuthorizationReceipt(ctx, tx, locator.authorizationRef)
		if err != nil {
			return fmt.Errorf("sqlite.recovery_v23_intake_dossier_authorization_invalid: %w", err)
		}
		expectedRequestRef, err := application.IntakeDossierAuthorizationRequestRef(
			record.Receipt.RequestRef,
		)
		request := authorization.Decision().Request()
		if err != nil ||
			authorization.Ref() != record.Receipt.AuthorizationReceiptRef ||
			request.RequestRef() != expectedRequestRef ||
			request.Principal().ActorRef != record.ActorRef ||
			request.ProjectRef() != record.ProjectRef ||
			request.Permission() != identity.PermissionGoalsCreate ||
			request.ResourceRef() != record.ProjectRef.String() ||
			authorization.Decision().Outcome() != identity.AuthorizationAllowed ||
			!identity.RoleAllows(
				authorization.Decision().Role(), identity.PermissionGoalsCreate,
			) {
			return errors.New("sqlite.recovery_v23_intake_dossier_authorization_invalid")
		}
	}
	return nil
}
