package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

const intakeDossierGenerationReceiptPrefix = "intake-dossier-generation-receipt:"

var _ application.IntakeDossierStore = (*Repository)(nil)

func (repository *Repository) ReplayIntakeDossier(
	ctx context.Context,
	request application.IntakeDossierReplayRequest,
) (application.IntakeDossierRecord, bool, error) {
	if err := validateIntakeDossierReplayRequest(request); err != nil {
		return application.IntakeDossierRecord{}, false, invalid(err)
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	record, found, err := readIntakeDossierReplay(ctx, transaction, request)
	if err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	return record, found, nil
}

func (repository *Repository) CreateIntakeDossier(
	ctx context.Context,
	state application.IntakeDossierCreateState,
) (application.IntakeDossierRecord, bool, error) {
	snapshotJSON, err := validateIntakeDossierCreateState(state)
	if err != nil {
		return application.IntakeDossierRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	replayRequest := application.IntakeDossierReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		DossierRef: state.Dossier.Ref(), StateRef: state.Dossier.StateRef(),
		ExpectedRevision:        state.ExpectedRevision,
		SourceIntakeReceiptRef:  state.SourceIntakeReceiptRef,
		PlanDigest:              state.Dossier.PlanDigest(),
		AuthorizationReceiptRef: state.Receipt.AuthorizationReceiptRef,
	}
	record, found, err := readIntakeDossierReplay(ctx, transaction, replayRequest)
	if err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.IntakeDossierRecord{}, false, err
		}
		return record, false, nil
	}
	if frozen, err := intakeDossierConfirmed(
		ctx, transaction, state.ActorRef, state.ProjectRef, state.Dossier.StateRef(),
	); err != nil {
		return application.IntakeDossierRecord{}, false, err
	} else if frozen {
		return application.IntakeDossierRecord{}, false, conflict(
			errors.New("sqlite.intake_dossier_confirmed"),
		)
	}
	if err := requirePersistedIntakeDossierAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.ActorRef, state.ProjectRef,
	); err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	current, err := readCurrentIntakeRecord(
		ctx, transaction, state.ActorRef, state.ProjectRef, state.Dossier.StateRef(),
	)
	if err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	if current.State.Revision() != state.Dossier.StateRevision() ||
		current.Receipt.Ref != state.Dossier.SourceIntakeReceiptRef() ||
		current.Receipt.StateDigest != state.Dossier.StateDigest() {
		return application.IntakeDossierRecord{}, false, conflict(
			errors.New("sqlite.intake_dossier_source_conflict"),
		)
	}
	if err := insertIntakeDossierGenerationReceipt(ctx, transaction, state.Receipt); err != nil {
		return application.IntakeDossierRecord{}, false, err
	}

	var existingReceiptRef string
	err = transaction.QueryRowContext(ctx, `
SELECT generation_receipt_ref
FROM intake_dossiers
WHERE ref = ?`, state.Dossier.Ref()).Scan(&existingReceiptRef)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = transaction.ExecContext(ctx, `
INSERT INTO intake_dossiers(
    ref, actor_ref, project_ref, state_ref, state_revision, state_digest,
    source_intake_receipt_ref, schema_ref, plan_digest, dossier_digest,
    snapshot_json, generation_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			state.Dossier.Ref(), state.ActorRef.String(), state.ProjectRef.String(),
			state.Dossier.StateRef(), int64(state.Dossier.StateRevision()),
			state.Dossier.StateDigest(), state.Dossier.SourceIntakeReceiptRef(),
			state.Dossier.Schema(), state.Dossier.PlanDigest(), state.Dossier.Digest(),
			string(snapshotJSON), state.Receipt.Ref,
		)
		if err != nil {
			return application.IntakeDossierRecord{}, false, mapDatabaseError(err)
		}
	case err != nil:
		return application.IntakeDossierRecord{}, false, mapDatabaseError(err)
	}

	record, err = readIntakeDossierRecordByReceipt(ctx, transaction, state.Receipt.Ref)
	if err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	if err := validatePersistedIntakeDossierCandidate(state, snapshotJSON, record); err != nil {
		return application.IntakeDossierRecord{}, false, invalid(err)
	}
	if err := commit(transaction); err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	return record, true, nil
}

func (repository *Repository) GetIntakeDossier(
	ctx context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	dossierRef application.IntakeDossierRef,
) (application.IntakeDossierRecord, error) {
	if err := validateIntakeDossierScope(actorRef, projectRef, dossierRef); err != nil {
		return application.IntakeDossierRecord{}, invalid(err)
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	var receiptRef string
	err = transaction.QueryRowContext(ctx, `
SELECT generation_receipt_ref
FROM intake_dossiers
WHERE actor_ref = ? AND project_ref = ? AND ref = ?`,
		actorRef.String(), projectRef.String(), dossierRef,
	).Scan(&receiptRef)
	if err != nil {
		return application.IntakeDossierRecord{}, mapDatabaseError(err)
	}
	record, err := readIntakeDossierRecordByReceipt(ctx, transaction, receiptRef)
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	if err := commit(transaction); err != nil {
		return application.IntakeDossierRecord{}, err
	}
	return record, nil
}

func insertIntakeDossierGenerationReceipt(
	ctx context.Context,
	transaction *sql.Tx,
	receipt application.IntakeDossierGenerationReceipt,
) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO intake_dossier_generation_receipts(
    ref, request_ref, request_fingerprint, actor_ref, project_ref,
    state_ref, state_revision, state_digest, source_intake_receipt_ref,
    dossier_ref, dossier_digest, plan_digest, authorization_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		receipt.Ref, receipt.RequestRef, receipt.RequestFingerprint,
		receipt.ActorRef.String(), receipt.ProjectRef.String(), receipt.StateRef,
		int64(receipt.StateRevision), receipt.StateDigest, receipt.SourceIntakeReceiptRef,
		receipt.DossierRef, receipt.DossierDigest, receipt.PlanDigest,
		receipt.AuthorizationReceiptRef,
	)
	return mapDatabaseError(err)
}

func readIntakeDossierReplay(
	ctx context.Context,
	source queryer,
	request application.IntakeDossierReplayRequest,
) (application.IntakeDossierRecord, bool, error) {
	var receiptRef, fingerprint, dossierRef, stateRef, sourceReceiptRef, planDigest, authorizationReceiptRef string
	var stateRevision int64
	err := source.QueryRowContext(ctx, `
SELECT ref, request_fingerprint, dossier_ref, state_ref, state_revision,
       source_intake_receipt_ref, plan_digest, authorization_receipt_ref
FROM intake_dossier_generation_receipts
WHERE actor_ref = ? AND project_ref = ? AND request_ref = ?`,
		request.ActorRef.String(), request.ProjectRef.String(), request.RequestRef,
	).Scan(
		&receiptRef, &fingerprint, &dossierRef, &stateRef, &stateRevision,
		&sourceReceiptRef, &planDigest, &authorizationReceiptRef,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return application.IntakeDossierRecord{}, false, nil
	case err != nil:
		return application.IntakeDossierRecord{}, false, mapDatabaseError(err)
	case fingerprint != request.RequestFingerprint ||
		(request.DossierRef != "" &&
			application.IntakeDossierRef(dossierRef) != request.DossierRef) ||
		intake.Ref(stateRef) != request.StateRef ||
		stateRevision != int64(request.ExpectedRevision) ||
		sourceReceiptRef != request.SourceIntakeReceiptRef ||
		planDigest != request.PlanDigest ||
		authorizationReceiptRef != request.AuthorizationReceiptRef:
		return application.IntakeDossierRecord{}, false, conflict(
			errors.New("sqlite.intake_dossier_replay_conflict"),
		)
	}
	record, err := readIntakeDossierRecordByReceipt(ctx, source, receiptRef)
	if err != nil {
		return application.IntakeDossierRecord{}, false, err
	}
	return record, true, nil
}

func readIntakeDossierRecordByReceipt(
	ctx context.Context,
	source queryer,
	receiptRef string,
) (application.IntakeDossierRecord, error) {
	var record application.IntakeDossierRecord
	var actorValue, projectValue, stateValue, dossierValue string
	var stateRevision int64
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, actor_ref, project_ref,
       state_ref, state_revision, state_digest, source_intake_receipt_ref,
       dossier_ref, dossier_digest, plan_digest, authorization_receipt_ref
FROM intake_dossier_generation_receipts
WHERE ref = ?`, receiptRef).Scan(
		&record.Receipt.Ref, &record.Receipt.RequestRef, &record.Receipt.RequestFingerprint,
		&actorValue, &projectValue, &stateValue, &stateRevision,
		&record.Receipt.StateDigest, &record.Receipt.SourceIntakeReceiptRef,
		&dossierValue, &record.Receipt.DossierDigest, &record.Receipt.PlanDigest,
		&record.Receipt.AuthorizationReceiptRef,
	)
	if err != nil {
		return application.IntakeDossierRecord{}, mapDatabaseError(err)
	}
	var restoreErr error
	if record.ActorRef, restoreErr = goal.NewActorRef(actorValue); restoreErr != nil {
		return application.IntakeDossierRecord{}, invalid(restoreErr)
	}
	if record.ProjectRef, restoreErr = goal.NewProjectRef(projectValue); restoreErr != nil {
		return application.IntakeDossierRecord{}, invalid(restoreErr)
	}
	if stateRevision <= 0 {
		return application.IntakeDossierRecord{}, invalid(
			errors.New("sqlite.intake_dossier_revision_invalid"),
		)
	}
	record.Receipt.ActorRef = record.ActorRef
	record.Receipt.ProjectRef = record.ProjectRef
	record.Receipt.StateRef = intake.Ref(stateValue)
	record.Receipt.StateRevision = intake.Revision(stateRevision)
	record.Receipt.DossierRef = application.IntakeDossierRef(dossierValue)

	var schemaRef, stateRef, stateDigest, sourceReceiptRef, planDigest, dossierDigest string
	var snapshotJSON string
	var dossierRevision int64
	err = source.QueryRowContext(ctx, `
SELECT schema_ref, state_ref, state_revision, state_digest,
       source_intake_receipt_ref, plan_digest, dossier_digest, snapshot_json
FROM intake_dossiers
WHERE actor_ref = ? AND project_ref = ? AND ref = ?`,
		record.ActorRef.String(), record.ProjectRef.String(), record.Receipt.DossierRef,
	).Scan(
		&schemaRef, &stateRef, &dossierRevision, &stateDigest,
		&sourceReceiptRef, &planDigest, &dossierDigest, &snapshotJSON,
	)
	if err != nil {
		return application.IntakeDossierRecord{}, mapDatabaseError(err)
	}
	var snapshot application.IntakeDossierSnapshot
	if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
		return application.IntakeDossierRecord{}, invalid(errors.Join(
			errors.New("sqlite.intake_dossier_snapshot_invalid"), err,
		))
	}
	record.Dossier, restoreErr = application.RestoreIntakeDossier(snapshot)
	if restoreErr != nil {
		return application.IntakeDossierRecord{}, invalid(restoreErr)
	}
	if err := validatePersistedIntakeDossierRecord(
		record, schemaRef, intake.Ref(stateRef), dossierRevision, stateDigest,
		sourceReceiptRef, planDigest, dossierDigest, []byte(snapshotJSON),
	); err != nil {
		return application.IntakeDossierRecord{}, invalid(err)
	}
	return record, nil
}

