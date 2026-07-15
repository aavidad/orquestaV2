package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type membershipAuditRow struct {
	ref              string
	fingerprint      string
	action           string
	targetRef        string
	projectRef       string
	role             string
	previousRevision int64
	revision         int64
	occurredAt       int64
}

func (repository *Repository) GrantMembership(
	ctx context.Context,
	state application.MembershipGrantState,
) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	request := state.Request
	if err := validateMembershipGrantState(state); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	authority, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, request.Actor().Ref,
		request.ProjectRef(), identity.PermissionProjectMembershipManage, request.TargetRef().String(),
	)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if !identity.CanDelegateMembershipRole(authority.Role(), request.Role()) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, conflict(errors.New("sqlite.membership_delegation_denied"))
	}
	if err := ensurePrincipal(ctx, transaction, state.Target); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	fingerprint := membershipGrantFingerprint(state)
	stored, found, err := findMembershipAudit(ctx, transaction, request.Actor().Ref, request.RequestRef())
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if found {
		audit, restoreErr := restoreMembershipAudit(request.Actor().Ref, request.RequestRef(), fingerprint, stored)
		if restoreErr != nil || !grantAuditMatches(audit, request) {
			if restoreErr != nil {
				return identity.Membership{}, identity.MembershipAuditReceipt{}, false, restoreErr
			}
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, conflict(errors.New("sqlite.membership_request_conflict"))
		}
		membership, restoreErr := membershipFromGrantAudit(audit)
		if restoreErr != nil {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, restoreErr
		}
		if err := commit(transaction); err != nil {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
		}
		return membership, audit, false, nil
	}

	current, currentErr := readMembership(ctx, transaction, request.TargetRef(), request.ProjectRef())
	switch {
	case application.IsStateError(currentErr, application.StateNotFound):
		if request.ExpectedRevision() != 0 {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, conflict(errors.New("sqlite.membership_revision_conflict"))
		}
	case currentErr != nil:
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, currentErr
	default:
		if current.Revision() != request.ExpectedRevision() ||
			request.RequestedAt().Before(current.GrantedAt()) ||
			(!current.RevokedAt().IsZero() && request.RequestedAt().Before(current.RevokedAt())) ||
			!identity.CanDelegateMembershipRole(authority.Role(), current.Role()) {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, conflict(errors.New("sqlite.membership_revision_conflict"))
		}
		if current.IsActive() && identity.IsProjectAuthority(current.Role()) && !identity.IsProjectAuthority(request.Role()) {
			if err := requireOtherProjectAuthority(ctx, transaction, request.ProjectRef().String(), request.TargetRef().String()); err != nil {
				return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
			}
		}
	}

	revision := request.ExpectedRevision() + 1
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: request.Role(),
		Revision: revision, Status: identity.MembershipActive,
		GrantedBy: request.Actor().Ref, GrantedAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, invalid(err)
	}
	if request.ExpectedRevision() == 0 {
		_, err = transaction.ExecContext(ctx, `
INSERT INTO project_memberships(
    principal_ref, project_ref, role, revision, status, granted_by_ref, granted_at,
    revoked_by_ref, revoked_at
) VALUES (?, ?, ?, 1, 'active', ?, ?, NULL, NULL)`,
			request.TargetRef().String(), request.ProjectRef().String(), string(request.Role()),
			request.Actor().Ref.String(), requiredTime(request.RequestedAt()),
		)
	} else {
		var result sql.Result
		result, err = transaction.ExecContext(ctx, `
UPDATE project_memberships
SET role = ?, revision = ?, status = 'active', granted_by_ref = ?, granted_at = ?,
    revoked_by_ref = NULL, revoked_at = NULL
WHERE principal_ref = ? AND project_ref = ? AND revision = ?`,
			string(request.Role()), int64(revision), request.Actor().Ref.String(), requiredTime(request.RequestedAt()),
			request.TargetRef().String(), request.ProjectRef().String(), int64(request.ExpectedRevision()),
		)
		if err == nil {
			err = requireOneRow(result)
		}
	}
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, mapDatabaseError(err)
	}
	audit, err := newMembershipAudit(
		fingerprint, request.RequestRef(), identity.MembershipAuditGranted,
		request.Actor().Ref, request.TargetRef(), request.ProjectRef(), request.Role(),
		request.ExpectedRevision(), revision, request.RequestedAt(),
	)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if err := insertMembershipAudit(ctx, transaction, fingerprint, audit); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if err := commit(transaction); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	return membership, audit, true, nil
}

