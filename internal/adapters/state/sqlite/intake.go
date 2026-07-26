package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

var _ application.IntakeStore = (*Repository)(nil)

func (repository *Repository) ReplayIntake(
	ctx context.Context,
	request application.IntakeReplayRequest,
) (application.IntakeRecord, bool, error) {
	if err := validateIntakeReplayRequest(request); err != nil {
		return application.IntakeRecord{}, false, invalid(err)
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	record, found, err := readIntakeReplay(ctx, transaction, request)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.IntakeRecord{}, false, err
	}
	return record, found, nil
}

func (repository *Repository) CreateIntake(
	ctx context.Context,
	state application.IntakeCreateState,
) (application.IntakeRecord, bool, error) {
	snapshotJSON, err := validateIntakeCreateState(state)
	if err != nil {
		return application.IntakeRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	replayRequest := application.IntakeReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Operation: state.Receipt.Operation, ActorRef: state.ActorRef,
		ProjectRef: state.ProjectRef, StateRef: state.State.Ref(),
		AuthorizationReceiptRef: state.Receipt.AuthorizationReceiptRef,
	}
	record, found, err := readIntakeReplay(ctx, transaction, replayRequest)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.IntakeRecord{}, false, err
		}
		return record, false, nil
	}
	if err := requirePersistedIntakeAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.ActorRef, state.ProjectRef,
	); err != nil {
		return application.IntakeRecord{}, false, err
	}
	if err := insertIntakeReceipt(ctx, transaction, state.Receipt, snapshotJSON); err != nil {
		return application.IntakeRecord{}, false, err
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO intake_states(
    state_ref, actor_ref, project_ref, schema_ref, revision,
    max_question_rounds, question_rounds, state_digest, snapshot_json, receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.State.Ref(), state.ActorRef.String(), state.ProjectRef.String(),
		state.State.Schema(), int64(state.State.Revision()),
		int64(state.State.Policy().MaxQuestionRounds), int64(state.State.QuestionRounds()),
		state.Receipt.StateDigest, string(snapshotJSON), state.Receipt.Ref,
	)
	if err != nil {
		return application.IntakeRecord{}, false, mapDatabaseError(err)
	}
	record, err = readIntakeRecordByReceipt(ctx, transaction, state.Receipt.Ref)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.IntakeRecord{}, false, err
	}
	return record, true, nil
}

