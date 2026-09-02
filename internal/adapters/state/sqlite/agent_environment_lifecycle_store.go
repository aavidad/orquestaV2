package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

var _ application.AgentEnvironmentLifecycleStore = (*Repository)(nil)

type agentEnvironmentLifecycleRow struct {
	snapshot                   application.AgentEnvironmentLifecycleSnapshot
	goalRef, workItemRef       string
	claimActionRef             sql.NullString
	claimToken                 sql.NullString
	claimWorkerRef             sql.NullString
	claimDeliveryAttempt       sql.NullInt64
	claimFence                 sql.NullInt64
	claimLeaseUntil            sql.NullInt64
	actionApprovalAttached     bool
	attemptRef                 sql.NullString
	preservationRef            sql.NullString
	nextActionRef              sql.NullString
	nextActionApprovalAttached bool
	readyToFinalize            bool
}

func (repository *Repository) GetAgentEnvironmentLifecycle(
	ctx context.Context,
	executionRef goal.ExecutionRef,
) (application.AgentEnvironmentLifecycleStoredState, bool, error) {
	if executionRef.String() == "" {
		return application.AgentEnvironmentLifecycleStoredState{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_execution_invalid"))
	}
	tx, err := repository.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return application.AgentEnvironmentLifecycleStoredState{}, false, mapDatabaseError(err)
	}
	defer tx.Rollback()
	stored, found, err := readAgentEnvironmentLifecycle(ctx, tx, executionRef.String())
	if err != nil {
		return application.AgentEnvironmentLifecycleStoredState{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return application.AgentEnvironmentLifecycleStoredState{}, false, mapDatabaseError(err)
	}
	return stored, found, nil
}

func (repository *Repository) RecordAgentEnvironmentLifecycleInitial(
	ctx context.Context,
	state application.AgentEnvironmentLifecycleInitialState,
) (application.AgentEnvironmentLifecycleSnapshot, bool, error) {
	if application.ValidateAgentEnvironmentLifecycleInitialState(state) != nil ||
		state.Snapshot.Revision > maxSQLiteInteger {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_initial_invalid"))
	}
	payload, err := encodeAgentEnvironmentLifecycleSnapshot(state.Snapshot)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	defer tx.Rollback()
	winner, found, err := readAgentEnvironmentLifecycle(ctx, tx, state.Snapshot.Subject.ExecutionRef.String())
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if found {
		if err := commit(tx); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		return winner.Snapshot, false, nil
	}
	if err := validateAgentEnvironmentLifecycleSnapshotAgainstRecord(
		ctx, tx, state.Snapshot, nil,
	); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_initial_causality_invalid"))
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO agent_environment_lifecycles(
 execution_ref,goal_ref,work_item_ref,snapshot_schema,revision,launch_receipt_ref,
 token_state,snapshot_json,next_action_ref,next_action_approval_attached,recorded_at
) VALUES(?,?,?,?,?,?,?,?,NULL,0,?)`,
		state.Snapshot.Subject.ExecutionRef.String(), state.Snapshot.Subject.GoalRef.String(),
		state.Snapshot.Subject.WorkItemRef.String(), state.Snapshot.Schema, int64(state.Snapshot.Revision),
		state.Snapshot.LaunchReceiptRef, string(state.Snapshot.Token.State), string(payload),
		requiredTime(state.Snapshot.RecordedAt))
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, mapDatabaseError(err)
	}
	if state.NextAction != nil {
		if err := repository.requireLiveLease(state.Claim); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		if err := requireClaim(ctx, tx, state.Claim); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		if err := completeClaim(ctx, tx, state.Claim, state.OperationAt, "", false); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		if err := insertAction(ctx, tx, *state.NextAction); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		result, err := tx.ExecContext(ctx, `
UPDATE agent_environment_lifecycles
SET next_action_ref=?,next_action_approval_attached=1
WHERE execution_ref=? AND revision=? AND next_action_ref IS NULL AND ready_to_finalize=0`,
			state.NextAction.Ref, state.Snapshot.Subject.ExecutionRef.String(), int64(state.Snapshot.Revision))
		if err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, mapDatabaseError(err)
		}
		if err := requireOneRow(result); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
	}
	if err := commit(tx); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	return state.Snapshot, true, nil
}

func (repository *Repository) RecordAgentEnvironmentLifecycleAttempt(
	ctx context.Context,
	state application.AgentEnvironmentLifecyclePreEffectState,
) (application.AgentEnvironmentLifecycleSnapshot, bool, error) {
	if application.ValidateAgentEnvironmentLifecycleAttemptState(state) != nil ||
		state.ExpectedRevision > maxSQLiteInteger || state.Snapshot.Revision > maxSQLiteInteger {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_attempt_invalid"))
	}
	payload, err := encodeAgentEnvironmentLifecycleSnapshot(state.Snapshot)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	defer tx.Rollback()
	current, found, err := readAgentEnvironmentLifecycle(ctx, tx, state.Snapshot.Subject.ExecutionRef.String())
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if !found {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			conflict(errors.New("sqlite.agent_environment_lifecycle_missing"))
	}
	if current.Snapshot.Revision != state.ExpectedRevision {
		if err := commit(tx); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		return current.Snapshot, false, nil
	}
	if err := validateAgentEnvironmentAttemptTransition(current, state); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if err := requireClaim(ctx, tx, state.Claim); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if err := insertEffectAttempt(ctx, tx, state.Attempt); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE agent_environment_lifecycles SET
 revision=?,token_state=?,snapshot_json=?,claim_action_ref=?,claim_token=?,claim_worker_ref=?,
 claim_delivery_attempt=?,claim_fence=?,claim_lease_until=?,action_approval_attached=?,attempt_ref=?,
 next_action_ref=NULL,next_action_approval_attached=0,ready_to_finalize=0,recorded_at=?
WHERE execution_ref=? AND revision=?`,
		int64(state.Snapshot.Revision), string(state.Snapshot.Token.State), string(payload),
		state.Claim.Action.Ref, state.Claim.Token, state.Claim.WorkerRef,
		int64(state.Claim.DeliveryAttempt), int64(state.Claim.Fence), requiredTime(state.Claim.LeaseUntil),
		storedBool(state.Claim.Action.EffectApproval != nil), state.Attempt.Ref,
		requiredTime(state.Snapshot.RecordedAt), state.Snapshot.Subject.ExecutionRef.String(),
		int64(state.ExpectedRevision))
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if err := commit(tx); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	return state.Snapshot, true, nil
}

