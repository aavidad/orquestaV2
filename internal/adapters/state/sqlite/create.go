package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
)

func (repository *Repository) CreateGoal(
	ctx context.Context,
	state application.CreateGoalState,
) (application.GoalRecord, bool, error) {
	if err := validateCreateState(state); err != nil {
		return application.GoalRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.GoalRecord{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	var existingGoalRef, existingFingerprint string
	snapshot := state.Goal.Snapshot()
	err = transaction.QueryRowContext(
		ctx,
		`SELECT ref, request_fingerprint
FROM goals
WHERE actor_ref = ? AND project_ref = ? AND request_ref = ?`,
		snapshot.ActorRef,
		snapshot.ProjectRef,
		state.RequestRef,
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
		existingSpec := record.Goal.AppSpec()
		candidateSpec := state.Goal.AppSpec()
		existingIntent := existingSpec.Intent()
		candidateIntent := candidateSpec.Intent()
		if existingIntent.Actor() != candidateIntent.Actor() || existingIntent.Project() != candidateIntent.Project() ||
			existingIntent.Statement() != candidateIntent.Statement() ||
			existingSpec.Objective() != candidateSpec.Objective() || existingSpec.Reason() != candidateSpec.Reason() ||
			existingSpec.ConfirmedBy() != candidateSpec.ConfirmedBy() {
			return application.GoalRecord{}, false, conflict(errors.New("sqlite.request_semantic_conflict"))
		}
		if err := commit(transaction); err != nil {
			return application.GoalRecord{}, false, err
		}
		return record, false, nil
	case !errors.Is(err, sql.ErrNoRows):
		return application.GoalRecord{}, false, mapDatabaseError(err)
	}

	if err := insertCreateState(ctx, transaction, state); err != nil {
		return application.GoalRecord{}, false, err
	}
	record, err := readGoalRecord(ctx, transaction, state.Goal.Ref().String())
	if err != nil {
		return application.GoalRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.GoalRecord{}, false, err
	}
	return record, true, nil
}