func (repository *Repository) RevokeMembership(
	ctx context.Context,
	state application.MembershipRevokeState,
) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	request := state.Request
	if err := validateMembershipRevokeState(state); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	authority, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, request.Actor().Ref,
		request.ProjectRef(), identity.PermissionProjectMembershipManage, request.TargetRef().String(),
	)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	fingerprint := membershipRevokeFingerprint(state)
	stored, found, err := findMembershipAudit(ctx, transaction, request.Actor().Ref, request.RequestRef())
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if found {
		audit, restoreErr := restoreMembershipAudit(request.Actor().Ref, request.RequestRef(), fingerprint, stored)
		if restoreErr != nil || !revokeAuditMatches(audit, request) {
			if restoreErr != nil {
				return identity.Membership{}, identity.MembershipAuditReceipt{}, false, restoreErr
			}
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, conflict(errors.New("sqlite.membership_request_conflict"))
		}
		membership, readErr := readMembership(ctx, transaction, request.TargetRef(), request.ProjectRef())
		if readErr != nil || membership.Revision() != audit.Revision() ||
			membership.Status() != identity.MembershipRevoked {
			if readErr != nil && !application.IsStateError(readErr, application.StateNotFound) {
				return identity.Membership{}, identity.MembershipAuditReceipt{}, false, readErr
			}
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, conflict(errors.New("sqlite.membership_replay_superseded"))
		}
		if err := commit(transaction); err != nil {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
		}
		return membership, audit, false, nil
	}

	current, err := readMembership(ctx, transaction, request.TargetRef(), request.ProjectRef())
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if !current.IsActive() || current.Revision() != request.ExpectedRevision() ||
		request.RequestedAt().Before(current.GrantedAt()) || !identity.CanDelegateMembershipRole(authority.Role(), current.Role()) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, conflict(errors.New("sqlite.membership_revision_conflict"))
	}
	if identity.IsProjectAuthority(current.Role()) {
		if err := requireOtherProjectAuthority(ctx, transaction, request.ProjectRef().String(), request.TargetRef().String()); err != nil {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
		}
	}
	revision := request.ExpectedRevision() + 1
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: current.Role(),
		Revision: revision, Status: identity.MembershipRevoked,
		GrantedBy: current.GrantedBy(), GrantedAt: current.GrantedAt(),
		RevokedBy: request.Actor().Ref, RevokedAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, invalid(err)
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE project_memberships
SET revision = ?, status = 'revoked', revoked_by_ref = ?, revoked_at = ?
WHERE principal_ref = ? AND project_ref = ? AND revision = ? AND status = 'active'`,
		int64(revision), request.Actor().Ref.String(), requiredTime(request.RequestedAt()),
		request.TargetRef().String(), request.ProjectRef().String(), int64(request.ExpectedRevision()),
	)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	audit, err := newMembershipAudit(
		fingerprint, request.RequestRef(), identity.MembershipAuditRevoked,
		request.Actor().Ref, request.TargetRef(), request.ProjectRef(), current.Role(),
		request.ExpectedRevision(), revision, request.RequestedAt(),
	)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if err := insertMembershipAudit(ctx, transaction, fingerprint, audit); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if err := commit(transaction); err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	return membership, audit, true, nil
}

func validateMembershipGrantState(state application.MembershipGrantState) error {
	request := state.Request
	if request.RequestRef() == "" || identity.ValidatePrincipal(request.Actor()) != nil ||
		identity.ValidatePrincipal(state.Target) != nil || request.TargetRef() != state.Target.Ref ||
		request.ProjectRef().String() == "" || identity.ValidateRole(request.Role()) != nil || request.RequestedAt().IsZero() {
		return errors.New("sqlite.membership_grant_invalid")
	}
	return nil
}

func validateMembershipRevokeState(state application.MembershipRevokeState) error {
	request := state.Request
	if request.RequestRef() == "" || identity.ValidatePrincipal(request.Actor()) != nil ||
		request.TargetRef().String() == "" || request.ProjectRef().String() == "" ||
		request.ExpectedRevision() == 0 || request.RequestedAt().IsZero() {
		return errors.New("sqlite.membership_revoke_invalid")
	}
	return nil
}

func requireOtherProjectAuthority(
	ctx context.Context,
	transaction *sql.Tx,
	projectRef, excludedPrincipalRef string,
) error {
	var count int
	err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM project_memberships
WHERE project_ref = ? AND principal_ref <> ? AND status = 'active'
  AND role IN ('platform_admin', 'project_owner')`, projectRef, excludedPrincipalRef).Scan(&count)
	if err != nil {
		return mapDatabaseError(err)
	}
	if count == 0 {
		return conflict(errors.New("sqlite.last_project_authority"))
	}
	return nil
}

