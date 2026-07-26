package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

func (repository *Repository) ConfirmIntakeDossierAndCreateGoal(
	ctx context.Context,
	state application.ConfirmIntakeDossierState,
) (application.IntakeDossierConfirmationRecord, bool, error) {
	if err := validateCreateState(state.CreateGoal); err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, invalid(err)
	}
	if err := application.ValidateIntakeDossierConfirmationState(state); err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	record, found, err := readIntakeDossierConfirmationReplay(
		ctx, transaction, state.Confirmation,
	)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.IntakeDossierConfirmationRecord{}, false, err
		}
		return record, false, nil
	}

	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.CreateGoal.AuthorizationReceipt,
		state.Confirmation.PrincipalRef, state.Confirmation.ProjectRef,
		identity.PermissionGoalsCreate, state.Confirmation.ProjectRef.String(),
	); err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	persistedDossier, err := readPersistedIntakeDossierForConfirmation(
		ctx, transaction, state.Confirmation.ActorRef,
		state.Confirmation.ProjectRef, state.Confirmation.DossierRef,
	)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	if !reflect.DeepEqual(
		application.SnapshotIntakeDossier(persistedDossier.Dossier),
		application.SnapshotIntakeDossier(state.Dossier),
	) {
		return application.IntakeDossierConfirmationRecord{}, false, conflict(
			errors.New("sqlite.intake_dossier_confirmation_dossier_conflict"),
		)
	}
	current, err := readCurrentIntakeRecord(
		ctx, transaction, state.Confirmation.ActorRef,
		state.Confirmation.ProjectRef, state.Confirmation.StateRef,
	)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	if current.State.Revision() != state.Confirmation.StateRevision ||
		current.Receipt.StateDigest != state.Confirmation.StateDigest ||
		current.Receipt.Ref != state.Confirmation.SourceIntakeReceiptRef {
		return application.IntakeDossierConfirmationRecord{}, false, conflict(
			errors.New("sqlite.intake_dossier_confirmation_source_conflict"),
		)
	}
	if frozen, err := intakeDossierConfirmed(
		ctx, transaction, state.Confirmation.ActorRef,
		state.Confirmation.ProjectRef, state.Confirmation.StateRef,
	); err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	} else if frozen {
		return application.IntakeDossierConfirmationRecord{}, false, conflict(
			errors.New("sqlite.intake_dossier_confirmed"),
		)
	}

	if err := insertCreateState(ctx, transaction, state.CreateGoal); err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	if err := insertIntakeDossierConfirmation(
		ctx, transaction, state.Confirmation,
	); err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	record, err = readIntakeDossierConfirmationCommit(
		ctx, transaction, state.Confirmation.Ref,
	)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	if record.Confirmation != state.Confirmation ||
		!reflect.DeepEqual(
			record.Goal.Goal.Snapshot(), state.CreateGoal.Goal.Snapshot(),
		) ||
		!reflect.DeepEqual(record.Goal.Executions, state.CreateGoal.Executions) {
		return application.IntakeDossierConfirmationRecord{}, false, invalid(
			errors.New("sqlite.intake_dossier_confirmation_candidate_conflict"),
		)
	}
	if err := commit(transaction); err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	return record, true, nil
}

func insertIntakeDossierConfirmation(
	ctx context.Context,
	tx *sql.Tx,
	confirmation application.IntakeDossierConfirmation,
) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO intake_dossier_confirmations(
    ref, request_ref, request_fingerprint, principal_ref, actor_ref, project_ref,
    state_ref, state_revision, state_digest, source_intake_receipt_ref,
    dossier_ref, dossier_digest, plan_digest, goal_ref, app_spec_ref, spec_hash,
    authorization_receipt_ref, confirmed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		confirmation.Ref, confirmation.RequestRef, confirmation.RequestFingerprint,
		confirmation.PrincipalRef.String(), confirmation.ActorRef.String(),
		confirmation.ProjectRef.String(), confirmation.StateRef,
		int64(confirmation.StateRevision), confirmation.StateDigest,
		confirmation.SourceIntakeReceiptRef, confirmation.DossierRef,
		confirmation.DossierDigest, confirmation.PlanDigest,
		confirmation.GoalRef.String(), confirmation.AppSpecRef.String(),
		confirmation.SpecHash, confirmation.AuthorizationReceiptRef,
		requiredTime(confirmation.ConfirmedAt),
	)
	return mapDatabaseError(err)
}

