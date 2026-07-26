package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/application"
)

func validateRecoveryV23DossierConfirmations(
	ctx context.Context,
	tx *sql.Tx,
) error {
	var invalid int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM intake_dossier_confirmations confirmation
LEFT JOIN principals principal
  ON principal.ref = confirmation.principal_ref
LEFT JOIN authorization_receipts authorization
  ON authorization.ref = confirmation.authorization_receipt_ref
LEFT JOIN intake_dossiers dossier
  ON dossier.ref = confirmation.dossier_ref
LEFT JOIN intake_states state
  ON state.state_ref = confirmation.state_ref
 AND state.actor_ref = confirmation.actor_ref
 AND state.project_ref = confirmation.project_ref
LEFT JOIN goals goal
  ON goal.ref = confirmation.goal_ref
LEFT JOIN app_specs spec
  ON spec.ref = confirmation.app_spec_ref
LEFT JOIN intents intent
  ON intent.ref = spec.intent_ref
WHERE principal.ref IS NULL
   OR principal.actor_ref <> confirmation.actor_ref
   OR authorization.ref IS NULL
   OR authorization.request_ref <>
      'authorization-request:intake-dossier-confirm:' || confirmation.request_ref
   OR authorization.principal_ref <> confirmation.principal_ref
   OR authorization.project_ref <> confirmation.project_ref
   OR authorization.permission <> 'goals.create'
   OR authorization.resource_ref <> confirmation.project_ref
   OR authorization.outcome <> 'allowed'
   OR dossier.ref IS NULL
   OR dossier.actor_ref <> confirmation.actor_ref
   OR dossier.project_ref <> confirmation.project_ref
   OR dossier.state_ref <> confirmation.state_ref
   OR dossier.state_revision <> confirmation.state_revision
   OR dossier.state_digest <> confirmation.state_digest
   OR dossier.source_intake_receipt_ref <> confirmation.source_intake_receipt_ref
   OR dossier.dossier_digest <> confirmation.dossier_digest
   OR dossier.plan_digest <> confirmation.plan_digest
   OR state.state_ref IS NULL
   OR state.revision <> confirmation.state_revision
   OR state.state_digest <> confirmation.state_digest
   OR state.receipt_ref <> confirmation.source_intake_receipt_ref
   OR goal.ref IS NULL
   OR goal.request_ref <> confirmation.request_ref
   OR goal.request_fingerprint <> confirmation.request_fingerprint
   OR goal.requested_by_ref <> confirmation.principal_ref
   OR goal.actor_ref <> confirmation.actor_ref
   OR goal.project_ref <> confirmation.project_ref
   OR goal.app_spec_ref <> confirmation.app_spec_ref
   OR goal.plan_generation < 1
   OR spec.ref IS NULL
   OR spec.generation <> 1
   OR spec.parent_ref IS NOT NULL
   OR spec.parent_hash IS NOT NULL
   OR spec.hash <> confirmation.spec_hash
   OR spec.confirmed_by <> confirmation.actor_ref
   OR spec.confirmed_at <> confirmation.confirmed_at
   OR spec.reason <>
      'operator.intake_dossier_confirmation:' || confirmation.dossier_ref
   OR intent.ref IS NULL
   OR intent.actor_ref <> confirmation.actor_ref
   OR intent.project_ref <> confirmation.project_ref
   OR intent.statement <> json_extract(dossier.snapshot_json, '$.statement')
   OR spec.objective <> json_extract(dossier.snapshot_json, '$.objective')
   OR EXISTS (
       SELECT 1
       FROM intake_receipts later
       WHERE later.actor_ref = confirmation.actor_ref
         AND later.project_ref = confirmation.project_ref
         AND later.state_ref = confirmation.state_ref
         AND later.revision > confirmation.state_revision
   )`).Scan(&invalid); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v23_intake_dossier_confirmation_binding_invalid")
	}

	rows, err := tx.QueryContext(ctx, `
SELECT ref
FROM intake_dossier_confirmations
ORDER BY principal_ref, project_ref, request_ref, ref`)
	if err != nil {
		return err
	}
	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			_ = rows.Close()
			return err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, ref := range refs {
		commit, err := readIntakeDossierConfirmationCommit(ctx, tx, ref)
		if err != nil {
			return fmt.Errorf(
				"sqlite.recovery_v23_intake_dossier_confirmation_record_invalid: %w",
				err,
			)
		}
		dossier, err := readPersistedIntakeDossierForConfirmation(
			ctx, tx, commit.Confirmation.ActorRef,
			commit.Confirmation.ProjectRef, commit.Confirmation.DossierRef,
		)
		if err != nil {
			return fmt.Errorf(
				"sqlite.recovery_v23_intake_dossier_confirmation_dossier_invalid: %w",
				err,
			)
		}
		authorization, err := readAuthorizationReceipt(
			ctx, tx, commit.Confirmation.AuthorizationReceiptRef,
		)
		if err != nil {
			return fmt.Errorf(
				"sqlite.recovery_v23_intake_dossier_confirmation_authorization_invalid: %w",
				err,
			)
		}
		if err := application.ValidatePersistedIntakeDossierConfirmationRecord(
			dossier.Dossier, authorization, commit,
		); err != nil {
			return fmt.Errorf(
				"sqlite.recovery_v23_intake_dossier_confirmation_semantic_invalid: %w",
				err,
			)
		}
	}
	return nil
}