func membershipGrantFingerprint(state application.MembershipGrantState) string {
	request, target := state.Request, state.Target
	return canonicalFingerprint(
		"membership-grant.v1", request.RequestRef(),
		request.Actor().Ref.String(), request.Actor().ActorRef.String(), string(request.Actor().Kind), request.Actor().Method,
		request.TargetRef().String(), target.ActorRef.String(), string(target.Kind), target.Method,
		request.ProjectRef().String(), string(request.Role()),
		canonicalRevision(request.ExpectedRevision()), canonicalTime(request.RequestedAt()),
	)
}

func membershipRevokeFingerprint(state application.MembershipRevokeState) string {
	request := state.Request
	return canonicalFingerprint(
		"membership-revoke.v1", request.RequestRef(),
		request.Actor().Ref.String(), request.Actor().ActorRef.String(), string(request.Actor().Kind), request.Actor().Method,
		request.TargetRef().String(), request.ProjectRef().String(),
		canonicalRevision(request.ExpectedRevision()), canonicalTime(request.RequestedAt()),
	)
}

func canonicalRevision(revision identity.MembershipRevision) string {
	return strconv.FormatUint(uint64(revision), 10)
}

func newMembershipAudit(
	fingerprint, requestRef string,
	action identity.MembershipAuditAction,
	actorRef, targetRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	role identity.Role,
	previous, revision identity.MembershipRevision,
	occurredAt time.Time,
) (identity.MembershipAuditReceipt, error) {
	audit, err := identity.NewMembershipAuditReceipt(identity.MembershipAuditReceiptInput{
		Ref: deterministicRef("membership-audit", fingerprint), RequestRef: requestRef,
		Action: action, ActorRef: actorRef, TargetRef: targetRef, ProjectRef: projectRef,
		Role: role, PreviousRevision: previous, Revision: revision, OccurredAt: occurredAt,
	})
	if err != nil {
		return identity.MembershipAuditReceipt{}, invalid(err)
	}
	return audit, nil
}

func insertMembershipAudit(
	ctx context.Context,
	transaction *sql.Tx,
	fingerprint string,
	audit identity.MembershipAuditReceipt,
) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO membership_audit_receipts(
    ref, request_ref, request_fingerprint, action, actor_ref, target_ref,
    project_ref, role, previous_revision, revision, occurred_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		audit.Ref(), audit.RequestRef(), fingerprint, string(audit.Action()),
		audit.ActorRef().String(), audit.TargetRef().String(), audit.ProjectRef().String(),
		string(audit.Role()), int64(audit.PreviousRevision()), int64(audit.Revision()),
		requiredTime(audit.OccurredAt()),
	)
	return mapDatabaseError(err)
}