func (repository *Repository) RecordAgentEnvironmentLifecycleTerminal(
	ctx context.Context,
	state application.AgentEnvironmentLifecyclePostEffectState,
) (application.AgentEnvironmentLifecycleSnapshot, bool, error) {
	if application.ValidateAgentEnvironmentLifecycleTerminalState(state) != nil ||
		state.ExpectedRevision > maxSQLiteInteger || state.Snapshot.Revision > maxSQLiteInteger {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_terminal_invalid"))
	}
	payload, err := encodeAgentEnvironmentLifecycleSnapshot(state.Snapshot)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	defer tx.Rollback()
	current, found, err := readAgentEnvironmentLifecycle(ctx, tx, state.Snapshot.Subject.ExecutionRef.String())
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if !found {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			conflict(errors.New("sqlite.agent_environment_lifecycle_missing"))
	}
	if current.Snapshot.Revision != state.ExpectedRevision {
		if err := commit(tx); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		return current.Snapshot, false, nil
	}
	if !current.HasAttempt || !reflect.DeepEqual(current.Claim, state.Claim) || current.Attempt != state.Attempt ||
		current.Snapshot != lifecycleAttemptedSnapshotBeforeTerminal(state) {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			conflict(errors.New("sqlite.agent_environment_lifecycle_terminal_cas_conflict"))
	}
	if state.PreservationFact != nil {
		if _, _, err := insertAgentEnvironmentPreservation(ctx, tx, *state.PreservationFact); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false,
				fmt.Errorf("sqlite.agent_environment_lifecycle_terminal_preservation: %w", err)
		}
	}
	if err := insertAgentEnvironmentLifecycleEffectReceipt(ctx, tx, state.Attempt, *state.EffectReceipt); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			fmt.Errorf("sqlite.agent_environment_lifecycle_terminal_effect_receipt: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE outbox SET claim_token=?,claimed_by=?,claimed_until=?,delivery_attempt=?,fence=?,
 completed_at=?,quarantined_at=NULL,last_error_code=''
WHERE ref=? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		state.Claim.Token, state.Claim.WorkerRef, requiredTime(state.Claim.LeaseUntil),
		int64(state.Claim.DeliveryAttempt), int64(state.Claim.Fence),
		requiredTime(state.ConsumptionReceipt.ConsumedAt), state.Claim.Action.Ref)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if err := validateConsumptionReceipt(state.ConsumptionReceipt); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, invalid(err)
	}
	if err := insertActionConsumptionReceipt(ctx, tx, state.ConsumptionReceipt, 1, state.EffectReceipt); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			fmt.Errorf("sqlite.agent_environment_lifecycle_terminal_consumption: %w", err)
	}
	if state.NextAction != nil {
		if err := insertAction(ctx, tx, *state.NextAction); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false,
				fmt.Errorf("sqlite.agent_environment_lifecycle_terminal_next_action: %w", err)
		}
	}
	if state.FinalizationAction != nil {
		if err := insertAction(ctx, tx, *state.FinalizationAction); err != nil {
			return application.AgentEnvironmentLifecycleSnapshot{}, false,
				fmt.Errorf("sqlite.agent_environment_lifecycle_terminal_finalization_action: %w", err)
		}
	}
	preservation := current.Preservation
	if state.PreservationFact != nil {
		preservation = state.PreservationFact
	}
	if err := validateAgentEnvironmentLifecycleSnapshotAgainstRecord(
		ctx, tx, state.Snapshot, preservation,
	); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_terminal_causality_invalid"))
	}
	var preservationRef, nextActionRef any
	if state.Snapshot.Preservation.ApplicationReceiptRef != "" {
		preservationRef = state.Snapshot.Preservation.ApplicationReceiptRef
	}
	if state.NextAction != nil {
		nextActionRef = state.NextAction.Ref
	}
	result, err = tx.ExecContext(ctx, `
UPDATE agent_environment_lifecycles SET
 revision=?,token_state=?,snapshot_json=?,preservation_ref=?,next_action_ref=?,
 next_action_approval_attached=?,ready_to_finalize=?,recorded_at=?
WHERE execution_ref=? AND revision=? AND attempt_ref=?`,
		int64(state.Snapshot.Revision), string(state.Snapshot.Token.State), string(payload), preservationRef,
		nextActionRef, storedBool(state.NextAction != nil && state.NextAction.EffectApproval != nil),
		storedBool(state.ReadyToFinalize), requiredTime(state.Snapshot.RecordedAt),
		state.Snapshot.Subject.ExecutionRef.String(), int64(state.ExpectedRevision), state.Attempt.Ref)
	if err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if err := commit(tx); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	return state.Snapshot, true, nil
}

func validateAgentEnvironmentAttemptTransition(
	current application.AgentEnvironmentLifecycleStoredState,
	state application.AgentEnvironmentLifecyclePreEffectState,
) error {
	want := current.Snapshot
	want.Revision++
	want.Effect = state.Snapshot.Effect
	want.RecordedAt = state.Snapshot.RecordedAt
	if want != state.Snapshot || current.Snapshot.Effect.NeedsReconciliation() || current.ReadyToFinalize {
		return conflict(errors.New("sqlite.agent_environment_lifecycle_attempt_cas_conflict"))
	}
	if current.Snapshot.Effect.IsEmpty() {
		if current.HasAttempt || state.Claim.Action.Kind != application.ActionQuiesceAgent ||
			(current.NextAction != nil && !sameLifecycleActionIgnoringAttachedApproval(*current.NextAction, state.Claim.Action)) {
			return conflict(errors.New("sqlite.agent_environment_lifecycle_attempt_frontier_invalid"))
		}
		return nil
	}
	if current.NextAction == nil || !sameLifecycleActionIgnoringAttachedApproval(*current.NextAction, state.Claim.Action) {
		return conflict(errors.New("sqlite.agent_environment_lifecycle_next_action_conflict"))
	}
	return nil
}

