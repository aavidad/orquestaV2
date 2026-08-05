package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const recoveryV38MicroVMHostLaunchAuthorityInvalid = "sqlite.recovery_v38_microvm_host_launch_authority_invalid"

func validateRecoveryV38MicroVMHostLaunchAuthority(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT execution_ref,action_fence
FROM microvm_host_launch_authorities
ORDER BY execution_ref,action_fence`)
	if err != nil {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(err)
	}
	var keys []ports.MicroVMHostLaunchAuthorityKey
	for rows.Next() {
		var executionRef string
		var actionFence int64
		if err := rows.Scan(&executionRef, &actionFence); err != nil {
			_ = rows.Close()
			return invalidRecoveryV38MicroVMHostLaunchAuthority(err)
		}
		runRef, refErr := goal.NewExecutionRef(executionRef)
		key := ports.MicroVMHostLaunchAuthorityKey{
			RunRef: runRef, ActionFence: uint64(actionFence),
		}
		if refErr != nil || actionFence <= 0 || ports.ValidateMicroVMHostLaunchAuthorityKey(key) != nil {
			_ = rows.Close()
			return invalidRecoveryV38MicroVMHostLaunchAuthority(nil)
		}
		keys = append(keys, key)
	}
	if err := rows.Close(); err != nil {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(err)
	}
	if err := rows.Err(); err != nil {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(err)
	}

	for _, key := range keys {
		authority, found, err := readMicroVMHostLaunchAuthority(ctx, tx, key)
		if err != nil || !found {
			return invalidRecoveryV38MicroVMHostLaunchAuthority(err)
		}
		if err := validateRecoveryV38MicroVMHostLaunchAuthorityCausality(ctx, tx, authority); err != nil {
			return err
		}
	}
	return nil
}

func validateRecoveryV38MicroVMHostLaunchAuthorityCausality(
	ctx context.Context,
	tx *sql.Tx,
	authority ports.MicroVMHostLaunchAuthorityV1,
) error {
	var attemptExecution, attemptActor, attemptProject string
	var intentKind, intentExecution, intentActor, intentProject string
	var executionSession, executionExternal string
	var attemptFence int64
	var receiptExternal sql.NullString
	err := tx.QueryRowContext(ctx, `
SELECT attempt.execution_ref,attempt.action_fence,attempt.actor_ref,attempt.project_ref,
 intent.kind,intent.execution_ref,intent.actor_ref,intent.project_ref,
 receipt.external_ref,execution.execution_session_ref,execution.external_ref
FROM effect_attempts attempt
JOIN effect_intents intent ON intent.ref=attempt.intent_ref
JOIN executions execution ON execution.ref=attempt.execution_ref
LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=attempt.ref
WHERE attempt.ref=?`, authority.EffectAttemptRef).Scan(
		&attemptExecution, &attemptFence, &attemptActor, &attemptProject,
		&intentKind, &intentExecution, &intentActor, &intentProject,
		&receiptExternal, &executionSession, &executionExternal,
	)
	if err != nil {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(err)
	}
	claim := authority.OneShotClaim
	if attemptFence <= 0 || attemptExecution != authority.Key.RunRef.String() ||
		uint64(attemptFence) != authority.Key.ActionFence ||
		attemptActor != claim.ActorRef || attemptActor != claim.OwnerRef.String() ||
		attemptProject != claim.ScopeRef.String() || intentKind != "agent_launch" ||
		intentExecution != attemptExecution || intentActor != attemptActor || intentProject != attemptProject ||
		executionSession != authority.SessionRef.String() {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(nil)
	}
	if receiptExternal.Valid &&
		(authority.ExternalRef == "" || receiptExternal.String != authority.ExternalRef) {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(nil)
	}
	if executionExternal != "" &&
		(authority.ExternalRef == "" || executionExternal != authority.ExternalRef) {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(nil)
	}
	if (receiptExternal.Valid || executionExternal != "") &&
		ports.ValidateMicroVMHostLaunchAuthorityBoundV1(authority) != nil {
		return invalidRecoveryV38MicroVMHostLaunchAuthority(nil)
	}
	return nil
}

func invalidRecoveryV38MicroVMHostLaunchAuthority(cause error) error {
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		return cause
	}
	return invalid(errors.Join(cause, errors.New(recoveryV38MicroVMHostLaunchAuthorityInvalid)))
}
