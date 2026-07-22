package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
)

type preparedControlApplication struct {
	existing application.ControlRecord
	current  application.GoalRecord
	found    bool
	replay   bool
}

func prepareControlApplication(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
) (preparedControlApplication, error) {
	existing, found, err := readControlByRequest(
		ctx, transaction, state.PrincipalRef, state.ProjectRef, state.RequestRef,
	)
	if err != nil {
		return preparedControlApplication{}, err
	}
	prepared := preparedControlApplication{existing: existing, found: found}
	if found && state.ExpectedControlStatus == "" {
		// Concurrent first attempts generate different local refs/timestamps
		// before either observes the durable request key. Once one transaction
		// wins, the other converges on that record if and only if the complete
		// canonical request identity is equal.
		if !controlRequestIdentityMatches(state.Control, existing) {
			return preparedControlApplication{}, conflict(errors.New("sqlite.control_replay_conflict"))
		}
		prepared.replay = true
		return prepared, nil
	}
	if !found && state.ExpectedControlStatus != "" {
		return preparedControlApplication{}, conflict(errors.New("sqlite.control_missing"))
	}
	if found && !controlIdentityMatches(state.Control, existing) {
		return preparedControlApplication{}, conflict(errors.New("sqlite.control_replay_conflict"))
	}
	if found && existing.Status != state.ExpectedControlStatus {
		return preparedControlApplication{}, conflict(errors.New("sqlite.control_status_conflict"))
	}
	prepared.current, err = readGoalRecord(ctx, transaction, state.GoalRef.String())
	if err != nil {
		return preparedControlApplication{}, err
	}
	if err := validateControlTransition(state, prepared.current); err != nil {
		return preparedControlApplication{}, conflict(err)
	}
	if err := validateControlStopReceipt(state, prepared.current); err != nil {
		return preparedControlApplication{}, invalid(err)
	}
	if err := validateStoredSupersededControl(ctx, transaction, state); err != nil {
		return preparedControlApplication{}, err
	}
	return prepared, nil
}

func validateStoredSupersededControl(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
) error {
	if state.SupersededControl == nil {
		return nil
	}
	stored, exists, err := readControlByRef(ctx, transaction, state.SupersededControl.Ref)
	if err != nil {
		return err
	}
	if !exists || stored.Status != application.ControlRequested ||
		!controlIdentityMatches(*state.SupersededControl, stored) {
		return conflict(errors.New("sqlite.control_supersession_conflict"))
	}
	status, err := supersededStopActionStatus(ctx, transaction, stored)
	if err != nil {
		return err
	}
	if status != "active" && status != "quarantined" {
		return conflict(errors.New("sqlite.control_supersession_action_inactive"))
	}
	return nil
}

func (repository *Repository) persistControlApplication(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
	prepared preparedControlApplication,
) error {
	preRetiredAction, err := persistInitialControlState(ctx, transaction, state, prepared.found)
	if err != nil {
		return err
	}
	if err := persistControlGoalAndExecutions(ctx, transaction, state, prepared.current); err != nil {
		return err
	}
	if err := persistControlActionsAndMailboxes(ctx, transaction, state); err != nil {
		return err
	}
	return repository.settleControlApplication(
		ctx, transaction, state, prepared.found, preRetiredAction,
	)
}

func readPersistedAppliedControl(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ApplyControlState,
) (application.ControlRecord, error) {
	persisted, found, err := readControlByRequest(
		ctx, transaction, state.PrincipalRef, state.ProjectRef, state.RequestRef,
	)
	if err != nil {
		return application.ControlRecord{}, err
	}
	if !found {
		return application.ControlRecord{}, conflict(errors.New("sqlite.control_missing_after_write"))
	}
	return persisted, nil
}
