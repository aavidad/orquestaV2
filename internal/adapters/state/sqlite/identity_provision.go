package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

// ProvisionLocalAccess atomically installs one explicit logical hierarchy and
// its initial local membership. It is bootstrap wiring, not ambient authority:
// callers still need normal authorization receipts for every use case.
func (repository *Repository) ProvisionLocalAccess(
	ctx context.Context,
	principal identity.Principal,
	hierarchy identity.ProjectHierarchy,
	role identity.Role,
	at time.Time,
) error {
	if identity.ValidatePrincipal(principal) != nil || hierarchy.ProjectRef().String() == "" ||
		identity.ValidateRole(role) != nil || !isProjectAuthority(role) || at.IsZero() {
		return invalid(errors.New("sqlite.local_access_invalid"))
	}
	at = at.Round(0).UTC()
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	if err := ensurePrincipal(ctx, transaction, principal); err != nil {
		return err
	}

	existing, membershipErr := readMembership(ctx, transaction, principal.Ref, hierarchy.ProjectRef())
	existingSeed := membershipErr == nil
	switch {
	case application.IsStateError(membershipErr, application.StateNotFound):
		var projectExists int
		err := transaction.QueryRowContext(
			ctx, `SELECT 1 FROM projects WHERE ref = ?`, hierarchy.ProjectRef().String(),
		).Scan(&projectExists)
		if err == nil {
			return conflict(errors.New("sqlite.local_access_project_already_provisioned"))
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return mapDatabaseError(err)
		}
		if err := ensureHierarchy(ctx, transaction, hierarchy); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO project_memberships(
    principal_ref, project_ref, role, revision, status, granted_by_ref, granted_at,
    revoked_by_ref, revoked_at
) VALUES (?, ?, ?, 1, 'active', ?, ?, NULL, NULL)`,
			principal.Ref.String(), hierarchy.ProjectRef().String(), string(role),
			principal.Ref.String(), requiredTime(at),
		); err != nil {
			return mapDatabaseError(err)
		}
	case membershipErr != nil:
		return membershipErr
	default:
		if !existing.IsActive() || existing.Role() != role || existing.Revision() != 1 ||
			existing.GrantedBy() != principal.Ref {
			return conflict(errors.New("sqlite.local_access_membership_conflict"))
		}
		// A restart supplies a fresh clock value. The immutable first grant is
		// the canonical bootstrap time and must be replayed, not re-granted.
		at = existing.GrantedAt()
	}
	fingerprint := canonicalFingerprint(
		"local-access-provision.v1", principal.Ref.String(), principal.ActorRef.String(),
		string(principal.Kind), principal.Method,
		hierarchy.WorkspaceRef().String(), hierarchy.GroupRef().String(),
		hierarchy.ProjectRef().String(), hierarchy.RepositoryRef().String(),
		string(role), canonicalTime(at),
	)
	requestRef := deterministicRef("local-access-request", fingerprint)

	audit, err := newMembershipAudit(
		fingerprint, requestRef, identity.MembershipAuditGranted,
		principal.Ref, principal.Ref, hierarchy.ProjectRef(), role, 0, 1, at,
	)
	if err != nil {
		return err
	}
	stored, found, err := findMembershipAudit(ctx, transaction, principal.Ref, requestRef)
	if err != nil {
		return err
	}
	if existingSeed {
		var initialAudits int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM membership_audit_receipts
WHERE actor_ref = ? AND target_ref = ? AND project_ref = ?
  AND action = 'membership.granted' AND previous_revision = 0 AND revision = 1`,
			principal.Ref.String(), principal.Ref.String(), hierarchy.ProjectRef().String(),
		).Scan(&initialAudits); err != nil {
			return mapDatabaseError(err)
		}
		if initialAudits != 1 || !found {
			return conflict(errors.New("sqlite.local_access_audit_conflict"))
		}
		if err := ensureHierarchy(ctx, transaction, hierarchy); err != nil {
			return err
		}
	}
	if found {
		restored, restoreErr := restoreMembershipAudit(principal.Ref, requestRef, fingerprint, stored)
		if restoreErr != nil || !sameMembershipAudit(restored, audit) {
			if restoreErr != nil {
				return restoreErr
			}
			return conflict(errors.New("sqlite.local_access_audit_conflict"))
		}
	} else if err := insertMembershipAudit(ctx, transaction, fingerprint, audit); err != nil {
		return err
	}
	return commit(transaction)
}

func ensureHierarchy(ctx context.Context, transaction *sql.Tx, hierarchy identity.ProjectHierarchy) error {
	if err := ensureSingleRef(ctx, transaction, "workspaces", hierarchy.WorkspaceRef().String()); err != nil {
		return err
	}
	if err := ensureParentedRef(
		ctx, transaction, "groups", "workspace_ref",
		hierarchy.GroupRef().String(), hierarchy.WorkspaceRef().String(),
	); err != nil {
		return err
	}
	if err := ensureParentedRef(
		ctx, transaction, "projects", "group_ref",
		hierarchy.ProjectRef().String(), hierarchy.GroupRef().String(),
	); err != nil {
		return err
	}
	return ensureParentedRef(
		ctx, transaction, "repositories", "project_ref",
		hierarchy.RepositoryRef().String(), hierarchy.ProjectRef().String(),
	)
}

func ensureSingleRef(ctx context.Context, transaction *sql.Tx, table, ref string) error {
	query := "SELECT ref FROM " + table + " WHERE ref = ?" // table is an internal constant.
	var stored string
	err := transaction.QueryRowContext(ctx, query, ref).Scan(&stored)
	if err == nil {
		if stored != ref {
			return conflict(errors.New("sqlite.hierarchy_conflict"))
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return mapDatabaseError(err)
	}
	_, err = transaction.ExecContext(ctx, "INSERT INTO "+table+"(ref) VALUES (?)", ref)
	return mapDatabaseError(err)
}

func ensureParentedRef(
	ctx context.Context,
	transaction *sql.Tx,
	table, parentColumn, ref, parentRef string,
) error {
	query := "SELECT " + parentColumn + " FROM " + table + " WHERE ref = ?" // identifiers are internal constants.
	var storedParent string
	err := transaction.QueryRowContext(ctx, query, ref).Scan(&storedParent)
	if err == nil {
		if storedParent != parentRef {
			return conflict(errors.New("sqlite.hierarchy_parent_conflict"))
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return mapDatabaseError(err)
	}
	_, err = transaction.ExecContext(
		ctx, "INSERT INTO "+table+"(ref, "+parentColumn+") VALUES (?, ?)", ref, parentRef,
	)
	return mapDatabaseError(err)
}

func sameMembershipAudit(left, right identity.MembershipAuditReceipt) bool {
	return left.Ref() == right.Ref() && left.RequestRef() == right.RequestRef() &&
		left.Action() == right.Action() && left.ActorRef() == right.ActorRef() &&
		left.TargetRef() == right.TargetRef() && left.ProjectRef() == right.ProjectRef() &&
		left.Role() == right.Role() && left.PreviousRevision() == right.PreviousRevision() &&
		left.Revision() == right.Revision() && left.OccurredAt().Equal(right.OccurredAt())
}