func validateIntakeDossierReplayRequest(request application.IntakeDossierReplayRequest) error {
	if !validIntakeRequestRef(request.RequestRef) ||
		!validCanonicalHash(request.RequestFingerprint) ||
		!validText(request.AuthorizationReceiptRef) ||
		request.ExpectedRevision == 0 ||
		uint64(request.ExpectedRevision) > maxSQLiteInteger ||
		!validCanonicalHash(request.PlanDigest) ||
		!validIntakeReceiptRef(request.SourceIntakeReceiptRef) {
		return errors.New("sqlite.intake_dossier_replay_invalid")
	}
	if request.DossierRef != "" && !validIntakeDossierRef(request.DossierRef) {
		return errors.New("sqlite.intake_dossier_replay_invalid")
	}
	if err := validateIntakeScope(request.ActorRef, request.ProjectRef, request.StateRef); err != nil {
		return err
	}
	return nil
}

func validateIntakeDossierCreateState(
	state application.IntakeDossierCreateState,
) ([]byte, error) {
	expectedFingerprint := application.IntakeDossierGenerationFingerprint(
		state.Dossier, state.AuthorizationReceipt.Ref(),
	)
	expectedReceipt, expectedReceiptErr := application.BuildIntakeDossierGenerationReceipt(
		state.RequestRef, state.RequestFingerprint, state.Dossier,
		state.AuthorizationReceipt.Ref(),
	)
	if !validIntakeRequestRef(state.RequestRef) ||
		!validCanonicalHash(state.RequestFingerprint) ||
		state.RequestFingerprint != expectedFingerprint ||
		expectedReceiptErr != nil ||
		state.Receipt != expectedReceipt ||
		state.Dossier.ActorRef() != state.ActorRef ||
		state.Dossier.ProjectRef() != state.ProjectRef ||
		state.ExpectedRevision != state.Dossier.StateRevision() ||
		uint64(state.ExpectedRevision) > maxSQLiteInteger ||
		state.SourceIntakeReceiptRef != state.Dossier.SourceIntakeReceiptRef() ||
		state.Receipt.RequestRef != state.RequestRef ||
		state.Receipt.RequestFingerprint != state.RequestFingerprint ||
		state.Receipt.ActorRef != state.ActorRef ||
		state.Receipt.ProjectRef != state.ProjectRef ||
		state.Receipt.StateRef != state.Dossier.StateRef() ||
		state.Receipt.StateRevision != state.Dossier.StateRevision() ||
		state.Receipt.StateDigest != state.Dossier.StateDigest() ||
		state.Receipt.SourceIntakeReceiptRef != state.Dossier.SourceIntakeReceiptRef() ||
		state.Receipt.DossierRef != state.Dossier.Ref() ||
		state.Receipt.DossierDigest != state.Dossier.Digest() ||
		state.Receipt.PlanDigest != state.Dossier.PlanDigest() ||
		state.Receipt.AuthorizationReceiptRef != state.AuthorizationReceipt.Ref() ||
		!validIntakeDossierGenerationReceiptRef(state.Receipt.Ref) {
		return nil, errors.New("sqlite.intake_dossier_create_invalid")
	}
	if err := validateIntakeDossierScope(
		state.ActorRef, state.ProjectRef, state.Dossier.Ref(),
	); err != nil {
		return nil, err
	}
	if err := validateIntakeDossierAuthorization(
		state.AuthorizationReceipt, state.ActorRef, state.ProjectRef,
		state.RequestRef, state.Receipt.AuthorizationReceiptRef,
	); err != nil {
		return nil, err
	}
	return encodeIntakeDossier(state.Dossier)
}