func (repository *Repository) GetIntake(
	ctx context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
) (application.IntakeRecord, error) {
	if err := validateIntakeScope(actorRef, projectRef, stateRef); err != nil {
		return application.IntakeRecord{}, invalid(err)
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.IntakeRecord{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	record, err := readCurrentIntakeRecord(ctx, transaction, actorRef, projectRef, stateRef)
	if err != nil {
		return application.IntakeRecord{}, err
	}
	if err := commit(transaction); err != nil {
		return application.IntakeRecord{}, err
	}
	return record, nil
}

func (repository *Repository) ApplyIntake(
	ctx context.Context,
	state application.IntakeApplyState,
) (application.IntakeRecord, bool, error) {
	snapshotJSON, err := validateIntakeApplyState(state)
	if err != nil {
		return application.IntakeRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	replayRequest := application.IntakeReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Operation: state.Receipt.Operation, ActorRef: state.ActorRef,
		ProjectRef: state.ProjectRef, StateRef: state.State.Ref(),
		AuthorizationReceiptRef: state.Receipt.AuthorizationReceiptRef,
	}
	record, found, err := readIntakeReplay(ctx, transaction, replayRequest)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.IntakeRecord{}, false, err
		}
		return record, false, nil
	}
	if err := requirePersistedIntakeAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.ActorRef, state.ProjectRef,
	); err != nil {
		return application.IntakeRecord{}, false, err
	}
	chain, err := readIntakeChain(
		ctx, transaction, state.ActorRef, state.ProjectRef, state.State.Ref(),
	)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	if len(chain) == 0 || chain[len(chain)-1].State.Revision() != state.ExpectedRevision {
		return application.IntakeRecord{}, false, conflict(
			errors.New("sqlite.intake_revision_conflict"),
		)
	}
	chain = append(chain, application.IntakeRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		State: state.State, Receipt: state.Receipt,
	})
	if err := application.ValidateIntakeChain(chain); err != nil {
		return application.IntakeRecord{}, false, invalid(errors.Join(
			errors.New("sqlite.intake_apply_chain_invalid"), err,
		))
	}
	if err := insertIntakeReceipt(ctx, transaction, state.Receipt, snapshotJSON); err != nil {
		return application.IntakeRecord{}, false, err
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE intake_states
SET revision = ?, question_rounds = ?, state_digest = ?,
    snapshot_json = ?, receipt_ref = ?
WHERE actor_ref = ? AND project_ref = ? AND state_ref = ? AND revision = ?`,
		int64(state.State.Revision()), int64(state.State.QuestionRounds()),
		state.Receipt.StateDigest, string(snapshotJSON), state.Receipt.Ref,
		state.ActorRef.String(), state.ProjectRef.String(), state.State.Ref(),
		int64(state.ExpectedRevision),
	)
	if err != nil {
		return application.IntakeRecord{}, false, mapDatabaseError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return application.IntakeRecord{}, false, mapDatabaseError(err)
	}
	if affected != 1 {
		return application.IntakeRecord{}, false, conflict(errors.New("sqlite.intake_revision_conflict"))
	}
	record, err = readIntakeRecordByReceipt(ctx, transaction, state.Receipt.Ref)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.IntakeRecord{}, false, err
	}
	return record, true, nil
}

func insertIntakeReceipt(
	ctx context.Context,
	transaction *sql.Tx,
	receipt application.IntakeReceipt,
	snapshotJSON []byte,
) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO intake_receipts(
    ref, request_ref, request_fingerprint, operation, actor_ref, project_ref,
    state_ref, previous_revision, revision, state_digest,
    authorization_receipt_ref, snapshot_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		receipt.Ref, receipt.RequestRef, receipt.RequestFingerprint, string(receipt.Operation),
		receipt.ActorRef.String(), receipt.ProjectRef.String(), receipt.StateRef,
		int64(receipt.PreviousRevision), int64(receipt.Revision),
		receipt.StateDigest, receipt.AuthorizationReceiptRef, string(snapshotJSON),
	)
	return mapDatabaseError(err)
}

func requirePersistedIntakeAuthorization(
	ctx context.Context,
	transaction *sql.Tx,
	receipt identity.AuthorizationReceipt,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
) error {
	request := receipt.Decision().Request()
	if request.Principal().ActorRef != actorRef {
		return conflict(errors.New("sqlite.intake_authorization_actor_conflict"))
	}
	_, err := requirePersistedAuthorization(
		ctx, transaction, receipt, request.Principal().Ref, projectRef,
		identity.PermissionGoalsCreate, projectRef.String(),
	)
	return err
}

func readIntakeReplay(
	ctx context.Context,
	source queryer,
	request application.IntakeReplayRequest,
) (application.IntakeRecord, bool, error) {
	var receiptRef, fingerprint, operation, stateRef, authorizationReceiptRef string
	err := source.QueryRowContext(ctx, `
SELECT ref, request_fingerprint, operation, state_ref, authorization_receipt_ref
FROM intake_receipts
WHERE actor_ref = ? AND project_ref = ? AND request_ref = ?`,
		request.ActorRef.String(), request.ProjectRef.String(), request.RequestRef,
	).Scan(&receiptRef, &fingerprint, &operation, &stateRef, &authorizationReceiptRef)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return application.IntakeRecord{}, false, nil
	case err != nil:
		return application.IntakeRecord{}, false, mapDatabaseError(err)
	case fingerprint != request.RequestFingerprint ||
		application.IntakeOperation(operation) != request.Operation ||
		intake.Ref(stateRef) != request.StateRef ||
		authorizationReceiptRef != request.AuthorizationReceiptRef:
		return application.IntakeRecord{}, false, conflict(errors.New("sqlite.intake_replay_conflict"))
	}
	record, err := readIntakeRecordByReceipt(ctx, source, receiptRef)
	if err != nil {
		return application.IntakeRecord{}, false, err
	}
	return record, true, nil
}

func readIntakeRecordByReceipt(
	ctx context.Context,
	source queryer,
	receiptRef string,
) (application.IntakeRecord, error) {
	var record application.IntakeRecord
	var actorValue, projectValue, stateValue, operation, snapshotJSON string
	var previousRevision, revision int64
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, operation, actor_ref, project_ref,
       state_ref, previous_revision, revision, state_digest,
       authorization_receipt_ref, snapshot_json
FROM intake_receipts
WHERE ref = ?`, receiptRef).Scan(
		&record.Receipt.Ref, &record.Receipt.RequestRef, &record.Receipt.RequestFingerprint,
		&operation, &actorValue, &projectValue, &stateValue,
		&previousRevision, &revision, &record.Receipt.StateDigest,
		&record.Receipt.AuthorizationReceiptRef, &snapshotJSON,
	)
	if err != nil {
		return application.IntakeRecord{}, mapDatabaseError(err)
	}
	var restoreErr error
	if record.ActorRef, restoreErr = goal.NewActorRef(actorValue); restoreErr != nil {
		return application.IntakeRecord{}, invalid(restoreErr)
	}
	if record.ProjectRef, restoreErr = goal.NewProjectRef(projectValue); restoreErr != nil {
		return application.IntakeRecord{}, invalid(restoreErr)
	}
	record.Receipt.Operation = application.IntakeOperation(operation)
	record.Receipt.ActorRef = record.ActorRef
	record.Receipt.ProjectRef = record.ProjectRef
	record.Receipt.StateRef = intake.Ref(stateValue)
	if previousRevision < 0 || revision <= 0 {
		return application.IntakeRecord{}, invalid(errors.New("sqlite.intake_revision_invalid"))
	}
	record.Receipt.PreviousRevision = intake.Revision(previousRevision)
	record.Receipt.Revision = intake.Revision(revision)

	var snapshot application.IntakeSnapshot
	if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
		return application.IntakeRecord{}, invalid(errors.Join(errors.New("sqlite.intake_snapshot_invalid"), err))
	}
	record.State, restoreErr = application.RestoreIntake(snapshot)
	if restoreErr != nil {
		return application.IntakeRecord{}, invalid(restoreErr)
	}
	if err := validatePersistedIntakeRecord(record, []byte(snapshotJSON)); err != nil {
		return application.IntakeRecord{}, invalid(err)
	}
	return record, nil
}

