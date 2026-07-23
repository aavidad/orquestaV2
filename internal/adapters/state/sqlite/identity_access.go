package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

const (
	authorizationReasonAllowed           = "rbac.allowed"
	authorizationReasonProjectUnknown    = "rbac.project_unknown"
	authorizationReasonMembershipMissing = "rbac.membership_missing"
	authorizationReasonMembershipRevoked = "rbac.membership_revoked"
	authorizationReasonPermissionDenied  = "rbac.permission_denied"
	authorizationReasonExecutionBound    = "execution.bound.allowed"
)

type authorizationRow struct {
	ref                string
	fingerprint        string
	projectRef         string
	permission         string
	resourceRef        string
	requestedAt        int64
	outcome            string
	role               string
	membershipRevision int64
	reasonCode         string
	decidedAt          int64
	recordedAt         int64
}

func (repository *Repository) Authorize(
	ctx context.Context,
	request identity.AuthorizationRequest,
) (identity.AuthorizationReceipt, error) {
	if err := validateAuthorizationRequest(request); err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	defer func() { _ = transaction.Rollback() }()

	if err := ensurePrincipal(ctx, transaction, request.Principal()); err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	executionBound, executionRevision, err := executionAuthorization(
		ctx, transaction, request,
	)
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	if request.Principal().Kind == identity.PrincipalKindService && !executionBound {
		return identity.AuthorizationReceipt{}, application.ErrForbidden
	}
	fingerprint := authorizationRequestFingerprint(request)
	stored, found, err := findAuthorizationByRequest(ctx, transaction, request.Principal().Ref, request.RequestRef())
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	if found {
		storedRequest, restoreRequestErr := authorizationReplayRequest(request, stored)
		if restoreRequestErr != nil {
			return identity.AuthorizationReceipt{}, restoreRequestErr
		}
		fingerprint = authorizationRequestFingerprint(storedRequest)
		receipt, restoreErr := restoreAuthorizationReceipt(storedRequest, fingerprint, stored)
		if restoreErr != nil {
			return identity.AuthorizationReceipt{}, restoreErr
		}
		if err := commit(transaction); err != nil {
			return identity.AuthorizationReceipt{}, err
		}
		return receipt, nil
	}

	now, err := repository.transactionTime()
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	if now.Before(request.RequestedAt()) {
		now = request.RequestedAt()
	}
	outcome, role, revision, reason := identity.AuthorizationAllowed, identity.RoleExecutionService,
		executionRevision, authorizationReasonExecutionBound
	if !executionBound {
		outcome, role, revision, reason, err = authorizationDecision(ctx, transaction, request)
		if err != nil {
			return identity.AuthorizationReceipt{}, err
		}
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: outcome, Role: role,
		MembershipRevision: revision, ReasonCode: reason, DecidedAt: now,
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: deterministicRef("authorization-receipt", fingerprint), Decision: decision, RecordedAt: now,
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO authorization_receipts(
    ref, request_ref, request_fingerprint, principal_ref, project_ref,
    permission, resource_ref, requested_at, outcome, role,
    membership_revision, reason_code, decided_at, recorded_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		receipt.Ref(), request.RequestRef(), fingerprint, request.Principal().Ref.String(),
		request.ProjectRef().String(), string(request.Permission()), request.ResourceRef(),
		requiredTime(request.RequestedAt()), string(outcome), string(role), int64(revision),
		reason, requiredTime(now), requiredTime(now),
	); err != nil {
		return identity.AuthorizationReceipt{}, mapDatabaseError(err)
	}
	if err := commit(transaction); err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	return receipt, nil
}

func executionAuthorization(
	ctx context.Context,
	transaction *sql.Tx,
	request identity.AuthorizationRequest,
) (bool, identity.MembershipRevision, error) {
	principal := request.Principal()
	if principal.Kind != identity.PrincipalKindService {
		return false, 0, nil
	}
	if request.Permission() != identity.PermissionGoalsGet &&
		request.Permission() != identity.PermissionGoalsDirect &&
		request.Permission() != identity.PermissionArtifactsRead {
		return false, 0, nil
	}
	rows, err := transaction.QueryContext(ctx, `
SELECT g.project_ref,e.goal_ref,e.work_item_ref,e.ref,COALESCE(e.replaces_execution_ref,''),
       e.attempt_no,e.plan_generation,e.app_spec_generation,e.spec_hash
FROM executions e JOIN goals g ON g.ref=e.goal_ref
WHERE g.project_ref=? AND g.state='running' AND
 (e.state IN ('dispatching','running') OR
  (e.state='succeeded' AND EXISTS (
    SELECT 1 FROM outbox action
    WHERE action.goal_ref=e.goal_ref AND action.execution_ref=e.ref
      AND action.kind='admit_mailbox' AND action.completed_at IS NULL
      AND action.retired_at IS NULL AND action.quarantined_at IS NULL)))`,
		request.ProjectRef().String(),
	)
	if err != nil {
		return false, 0, mapDatabaseError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var projectValue, goalValue, itemValue, executionValue, replacementValue, specHash string
		var attempt, planGeneration, appSpecGeneration int64
		if err := rows.Scan(&projectValue, &goalValue, &itemValue, &executionValue,
			&replacementValue, &attempt, &planGeneration, &appSpecGeneration, &specHash); err != nil {
			return false, 0, mapDatabaseError(err)
		}
		projectRef, _ := goal.NewProjectRef(projectValue)
		goalRef, _ := goal.NewGoalRef(goalValue)
		itemRef, _ := goal.NewWorkItemRef(itemValue)
		executionRef, _ := goal.NewExecutionRef(executionValue)
		var replacement goal.ExecutionRef
		if replacementValue != "" {
			replacement, _ = goal.NewExecutionRef(replacementValue)
		}
		session := ports.ExecutionSessionEnsureRequest{
			ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: itemRef,
			ExecutionRef: executionRef, ExecutionAttempt: uint64(attempt),
			ReplacesExecutionRef: replacement, PlanGeneration: goal.PlanGeneration(planGeneration),
			AppSpecGeneration: goal.AppSpecGeneration(appSpecGeneration), SpecHash: specHash,
		}
		authority, deriveErr := application.DeriveExecutionSessionAuthority(session, principal.Method)
		if deriveErr != nil || authority.ServicePrincipal != principal {
			continue
		}
		scoped, scopeErr := executionAuthorizationScope(ctx, transaction, request, authority)
		if scopeErr != nil {
			return false, 0, scopeErr
		}
		if scoped {
			return true, identity.MembershipRevision(attempt), nil
		}
		return false, 0, nil
	}
	if err := rows.Err(); err != nil {
		return false, 0, mapDatabaseError(err)
	}
	return false, 0, nil
}

func executionAuthorizationScope(
	ctx context.Context,
	transaction *sql.Tx,
	request identity.AuthorizationRequest,
	authority ports.ExecutionSessionAuthority,
) (bool, error) {
	switch request.Permission() {
	case identity.PermissionGoalsDirect:
		return request.ResourceRef() == authority.Request.GoalRef.String(), nil
	case identity.PermissionArtifactsRead:
		var count int
		err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM mailbox_artifact_refs artifact
JOIN mailbox_envelopes envelope
 ON envelope.ref=artifact.mailbox_message_ref AND envelope.goal_ref=artifact.goal_ref
WHERE artifact.artifact_ref=? AND envelope.goal_ref=? AND envelope.project_ref=?
 AND envelope.recipient_execution_ref=? AND envelope.recipient_principal_ref=?`,
			request.ResourceRef(), authority.Request.GoalRef.String(),
			authority.Request.ProjectRef.String(), authority.Request.ExecutionRef.String(),
			authority.ServicePrincipal.Ref.String(),
		).Scan(&count)
		if err != nil {
			return false, mapDatabaseError(err)
		}
		return count > 0, nil
	case identity.PermissionGoalsGet:
		if request.ResourceRef() == authority.Request.GoalRef.String() {
			return true, nil
		}
		var count int
		err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM mailbox_envelopes WHERE ref=? AND goal_ref=? AND project_ref=?
 AND ((source_execution_ref=? AND source_principal_ref=?) OR
      (recipient_execution_ref=? AND recipient_principal_ref=?))`,
			request.ResourceRef(), authority.Request.GoalRef.String(), authority.Request.ProjectRef.String(),
			authority.Request.ExecutionRef.String(), authority.ServicePrincipal.Ref.String(),
			authority.Request.ExecutionRef.String(), authority.ServicePrincipal.Ref.String(),
		).Scan(&count)
		if err != nil {
			return false, mapDatabaseError(err)
		}
		return count > 0, nil
	default:
		return false, nil
	}
}