func validateIntakeDossierAuthorization(
	receipt identity.AuthorizationReceipt,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	requestRef string,
	receiptRef string,
) error {
	expectedRequestRef, err := application.IntakeDossierAuthorizationRequestRef(requestRef)
	request := receipt.Decision().Request()
	if err != nil ||
		receipt.Ref() == "" || receipt.Ref() != receiptRef ||
		identity.ValidatePrincipal(request.Principal()) != nil ||
		request.RequestRef() != expectedRequestRef ||
		request.Principal().ActorRef != actorRef ||
		request.ProjectRef() != projectRef ||
		request.Permission() != identity.PermissionGoalsCreate ||
		request.ResourceRef() != projectRef.String() ||
		receipt.Decision().Outcome() != identity.AuthorizationAllowed ||
		!identity.RoleAllows(receipt.Decision().Role(), identity.PermissionGoalsCreate) {
		return errors.New("sqlite.intake_dossier_authorization_invalid")
	}
	return nil
}

func requirePersistedIntakeDossierAuthorization(
	ctx context.Context,
	transaction *sql.Tx,
	receipt identity.AuthorizationReceipt,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
) error {
	request := receipt.Decision().Request()
	if request.Principal().ActorRef != actorRef {
		return conflict(errors.New("sqlite.intake_dossier_authorization_actor_conflict"))
	}
	_, err := requirePersistedAuthorization(
		ctx, transaction, receipt, request.Principal().Ref, projectRef,
		identity.PermissionGoalsCreate, projectRef.String(),
	)
	return err
}

