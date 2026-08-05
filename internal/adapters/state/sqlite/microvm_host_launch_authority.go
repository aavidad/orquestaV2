package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"reflect"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const microVMHostLaunchAuthoritySelect = `SELECT
 execution_ref,action_fence,effect_attempt_ref,session_ref,
 plan_sha256,concession_sha256,
 control_service_ref,control_port,control_identity_ref,control_identity_sha256,
 proxy_service_ref,proxy_port,proxy_identity_ref,proxy_identity_sha256,
 external_ref,credential_ref,owner_ref,scope_ref,purpose_ref,
 credential_version,actor_ref,request_ref
FROM microvm_host_launch_authorities
WHERE execution_ref=? AND action_fence=?`

var _ ports.MicroVMHostLaunchAuthorityRegistry = (*Repository)(nil)

func (repository *Repository) Prepare(
	ctx context.Context,
	authority ports.MicroVMHostLaunchAuthorityV1,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	if ctx == nil || authority.Key.ActionFence > math.MaxInt64 ||
		uint64(authority.OneShotClaim.Version) > math.MaxInt64 ||
		ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(authority) != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, invalid(
			errors.New("sqlite.microvm_host_launch_authority_invalid"),
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
	}
	defer transaction.Rollback()

	persisted, found, err := readMicroVMHostLaunchAuthority(ctx, transaction, authority.Key)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	if found {
		if !sameMicroVMHostLaunchAuthorityExceptExternal(persisted, authority) {
			return ports.MicroVMHostLaunchAuthorityV1{}, conflict(
				errors.New("sqlite.microvm_host_launch_authority_conflict"),
			)
		}
		if err := commit(transaction); err != nil {
			return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
		}
		return ports.CloneMicroVMHostLaunchAuthorityV1(persisted), nil
	}

	proxyServiceRef, proxyPort, proxyIdentityRef, proxyIdentitySHA256 := storedMicroVMHostLaunchProxy(authority.Services)
	claim := authority.OneShotClaim
	result, err := transaction.ExecContext(ctx, `
INSERT INTO microvm_host_launch_authorities(
 execution_ref,action_fence,effect_attempt_ref,session_ref,
 plan_sha256,concession_sha256,
 control_service_ref,control_port,control_identity_ref,control_identity_sha256,
 proxy_service_ref,proxy_port,proxy_identity_ref,proxy_identity_sha256,
 credential_ref,owner_ref,scope_ref,purpose_ref,credential_version,
 actor_ref,request_ref
)
SELECT ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?
FROM effect_attempts attempt
JOIN effect_intents intent ON intent.ref=attempt.intent_ref
JOIN executions execution ON execution.ref=attempt.execution_ref
 WHERE attempt.ref=? AND attempt.execution_ref=? AND attempt.action_fence=?
 AND intent.kind='agent_launch' AND intent.execution_ref=attempt.execution_ref
 AND execution.state='dispatching'
 AND attempt.actor_ref=? AND attempt.project_ref=?
 AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref
 )`,
		authority.Key.RunRef.String(), authority.Key.ActionFence,
		authority.EffectAttemptRef, authority.SessionRef.String(),
		authority.PlanSHA256, authority.ConcessionSHA256,
		authority.Services[0].ServiceRef, authority.Services[0].Port,
		authority.Services[0].IdentityRef, authority.Services[0].IdentitySHA256,
		proxyServiceRef, proxyPort, proxyIdentityRef, proxyIdentitySHA256,
		claim.CredentialRef.String(), claim.OwnerRef.String(), claim.ScopeRef.String(),
		claim.PurposeRef.String(), claim.Version, claim.ActorRef, claim.RequestRef,
		authority.EffectAttemptRef, authority.Key.RunRef.String(), authority.Key.ActionFence,
		claim.ActorRef, claim.ScopeRef.String(),
	)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchDatabaseError(ctx, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchDatabaseError(ctx, err)
	}
	if rows != 1 {
		return ports.MicroVMHostLaunchAuthorityV1{}, conflict(
			errors.New("sqlite.microvm_host_launch_authority_causality_conflict"),
		)
	}
	persisted, found, err = readMicroVMHostLaunchAuthority(ctx, transaction, authority.Key)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	if !found || !reflect.DeepEqual(persisted, authority) {
		return ports.MicroVMHostLaunchAuthorityV1{}, invalid(
			errors.New("sqlite.microvm_host_launch_authority_insert_invalid"),
		)
	}
	if err := commit(transaction); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(persisted), nil
}

