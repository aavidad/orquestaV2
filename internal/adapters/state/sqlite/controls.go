package sqlite

import (
	"context"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

func (repository *Repository) ControlReplay(
	ctx context.Context,
	request application.ControlReplayRequest,
) (application.ControlRecord, bool, error) {
	if !validText(request.RequestRef) || !validText(request.RequestFingerprint) ||
		request.PrincipalRef.String() == "" || request.ProjectRef.String() == "" || request.GoalRef.String() == "" {
		return application.ControlRecord{}, false, invalid(errors.New("sqlite.control_replay_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.ControlRecord{}, false, err
	}
	defer transaction.Rollback()
	record, found, err := readControlByRequest(
		ctx, transaction, request.PrincipalRef, request.ProjectRef, request.RequestRef,
	)
	if err != nil {
		return application.ControlRecord{}, false, err
	}
	if found && (record.RequestFingerprint != request.RequestFingerprint || record.GoalRef != request.GoalRef) {
		return application.ControlRecord{}, false, conflict(errors.New("sqlite.control_replay_conflict"))
	}
	if err := commit(transaction); err != nil {
		return application.ControlRecord{}, false, err
	}
	return record, found, nil
}

func (repository *Repository) ApplyControl(
	ctx context.Context,
	state application.ApplyControlState,
) (application.ControlRecord, bool, error) {
	if err := validateApplyControlState(state); err != nil {
		return application.ControlRecord{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.ControlRecord{}, false, err
	}
	defer transaction.Rollback()
	now, err := repository.transactionTime()
	if err != nil {
		return application.ControlRecord{}, false, err
	}
	if state.OperationAt.After(now) {
		return application.ControlRecord{}, false, invalid(errors.New("sqlite.control_operation_time_future"))
	}
	if localTerminalControlApplication(state) {
		if _, err := requirePersistedAuthorizationFact(
			ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef,
			state.ProjectRef, identity.PermissionGoalsDirect, state.GoalRef.String(),
		); err != nil {
			return application.ControlRecord{}, false, err
		}
	} else if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef,
		state.ProjectRef, identity.PermissionGoalsDirect, state.GoalRef.String(),
	); err != nil {
		return application.ControlRecord{}, false, err
	}
	prepared, err := prepareControlApplication(ctx, transaction, state)
	if err != nil {
		return application.ControlRecord{}, false, err
	}
	if prepared.replay {
		if err := commit(transaction); err != nil {
			return application.ControlRecord{}, false, err
		}
		return prepared.existing, false, nil
	}
	if err := repository.persistControlApplication(ctx, transaction, state, prepared); err != nil {
		return application.ControlRecord{}, false, err
	}
	persisted, err := readPersistedAppliedControl(ctx, transaction, state)
	if err != nil {
		return application.ControlRecord{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.ControlRecord{}, false, err
	}
	return persisted, !prepared.found, nil
}

func localTerminalControlApplication(state application.ApplyControlState) bool {
	terminal := state.ExpectedExecutionState == application.ExecutionSucceeded ||
		state.ExpectedExecutionState == application.ExecutionFailed ||
		state.ExpectedExecutionState == application.ExecutionCanceled ||
		state.ExpectedExecutionState == application.ExecutionStopped
	return terminal && state.Claim.Action.Kind == application.ActionStopAgent && state.EffectReceipt == nil &&
		state.Control.Operation == application.ControlStop && state.Control.Status == application.ControlConfirmed &&
		state.Control.ReceiptRef == "receipt:local-terminal:"+state.Control.ExecutionRef.String() &&
		state.Control.ConfirmedAt.Equal(state.OperationAt)
}