func validateIntakeDossierScope(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	dossierRef application.IntakeDossierRef,
) error {
	if actorRef.String() == "" || projectRef.String() == "" ||
		!validIntakeDossierRef(dossierRef) {
		return errors.New("sqlite.intake_dossier_scope_invalid")
	}
	if restored, err := goal.NewActorRef(actorRef.String()); err != nil || restored != actorRef {
		return errors.New("sqlite.intake_dossier_actor_invalid")
	}
	if restored, err := goal.NewProjectRef(projectRef.String()); err != nil || restored != projectRef {
		return errors.New("sqlite.intake_dossier_project_invalid")
	}
	return nil
}

func validatePersistedIntakeDossierCandidate(
	state application.IntakeDossierCreateState,
	expectedJSON []byte,
	record application.IntakeDossierRecord,
) error {
	if record.ActorRef != state.ActorRef || record.ProjectRef != state.ProjectRef ||
		record.Receipt != state.Receipt ||
		record.Dossier.Ref() != state.Dossier.Ref() ||
		record.Dossier.Digest() != state.Dossier.Digest() {
		return errors.New("sqlite.intake_dossier_candidate_conflict")
	}
	actualJSON, err := encodeIntakeDossier(record.Dossier)
	if err != nil || string(actualJSON) != string(expectedJSON) {
		return errors.New("sqlite.intake_dossier_candidate_conflict")
	}
	return nil
}