func (repository *Repository) BindExternal(
	ctx context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
	externalRef string,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	if ctx == nil || key.ActionFence > math.MaxInt64 ||
		ports.ValidateMicroVMHostLaunchAuthorityKey(key) != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, invalid(
			errors.New("sqlite.microvm_host_launch_authority_key_invalid"),
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
	}
	defer transaction.Rollback()

	persisted, found, err := readMicroVMHostLaunchAuthority(ctx, transaction, key)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	if !found {
		return ports.MicroVMHostLaunchAuthorityV1{}, stateError(
			application.StateNotFound,
			errors.New("sqlite.microvm_host_launch_authority_not_found"),
		)
	}
	candidate := ports.CloneMicroVMHostLaunchAuthorityV1(persisted)
	candidate.ExternalRef = externalRef
	if err := ports.ValidateMicroVMHostLaunchAuthorityBoundV1(candidate); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, invalid(
			errors.New("sqlite.microvm_host_launch_authority_external_ref_invalid"),
		)
	}
	if persisted.ExternalRef != "" {
		if persisted.ExternalRef != externalRef {
			return ports.MicroVMHostLaunchAuthorityV1{}, conflict(
				errors.New("sqlite.microvm_host_launch_authority_external_ref_conflict"),
			)
		}
		if err := commit(transaction); err != nil {
			return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
		}
		return ports.CloneMicroVMHostLaunchAuthorityV1(persisted), nil
	}

	result, err := transaction.ExecContext(ctx, `
UPDATE microvm_host_launch_authorities SET external_ref=?
WHERE execution_ref=? AND action_fence=? AND external_ref IS NULL`,
		externalRef, key.RunRef.String(), key.ActionFence,
	)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchDatabaseError(ctx, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchDatabaseError(ctx, err)
	}
	if rows != 1 {
		return ports.MicroVMHostLaunchAuthorityV1{}, conflict(
			errors.New("sqlite.microvm_host_launch_authority_external_ref_conflict"),
		)
	}
	persisted, found, err = readMicroVMHostLaunchAuthority(ctx, transaction, key)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	if !found || !reflect.DeepEqual(persisted, candidate) {
		return ports.MicroVMHostLaunchAuthorityV1{}, invalid(
			errors.New("sqlite.microvm_host_launch_authority_bind_invalid"),
		)
	}
	if err := commit(transaction); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(persisted), nil
}

func (repository *Repository) Resolve(
	ctx context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	if ctx == nil || key.ActionFence > math.MaxInt64 ||
		ports.ValidateMicroVMHostLaunchAuthorityKey(key) != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, invalid(
			errors.New("sqlite.microvm_host_launch_authority_key_invalid"),
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
	}
	defer transaction.Rollback()
	persisted, found, err := readMicroVMHostLaunchAuthority(ctx, transaction, key)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	if !found {
		return ports.MicroVMHostLaunchAuthorityV1{}, stateError(
			application.StateNotFound,
			errors.New("sqlite.microvm_host_launch_authority_not_found"),
		)
	}
	if err := commit(transaction); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, microVMHostLaunchContextError(ctx, err)
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(persisted), nil
}

