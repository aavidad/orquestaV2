package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

func (repository *Repository) AmendGoal(
	ctx context.Context,
	state application.AmendGoalState,
) (application.GoalRecord, bool, error) {
	if err := validateAmendState(state); err != nil {
		return application.GoalRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.GoalRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.RequestedBy,
		state.ProjectRef, identity.PermissionGoalsAmend, state.SourceGoalRef.String(),
	); err != nil {
		return application.GoalRecord{}, false, err
	}

	var existingGoalRef, existingFingerprint string
	err = transaction.QueryRowContext(ctx, `
SELECT ref, request_fingerprint
FROM goals
WHERE requested_by_ref = ? AND project_ref = ? AND request_ref = ?`,
		state.RequestedBy.String(), state.ProjectRef.String(), state.RequestRef,
	).Scan(&existingGoalRef, &existingFingerprint)
	switch {
	case err == nil:
		if existingFingerprint != state.RequestFingerprint {
			return application.GoalRecord{}, false, conflict(errors.New("sqlite.request_fingerprint_conflict"))
		}
		record, readErr := readGoalRecord(ctx, transaction, existingGoalRef)
		if readErr != nil {
			return application.GoalRecord{}, false, readErr
		}
		if !sameAmendmentSemantics(record, state) {
			return application.GoalRecord{}, false, conflict(errors.New("sqlite.request_semantic_conflict"))
		}
		if err := commit(transaction); err != nil {
			return application.GoalRecord{}, false, err
		}
		return record, false, nil
	case !errors.Is(err, sql.ErrNoRows):
		return application.GoalRecord{}, false, mapDatabaseError(err)
	}

	var sourceActor, sourceProject, sourceState, sourceAppSpecRef, sourceSpecHash string
	var sourceRevision int64
	var sourceClosedAt sql.NullInt64
	err = transaction.QueryRowContext(ctx, `
SELECT g.actor_ref, g.project_ref, g.state, g.revision, g.app_spec_ref, spec.hash, g.closed_at
FROM goals g
JOIN app_specs spec ON spec.ref = g.app_spec_ref
WHERE g.ref = ?`, state.SourceGoalRef.String()).Scan(
		&sourceActor, &sourceProject, &sourceState, &sourceRevision, &sourceAppSpecRef, &sourceSpecHash, &sourceClosedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.GoalRecord{}, false, stateError(application.StateNotFound, err)
	}
	if err != nil {
		return application.GoalRecord{}, false, mapDatabaseError(err)
	}
	if sourceActor != state.Successor.Actor().String() || sourceProject != state.ProjectRef.String() {
		return application.GoalRecord{}, false, stateError(application.StateNotFound, sql.ErrNoRows)
	}
	if sourceRevision != int64(state.ExpectedSourceRevision) ||
		sourceSpecHash != state.ExpectedSourceSpecHash ||
		(sourceState != "succeeded" && sourceState != "failed") || !sourceClosedAt.Valid {
		return application.GoalRecord{}, false, conflict(errors.New("sqlite.amend_source_fence_conflict"))
	}
	successorSpec := state.Successor.AppSpec()
	if requiredTime(successorSpec.Intent().SubmittedAt()) < sourceClosedAt.Int64 ||
		requiredTime(successorSpec.ConfirmedAt()) < sourceClosedAt.Int64 ||
		requiredTime(state.Successor.CreatedAt()) < sourceClosedAt.Int64 {
		return application.GoalRecord{}, false, conflict(errors.New("sqlite.amend_source_time_conflict"))
	}
	parentRef, _ := state.Successor.AppSpec().ParentRef()
	if parentRef.String() != sourceAppSpecRef {
		return application.GoalRecord{}, false, conflict(errors.New("sqlite.amend_source_parent_conflict"))
	}

	if err := insertGoalHeader(
		ctx, transaction, state.RequestRef, state.RequestFingerprint,
		state.RequestedBy, state.Successor.Snapshot(),
	); err != nil {
		return application.GoalRecord{}, false, err
	}
	if err := insertEvents(ctx, transaction, state.Events); err != nil {
		return application.GoalRecord{}, false, err
	}
	record, err := readGoalRecord(ctx, transaction, state.Successor.Ref().String())
	if err != nil {
		return application.GoalRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.GoalRecord{}, false, err
	}
	return record, true, nil
}

func sameAmendmentSemantics(record application.GoalRecord, state application.AmendGoalState) bool {
	existing := record.Goal.AppSpec()
	candidate := state.Successor.AppSpec()
	existingParent, existingHasParent := existing.ParentRef()
	candidateParent, candidateHasParent := candidate.ParentRef()
	return record.RequestedBy == state.RequestedBy &&
		existingHasParent && candidateHasParent && existingParent == candidateParent &&
		existing.ParentHash() == candidate.ParentHash() &&
		existing.Generation() == candidate.Generation() &&
		existing.Intent().Actor() == candidate.Intent().Actor() &&
		existing.Intent().Project() == candidate.Intent().Project() &&
		existing.Intent().Statement() == candidate.Intent().Statement() &&
		existing.Objective() == candidate.Objective() && existing.Reason() == candidate.Reason() &&
		existing.ConfirmedBy() == candidate.ConfirmedBy()
}