func readIntakeDossierConfirmationReplay(
	ctx context.Context,
	source queryer,
	want application.IntakeDossierConfirmation,
) (application.IntakeDossierConfirmationRecord, bool, error) {
	var ref string
	err := source.QueryRowContext(ctx, `
SELECT ref
FROM intake_dossier_confirmations
WHERE principal_ref = ? AND project_ref = ? AND request_ref = ?`,
		want.PrincipalRef.String(), want.ProjectRef.String(), want.RequestRef,
	).Scan(&ref)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return application.IntakeDossierConfirmationRecord{}, false, nil
	case err != nil:
		return application.IntakeDossierConfirmationRecord{}, false, mapDatabaseError(err)
	}
	result, err := readIntakeDossierConfirmationCommit(ctx, source, ref)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, false, err
	}
	got := result.Confirmation
	if got.RequestFingerprint != want.RequestFingerprint ||
		got.PrincipalRef != want.PrincipalRef ||
		got.ActorRef != want.ActorRef ||
		got.ProjectRef != want.ProjectRef ||
		got.StateRef != want.StateRef ||
		got.StateRevision != want.StateRevision ||
		got.StateDigest != want.StateDigest ||
		got.SourceIntakeReceiptRef != want.SourceIntakeReceiptRef ||
		got.DossierRef != want.DossierRef ||
		got.DossierDigest != want.DossierDigest ||
		got.PlanDigest != want.PlanDigest ||
		got.AuthorizationReceiptRef != want.AuthorizationReceiptRef {
		return application.IntakeDossierConfirmationRecord{}, false, conflict(
			errors.New("sqlite.intake_dossier_confirmation_replay_conflict"),
		)
	}
	return result, true, nil
}

func readIntakeDossierConfirmationCommit(
	ctx context.Context,
	source queryer,
	ref string,
) (application.IntakeDossierConfirmationRecord, error) {
	var confirmation application.IntakeDossierConfirmation
	var requestedBy, actorValue, projectValue, stateValue string
	var dossierValue, goalValue, appSpecValue string
	var stateRevision, confirmedAt int64
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, principal_ref, actor_ref, project_ref,
       state_ref, state_revision, state_digest, source_intake_receipt_ref,
       dossier_ref, dossier_digest, plan_digest, goal_ref, app_spec_ref, spec_hash,
       authorization_receipt_ref, confirmed_at
FROM intake_dossier_confirmations
WHERE ref = ?`, ref).Scan(
		&confirmation.Ref, &confirmation.RequestRef, &confirmation.RequestFingerprint,
		&requestedBy, &actorValue, &projectValue, &stateValue, &stateRevision,
		&confirmation.StateDigest, &confirmation.SourceIntakeReceiptRef,
		&dossierValue, &confirmation.DossierDigest, &confirmation.PlanDigest,
		&goalValue, &appSpecValue, &confirmation.SpecHash,
		&confirmation.AuthorizationReceiptRef, &confirmedAt,
	)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, mapDatabaseError(err)
	}
	var restoreErr error
	if confirmation.PrincipalRef, restoreErr = identity.NewPrincipalRef(requestedBy); restoreErr != nil {
		return application.IntakeDossierConfirmationRecord{}, invalid(restoreErr)
	}
	if confirmation.ActorRef, restoreErr = goal.NewActorRef(actorValue); restoreErr != nil {
		return application.IntakeDossierConfirmationRecord{}, invalid(restoreErr)
	}
	if confirmation.ProjectRef, restoreErr = goal.NewProjectRef(projectValue); restoreErr != nil {
		return application.IntakeDossierConfirmationRecord{}, invalid(restoreErr)
	}
	if stateRevision <= 0 || confirmedAt <= 0 {
		return application.IntakeDossierConfirmationRecord{}, invalid(
			errors.New("sqlite.intake_dossier_confirmation_values_invalid"),
		)
	}
	confirmation.StateRef = intake.Ref(stateValue)
	confirmation.StateRevision = intake.Revision(stateRevision)
	confirmation.DossierRef = application.IntakeDossierRef(dossierValue)
	if confirmation.GoalRef, restoreErr = goal.NewGoalRef(goalValue); restoreErr != nil {
		return application.IntakeDossierConfirmationRecord{}, invalid(restoreErr)
	}
	if confirmation.AppSpecRef, restoreErr = goal.NewAppSpecRef(appSpecValue); restoreErr != nil {
		return application.IntakeDossierConfirmationRecord{}, invalid(restoreErr)
	}
	confirmation.ConfirmedAt = time.Unix(0, confirmedAt).UTC()
	record, err := readGoalRecord(ctx, source, goalValue)
	if err != nil {
		return application.IntakeDossierConfirmationRecord{}, err
	}
	return application.IntakeDossierConfirmationRecord{
		Goal: record, Confirmation: confirmation,
	}, nil
}

func readPersistedIntakeDossierForConfirmation(
	ctx context.Context,
	source queryer,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	dossierRef application.IntakeDossierRef,
) (application.IntakeDossierRecord, error) {
	var receiptRef string
	err := source.QueryRowContext(ctx, `
SELECT generation_receipt_ref
FROM intake_dossiers
WHERE actor_ref = ? AND project_ref = ? AND ref = ?`,
		actorRef.String(), projectRef.String(), dossierRef,
	).Scan(&receiptRef)
	if err != nil {
		return application.IntakeDossierRecord{}, mapDatabaseError(err)
	}
	return readIntakeDossierRecordByReceipt(ctx, source, receiptRef)
}

func intakeDossierConfirmed(
	ctx context.Context,
	source queryer,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
) (bool, error) {
	var count int
	err := source.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM intake_dossier_confirmations
WHERE actor_ref = ? AND project_ref = ? AND state_ref = ?`,
		actorRef.String(), projectRef.String(), stateRef,
	).Scan(&count)
	if err != nil {
		return false, mapDatabaseError(err)
	}
	return count != 0, nil
}
