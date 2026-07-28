package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

var _ application.WizardGapsOutcomeStore = (*Repository)(nil)

func (repository *Repository) ReplayWizardGapsNoOp(
	ctx context.Context,
	request application.WizardGapsNoOpReplayRequest,
) (application.WizardGapsNoOpOutcome, bool, error) {
	if err := validateWizardGapsNoOpReplayRequest(request); err != nil {
		return application.WizardGapsNoOpOutcome{}, false, invalid(err)
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	outcome, found, err := readWizardGapsNoOp(ctx, transaction, request)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	return outcome, found, nil
}

func (repository *Repository) ReserveWizardGapsNoOp(
	ctx context.Context,
	reservation application.WizardGapsNoOpReservation,
) (application.WizardGapsNoOpOutcome, bool, error) {
	outcome := reservation.Outcome
	if err := validateWizardGapsNoOpReservation(reservation); err != nil {
		return application.WizardGapsNoOpOutcome{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	replayRequest := wizardGapsNoOpReplayRequest(outcome)
	replayed, found, err := readWizardGapsNoOp(
		ctx, transaction, replayRequest,
	)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.WizardGapsNoOpOutcome{}, false, err
		}
		return replayed, false, nil
	}
	if err := requirePersistedIntakeAuthorization(
		ctx,
		transaction,
		reservation.AuthorizationReceipt,
		outcome.ActorRef,
		outcome.ProjectRef,
	); err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	current, err := readCurrentIntakeRecord(
		ctx,
		transaction,
		outcome.ActorRef,
		outcome.ProjectRef,
		outcome.StateRef,
	)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	if current.State.Revision() != outcome.ExpectedRevision ||
		current.Receipt.Ref != outcome.SourceIntakeReceiptRef ||
		current.Receipt != outcome.Record.Receipt {
		return application.WizardGapsNoOpOutcome{}, false, conflict(
			errors.New("sqlite.wizard_gaps_outcome_revision_conflict"),
		)
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO wizard_gaps_outcomes(
    ref, request_ref, request_fingerprint, actor_ref, project_ref, state_ref,
    expected_revision, source_intake_receipt_ref,
    evaluator_schema, evaluator_version, evaluator_semantic_digest,
    evaluation_digest, authorization_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		outcome.Ref, outcome.RequestRef, outcome.RequestFingerprint,
		outcome.ActorRef.String(), outcome.ProjectRef.String(), outcome.StateRef,
		int64(outcome.ExpectedRevision), outcome.SourceIntakeReceiptRef,
		outcome.EvaluatorIdentity.Schema, outcome.EvaluatorIdentity.Version,
		outcome.EvaluatorIdentity.SemanticDigest, outcome.EvaluationDigest,
		outcome.AuthorizationReceiptRef,
	)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, mapDatabaseError(err)
	}
	persisted, found, err := readWizardGapsNoOp(
		ctx, transaction, replayRequest,
	)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	if !found {
		return application.WizardGapsNoOpOutcome{}, false, invalid(
			errors.New("sqlite.wizard_gaps_outcome_missing_after_insert"),
		)
	}
	if err := commit(transaction); err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	return persisted, true, nil
}

func readWizardGapsNoOp(
	ctx context.Context,
	source queryer,
	request application.WizardGapsNoOpReplayRequest,
) (application.WizardGapsNoOpOutcome, bool, error) {
	var outcome application.WizardGapsNoOpOutcome
	var actorRef, projectRef, stateRef string
	var expectedRevision int64
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, actor_ref, project_ref, state_ref,
       expected_revision, source_intake_receipt_ref,
       evaluator_schema, evaluator_version, evaluator_semantic_digest,
       evaluation_digest, authorization_receipt_ref
FROM wizard_gaps_outcomes
WHERE actor_ref = ? AND project_ref = ? AND request_ref = ?`,
		request.ActorRef.String(), request.ProjectRef.String(), request.RequestRef,
	).Scan(
		&outcome.Ref, &outcome.RequestRef, &outcome.RequestFingerprint,
		&actorRef, &projectRef, &stateRef, &expectedRevision,
		&outcome.SourceIntakeReceiptRef,
		&outcome.EvaluatorIdentity.Schema, &outcome.EvaluatorIdentity.Version,
		&outcome.EvaluatorIdentity.SemanticDigest, &outcome.EvaluationDigest,
		&outcome.AuthorizationReceiptRef,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return application.WizardGapsNoOpOutcome{}, false, nil
	case err != nil:
		return application.WizardGapsNoOpOutcome{}, false, mapDatabaseError(err)
	case expectedRevision <= 0 || uint64(expectedRevision) > maxSQLiteInteger:
		return application.WizardGapsNoOpOutcome{}, false, invalid(
			errors.New("sqlite.wizard_gaps_outcome_revision_invalid"),
		)
	}
	outcome.ActorRef = request.ActorRef
	outcome.ProjectRef = request.ProjectRef
	outcome.StateRef = intake.Ref(stateRef)
	outcome.ExpectedRevision = intake.Revision(expectedRevision)
	if actorRef != request.ActorRef.String() ||
		projectRef != request.ProjectRef.String() ||
		outcome.RequestFingerprint != request.RequestFingerprint ||
		outcome.StateRef != request.StateRef ||
		outcome.ExpectedRevision != request.ExpectedRevision ||
		outcome.EvaluatorIdentity != request.EvaluatorIdentity ||
		outcome.AuthorizationReceiptRef != request.AuthorizationReceiptRef {
		return application.WizardGapsNoOpOutcome{}, false, conflict(
			errors.New("sqlite.wizard_gaps_outcome_replay_conflict"),
		)
	}
	outcome.Record, err = readIntakeRecordByReceipt(
		ctx, source, outcome.SourceIntakeReceiptRef,
	)
	if err != nil {
		return application.WizardGapsNoOpOutcome{}, false, err
	}
	if err := application.ValidateWizardGapsNoOpOutcomeRecord(outcome); err != nil {
		return application.WizardGapsNoOpOutcome{}, false, invalid(
			errors.Join(errors.New("sqlite.wizard_gaps_outcome_invalid"), err),
		)
	}
	return outcome, true, nil
}

func validateWizardGapsNoOpReservation(
	reservation application.WizardGapsNoOpReservation,
) error {
	outcome := reservation.Outcome
	if err := application.ValidateWizardGapsNoOpOutcomeRecord(outcome); err != nil {
		return err
	}
	if uint64(outcome.ExpectedRevision) > maxSQLiteInteger {
		return errors.New("sqlite.wizard_gaps_outcome_revision_invalid")
	}
	return validateIntakeAuthorization(
		reservation.AuthorizationReceipt,
		outcome.ActorRef,
		outcome.ProjectRef,
		outcome.AuthorizationReceiptRef,
		application.IntakeOperationApply,
		outcome.RequestRef,
	)
}

func validateWizardGapsNoOpReplayRequest(
	request application.WizardGapsNoOpReplayRequest,
) error {
	if !validIntakeRequestRef(request.RequestRef) ||
		!validCanonicalHash(request.RequestFingerprint) ||
		!validText(request.AuthorizationReceiptRef) ||
		request.ExpectedRevision == 0 ||
		uint64(request.ExpectedRevision) > maxSQLiteInteger {
		return errors.New("sqlite.wizard_gaps_outcome_replay_invalid")
	}
	identity, err := intake.NewDerivationIdentity(
		request.EvaluatorIdentity.Schema,
		request.EvaluatorIdentity.Version,
		request.EvaluatorIdentity.SemanticDigest,
	)
	if err != nil || identity != request.EvaluatorIdentity {
		return errors.New("sqlite.wizard_gaps_outcome_evaluator_invalid")
	}
	return validateIntakeScope(
		request.ActorRef, request.ProjectRef, request.StateRef,
	)
}

func wizardGapsNoOpReplayRequest(
	outcome application.WizardGapsNoOpOutcome,
) application.WizardGapsNoOpReplayRequest {
	return application.WizardGapsNoOpReplayRequest{
		RequestRef: outcome.RequestRef, RequestFingerprint: outcome.RequestFingerprint,
		ActorRef: outcome.ActorRef, ProjectRef: outcome.ProjectRef,
		StateRef: outcome.StateRef, ExpectedRevision: outcome.ExpectedRevision,
		EvaluatorIdentity:       outcome.EvaluatorIdentity,
		AuthorizationReceiptRef: outcome.AuthorizationReceiptRef,
	}
}