func readMicroVMHostLaunchAuthority(
	ctx context.Context,
	source queryer,
	key ports.MicroVMHostLaunchAuthorityKey,
) (ports.MicroVMHostLaunchAuthorityV1, bool, error) {
	var executionRef, effectAttemptRef, sessionRef string
	var planSHA256, concessionSHA256 string
	var controlServiceRef, controlIdentityRef, controlIdentitySHA256 string
	var proxyServiceRef, proxyIdentityRef, proxyIdentitySHA256 sql.NullString
	var externalRef sql.NullString
	var credentialRef, ownerRef, scopeRef, purposeRef, actorRef, requestRef string
	var actionFence, controlPort, credentialVersion int64
	var proxyPort sql.NullInt64
	err := source.QueryRowContext(ctx, microVMHostLaunchAuthoritySelect,
		key.RunRef.String(), key.ActionFence,
	).Scan(
		&executionRef, &actionFence, &effectAttemptRef, &sessionRef,
		&planSHA256, &concessionSHA256,
		&controlServiceRef, &controlPort, &controlIdentityRef, &controlIdentitySHA256,
		&proxyServiceRef, &proxyPort, &proxyIdentityRef, &proxyIdentitySHA256,
		&externalRef, &credentialRef, &ownerRef, &scopeRef, &purposeRef,
		&credentialVersion, &actorRef, &requestRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.MicroVMHostLaunchAuthorityV1{}, false, nil
	}
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, false, microVMHostLaunchDatabaseError(ctx, err)
	}
	allProxyNull := !proxyServiceRef.Valid && !proxyPort.Valid && !proxyIdentityRef.Valid && !proxyIdentitySHA256.Valid
	allProxyPresent := proxyServiceRef.Valid && proxyPort.Valid && proxyIdentityRef.Valid && proxyIdentitySHA256.Valid
	if actionFence <= 0 || controlPort <= 0 || controlPort > math.MaxUint32 || credentialVersion <= 0 ||
		(!allProxyNull && !allProxyPresent) ||
		(allProxyPresent && (proxyPort.Int64 <= 0 || proxyPort.Int64 > math.MaxUint32)) ||
		(externalRef.Valid && externalRef.String == "") {
		return ports.MicroVMHostLaunchAuthorityV1{}, false, invalid(
			errors.New("sqlite.microvm_host_launch_authority_corrupt"),
		)
	}
	runRef, runErr := goal.NewExecutionRef(executionRef)
	session, sessionErr := ports.NewExecutionSessionRef(sessionRef)
	if runErr != nil || sessionErr != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, false, invalid(
			errors.New("sqlite.microvm_host_launch_authority_corrupt"),
		)
	}
	authority := ports.MicroVMHostLaunchAuthorityV1{
		Key: ports.MicroVMHostLaunchAuthorityKey{
			RunRef: runRef, ActionFence: uint64(actionFence),
		},
		EffectAttemptRef: effectAttemptRef,
		SessionRef:       session,
		OneShotClaim: credentials.OneShotUseRequest{
			ActorRef: actorRef, RequestRef: requestRef,
			CredentialRef: credentials.CredentialRef(credentialRef),
			OwnerRef:      credentials.OwnerRef(ownerRef),
			ScopeRef:      credentials.ScopeRef(scopeRef),
			PurposeRef:    credentials.PurposeRef(purposeRef),
			Version:       credentials.Version(credentialVersion),
		},
		PlanSHA256:       planSHA256,
		ConcessionSHA256: concessionSHA256,
		Services: []ports.MicroVMHostServiceAuthorityV1{{
			Role: ports.MicroVMHostServiceControlBroker, ServiceRef: controlServiceRef,
			Port: uint32(controlPort), IdentityRef: controlIdentityRef,
			IdentitySHA256: controlIdentitySHA256,
		}},
	}
	if allProxyPresent {
		authority.Services = append(authority.Services, ports.MicroVMHostServiceAuthorityV1{
			Role: ports.MicroVMHostServiceControlledEgressProxy, ServiceRef: proxyServiceRef.String,
			Port: uint32(proxyPort.Int64), IdentityRef: proxyIdentityRef.String,
			IdentitySHA256: proxyIdentitySHA256.String,
		})
	}
	if externalRef.Valid {
		authority.ExternalRef = externalRef.String
	}
	if authority.Key != key || ports.ValidateMicroVMHostLaunchAuthorityV1(authority) != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, false, invalid(
			errors.New("sqlite.microvm_host_launch_authority_corrupt"),
		)
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(authority), true, nil
}

func storedMicroVMHostLaunchProxy(services []ports.MicroVMHostServiceAuthorityV1) (any, any, any, any) {
	if len(services) != 2 {
		return nil, nil, nil, nil
	}
	proxy := services[1]
	return proxy.ServiceRef, proxy.Port, proxy.IdentityRef, proxy.IdentitySHA256
}

func sameMicroVMHostLaunchAuthorityExceptExternal(
	left ports.MicroVMHostLaunchAuthorityV1,
	right ports.MicroVMHostLaunchAuthorityV1,
) bool {
	left.ExternalRef, right.ExternalRef = "", ""
	return reflect.DeepEqual(left, right)
}

func microVMHostLaunchContextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return err
}

func microVMHostLaunchDatabaseError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return mapDatabaseError(err)
}
