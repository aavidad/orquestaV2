package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/intake"
)

func validateRecoveryV23WizardGapsOutcomes(
	ctx context.Context,
	tx *sql.Tx,
) error {
	var invalid int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM wizard_gaps_outcomes outcome
LEFT JOIN intake_receipts source
  ON source.ref = outcome.source_intake_receipt_ref
LEFT JOIN authorization_receipts authorization
  ON authorization.ref = outcome.authorization_receipt_ref
LEFT JOIN principals principal
  ON principal.ref = authorization.principal_ref
WHERE source.ref IS NULL
   OR source.actor_ref <> outcome.actor_ref
   OR source.project_ref <> outcome.project_ref
   OR source.state_ref <> outcome.state_ref
   OR source.revision <> outcome.expected_revision
   OR authorization.ref IS NULL
   OR authorization.request_ref <>
      'authorization-request:intake-apply:' || outcome.request_ref
   OR authorization.project_ref <> outcome.project_ref
   OR authorization.permission <> 'goals.create'
   OR authorization.resource_ref <> outcome.project_ref
   OR authorization.outcome <> 'allowed'
   OR principal.actor_ref <> outcome.actor_ref
   OR EXISTS (
      SELECT 1
      FROM intake_receipts mutation
      WHERE mutation.actor_ref = outcome.actor_ref
        AND mutation.project_ref = outcome.project_ref
        AND mutation.request_ref = outcome.request_ref
   )`).Scan(&invalid); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v23_wizard_gaps_binding_invalid")
	}

	rows, err := tx.QueryContext(ctx, `
SELECT actor_ref, project_ref, request_ref, request_fingerprint, state_ref,
       expected_revision, evaluator_schema, evaluator_version,
       evaluator_semantic_digest, authorization_receipt_ref
FROM wizard_gaps_outcomes
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
	for _, record := range records {
		actorRef, err := goal.NewActorRef(record.actor)
		if err != nil {
			return errors.New("sqlite.recovery_v23_wizard_gaps_scope_invalid")
		}
		projectRef, err := goal.NewProjectRef(record.project)
		if err != nil || record.revision <= 0 {
			return errors.New("sqlite.recovery_v23_wizard_gaps_scope_invalid")
		}
		request := application.WizardGapsNoOpReplayRequest{
			RequestRef: record.requestRef, RequestFingerprint: record.fingerprint,
			ActorRef: actorRef, ProjectRef: projectRef,
			StateRef:                intake.Ref(record.stateRef),
			ExpectedRevision:        intake.Revision(record.revision),
			EvaluatorIdentity:       record.evaluator,
			AuthorizationReceiptRef: record.authorizationRef,
		}
		outcome, found, err := readWizardGapsNoOp(ctx, tx, request)
		if err != nil || !found {
			return errors.New(
				"sqlite.recovery_v23_wizard_gaps_outcome_invalid",
			)
		}
		authorization, err := readAuthorizationReceipt(
			ctx, tx, outcome.AuthorizationReceiptRef,
		)
		if err != nil || validateIntakeAuthorization(
			authorization,
			outcome.ActorRef,
			outcome.ProjectRef,
			outcome.AuthorizationReceiptRef,
			application.IntakeOperationApply,
			outcome.RequestRef,
		) != nil {
			return errors.New(
				"sqlite.recovery_v23_wizard_gaps_authorization_invalid",
			)
		}
	}
	return nil
}
