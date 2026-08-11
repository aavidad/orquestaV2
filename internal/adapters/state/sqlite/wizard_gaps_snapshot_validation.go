package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/intake"
)

func validateRecoveryV23WizardGapsSnapshots(ctx context.Context, tx *sql.Tx) error {
	var invalid int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM wizard_gaps_result_snapshots snapshot
LEFT JOIN wizard_gaps_input_receipts input ON input.ref=snapshot.input_receipt_ref
WHERE input.ref IS NULL`).Scan(&invalid); err != nil {
		return err
	}
	if invalid != 0 {
		return errors.New("sqlite.recovery_v23_wizard_gaps_snapshot_binding_invalid")
	}

	rows, err := tx.QueryContext(ctx, `
SELECT input.actor_ref,input.project_ref,input.request_ref,input.request_fingerprint,
       input.state_ref,input.expected_revision,input.evaluator_schema,
       input.evaluator_version,input.evaluator_semantic_digest,
       input.authorization_receipt_ref,snapshot.expected_revision,
       snapshot.evaluator_schema,snapshot.evaluator_version,
       snapshot.evaluator_semantic_digest
FROM wizard_gaps_result_snapshots snapshot
JOIN wizard_gaps_input_receipts input ON input.ref=snapshot.input_receipt_ref
ORDER BY snapshot.ref`)
	if err != nil {
		return err
	}
	type persistedSnapshot struct {
		actor, project, requestRef, fingerprint, stateRef string
		revision                                          int64
		evaluator                                         intake.DerivationIdentity
		authorizationRef                                  string
		snapshotRevision                                  int64
		snapshotEvaluator                                 intake.DerivationIdentity
	}
	var persisted []persistedSnapshot
	for rows.Next() {
		var value persistedSnapshot
		if err := rows.Scan(
			&value.actor, &value.project, &value.requestRef, &value.fingerprint,
			&value.stateRef, &value.revision, &value.evaluator.Schema,
			&value.evaluator.Version, &value.evaluator.SemanticDigest,
			&value.authorizationRef, &value.snapshotRevision,
			&value.snapshotEvaluator.Schema, &value.snapshotEvaluator.Version,
			&value.snapshotEvaluator.SemanticDigest,
		); err != nil {
			_ = rows.Close()
			return err
		}
		persisted = append(persisted, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, value := range persisted {
		actor, actorErr := goal.NewActorRef(value.actor)
		project, projectErr := goal.NewProjectRef(value.project)
		if actorErr != nil || projectErr != nil || value.revision <= 0 {
			return errors.New("sqlite.recovery_v23_wizard_gaps_snapshot_binding_invalid")
		}
		request := application.WizardGapsInputReplayRequest{
			RequestRef: value.requestRef, RequestFingerprint: value.fingerprint,
			ActorRef: actor, ProjectRef: project, StateRef: intake.Ref(value.stateRef),
			ExpectedRevision: intake.Revision(value.revision), EvaluatorIdentity: value.evaluator,
			AuthorizationReceiptRef: value.authorizationRef,
		}
		withoutSnapshot, found, err := readWizardGapsInputWithoutResultSnapshot(ctx, tx, request)
		if err != nil || !found || application.ValidateWizardGapsInputEvaluation(withoutSnapshot) != nil {
			continue
		}
		if value.snapshotRevision != value.revision || value.snapshotEvaluator != value.evaluator {
			return errors.New("sqlite.recovery_v23_wizard_gaps_snapshot_binding_invalid")
		}
		record, found, err := readWizardGapsInput(ctx, tx, request)
		if err != nil || !found || record.Receipt.ResultSnapshot.Ref == "" {
			return errors.New("sqlite.recovery_v23_wizard_gaps_snapshot_invalid")
		}
		if application.ValidateWizardGapsInputEvaluation(record) != nil {
			return errors.New("sqlite.recovery_v23_wizard_gaps_snapshot_invalid")
		}
	}
	return nil
}