func findMembershipAudit(
	ctx context.Context,
	source queryer,
	actorRef identity.PrincipalRef,
	requestRef string,
) (membershipAuditRow, bool, error) {
	var stored membershipAuditRow
	err := source.QueryRowContext(ctx, `
SELECT ref, request_fingerprint, action, target_ref, project_ref, role,
       previous_revision, revision, occurred_at
FROM membership_audit_receipts WHERE actor_ref = ? AND request_ref = ?`,
		actorRef.String(), requestRef,
	).Scan(
		&stored.ref, &stored.fingerprint, &stored.action, &stored.targetRef,
		&stored.projectRef, &stored.role, &stored.previousRevision,
		&stored.revision, &stored.occurredAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return membershipAuditRow{}, false, nil
	}
	if err != nil {
		return membershipAuditRow{}, false, mapDatabaseError(err)
	}
	return stored, true, nil
}

func restoreMembershipAudit(
	actorRef identity.PrincipalRef,
	requestRef, fingerprint string,
	stored membershipAuditRow,
) (identity.MembershipAuditReceipt, error) {
	if stored.fingerprint != fingerprint || stored.previousRevision < 0 || stored.revision <= 0 {
		return identity.MembershipAuditReceipt{}, conflict(errors.New("sqlite.membership_request_conflict"))
	}
	targetRef, err := identity.NewPrincipalRef(stored.targetRef)
	if err != nil {
		return identity.MembershipAuditReceipt{}, invalid(err)
	}
	projectRef, err := goal.NewProjectRef(stored.projectRef)
	if err != nil {
		return identity.MembershipAuditReceipt{}, invalid(err)
	}
	audit, err := identity.NewMembershipAuditReceipt(identity.MembershipAuditReceiptInput{
		Ref: stored.ref, RequestRef: requestRef, Action: identity.MembershipAuditAction(stored.action),
		ActorRef: actorRef, TargetRef: targetRef, ProjectRef: projectRef, Role: identity.Role(stored.role),
		PreviousRevision: identity.MembershipRevision(stored.previousRevision),
		Revision:         identity.MembershipRevision(stored.revision), OccurredAt: time.Unix(0, stored.occurredAt).UTC(),
	})
	if err != nil {
		return identity.MembershipAuditReceipt{}, invalid(err)
	}
	return audit, nil
}

func grantAuditMatches(audit identity.MembershipAuditReceipt, request identity.MembershipGrantRequest) bool {
	return audit.Action() == identity.MembershipAuditGranted && audit.RequestRef() == request.RequestRef() &&
		audit.ActorRef() == request.Actor().Ref && audit.TargetRef() == request.TargetRef() &&
		audit.ProjectRef() == request.ProjectRef() && audit.Role() == request.Role() &&
		audit.PreviousRevision() == request.ExpectedRevision() &&
		audit.Revision() == request.ExpectedRevision()+1 && audit.OccurredAt().Equal(request.RequestedAt())
}

func revokeAuditMatches(audit identity.MembershipAuditReceipt, request identity.MembershipRevokeRequest) bool {
	return audit.Action() == identity.MembershipAuditRevoked && audit.RequestRef() == request.RequestRef() &&
		audit.ActorRef() == request.Actor().Ref && audit.TargetRef() == request.TargetRef() &&
		audit.ProjectRef() == request.ProjectRef() && audit.PreviousRevision() == request.ExpectedRevision() &&
		audit.Revision() == request.ExpectedRevision()+1 && audit.OccurredAt().Equal(request.RequestedAt())
}

func membershipFromGrantAudit(audit identity.MembershipAuditReceipt) (identity.Membership, error) {
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: audit.TargetRef(), ProjectRef: audit.ProjectRef(), Role: audit.Role(),
		Revision: audit.Revision(), Status: identity.MembershipActive,
		GrantedBy: audit.ActorRef(), GrantedAt: audit.OccurredAt(),
	})
	if err != nil {
		return identity.Membership{}, invalid(err)
	}
	return membership, nil
}
