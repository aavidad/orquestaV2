package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func validateRecoveryV10Identity(ctx context.Context, transaction *sql.Tx) error {
	if err := validateRecoveryV10Principals(ctx, transaction); err != nil {
		return err
	}
	if err := validateRecoveryV10Hierarchy(ctx, transaction); err != nil {
		return err
	}
	if err := validateRecoveryV10Memberships(ctx, transaction); err != nil {
		return err
	}
	if err := validateRecoveryV10AuthorizationReceipts(ctx, transaction); err != nil {
		return err
	}
	return validateRecoveryV10RequestedBy(ctx, transaction)
}

func validateRecoveryV10Principals(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT ref, actor_ref, kind, authentication_method FROM principals ORDER BY ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var refValue, actorValue, kind, method string
		if err := rows.Scan(&refValue, &actorValue, &kind, &method); err != nil {
			return err
		}
		ref, err := identity.NewPrincipalRef(refValue)
		if err != nil {
			return err
		}
		actor, err := goal.NewActorRef(actorValue)
		if err != nil {
			return err
		}
		if _, err := identity.NewPrincipal(ref, actor, identity.PrincipalKind(kind), method); err != nil {
			return err
		}
	}
	return rows.Err()
}

func validateRecoveryV10Hierarchy(ctx context.Context, transaction *sql.Tx) error {
	descriptors := []struct {
		query    string
		validate func(string) error
	}{
		{"SELECT ref FROM workspaces ORDER BY ref", func(value string) error {
			_, err := identity.NewWorkspaceRef(value)
			return err
		}},
		{"SELECT ref FROM groups ORDER BY ref", func(value string) error {
			_, err := identity.NewGroupRef(value)
			return err
		}},
		{"SELECT ref FROM projects ORDER BY ref", func(value string) error {
			_, err := goal.NewProjectRef(value)
			return err
		}},
		{"SELECT ref FROM repositories ORDER BY ref", func(value string) error {
			_, err := identity.NewRepositoryRef(value)
			return err
		}},
	}
	for _, descriptor := range descriptors {
		rows, err := transaction.QueryContext(ctx, descriptor.query)
		if err != nil {
			return err
		}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				_ = rows.Close()
				return err
			}
			if err := descriptor.validate(value); err != nil {
				_ = rows.Close()
				return err
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	rows, err := transaction.QueryContext(ctx, `
SELECT w.ref, g.ref, g.workspace_ref, p.ref, p.group_ref, r.ref, r.project_ref
FROM repositories r
JOIN projects p ON p.ref = r.project_ref
JOIN groups g ON g.ref = p.group_ref
JOIN workspaces w ON w.ref = g.workspace_ref
ORDER BY r.ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var workspaceValue, groupValue, groupParent, projectValue, projectParent, repositoryValue, repositoryParent string
		if err := rows.Scan(
			&workspaceValue, &groupValue, &groupParent, &projectValue,
			&projectParent, &repositoryValue, &repositoryParent,
		); err != nil {
			return err
		}
		workspace, err := identity.NewWorkspaceRef(workspaceValue)
		if err != nil {
			return err
		}
		group, err := identity.NewGroupRef(groupValue)
		if err != nil {
			return err
		}
		groupWorkspace, err := identity.NewWorkspaceRef(groupParent)
		if err != nil {
			return err
		}
		project, err := goal.NewProjectRef(projectValue)
		if err != nil {
			return err
		}
		projectGroup, err := identity.NewGroupRef(projectParent)
		if err != nil {
			return err
		}
		repository, err := identity.NewRepositoryRef(repositoryValue)
		if err != nil {
			return err
		}
		repositoryProject, err := goal.NewProjectRef(repositoryParent)
		if err != nil {
			return err
		}
		if _, err := identity.NewProjectHierarchy(identity.ProjectHierarchyInput{
			WorkspaceRef: workspace, GroupRef: group, GroupParentWorkspaceRef: groupWorkspace,
			ProjectRef: project, ProjectParentGroupRef: projectGroup,
			RepositoryRef: repository, RepositoryParentProjectRef: repositoryProject,
		}); err != nil {
			return err
		}
	}
	return rows.Err()
}

type recoveryMembershipKey struct {
	principal identity.PrincipalRef
	project   goal.ProjectRef
}

type recoveryMembershipHistory struct {
	membership identity.Membership
	audits     []identity.MembershipAuditReceipt
}

func validateRecoveryV10Memberships(ctx context.Context, transaction *sql.Tx) error {
	histories := make(map[recoveryMembershipKey]*recoveryMembershipHistory)
	var orderedHistories []*recoveryMembershipHistory
	rows, err := transaction.QueryContext(ctx, `
SELECT principal_ref, project_ref, role, revision, status,
       granted_by_ref, granted_at, revoked_by_ref, revoked_at
FROM project_memberships ORDER BY principal_ref, project_ref`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var principalValue, projectValue, role, status, grantorValue string
		var revision, grantedAt int64
		var revokerValue sql.NullString
		var revokedAt sql.NullInt64
		if err := rows.Scan(
			&principalValue, &projectValue, &role, &revision, &status,
			&grantorValue, &grantedAt, &revokerValue, &revokedAt,
		); err != nil {
			_ = rows.Close()
			return err
		}
		principal, err := identity.NewPrincipalRef(principalValue)
		if err != nil {
			_ = rows.Close()
			return err
		}
		project, err := goal.NewProjectRef(projectValue)
		if err != nil {
			_ = rows.Close()
			return err
		}
		grantor, err := identity.NewPrincipalRef(grantorValue)
		if err != nil {
			_ = rows.Close()
			return err
		}
		var revoker identity.PrincipalRef
		if revokerValue.Valid {
			revoker, err = identity.NewPrincipalRef(revokerValue.String)
			if err != nil {
				_ = rows.Close()
				return err
			}
		}
		input := identity.MembershipInput{
			PrincipalRef: principal, ProjectRef: project, Role: identity.Role(role),
			Revision: identity.MembershipRevision(revision), Status: identity.MembershipStatus(status),
			GrantedBy: grantor, GrantedAt: time.Unix(0, grantedAt).UTC(), RevokedBy: revoker,
		}
		if revokedAt.Valid {
			input.RevokedAt = time.Unix(0, revokedAt.Int64).UTC()
		}
		membership, err := identity.NewMembership(input)
		if err != nil {
			_ = rows.Close()
			return err
		}
		key := recoveryMembershipKey{principal: principal, project: project}
		history := &recoveryMembershipHistory{membership: membership}
		histories[key] = history
		orderedHistories = append(orderedHistories, history)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	audits, err := transaction.QueryContext(ctx, `
SELECT ref, request_ref, request_fingerprint, action, actor_ref, target_ref,
       project_ref, role, previous_revision, revision, occurred_at
FROM membership_audit_receipts
ORDER BY target_ref, project_ref, revision, ref`)
	if err != nil {
		return err
	}
	for audits.Next() {
		var ref, requestRef, fingerprint, action, actorValue, targetValue, projectValue, role string
		var previousRevision, revision, occurredAt int64
		if err := audits.Scan(
			&ref, &requestRef, &fingerprint, &action, &actorValue, &targetValue,
			&projectValue, &role, &previousRevision, &revision, &occurredAt,
		); err != nil {
			_ = audits.Close()
			return err
		}
		actor, err := identity.NewPrincipalRef(actorValue)
		if err != nil {
			_ = audits.Close()
			return err
		}
		target, err := identity.NewPrincipalRef(targetValue)
		if err != nil {
			_ = audits.Close()
			return err
		}
		project, err := goal.NewProjectRef(projectValue)
		if err != nil {
			_ = audits.Close()
			return err
		}
		if !validText(fingerprint) {
			_ = audits.Close()
			return errors.New("sqlite.recovery_membership_audit_fingerprint_invalid")
		}
		audit, err := identity.NewMembershipAuditReceipt(identity.MembershipAuditReceiptInput{
			Ref: ref, RequestRef: requestRef, Action: identity.MembershipAuditAction(action),
			ActorRef: actor, TargetRef: target, ProjectRef: project, Role: identity.Role(role),
			PreviousRevision: identity.MembershipRevision(previousRevision),
			Revision:         identity.MembershipRevision(revision), OccurredAt: time.Unix(0, occurredAt).UTC(),
		})
		if err != nil {
			_ = audits.Close()
			return err
		}
		history, found := histories[recoveryMembershipKey{principal: target, project: project}]
		if !found {
			_ = audits.Close()
			return errors.New("sqlite.recovery_membership_audit_orphan")
		}
		history.audits = append(history.audits, audit)
	}
	if err := audits.Err(); err != nil {
		_ = audits.Close()
		return err
	}
	if err := audits.Close(); err != nil {
		return err
	}
	for _, history := range orderedHistories {
		if err := validateRecoveryMembershipHistory(*history); err != nil {
			return err
		}
	}
	return nil
}

func validateRecoveryMembershipHistory(history recoveryMembershipHistory) error {
	membership := history.membership
	if uint64(len(history.audits)) != uint64(membership.Revision()) {
		return errors.New("sqlite.recovery_membership_audit_chain_invalid")
	}
	var previous, lastGrant identity.MembershipAuditReceipt
	for index, audit := range history.audits {
		revision := identity.MembershipRevision(index + 1)
		if audit.PreviousRevision() != revision-1 || audit.Revision() != revision ||
			audit.TargetRef() != membership.PrincipalRef() || audit.ProjectRef() != membership.ProjectRef() ||
			(index > 0 && audit.OccurredAt().Before(previous.OccurredAt())) {
			return errors.New("sqlite.recovery_membership_audit_chain_invalid")
		}
		switch audit.Action() {
		case identity.MembershipAuditGranted:
			lastGrant = audit
		case identity.MembershipAuditRevoked:
			if index == 0 || previous.Action() != identity.MembershipAuditGranted ||
				lastGrant.Ref() == "" || audit.Role() != lastGrant.Role() {
				return errors.New("sqlite.recovery_membership_audit_chain_invalid")
			}
		default:
			return errors.New("sqlite.recovery_membership_audit_chain_invalid")
		}
		previous = audit
	}
	if lastGrant.Ref() == "" || membership.Role() != lastGrant.Role() ||
		membership.GrantedBy() != lastGrant.ActorRef() ||
		!membership.GrantedAt().Equal(lastGrant.OccurredAt()) {
		return errors.New("sqlite.recovery_membership_audit_binding_invalid")
	}
	latest := history.audits[len(history.audits)-1]
	switch latest.Action() {
	case identity.MembershipAuditGranted:
		if !membership.IsActive() {
			return errors.New("sqlite.recovery_membership_audit_binding_invalid")
		}
	case identity.MembershipAuditRevoked:
		if membership.Status() != identity.MembershipRevoked ||
			membership.RevokedBy() != latest.ActorRef() ||
			!membership.RevokedAt().Equal(latest.OccurredAt()) {
			return errors.New("sqlite.recovery_membership_audit_binding_invalid")
		}
	default:
		return errors.New("sqlite.recovery_membership_audit_chain_invalid")
	}
	return nil
}

func validateRecoveryV10AuthorizationReceipts(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT a.ref, a.request_ref, a.request_fingerprint,
       p.ref, p.actor_ref, p.kind, p.authentication_method,
       a.project_ref, a.permission, a.resource_ref, a.requested_at,
       a.outcome, a.role, a.membership_revision, a.reason_code,
       a.decided_at, a.recorded_at
FROM authorization_receipts a
JOIN principals p ON p.ref = a.principal_ref
ORDER BY a.ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var receiptRef, requestRef, fingerprint string
		var principalValue, actorValue, kind, method string
		var projectValue, permission, resourceRef, outcome, role, reason string
		var requestedAt, membershipRevision, decidedAt, recordedAt int64
		if err := rows.Scan(
			&receiptRef, &requestRef, &fingerprint,
			&principalValue, &actorValue, &kind, &method,
			&projectValue, &permission, &resourceRef, &requestedAt,
			&outcome, &role, &membershipRevision, &reason, &decidedAt, &recordedAt,
		); err != nil {
			return err
		}
		principalRef, err := identity.NewPrincipalRef(principalValue)
		if err != nil {
			return err
		}
		actorRef, err := goal.NewActorRef(actorValue)
		if err != nil {
			return err
		}
		principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKind(kind), method)
		if err != nil {
			return err
		}
		projectRef, err := goal.NewProjectRef(projectValue)
		if err != nil {
			return err
		}
		request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
			RequestRef: requestRef, Principal: principal, ProjectRef: projectRef,
			Permission: identity.Permission(permission), ResourceRef: resourceRef,
			RequestedAt: time.Unix(0, requestedAt).UTC(),
		})
		if err != nil {
			return err
		}
		if fingerprint != authorizationRequestFingerprint(request) ||
			receiptRef != deterministicRef("authorization-receipt", fingerprint) {
			return errors.New("sqlite.recovery_authorization_fingerprint_invalid")
		}
		decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
			Request: request, Outcome: identity.AuthorizationOutcome(outcome), Role: identity.Role(role),
			MembershipRevision: identity.MembershipRevision(membershipRevision), ReasonCode: reason,
			DecidedAt: time.Unix(0, decidedAt).UTC(),
		})
		if err != nil {
			return err
		}
		if err := validateRecoveryAuthorizationTuple(decision); err != nil {
			return err
		}
		if _, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
			Ref: receiptRef, Decision: decision, RecordedAt: time.Unix(0, recordedAt).UTC(),
		}); err != nil {
			return err
		}
	}
	return rows.Err()
}

func validateRecoveryAuthorizationTuple(decision identity.AuthorizationDecision) error {
	permission := decision.Request().Permission()
	role := decision.Role()
	revision := decision.MembershipRevision()
	valid := false
	switch decision.ReasonCode() {
	case authorizationReasonAllowed:
		valid = decision.Outcome() == identity.AuthorizationAllowed &&
			identity.ValidateRole(role) == nil && identity.RoleAllows(role, permission) && revision > 0
	case authorizationReasonProjectUnknown, authorizationReasonMembershipMissing:
		valid = decision.Outcome() == identity.AuthorizationDenied && role == "" && revision == 0
	case authorizationReasonMembershipRevoked:
		valid = decision.Outcome() == identity.AuthorizationDenied &&
			identity.ValidateRole(role) == nil && revision > 0
	case authorizationReasonPermissionDenied:
		valid = decision.Outcome() == identity.AuthorizationDenied &&
			identity.ValidateRole(role) == nil && revision > 0 && !identity.RoleAllows(role, permission)
	}
	if !valid {
		return errors.New("sqlite.recovery_authorization_tuple_invalid")
	}
	return nil
}

func validateRecoveryV10RequestedBy(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT g.requested_by_ref, p.actor_ref, p.kind, p.authentication_method,
       spec.confirmed_by
FROM goals g
JOIN principals p ON p.ref = g.requested_by_ref
JOIN app_specs spec ON spec.ref = g.app_spec_ref
ORDER BY g.ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var principalValue, actorValue, kind, method, confirmedBy string
		if err := rows.Scan(&principalValue, &actorValue, &kind, &method, &confirmedBy); err != nil {
			return err
		}
		principalRef, err := identity.NewPrincipalRef(principalValue)
		if err != nil {
			return err
		}
		actorRef, err := goal.NewActorRef(actorValue)
		if err != nil {
			return err
		}
		if _, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKind(kind), method); err != nil {
			return err
		}
		if actorValue != confirmedBy {
			return errors.New("sqlite.recovery_requested_by_binding_invalid")
		}
	}
	return rows.Err()
}
