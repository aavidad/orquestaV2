package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/intake"
)

func validateRecoveryV23WizardGapsInputs(
	ctx context.Context,
	tx *sql.Tx,
) error {
	var invalid int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM wizard_gaps_input_receipts input
LEFT JOIN intake_receipts source
  ON source.ref = input.source_intake_receipt_ref
LEFT JOIN authorization_receipts authorization
  ON authorization.ref = input.authorization_receipt_ref
LEFT JOIN principals principal
  ON principal.ref = authorization.principal_ref
LEFT JOIN intake_receipts mutation
  ON input.outcome_kind = 'intake_mutation'
 AND mutation.ref = input.outcome_receipt_ref
LEFT JOIN wizard_gaps_outcomes noop
  ON input.outcome_kind = 'wizard_gaps_noop'
 AND noop.ref = input.outcome_receipt_ref
WHERE source.ref IS NULL
   OR source.actor_ref <> input.actor_ref
   OR source.project_ref <> input.project_ref
   OR source.state_ref <> input.state_ref
   OR source.revision <> input.expected_revision
   OR authorization.ref IS NULL
   OR authorization.request_ref <>
      'authorization-request:intake-apply:' || input.request_ref
   OR authorization.project_ref <> input.project_ref
   OR authorization.permission <> 'goals.create'
   OR authorization.resource_ref <> input.project_ref
   OR authorization.outcome <> 'allowed'
   OR principal.actor_ref <> input.actor_ref
   OR (
      input.outcome_kind = 'intake_mutation'
      AND (
         mutation.ref IS NULL
         OR mutation.request_ref <> input.request_ref
         OR mutation.actor_ref <> input.actor_ref
         OR mutation.project_ref <> input.project_ref
         OR mutation.state_ref <> input.state_ref
         OR mutation.previous_revision <> input.expected_revision
         OR mutation.revision <> input.expected_revision + 1
         OR mutation.authorization_receipt_ref <>
            input.authorization_receipt_ref
      )
   )
   OR (
      input.outcome_kind = 'wizard_gaps_noop'
      AND (
         noop.ref IS NULL
         OR noop.request_ref <> input.request_ref
         OR noop.actor_ref <> input.actor_ref
         OR noop.project_ref <> input.project_ref
         OR noop.state_ref <> input.state_ref
         OR noop.expected_revision <> input.expected_revision
         OR noop.source_intake_receipt_ref <>
            input.source_intake_receipt_ref
         OR noop.authorization_receipt_ref <>
            input.authorization_receipt_ref
      )
   )`).Scan(&invalid); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v23_wizard_gaps_input_binding_invalid")
	}

	rows, err := tx.QueryContext(ctx, `
SELECT actor_ref, project_ref, request_ref, request_fingerprint, state_ref,
       expected_revision, evaluator_schema, evaluator_version,
       evaluator_semantic_digest, authorization_receipt_ref
FROM wizard_gaps_input_receipts
ORDER BY actor_ref, project_ref, request_ref`)
	if err != nil {
		return err
	}
	type replayRow struct {
		actor, project, requestRef, fingerprint, stateRef string
		revision                                          int64
		evaluator                                         intake.DerivationIdentity
		authorizationRef                                  string
	}
	var records []replayRow
	for rows.Next() {
		var record replayRow
		if err := rows.Scan(
			&record.actor, &record.project, &record.requestRef,
			&record.fingerprint, &record.stateRef, &record.revision,
			&record.evaluator.Schema, &record.evaluator.Version,
			&record.evaluator.SemanticDigest, &record.authorizationRef,
		); err != nil {
			_ = rows.Close()
			return err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, persisted := range records {
		actorRef, err := goal.NewActorRef(persisted.actor)
		if err != nil {
			return errors.New("sqlite.recovery_v23_wizard_gaps_input_scope_invalid")
		}
		projectRef, err := goal.NewProjectRef(persisted.project)
		if err != nil || persisted.revision <= 0 {
			return errors.New("sqlite.recovery_v23_wizard_gaps_input_scope_invalid")
		}
		request := application.WizardGapsInputReplayRequest{
			RequestRef:         persisted.requestRef,
			RequestFingerprint: persisted.fingerprint,
			ActorRef:           actorRef, ProjectRef: projectRef,
			StateRef:                intake.Ref(persisted.stateRef),
			ExpectedRevision:        intake.Revision(persisted.revision),
			EvaluatorIdentity:       persisted.evaluator,
			AuthorizationReceiptRef: persisted.authorizationRef,
		}
		record, found, err := readWizardGapsInput(ctx, tx, request)
		if err != nil || !found {
			return errors.New("sqlite.recovery_v23_wizard_gaps_input_invalid")
		}
		if err := application.ValidateWizardGapsInputEvaluation(record); err != nil {
			return errors.New(
				"sqlite.recovery_v23_wizard_gaps_input_evaluation_invalid",
			)
		}
		authorization, err := readAuthorizationReceipt(
			ctx, tx, record.Receipt.AuthorizationReceiptRef,
		)
		if err != nil || validateIntakeAuthorization(
			authorization,
			record.Receipt.ActorRef,
			record.Receipt.ProjectRef,
			record.Receipt.AuthorizationReceiptRef,
			application.IntakeOperationApply,
			record.Receipt.RequestRef,
		) != nil {
			return errors.New(
				"sqlite.recovery_v23_wizard_gaps_input_authorization_invalid",
			)
		}
	}
	return nil
}