func validatePersistedIntakeDossierRecord(
	record application.IntakeDossierRecord,
	schemaRef string,
	stateRef intake.Ref,
	stateRevision int64,
	stateDigest string,
	sourceReceiptRef string,
	planDigest string,
	dossierDigest string,
	storedJSON []byte,
) error {
	dossier, receipt := record.Dossier, record.Receipt
	expectedFingerprint := application.IntakeDossierGenerationFingerprint(
		dossier, receipt.AuthorizationReceiptRef,
	)
	expectedReceipt, expectedReceiptErr := application.BuildIntakeDossierGenerationReceipt(
		receipt.RequestRef, receipt.RequestFingerprint, dossier,
		receipt.AuthorizationReceiptRef,
	)
	if schemaRef != application.IntakeDossierSchema ||
		dossier.Schema() != schemaRef ||
		dossier.ActorRef() != record.ActorRef ||
		dossier.ProjectRef() != record.ProjectRef ||
		dossier.StateRef() != stateRef ||
		int64(dossier.StateRevision()) != stateRevision ||
		dossier.StateDigest() != stateDigest ||
		dossier.SourceIntakeReceiptRef() != sourceReceiptRef ||
		dossier.PlanDigest() != planDigest ||
		dossier.Digest() != dossierDigest ||
		dossier.Ref() != receipt.DossierRef ||
		receipt.ActorRef != record.ActorRef ||
		receipt.ProjectRef != record.ProjectRef ||
		receipt.StateRef != dossier.StateRef() ||
		receipt.StateRevision != dossier.StateRevision() ||
		receipt.StateDigest != dossier.StateDigest() ||
		receipt.SourceIntakeReceiptRef != dossier.SourceIntakeReceiptRef() ||
		receipt.DossierDigest != dossier.Digest() ||
		receipt.PlanDigest != dossier.PlanDigest() ||
		receipt.RequestFingerprint != expectedFingerprint ||
		expectedReceiptErr != nil ||
		receipt != expectedReceipt ||
		!validIntakeDossierGenerationReceiptRef(receipt.Ref) ||
		!validIntakeRequestRef(receipt.RequestRef) ||
		!validCanonicalHash(receipt.RequestFingerprint) ||
		!validCanonicalHash(dossier.StateDigest()) ||
		!validCanonicalHash(dossier.PlanDigest()) ||
		!validCanonicalHash(dossier.Digest()) ||
		uint64(dossier.StateRevision()) > maxSQLiteInteger {
		return errors.New("sqlite.intake_dossier_record_invalid")
	}
	canonical, err := encodeIntakeDossier(dossier)
	if err != nil || string(canonical) != string(storedJSON) {
		return errors.New("sqlite.intake_dossier_snapshot_not_canonical")
	}
	return nil
}

func encodeIntakeDossier(dossier application.IntakeDossier) ([]byte, error) {
	encoded, err := json.Marshal(application.SnapshotIntakeDossier(dossier))
	if err != nil {
		return nil, errors.Join(errors.New("sqlite.intake_dossier_snapshot_invalid"), err)
	}
	var snapshot application.IntakeDossierSnapshot
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return nil, errors.Join(errors.New("sqlite.intake_dossier_snapshot_invalid"), err)
	}
	restored, err := application.RestoreIntakeDossier(snapshot)
	if err != nil {
		return nil, errors.Join(errors.New("sqlite.intake_dossier_snapshot_invalid"), err)
	}
	if restored.Ref() != dossier.Ref() ||
		restored.Digest() != dossier.Digest() ||
		restored.PlanDigest() != dossier.PlanDigest() {
		return nil, errors.New("sqlite.intake_dossier_snapshot_invalid")
	}
	return encoded, nil
}

func validIntakeDossierRef(value application.IntakeDossierRef) bool {
	text := string(value)
	return len(text) == len("intake-dossier:")+64 &&
		text[:len("intake-dossier:")] == "intake-dossier:" &&
		validCanonicalHash(text[len("intake-dossier:"):])
}

func validIntakeDossierGenerationReceiptRef(value string) bool {
	return len(value) == len(intakeDossierGenerationReceiptPrefix)+64 &&
		value[:len(intakeDossierGenerationReceiptPrefix)] ==
			intakeDossierGenerationReceiptPrefix &&
		validCanonicalHash(value[len(intakeDossierGenerationReceiptPrefix):])
}