func authorizationReplayRequest(
	request identity.AuthorizationRequest,
	stored authorizationRow,
) (identity.AuthorizationRequest, error) {
	if stored.projectRef != request.ProjectRef().String() ||
		stored.permission != string(request.Permission()) || stored.resourceRef != request.ResourceRef() {
		return identity.AuthorizationRequest{}, conflict(errors.New("sqlite.authorization_request_conflict"))
	}
	restored, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: request.RequestRef(), Principal: request.Principal(), ProjectRef: request.ProjectRef(),
		Permission: request.Permission(), ResourceRef: request.ResourceRef(),
		RequestedAt: time.Unix(0, stored.requestedAt).UTC(),
	})
	if err != nil {
		return identity.AuthorizationRequest{}, invalid(err)
	}
	return restored, nil
}

func (repository *Repository) Membership(
	ctx context.Context,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
) (identity.Membership, error) {
	if principalRef.String() == "" || projectRef.String() == "" {
		return identity.Membership{}, invalid(errors.New("sqlite.membership_query_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return identity.Membership{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	membership, err := readMembership(ctx, transaction, principalRef, projectRef)
	if err != nil {
		return identity.Membership{}, err
	}
	if err := commit(transaction); err != nil {
		return identity.Membership{}, err
	}
	return membership, nil
}

func authorizationDecision(
	ctx context.Context,
	transaction *sql.Tx,
	request identity.AuthorizationRequest,
) (identity.AuthorizationOutcome, identity.Role, identity.MembershipRevision, string, error) {
	var projectExists int
	err := transaction.QueryRowContext(ctx, `SELECT 1 FROM projects WHERE ref = ?`, request.ProjectRef().String()).Scan(&projectExists)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.AuthorizationDenied, "", 0, authorizationReasonProjectUnknown, nil
	}
	if err != nil {
		return "", "", 0, "", mapDatabaseError(err)
	}
	membership, err := readMembership(ctx, transaction, request.Principal().Ref, request.ProjectRef())
	if application.IsStateError(err, application.StateNotFound) {
		return identity.AuthorizationDenied, "", 0, authorizationReasonMembershipMissing, nil
	}
	if err != nil {
		return "", "", 0, "", err
	}
	if !membership.IsActive() {
		return identity.AuthorizationDenied, membership.Role(), membership.Revision(), authorizationReasonMembershipRevoked, nil
	}
	if !identity.RoleAllows(membership.Role(), request.Permission()) {
		return identity.AuthorizationDenied, membership.Role(), membership.Revision(), authorizationReasonPermissionDenied, nil
	}
	return identity.AuthorizationAllowed, membership.Role(), membership.Revision(), authorizationReasonAllowed, nil
}

func requirePersistedAuthorization(
	ctx context.Context,
	transaction *sql.Tx,
	receipt identity.AuthorizationReceipt,
	requestedBy identity.PrincipalRef,
	projectRef goal.ProjectRef,
	permission identity.Permission,
	resourceRef string,
) (identity.Membership, error) {
	persisted, err := requirePersistedAuthorizationFact(
		ctx, transaction, receipt, requestedBy, projectRef, permission, resourceRef,
	)
	if err != nil {
		return identity.Membership{}, err
	}
	decision := persisted.Decision()
	membership, err := readMembership(ctx, transaction, requestedBy, projectRef)
	if err != nil {
		if application.IsStateError(err, application.StateNotFound) {
			return identity.Membership{}, conflict(errors.New("sqlite.authorization_membership_missing"))
		}
		return identity.Membership{}, err
	}
	if !membership.IsActive() || membership.Role() != decision.Role() ||
		membership.Revision() != decision.MembershipRevision() {
		return identity.Membership{}, conflict(errors.New("sqlite.authorization_membership_stale"))
	}
	return membership, nil
}

func requirePersistedAuthorizationFact(
	ctx context.Context,
	transaction *sql.Tx,
	receipt identity.AuthorizationReceipt,
	requestedBy identity.PrincipalRef,
	projectRef goal.ProjectRef,
	permission identity.Permission,
	resourceRef string,
) (identity.AuthorizationReceipt, error) {
	request := receipt.Decision().Request()
	if receipt.Ref() == "" || requestedBy.String() == "" || projectRef.String() == "" ||
		request.Principal().Ref != requestedBy || request.ProjectRef() != projectRef ||
		request.Permission() != permission || request.ResourceRef() != resourceRef {
		return identity.AuthorizationReceipt{}, conflict(errors.New("sqlite.authorization_scope_conflict"))
	}
	fingerprint := authorizationRequestFingerprint(request)
	stored, found, err := findAuthorizationByRef(ctx, transaction, receipt.Ref())
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	if !found {
		return identity.AuthorizationReceipt{}, conflict(errors.New("sqlite.authorization_receipt_missing"))
	}
	persisted, err := restoreAuthorizationReceipt(request, fingerprint, stored)
	if err != nil || !sameAuthorizationReceipt(persisted, receipt) {
		if err != nil {
			return identity.AuthorizationReceipt{}, err
		}
		return identity.AuthorizationReceipt{}, conflict(errors.New("sqlite.authorization_receipt_conflict"))
	}
	decision := persisted.Decision()
	if decision.Outcome() != identity.AuthorizationAllowed || !identity.RoleAllows(decision.Role(), permission) {
		return identity.AuthorizationReceipt{}, conflict(errors.New("sqlite.authorization_denied"))
	}
	if err := requirePrincipal(ctx, transaction, request.Principal()); err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	return persisted, nil
}

func validateAuthorizationRequest(request identity.AuthorizationRequest) error {
	if !validText(request.RequestRef()) || identity.ValidatePrincipal(request.Principal()) != nil ||
		request.ProjectRef().String() == "" || identity.ValidatePermission(request.Permission()) != nil ||
		!validText(request.ResourceRef()) || request.RequestedAt().IsZero() {
		return errors.New("sqlite.authorization_request_invalid")
	}
	return nil
}

func ensurePrincipal(ctx context.Context, transaction *sql.Tx, principal identity.Principal) error {
	if err := identity.ValidatePrincipal(principal); err != nil {
		return invalid(err)
	}
	err := requirePrincipal(ctx, transaction, principal)
	if err == nil {
		return nil
	}
	if !application.IsStateError(err, application.StateNotFound) {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO principals(ref, actor_ref, kind, authentication_method)
VALUES (?, ?, ?, ?)`, principal.Ref.String(), principal.ActorRef.String(), string(principal.Kind), principal.Method); err != nil {
		return mapDatabaseError(err)
	}
	return nil
}

func requirePrincipal(ctx context.Context, source queryer, principal identity.Principal) error {
	var actorRef, kind, method string
	err := source.QueryRowContext(ctx, `
SELECT actor_ref, kind, authentication_method FROM principals WHERE ref = ?`, principal.Ref.String()).Scan(
		&actorRef, &kind, &method,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if actorRef != principal.ActorRef.String() || kind != string(principal.Kind) || method != principal.Method {
		return conflict(errors.New("sqlite.principal_conflict"))
	}
	return nil
}

func readMembership(
	ctx context.Context,
	source queryer,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
) (identity.Membership, error) {
	var role, status, grantedBy string
	var revision, grantedAt int64
	var revokedBy sql.NullString
	var revokedAt sql.NullInt64
	err := source.QueryRowContext(ctx, `
SELECT role, revision, status, granted_by_ref, granted_at, revoked_by_ref, revoked_at
FROM project_memberships WHERE principal_ref = ? AND project_ref = ?`,
		principalRef.String(), projectRef.String(),
	).Scan(&role, &revision, &status, &grantedBy, &grantedAt, &revokedBy, &revokedAt)
	if err != nil {
		return identity.Membership{}, mapDatabaseError(err)
	}
	return restoreMembership(principalRef.String(), projectRef.String(), role, revision, status, grantedBy, grantedAt, revokedBy, revokedAt)
}

func restoreMembership(
	principalValue, projectValue, role string,
	revision int64,
	status, grantedBy string,
	grantedAt int64,
	revokedBy sql.NullString,
	revokedAt sql.NullInt64,
) (identity.Membership, error) {
	if revision <= 0 {
		return identity.Membership{}, invalid(errors.New("sqlite.membership_revision_invalid"))
	}
	principalRef, err := identity.NewPrincipalRef(principalValue)
	if err != nil {
		return identity.Membership{}, invalid(err)
	}
	projectRef, err := goal.NewProjectRef(projectValue)
	if err != nil {
		return identity.Membership{}, invalid(err)
	}
	grantor, err := identity.NewPrincipalRef(grantedBy)
	if err != nil {
		return identity.Membership{}, invalid(err)
	}
	var revoker identity.PrincipalRef
	if revokedBy.Valid {
		revoker, err = identity.NewPrincipalRef(revokedBy.String)
		if err != nil {
			return identity.Membership{}, invalid(err)
		}
	}
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: principalRef, ProjectRef: projectRef, Role: identity.Role(role),
		Revision: identity.MembershipRevision(revision), Status: identity.MembershipStatus(status),
		GrantedBy: grantor, GrantedAt: time.Unix(0, grantedAt).UTC(),
		RevokedBy: revoker, RevokedAt: restoredTime(revokedAt),
	})
	if err != nil {
		return identity.Membership{}, invalid(err)
	}
	return membership, nil
}

func findAuthorizationByRequest(
	ctx context.Context,
	source queryer,
	principalRef identity.PrincipalRef,
	requestRef string,
) (authorizationRow, bool, error) {
	return scanAuthorization(source.QueryRowContext(ctx, `
SELECT ref, request_fingerprint, project_ref, permission, resource_ref,
       requested_at, outcome, role, membership_revision, reason_code,
       decided_at, recorded_at
FROM authorization_receipts WHERE principal_ref = ? AND request_ref = ?`,
		principalRef.String(), requestRef,
	))
}

func findAuthorizationByRef(ctx context.Context, source queryer, ref string) (authorizationRow, bool, error) {
	return scanAuthorization(source.QueryRowContext(ctx, `
SELECT ref, request_fingerprint, project_ref, permission, resource_ref,
       requested_at, outcome, role, membership_revision, reason_code,
       decided_at, recorded_at
FROM authorization_receipts WHERE ref = ?`, ref))
}

func scanAuthorization(row *sql.Row) (authorizationRow, bool, error) {
	var stored authorizationRow
	err := row.Scan(
		&stored.ref, &stored.fingerprint, &stored.projectRef, &stored.permission,
		&stored.resourceRef, &stored.requestedAt, &stored.outcome, &stored.role,
		&stored.membershipRevision, &stored.reasonCode, &stored.decidedAt, &stored.recordedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return authorizationRow{}, false, nil
	}
	if err != nil {
		return authorizationRow{}, false, mapDatabaseError(err)
	}
	return stored, true, nil
}

func restoreAuthorizationReceipt(
	request identity.AuthorizationRequest,
	fingerprint string,
	stored authorizationRow,
) (identity.AuthorizationReceipt, error) {
	if stored.fingerprint != fingerprint || stored.projectRef != request.ProjectRef().String() ||
		stored.permission != string(request.Permission()) || stored.resourceRef != request.ResourceRef() ||
		stored.requestedAt != requiredTime(request.RequestedAt()) || stored.membershipRevision < 0 {
		return identity.AuthorizationReceipt{}, conflict(errors.New("sqlite.authorization_request_conflict"))
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: identity.AuthorizationOutcome(stored.outcome), Role: identity.Role(stored.role),
		MembershipRevision: identity.MembershipRevision(stored.membershipRevision),
		ReasonCode:         stored.reasonCode, DecidedAt: time.Unix(0, stored.decidedAt).UTC(),
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: stored.ref, Decision: decision, RecordedAt: time.Unix(0, stored.recordedAt).UTC(),
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, invalid(err)
	}
	return receipt, nil
}

func sameAuthorizationReceipt(left, right identity.AuthorizationReceipt) bool {
	if left.Ref() != right.Ref() || !left.RecordedAt().Equal(right.RecordedAt()) {
		return false
	}
	l, r := left.Decision(), right.Decision()
	return l.Request() == r.Request() && l.Outcome() == r.Outcome() && l.Role() == r.Role() &&
		l.MembershipRevision() == r.MembershipRevision() && l.ReasonCode() == r.ReasonCode() &&
		l.DecidedAt().Equal(r.DecidedAt())
}

func authorizationRequestFingerprint(request identity.AuthorizationRequest) string {
	principal := request.Principal()
	return canonicalFingerprint(
		"authorization-request.v1", request.RequestRef(), principal.Ref.String(),
		principal.ActorRef.String(), string(principal.Kind), principal.Method,
		request.ProjectRef().String(),
		string(request.Permission()), request.ResourceRef(), canonicalTime(request.RequestedAt()),
	)
}

func deterministicRef(prefix, fingerprint string) string {
	return prefix + "-" + canonicalFingerprint(prefix+".v1", fingerprint)
}

func canonicalFingerprint(parts ...string) string {
	hash := sha256.New()
	var length [8]byte
	for _, part := range parts {
		binary.BigEndian.PutUint64(length[:], uint64(len(part)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(part))
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func canonicalTime(value time.Time) string {
	return strconv.FormatInt(value.Round(0).UTC().UnixNano(), 10)
}