func sameLifecycleActionIgnoringAttachedApproval(left, right application.ActionRecord) bool {
	left.EffectApproval, right.EffectApproval = nil, nil
	return reflect.DeepEqual(left, right)
}

func lifecycleAttemptedSnapshotBeforeTerminal(
	state application.AgentEnvironmentLifecyclePostEffectState,
) application.AgentEnvironmentLifecycleSnapshot {
	previous := state.Snapshot
	previous.Revision--
	previous.Token = state.Snapshot.Effect.ExpectedToken
	previous.Effect.OutcomeToken = ports.AgentEnvironmentLifecycleToken{}
	previous.Effect.PhysicalReceipt = ""
	previous.Effect.EffectReceiptRef = ""
	if state.Claim.Action.Kind == application.ActionPreserveAgentEnvironment {
		previous.Preservation = ports.AgentPreservationBinding{}
		previous.PreservationReceiptRef = ""
	}
	previous.RecordedAt = state.Attempt.StartedAt
	return previous
}

func readAgentEnvironmentLifecycle(
	ctx context.Context,
	source queryer,
	executionRef string,
) (application.AgentEnvironmentLifecycleStoredState, bool, error) {
	var row agentEnvironmentLifecycleRow
	var payload string
	var actionApproval, nextApproval, ready int64
	var revision, recordedAt int64
	var schema, launchReceiptRef, tokenState string
	err := source.QueryRowContext(ctx, `
SELECT goal_ref,work_item_ref,snapshot_schema,revision,launch_receipt_ref,token_state,snapshot_json,
 claim_action_ref,claim_token,claim_worker_ref,claim_delivery_attempt,claim_fence,claim_lease_until,
 action_approval_attached,attempt_ref,preservation_ref,next_action_ref,next_action_approval_attached,
 ready_to_finalize,recorded_at
FROM agent_environment_lifecycles WHERE execution_ref=?`, executionRef).Scan(
		&row.goalRef, &row.workItemRef, &schema, &revision, &launchReceiptRef, &tokenState, &payload,
		&row.claimActionRef, &row.claimToken, &row.claimWorkerRef, &row.claimDeliveryAttempt,
		&row.claimFence, &row.claimLeaseUntil, &actionApproval, &row.attemptRef, &row.preservationRef,
		&row.nextActionRef, &nextApproval, &ready, &recordedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return application.AgentEnvironmentLifecycleStoredState{}, false, nil
	}
	if err != nil {
		return application.AgentEnvironmentLifecycleStoredState{}, false, mapDatabaseError(err)
	}
	if revision <= 0 || recordedAt == 0 || (actionApproval != 0 && actionApproval != 1) ||
		(nextApproval != 0 && nextApproval != 1) || (ready != 0 && ready != 1) {
		return application.AgentEnvironmentLifecycleStoredState{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_corrupt"))
	}
	row.actionApprovalAttached, row.nextActionApprovalAttached, row.readyToFinalize =
		actionApproval == 1, nextApproval == 1, ready == 1
	row.snapshot, err = decodeAgentEnvironmentLifecycleSnapshot([]byte(payload))
	if err != nil || row.snapshot.Schema != schema || row.snapshot.Revision != uint64(revision) ||
		row.snapshot.LaunchReceiptRef != launchReceiptRef || string(row.snapshot.Token.State) != tokenState ||
		row.snapshot.Subject.ExecutionRef.String() != executionRef || row.snapshot.Subject.GoalRef.String() != row.goalRef ||
		row.snapshot.Subject.WorkItemRef.String() != row.workItemRef ||
		requiredTime(row.snapshot.RecordedAt) != recordedAt {
		return application.AgentEnvironmentLifecycleStoredState{}, false,
			invalid(errors.New("sqlite.agent_environment_lifecycle_projection_corrupt"))
	}
	stored, err := expandAgentEnvironmentLifecycleRow(ctx, source, row)
	if err != nil {
		return application.AgentEnvironmentLifecycleStoredState{}, false, err
	}
	return stored, true, nil
}

func expandAgentEnvironmentLifecycleRow(
	ctx context.Context,
	source queryer,
	row agentEnvironmentLifecycleRow,
) (application.AgentEnvironmentLifecycleStoredState, error) {
	stored := application.AgentEnvironmentLifecycleStoredState{
		Snapshot: row.snapshot, ReadyToFinalize: row.readyToFinalize,
	}
	if !row.claimActionRef.Valid {
		if row.snapshot.Effect.IsEmpty() && !row.attemptRef.Valid && !row.preservationRef.Valid &&
			!row.readyToFinalize {
			var next *application.ActionRecord
			if row.nextActionRef.Valid {
				action, actionErr := readAgentEnvironmentLifecycleAction(
					ctx, source, row.nextActionRef.String, row.nextActionApprovalAttached, nil,
				)
				if actionErr != nil {
					return stored, actionErr
				}
				next = &action
			}
			if application.ValidateAgentEnvironmentLifecycleInitialState(
				application.AgentEnvironmentLifecycleInitialState{
					Snapshot: row.snapshot, OperationAt: row.snapshot.RecordedAt,
				},
			) != nil {
				return stored, invalid(errors.New("sqlite.agent_environment_lifecycle_initial_corrupt"))
			}
			stored.NextAction = next
			return stored, nil
		}
		return stored, invalid(errors.New("sqlite.agent_environment_lifecycle_authority_corrupt"))
	}
	if !row.claimToken.Valid || !row.claimWorkerRef.Valid || !row.claimDeliveryAttempt.Valid ||
		!row.claimFence.Valid || !row.claimLeaseUntil.Valid || !row.attemptRef.Valid ||
		row.claimDeliveryAttempt.Int64 <= 0 || row.claimFence.Int64 <= 0 {
		return stored, invalid(errors.New("sqlite.agent_environment_lifecycle_claim_corrupt"))
	}
	attempt, found, err := readEffectAttemptByFence(ctx, source, row.claimActionRef.String, uint64(row.claimFence.Int64))
	if err != nil || !found || attempt.Ref != row.attemptRef.String {
		if err != nil {
			return stored, err
		}
		return stored, invalid(errors.New("sqlite.agent_environment_lifecycle_attempt_corrupt"))
	}
	approval, found, err := readEffectApprovalByRef(ctx, source, attempt.ApprovalRef)
	if err != nil || !found {
		if err != nil {
			return stored, err
		}
		return stored, invalid(errors.New("sqlite.agent_environment_lifecycle_approval_corrupt"))
	}
	action, err := readAgentEnvironmentLifecycleAction(ctx, source, row.claimActionRef.String,
		row.actionApprovalAttached, &approval)
	if err != nil {
		return stored, err
	}
	stored.Claim = application.ActionClaim{
		Action: action, Token: row.claimToken.String, WorkerRef: row.claimWorkerRef.String,
		DeliveryAttempt: uint64(row.claimDeliveryAttempt.Int64), Fence: uint64(row.claimFence.Int64),
		EffectApproval: approval, LeaseUntil: time.Unix(0, row.claimLeaseUntil.Int64).UTC(),
	}
	stored.Attempt, stored.HasAttempt = attempt, true
	if row.preservationRef.Valid {
		fact, found, err := leerPreservacionEntorno(ctx, source,
			consultaPreservacionEntorno+` WHERE ref=?`, row.preservationRef.String)
		if err != nil || !found {
			if err != nil {
				return stored, err
			}
			return stored, invalid(errors.New("sqlite.agent_environment_lifecycle_preservation_corrupt"))
		}
		stored.Preservation = &fact
	}
	if row.nextActionRef.Valid {
		next, err := readAgentEnvironmentLifecycleAction(ctx, source, row.nextActionRef.String,
			row.nextActionApprovalAttached, nil)
		if err != nil {
			return stored, err
		}
		stored.NextAction = &next
	}
	if err := validateExpandedAgentEnvironmentLifecycle(ctx, source, stored); err != nil {
		return stored, err
	}
	return stored, nil
}

func validateExpandedAgentEnvironmentLifecycle(
	ctx context.Context,
	source queryer,
	stored application.AgentEnvironmentLifecycleStoredState,
) error {
	if stored.Snapshot.Effect.NeedsReconciliation() {
		required := ""
		if stored.Claim.Action.Kind == application.ActionCloseAgentEnvironment {
			required = stored.Snapshot.Preservation.ApplicationReceiptRef
		}
		state := application.AgentEnvironmentLifecyclePreEffectState{
			Claim: stored.Claim, ExpectedRevision: stored.Snapshot.Revision - 1,
			RequiredApplicationReceiptRef: required, Attempt: stored.Attempt,
			Snapshot: stored.Snapshot, OperationAt: stored.Snapshot.RecordedAt,
		}
		if application.ValidateAgentEnvironmentLifecycleAttemptState(state) != nil ||
			stored.NextAction != nil || stored.ReadyToFinalize {
			return invalid(errors.New("sqlite.agent_environment_lifecycle_attempted_corrupt"))
		}
		return nil
	}
	receipt, found, err := readAgentEnvironmentLifecycleEffectReceipt(
		ctx, source, stored.Snapshot.Subject.GoalRef.String(), stored.Snapshot.Effect.EffectReceiptRef)
	if err != nil || !found {
		if err != nil {
			return err
		}
		return invalid(errors.New("sqlite.agent_environment_lifecycle_effect_receipt_corrupt"))
	}
	consumption, found, err := readAgentEnvironmentLifecycleConsumption(
		ctx, source, stored.Snapshot.Subject.GoalRef.String(), stored.Claim.Action.Ref)
	if err != nil || !found {
		if err != nil {
			return err
		}
		return invalid(errors.New("sqlite.agent_environment_lifecycle_consumption_corrupt"))
	}
	post := application.AgentEnvironmentLifecyclePostEffectState{
		Claim: stored.Claim, ExpectedRevision: stored.Snapshot.Revision - 1,
		Attempt: stored.Attempt, Snapshot: stored.Snapshot, EffectReceipt: &receipt,
		ConsumptionReceipt: consumption, NextAction: stored.NextAction,
		ReadyToFinalize: stored.ReadyToFinalize, OperationAt: stored.Snapshot.RecordedAt,
	}
	if stored.ReadyToFinalize && stored.Snapshot.Token.State == ports.AgentEnvironmentClosed {
		finalization, finalizationErr := readAgentEnvironmentLifecycleAction(
			ctx, source, "action:observe-finalize:"+stored.Snapshot.Subject.ExecutionRef.String(), false, nil,
		)
		if finalizationErr != nil {
			return finalizationErr
		}
		post.FinalizationAction = &finalization
	}
	if stored.Claim.Action.Kind == application.ActionPreserveAgentEnvironment {
		post.PreservationFact = stored.Preservation
	}
	if application.ValidateAgentEnvironmentLifecycleTerminalState(post) != nil {
		return invalid(errors.New("sqlite.agent_environment_lifecycle_terminal_corrupt"))
	}
	return nil
}

func readAgentEnvironmentLifecycleAction(
	ctx context.Context,
	source queryer,
	ref string,
	approvalAttached bool,
	knownApproval *application.EffectApproval,
) (application.ActionRecord, error) {
	var action application.ActionRecord
	var kind, goalRef, workItemRef, executionRef, changeRef string
	var controlRef, intentRef sql.NullString
	var planGeneration, workItemGeneration, availableAt, governanceVersion int64
	err := source.QueryRowContext(ctx, `
SELECT ref,kind,goal_ref,work_item_ref,execution_ref,control_ref,change_ref,expected_target_oid,
 review_gate_digest,plan_generation,work_item_generation,available_at,governance_version,effect_intent_ref
FROM outbox WHERE ref=?`, ref).Scan(
		&action.Ref, &kind, &goalRef, &workItemRef, &executionRef, &controlRef, &changeRef,
		&action.ExpectedTargetOID, &action.ReviewGateDigest, &planGeneration, &workItemGeneration,
		&availableAt, &governanceVersion, &intentRef)
	if err != nil {
		return action, mapDatabaseError(err)
	}
	isFinalization := governanceVersion == 0 && !intentRef.Valid &&
		application.ActionKind(kind) == application.ActionObserveAgent
	if planGeneration <= 0 || workItemGeneration <= 0 ||
		(!isFinalization && (governanceVersion != 1 || !intentRef.Valid)) {
		return action, invalid(errors.New("sqlite.agent_environment_lifecycle_action_corrupt"))
	}
	var refErr error
	if action.GoalRef, refErr = goal.NewGoalRef(goalRef); refErr == nil {
		action.WorkItemRef, refErr = goal.NewWorkItemRef(workItemRef)
	}
	if refErr == nil {
		action.ExecutionRef, refErr = goal.NewExecutionRef(executionRef)
	}
	if refErr == nil && changeRef != "" {
		action.ChangeRef, refErr = ports.NewChangeSetRef(changeRef)
	}
	if refErr != nil {
		return action, invalid(refErr)
	}
	action.Kind, action.ControlRef = application.ActionKind(kind), controlRef.String
	action.PlanGeneration, action.WorkItemGeneration = goal.PlanGeneration(planGeneration), goal.Revision(workItemGeneration)
	action.AvailableAt, action.EffectIntentRef = time.Unix(0, availableAt).UTC(), intentRef.String
	if isFinalization {
		if approvalAttached || knownApproval != nil || validateAction(action) != nil {
			return action, invalid(errors.New("sqlite.agent_environment_lifecycle_action_corrupt"))
		}
		return action, nil
	}
	action.EffectIntent, err = readEffectIntent(ctx, source, action.EffectIntentRef)
	if err != nil {
		return action, err
	}
	if approvalAttached {
		var approval application.EffectApproval
		if knownApproval != nil && knownApproval.IntentRef == action.EffectIntentRef {
			approval = *knownApproval
		} else {
			approval, err = readUniqueEffectApprovalForIntent(ctx, source, action.EffectIntentRef)
			if err != nil {
				return action, err
			}
		}
		action.EffectApproval = &approval
	}
	if validateAction(action) != nil || application.ValidateEffectIntent(action.EffectIntent) != nil ||
		action.EffectIntent.Ref != action.EffectIntentRef || action.EffectIntent.ActionRef != action.Ref ||
		action.EffectIntent.ActionKind != action.Kind {
		return action, invalid(errors.New("sqlite.agent_environment_lifecycle_action_invalid"))
	}
	return action, nil
}

func readUniqueEffectApprovalForIntent(
	ctx context.Context,
	source queryer,
	intentRef string,
) (application.EffectApproval, error) {
	var count int
	if err := source.QueryRowContext(ctx, `SELECT COUNT(*) FROM effect_approvals WHERE intent_ref=?`, intentRef).Scan(&count); err != nil {
		return application.EffectApproval{}, mapDatabaseError(err)
	}
	if count != 1 {
		return application.EffectApproval{}, invalid(errors.New("sqlite.agent_environment_lifecycle_approval_ambiguous"))
	}
	approval, found, err := scanEffectApproval(ctx, source, effectApprovalSelect+` WHERE intent_ref=?`, intentRef)
	if err != nil || !found {
		if err != nil {
			return application.EffectApproval{}, err
		}
		return application.EffectApproval{}, invalid(errors.New("sqlite.agent_environment_lifecycle_approval_missing"))
	}
	return approval, nil
}

func readAgentEnvironmentLifecycleEffectReceipt(
	ctx context.Context,
	source queryer,
	goalRef, receiptRef string,
) (application.EffectReceipt, bool, error) {
	receipts, err := readEffectReceiptsForGoal(ctx, source, goalRef)
	if err != nil {
		return application.EffectReceipt{}, false, err
	}
	var found *application.EffectReceipt
	for index := range receipts {
		if receipts[index].Ref == receiptRef {
			if found != nil {
				return application.EffectReceipt{}, false, invalid(errors.New("sqlite.agent_environment_lifecycle_receipt_duplicate"))
			}
			candidate := receipts[index]
			found = &candidate
		}
	}
	if found == nil {
		return application.EffectReceipt{}, false, nil
	}
	return *found, true, nil
}

func readAgentEnvironmentLifecycleConsumption(
	ctx context.Context,
	source queryer,
	goalRef, actionRef string,
) (application.ActionConsumptionReceipt, bool, error) {
	receipts, err := readConsumptionReceipts(ctx, source, goalRef)
	if err != nil {
		return application.ActionConsumptionReceipt{}, false, err
	}
	for _, receipt := range receipts {
		if receipt.ActionRef == actionRef {
			return receipt, true, nil
		}
	}
	return application.ActionConsumptionReceipt{}, false, nil
}

func insertAgentEnvironmentLifecycleEffectReceipt(
	ctx context.Context,
	tx *sql.Tx,
	attempt application.EffectAttempt,
	receipt application.EffectReceipt,
) error {
	if receipt.AttemptRef != attempt.Ref || receipt.ActionRef != attempt.ActionRef ||
		receipt.ActionFence != attempt.ActionFence || receipt.IntentRef != attempt.IntentRef ||
		receipt.IntentDigest != attempt.IntentDigest || receipt.ApprovalRef != attempt.ApprovalRef ||
		receipt.Subject != attempt.Subject || receipt.IdempotencyKey != attempt.IdempotencyKey ||
		receipt.ConfirmedAt.Before(attempt.StartedAt) || governance.ValidateResourceUsage(receipt.Usage) != nil ||
		!validSQLiteEffectStatus(receipt.Status) {
		return invalid(errors.New("sqlite.agent_environment_lifecycle_effect_receipt_invalid"))
	}
	intent, err := readEffectIntent(ctx, tx, receipt.IntentRef)
	if err != nil {
		return err
	}
	if intent.Digest != receipt.IntentDigest || !validSQLiteEffectStatusForKind(intent.Kind, receipt.Status) {
		return invalid(errors.New("sqlite.agent_environment_lifecycle_effect_status_invalid"))
	}
	usage := receipt.Usage.Resources
	_, err = tx.ExecContext(ctx, `
INSERT INTO effect_receipts(
 ref,intent_ref,intent_digest,approval_ref,attempt_ref,project_ref,goal_ref,work_item_ref,
 execution_ref,plan_generation,app_spec_generation,spec_hash,actor_ref,action_ref,action_fence,
 idempotency_key,external_ref,status,usage_tokens,usage_money_micros,usage_currency,
 usage_active_time_ns,usage_process_slots,usage_disk_bytes,usage_known,usage_quality,confirmed_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		receipt.Ref, receipt.IntentRef, receipt.IntentDigest, receipt.ApprovalRef, receipt.AttemptRef,
		receipt.Subject.ProjectRef.String(), receipt.Subject.GoalRef.String(), receipt.Subject.WorkItemRef.String(),
		receipt.Subject.ExecutionRef.String(), int64(receipt.Subject.PlanGeneration),
		int64(receipt.Subject.AppSpecGeneration), receipt.Subject.SpecHash, receipt.Subject.ActorRef.String(),
		receipt.ActionRef, int64(receipt.ActionFence), receipt.IdempotencyKey, receipt.ExternalRef,
		string(receipt.Status), usage.Tokens, usage.MoneyMicros, string(usage.Currency), usage.ActiveTimeNS,
		usage.ProcessSlots, usage.DiskBytes, int64(receipt.Usage.Known), string(receipt.Usage.Quality),
		requiredTime(receipt.ConfirmedAt))
	return mapDatabaseError(err)
}

type lifecycleSnapshotJSON struct {
	Schema                 string                    `json:"schema"`
	Revision               uint64                    `json:"revision"`
	LaunchReceiptRef       string                    `json:"launch_receipt_ref"`
	Subject                lifecycleSubjectJSON      `json:"subject"`
	Token                  lifecycleTokenJSON        `json:"token"`
	Effect                 lifecycleEffectJSON       `json:"effect"`
	Preservation           lifecyclePreservationJSON `json:"preservation"`
	PreservationReceiptRef string                    `json:"preservation_receipt_ref"`
	RecordedAt             int64                     `json:"recorded_at_unix_nano"`
}

type lifecycleSubjectJSON struct {
	ExecutionRef, GoalRef, WorkItemRef                     string
	PlanGeneration, AppSpecGeneration, ExecutionAttempt    uint64
	SpecHash, ProviderRef, ModelRef, AgentRef, ExternalRef string
}

type lifecycleTokenJSON struct {
	PhysicalToken, Revision, Fence, State string
}

type lifecycleEffectJSON struct {
	ActionRef, ActionKind, AttemptRef                string
	ActionFence, DeliveryAttempt, WorkItemGeneration uint64
	ClaimToken, WorkerRef, IdempotencyKey            string
	ExpectedToken, OutcomeToken                      lifecycleTokenJSON
	PhysicalReceipt, EffectReceiptRef                string
}

type lifecyclePreservationJSON struct {
	ApplicationReceiptRef, ManifestRef, ManifestSHA256 string
}

func encodeAgentEnvironmentLifecycleSnapshot(
	snapshot application.AgentEnvironmentLifecycleSnapshot,
) ([]byte, error) {
	payload, err := json.Marshal(snapshotToLifecycleJSON(snapshot))
	if err != nil {
		return nil, invalid(err)
	}
	return payload, nil
}

func snapshotToLifecycleJSON(snapshot application.AgentEnvironmentLifecycleSnapshot) lifecycleSnapshotJSON {
	return lifecycleSnapshotJSON{
		Schema: snapshot.Schema, Revision: snapshot.Revision, LaunchReceiptRef: snapshot.LaunchReceiptRef,
		Subject: lifecycleSubjectJSON{
			ExecutionRef: snapshot.Subject.ExecutionRef.String(), GoalRef: snapshot.Subject.GoalRef.String(),
			WorkItemRef: snapshot.Subject.WorkItemRef.String(), PlanGeneration: uint64(snapshot.Subject.PlanGeneration),
			AppSpecGeneration: uint64(snapshot.Subject.AppSpecGeneration), ExecutionAttempt: snapshot.Subject.ExecutionAttempt,
			SpecHash: snapshot.Subject.SpecHash, ProviderRef: snapshot.Subject.ProviderRef, ModelRef: snapshot.Subject.ModelRef,
			AgentRef: snapshot.Subject.AgentRef, ExternalRef: snapshot.Subject.ExternalRef,
		},
		Token: lifecycleTokenToJSON(snapshot.Token),
		Effect: lifecycleEffectJSON{
			ActionRef: snapshot.Effect.ActionRef, ActionKind: string(snapshot.Effect.ActionKind),
			AttemptRef: snapshot.Effect.AttemptRef, ActionFence: snapshot.Effect.ActionFence,
			DeliveryAttempt:    snapshot.Effect.DeliveryAttempt,
			WorkItemGeneration: uint64(snapshot.Effect.WorkItemGeneration), ClaimToken: snapshot.Effect.ClaimToken,
			WorkerRef: snapshot.Effect.WorkerRef, IdempotencyKey: snapshot.Effect.IdempotencyKey,
			ExpectedToken:   lifecycleTokenToJSON(snapshot.Effect.ExpectedToken),
			OutcomeToken:    lifecycleTokenToJSON(snapshot.Effect.OutcomeToken),
			PhysicalReceipt: snapshot.Effect.PhysicalReceipt, EffectReceiptRef: snapshot.Effect.EffectReceiptRef,
		},
		Preservation: lifecyclePreservationJSON{
			ApplicationReceiptRef: snapshot.Preservation.ApplicationReceiptRef,
			ManifestRef:           snapshot.Preservation.PhysicalManifest.ManifestRef,
			ManifestSHA256:        snapshot.Preservation.PhysicalManifest.ManifestSHA256,
		},
		PreservationReceiptRef: snapshot.PreservationReceiptRef,
		RecordedAt:             requiredTime(snapshot.RecordedAt),
	}
}

func lifecycleTokenToJSON(token ports.AgentEnvironmentLifecycleToken) lifecycleTokenJSON {
	return lifecycleTokenJSON{PhysicalToken: token.PhysicalToken.String(), Revision: token.Revision.String(),
		Fence: token.Fence.String(), State: string(token.State)}
}

func decodeAgentEnvironmentLifecycleSnapshot(payload []byte) (application.AgentEnvironmentLifecycleSnapshot, error) {
	var stored lifecycleSnapshotJSON
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&stored); err != nil {
		return application.AgentEnvironmentLifecycleSnapshot{}, invalid(errors.New("sqlite.agent_environment_lifecycle_snapshot_json_invalid"))
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return application.AgentEnvironmentLifecycleSnapshot{}, invalid(errors.New("sqlite.agent_environment_lifecycle_snapshot_json_trailing"))
	}
	canonical, err := json.Marshal(stored)
	if err != nil || !bytes.Equal(canonical, payload) {
		return application.AgentEnvironmentLifecycleSnapshot{}, invalid(errors.New("sqlite.agent_environment_lifecycle_snapshot_json_noncanonical"))
	}
	return lifecycleJSONToSnapshot(stored)
}

func lifecycleJSONToSnapshot(stored lifecycleSnapshotJSON) (application.AgentEnvironmentLifecycleSnapshot, error) {
	var snapshot application.AgentEnvironmentLifecycleSnapshot
	var err error
	if snapshot.Subject.ExecutionRef, err = goal.NewExecutionRef(stored.Subject.ExecutionRef); err == nil {
		snapshot.Subject.GoalRef, err = goal.NewGoalRef(stored.Subject.GoalRef)
	}
	if err == nil {
		snapshot.Subject.WorkItemRef, err = goal.NewWorkItemRef(stored.Subject.WorkItemRef)
	}
	if err != nil || stored.Subject.PlanGeneration == 0 || stored.Subject.AppSpecGeneration == 0 ||
		stored.Subject.PlanGeneration > maxSQLiteInteger || stored.Subject.AppSpecGeneration > maxSQLiteInteger ||
		stored.Effect.WorkItemGeneration > maxSQLiteInteger {
		return snapshot, invalid(errors.New("sqlite.agent_environment_lifecycle_snapshot_subject_invalid"))
	}
	snapshot.Subject.PlanGeneration = goal.PlanGeneration(stored.Subject.PlanGeneration)
	snapshot.Subject.AppSpecGeneration = goal.AppSpecGeneration(stored.Subject.AppSpecGeneration)
	snapshot.Subject.ExecutionAttempt = stored.Subject.ExecutionAttempt
	snapshot.Subject.SpecHash, snapshot.Subject.ProviderRef = stored.Subject.SpecHash, stored.Subject.ProviderRef
	snapshot.Subject.ModelRef, snapshot.Subject.AgentRef, snapshot.Subject.ExternalRef =
		stored.Subject.ModelRef, stored.Subject.AgentRef, stored.Subject.ExternalRef
	if snapshot.Token, err = lifecycleJSONToToken(stored.Token); err != nil {
		return snapshot, err
	}
	expected, err := lifecycleJSONToToken(stored.Effect.ExpectedToken)
	if err != nil {
		return snapshot, err
	}
	outcome, err := lifecycleJSONToToken(stored.Effect.OutcomeToken)
	if err != nil {
		return snapshot, err
	}
	snapshot.Schema, snapshot.Revision, snapshot.LaunchReceiptRef = stored.Schema, stored.Revision, stored.LaunchReceiptRef
	snapshot.Effect = application.AgentEnvironmentLifecycleEffectSnapshot{
		ActionRef: stored.Effect.ActionRef, ActionKind: application.ActionKind(stored.Effect.ActionKind),
		AttemptRef: stored.Effect.AttemptRef, ActionFence: stored.Effect.ActionFence,
		DeliveryAttempt:    stored.Effect.DeliveryAttempt,
		WorkItemGeneration: goal.Revision(stored.Effect.WorkItemGeneration), ClaimToken: stored.Effect.ClaimToken,
		WorkerRef: stored.Effect.WorkerRef, IdempotencyKey: stored.Effect.IdempotencyKey,
		ExpectedToken: expected, OutcomeToken: outcome, PhysicalReceipt: stored.Effect.PhysicalReceipt,
		EffectReceiptRef: stored.Effect.EffectReceiptRef,
	}
	snapshot.Preservation = ports.AgentPreservationBinding{
		ApplicationReceiptRef: stored.Preservation.ApplicationReceiptRef,
		PhysicalManifest: ports.AgentPhysicalPreservationBinding{
			ManifestRef: stored.Preservation.ManifestRef, ManifestSHA256: stored.Preservation.ManifestSHA256,
		},
	}
	snapshot.PreservationReceiptRef = stored.PreservationReceiptRef
	snapshot.RecordedAt = time.Unix(0, stored.RecordedAt).UTC()
	return snapshot, nil
}

func lifecycleJSONToToken(stored lifecycleTokenJSON) (ports.AgentEnvironmentLifecycleToken, error) {
	if stored == (lifecycleTokenJSON{}) {
		return ports.AgentEnvironmentLifecycleToken{}, nil
	}
	physical, err := ports.NewAgentPhysicalToken(stored.PhysicalToken)
	if err != nil {
		return ports.AgentEnvironmentLifecycleToken{}, invalid(err)
	}
	revision, err := ports.NewAgentPhysicalRevision(stored.Revision)
	if err != nil {
		return ports.AgentEnvironmentLifecycleToken{}, invalid(err)
	}
	fence, err := ports.NewAgentPhysicalFence(stored.Fence)
	if err != nil {
		return ports.AgentEnvironmentLifecycleToken{}, invalid(err)
	}
	return ports.AgentEnvironmentLifecycleToken{
		PhysicalToken: physical, Revision: revision, Fence: fence,
		State: ports.AgentEnvironmentLifecycleState(stored.State),
	}, nil
}

func validarRecuperacionAgentEnvironmentLifecycles(ctx context.Context, tx *sql.Tx) error {
	refs, err := readSingleColumn(ctx, tx, `SELECT execution_ref
FROM agent_environment_lifecycles ORDER BY execution_ref`)
	if err != nil {
		return err
	}
	for _, executionRef := range refs {
		stored, found, err := readAgentEnvironmentLifecycle(ctx, tx, executionRef)
		if err != nil || !found {
			return errors.New("sqlite.recovery_agent_environment_lifecycle_invalid")
		}
		if err := validateAgentEnvironmentLifecycleSnapshotAgainstRecord(
			ctx, tx, stored.Snapshot, stored.Preservation,
		); err != nil {
			return errors.New("sqlite.recovery_agent_environment_lifecycle_history_invalid")
		}
	}
	var orphans int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM effect_intents intent
LEFT JOIN agent_environment_lifecycles lifecycle ON lifecycle.execution_ref=intent.execution_ref
WHERE intent.kind IN ('agent_quiesce','agent_environment_preserve','agent_environment_close')
 AND lifecycle.execution_ref IS NULL`).Scan(&orphans)
	if err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("sqlite.recovery_agent_environment_lifecycle_orphan_effect")
	}
	var invalidActions int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM outbox action
LEFT JOIN effect_intents intent ON intent.ref=action.effect_intent_ref
LEFT JOIN agent_environment_lifecycles lifecycle ON lifecycle.execution_ref=action.execution_ref
WHERE action.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment') AND (
 action.governance_version<>1 OR intent.ref IS NULL OR intent.action_ref<>action.ref
 OR intent.action_kind<>action.kind OR intent.goal_ref<>action.goal_ref
 OR intent.work_item_ref<>action.work_item_ref OR intent.execution_ref<>action.execution_ref
 OR intent.plan_generation<>action.plan_generation
 OR intent.kind<>CASE action.kind
  WHEN 'quiesce_agent' THEN 'agent_quiesce'
  WHEN 'preserve_agent_environment' THEN 'agent_environment_preserve'
  WHEN 'close_agent_environment' THEN 'agent_environment_close' END
 OR lifecycle.execution_ref IS NULL OR lifecycle.goal_ref<>action.goal_ref
 OR lifecycle.work_item_ref<>action.work_item_ref OR NOT (
  (action.completed_at IS NULL AND lifecycle.token_state=CASE action.kind
    WHEN 'quiesce_agent' THEN 'active'
    WHEN 'preserve_agent_environment' THEN 'quiesced'
    WHEN 'close_agent_environment' THEN 'preserved' END
   AND ((action.kind='quiesce_agent' AND lifecycle.revision=1
         AND lifecycle.claim_action_ref IS NULL AND lifecycle.attempt_ref IS NULL
         AND lifecycle.next_action_ref IS NULL AND lifecycle.ready_to_finalize=0)
        OR lifecycle.next_action_ref=action.ref OR lifecycle.claim_action_ref=action.ref))
  OR
  (action.completed_at IS NOT NULL AND EXISTS (
   SELECT 1 FROM effect_attempts attempt
   JOIN effect_receipts receipt ON receipt.attempt_ref=attempt.ref
   JOIN action_consumption_receipts consumed
    ON consumed.action_ref=action.ref AND consumed.effect_receipt_ref=receipt.ref
   WHERE attempt.action_ref=action.ref AND attempt.intent_ref=intent.ref
    AND receipt.action_ref=action.ref AND receipt.action_fence=attempt.action_fence
    AND consumed.fence=attempt.action_fence AND consumed.claim_token=action.claim_token
    AND consumed.worker_ref=action.claimed_by AND consumed.delivery_attempt=action.delivery_attempt
    AND consumed.consumed_at=action.completed_at AND consumed.outcome='completed'
    AND consumed.error_code=''))))`).Scan(&invalidActions)
	if err != nil {
		return err
	}
	if invalidActions != 0 {
		return errors.New("sqlite.recovery_agent_environment_lifecycle_action_invalid")
	}
	return nil
}

func validateAgentEnvironmentLifecycleSnapshotAgainstRecord(
	ctx context.Context,
	source queryer,
	snapshot application.AgentEnvironmentLifecycleSnapshot,
	preservation *application.ComprobantePreservacionEntornoAgente,
) error {
	record, err := readGoalRecord(ctx, source, snapshot.Subject.GoalRef.String())
	if err != nil {
		return err
	}
	execution, found := lifecycleExecutionForRecovery(record, snapshot.Subject.ExecutionRef)
	if !found {
		return errors.New("sqlite.agent_environment_lifecycle_execution_invalid")
	}
	launchEffect, found := lifecycleLaunchReceiptForRecovery(record, execution.LaunchReceiptRef)
	if !found {
		return errors.New("sqlite.agent_environment_lifecycle_launch_invalid")
	}
	request := ports.AgentLaunchRequest{
		ExecutionRef: execution.Ref, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		ExecutionAttempt: execution.AttemptNo, SpecHash: execution.SpecHash,
		IdempotencyKey:              execution.IdempotencyKey,
		RequierePreservacionEntorno: execution.RequierePreservacionEntorno,
	}
	physical := ports.AgentLaunchReceipt{
		ExecutionRef: execution.Ref, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		ExecutionAttempt: execution.AttemptNo, SpecHash: execution.SpecHash,
		ProviderRef: execution.ProviderRef, ModelRef: execution.ModelRef, AgentRef: execution.AgentRef,
		ExternalRef: execution.ExternalRef, IdempotencyKey: execution.IdempotencyKey,
		ReceiptRef: launchEffect.ExternalRef, AcceptedAt: execution.ProviderAcceptedAt,
		RequierePreservacionEntorno: execution.RequierePreservacionEntorno,
	}
	replayed, err := application.ReplayAgentEnvironmentLifecycleSnapshot(
		snapshot, request, physical, record, preservation,
	)
	if err != nil || replayed != snapshot {
		return errors.New("sqlite.agent_environment_lifecycle_history_invalid")
	}
	return nil
}

func lifecycleExecutionForRecovery(
	record application.GoalRecord,
	ref goal.ExecutionRef,
) (application.ExecutionRecord, bool) {
	var selected application.ExecutionRecord
	matches := 0
	for _, execution := range record.Executions {
		if execution.Ref == ref {
			selected = execution
			matches++
		}
	}
	return selected, matches == 1
}

func lifecycleLaunchReceiptForRecovery(
	record application.GoalRecord,
	ref string,
) (application.EffectReceipt, bool) {
	var selected application.EffectReceipt
	matches := 0
	for _, receipt := range record.EffectReceipts {
		if receipt.Ref == ref {
			selected = receipt
			matches++
		}
	}
	return selected, matches == 1 && selected.Status == application.EffectStatusAccepted
}