func readCurrentIntakeRecord(
	ctx context.Context,
	source queryer,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
) (application.IntakeRecord, error) {
	var receiptRef, schemaRef, stateDigest, snapshotJSON string
	var revision, maxQuestionRounds, questionRounds int64
	err := source.QueryRowContext(ctx, `
SELECT receipt_ref, schema_ref, revision, max_question_rounds, question_rounds,
       state_digest, snapshot_json
FROM intake_states
WHERE actor_ref = ? AND project_ref = ? AND state_ref = ?`,
		actorRef.String(), projectRef.String(), stateRef,
	).Scan(
		&receiptRef, &schemaRef, &revision, &maxQuestionRounds, &questionRounds,
		&stateDigest, &snapshotJSON,
	)
	if err != nil {
		return application.IntakeRecord{}, mapDatabaseError(err)
	}
	record, err := readIntakeRecordByReceipt(ctx, source, receiptRef)
	if err != nil {
		return application.IntakeRecord{}, err
	}
	canonicalJSON, err := encodeIntakeState(record.State)
	if err != nil {
		return application.IntakeRecord{}, invalid(err)
	}
	if schemaRef != record.State.Schema() ||
		record.ActorRef != actorRef ||
		record.ProjectRef != projectRef ||
		record.State.Ref() != stateRef ||
		revision != int64(record.State.Revision()) ||
		maxQuestionRounds != int64(record.State.Policy().MaxQuestionRounds) ||
		questionRounds != int64(record.State.QuestionRounds()) ||
		stateDigest != record.Receipt.StateDigest ||
		snapshotJSON != string(canonicalJSON) {
		return application.IntakeRecord{}, invalid(errors.New("sqlite.intake_current_snapshot_invalid"))
	}
	return record, nil
}

func readIntakeChain(
	ctx context.Context,
	source queryer,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
) ([]application.IntakeRecord, error) {
	rows, err := source.QueryContext(ctx, `
SELECT ref
FROM intake_receipts
WHERE actor_ref = ? AND project_ref = ? AND state_ref = ?
ORDER BY revision`,
		actorRef.String(), projectRef.String(), stateRef,
	)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	var receiptRefs []string
	for rows.Next() {
		var receiptRef string
		if err := rows.Scan(&receiptRef); err != nil {
			_ = rows.Close()
			return nil, mapDatabaseError(err)
		}
		receiptRefs = append(receiptRefs, receiptRef)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, mapDatabaseError(err)
	}
	if err := rows.Close(); err != nil {
		return nil, mapDatabaseError(err)
	}
	records := make([]application.IntakeRecord, 0, len(receiptRefs))
	for _, receiptRef := range receiptRefs {
		record, err := readIntakeRecordByReceipt(ctx, source, receiptRef)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func encodeIntakeState(state intake.State) ([]byte, error) {
	snapshot := application.SnapshotIntake(state)
	if _, err := application.RestoreIntake(snapshot); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func intakeStateDigest(snapshotJSON []byte) string {
	digest := sha256.Sum256(snapshotJSON)
	return hex.EncodeToString(digest[:])
}
